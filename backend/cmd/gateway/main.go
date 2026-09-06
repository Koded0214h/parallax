// Command gateway is the Parallax event gateway and HTTP entrypoint.
//
// For the hackathon prototype it exposes a minimal API surface:
//
//	GET  /healthz                       liveness probe
//	POST /v1/events                     ingest a session event
//	GET  /v1/sessions/{id}/intent       current intent inference for a session
//
// Everything runs in-memory. See docs/ and prd.md for the full design.
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
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	addr := envOr("PARALLAX_ADDR", ":8080")

	// Runtime state and pipeline collaborators. The intent engine plugs in here later.
	router := httpapi.NewRouterWithDeps(httpapi.Deps{
		Logger:     logger,
		Normalizer: ingest.New(),
		Sessions:   session.New(),
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
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "err", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
