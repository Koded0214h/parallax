package wal

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/internal/storage"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

var (
	ErrWALClosed = errors.New("wal is closed")
)

// Status represents the operational health and sync metrics of the WAL (prd.md §28).
type Status struct {
	IsOnline     bool   `json:"is_online"`
	Buffered     int    `json:"buffered_events"`
	SyncedTotal  int    `json:"synced_total"`
	LastSyncTime int64  `json:"last_sync_time,omitempty"`
	Mode         string `json:"mode"`
}

// WAL provides offline event buffering and reconnect replay synchronization.
type WAL struct {
	mu           sync.RWMutex
	store        storage.Storage
	buffer       []contracts.Event
	maxBuffer    int
	isOnline     bool
	syncedTotal  int
	lastSyncTime time.Time
	closed       bool
}

// Options configure the Write-Ahead Log buffer.
type Options struct {
	Storage      storage.Storage
	MaxBuffer    int
	StartOnline  bool
}

// New constructs a WAL instance.
func New(opts Options) *WAL {
	maxBuf := opts.MaxBuffer
	if maxBuf <= 0 {
		maxBuf = 10000
	}
	return &WAL{
		store:     opts.Storage,
		buffer:    make([]contracts.Event, 0, 128),
		maxBuffer: maxBuf,
		isOnline:  opts.StartOnline,
	}
}

// Write appends an event to the WAL. If online and storage is present, flushes directly;
// if offline, buffers the event safely until reconnect.
func (w *WAL) Write(ctx context.Context, e contracts.Event) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return ErrWALClosed
	}

	if w.isOnline && w.store != nil {
		if err := w.store.SaveEvent(ctx, e); err == nil {
			w.syncedTotal++
			w.lastSyncTime = time.Now().UTC()
			return nil
		}
		// If save to remote storage failed due to network glitch, automatically degrade to offline buffering
		w.isOnline = false
	}

	// Offline buffering
	if len(w.buffer) >= w.maxBuffer {
		// Drop oldest if ring exceeds bounded capacity
		w.buffer = w.buffer[1:]
	}
	w.buffer = append(w.buffer, e)
	return nil
}

// SetOnline sets the network connectivity state. When set to true, triggers reconnect synchronization.
func (w *WAL) SetOnline(ctx context.Context, online bool) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return 0, ErrWALClosed
	}

	w.isOnline = online
	if online && len(w.buffer) > 0 && w.store != nil {
		return w.flushLocked(ctx)
	}
	return 0, nil
}

// Flush synchronizes all buffered events to persistent storage.
func (w *WAL) Flush(ctx context.Context) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return 0, ErrWALClosed
	}
	if w.store == nil {
		return 0, errors.New("no destination storage configured")
	}

	return w.flushLocked(ctx)
}

func (w *WAL) flushLocked(ctx context.Context) (int, error) {
	if len(w.buffer) == 0 {
		return 0, nil
	}

	flushed := 0
	for len(w.buffer) > 0 {
		e := w.buffer[0]
		if err := w.store.SaveEvent(ctx, e); err != nil {
			return flushed, fmt.Errorf("sync failed at event %s: %w", e.EventID, err)
		}
		w.buffer = w.buffer[1:]
		flushed++
		w.syncedTotal++
	}

	w.lastSyncTime = time.Now().UTC()
	return flushed, nil
}

// Status returns the current synchronization state.
func (w *WAL) Status() Status {
	w.mu.RLock()
	defer w.mu.RUnlock()

	mode := "OFFLINE_BUFFERING"
	if w.isOnline {
		mode = "ONLINE_SYNCHRONIZED"
	}

	var lastSync int64
	if !w.lastSyncTime.IsZero() {
		lastSync = w.lastSyncTime.Unix()
	}

	return Status{
		IsOnline:     w.isOnline,
		Buffered:     len(w.buffer),
		SyncedTotal:  w.syncedTotal,
		LastSyncTime: lastSync,
		Mode:         mode,
	}
}

// Close flushes if online and closes the WAL.
func (w *WAL) Close(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}
	w.closed = true
	if w.isOnline && w.store != nil && len(w.buffer) > 0 {
		_, _ = w.flushLocked(ctx)
	}
	return nil
}
