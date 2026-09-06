package session

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

func evt(sessionID, user string, typ contracts.EventType) contracts.Event {
	return contracts.Event{SessionID: sessionID, UserID: user, Type: typ}
}

func TestAppendAssignsSequence(t *testing.T) {
	s := New()
	for i := 1; i <= 3; i++ {
		stored, count := s.Append(evt("s1", "u1", contracts.EventLogin))
		if stored.Seq != uint64(i) {
			t.Errorf("append %d: seq = %d, want %d", i, stored.Seq, i)
		}
		if count != i {
			t.Errorf("append %d: count = %d, want %d", i, count, i)
		}
	}
	if s.Len("s1") != 3 {
		t.Errorf("Len = %d, want 3", s.Len("s1"))
	}
	if s.Count() != 1 {
		t.Errorf("Count = %d, want 1", s.Count())
	}
}

func TestSequenceIsPerSession(t *testing.T) {
	s := New()
	s.Append(evt("s1", "u1", contracts.EventLogin))
	s.Append(evt("s2", "u2", contracts.EventLogin))
	stored, _ := s.Append(evt("s2", "u2", contracts.EventOTPVerified))
	if stored.Seq != 2 {
		t.Errorf("s2 second event seq = %d, want 2", stored.Seq)
	}
	first, _ := s.Trajectory("s1")
	if first[0].Seq != 1 {
		t.Errorf("s1 first event seq = %d, want 1", first[0].Seq)
	}
}

func TestTrajectoryIsACopy(t *testing.T) {
	s := New()
	s.Append(evt("s1", "u1", contracts.EventLogin))
	tr, ok := s.Trajectory("s1")
	if !ok || len(tr) != 1 {
		t.Fatalf("Trajectory = %v, %v", tr, ok)
	}
	tr[0].Type = "MUTATED"
	again, _ := s.Trajectory("s1")
	if again[0].Type != contracts.EventLogin {
		t.Error("Trajectory returned a mutable reference to internal state")
	}
}

func TestSnapshotIsImmutableAndComplete(t *testing.T) {
	s := New()
	s.Append(evt("s1", "u1", contracts.EventLogin))
	s.Append(evt("s1", "u1", contracts.EventAmountEntered))

	snap, ok := s.Snapshot("s1")
	if !ok {
		t.Fatal("Snapshot !ok for known session")
	}
	if snap.ID != "s1" || snap.UserID != "u1" || len(snap.Events) != 2 {
		t.Errorf("unexpected snapshot: %+v", snap)
	}
	if snap.CreatedAt.IsZero() || snap.LastSeen.IsZero() {
		t.Errorf("timestamps not set: %+v", snap)
	}

	snap.Events[0].Type = "MUTATED"
	s.Append(evt("s1", "u1", contracts.EventOTPVerified))
	fresh, _ := s.Snapshot("s1")
	if fresh.Events[0].Type != contracts.EventLogin {
		t.Error("mutating a snapshot leaked into the store")
	}
	if len(fresh.Events) != 3 {
		t.Errorf("store did not keep growing: %d events", len(fresh.Events))
	}
}

func TestLastSeenAdvances(t *testing.T) {
	tick := time.Unix(1_000, 0)
	s := New()
	s.now = func() time.Time { return tick }
	s.Append(evt("s1", "u1", contracts.EventLogin))
	created, _ := s.Snapshot("s1")

	tick = tick.Add(30 * time.Second)
	s.Append(evt("s1", "u1", contracts.EventAmountEntered))
	later, _ := s.Snapshot("s1")

	if !later.LastSeen.After(created.LastSeen) {
		t.Errorf("LastSeen did not advance: %v -> %v", created.LastSeen, later.LastSeen)
	}
	if !later.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt changed: %v -> %v", created.CreatedAt, later.CreatedAt)
	}
}

func TestUserIDBackfilled(t *testing.T) {
	s := New()
	s.Append(contracts.Event{SessionID: "s1", Type: contracts.EventLogin}) // no user yet
	s.Append(evt("s1", "u1", contracts.EventAmountEntered))
	snap, _ := s.Snapshot("s1")
	if snap.UserID != "u1" {
		t.Errorf("UserID not backfilled: %q", snap.UserID)
	}
}

func TestUnknownSession(t *testing.T) {
	s := New()
	if s.Len("nope") != 0 {
		t.Error("Len of unknown session should be 0")
	}
	if _, ok := s.Snapshot("nope"); ok {
		t.Error("Snapshot of unknown session should report !ok")
	}
	if _, ok := s.Trajectory("nope"); ok {
		t.Error("Trajectory of unknown session should report !ok")
	}
}

func TestNewWithShardsClamps(t *testing.T) {
	for _, n := range []int{-5, 0, 1} {
		s := NewWithShards(n)
		if len(s.shards) < 1 {
			t.Errorf("NewWithShards(%d) produced %d shards", n, len(s.shards))
		}
		s.Append(evt("s1", "u1", contracts.EventLogin))
		if s.Len("s1") != 1 {
			t.Errorf("NewWithShards(%d): store unusable", n)
		}
	}
}

func TestConcurrentAppendsDistinctSessions(t *testing.T) {
	s := NewWithShards(8)
	const sessions, perSession = 50, 20
	var wg sync.WaitGroup
	for i := range sessions {
		wg.Go(func() {
			id := fmt.Sprintf("s%d", i)
			for range perSession {
				s.Append(evt(id, "u", contracts.EventLogin))
			}
		})
	}
	wg.Wait()

	if s.Count() != sessions {
		t.Errorf("Count = %d, want %d", s.Count(), sessions)
	}
	for i := range sessions {
		if got := s.Len(fmt.Sprintf("s%d", i)); got != perSession {
			t.Errorf("s%d Len = %d, want %d", i, got, perSession)
		}
	}
}

func TestConcurrentAppendsSameSessionAreSerialized(t *testing.T) {
	s := New()
	const writers, each = 16, 50
	var wg sync.WaitGroup
	for range writers {
		wg.Go(func() {
			for range each {
				s.Append(evt("hot", "u1", contracts.EventLogin))
			}
		})
	}
	wg.Wait()

	tr, _ := s.Trajectory("hot")
	if len(tr) != writers*each {
		t.Fatalf("event count = %d, want %d", len(tr), writers*each)
	}
	seen := make(map[uint64]bool, len(tr))
	for _, e := range tr {
		if e.Seq < 1 || e.Seq > uint64(writers*each) {
			t.Fatalf("seq out of range: %d", e.Seq)
		}
		if seen[e.Seq] {
			t.Fatalf("duplicate seq under concurrency: %d", e.Seq)
		}
		seen[e.Seq] = true
	}
}
