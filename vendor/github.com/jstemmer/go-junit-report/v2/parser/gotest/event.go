package gotest

import (
	"time"

	"github.com/jstemmer/go-junit-report/v2/parser/gotest/internal/reader"
)

// Event is a single event in a Go test or benchmark.
type Event struct {
	Type string `json:"type"`

	Name     string        `json:"name,omitempty"`
	Package  string        `json:"pkg,omitempty"`
	Result   string        `json:"result,omitempty"`
	Duration time.Duration `json:"duration,omitempty"`
	Data     string        `json:"data,omitempty"`
	Indent   int           `json:"indent,omitempty"`

	// Code coverage
	CovPct      float64  `json:"coverage_percentage,omitempty"`
	CovPackages []string `json:"coverage_packages,omitempty"`

	// Benchmarks
	Iterations  int64   `json:"benchmark_iterations,omitempty"`
	NsPerOp     float64 `json:"benchmark_ns_per_op,omitempty"`
	MBPerSec    float64 `json:"benchmark_mb_per_sec,omitempty"`
	BytesPerOp  int64   `json:"benchmark_bytes_per_op,omitempty"`
	AllocsPerOp int64   `json:"benchmark_allocs_per_op,omitempty"`
}

func (e *Event) applyMetadata(m *reader.Metadata) {
	if e == nil || m == nil {
		return
	}
	e.Package = m.Package
}
