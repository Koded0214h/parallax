package worker

import (
	"math"
	"math/rand/v2"
	"sort"
	"sync"
	"time"
)

// LatencyStats is a point-in-time summary of processing latency.
type LatencyStats struct {
	Count uint64
	Min   time.Duration
	Max   time.Duration
	Mean  time.Duration
	P50   time.Duration
	P95   time.Duration
	P99   time.Duration
}

// reservoir keeps a bounded uniform random sample of observed durations
// (Vitter's Algorithm R) plus exact count / min / max / sum. Percentiles are
// estimated from the sample; count and extremes are exact. Concurrency-safe.
type reservoir struct {
	mu      sync.Mutex
	cap     int
	samples []time.Duration
	n       uint64 // total observed
	sumNS   int64
	min     time.Duration
	max     time.Duration
}

func newReservoir(capacity int) *reservoir {
	if capacity < 1 {
		capacity = 1
	}
	return &reservoir{cap: capacity, samples: make([]time.Duration, 0, capacity)}
}

// Observe records one duration.
func (r *reservoir) Observe(d time.Duration) {
	if d < 0 {
		d = 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	r.n++
	r.sumNS += int64(d)
	if r.n == 1 {
		r.min, r.max = d, d
	} else {
		if d < r.min {
			r.min = d
		}
		if d > r.max {
			r.max = d
		}
	}

	if len(r.samples) < r.cap {
		r.samples = append(r.samples, d)
		return
	}
	// Replace an existing sample with probability cap/n.
	if j := rand.Uint64N(r.n); j < uint64(r.cap) {
		r.samples[j] = d
	}
}

// Snapshot returns the current summary. Safe to call concurrently with Observe.
func (r *reservoir) Snapshot() LatencyStats {
	r.mu.Lock()
	n, sumNS, mn, mx := r.n, r.sumNS, r.min, r.max
	cp := make([]time.Duration, len(r.samples))
	copy(cp, r.samples)
	r.mu.Unlock()

	if n == 0 {
		return LatencyStats{}
	}
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })

	q := func(p float64) time.Duration {
		if len(cp) == 0 {
			return 0
		}
		idx := int(math.Ceil(p*float64(len(cp)))) - 1
		if idx < 0 {
			idx = 0
		}
		if idx >= len(cp) {
			idx = len(cp) - 1
		}
		return cp[idx]
	}

	return LatencyStats{
		Count: n,
		Min:   mn,
		Max:   mx,
		Mean:  time.Duration(sumNS / int64(n)),
		P50:   q(0.50),
		P95:   q(0.95),
		P99:   q(0.99),
	}
}
