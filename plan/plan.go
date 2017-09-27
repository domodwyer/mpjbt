package plan

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/domodwyer/dstats"
	"github.com/domodwyer/mpjbt/idgen"
	"github.com/domodwyer/mpjbt/record"
)

// ErrStopped is returned when the Stop method has been called on the Plan
// instance. A new Plan must be created.
var ErrStopped = errors.New("plan stopped")

// Plan defines a series of operations to perform.
//
// A Plan concurrently runs the configured number of workers, each performing
// operations in sequence and measuring their latency.
//
// A Plan stops the workers when the configured maximum number of operations is
// reached, or Stop is called.
type Plan struct {
	id          idgen.GeneratorSource
	ops         []operation
	paddingSize uint64

	// Operation limits
	opsMax   uint64
	opsCount uint64

	wg     sync.WaitGroup
	mu     sync.Mutex
	stop   chan struct{}
	once   sync.Once
	closed bool
}

// Result provides the name of an operation run as part of a Plan, and the
// associated latency histogram for all it's calls.
type Result struct {
	Name      string
	Histogram *dstats.Histogram
}

// Run starts workers number of concurrent workers, and writes a description
// (including approximate average throughput) of each operation every second to
// w.
//
// When complete, the operation statistics from each worker are aggregated and
// returned.
func (p *Plan) Run(workers uint64, statusW io.Writer) []Result {
	p.mu.Lock()
	defer p.mu.Unlock()
	defer p.Stop()

	// Don't start if we've stopped
	if p.closed {
		return nil
	}

	// Print out status updates
	go p.statusTicker(statusW)

	// Run workers and wait
	wg := &sync.WaitGroup{}
	for i := uint64(0); i < workers; i++ {
		wg.Add(1)
		go p.worker(wg)
	}
	wg.Wait()

	// Collect results and return
	var results = make([]Result, len(p.ops))
	for i, op := range p.ops {
		op.histogram.Merge()
		results[i] = Result{
			Name:      op.name,
			Histogram: op.histogram,
		}
	}
	return results
}

// Stop causes any remaining work to be abandoned and immediately aggregates all
// the statistics from the workers.
//
// A Plan will complete any in-progress blocking operations before returning
// from Run. Once stopped, a Plan cannot be resumed.
func (p *Plan) Stop() {
	p.once.Do(func() {
		p.closed = true
		close(p.stop)
	})
}

// Add pushes a new operation into the Plan run list.
//
// Each worker will run all operations in the sequence provided to Add.
func (p *Plan) Add(name string, f DoFunc) {
	p.mu.Lock()
	defer p.mu.Unlock()

	op := operation{
		name:    name,
		doFunc:  f,
		counter: &dstats.DurationObserver{},
		histogram: dstats.NewHistogram(dstats.HistogramOptions{
			NumBuckets:     100,
			GrowthFactor:   0.1,
			BaseBucketSize: float64(1),
		}),
	}

	p.ops = append(p.ops, op)
}

// worker performs all the Plan operations in sequence until either the maximum
// number of operations is reached, or the Plan is stopped.
//
// To allow each worker to operate without contention, they do not use mutexes
// to synchronise access to a shared statistics datastructure, instead each
// worker maintains it's own statistics and Run aggregates them when the workers
// return.
//
// worker must be called while the Plan mutex is held.
func (p *Plan) worker(wg *sync.WaitGroup) {
	defer wg.Done()

	// Build a slice of our child histograms
	counters := map[string]*dstats.DurationObserver{}
	histograms := map[string]*dstats.HistogramChild{}
	for _, op := range p.ops {
		if _, exists := histograms[op.name]; !exists {
			counters[op.name] = op.counter
			histograms[op.name] = op.histogram.Split()
		}
	}

	// Defer merging all the histograms
	defer func() {
		go func() {
			for _, op := range p.ops {
				// Important to call done in the same order as Split() was
				// called to prevent a deadlock - Done() is idempotent so this
				// is fine.
				histograms[op.name].Done()
			}
		}()
	}()

	// Calls to rand.Rand methods lock an underlying mutex, so each worker gets
	// it's own instance.
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	record := &record.Person{Padding: make([]byte, p.paddingSize)}
	record.Randomise(rnd)

	// Get a ID Generator safe for concurrent access
	id := p.id.New()

	for {
		select {
		case <-p.stop:
			return
		default:
		}

		for _, op := range p.ops {
			start := time.Now()
			measure := op.doFunc(record, id, rnd)
			delta := time.Since(start)

			if !measure {
				// If the DoFunc returns false, an error occurred and this
				// measurement should be dropped.
				//
				// This does not count towards the operation count.
				continue
			}

			// Record in the histogram as milliseconds
			histograms[op.name].Add(int64(delta / time.Millisecond))

			// Record in the operation counter - safe for concurrent access
			counters[op.name].Observe(delta, 1)

			// Stop when we hit the ops limit
			if p.opsMax != 0 && atomic.AddUint64(&p.opsCount, 1) > p.opsMax {
				p.Stop()
				return
			}

			select {
			case <-p.stop:
				return
			default:
			}
		}
	}
}

// statusTicker prints an approximate throughput measurement of each operation
// to w every second.
//
// Every 10 seconds, an empty line is wrote to break up the output.
func (p *Plan) statusTicker(w io.Writer) {
	// init logger
	log.New(w, "", log.LstdFlags)

	var count uint8
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		select {
		case <-p.stop:
			ticker.Stop()
			return
		default:
			if count == 10 {
				count = 0
				fmt.Fprintf(w, "\n")
			}
			count++

			log.Println(p.buildLine())
		}
	}
}

// buildLine returns a string describing the average throughput of all
// operations since the last call to buildLine.
func (p *Plan) buildLine() string {
	var line string
	var lastName string
	for _, op := range p.ops {
		if lastName == op.name {
			continue
		}
		count, avg := op.counter.Reset()
		line = fmt.Sprintf("%s\t%s %dop/s avg.%dms", line, op.name, count, avg)
		lastName = op.name
	}
	return strings.TrimLeft(line, "\t")
}

// SetIDGenerator configures PLan to use id as it's ID generator.
func (p *Plan) SetIDGenerator(id idgen.GeneratorSource) {
	p.id = id
}

// New returns an empty Plan, configured to run opsMax number of operations,
// with paddingSize amount of randomised binary record padding.
func New(opsMax uint64, paddingSize uint64) *Plan {
	// TODO: move paddingSize into a record provider and keep it out of the
	// plan.
	return &Plan{
		id:          &idgen.MonotonicSource{},
		stop:        make(chan struct{}),
		paddingSize: paddingSize,
		opsMax:      opsMax,
	}
}
