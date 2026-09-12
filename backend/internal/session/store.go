// Package session holds Parallax's in-memory session state: the ordered event
// trajectory for each session, its per-session sequence counter, and lightweight
// lifecycle metadata.
//
// State is sharded by session id so concurrent sessions rarely contend on the
// same lock (prd.md §19: "lock-efficient state access"). Nothing here is
// persistent; durability is the job of internal/wal and internal/storage.
package session

import (
	"hash/fnv"
	"sort"
	"sync"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

// DefaultShards is the shard count used by New. Must be a power of two is not
// required, but keeping it modest keeps memory predictable for the prototype.
const DefaultShards = 32

// Session is a single session's accumulated state.
type Session struct {
	ID        string
	UserID    string
	CreatedAt time.Time
	LastSeen  time.Time
	Events    []contracts.Event
}

// View is an immutable snapshot of a session, safe to hand to callers outside
// the store's lock.
type View struct {
	ID        string
	UserID    string
	CreatedAt time.Time
	LastSeen  time.Time
	Events    []contracts.Event
}

type shard struct {
	mu sync.RWMutex
	m  map[string]*Session
}

// Store is a sharded, concurrency-safe collection of sessions.
type Store struct {
	shards []*shard
	now    func() time.Time
}

// New creates a Store with DefaultShards shards.
func New() *Store { return NewWithShards(DefaultShards) }

// NewWithShards creates a Store with n shards (n is clamped to >= 1).
func NewWithShards(n int) *Store {
	if n < 1 {
		n = 1
	}
	s := &Store{shards: make([]*shard, n), now: time.Now}
	for i := range s.shards {
		s.shards[i] = &shard{m: make(map[string]*Session)}
	}
	return s
}

func (s *Store) shardFor(sessionID string) *shard {
	h := fnv.New32a()
	_, _ = h.Write([]byte(sessionID))
	return s.shards[h.Sum32()%uint32(len(s.shards))]
}

// Append records e against its session, creating the session on first sight.
// It assigns e.Seq (1-based, per session) and returns the stored copy along
// with the new event count for that session.
func (s *Store) Append(e contracts.Event) (contracts.Event, int) {
	sh := s.shardFor(e.SessionID)
	sh.mu.Lock()
	defer sh.mu.Unlock()

	sess := sh.m[e.SessionID]
	if sess == nil {
		sess = &Session{ID: e.SessionID, UserID: e.UserID, CreatedAt: s.now()}
		sh.m[e.SessionID] = sess
	}
	sess.LastSeen = s.now()
	if sess.UserID == "" {
		sess.UserID = e.UserID
	}

	e.Seq = uint64(len(sess.Events)) + 1
	sess.Events = append(sess.Events, e)
	return e, len(sess.Events)
}

// Len returns the number of events recorded for sessionID (0 if unknown).
func (s *Store) Len(sessionID string) int {
	sh := s.shardFor(sessionID)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	if sess := sh.m[sessionID]; sess != nil {
		return len(sess.Events)
	}
	return 0
}

// Trajectory returns a copy of the ordered events for sessionID.
func (s *Store) Trajectory(sessionID string) ([]contracts.Event, bool) {
	sh := s.shardFor(sessionID)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	sess := sh.m[sessionID]
	if sess == nil {
		return nil, false
	}
	out := make([]contracts.Event, len(sess.Events))
	copy(out, sess.Events)
	return out, true
}

// Snapshot returns an immutable View of sessionID.
func (s *Store) Snapshot(sessionID string) (View, bool) {
	sh := s.shardFor(sessionID)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	sess := sh.m[sessionID]
	if sess == nil {
		return View{}, false
	}
	events := make([]contracts.Event, len(sess.Events))
	copy(events, sess.Events)
	return View{
		ID:        sess.ID,
		UserID:    sess.UserID,
		CreatedAt: sess.CreatedAt,
		LastSeen:  sess.LastSeen,
		Events:    events,
	}, true
}

// List returns up to limit sessions across all shards, most-recently-active
// first. It copies each session's event slice, same as Snapshot, so callers
// can't mutate store state. limit <= 0 means "all".
func (s *Store) List(limit int) []View {
	var all []View
	for _, sh := range s.shards {
		sh.mu.RLock()
		for _, sess := range sh.m {
			events := make([]contracts.Event, len(sess.Events))
			copy(events, sess.Events)
			all = append(all, View{
				ID:        sess.ID,
				UserID:    sess.UserID,
				CreatedAt: sess.CreatedAt,
				LastSeen:  sess.LastSeen,
				Events:    events,
			})
		}
		sh.mu.RUnlock()
	}
	sort.Slice(all, func(i, j int) bool { return all[i].LastSeen.After(all[j].LastSeen) })
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all
}

// Count returns the total number of live sessions across all shards.
func (s *Store) Count() int {
	total := 0
	for _, sh := range s.shards {
		sh.mu.RLock()
		total += len(sh.m)
		sh.mu.RUnlock()
	}
	return total
}

// Reset clears all sessions across all shards.
func (s *Store) Reset() {
	for _, sh := range s.shards {
		sh.mu.Lock()
		sh.m = make(map[string]*Session)
		sh.mu.Unlock()
	}
}
