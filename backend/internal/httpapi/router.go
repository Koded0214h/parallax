// Package httpapi wires the Parallax HTTP surface for the prototype.
//
// The handlers here are deliberately thin placeholders: they define the
// request/response contracts the frontend and simulator code against, and
// keep just enough in-memory state to make the demo flow work. The real
// feature / sequence / intent / policy engines plug in behind this later.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Event is a single session or transaction event, per prd.md §33.
type Event struct {
	SessionID string         `json:"session_id"`
	UserID    string         `json:"user_id"`
	Type      string         `json:"type"`
	Timestamp int64          `json:"timestamp"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// IntentResponse is the current inference for a session, per prd.md §33.
type IntentResponse struct {
	Hypotheses  map[string]float64 `json:"hypotheses"`
	Uncertainty float64            `json:"uncertainty"`
	Action      string             `json:"action"`
	Evidence    []string           `json:"evidence"`
}

// store is a placeholder in-memory session buffer. Not the real state layer.
type store struct {
	mu     sync.RWMutex
	events map[string][]Event
}

func newStore() *store { return &store{events: make(map[string][]Event)} }

func (s *store) append(e Event) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events[e.SessionID] = append(s.events[e.SessionID], e)
	return len(s.events[e.SessionID])
}

func (s *store) count(sessionID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.events[sessionID])
}

// NewRouter builds the HTTP handler for the gateway.
func NewRouter(logger *slog.Logger) http.Handler {
	st := newStore()
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("POST /v1/events", func(w http.ResponseWriter, r *http.Request) {
		var e Event
		if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		if e.SessionID == "" || e.Type == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "session_id and type are required"})
			return
		}
		if e.Timestamp == 0 {
			e.Timestamp = time.Now().Unix()
		}
		n := st.append(e)
		logger.Info("event ingested", "session", e.SessionID, "type", e.Type, "seq", n)
		writeJSON(w, http.StatusAccepted, map[string]any{"session_id": e.SessionID, "events": n})
	})

	mux.HandleFunc("GET /v1/sessions/{id}/intent", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if st.count(id) == 0 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown session"})
			return
		}
		// Placeholder inference. Replaced by the real intent engine.
		writeJSON(w, http.StatusOK, IntentResponse{
			Hypotheses: map[string]float64{
				"legitimate":         0.25,
				"accidental":         0.25,
				"social_engineering": 0.25,
				"account_takeover":   0.25,
			},
			Uncertainty: 1.0,
			Action:      "PROBE",
			Evidence:    []string{"inference_engine_not_wired_yet"},
		})
	})

	return withCORS(mux)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// withCORS allows the Vite dev server to call the gateway during development.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
