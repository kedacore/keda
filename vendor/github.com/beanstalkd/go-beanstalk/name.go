package beanstalk

import (
	"errors"
)

// NameChars are the allowed name characters in the beanstalkd protocol.
const NameChars = `\-+/;.$_()0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz`

// NameError indicates that a name was malformed and the specific error
// describing how.
type NameError struct {
	Name string
	Err  error
}

func (e NameError) Error() string {
	return e.Err.Error() + ": " + e.Name
}

func (e NameError) Unwrap() error {
	return e.Err
}

// Name format errors. The Err field of NameError contains one of these.
var (
	ErrEmpty   = errors.New("name is empty")
	ErrBadChar = errors.New("name has bad char") // contains a character not in NameChars
	ErrTooLong = errors.New("name is too long")
)

func checkName(s string) error {
	switch {
	case len(s) == 0:
		return NameError{s, ErrEmpty}
	case len(s) >= 200:
		return NameError{s, ErrTooLong}
	case !containsOnly(s, NameChars):
		return NameError{s, ErrBadChar}
	}
	return nil
}

func containsOnly(s, chars string) bool {
outer:
	for _, c := range s {
		for _, m := range chars {
			if c == m {
				continue outer
			}
		}
		return false
	}
	return true
}
