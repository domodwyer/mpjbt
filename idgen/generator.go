package idgen

// Generator returns an ID number for an operation.
//
// Implementations of Generator are free to return both monotonic and random
// numbers with varying distribution according to requirements.
//
// Must be concurrency safe.
type Generator interface {
	GetNew() uint64
	GetExisting() uint64
}

// GeneratorSource returns a Generator
type GeneratorSource interface {
	New() Generator
}
