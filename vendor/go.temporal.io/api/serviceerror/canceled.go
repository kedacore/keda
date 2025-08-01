package serviceerror

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	// Canceled represents canceled error.
	Canceled struct {
		Message string
		st      *status.Status
	}
)

// NewCanceled returns new Canceled error.
func NewCanceled(message string) error {
	return &Canceled{
		Message: message,
	}
}

// Error returns string message.
func (e *Canceled) Error() string {
	return e.Message
}

func (e *Canceled) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	return status.New(codes.Canceled, e.Message)
}

func newCanceled(st *status.Status) error {
	return &Canceled{
		Message: st.Message(),
		st:      st,
	}
}
