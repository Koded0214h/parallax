// Command loadgen drives synthetic traffic through the Parallax end-to-end
// pipeline (ingest -> session -> stream bus -> worker pool -> engine) and
// prints a throughput / latency / allocation report (prd.md §19).
//
//	go run ./cmd/loadgen -events 500000 -concurrency 8
//	go run ./cmd/loadgen -events 100000 -runs 5 -json
//
// It is a thin CLI over bench.Harness; the harness runs the real decision
// engine as the pool Processor, so these numbers are end-to-end.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/bench"
)

func main() {
	var (
		events      = flag.Int("events", 200_000, "total events per run")
		concurrency = flag.Int("concurrency", 0, "concurrent producers + worker pool size (0 = GOMAXPROCS)")
		runs        = flag.Int("runs", 1, "number of runs")
		asJSON      = flag.Bool("json", false, "emit JSON instead of a table")
		verbose     = flag.Bool("v", false, "keep the engine's per-event logs (noisy)")
	)
	flag.Parse()

	// The decision engine logs one line per event at INFO; silence it for a
	// load run unless -v is set.
	if !*verbose {
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	}

	if *events < 1 {
		*events = 1
	}
	if *runs < 1 {
		*runs = 1
	}

	h := bench.NewHarness(*concurrency)
	results := make([]bench.Result, 0, *runs)
	for i := 0; i < *runs; i++ {
		res, err := h.Run(*events)
		if err != nil {
			fmt.Fprintf(os.Stderr, "run %d failed: %v\n", i+1, err)
			os.Exit(1)
		}
		results = append(results, res)
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		var payload any = results
		if *runs == 1 {
			payload = results[0]
		}
		if err := enc.Encode(payload); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	printTable(results)
}

func printTable(rs []bench.Result) {
	fmt.Printf("%-4s %10s %10s %13s %10s %10s %10s %10s %9s %4s\n",
		"run", "events", "dur", "events/sec", "p50", "p95", "p99", "max", "B/evt", "GC")
	for i, r := range rs {
		fmt.Printf("%-4d %10d %10s %13.0f %10s %10s %10s %10s %9d %4d\n",
			i+1,
			r.TotalEvents,
			r.Duration.Round(time.Millisecond),
			r.EventsPerSec,
			short(r.LatencyP50), short(r.LatencyP95), short(r.LatencyP99), short(r.LatencyMax),
			bytesPerEvent(r),
			r.NumGC,
		)
	}
	if len(rs) > 1 {
		fmt.Println("----")
		fmt.Printf("%-4s %10s %10s %13.0f %10s %10s %10s\n",
			"med", "", "",
			median(rs, func(r bench.Result) float64 { return r.EventsPerSec }),
			short(medianDur(rs, func(r bench.Result) time.Duration { return r.LatencyP50 })),
			short(medianDur(rs, func(r bench.Result) time.Duration { return r.LatencyP95 })),
			short(medianDur(rs, func(r bench.Result) time.Duration { return r.LatencyP99 })),
		)
	}
}

func bytesPerEvent(r bench.Result) uint64 {
	if r.TotalEvents == 0 {
		return 0
	}
	return r.TotalAlloc / uint64(r.TotalEvents)
}

func short(d time.Duration) string {
	switch {
	case d >= time.Millisecond:
		return fmt.Sprintf("%.2fms", float64(d)/float64(time.Millisecond))
	case d >= time.Microsecond:
		return fmt.Sprintf("%.0fµs", float64(d)/float64(time.Microsecond))
	default:
		return d.String()
	}
}

func median(rs []bench.Result, f func(bench.Result) float64) float64 {
	xs := make([]float64, len(rs))
	for i, r := range rs {
		xs[i] = f(r)
	}
	slices.Sort(xs)
	return xs[len(xs)/2]
}

func medianDur(rs []bench.Result, f func(bench.Result) time.Duration) time.Duration {
	xs := make([]time.Duration, len(rs))
	for i, r := range rs {
		xs[i] = f(r)
	}
	slices.Sort(xs)
	return xs[len(xs)/2]
}
