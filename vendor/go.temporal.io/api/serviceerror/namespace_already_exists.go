package serviceerror

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.temporal.io/api/errordetails/v1"
)

type (
	// NamespaceAlreadyExists represents namespace already exists error.
	NamespaceAlreadyExists struct {
		Message string
		st      *status.Status
	}
)

// NewNamespaceAlreadyExists returns new NamespaceAlreadyExists error.
func NewNamespaceAlreadyExists(message string) error {
	return &NamespaceAlreadyExists{
		Message: message,
	}
}

// Error returns string message.
func (e *NamespaceAlreadyExists) Error() string {
	return e.Message
}

func (e *NamespaceAlreadyExists) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	st := status.New(codes.AlreadyExists, e.Message)
	st, _ = st.WithDetails(
		&errordetails.NamespaceAlreadyExistsFailure{},
	)
	return st
}

func newNamespaceAlreadyExists(st *status.Status) error {
	return &NamespaceAlreadyExists{
		Message: st.Message(),
		st:      st,
	}
}
