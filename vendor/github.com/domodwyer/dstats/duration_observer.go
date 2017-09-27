package dstats

import (
	"sync/atomic"
	"time"
)

// DurationObserver records durations, returning an average duration in
// milliseconds and observation count.
//
// Measurements are approximate to avoid locks, but pretty damn close.
//
// DurationObserver is safe for concurrent use.
type DurationObserver struct {
	// count is the number of ops
	count uint64

	// cumulativeTime is the total duration observed for count ops in
	// milliseconds
	cumulativeTime uint64
}

// Observe records d and increments the operation counter by delta
func (o *DurationObserver) Observe(d time.Duration, delta uint64) {
	atomic.AddUint64(&o.cumulativeTime, uint64(d/time.Millisecond))
	atomic.AddUint64(&o.count, delta)
}

// Reset returns (count, avg latency in milliseconds) of all the calls to
// Observe since the last Reset call
func (o *DurationObserver) Reset() (count uint64, latency uint64) {
	count = atomic.SwapUint64(&o.count, 0)
	time := atomic.SwapUint64(&o.cumulativeTime, 0)

	// No devide by 0 thanks
	if count == 0 {
		return 0, 0
	}

	// Return count, and avg. in milliseconds
	return count, (time / count)
}
