package serviceerror

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.temporal.io/api/errordetails/v1"
)

type (
	// NexusOperationExecutionAlreadyStarted represents a nexus operation execution already started error.
	NexusOperationExecutionAlreadyStarted struct {
		Message        string
		StartRequestId string
		RunId          string
		st             *status.Status
	}
)

// NewNexusOperationExecutionAlreadyStarted returns new NexusOperationExecutionAlreadyStarted error.
func NewNexusOperationExecutionAlreadyStarted(message, startRequestId, runId string) error {
	return &NexusOperationExecutionAlreadyStarted{
		Message:        message,
		StartRequestId: startRequestId,
		RunId:          runId,
	}
}

// NewNexusOperationExecutionAlreadyStartedf returns new NexusOperationExecutionAlreadyStarted error with formatted message.
func NewNexusOperationExecutionAlreadyStartedf(startRequestId, runId, format string, args ...any) error {
	return &NexusOperationExecutionAlreadyStarted{
		Message:        fmt.Sprintf(format, args...),
		StartRequestId: startRequestId,
		RunId:          runId,
	}
}

// Error returns string message.
func (e *NexusOperationExecutionAlreadyStarted) Error() string {
	return e.Message
}

func (e *NexusOperationExecutionAlreadyStarted) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	st := status.New(codes.AlreadyExists, e.Message)
	st, _ = st.WithDetails(
		&errordetails.NexusOperationExecutionAlreadyStartedFailure{
			StartRequestId: e.StartRequestId,
			RunId:          e.RunId,
		},
	)
	return st
}

func newNexusOperationExecutionAlreadyStarted(st *status.Status, errDetails *errordetails.NexusOperationExecutionAlreadyStartedFailure) error {
	return &NexusOperationExecutionAlreadyStarted{
		Message:        st.Message(),
		StartRequestId: errDetails.GetStartRequestId(),
		RunId:          errDetails.GetRunId(),
		st:             st,
	}
}
