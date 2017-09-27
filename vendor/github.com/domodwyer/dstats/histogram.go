package dstats

import (
	"encoding/csv"
	"io"
	"strconv"
	"sync"

	"google.golang.org/grpc/benchmark/stats"
)

// HistogramOptions aliases stats.HistogramOptions to play nicely with
// vendoring.
type HistogramOptions stats.HistogramOptions

// Histogram is a splitable extension of Google's grpc stats.Histogram.
//
// Histogram cannot be used concurrently, so Split children and Merge them for
// concurrent measurements.
//
// A Histogram should not be copied.
type Histogram struct {
	stats.Histogram

	count    uint
	children chan *stats.Histogram

	mu     sync.Mutex
	once   sync.Once
	closed bool
}

// Split creates a child histogram for concurrent use.
//
// If split is called after Merge the returned value is nil.
func (h *Histogram) Split() *HistogramChild {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return nil
	}

	h.count++
	return &HistogramChild{
		Histogram: *stats.NewHistogram(h.Histogram.Opts()),
		parent:    h.children,
	}
}

// Merge consolidates all HistogramChild instances.
//
// Merge waits for all the split HistogramChild instances to call Done before
// returning.
//
// Calls to Split return nil after Merge has been called.
func (h *Histogram) Merge() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.once.Do(func() {
		for i := h.count; i > 0; i-- {
			h.Histogram.Merge(<-h.children)
		}
		close(h.children)
		h.closed = true
	})
}

// WriteCSV encodes h into CSV format, writing the result to w.
//
// The fields are:
//
// 		lower-bound, upper-bound, count, percent, accumulative-percent
//
func (h *Histogram) WriteCSV(w io.Writer) error {
	// Wrap w in a CSV writer
	enc := csv.NewWriter(w)
	defer enc.Flush()

	// Write headers
	if err := enc.Write([]string{"LowerBound", "UpperBound", "Count", "Percent", "AccumulativePercent"}); err != nil {
		return err
	}

	row := make([]string, 5)
	accCount := int64(0)
	percentMulti := 100 / float64(h.Histogram.Count)
	for i, b := range h.Histogram.Buckets {
		// Lower bound
		row[0] = strconv.FormatFloat(b.LowBound, 'f', 1, 64)

		// Upper bound
		if i+1 < len(h.Histogram.Buckets) {
			row[1] = strconv.FormatFloat(h.Histogram.Buckets[i+1].LowBound, 'f', 1, 64)
		} else {
			row[1] = "inf"
		}

		// Track an accumulating count
		accCount += b.Count

		// This bucket count
		row[2] = strconv.FormatInt(b.Count, 10)

		// Percentages
		row[3] = strconv.FormatFloat(float64(b.Count)*percentMulti, 'f', 1, 64)
		row[4] = strconv.FormatFloat(float64(accCount)*percentMulti, 'f', 1, 64)

		if err := enc.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// NewHistogram returns a Histogram configured using opts.
func NewHistogram(opts HistogramOptions) *Histogram {
	return &Histogram{
		Histogram: *stats.NewHistogram((stats.HistogramOptions)(opts)),
		children:  make(chan *stats.Histogram),
	}
}
