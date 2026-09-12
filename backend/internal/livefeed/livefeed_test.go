package livefeed

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/internal/ingest"
	"github.com/holiday-heartbreaks/parallax/backend/internal/session"
)

func TestFeedProducesLiveSessions(t *testing.T) {
	sessions := session.New()
	f := New(Options{
		Normalizer:    ingest.New(),
		Sessions:      sessions,
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		Seed:          1,
		MinSessionGap: time.Millisecond,
		MaxSessionGap: 2 * time.Millisecond,
		MinEventGap:   time.Millisecond,
		MaxEventGap:   2 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	f.Run(ctx) // blocks until ctx expires

	if sessions.Count() == 0 {
		t.Fatal("live feed produced no sessions")
	}
	for _, v := range sessions.List(0) {
		if len(v.Events) == 0 {
			t.Errorf("session %s has no events", v.ID)
		}
		if v.UserID == "" {
			t.Errorf("session %s has no user id", v.ID)
		}
	}
}

func TestFeedStopsOnContextCancel(t *testing.T) {
	sessions := session.New()
	f := New(Options{
		Normalizer:    ingest.New(),
		Sessions:      sessions,
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		MinSessionGap: time.Hour, // would never tick within the test
		MaxSessionGap: 2 * time.Hour,
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		f.Run(ctx)
		close(done)
	}()
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return promptly after context cancel")
	}
}

func TestRewriteIdentityMakesEventsConsistent(t *testing.T) {
	f := New(Options{Normalizer: ingest.New(), Sessions: session.New(), Seed: 42})
	_, sess := f.generate()

	if sess.SessionID == "" {
		t.Fatal("empty session id")
	}
	for _, e := range sess.Events {
		if e.SessionID != sess.SessionID {
			t.Errorf("event session id %q != session id %q", e.SessionID, sess.SessionID)
		}
		if e.UserID != sess.UserID {
			t.Errorf("event user id %q != session user %q", e.UserID, sess.UserID)
		}
	}
}
