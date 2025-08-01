package serviceerror

import (
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

// NewAlreadyExist returns new AlreadyExists error.
func NewAlreadyExist(message string) error {
	return &AlreadyExists{
		Message: message,
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
