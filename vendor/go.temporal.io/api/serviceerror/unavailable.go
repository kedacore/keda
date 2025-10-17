package serviceerror

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	// Unavailable represents unavailable error.
	Unavailable struct {
		Message string
		st      *status.Status
	}
)

// NewUnavailable returns new Unavailable error.
func NewUnavailable(message string) error {
	return &Unavailable{
		Message: message,
	}
}

// NewUnavailablef returns new Unavailable error with formatted message.
func NewUnavailablef(format string, args ...any) error {
	return &Unavailable{
		Message: fmt.Sprintf(format, args...),
	}
}

// Error returns string message.
func (e *Unavailable) Error() string {
	return e.Message
}

func (e *Unavailable) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	return status.New(codes.Unavailable, e.Message)
}

func newUnavailable(st *status.Status) error {
	return &Unavailable{
		Message: st.Message(),
		st:      st,
	}
}
