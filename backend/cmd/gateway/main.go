// Command gateway is the Parallax event gateway and HTTP entrypoint.
//
// Exposes:
//
//	GET  /health                        liveness + uptime + counters (for monitoring & Render cron)
//	GET  /healthz                       standard health check
//	GET  /v1/metrics                    bus, worker pool, and WAL metrics
//	POST /v1/events                     ingest a session event
//	GET  /v1/sessions                   recent sessions with their last-known decision
//	GET  /v1/sessions/{id}/events       raw event trajectory for a session
//	GET  /v1/sessions/{id}/intent       real-time intent inference for a session
//	POST /v1/probes/respond             submit intent probe contextual response
//	POST /v1/sessions/{id}/probe        submit probe response by session path
//	GET  /v1/scenarios                  list available demo scenarios
//	POST /v1/scenarios/{id}/run         execute scripted demo scenario
//	GET  /v1/baselines/{id}             inspect user baseline
//	POST /v1/reset                      reset demo state and baselines
//	GET  /v1/evaluation                 audit detection accuracy and false positive rates
//	GET  /v1/stream                     Server-Sent Events feed of the live event stream
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

	"github.com/holiday-heartbreaks/parallax/backend/internal/baseline"
	"github.com/holiday-heartbreaks/parallax/backend/internal/engine"
	"github.com/holiday-heartbreaks/parallax/backend/internal/httpapi"
	"github.com/holiday-heartbreaks/parallax/backend/internal/ingest"
	"github.com/holiday-heartbreaks/parallax/backend/internal/livefeed"
	"github.com/holiday-heartbreaks/parallax/backend/internal/session"
	"github.com/holiday-heartbreaks/parallax/backend/internal/storage"
	"github.com/holiday-heartbreaks/parallax/backend/internal/stream"
	"github.com/holiday-heartbreaks/parallax/backend/internal/wal"
	"github.com/holiday-heartbreaks/parallax/backend/internal/worker"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	addr := envOr("PARALLAX_ADDR", ":8080")

	// Runtime components
	bus := stream.New(stream.Options{Logger: logger})
	sessions := session.New()
	normalizer := ingest.New()
	baselines := baseline.NewStore()
	storageStore := storage.NewMemoryStorage()
	walBuffer := wal.New(wal.Options{
		Storage:     storageStore,
		StartOnline: true,
	})

	// Unified Intelligence & Decision Engine
	engService := engine.NewService(engine.Options{
		Logger:    logger,
		Sessions:  sessions,
		Baselines: baselines,
		Bus:       bus,
	})

	// Bounded worker pool consuming from event stream bus
	poolSub := bus.Subscribe("worker-pool", stream.WithPolicy(stream.Block), stream.WithBuffer(1024))
	pool := worker.New(poolSub, engService, worker.Options{
		Logger: logger,
	})

	startTime := time.Now()
	router := httpapi.NewRouterWithDeps(httpapi.Deps{
		Logger:     logger,
		Normalizer: normalizer,
		Sessions:   sessions,
		Baselines:  baselines,
		Bus:        bus,
		Pool:       pool,
		Engine:     engService,
		Storage:    storageStore,
		WAL:        walBuffer,
		StartTime:  startTime,
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

	// Self-ping worker: hits /health every 2 minutes so Render free instance never sleeps
	selfPingURL := envOr("PARALLAX_SELF_PING_URL", "https://parallax-n4it.onrender.com/health")
	if selfPingURL != "" && os.Getenv("PARALLAX_SELF_PING_DISABLED") != "true" {
		go startSelfPing(ctx, logger, selfPingURL, 2*time.Minute)
	}

	// Live feed: continuously replays synthetic sessions through the real
	// pipeline so the dashboard always has genuine, varied live activity to
	// show (report.md §21-22). Disable for benchmarking or a quiet backend.
	if os.Getenv("PARALLAX_LIVE_FEED_DISABLED") != "true" {
		feed := livefeed.New(livefeed.Options{
			Normalizer: normalizer,
			Sessions:   sessions,
			Bus:        bus,
			Logger:     logger,
		})
		go feed.Run(ctx)
	}

	<-ctx.Done()

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Order: stop accepting requests, drain the pool, then close the bus and WAL.
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("http shutdown failed", "err", err)
	}
	if err := pool.Stop(shutdownCtx); err != nil {
		logger.Error("pool drain incomplete", "err", err)
	}
	_ = walBuffer.Close(shutdownCtx)
	_ = storageStore.Close()
	bus.Close()
	logger.Info("stopped", "pool", pool.Stats(), "bus", bus.Stats(), "wal", walBuffer.Status())
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// startSelfPing periodically sends a GET request to targetURL to keep the instance warm.
func startSelfPing(ctx context.Context, logger *slog.Logger, targetURL string, interval time.Duration) {
	logger.Info("starting self-ping heartbeat worker", "target", targetURL, "interval", interval)
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("self-ping heartbeat worker exiting")
			return
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
			if err != nil {
				logger.Warn("failed to construct self-ping request", "err", err)
				continue
			}
			resp, err := client.Do(req)
			if err != nil {
				logger.Warn("self-ping heartbeat failed", "target", targetURL, "err", err)
				continue
			}
			resp.Body.Close()
			logger.Info("self-ping heartbeat succeeded", "target", targetURL, "status", resp.StatusCode)
		}
	}
}
