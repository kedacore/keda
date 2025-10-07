package serviceerror

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	// Aborted represents an aborted error.
	Aborted struct {
		Message string
		st      *status.Status
	}
)

// NewAborted returns new Aborted error.
func NewAborted(message string) error {
	return &Aborted{
		Message: message,
	}
}

// NewAbortedf returns new Aborted error with formatted message.
func NewAbortedf(format string, args ...any) error {
	return &Aborted{
		Message: fmt.Sprintf(format, args...),
	}
}

// Error returns string message.
func (e *Aborted) Error() string {
	return e.Message
}

func (e *Aborted) Status() *status.Status {
	if e.st != nil {
		return e.st
	}
	return status.New(codes.Aborted, e.Message)
}

func newAborted(st *status.Status) error {
	return &Aborted{
		Message: st.Message(),
		st:      st,
	}
}
