//go:build go1.20
// +build go1.20

package errors

import "errors"

// Join returns an error that wraps the given errors.
// Any nil error values are discarded.
// Join returns nil if every value in errs is nil.
// The error formats as the concatenation of the strings obtained
// by calling the Error method of each element of errs, with a newline
// between each string.
//
// A non-nil error returned by Join implements the Unwrap() []error method.
//
// Available only for go 1.20 or superior.
func Join(errs ...error) error {
	return errors.Join(errs...)
}
