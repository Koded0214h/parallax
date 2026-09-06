// Package ingest is the front of the event pipeline (prd.md §21.1). It takes a
// client-supplied event, validates its structure, normalizes it, and stamps the
// server-owned fields the rest of the system relies on.
//
// It does NOT assign the per-session sequence number — that belongs to
// internal/session, which owns the per-session lock.
package ingest

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

// Normalizer validates and canonicalizes incoming events. The zero value is not
// usable; construct one with New.
type Normalizer struct {
	now   func() time.Time
	newID func() string
}

// Option customizes a Normalizer (used mainly by tests).
type Option func(*Normalizer)

// WithClock overrides the time source.
func WithClock(now func() time.Time) Option { return func(n *Normalizer) { n.now = now } }

// WithIDGen overrides the event-id generator.
func WithIDGen(gen func() string) Option { return func(n *Normalizer) { n.newID = gen } }

// New returns a Normalizer with real time and random ids.
func New(opts ...Option) *Normalizer {
	n := &Normalizer{now: time.Now, newID: randomID}
	for _, o := range opts {
		o(n)
	}
	return n
}

// Normalize returns a canonical copy of e, or an error wrapping
// contracts.ErrInvalidEvent if the client fields are invalid.
//
// Guarantees on the returned event:
//   - Type is trimmed and upper-cased, and is a known type
//   - EventID is set (generated if absent)
//   - Timestamp is set (server receive time in unix seconds if absent)
//   - IngestedAtNanos is set to the server receive time
//   - SessionID and UserID are trimmed
func (n *Normalizer) Normalize(e contracts.Event) (contracts.Event, error) {
	e.SessionID = strings.TrimSpace(e.SessionID)
	e.UserID = strings.TrimSpace(e.UserID)
	e.Type = contracts.EventType(strings.ToUpper(strings.TrimSpace(string(e.Type))))

	if err := e.ValidateForIngest(); err != nil {
		return contracts.Event{}, err
	}

	now := n.now()
	if e.EventID == "" {
		e.EventID = n.newID()
	}
	if e.Timestamp == 0 {
		e.Timestamp = now.Unix()
	}
	e.IngestedAtNanos = now.UnixNano()
	e.Seq = 0 // assigned by the session store on append
	return e, nil
}

func randomID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return "evt_" + hex.EncodeToString(b[:])
}
