// Package httpapi wires the Parallax HTTP surface (prd.md §21.1, §21.10, §33).
package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/evaluation"
	"github.com/holiday-heartbreaks/parallax/backend/internal/baseline"
	"github.com/holiday-heartbreaks/parallax/backend/internal/engine"
	"github.com/holiday-heartbreaks/parallax/backend/internal/ingest"
	"github.com/holiday-heartbreaks/parallax/backend/internal/scenarios"
	"github.com/holiday-heartbreaks/parallax/backend/internal/session"
	"github.com/holiday-heartbreaks/parallax/backend/internal/storage"
	"github.com/holiday-heartbreaks/parallax/backend/internal/stream"
	"github.com/holiday-heartbreaks/parallax/backend/internal/wal"
	"github.com/holiday-heartbreaks/parallax/backend/internal/worker"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
	"github.com/holiday-heartbreaks/parallax/backend/simulator"
)

// Deps are the collaborators a Router needs. NewRouter fills nil fields with
// production defaults.
type Deps struct {
	Logger     *slog.Logger
	Normalizer *ingest.Normalizer
	Sessions   *session.Store
	Baselines  *baseline.Store
	Bus        *stream.Bus
	Pool       *worker.Pool
	Engine     *engine.Service
	Storage    storage.Storage
	WAL        *wal.WAL
	StartTime  time.Time
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
	if d.Baselines == nil {
		d.Baselines = baseline.NewStore()
	}
	if d.Engine == nil {
		d.Engine = engine.NewService(engine.Options{
			Logger:    d.Logger,
			Sessions:  d.Sessions,
			Baselines: d.Baselines,
			Bus:       d.Bus,
		})
	}
	if d.StartTime.IsZero() {
		d.StartTime = time.Now()
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

	// Health and cron ping endpoints
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, d.snapshot("ok"))
	})

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, d.snapshot("ok"))
	})

	mux.HandleFunc("GET /v1/metrics", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, d.snapshot(""))
	})

	// Ingestion endpoint
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
		if d.WAL != nil {
			_ = d.WAL.Write(r.Context(), stored)
		}
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

	// Decision & Intent Inference endpoint
	mux.HandleFunc("GET /v1/sessions/{id}/intent", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		n := d.Sessions.Len(id)
		if n == 0 {
			writeErr(w, http.StatusNotFound, "unknown session")
			return
		}

		if d.Engine != nil {
			resp, ok := d.Engine.GetIntent(id)
			if !ok {
				var err error
				resp, err = d.Engine.EvaluateSession(id)
				if err != nil {
					writeErr(w, http.StatusInternalServerError, err.Error())
					return
				}
			}
			writeJSON(w, http.StatusOK, resp)
			return
		}

		// Fallback placeholder if no engine configured
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
			Evidence:    []string{"engine_not_configured"},
			EventsSeen:  n,
		})
	})

	// Intent Probe response endpoints (prd.md §15)
	probeHandler := func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			SessionID string `json:"session_id"`
			Response  string `json:"response"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}

		sessID := r.PathValue("id")
		if sessID == "" {
			sessID = req.SessionID
		}
		if sessID == "" || req.Response == "" {
			writeErr(w, http.StatusBadRequest, "session_id and response are required")
			return
		}

		if d.Engine == nil {
			writeErr(w, http.StatusInternalServerError, "engine not configured")
			return
		}

		updatedIntent, err := d.Engine.SubmitProbeResponse(sessID, req.Response)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, updatedIntent)
	}

	mux.HandleFunc("POST /v1/probes/respond", probeHandler)
	mux.HandleFunc("POST /v1/sessions/{id}/probe", probeHandler)

	// Scenarios catalogue & runner endpoints (prd.md §23, §24, §35)
	mux.HandleFunc("GET /v1/scenarios", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"scenarios": scenarios.List(),
		})
	})

	mux.HandleFunc("POST /v1/scenarios/{id}/run", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		sessID, events, err := scenarios.BuildEvents(id)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}

		for _, raw := range events {
			norm, normErr := d.Normalizer.Normalize(raw)
			if normErr != nil {
				continue
			}
			stored, _ := d.Sessions.Append(norm)
			if d.WAL != nil {
				_ = d.WAL.Write(r.Context(), stored)
			}
			if d.Bus != nil {
				d.Bus.Publish(stored)
			}
		}

		// Ensure evaluated before returning response
		resp, evalErr := d.Engine.EvaluateSession(sessID)
		if evalErr != nil {
			writeErr(w, http.StatusInternalServerError, evalErr.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"scenario":        id,
			"session_id":      sessID,
			"events_ingested": len(events),
			"intent":          resp,
		})
	})

	// User baseline inspection endpoint
	mux.HandleFunc("GET /v1/baselines/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		b := d.Baselines.GetOrCreate(id)
		writeJSON(w, http.StatusOK, b)
	})

	// Demo state reset endpoint
	mux.HandleFunc("POST /v1/reset", func(w http.ResponseWriter, _ *http.Request) {
		d.Sessions.Reset()
		d.Baselines.Reset()
		if d.Engine != nil {
			d.Engine.Reset()
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"message": "Demo state reset successfully",
		})
	})

	// Synthetic dataset evaluation report endpoint
	mux.HandleFunc("GET /v1/evaluation", func(w http.ResponseWriter, _ *http.Request) {
		gen := simulator.NewGenerator(42)
		dataset := gen.GenerateDataset(100, 25, 25, 25)
		ev := evaluation.NewEvaluator()
		summary := ev.Evaluate(dataset)
		writeJSON(w, http.StatusOK, summary)
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
			fmt.Fprintf(w, "data: %s\n\n", payload)
			flusher.Flush()
		}
	}
}

// snapshot builds the health and metrics payload.
func (d Deps) snapshot(status string) map[string]any {
	m := map[string]any{
		"sessions": d.Sessions.Count(),
	}
	if status != "" {
		m["status"] = status
	}
	if !d.StartTime.IsZero() {
		m["uptime_seconds"] = int64(time.Since(d.StartTime).Seconds())
	}
	if d.Bus != nil {
		m["bus"] = d.Bus.Stats()
	}
	if d.Pool != nil {
		m["pool"] = d.Pool.Stats()
	}
	if d.WAL != nil {
		m["wal"] = d.WAL.Status()
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

// withCORS allows the frontend dev server and external clients to call the gateway.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
