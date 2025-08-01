package serviceerror

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	// FailedPrecondition represents failed precondition error.
	FailedPrecondition struct {
		Message string
		st      *status.Status
	}
)

// NewFailedPrecondition returns new FailedPrecondition error.
func NewFailedPrecondition(message string) error {
	return &FailedPrecondition{
		Message: message,
	}
}

// Error returns string message.
func (e *FailedPrecondition) Error() string {
	return e.Message
}

func (e *FailedPrecondition) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	return status.New(codes.FailedPrecondition, e.Message)
}

func newFailedPrecondition(st *status.Status) error {
	return &FailedPrecondition{
		Message: st.Message(),
		st:      st,
	}
}
