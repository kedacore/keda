package serviceerror

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/errordetails/v1"
)

type (
	// ResourceExhausted represents resource exhausted error.
	ResourceExhausted struct {
		Cause   enumspb.ResourceExhaustedCause
		Scope   enumspb.ResourceExhaustedScope
		Message string
		st      *status.Status
	}
)

// NewResourceExhausted returns new ResourceExhausted error.
func NewResourceExhausted(cause enumspb.ResourceExhaustedCause, message string) error {
	return &ResourceExhausted{
		Cause:   cause,
		Message: message,
	}
}

// NewResourceExhaustedf returns new ResourceExhausted error with formatted message.
func NewResourceExhaustedf(cause enumspb.ResourceExhaustedCause, format string, args ...any) error {
	return &ResourceExhausted{
		Cause:   cause,
		Message: fmt.Sprintf(format, args...),
	}
}

// Error returns string message.
func (e *ResourceExhausted) Error() string {
	return e.Message
}

func (e *ResourceExhausted) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	st := status.New(codes.ResourceExhausted, e.Message)
	st, _ = st.WithDetails(
		&errordetails.ResourceExhaustedFailure{
			Cause: e.Cause,
			Scope: e.Scope,
		},
	)
	return st
}

func newResourceExhausted(st *status.Status, errDetails *errordetails.ResourceExhaustedFailure) error {
	return &ResourceExhausted{
		Cause:   errDetails.GetCause(),
		Scope:   errDetails.GetScope(),
		Message: st.Message(),
		st:      st,
	}
}
