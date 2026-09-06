package ingest

import (
	"errors"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

func fixedNormalizer() *Normalizer {
	ts := time.Unix(1_788_594_121, 500)
	return New(
		WithClock(func() time.Time { return ts }),
		WithIDGen(func() string { return "evt_fixed" }),
	)
}

func TestNormalizeFillsServerFields(t *testing.T) {
	n := fixedNormalizer()
	got, err := n.Normalize(contracts.Event{
		SessionID: "  s1 ", UserID: " u1 ", Type: contracts.EventType("  login "),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.SessionID != "s1" || got.UserID != "u1" {
		t.Errorf("fields not trimmed: %+v", got)
	}
	if got.Type != contracts.EventLogin {
		t.Errorf("type not canonicalized: %q", got.Type)
	}
	if got.EventID != "evt_fixed" {
		t.Errorf("event id not generated: %q", got.EventID)
	}
	if got.Timestamp != 1_788_594_121 {
		t.Errorf("timestamp not defaulted: %d", got.Timestamp)
	}
	if got.IngestedAtNanos != time.Unix(1_788_594_121, 500).UnixNano() {
		t.Errorf("ingested_at not stamped: %d", got.IngestedAtNanos)
	}
	if got.Seq != 0 {
		t.Errorf("seq must be left for the session store, got %d", got.Seq)
	}
}

func TestNormalizeKeepsClientTimestampAndID(t *testing.T) {
	n := fixedNormalizer()
	got, err := n.Normalize(contracts.Event{
		EventID: "evt_client", SessionID: "s1", UserID: "u1",
		Type: contracts.EventLogin, Timestamp: 42,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.EventID != "evt_client" || got.Timestamp != 42 {
		t.Errorf("client-supplied id/timestamp overwritten: %+v", got)
	}
}

func TestNormalizePreservesMetadata(t *testing.T) {
	n := fixedNormalizer()
	got, err := n.Normalize(contracts.Event{
		SessionID: "s1", UserID: "u1", Type: contracts.EventAmountEntered,
		Metadata: map[string]any{"amount": 450000.0, "currency": "NGN"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Metadata["amount"] != 450000.0 || got.Metadata["currency"] != "NGN" {
		t.Errorf("metadata not preserved: %+v", got.Metadata)
	}
}

func TestNormalizeDoesNotMutateInput(t *testing.T) {
	n := fixedNormalizer()
	in := contracts.Event{SessionID: " s1 ", UserID: "u1", Type: contracts.EventType("login")}
	_, err := n.Normalize(in)
	if err != nil {
		t.Fatal(err)
	}
	if in.SessionID != " s1 " || in.Type != contracts.EventType("login") || in.EventID != "" {
		t.Errorf("Normalize mutated its argument: %+v", in)
	}
}

func TestNormalizeRejectsInvalid(t *testing.T) {
	n := fixedNormalizer()
	cases := map[string]contracts.Event{
		"no user":       {SessionID: "s1", Type: contracts.EventLogin},
		"blank session": {SessionID: "   ", UserID: "u1", Type: contracts.EventLogin},
		"unknown type":  {SessionID: "s1", UserID: "u1", Type: contracts.EventType("frobnicate")},
		"negative ts":   {SessionID: "s1", UserID: "u1", Type: contracts.EventLogin, Timestamp: -5},
	}
	for name, e := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := n.Normalize(e)
			if !errors.Is(err, contracts.ErrInvalidEvent) {
				t.Fatalf("want ErrInvalidEvent, got %v", err)
			}
			if got.SessionID != "" || got.EventID != "" || got.Type != "" || got.IngestedAtNanos != 0 {
				t.Errorf("expected zero Event on error, got %+v", got)
			}
		})
	}
}

func TestRandomIDFormatAndUniqueness(t *testing.T) {
	n := New() // real id generator
	re := regexp.MustCompile(`^evt_[0-9a-f]{24}$`)
	seen := make(map[string]struct{}, 1000)
	for range 1000 {
		got, err := n.Normalize(contracts.Event{SessionID: "s1", UserID: "u1", Type: contracts.EventLogin})
		if err != nil {
			t.Fatal(err)
		}
		if !re.MatchString(got.EventID) {
			t.Fatalf("id %q does not match %s", got.EventID, re)
		}
		if _, dup := seen[got.EventID]; dup {
			t.Fatalf("duplicate id generated: %s", got.EventID)
		}
		seen[got.EventID] = struct{}{}
	}
}

func TestNormalizeConcurrent(t *testing.T) {
	n := New()
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			for range 100 {
				if _, err := n.Normalize(contracts.Event{
					SessionID: "s1", UserID: "u1", Type: contracts.EventLogin,
				}); err != nil {
					t.Errorf("concurrent normalize failed: %v", err)
					return
				}
			}
		})
	}
	wg.Wait()
}
