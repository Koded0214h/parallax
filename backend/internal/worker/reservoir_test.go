package worker

import (
	"sync"
	"testing"
	"time"
)

func TestReservoirExactWhenUnderCap(t *testing.T) {
	r := newReservoir(2000)
	for i := 1; i <= 1000; i++ {
		r.Observe(time.Duration(i) * time.Millisecond)
	}
	s := r.Snapshot()

	if s.Count != 1000 {
		t.Errorf("Count = %d, want 1000", s.Count)
	}
	if s.Min != time.Millisecond || s.Max != 1000*time.Millisecond {
		t.Errorf("Min/Max = %v/%v, want 1ms/1000ms", s.Min, s.Max)
	}
	within := func(got, want, tol time.Duration) bool { return got >= want-tol && got <= want+tol }
	if !within(s.Mean, 500*time.Millisecond, 20*time.Millisecond) {
		t.Errorf("Mean = %v, want ~500ms", s.Mean)
	}
	if !within(s.P50, 500*time.Millisecond, 25*time.Millisecond) {
		t.Errorf("P50 = %v, want ~500ms", s.P50)
	}
	if !within(s.P95, 950*time.Millisecond, 25*time.Millisecond) {
		t.Errorf("P95 = %v, want ~950ms", s.P95)
	}
	if !within(s.P99, 990*time.Millisecond, 15*time.Millisecond) {
		t.Errorf("P99 = %v, want ~990ms", s.P99)
	}
	if !(s.P50 <= s.P95 && s.P95 <= s.P99 && s.P99 <= s.Max) {
		t.Errorf("percentiles not monotonic: %+v", s)
	}
}

func TestReservoirBoundedMemory(t *testing.T) {
	r := newReservoir(128)
	const n = 100_000
	for i := 1; i <= n; i++ {
		r.Observe(time.Duration(i) * time.Microsecond)
	}
	if len(r.samples) > r.cap {
		t.Fatalf("sample slice grew to %d, cap %d", len(r.samples), r.cap)
	}
	s := r.Snapshot()
	if s.Count != n {
		t.Errorf("Count = %d, want %d", s.Count, n)
	}
	if s.Min != time.Microsecond || s.Max != n*time.Microsecond {
		t.Errorf("Min/Max = %v/%v, want exact extremes", s.Min, s.Max)
	}
	// Sampling error with 128 samples is real; allow a wide band.
	mid := time.Duration(n/2) * time.Microsecond
	if s.P50 < mid*70/100 || s.P50 > mid*130/100 {
		t.Errorf("P50 = %v, want roughly %v", s.P50, mid)
	}
}

func TestReservoirEmpty(t *testing.T) {
	if got := newReservoir(16).Snapshot(); got != (LatencyStats{}) {
		t.Errorf("empty Snapshot = %+v, want zero", got)
	}
}

func TestReservoirClampsCapacity(t *testing.T) {
	r := newReservoir(0)
	r.Observe(5 * time.Millisecond)
	if s := r.Snapshot(); s.Count != 1 || s.Min != 5*time.Millisecond {
		t.Errorf("Snapshot = %+v", s)
	}
}

func TestReservoirNegativeClampedToZero(t *testing.T) {
	r := newReservoir(8)
	r.Observe(-3 * time.Second)
	if s := r.Snapshot(); s.Min != 0 || s.Max != 0 {
		t.Errorf("negative not clamped: %+v", s)
	}
}

func TestReservoirConcurrent(t *testing.T) {
	r := newReservoir(512)
	const goroutines, each = 20, 1000
	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Go(func() {
			for i := range each {
				r.Observe(time.Duration(g*each+i+1) * time.Microsecond)
			}
		})
	}
	wg.Wait()

	if s := r.Snapshot(); s.Count != goroutines*each {
		t.Errorf("Count = %d, want %d", s.Count, goroutines*each)
	}
}
