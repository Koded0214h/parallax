package bench

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/internal/baseline"
	"github.com/holiday-heartbreaks/parallax/backend/internal/engine"
	"github.com/holiday-heartbreaks/parallax/backend/internal/ingest"
	"github.com/holiday-heartbreaks/parallax/backend/internal/session"
	"github.com/holiday-heartbreaks/parallax/backend/internal/stream"
	"github.com/holiday-heartbreaks/parallax/backend/internal/worker"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

// Result holds the measured throughput and latency metrics (docs/backend.md M9, prd.md §19).
type Result struct {
	TotalEvents      int           `json:"total_events"`
	Duration         time.Duration `json:"duration"`
	EventsPerSec     float64       `json:"events_per_sec"`
	LatencyP50       time.Duration `json:"latency_p50"`
	LatencyP95       time.Duration `json:"latency_p95"`
	LatencyP99       time.Duration `json:"latency_p99"`
	LatencyMin       time.Duration `json:"latency_min"`
	LatencyMax       time.Duration `json:"latency_max"`
	AllocBytes       uint64        `json:"alloc_bytes"`
	TotalAlloc       uint64        `json:"total_alloc"`
	NumGC            uint32        `json:"num_gc"`
}

// Harness executes concurrent load against the end-to-end pipeline.
type Harness struct {
	concurrency int
}

// NewHarness creates a benchmark harness with worker concurrency.
func NewHarness(concurrency int) *Harness {
	if concurrency <= 0 {
		concurrency = runtime.GOMAXPROCS(0)
	}
	return &Harness{concurrency: concurrency}
}

// Run executes the full ingest-to-decision pipeline on N events and reports metrics.
func (h *Harness) Run(totalEvents int) (Result, error) {
	normalizer := ingest.New()
	sessions := session.New()
	baselines := baseline.NewStore()
	bus := stream.New(stream.Options{})
	defer bus.Close()

	svc := engine.NewService(engine.Options{
		Sessions:  sessions,
		Baselines: baselines,
		Bus:       bus,
	})

	sub := bus.Subscribe("bench-worker-pool", stream.WithPolicy(stream.Block), stream.WithBuffer(2048))
	pool := worker.New(sub, svc, worker.Options{
		Workers: h.concurrency,
	})
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = pool.Stop(ctx)
	}()

	var memStart runtime.MemStats
	runtime.ReadMemStats(&memStart)

	start := time.Now()
	var wg sync.WaitGroup
	eventsPerWorker := totalEvents / h.concurrency

	for w := 0; w < h.concurrency; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			sessID := fmt.Sprintf("sess_bench_%d", workerID)
			userID := "user_001"

			for i := 0; i < eventsPerWorker; i++ {
				ev := contracts.Event{
					EventID:   fmt.Sprintf("ev_b_%d_%d", workerID, i),
					SessionID: sessID,
					UserID:    userID,
					Type:      contracts.EventAmountEntered,
					Timestamp: time.Now().Unix(),
					Metadata: map[string]any{
						"amount":         35000.0,
						"beneficiary_id": "ben_mother",
					},
				}
				norm, err := normalizer.Normalize(ev)
				if err != nil {
					continue
				}
				stored, _ := sessions.Append(norm)
				bus.Publish(stored)
			}
		}(w)
	}

	wg.Wait()

	// Wait briefly for worker pool to drain
	deadline := time.Now().Add(5 * time.Second)
	for pool.Stats().Processed < uint64(totalEvents) && time.Now().Before(deadline) {
		time.Sleep(2 * time.Millisecond)
	}

	elapsed := time.Since(start)

	var memEnd runtime.MemStats
	runtime.ReadMemStats(&memEnd)

	poolStats := pool.Stats()
	eps := float64(poolStats.Processed) / elapsed.Seconds()

	return Result{
		TotalEvents:  int(poolStats.Processed),
		Duration:     elapsed,
		EventsPerSec: eps,
		LatencyP50:   poolStats.Latency.P50,
		LatencyP95:   poolStats.Latency.P95,
		LatencyP99:   poolStats.Latency.P99,
		LatencyMin:   poolStats.Latency.Min,
		LatencyMax:   poolStats.Latency.Max,
		AllocBytes:   memEnd.Alloc - memStart.Alloc,
		TotalAlloc:   memEnd.TotalAlloc - memStart.TotalAlloc,
		NumGC:        memEnd.NumGC - memStart.NumGC,
	}, nil
}
