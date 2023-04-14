//go:build !debug
// +build !debug

package debug

// dummy functions used when debugging is not enabled

// Log writes the formatted string to stderr.
// Level indicates the verbosity of the messages to log.
// The greater the value, the more verbose messages will be logged.
func Log(_ int, _ string, _ ...any) {}

// Assert panics if the specified condition is false.
func Assert(bool) {}

// Assert panics with the provided message if the specified condition is false.
func Assertf(bool, string, ...any) {}
