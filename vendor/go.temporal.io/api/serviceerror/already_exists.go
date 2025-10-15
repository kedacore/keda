package serviceerror

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	// AlreadyExists represents general AlreadyExists gRPC error.
	AlreadyExists struct {
		Message string
		st      *status.Status
	}
)

// Deprecated. Typo in the name. Use NewAlreadyExists instead.
func NewAlreadyExist(message string) error {
	return NewAlreadyExists(message)
}

// NewAlreadyExists returns new AlreadyExists error.
func NewAlreadyExists(message string) error {
	return &AlreadyExists{
		Message: message,
	}
}

// NewAlreadyExistsf returns new AlreadyExists error with formatted message.
func NewAlreadyExistsf(format string, args ...any) error {
	return &AlreadyExists{
		Message: fmt.Sprintf(format, args...),
	}
}

// Error returns string message.
func (e *AlreadyExists) Error() string {
	return e.Message
}

func (e *AlreadyExists) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	return status.New(codes.AlreadyExists, e.Message)
}

func newAlreadyExists(st *status.Status) error {
	return &AlreadyExists{
		Message: st.Message(),
		st:      st,
	}
}
