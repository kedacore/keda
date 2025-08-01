package serviceerror

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.temporal.io/api/errordetails/v1"
)

type (
	// NamespaceUnavailable is returned by the service when a request addresses a namespace that is unavailable. For
	// example, when a namespace is in the process of failing over between clusters. This is a transient error that
	// should be automatically retried by clients.
	NamespaceUnavailable struct {
		Namespace string
		st        *status.Status
	}
)

// NewNamespaceUnavailable returns new NamespaceUnavailable error.
func NewNamespaceUnavailable(namespace string) error {
	return &NamespaceUnavailable{
		Namespace: namespace,
	}
}

// Error returns string message.
func (e *NamespaceUnavailable) Error() string {
	// No need to do a nil check, that's handled in Message().
	if e.st.Message() != "" {
		return e.st.Message()
	}
	// Continuing the practice of starting errors with upper case and ending with periods even if it's not
	// idiomatic.
	return fmt.Sprintf("Namespace unavailable: %q.", e.Namespace)
}

func (e *NamespaceUnavailable) Status() *status.Status {
	if e.st != nil {
		return e.st
	}
	st := status.New(codes.Unavailable, e.Error())
	// We seem to be okay ignoring these errors everywhere else, doing this here too.
	st, _ = st.WithDetails(
		&errordetails.NamespaceUnavailableFailure{
			Namespace: e.Namespace,
		},
	)
	return st
}

func newNamespaceUnavailable(st *status.Status, errDetails *errordetails.NamespaceUnavailableFailure) error {
	return &NamespaceUnavailable{
		st:        st,
		Namespace: errDetails.GetNamespace(),
	}
}
