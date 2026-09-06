package stream

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

func ev(seq uint64) contracts.Event {
	return contracts.Event{
		SessionID: "s1", UserID: "u1", Type: contracts.EventLogin, Seq: seq,
	}
}

// drain reads n events from sub within the timeout, returning their Seq values.
func drain(t *testing.T, sub *Subscription, n int, timeout time.Duration) []uint64 {
	t.Helper()
	got := make([]uint64, 0, n)
	deadline := time.After(timeout)
	for len(got) < n {
		select {
		case e, ok := <-sub.C():
			if !ok {
				t.Fatalf("channel closed after %d/%d events", len(got), n)
			}
			got = append(got, e.Seq)
		case <-deadline:
			t.Fatalf("timeout after %d/%d events", len(got), n)
		}
	}
	return got
}

func TestOverflowPolicyString(t *testing.T) {
	cases := map[OverflowPolicy]string{
		DropOldest: "drop-oldest", DropNewest: "drop-newest",
		Block: "block", OverflowPolicy(99): "unknown",
	}
	for p, want := range cases {
		if got := p.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", p, got, want)
		}
	}
}

func TestSubscriptionAccessors(t *testing.T) {
	b := New(Options{})
	defer b.Close()
	sub := b.Subscribe("frontend", WithPolicy(DropNewest))
	if sub.Name() != "frontend" {
		t.Errorf("Name() = %q", sub.Name())
	}
	if sub.Policy() != DropNewest {
		t.Errorf("Policy() = %v, want drop-newest", sub.Policy())
	}
}

func TestPublishFanOutInOrder(t *testing.T) {
	b := New(Options{})
	defer b.Close()

	a := b.Subscribe("a")
	c := b.Subscribe("c")

	for i := uint64(1); i <= 5; i++ {
		b.Publish(ev(i))
	}

	for _, sub := range []*Subscription{a, c} {
		got := drain(t, sub, 5, time.Second)
		for i, seq := range got {
			if seq != uint64(i+1) {
				t.Fatalf("%s: got %v, want ordered 1..5", sub.Name(), got)
			}
		}
	}
	if s := b.Stats(); s.Published != 5 || s.Subscribers != 2 {
		t.Errorf("bus stats = %+v, want Published 5 / Subscribers 2", s)
	}
}

func TestDropOldestKeepsLatest(t *testing.T) {
	b := New(Options{})
	defer b.Close()
	sub := b.Subscribe("slow", WithBuffer(4), WithPolicy(DropOldest))

	for i := uint64(1); i <= 20; i++ {
		b.Publish(ev(i))
	}
	// Let the hub finish fanning out before we read.
	waitFor(t, func() bool { return sub.Stats().Seen == 20 })

	st := sub.Stats()
	if st.Seen != 20 || st.Received != 4 || st.Dropped != 16 {
		t.Fatalf("stats = %+v, want Seen 20 / Received 4 / Dropped 16", st)
	}
	got := drain(t, sub, 4, time.Second)
	if got[len(got)-1] != 20 {
		t.Errorf("DropOldest lost the newest event: %v", got)
	}
	if b.Stats().Dropped != 16 {
		t.Errorf("bus Dropped = %d, want 16", b.Stats().Dropped)
	}
}

func TestDropNewestKeepsEarliest(t *testing.T) {
	b := New(Options{})
	defer b.Close()
	sub := b.Subscribe("slow", WithBuffer(4), WithPolicy(DropNewest))

	for i := uint64(1); i <= 20; i++ {
		b.Publish(ev(i))
	}
	waitFor(t, func() bool { return sub.Stats().Seen == 20 })

	if st := sub.Stats(); st.Received != 4 || st.Dropped != 16 {
		t.Fatalf("stats = %+v, want Received 4 / Dropped 16", st)
	}
	got := drain(t, sub, 4, time.Second)
	want := []uint64{1, 2, 3, 4}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("DropNewest kept %v, want %v", got, want)
		}
	}
}

func TestBlockPolicyDeliversEverything(t *testing.T) {
	b := New(Options{})
	defer b.Close()
	sub := b.Subscribe("primary", WithBuffer(2), WithPolicy(Block))

	const n = 200
	var got []uint64
	done := make(chan struct{})
	go func() {
		defer close(done)
		for e := range sub.C() {
			got = append(got, e.Seq)
			if len(got) == n {
				return
			}
		}
	}()

	for i := uint64(1); i <= n; i++ {
		b.Publish(ev(i)) // small buffer + Block forces lockstep, must not drop
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("only received %d/%d", len(got), n)
	}

	if sub.Stats().Dropped != 0 {
		t.Errorf("Block policy dropped %d events", sub.Stats().Dropped)
	}
	for i, seq := range got {
		if seq != uint64(i+1) {
			t.Fatalf("out of order at %d: %v", i, got[:i+1])
		}
	}
}

func TestBlockConsumerStallsHubButCloseRecovers(t *testing.T) {
	b := New(Options{Intake: 2})
	stuck := b.Subscribe("stuck", WithBuffer(1), WithPolicy(Block)) // nobody reads it

	// Push until the hub wedges on the Block sub and intake fills.
	full := false
	for i := uint64(1); i <= 50; i++ {
		if !b.TryPublish(ev(i)) {
			full = true
			break
		}
	}
	if !full {
		t.Fatal("expected TryPublish to start failing once the hub wedged")
	}

	// Unsubscribing the wedged Block consumer must free the hub.
	stuck.Unsubscribe()
	waitFor(t, func() bool { return b.TryPublish(ev(999)) })

	b.Close() // must not hang
}

func TestUnsubscribeStopsDeliveryAndClosesChannel(t *testing.T) {
	b := New(Options{})
	defer b.Close()

	keep := b.Subscribe("keep")
	drop := b.Subscribe("drop")

	b.Publish(ev(1))
	drain(t, keep, 1, time.Second)
	drain(t, drop, 1, time.Second)

	drop.Unsubscribe()
	if _, ok := <-drop.C(); ok {
		t.Fatal("channel should be closed after Unsubscribe")
	}

	b.Publish(ev(2))
	if got := drain(t, keep, 1, time.Second); got[0] != 2 {
		t.Errorf("remaining subscriber missed event: %v", got)
	}
	if s := b.Stats(); s.Subscribers != 1 {
		t.Errorf("Subscribers = %d, want 1", s.Subscribers)
	}
}

func TestUnsubscribeIdempotent(t *testing.T) {
	b := New(Options{})
	defer b.Close()
	sub := b.Subscribe("x")
	sub.Unsubscribe()
	sub.Unsubscribe() // must not panic
	sub.Unsubscribe()
}

func TestCloseIsIdempotentAndClosesAll(t *testing.T) {
	b := New(Options{})
	subs := []*Subscription{b.Subscribe("a"), b.Subscribe("b"), b.Subscribe("c")}

	b.Close()
	b.Close() // idempotent

	for _, s := range subs {
		if _, ok := <-s.C(); ok {
			t.Errorf("%s channel not closed by Close", s.Name())
		}
	}
	// Publish / TryPublish after close are inert, not panics.
	b.Publish(ev(1))
	if b.TryPublish(ev(2)) {
		t.Error("TryPublish returned true after Close")
	}
}

func TestStatsAfterCloseDoesNotHang(t *testing.T) {
	b := New(Options{})
	b.Subscribe("a")
	b.Publish(ev(1))
	b.Close()

	done := make(chan Stats, 1)
	go func() { done <- b.Stats() }()
	select {
	case s := <-done:
		if s.Published != 1 {
			t.Errorf("Published = %d, want 1", s.Published)
		}
	case <-time.After(time.Second):
		t.Fatal("Stats hung after Close")
	}
}

func TestCloseWhileHubWedgedOnBlockSub(t *testing.T) {
	b := New(Options{Intake: 2})
	b.Subscribe("stuck", WithBuffer(1), WithPolicy(Block)) // never read

	for range 10 {
		b.TryPublish(ev(1)) // wedge the hub, fill intake
	}

	done := make(chan struct{})
	go func() { b.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Close hung with the hub wedged on a Block subscriber")
	}
}

func TestSubscribeAfterCloseYieldsClosedChannel(t *testing.T) {
	b := New(Options{})
	b.Close()

	sub := b.Subscribe("late")
	if _, ok := <-sub.C(); ok {
		t.Fatal("subscription after Close should have a closed channel")
	}
	sub.Unsubscribe() // must not panic
}

func TestConcurrentPublishersAccounting(t *testing.T) {
	b := New(Options{Intake: 64})
	defer b.Close()

	const publishers, each = 8, 250
	const total = publishers * each

	subs := []*Subscription{
		b.Subscribe("big", WithBuffer(total), WithPolicy(DropOldest)),
		b.Subscribe("small", WithBuffer(16), WithPolicy(DropNewest)),
	}

	var wg sync.WaitGroup
	for p := range publishers {
		wg.Go(func() {
			for i := range each {
				b.Publish(ev(uint64(p*each + i + 1)))
			}
		})
	}
	wg.Wait()

	waitFor(t, func() bool {
		for _, s := range subs {
			st := s.Stats()
			if st.Received+st.Dropped != total {
				return false
			}
		}
		return true
	})

	if got := b.Stats().Published; got != total {
		t.Errorf("bus Published = %d, want %d", got, total)
	}
	for _, s := range subs {
		st := s.Stats()
		if st.Received+st.Dropped != total {
			t.Errorf("%s: received %d + dropped %d != %d", st.Name, st.Received, st.Dropped, total)
		}
	}
	// The big DropOldest buffer should have swallowed everything.
	if st := subs[0].Stats(); st.Dropped != 0 {
		t.Errorf("big subscriber dropped %d despite ample buffer", st.Dropped)
	}
}

func TestConcurrentSubscribeUnsubscribeDuringPublish(t *testing.T) {
	b := New(Options{})
	defer b.Close()

	stop := make(chan struct{})
	var pub sync.WaitGroup
	pub.Go(func() {
		var i uint64
		for {
			select {
			case <-stop:
				return
			default:
				i++
				b.Publish(ev(i))
			}
		}
	})

	var churn sync.WaitGroup
	for range 16 {
		churn.Go(func() {
			for range 50 {
				s := b.Subscribe(fmt.Sprintf("c-%p", new(int)))
				select {
				case <-s.C():
				default:
				}
				s.Unsubscribe()
			}
		})
	}
	churn.Wait()
	close(stop)
	pub.Wait()

	if b.Stats().Subscribers != 0 {
		t.Errorf("Subscribers = %d, want 0 after churn", b.Stats().Subscribers)
	}
}

// waitFor polls cond up to two seconds.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition not met within 2s")
}
