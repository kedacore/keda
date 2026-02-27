//go:build noerrtrace
// +build noerrtrace

package errors

// enableTrace does nothing.
func enableTrace() {}

// DisableTrace does nothing.
func DisableTrace() {}

// Trace always returns false.
func Trace() bool { return false }
