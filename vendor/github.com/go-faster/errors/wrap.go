// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package errors

import (
	"errors"
	"fmt"
)

// A Wrapper provides context around another error.
type Wrapper interface {
	// Unwrap returns the next error in the error chain.
	// If there is no next error, Unwrap returns nil.
	Unwrap() error
}

// Opaque returns an error with the same error formatting as err
// but that does not match err and cannot be unwrapped.
func Opaque(err error) error {
	return noWrapper{err}
}

type noWrapper struct {
	error
}

func (e noWrapper) FormatError(p Printer) (next error) {
	if f, ok := e.error.(Formatter); ok {
		return f.FormatError(p)
	}
	p.Print(e.error)
	return nil
}

// Unwrap returns the result of calling the Unwrap method on err, if err's
// type contains an Unwrap method returning error.
// Otherwise, Unwrap returns nil.
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// Cause returns first recorded Frame.
func Cause(err error) (f Frame, r bool) {
	for {
		we, ok := err.(*wrapError)
		if !ok {
			return f, r
		}
		f = we.frame
		r = r || ok

		err = we.err
	}
}

type wrapError struct {
	msg   string
	err   error
	frame Frame
}

func (e *wrapError) Error() string {
	return fmt.Sprint(e)
}

func (e *wrapError) Format(s fmt.State, v rune) { FormatError(e, s, v) }

func (e *wrapError) FormatError(p Printer) (next error) {
	p.Print(e.msg)
	e.frame.Format(p)
	return e.err
}

func (e *wrapError) Unwrap() error {
	return e.err
}

// Wrap error with message and caller.
func Wrap(err error, message string) error {
	frame := Frame{}
	if Trace() {
		frame = Caller(1)
	}
	return &wrapError{msg: message, err: err, frame: frame}
}

// Wrapf wraps error with formatted message and caller.
func Wrapf(err error, format string, a ...interface{}) error {
	frame := Frame{}
	if Trace() {
		frame = Caller(1)
	}
	msg := fmt.Sprintf(format, a...)
	return &wrapError{msg: msg, err: err, frame: frame}
}

// Is reports whether any error in err's chain matches target.
//
// The chain consists of err itself followed by the sequence of errors obtained by
// repeatedly calling Unwrap.
//
// An error is considered to match a target if it is equal to that target or if
// it implements a method Is(error) bool such that Is(target) returns true.
//
// An error type might provide an Is method so it can be treated as equivalent
// to an existing error. For example, if MyError defines
//
//	func (m MyError) Is(target error) bool { return target == fs.ErrExist }
//
// then Is(MyError{}, fs.ErrExist) returns true. See syscall.Errno.Is for
// an example in the standard library.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target, and if so, sets
// target to that error value and returns true. Otherwise, it returns false.
//
// The chain consists of err itself followed by the sequence of errors obtained by
// repeatedly calling Unwrap.
//
// An error matches target if the error's concrete value is assignable to the value
// pointed to by target, or if the error has a method As(interface{}) bool such that
// As(target) returns true. In the latter case, the As method is responsible for
// setting target.
//
// An error type might provide an As method so it can be treated as if it were a
// different error type.
//
// As panics if target is not a non-nil pointer to either a type that implements
// error, or to any interface type.
func As(err error, target interface{}) bool { return errors.As(err, target) }
