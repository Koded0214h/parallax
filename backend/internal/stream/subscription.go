package stream

import (
	"sync"
	"sync/atomic"

	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

// Subscription is one consumer's view of the bus. Obtain it from Bus.Subscribe.
// Receive events from C(); it is closed when the subscription ends.
type Subscription struct {
	name   string
	bus    *Bus
	buffer int
	policy OverflowPolicy
	ch     chan contracts.Event

	once    sync.Once
	done    atomic.Bool
	unblock chan struct{} // closed by Unsubscribe to free a wedged Block delivery

	seen    atomic.Uint64 // events fanned out to this subscription
	dropped atomic.Uint64 // of those, ones the consumer will never see
}

// C is the receive channel. It is closed on Unsubscribe or Bus.Close.
func (s *Subscription) C() <-chan contracts.Event { return s.ch }

// Name is the label passed to Subscribe.
func (s *Subscription) Name() string { return s.name }

// Policy is this subscription's overflow policy.
func (s *Subscription) Policy() OverflowPolicy { return s.policy }

// Unsubscribe removes the subscription from the bus and closes its channel.
// Safe to call multiple times and concurrently.
func (s *Subscription) Unsubscribe() {
	s.once.Do(func() {
		s.done.Store(true)
		close(s.unblock) // frees the hub if it is mid-send to this Block sub
		select {
		case s.bus.unregister <- s:
		case <-s.bus.closed:
		}
	})
}

// SubStats is a point-in-time view of a subscription's counters.
// Seen == Received + Dropped by construction.
type SubStats struct {
	Name     string
	Seen     uint64
	Received uint64
	Dropped  uint64
}

// Stats returns this subscription's counters.
func (s *Subscription) Stats() SubStats {
	seen, dropped := s.seen.Load(), s.dropped.Load()
	return SubStats{
		Name:     s.name,
		Seen:     seen,
		Received: seen - dropped,
		Dropped:  dropped,
	}
}
