package serviceerror

import (
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
