// Package worker is Parallax's bounded event-processing pool (prd.md §19, §20).
//
// The pool drains one stream subscription with N goroutines and hands each
// event to a Processor. The runtime owns the pool — concurrency, draining,
// panic isolation, latency measurement. The intelligence team owns what runs
// inside it by implementing Processor (feature / sequence / context / intent).
package worker

import (
	"context"
	"log/slog"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

// Processor consumes a single normalized event. Implementations must be safe
// for concurrent calls (the pool runs several at once) and must respect ctx —
// on a forced shutdown the pool cancels it and waits.
type Processor interface {
	Process(ctx context.Context, e contracts.Event) error
}

// ProcessorFunc adapts a function to Processor.
type ProcessorFunc func(context.Context, contracts.Event) error

// Process implements Processor.
func (f ProcessorFunc) Process(ctx context.Context, e contracts.Event) error { return f(ctx, e) }

// Source is what the pool drains. *stream.Subscription satisfies it.
type Source interface {
	C() <-chan contracts.Event
	Unsubscribe()
}

// Options configures a Pool.
type Options struct {
	// Workers is the number of goroutines. <=0 uses GOMAXPROCS.
	Workers int
	// SampleSize is the latency reservoir capacity. <=0 uses defaultSampleSize.
	SampleSize int
	// Logger is optional.
	Logger *slog.Logger
}

const defaultSampleSize = 8192

// Pool runs a Processor across a bounded set of workers.
type Pool struct {
	src     Source
	proc    Processor
	workers int
	log     *slog.Logger

	ctx      context.Context
	cancel   context.CancelFunc
	hardStop chan struct{}
	wg       sync.WaitGroup
	stopOnce sync.Once

	processed atomic.Uint64
	failed    atomic.Uint64
	panicked  atomic.Uint64
	inFlight  atomic.Int64
	latency   *reservoir
}

// New starts a Pool draining src into proc. Call Stop to shut it down.
func New(src Source, proc Processor, opts Options) *Pool {
	if opts.Workers <= 0 {
		opts.Workers = runtime.GOMAXPROCS(0)
	}
	if opts.Workers < 1 {
		opts.Workers = 1
	}
	if opts.SampleSize <= 0 {
		opts.SampleSize = defaultSampleSize
	}
	if opts.Logger == nil {
		opts.Logger = slog.New(slog.DiscardHandler)
	}

	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{
		src:      src,
		proc:     proc,
		workers:  opts.Workers,
		log:      opts.Logger,
		ctx:      ctx,
		cancel:   cancel,
		hardStop: make(chan struct{}),
		latency:  newReservoir(opts.SampleSize),
	}

	p.wg.Add(p.workers)
	for range p.workers {
		go p.run()
	}
	return p
}

func (p *Pool) run() {
	defer p.wg.Done()
	for {
		select {
		case <-p.hardStop:
			return
		case e, ok := <-p.src.C():
			if !ok {
				return
			}
			p.handle(e)
		}
	}
}

func (p *Pool) handle(e contracts.Event) {
	p.inFlight.Add(1)
	start := time.Now()
	defer func() {
		p.latency.Observe(time.Since(start))
		p.inFlight.Add(-1)
		if r := recover(); r != nil {
			p.panicked.Add(1)
			p.failed.Add(1)
			p.log.Error("processor panicked",
				"event_id", e.EventID, "session", e.SessionID, "panic", r,
				"stack", string(debug.Stack()))
		}
	}()

	if err := p.proc.Process(p.ctx, e); err != nil {
		p.failed.Add(1)
		p.log.Warn("processor error", "event_id", e.EventID, "session", e.SessionID, "err", err)
		return
	}
	p.processed.Add(1)
}

// Stop shuts the pool down. It unsubscribes from the source and waits for
// in-flight work to finish. If ctx is cancelled first it cancels the
// Processor context, stops the workers, waits for them to unwind, and returns
// ctx.Err(). Safe to call more than once.
func (p *Pool) Stop(ctx context.Context) error {
	p.stopOnce.Do(func() { p.src.Unsubscribe() })

	done := make(chan struct{})
	go func() { p.wg.Wait(); close(done) }()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		p.cancel()
		close(p.hardStop)
		<-done
		return ctx.Err()
	}
}

// Stats is a point-in-time snapshot of pool counters.
type Stats struct {
	Workers   int
	Processed uint64
	Failed    uint64
	Panicked  uint64
	InFlight  int64
	Latency   LatencyStats
}

// Stats returns current counters.
func (p *Pool) Stats() Stats {
	return Stats{
		Workers:   p.workers,
		Processed: p.processed.Load(),
		Failed:    p.failed.Load(),
		Panicked:  p.panicked.Load(),
		InFlight:  p.inFlight.Load(),
		Latency:   p.latency.Snapshot(),
	}
}
