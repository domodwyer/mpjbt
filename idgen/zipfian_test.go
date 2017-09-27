package idgen

import (
	"fmt"
	"testing"
)

func TestExample_ZipFianGetNew(t *testing.T) {
	source := ZipfianSource{
		Max: 100000,
	}

	gen := source.New()
	for i := 0; i < 20; i++ {
		fmt.Println(gen.GetExisting())
	}
}
