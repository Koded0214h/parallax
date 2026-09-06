// Package stream is Parallax's in-process event bus (prd.md §20). Ingested and
// stored events are published once; independent consumers — the worker pool,
// the frontend WebSocket bridge, the write-ahead log — each subscribe and get
// their own bounded buffer.
//
// Design:
//
//   - A single hub goroutine owns the subscriber set and every send. Callers
//     only Publish or receive, so there is no send-on-closed-channel race.
//   - Publish blocks on a bounded intake channel: when the system is saturated
//     that backpressure reaches the HTTP handler. TryPublish is the
//     non-blocking variant.
//   - Each subscription has its own overflow policy. A slow DropOldest /
//     DropNewest consumer only loses its own events (counted). At most one
//     subscription should use Block, since a stalled Block consumer stalls the
//     hub for everyone — that is the intended end-to-end backpressure knob for
//     the primary consumer.
package stream

import (
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

// OverflowPolicy decides what happens when a subscriber's buffer is full.
type OverflowPolicy int

const (
	// DropOldest evicts the oldest queued event to make room for the new one.
	// Good for "show me the latest state" consumers like the frontend.
	DropOldest OverflowPolicy = iota
	// DropNewest discards the incoming event and keeps the queue as-is.
	DropNewest
	// Block makes the hub wait for buffer space. Use for at most one
	// subscription (the primary consumer); it stalls delivery to all others.
	Block
)

func (p OverflowPolicy) String() string {
	switch p {
	case DropOldest:
		return "drop-oldest"
	case DropNewest:
		return "drop-newest"
	case Block:
		return "block"
	default:
		return "unknown"
	}
}

// Options configures a Bus.
type Options struct {
	// Intake is the capacity of the publish queue. <=0 uses defaultIntake.
	Intake int
	// SubBuffer is the default per-subscription buffer. <=0 uses defaultSubBuffer.
	SubBuffer int
	// Policy is the default per-subscription overflow policy.
	Policy OverflowPolicy
	// Logger is optional.
	Logger *slog.Logger
}

const (
	defaultIntake        = 1024
	defaultSubBuffer     = 256
	dropOldestMaxRetries = 4
)

// Bus is a fan-out event bus. Construct with New; the zero value is not usable.
type Bus struct {
	intake     chan contracts.Event
	register   chan *Subscription
	unregister chan *Subscription
	count      chan chan int
	closing    chan struct{}
	closed     chan struct{}
	closeOnce  sync.Once

	defBuffer int
	defPolicy OverflowPolicy
	log       *slog.Logger

	published atomic.Uint64
	dropped   atomic.Uint64

	// owned by run()
	subs map[*Subscription]struct{}
}

// New starts a Bus and its hub goroutine.
func New(opts Options) *Bus {
	if opts.Intake <= 0 {
		opts.Intake = defaultIntake
	}
	if opts.SubBuffer <= 0 {
		opts.SubBuffer = defaultSubBuffer
	}
	if opts.Logger == nil {
		opts.Logger = slog.New(slog.DiscardHandler)
	}
	b := &Bus{
		intake:     make(chan contracts.Event, opts.Intake),
		register:   make(chan *Subscription),
		unregister: make(chan *Subscription),
		count:      make(chan chan int),
		closing:    make(chan struct{}),
		closed:     make(chan struct{}),
		defBuffer:  opts.SubBuffer,
		defPolicy:  opts.Policy,
		log:        opts.Logger,
		subs:       make(map[*Subscription]struct{}),
	}
	go b.run()
	return b
}

// Publish enqueues e for fan-out, blocking if the intake queue is full. After
// Close it returns without delivering.
func (b *Bus) Publish(e contracts.Event) {
	if b.isClosing() {
		return
	}
	select {
	case b.intake <- e:
	case <-b.closing:
	}
}

// TryPublish enqueues e without blocking. It reports whether e was accepted;
// false means the intake queue was full or the bus is closed.
func (b *Bus) TryPublish(e contracts.Event) bool {
	if b.isClosing() {
		return false
	}
	select {
	case b.intake <- e:
		return true
	default:
		return false
	}
}

func (b *Bus) isClosing() bool {
	select {
	case <-b.closing:
		return true
	default:
		return false
	}
}

// SubOption customizes a single subscription.
type SubOption func(*Subscription)

// WithBuffer sets this subscription's buffer capacity.
func WithBuffer(n int) SubOption {
	return func(s *Subscription) {
		if n > 0 {
			s.buffer = n
		}
	}
}

// WithPolicy sets this subscription's overflow policy.
func WithPolicy(p OverflowPolicy) SubOption {
	return func(s *Subscription) { s.policy = p }
}

// Subscribe registers a new consumer. The returned Subscription's channel is
// closed when it is Unsubscribed or the Bus is closed, so `for e := range
// sub.C()` terminates cleanly.
func (b *Bus) Subscribe(name string, opts ...SubOption) *Subscription {
	s := &Subscription{
		name:   name,
		bus:    b,
		buffer: b.defBuffer,
		policy: b.defPolicy,
	}
	for _, o := range opts {
		o(s)
	}
	s.ch = make(chan contracts.Event, s.buffer)
	s.unblock = make(chan struct{})

	select {
	case b.register <- s:
	case <-b.closing:
		close(s.ch)
		s.done.Store(true)
	}
	return s
}

// Stats is a point-in-time view of bus counters.
type Stats struct {
	Published   uint64
	Dropped     uint64
	Subscribers int
}

// Stats returns current bus counters.
func (b *Bus) Stats() Stats {
	res := make(chan int, 1)
	select {
	case b.count <- res:
		return Stats{Published: b.published.Load(), Dropped: b.dropped.Load(), Subscribers: <-res}
	case <-b.closed:
		return Stats{Published: b.published.Load(), Dropped: b.dropped.Load()}
	}
}

// Close stops the bus, closes every subscription channel, and is idempotent.
func (b *Bus) Close() {
	b.closeOnce.Do(func() { close(b.closing) })
	<-b.closed
}

func (b *Bus) run() {
	defer close(b.closed)
	for {
		select {
		case e := <-b.intake:
			b.published.Add(1)
			for s := range b.subs {
				b.deliver(s, e)
			}
		case s := <-b.register:
			b.subs[s] = struct{}{}
		case s := <-b.unregister:
			if _, ok := b.subs[s]; ok {
				delete(b.subs, s)
				close(s.ch)
			}
		case res := <-b.count:
			res <- len(b.subs)
		case <-b.closing:
			for s := range b.subs {
				close(s.ch)
			}
			b.subs = nil
			return
		}
	}
}

// deliver fans e out to one subscription per its overflow policy. Only the hub
// goroutine calls this, so sends and the eventual channel close never race.
//
// Accounting: every call bumps s.seen once. A call bumps s.dropped once per
// event that the consumer will never see — the incoming event when it is
// refused, or an evicted older event under DropOldest. Received is derived as
// seen - dropped, so the invariant seen == received + dropped always holds.
func (b *Bus) deliver(s *Subscription, e contracts.Event) {
	s.seen.Add(1)

	switch s.policy {
	case Block:
		// Also wake on Unsubscribe / Close so a wedged Block consumer (buffer
		// full, nobody reading) can still be detached instead of deadlocking
		// the hub forever.
		select {
		case s.ch <- e:
		case <-s.unblock:
			b.drop(s)
		case <-b.closing:
			b.drop(s)
		}

	case DropNewest:
		select {
		case s.ch <- e:
		default:
			b.drop(s)
		}

	default: // DropOldest
		for range dropOldestMaxRetries {
			select {
			case s.ch <- e:
				return
			default:
			}
			select {
			case <-s.ch:
				b.drop(s) // evicted an older, already-seen event
			default:
			}
		}
		// Consumer is thrashing the buffer; give up and drop the new event.
		select {
		case s.ch <- e:
		default:
			b.drop(s)
		}
	}
}

func (b *Bus) drop(s *Subscription) {
	s.dropped.Add(1)
	b.dropped.Add(1)
}
