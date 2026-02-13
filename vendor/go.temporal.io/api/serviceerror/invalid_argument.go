package serviceerror

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	// InvalidArgument represents invalid argument error.
	InvalidArgument struct {
		Message string
		st      *status.Status
	}
)

// NewInvalidArgument returns new InvalidArgument error.
func NewInvalidArgument(message string) error {
	return &InvalidArgument{
		Message: message,
	}
}

// NewInvalidArgumentf returns new InvalidArgument error with formatted message.
func NewInvalidArgumentf(format string, args ...any) error {
	return &InvalidArgument{
		Message: fmt.Sprintf(format, args...),
	}
}

// Error returns string message.
func (e *InvalidArgument) Error() string {
	return e.Message
}

func (e *InvalidArgument) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	return status.New(codes.InvalidArgument, e.Message)
}

func newInvalidArgument(st *status.Status) error {
	return &InvalidArgument{
		Message: st.Message(),
		st:      st,
	}
}
