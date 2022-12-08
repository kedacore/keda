// Package unixtime provides helper functions to convert between Go time
// values and Unix time values in milli- and microseconds.
//
// The package has been created in response to Ian Lance Taylor's
// suggestion in the discussion of the Go issue #27782. A former issue
// discussing the same functionality has been #18935.
//
//    https://github.com/golang/go/issues/27782
//    https://github.com/golang/go/issues/18935
//
package unixtime

import "time"

// Micro converts a time value to the Unix time in microseconds.
func Micro(t time.Time) int64 {
	s := t.Unix() * 1e6
	µs := int64(t.Nanosecond()) / 1e3
	return s + µs
}

// FromMicro converts the Unix time in microseconds to a time value.
func FromMicro(µs int64) time.Time {
	s := µs / 1e6
	ns := (µs - s*1e6) * 1e3
	return time.Unix(s, ns)
}

// Milli converts a time value to the Unix time in milliseconds.
func Milli(t time.Time) int64 {
	s := t.Unix() * 1e3
	ms := int64(t.Nanosecond()) / 1e6
	return s + ms
}

// FromMilli converts the Unix time in milliseconds to a time value.
func FromMilli(ms int64) time.Time {
	s := ms / 1e3
	ns := (ms - s*1e3) * 1e6
	return time.Unix(s, ns)
}
