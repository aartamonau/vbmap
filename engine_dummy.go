package main

import (
	"fmt"
)

// RI generator that ignores tags. It just spreads 1s in the matrix so that
// row and column sums equal to number of slaves.
type DummyRIGenerator struct {
	DontAcceptRIGeneratorParams
}

func makeDummyRIGenerator() *DummyRIGenerator {
	return &DummyRIGenerator{}
}

func (_ DummyRIGenerator) String() string {
	return "dummy"
}

func (_ DummyRIGenerator) Generate(params VbmapParams) (ri RI, err error) {
	if params.Tags.TagsCount() != params.NumNodes {
		err = fmt.Errorf("Dummy RI generator is rack unaware and " +
			"doesn't support more than one node on the same tag")
		return
	}

	ri = make([][]bool, params.NumNodes)
	for i := range ri {
		ri[i] = make([]bool, params.NumNodes)
	}

	for i, row := range ri {
		for j := range row {
			k := (j - i + params.NumNodes - 1) % params.NumNodes
			if k < params.NumSlaves {
				ri[i][j] = true
			}
		}
	}

	return
}
