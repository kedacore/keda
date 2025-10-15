package serviceerror

import (
	"fmt"

	"go.temporal.io/api/errordetails/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	// PermissionDenied represents permission denied error.
	PermissionDenied struct {
		Message string
		Reason  string
		st      *status.Status
	}
)

// NewPermissionDenied returns new PermissionDenied error.
func NewPermissionDenied(message, reason string) error {
	return &PermissionDenied{
		Message: message,
		Reason:  reason,
	}
}

// NewPermissionDeniedf returns new PermissionDenied error with formatted message.
func NewPermissionDeniedf(reason, format string, args ...any) error {
	return &PermissionDenied{
		Message: fmt.Sprintf(format, args...),
		Reason:  reason,
	}
}

// Error returns string message.
func (e *PermissionDenied) Error() string {
	return e.Message
}

func (e *PermissionDenied) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	st := status.New(codes.PermissionDenied, e.Message)
	st, _ = st.WithDetails(
		&errordetails.PermissionDeniedFailure{
			Reason: e.Reason,
		},
	)
	return st
}

func newPermissionDenied(st *status.Status, errDetails *errordetails.PermissionDeniedFailure) error {
	return &PermissionDenied{
		Message: st.Message(),
		Reason:  errDetails.GetReason(),
		st:      st,
	}
}
