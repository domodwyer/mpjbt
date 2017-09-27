package idgen

import (
	"testing"
)

type mockGeneratorSource struct{}

func (m *mockGeneratorSource) New() Generator {
	return &mockGenerator{}
}

type mockGenerator struct {
	id uint64
}

func (m *mockGenerator) GetNew() uint64 {
	m.id++
	return m.id
}

func (m *mockGenerator) GetExisting() uint64 {
	m.id++
	return m.id
}

func TestConcurrentPersistent(t *testing.T) {
	const concurrency = 100
	const newIDsPerThread = 10
	const keepFor = 10

	var pProvider = PersistentSource{
		KeepFor: keepFor,
		Source:  &mockGeneratorSource{},
	}

	sem := make(chan struct{})
	results := make(chan bool)
	for i := 0; i < concurrency; i++ {
		go func() {
			ok := true
			idg := pProvider.New()
			seen := map[uint64]struct{}{}

			<-sem
			for i := 0; i < newIDsPerThread; i++ {
				tests := keepFor // first pass needs one less
				if i == 0 {
					tests--
				}

				id := idg.GetExisting()
				for j := 0; j < tests; j++ {
					if got := idg.GetExisting(); got != id {
						panic("wrong ID!")
					}
				}

				if _, exists := seen[id]; exists {
					ok = false
				}
				seen[id] = struct{}{}
			}

			// Send results to aggregator
			results <- ok
		}()
	}
	close(sem)

	// Read results from all threads
	for i := 0; i < concurrency; i++ {
		if ok := <-results; ok != true {
			t.Errorf("saw duplicates")
		}
	}
	close(results)
}

func TestPersistent(t *testing.T) {
	idProvider := &PersistentSource{
		KeepFor: 1,
		Source:  &mockGeneratorSource{},
	}
	id := idProvider.New()

	got := id.GetNew()

	if next := id.GetExisting(); got != next {
		t.Errorf("got %v, wanted %v", next, got)
	}

	got++
	if next := id.GetExisting(); next != got {
		t.Errorf("got %v, wanted %v", next, got)
	}
}
