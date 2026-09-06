// Package httpapi wires the Parallax HTTP surface (prd.md §21.1, §21.10).
//
// The transport here is real: events are validated and normalized by
// internal/ingest, then recorded in internal/session. Intent inference is still
// a placeholder — the intelligence engine plugs into GET .../intent later.
package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/holiday-heartbreaks/parallax/backend/internal/ingest"
	"github.com/holiday-heartbreaks/parallax/backend/internal/session"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

// Deps are the collaborators a Router needs. NewRouter fills nil fields with
// production defaults.
type Deps struct {
	Logger     *slog.Logger
	Normalizer *ingest.Normalizer
	Sessions   *session.Store
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
		writeJSON(w, http.StatusOK, map[string]any{
			"status":   "ok",
			"sessions": d.Sessions.Count(),
		})
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

	return withCORS(mux)
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
