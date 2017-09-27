package main

import (
	"fmt"
	"math/rand"

	"github.com/domodwyer/mpjbt/idgen"
	"github.com/domodwyer/mpjbt/plan"
	"github.com/domodwyer/mpjbt/record"
)

// dbProvider interfaces the available database methods for the underlying
// database type.
type dbProvider interface {
	InsertRecord(data *record.Person, id idgen.Generator, rnd *rand.Rand) bool
	UpdateRecord(data *record.Person, id idgen.Generator, rnd *rand.Rand) bool
	ReadRecord(data *record.Person, id idgen.Generator, rnd *rand.Rand) bool
	ReadRange(data *record.Person, id idgen.Generator, rnd *rand.Rand) bool
	ReadMostRecentRecord(data *record.Person, id idgen.Generator, rnd *rand.Rand) bool
	GetMaxID() (uint64, error)
}

// setWorkload configures p to run the workload identified by name, with methods
// provided by db.
func setWorkload(name string, p *plan.Plan, db dbProvider) error {
	var id idgen.GeneratorSource

	// Get the current maximum ID in the database - ignore any "no data" errors
	// when running insert workloads as the workload type doesn't require
	// existing data.
	max, err := db.GetMaxID()
	if err != nil && name != "insert" && name != "insert-update" {
		return err
	}

	switch name {
	case "insert":
		id = &idgen.MonotonicSource{Count: max}
		p.Add("insert", db.InsertRecord)

	case "insert-update":
		id = &idgen.PersistentSource{
			KeepFor: 1,
			Source:  &idgen.MonotonicSource{Count: max},
		}

		p.Add("insert", db.InsertRecord)
		p.Add("update", db.UpdateRecord)

	case "insert-select":
		id = &idgen.PersistentSource{
			KeepFor: 1,
			Source:  &idgen.MonotonicSource{Count: max},
		}

		p.Add("insert", db.InsertRecord)
		p.Add("select", db.ReadRecord)

	case "insert5-select95":
		id = &idgen.MonotonicSource{Count: max}

		p.Add("insert", db.InsertRecord)
		for i := 0; i < 19; i++ {
			p.Add("select", db.ReadMostRecentRecord)
		}

	case "select-zipfian":
		id = &idgen.ZipfianSource{Max: max}
		p.Add("select", db.ReadRecord)

	case "select-uniform":
		id = &idgen.UniformSource{Max: max}

		p.Add("select", db.ReadRecord)

	case "select-update-zipfian":
		id = &idgen.ZipfianSource{Max: max}

		p.Add("select", db.ReadRecord)
		p.Add("update", db.UpdateRecord)

	case "select-update-uniform":
		id = &idgen.UniformSource{Max: max}

		p.Add("select", db.ReadRecord)
		p.Add("update", db.UpdateRecord)

	case "update-zipfian":
		id = &idgen.ZipfianSource{Max: max}

		p.Add("update", db.UpdateRecord)

	case "update-uniform":
		id = &idgen.UniformSource{Max: max}

		p.Add("update", db.UpdateRecord)

	case "read-range":
		id = &idgen.MonotonicSource{Count: max}

		p.Add("range", db.ReadRange)

	default:
		return fmt.Errorf("unknown workload %q", name)
	}

	p.SetIDGenerator(id)

	return nil
}
