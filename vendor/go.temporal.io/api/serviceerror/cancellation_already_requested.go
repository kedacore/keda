package serviceerror

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.temporal.io/api/errordetails/v1"
)

type (
	// CancellationAlreadyRequested represents cancellation already requested error.
	CancellationAlreadyRequested struct {
		Message string
		st      *status.Status
	}
)

// NewCancellationAlreadyRequested returns new CancellationAlreadyRequested error.
func NewCancellationAlreadyRequested(message string) error {
	return &CancellationAlreadyRequested{
		Message: message,
	}
}

// NewCancellationAlreadyRequestedf returns new CancellationAlreadyRequested error with formatted message.
func NewCancellationAlreadyRequestedf(format string, args ...any) error {
	return &CancellationAlreadyRequested{
		Message: fmt.Sprintf(format, args...),
	}
}

// Error returns string message.
func (e *CancellationAlreadyRequested) Error() string {
	return e.Message
}

func (e *CancellationAlreadyRequested) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	st := status.New(codes.AlreadyExists, e.Message)
	st, _ = st.WithDetails(
		&errordetails.CancellationAlreadyRequestedFailure{},
	)
	return st
}

func newCancellationAlreadyRequested(st *status.Status) error {
	return &CancellationAlreadyRequested{
		Message: st.Message(),
		st:      st,
	}
}
