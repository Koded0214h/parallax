package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

func TestMemoryStorage(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStorage()
	defer s.Close()

	ev := contracts.Event{
		EventID:   "ev_1",
		SessionID: "sess_1",
		UserID:    "user_001",
		Type:      contracts.EventLogin,
		Timestamp: time.Now().Unix(),
	}

	if err := s.SaveEvent(ctx, ev); err != nil {
		t.Fatalf("save event failed: %v", err)
	}

	evs, err := s.GetEvents(ctx, "sess_1")
	if err != nil || len(evs) != 1 {
		t.Fatalf("expected 1 event, got %v (err: %v)", len(evs), err)
	}

	dec := contracts.IntentResponse{
		SessionID:   "sess_1",
		Action:      contracts.ActionAllow,
		Uncertainty: 0.1,
	}
	if err := s.SaveDecision(ctx, dec); err != nil {
		t.Fatalf("save decision failed: %v", err)
	}

	gotDec, ok, err := s.GetDecision(ctx, "sess_1")
	if err != nil || !ok || gotDec.Action != contracts.ActionAllow {
		t.Fatalf("expected decision retrieved, got %+v", gotDec)
	}
}

func TestFileStorageDurablePersistence(t *testing.T) {
	ctx := context.Background()
	tmpDir, err := os.MkdirTemp("", "parallax_storage_test_*")
	if err != nil {
		t.Fatalf("mktemp failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fs, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStorage failed: %v", err)
	}

	ev := contracts.Event{
		EventID:   "ev_disk_1",
		SessionID: "sess_disk",
		UserID:    "user_disk",
		Type:      contracts.EventLogin,
		Timestamp: time.Now().Unix(),
	}

	if err := fs.SaveEvent(ctx, ev); err != nil {
		t.Fatalf("save event failed: %v", err)
	}
	_ = fs.Close()

	// Reopen to verify durable recovery from disk
	fs2, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("reopen NewFileStorage failed: %v", err)
	}
	defer fs2.Close()

	evs, err := fs2.GetEvents(ctx, "sess_disk")
	if err != nil || len(evs) != 1 || evs[0].EventID != "ev_disk_1" {
		t.Fatalf("expected recovered event, got %+v", evs)
	}
}
