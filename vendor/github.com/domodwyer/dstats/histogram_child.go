package dstats

import (
	"sync"

	"google.golang.org/grpc/benchmark/stats"
)

// HistogramChild is a Histogram that has been split for concurrent
// measurements.
type HistogramChild struct {
	stats.Histogram

	once   sync.Once
	parent chan *stats.Histogram
}

// Done signals the measurements are ready to be merged.
//
// Done blocks waiting for the measurements to be merged. Subsequent calls to
// Done are a nop.
func (h *HistogramChild) Done() {
	h.once.Do(func() {
		h.parent <- &h.Histogram
	})
}
