package serviceerror

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.temporal.io/api/errordetails/v1"
)

type (
	// NamespaceNotActive represents namespace not active error.
	NamespaceNotActive struct {
		Message        string
		Namespace      string
		CurrentCluster string
		ActiveCluster  string
		st             *status.Status
	}
)

// NewNamespaceNotActive returns new NamespaceNotActive error.
func NewNamespaceNotActive(namespace, currentCluster, activeCluster string) error {
	return &NamespaceNotActive{
		Message: fmt.Sprintf(
			"Namespace: %s is active in cluster: %s, while current cluster %s is a standby cluster.",
			namespace,
			activeCluster,
			currentCluster,
		),
		Namespace:      namespace,
		CurrentCluster: currentCluster,
		ActiveCluster:  activeCluster,
	}
}

// Error returns string message.
func (e *NamespaceNotActive) Error() string {
	return e.Message
}

func (e *NamespaceNotActive) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	st := status.New(codes.FailedPrecondition, e.Message)
	st, _ = st.WithDetails(
		&errordetails.NamespaceNotActiveFailure{
			Namespace:      e.Namespace,
			CurrentCluster: e.CurrentCluster,
			ActiveCluster:  e.ActiveCluster,
		},
	)
	return st
}

func newNamespaceNotActive(st *status.Status, errDetails *errordetails.NamespaceNotActiveFailure) error {
	return &NamespaceNotActive{
		Message:        st.Message(),
		Namespace:      errDetails.GetNamespace(),
		CurrentCluster: errDetails.GetCurrentCluster(),
		ActiveCluster:  errDetails.GetActiveCluster(),
		st:             st,
	}
}
