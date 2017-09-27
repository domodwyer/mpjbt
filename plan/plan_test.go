package plan

import (
	"io/ioutil"
	"sync/atomic"
	"testing"

	"github.com/domodwyer/mpjbt/record"
)

func TestPlan_Run(t *testing.T) {
	const numCalls = 10000
	const concurrency = 100

	p := New(0)

	var seen uint64
	p.Add("counter", func(data *record.Person) bool {
		if data.ID > numCalls {
			p.Stop()
			return false
		}
		atomic.AddUint64(&seen, 1)
		return true
	})

	results := p.Run(numCalls+1, concurrency, ioutil.Discard)

	if seen != numCalls {
		t.Errorf("called %d times, want %d", seen, numCalls)
	}

	if len(results) != 1 {
		t.Errorf("got %d results, want 1", len(results))
	}

	// Ensure the histogram got everything
	if c := results[0].Histogram.Count; c != numCalls {
		t.Errorf("histogram saw %d, want %d", c, numCalls)
	}
}

func TestPlan_DoesNotCallNext(t *testing.T) {
	const numCalls = 1000
	const concurrency = 10

	p := New(0)

	p.Add("step1", func(data *record.Person) bool {
		if data.ID > numCalls {
			p.Stop()
		}
		return false
	})
	p.Add("step2", func(data *record.Person) bool {
		panic("unexpected call to step2")
		return true
	})

	results := p.Run(numCalls+1, concurrency, ioutil.Discard)

	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}

	// Ensure the histogram didn't measure a failed call
	if c := results[0].Histogram.Count; c != 0 {
		t.Errorf("histogram saw %d, want %d", c, 0)
	}
}

func TestPlan_StatusTicker(t *testing.T) {
	const concurrency = 1

	p := New(0)
	p.Add("step1", func(data *record.Person) bool {
		return true
	})
	p.Add("step2", func(data *record.Person) bool {
		p.Stop()
		return true
	})

	p.Run(1, concurrency, ioutil.Discard)

	want := "step1 1op/s avg.0ms\tstep2 1op/s avg.0ms"
	got := p.buildLine()

	if got != want {
		t.Errorf("got '%v', want '%v'", got, want)
	}
}
