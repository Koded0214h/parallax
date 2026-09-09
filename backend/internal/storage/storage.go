package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

var (
	ErrNotFound = errors.New("not found")
	ErrClosed   = errors.New("storage closed")
)

// Storage defines the persistence contract for events and decisions (docs/backend.md M8).
type Storage interface {
	SaveEvent(ctx context.Context, e contracts.Event) error
	SaveEvents(ctx context.Context, events []contracts.Event) error
	GetEvents(ctx context.Context, sessionID string) ([]contracts.Event, error)
	SaveDecision(ctx context.Context, resp contracts.IntentResponse) error
	GetDecision(ctx context.Context, sessionID string) (contracts.IntentResponse, bool, error)
	Close() error
}

// MemoryStorage is a fast, lock-guarded in-memory implementation.
type MemoryStorage struct {
	mu        sync.RWMutex
	events    map[string][]contracts.Event
	decisions map[string]contracts.IntentResponse
	closed    bool
}

// NewMemoryStorage creates an in-memory storage instance.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		events:    make(map[string][]contracts.Event),
		decisions: make(map[string]contracts.IntentResponse),
	}
}

func (s *MemoryStorage) SaveEvent(_ context.Context, e contracts.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	s.events[e.SessionID] = append(s.events[e.SessionID], e)
	return nil
}

func (s *MemoryStorage) SaveEvents(_ context.Context, events []contracts.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	for _, e := range events {
		s.events[e.SessionID] = append(s.events[e.SessionID], e)
	}
	return nil
}

func (s *MemoryStorage) GetEvents(_ context.Context, sessionID string) ([]contracts.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return nil, ErrClosed
	}
	evs, ok := s.events[sessionID]
	if !ok {
		return nil, nil
	}
	cp := make([]contracts.Event, len(evs))
	copy(cp, evs)
	return cp, nil
}

func (s *MemoryStorage) SaveDecision(_ context.Context, resp contracts.IntentResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	s.decisions[resp.SessionID] = resp
	return nil
}

func (s *MemoryStorage) GetDecision(_ context.Context, sessionID string) (contracts.IntentResponse, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return contracts.IntentResponse{}, false, ErrClosed
	}
	d, ok := s.decisions[sessionID]
	return d, ok, nil
}

func (s *MemoryStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

// FileStorage provides durable, append-only JSON-L on-disk persistence.
type FileStorage struct {
	mu           sync.RWMutex
	dir          string
	eventFile    *os.File
	eventWriter  *bufio.Writer
	decisionFile *os.File
	indexEvents  map[string][]contracts.Event
	indexDecs    map[string]contracts.IntentResponse
	closed       bool
}

// NewFileStorage initializes disk-backed storage at the specified directory.
func NewFileStorage(dir string) (*FileStorage, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage dir: %w", err)
	}

	evPath := filepath.Join(dir, "events.jsonl")
	decPath := filepath.Join(dir, "decisions.jsonl")

	evFile, err := os.OpenFile(evPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open events file: %w", err)
	}

	decFile, err := os.OpenFile(decPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		evFile.Close()
		return nil, fmt.Errorf("failed to open decisions file: %w", err)
	}

	fs := &FileStorage{
		dir:          dir,
		eventFile:    evFile,
		eventWriter:  bufio.NewWriter(evFile),
		decisionFile: decFile,
		indexEvents:  make(map[string][]contracts.Event),
		indexDecs:    make(map[string]contracts.IntentResponse),
	}

	// Warm in-memory indexes by replaying existing records
	if err := fs.recoverRecords(); err != nil {
		fs.Close()
		return nil, fmt.Errorf("storage recovery failed: %w", err)
	}

	return fs, nil
}

func (f *FileStorage) recoverRecords() error {
	if _, err := f.eventFile.Seek(0, 0); err != nil {
		return err
	}
	scanner := bufio.NewScanner(f.eventFile)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e contracts.Event
		if err := json.Unmarshal(line, &e); err == nil {
			f.indexEvents[e.SessionID] = append(f.indexEvents[e.SessionID], e)
		}
	}

	if _, err := f.decisionFile.Seek(0, 0); err != nil {
		return err
	}
	decScanner := bufio.NewScanner(f.decisionFile)
	for decScanner.Scan() {
		line := decScanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var d contracts.IntentResponse
		if err := json.Unmarshal(line, &d); err == nil {
			f.indexDecs[d.SessionID] = d
		}
	}

	return nil
}

func (f *FileStorage) SaveEvent(_ context.Context, e contracts.Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return ErrClosed
	}

	bytes, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if _, err := f.eventWriter.Write(append(bytes, '\n')); err != nil {
		return err
	}
	if err := f.eventWriter.Flush(); err != nil {
		return err
	}

	f.indexEvents[e.SessionID] = append(f.indexEvents[e.SessionID], e)
	return nil
}

func (f *FileStorage) SaveEvents(_ context.Context, events []contracts.Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return ErrClosed
	}

	for _, e := range events {
		bytes, err := json.Marshal(e)
		if err != nil {
			return err
		}
		if _, err := f.eventWriter.Write(append(bytes, '\n')); err != nil {
			return err
		}
		f.indexEvents[e.SessionID] = append(f.indexEvents[e.SessionID], e)
	}

	return f.eventWriter.Flush()
}

func (f *FileStorage) GetEvents(_ context.Context, sessionID string) ([]contracts.Event, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.closed {
		return nil, ErrClosed
	}

	evs := f.indexEvents[sessionID]
	cp := make([]contracts.Event, len(evs))
	copy(cp, evs)
	return cp, nil
}

func (f *FileStorage) SaveDecision(_ context.Context, resp contracts.IntentResponse) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return ErrClosed
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	if _, err := f.decisionFile.Write(append(bytes, '\n')); err != nil {
		return err
	}

	f.indexDecs[resp.SessionID] = resp
	return nil
}

func (f *FileStorage) GetDecision(_ context.Context, sessionID string) (contracts.IntentResponse, bool, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.closed {
		return contracts.IntentResponse{}, false, ErrClosed
	}

	d, ok := f.indexDecs[sessionID]
	return d, ok, nil
}

func (f *FileStorage) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return nil
	}
	f.closed = true
	_ = f.eventWriter.Flush()
	_ = f.eventFile.Close()
	_ = f.decisionFile.Close()
	return nil
}
