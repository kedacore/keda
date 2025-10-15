package serviceerror

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/errordetails/v1"
)

type (
	// SystemWorkflow represents an error that happens during execution of the underlying system workflow
	SystemWorkflow struct {
		WorkflowExecution *common.WorkflowExecution
		WorkflowError     string
		st                *status.Status
	}
)

// NewSystemWorkflow returns new SystemWorkflow error.
func NewSystemWorkflow(workflowExecution *common.WorkflowExecution, workflowError error) error {
	return &SystemWorkflow{
		WorkflowExecution: workflowExecution,
		WorkflowError:     workflowError.Error(),
	}
}

// NewSystemWorkflowf returns new SystemWorkflow error with formatted workflow error.
func NewSystemWorkflowf(workflowExecution *common.WorkflowExecution, format string, args ...any) error {
	return &SystemWorkflow{
		WorkflowExecution: workflowExecution,
		WorkflowError:     fmt.Sprintf(format, args...),
	}
}

// Error returns string message.
func (e *SystemWorkflow) Error() string {
	execution := e.WorkflowExecution
	return fmt.Sprintf("System Workflow with WorkflowId %s and RunId %s returned an error: %s",
		execution.WorkflowId, execution.RunId, e.WorkflowError)
}

func (e *SystemWorkflow) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	st := status.New(codes.Internal, e.Error())
	st, _ = st.WithDetails(
		&errordetails.SystemWorkflowFailure{
			WorkflowExecution: e.WorkflowExecution,
			WorkflowError:     e.WorkflowError,
		},
	)
	return st
}

func newSystemWorkflow(st *status.Status, errDetails *errordetails.SystemWorkflowFailure) error {
	return &SystemWorkflow{
		WorkflowExecution: errDetails.WorkflowExecution,
		WorkflowError:     errDetails.WorkflowError,
		st:                st,
	}
}
