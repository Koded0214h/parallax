// Command gateway is the Parallax event gateway and HTTP entrypoint.
//
// For the hackathon prototype it exposes:
//
//	GET  /healthz                       liveness + counters
//	GET  /v1/metrics                    bus + worker-pool counters
//	POST /v1/events                     ingest a session event
//	GET  /v1/sessions/{id}/intent       current intent inference for a session
//	GET  /v1/stream                     Server-Sent Events feed of the event stream
//
// Pipeline: ingest -> session store -> stream bus -> worker pool -> Processor.
// The Processor here is a stub; the intelligence engine replaces it.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/internal/httpapi"
	"github.com/holiday-heartbreaks/parallax/backend/internal/ingest"
	"github.com/holiday-heartbreaks/parallax/backend/internal/session"
	"github.com/holiday-heartbreaks/parallax/backend/internal/stream"
	"github.com/holiday-heartbreaks/parallax/backend/internal/worker"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	addr := envOr("PARALLAX_ADDR", ":8080")

	// Runtime pipeline.
	bus := stream.New(stream.Options{Logger: logger})

	// Stub Processor — the intelligence engine (feature / sequence / intent)
	// plugs in here.
	proc := worker.ProcessorFunc(func(_ context.Context, e contracts.Event) error {
		logger.Debug("processed", "session", e.SessionID, "type", e.Type, "seq", e.Seq)
		return nil
	})
	poolSub := bus.Subscribe("worker-pool", stream.WithPolicy(stream.Block), stream.WithBuffer(512))
	pool := worker.New(poolSub, proc, worker.Options{Logger: logger})

	router := httpapi.NewRouterWithDeps(httpapi.Deps{
		Logger:     logger,
		Normalizer: ingest.New(),
		Sessions:   session.New(),
		Bus:        bus,
		Pool:       pool,
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("gateway listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Order: stop accepting requests, drain the pool, then close the bus.
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("http shutdown failed", "err", err)
	}
	if err := pool.Stop(shutdownCtx); err != nil {
		logger.Error("pool drain incomplete", "err", err)
	}
	bus.Close()
	logger.Info("stopped", "pool", pool.Stats(), "bus", bus.Stats())
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
