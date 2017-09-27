package plan

import (
	"math/rand"

	"github.com/domodwyer/dstats"
	"github.com/domodwyer/mpjbt/idgen"
	"github.com/domodwyer/mpjbt/record"
)

// DoFunc defines a database operation.
//
// If a DoFunc returns false, it's latency measurement is abandoned and it's
// call does not count towards the operation limit. Implementations of DoFunc
// should return false when an error occurs.
type DoFunc func(data *record.Person, rid idgen.Generator, rnd *rand.Rand) bool

// operation combines a DoFunc and a collection of statistics.
type operation struct {
	counter   *dstats.DurationObserver
	histogram *dstats.Histogram // not to be accessed concurrently

	name   string
	doFunc DoFunc
}
