package idgen

// Persistent wraps a Generator to provide the same ID number to GetExisting for
// KeepFor number calls.
//
// Persistent should be used to chain together plan.DoFunc methods operating on
// the same ID number.
//
// Calling GetNew always returns a new ID number.
type Persistent struct {
	keepFor uint
	source  Generator

	usedCount uint
	last      uint64
}

// GetNew calls GetNew on the underlying Generator and resets the used count.
func (p *Persistent) GetNew() uint64 {
	p.last = p.source.GetNew()
	p.usedCount = 0
	return p.last
}

// GetExisting returns the same ID number for KeepFor number of calls after a
// call to GetNew.
//
// Once KeepFor number of calls to GetExisting has been made, GetExisting is
// called on source Generator and it's value returned for KeepFor number of
// calls.
func (p *Persistent) GetExisting() uint64 {
	if p.usedCount == p.keepFor {
		p.usedCount = 0
		p.last = p.source.GetExisting()
		return p.last
	}

	p.usedCount++
	return p.last
}

// PersistentSource returns a configured Persistent for each goroutine.
//
// Each Persistent has in individual KeepFor counter.
type PersistentSource struct {
	KeepFor uint
	Source  GeneratorSource
}

// New returns a Persistent Generator using PersistentSource.Source as the
// underlying Generator.
func (p *PersistentSource) New() Generator {
	gen := p.Source.New()
	return &Persistent{
		keepFor: p.KeepFor,
		source:  gen,
		last:    gen.GetExisting(),
	}
}
