// Package httpapi wires the Parallax HTTP surface (prd.md §21.1, §21.10).
//
// Ingest path: POST /v1/events -> ingest.Normalizer -> session.Store -> (if a
// Bus is configured) stream.Bus. Observability: GET /v1/stream is a
// Server-Sent Events feed of the live event stream; GET /v1/metrics exposes
// bus and worker-pool counters. Intent inference is still a placeholder — the
// intelligence engine plugs into GET .../intent later.
package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/internal/ingest"
	"github.com/holiday-heartbreaks/parallax/backend/internal/session"
	"github.com/holiday-heartbreaks/parallax/backend/internal/stream"
	"github.com/holiday-heartbreaks/parallax/backend/internal/worker"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

// Deps are the collaborators a Router needs. NewRouter fills nil fields with
// production defaults. Bus and Pool are optional: without a Bus, events are
// still ingested and stored but not published, and /v1/stream is not served.
type Deps struct {
	Logger     *slog.Logger
	Normalizer *ingest.Normalizer
	Sessions   *session.Store
	Bus        *stream.Bus
	Pool       *worker.Pool
}

func (d *Deps) withDefaults() {
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	if d.Normalizer == nil {
		d.Normalizer = ingest.New()
	}
	if d.Sessions == nil {
		d.Sessions = session.New()
	}
}

// NewRouter builds the HTTP handler for the gateway.
func NewRouter(logger *slog.Logger) http.Handler {
	return NewRouterWithDeps(Deps{Logger: logger})
}

// NewRouterWithDeps builds the handler with explicit collaborators.
func NewRouterWithDeps(d Deps) http.Handler {
	d.withDefaults()
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, d.snapshot("ok"))
	})

	mux.HandleFunc("GET /v1/metrics", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, d.snapshot(""))
	})

	mux.HandleFunc("POST /v1/events", func(w http.ResponseWriter, r *http.Request) {
		var raw contracts.Event
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}

		e, err := d.Normalizer.Normalize(raw)
		if err != nil {
			if errors.Is(err, contracts.ErrInvalidEvent) {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
			d.Logger.Error("normalize failed", "err", err)
			writeErr(w, http.StatusInternalServerError, "internal error")
			return
		}

		stored, count := d.Sessions.Append(e)
		if d.Bus != nil {
			d.Bus.Publish(stored) // bounded intake => backpressure to the caller
		}
		d.Logger.Info("event ingested",
			"session", stored.SessionID, "type", stored.Type, "seq", stored.Seq)
		writeJSON(w, http.StatusAccepted, map[string]any{
			"event_id":   stored.EventID,
			"session_id": stored.SessionID,
			"seq":        stored.Seq,
			"events":     count,
		})
	})

	mux.HandleFunc("GET /v1/sessions/{id}/intent", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		n := d.Sessions.Len(id)
		if n == 0 {
			writeErr(w, http.StatusNotFound, "unknown session")
			return
		}
		// Placeholder inference. Replaced by the real intent engine (fluxx).
		writeJSON(w, http.StatusOK, contracts.IntentResponse{
			SessionID: id,
			Hypotheses: map[string]float64{
				contracts.IntentLegitimate:        0.25,
				contracts.IntentAccidental:        0.25,
				contracts.IntentSocialEngineering: 0.25,
				contracts.IntentAccountTakeover:   0.25,
			},
			Uncertainty: 1.0,
			Action:      contracts.ActionProbe,
			Evidence:    []string{"inference_engine_not_wired_yet"},
			EventsSeen:  n,
		})
	})

	if d.Bus != nil {
		mux.HandleFunc("GET /v1/stream", d.handleStream)
	}

	return withCORS(mux)
}

// handleStream is a Server-Sent Events feed of the live event stream. An
// optional ?session_id= filters to one session. Each client gets its own
// DropOldest subscription, so a slow reader only lags itself.
func (d Deps) handleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	filter := r.URL.Query().Get("session_id")

	sub := d.Bus.Subscribe("sse:"+r.RemoteAddr,
		stream.WithPolicy(stream.DropOldest), stream.WithBuffer(256))
	defer sub.Unsubscribe()

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	ping := time.NewTicker(15 * time.Second)
	defer ping.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		case e, ok := <-sub.C():
			if !ok {
				return
			}
			if filter != "" && e.SessionID != filter {
				continue
			}
			payload, err := json.Marshal(e)
			if err != nil {
				continue
			}
			// Unnamed event so the browser's EventSource.onmessage receives it;
			// the type is in the payload.
			fmt.Fprintf(w, "data: %s\n\n", payload)
			flusher.Flush()
		}
	}
}

// snapshot builds the /healthz and /v1/metrics body. status is included when
// non-empty.
func (d Deps) snapshot(status string) map[string]any {
	m := map[string]any{"sessions": d.Sessions.Count()}
	if status != "" {
		m["status"] = status
	}
	if d.Bus != nil {
		m["bus"] = d.Bus.Stats()
	}
	if d.Pool != nil {
		m["pool"] = d.Pool.Stats()
	}
	return m
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
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
