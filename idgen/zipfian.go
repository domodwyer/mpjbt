package idgen

import (
	"math"
	"math/rand"
	"sync/atomic"
	"time"
)

// Zipfian is used to return a random ID number heavily skewed towards a record
// with a high ID numbers.
//
// Example output when Max is 100,000:
//		99991
//		99986
//		99993
//		99981
//		99979
//		99976
//		99972
//		99934
//		99996
//		99995
//		99998
//		99930
//		99997
//		99921
//		99973
//		99982
//
// Zipfian is good for performing operations on records that are probably still
// in the cache.
//
// All Zipfian from the same ZipfianSource share an internal maximum ID counter.
type Zipfian struct {
	max *uint64
	rnd *rand.Zipf
}

// GetNew increments the internal maximum ID counter and returns it's value.
func (z *Zipfian) GetNew() uint64 {
	return atomic.AddUint64(z.max, 1)
}

// GetExisting return an existing ID number heavily skewed towards the internal
// maximum ID counter.
func (z *Zipfian) GetExisting() uint64 {
	max := atomic.LoadUint64(z.max)

	n := z.rnd.Uint64()
	if n < max {
		return max - n
	}

	return max
}

// ZipfianSource is used to create linked instances of Zipfian.
//
// All Zipfian from the same source share the same Max ID counter.
type ZipfianSource struct {
	Max uint64
}

// New returns an instance of Zipfian using the same Max.
func (z *ZipfianSource) New() Generator {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &Zipfian{
		max: &z.Max,
		rnd: rand.NewZipf(r, 2.5, 50, math.MaxUint64),
	}
}
