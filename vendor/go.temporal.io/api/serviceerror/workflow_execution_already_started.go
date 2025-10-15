package serviceerror

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.temporal.io/api/errordetails/v1"
)

type (
	// WorkflowExecutionAlreadyStarted represents workflow execution already started error.
	WorkflowExecutionAlreadyStarted struct {
		Message        string
		StartRequestId string
		RunId          string
		st             *status.Status
	}
)

// NewWorkflowExecutionAlreadyStarted returns new WorkflowExecutionAlreadyStarted error.
func NewWorkflowExecutionAlreadyStarted(message, startRequestId, runId string) error {
	return &WorkflowExecutionAlreadyStarted{
		Message:        message,
		StartRequestId: startRequestId,
		RunId:          runId,
	}
}

// NewWorkflowExecutionAlreadyStartedf returns new WorkflowExecutionAlreadyStarted error with formatted message.
func NewWorkflowExecutionAlreadyStartedf(startRequestId, runId, format string, args ...any) error {
	return &WorkflowExecutionAlreadyStarted{
		Message:        fmt.Sprintf(format, args...),
		StartRequestId: startRequestId,
		RunId:          runId,
	}
}

// Error returns string message.
func (e *WorkflowExecutionAlreadyStarted) Error() string {
	return e.Message
}

func (e *WorkflowExecutionAlreadyStarted) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	st := status.New(codes.AlreadyExists, e.Message)
	st, _ = st.WithDetails(
		&errordetails.WorkflowExecutionAlreadyStartedFailure{
			StartRequestId: e.StartRequestId,
			RunId:          e.RunId,
		},
	)
	return st
}

func newWorkflowExecutionAlreadyStarted(st *status.Status, errDetails *errordetails.WorkflowExecutionAlreadyStartedFailure) error {
	return &WorkflowExecutionAlreadyStarted{
		Message:        st.Message(),
		StartRequestId: errDetails.GetStartRequestId(),
		RunId:          errDetails.GetRunId(),
		st:             st,
	}
}
