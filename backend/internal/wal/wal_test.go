package wal

import (
	"context"
	"testing"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/internal/storage"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

func TestWALOfflineBufferingAndReconnectReplay(t *testing.T) {
	ctx := context.Background()
	store := storage.NewMemoryStorage()
	defer store.Close()

	// Start in OFFLINE mode
	w := New(Options{
		Storage:     store,
		StartOnline: false,
		MaxBuffer:   100,
	})

	ev1 := contracts.Event{
		EventID:   "ev_offline_1",
		SessionID: "sess_offline",
		Type:      contracts.EventLogin,
		Timestamp: time.Now().Unix(),
	}
	ev2 := contracts.Event{
		EventID:   "ev_offline_2",
		SessionID: "sess_offline",
		Type:      contracts.EventAmountEntered,
		Timestamp: time.Now().Add(5 * time.Second).Unix(),
	}

	if err := w.Write(ctx, ev1); err != nil {
		t.Fatalf("write ev1 failed: %v", err)
	}
	if err := w.Write(ctx, ev2); err != nil {
		t.Fatalf("write ev2 failed: %v", err)
	}

	st := w.Status()
	if st.IsOnline {
		t.Errorf("expected WAL to be offline")
	}
	if st.Buffered != 2 {
		t.Errorf("expected 2 buffered events, got %d", st.Buffered)
	}

	// Verify storage has received 0 events so far
	storedEvs, _ := store.GetEvents(ctx, "sess_offline")
	if len(storedEvs) != 0 {
		t.Errorf("expected 0 events in store while offline, got %d", len(storedEvs))
	}

	// Simulate network reconnection
	flushed, err := w.SetOnline(ctx, true)
	if err != nil {
		t.Fatalf("SetOnline failed: %v", err)
	}
	if flushed != 2 {
		t.Errorf("expected 2 flushed events on reconnect, got %d", flushed)
	}

	stAfter := w.Status()
	if !stAfter.IsOnline {
		t.Errorf("expected WAL to be online")
	}
	if stAfter.Buffered != 0 {
		t.Errorf("expected buffer to be empty after sync, got %d", stAfter.Buffered)
	}

	// Verify all events are now safely synchronized in persistent store
	storedEvs, err = store.GetEvents(ctx, "sess_offline")
	if err != nil || len(storedEvs) != 2 {
		t.Fatalf("expected 2 events in store after reconnect replay, got %d (err: %v)", len(storedEvs), err)
	}
	if storedEvs[0].EventID != "ev_offline_1" || storedEvs[1].EventID != "ev_offline_2" {
		t.Errorf("unexpected event ordering after replay: %+v", storedEvs)
	}
}
