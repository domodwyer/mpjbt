package idgen

import "sync/atomic"

// Monotonic returns an increasing, ordered ID number.
//
// Calls to GetNew increases the internal counter, and GetExisting returns the
// current counter value.
//
// Monotonic is safe for concurrent access and all Monotonic from the same
// MonotonicSource share the same internal counter.
type Monotonic struct {
	count *uint64
}

// GetNew increments the internal counter by 1 and returns it's value.
func (m *Monotonic) GetNew() uint64 {
	return atomic.AddUint64(m.count, 1)
}

// GetExisting returns the internal counter value.
func (m *Monotonic) GetExisting() uint64 {
	return atomic.LoadUint64(m.count)
}

// MonotonicSource is used to create linked instances of Monotonic.
//
// Each Monotonic returned by New operates on the same internal counter.
type MonotonicSource struct {
	Count uint64
}

// New returns an instance of Monotonic using MonotonicSource.Counter as it's
// internal counter.
func (m *MonotonicSource) New() Generator {
	return &Monotonic{count: &m.Count}
}
