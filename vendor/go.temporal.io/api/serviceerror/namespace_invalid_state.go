package serviceerror

import (
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/errordetails/v1"
)

type (
	// NamespaceInvalidState represents namespace not active error.
	NamespaceInvalidState struct {
		Message       string
		Namespace     string
		State         enumspb.NamespaceState
		AllowedStates []enumspb.NamespaceState
		st            *status.Status
	}
)

// NewNamespaceInvalidState returns new NamespaceInvalidState error.
func NewNamespaceInvalidState(namespace string, state enumspb.NamespaceState, allowedStates []enumspb.NamespaceState) error {
	var allowedStatesStr []string
	for _, allowedState := range allowedStates {
		allowedStatesStr = append(allowedStatesStr, allowedState.String())
	}
	return &NamespaceInvalidState{
		Message: fmt.Sprintf(
			"Namespace has invalid state: %s. Must be %s.",
			state,
			strings.Join(allowedStatesStr, " or "),
		),
		Namespace:     namespace,
		State:         state,
		AllowedStates: allowedStates,
	}
}

// Error returns string message.
func (e *NamespaceInvalidState) Error() string {
	return e.Message
}

func (e *NamespaceInvalidState) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	st := status.New(codes.FailedPrecondition, e.Message)
	st, _ = st.WithDetails(
		&errordetails.NamespaceInvalidStateFailure{
			Namespace:     e.Namespace,
			State:         e.State,
			AllowedStates: e.AllowedStates,
		},
	)
	return st
}

func newNamespaceInvalidState(st *status.Status, errDetails *errordetails.NamespaceInvalidStateFailure) error {
	return &NamespaceInvalidState{
		Message:       st.Message(),
		Namespace:     errDetails.GetNamespace(),
		State:         errDetails.GetState(),
		AllowedStates: errDetails.GetAllowedStates(),
		st:            st,
	}
}
