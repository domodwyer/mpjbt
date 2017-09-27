package idgen

import (
	"math/rand"
	"sync/atomic"
	"time"
)

// Uniform returns a random ID number evenly distributed between 0 and the
// configured maximum ID counter.
//
// Uniform is good for randomly reading records from the entire table, causing a
// high number of cache misses.
type Uniform struct {
	max *uint64
	rnd *rand.Rand
}

// GetNew increments the internal maximum ID counter and returns it's value.
func (u *Uniform) GetNew() uint64 {
	return atomic.AddUint64(u.max, 1)
}

// GetExisting returns a uniformally distributed random ID number.
func (u *Uniform) GetExisting() uint64 {
	return (u.rnd.Uint64()%atomic.LoadUint64(u.max) + 1)
}

// UniformSource returns a Generator using Max as it's internal maximum ID
// counter.
//
// Each Uniform returned by New operates on the same internal maximum ID
// counter.
type UniformSource struct {
	Max uint64
}

// New returns a Uniform Generator using Max as it's internal maximum ID counter.
func (u *UniformSource) New() Generator {
	return &Uniform{
		max: &u.Max,
		rnd: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}
