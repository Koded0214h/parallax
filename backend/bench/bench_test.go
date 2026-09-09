package bench

import (
	"testing"
	"time"
)

func TestPipelineThroughputAndLatencyHarness(t *testing.T) {
	harness := NewHarness(4)
	res, err := harness.Run(2000)
	if err != nil {
		t.Fatalf("harness run failed: %v", err)
	}

	t.Logf("\nBenchmark Results (End-to-End Pipeline):\n"+
		"  Total Processed: %d events\n"+
		"  Elapsed Time: %v\n"+
		"  Throughput: %.0f events/sec\n"+
		"  Latency Min: %v\n"+
		"  Latency p50: %v\n"+
		"  Latency p95: %v\n"+
		"  Latency p99: %v\n"+
		"  Latency Max: %v\n"+
		"  GC Cycles: %d\n",
		res.TotalEvents,
		res.Duration,
		res.EventsPerSec,
		res.LatencyMin,
		res.LatencyP50,
		res.LatencyP95,
		res.LatencyP99,
		res.LatencyMax,
		res.NumGC,
	)

	// PRD §18 requirement: Decision latency must be sub-second, target p95 < 150ms
	if res.LatencyP95 > 150*time.Millisecond {
		t.Errorf("p95 latency (%v) exceeded target 150ms", res.LatencyP95)
	}
	if res.EventsPerSec < 150 {
		t.Errorf("throughput (%f eps) lower than expected", res.EventsPerSec)
	}
}
