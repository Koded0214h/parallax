package worker

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

type fakeSource struct {
	ch   chan contracts.Event
	once sync.Once
}

func newFakeSource(buf int) *fakeSource {
	return &fakeSource{ch: make(chan contracts.Event, buf)}
}

func (f *fakeSource) C() <-chan contracts.Event { return f.ch }
func (f *fakeSource) Unsubscribe()              { f.once.Do(func() { close(f.ch) }) }

func ev(seq uint64) contracts.Event {
	return contracts.Event{
		EventID: "evt", SessionID: "s1", UserID: "u1",
		Type: contracts.EventLogin, Seq: seq,
	}
}

// feed sends seqs 1..n into the source.
func feed(src *fakeSource, n int) {
	for i := 1; i <= n; i++ {
		src.ch <- ev(uint64(i))
	}
}

func TestProcessesEveryEvent(t *testing.T) {
	src := newFakeSource(200)
	var count atomic.Uint64
	p := New(src, ProcessorFunc(func(_ context.Context, _ contracts.Event) error {
		count.Add(1)
		return nil
	}), Options{Workers: 4})

	feed(src, 150)
	if err := p.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if count.Load() != 150 {
		t.Errorf("processor calls = %d, want 150", count.Load())
	}
	st := p.Stats()
	if st.Processed != 150 || st.Failed != 0 || st.InFlight != 0 {
		t.Errorf("stats = %+v, want Processed 150 / Failed 0 / InFlight 0", st)
	}
	if st.Latency.Count != 150 {
		t.Errorf("latency Count = %d, want 150", st.Latency.Count)
	}
}

func TestProcessorErrorsCounted(t *testing.T) {
	src := newFakeSource(20)
	p := New(src, ProcessorFunc(func(_ context.Context, e contracts.Event) error {
		if e.Seq%2 == 0 {
			return errors.New("boom")
		}
		return nil
	}), Options{Workers: 3})

	feed(src, 10)
	_ = p.Stop(context.Background())

	st := p.Stats()
	if st.Processed != 5 || st.Failed != 5 || st.Panicked != 0 {
		t.Errorf("stats = %+v, want Processed 5 / Failed 5 / Panicked 0", st)
	}
	if st.Latency.Count != 10 {
		t.Errorf("latency Count = %d, want 10 (errors are still timed)", st.Latency.Count)
	}
}

func TestPanicIsRecoveredAndIsolated(t *testing.T) {
	src := newFakeSource(20)
	p := New(src, ProcessorFunc(func(_ context.Context, e contracts.Event) error {
		if e.Seq == 3 {
			panic("processor blew up")
		}
		return nil
	}), Options{Workers: 2})

	feed(src, 8)
	if err := p.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	st := p.Stats()
	if st.Panicked != 1 {
		t.Errorf("Panicked = %d, want 1", st.Panicked)
	}
	if st.Failed != 1 {
		t.Errorf("Failed = %d, want 1", st.Failed)
	}
	if st.Processed != 7 {
		t.Errorf("Processed = %d, want 7 (pool survived the panic)", st.Processed)
	}
}

func TestGracefulStopDrainsBufferedWork(t *testing.T) {
	src := newFakeSource(50)
	var count atomic.Uint64
	p := New(src, ProcessorFunc(func(_ context.Context, _ contracts.Event) error {
		time.Sleep(time.Millisecond)
		count.Add(1)
		return nil
	}), Options{Workers: 2})

	feed(src, 50)
	if err := p.Stop(context.Background()); err != nil { // no deadline => drain fully
		t.Fatalf("Stop: %v", err)
	}
	if count.Load() != 50 {
		t.Errorf("processed %d/50 — graceful stop did not drain", count.Load())
	}
}

func TestForcedStopOnDeadline(t *testing.T) {
	baseline := runtime.NumGoroutine()

	src := newFakeSource(10)
	release := make(chan struct{})
	var started atomic.Int64
	p := New(src, ProcessorFunc(func(ctx context.Context, _ contracts.Event) error {
		started.Add(1)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err() // well-behaved: respects cancellation
		}
	}), Options{Workers: 4})

	feed(src, 4)
	// wait until workers are actually blocked inside Process
	deadline := time.Now().Add(time.Second)
	for started.Load() < 4 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	err := p.Stop(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Stop err = %v, want DeadlineExceeded", err)
	}

	close(release)
	// workers should have unwound; goroutine count returns near baseline
	settled := false
	for i := 0; i < 200 && !settled; i++ {
		if runtime.NumGoroutine() <= baseline+2 {
			settled = true
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !settled {
		t.Errorf("goroutines leaked: baseline %d, now %d", baseline, runtime.NumGoroutine())
	}
	if p.Stats().Failed != 4 {
		t.Errorf("Failed = %d, want 4 (all in-flight cancelled)", p.Stats().Failed)
	}
}

func TestWorkersRunInParallel(t *testing.T) {
	const workers = 4
	src := newFakeSource(workers)

	var arrived atomic.Int64
	var maxSeen atomic.Int64
	proceed := make(chan struct{})

	p := New(src, ProcessorFunc(func(_ context.Context, _ contracts.Event) error {
		n := arrived.Add(1)
		for {
			m := maxSeen.Load()
			if n <= m || maxSeen.CompareAndSwap(m, n) {
				break
			}
		}
		if n == workers {
			close(proceed) // only unblocks once all workers are inside Process together
		}
		select {
		case <-proceed:
		case <-time.After(2 * time.Second):
		}
		return nil
	}), Options{Workers: workers})

	feed(src, workers)
	if err := p.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if maxSeen.Load() != workers {
		t.Fatalf("max concurrent Process = %d, want %d", maxSeen.Load(), workers)
	}
}

func TestLatencyReflectsProcessingTime(t *testing.T) {
	src := newFakeSource(30)
	p := New(src, ProcessorFunc(func(_ context.Context, _ contracts.Event) error {
		time.Sleep(2 * time.Millisecond)
		return nil
	}), Options{Workers: 1})

	feed(src, 20)
	_ = p.Stop(context.Background())

	l := p.Stats().Latency
	if l.Count != 20 {
		t.Fatalf("Count = %d, want 20", l.Count)
	}
	if l.Min <= 0 || l.P50 < time.Millisecond {
		t.Errorf("latency too low for a 2ms processor: %+v", l)
	}
	if !(l.P50 <= l.P95 && l.P95 <= l.P99 && l.P99 <= l.Max) {
		t.Errorf("percentiles not monotonic: %+v", l)
	}
}

func TestStopIsIdempotent(t *testing.T) {
	src := newFakeSource(4)
	p := New(src, ProcessorFunc(func(_ context.Context, _ contracts.Event) error { return nil }),
		Options{Workers: 2})
	feed(src, 4)

	if err := p.Stop(context.Background()); err != nil {
		t.Fatalf("first Stop: %v", err)
	}
	if err := p.Stop(context.Background()); err != nil {
		t.Fatalf("second Stop: %v", err)
	}
}

func TestDefaultWorkerCount(t *testing.T) {
	src := newFakeSource(1)
	p := New(src, ProcessorFunc(func(_ context.Context, _ contracts.Event) error { return nil }), Options{})
	defer p.Stop(context.Background())

	if got := p.Stats().Workers; got != runtime.GOMAXPROCS(0) {
		t.Errorf("Workers = %d, want GOMAXPROCS %d", got, runtime.GOMAXPROCS(0))
	}
}

func TestConcurrentStatsWhileProcessing(t *testing.T) {
	src := newFakeSource(500)
	p := New(src, ProcessorFunc(func(_ context.Context, _ contracts.Event) error {
		return nil
	}), Options{Workers: 4})

	var wg sync.WaitGroup
	wg.Go(func() {
		for i := 1; i <= 500; i++ {
			src.ch <- ev(uint64(i))
		}
	})
	wg.Go(func() {
		for range 200 {
			_ = p.Stats() // must not race with worker counter writes
		}
	})
	wg.Wait()

	if err := p.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if p.Stats().Processed != 500 {
		t.Errorf("Processed = %d, want 500", p.Stats().Processed)
	}
}
