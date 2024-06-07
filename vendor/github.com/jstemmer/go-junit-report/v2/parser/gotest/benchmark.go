package gotest

import (
	"time"

	"github.com/jstemmer/go-junit-report/v2/gtr"
)

const (
	key = "gotest.benchmark"
)

// Benchmark contains benchmark results and is intended to be used as extra
// data in a gtr.Test.
type Benchmark struct {
	Iterations  int64
	NsPerOp     float64
	MBPerSec    float64
	BytesPerOp  int64
	AllocsPerOp int64
}

// ApproximateDuration returns the duration calculated by multiplying the
// iterations and average time per iteration (NsPerOp).
func (b Benchmark) ApproximateDuration() time.Duration {
	return time.Duration(float64(b.Iterations)*b.NsPerOp) * time.Nanosecond
}

// GetBenchmarkData is a helper function that returns the benchmark contained
// in the data field of the given gtr.Test t. If no (valid) benchmark is
// present, ok will be set to false.
func GetBenchmarkData(t gtr.Test) (b Benchmark, ok bool) {
	if t.Data != nil {
		if data, exists := t.Data[key]; exists {
			bm, ok := data.(Benchmark)
			return bm, ok
		}
	}
	return Benchmark{}, false
}

// SetBenchmarkData is a helper function that writes the benchmark b to the
// data field of the given gtr.Test t.
func SetBenchmarkData(t *gtr.Test, b Benchmark) {
	if t.Data != nil {
		t.Data[key] = b
	}
}
