package serviceerror

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.temporal.io/api/errordetails/v1"
)

type (
	// NotFound represents not found error.
	NotFound struct {
		Message        string
		CurrentCluster string
		ActiveCluster  string
		st             *status.Status
	}
)

// NewNotFound returns new NotFound error.
func NewNotFound(message string) error {
	return &NotFound{
		Message: message,
	}
}

// NewNotFoundf returns new NotFound error with formatted message.
func NewNotFoundf(format string, args ...any) error {
	return &NotFound{
		Message: fmt.Sprintf(format, args...),
	}
}

// Error returns string message.
func (e *NotFound) Error() string {
	return e.Message
}

func (e *NotFound) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	st := status.New(codes.NotFound, e.Message)
	st, _ = st.WithDetails(
		&errordetails.NotFoundFailure{
			CurrentCluster: e.CurrentCluster,
			ActiveCluster:  e.ActiveCluster,
		},
	)
	return st
}

func newNotFound(st *status.Status, errDetails *errordetails.NotFoundFailure) error {
	return &NotFound{
		Message:        st.Message(),
		CurrentCluster: errDetails.GetCurrentCluster(),
		ActiveCluster:  errDetails.GetActiveCluster(),
		st:             st,
	}
}
