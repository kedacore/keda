package serviceerror

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	// Unimplemented represents unimplemented error.
	Unimplemented struct {
		Message string
		st      *status.Status
	}
)

// NewUnimplemented returns new Unimplemented error.
func NewUnimplemented(message string) error {
	return &Unimplemented{
		Message: message,
	}
}

// NewUnimplementedf returns new Unimplemented error with formatted message.
func NewUnimplementedf(format string, args ...any) error {
	return &Unimplemented{
		Message: fmt.Sprintf(format, args...),
	}
}

// Error returns string message.
func (e *Unimplemented) Error() string {
	return e.Message
}

func (e *Unimplemented) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	return status.New(codes.Unimplemented, e.Message)
}

func newUnimplemented(st *status.Status) error {
	return &Unimplemented{
		Message: st.Message(),
		st:      st,
	}
}
