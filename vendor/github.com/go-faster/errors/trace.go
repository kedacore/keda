//go:build !noerrtrace
// +build !noerrtrace

package errors

import (
	"sync/atomic"
)

var traceFlag int64

const (
	traceEnabled  = 0 // enabled by default
	traceDisabled = 1
)

// setTrace sets tracing flag that controls capturing caller frames.
func setTrace(trace bool) {
	if trace {
		atomic.StoreInt64(&traceFlag, traceEnabled)
	} else {
		atomic.StoreInt64(&traceFlag, traceDisabled)
	}
}

// enableTrace enables capturing caller frames.
//
// Intentionally left unexported.
func enableTrace() { setTrace(true) }

// DisableTrace disables capturing caller frames.
func DisableTrace() { setTrace(false) }

// Trace reports whether caller stack capture is enabled.
func Trace() bool {
	return atomic.LoadInt64(&traceFlag) == traceEnabled
}
