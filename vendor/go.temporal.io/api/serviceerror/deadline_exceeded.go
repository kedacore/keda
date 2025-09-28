package serviceerror

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	// DeadlineExceeded represents deadline exceeded error.
	DeadlineExceeded struct {
		Message string
		st      *status.Status
	}
)

// NewDeadlineExceeded returns new DeadlineExceeded error.
func NewDeadlineExceeded(message string) error {
	return &DeadlineExceeded{
		Message: message,
	}
}

// NewDeadlineExceededf returns new DeadlineExceeded error with formatted message.
func NewDeadlineExceededf(format string, args ...any) error {
	return &DeadlineExceeded{
		Message: fmt.Sprintf(format, args...),
	}
}

// Error returns string message.
func (e *DeadlineExceeded) Error() string {
	return e.Message
}

func (e *DeadlineExceeded) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	return status.New(codes.DeadlineExceeded, e.Message)
}

func newDeadlineExceeded(st *status.Status) error {
	return &DeadlineExceeded{
		Message: st.Message(),
		st:      st,
	}
}
