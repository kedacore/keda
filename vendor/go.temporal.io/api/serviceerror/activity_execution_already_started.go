package serviceerror

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.temporal.io/api/errordetails/v1"
)

type (
	// ActivityExecutionAlreadyStarted represents workflow execution already started error.
	ActivityExecutionAlreadyStarted struct {
		Message        string
		StartRequestId string
		RunId          string
		st             *status.Status
	}
)

// NewActivityExecutionAlreadyStarted returns new ActivityExecutionAlreadyStarted error.
func NewActivityExecutionAlreadyStarted(message, startRequestId, runId string) error {
	return &ActivityExecutionAlreadyStarted{
		Message:        message,
		StartRequestId: startRequestId,
		RunId:          runId,
	}
}

// NewActivityExecutionAlreadyStartedf returns new ActivityExecutionAlreadyStarted error with formatted message.
func NewActivityExecutionAlreadyStartedf(startRequestId, runId, format string, args ...any) error {
	return &ActivityExecutionAlreadyStarted{
		Message:        fmt.Sprintf(format, args...),
		StartRequestId: startRequestId,
		RunId:          runId,
	}
}

// Error returns string message.
func (e *ActivityExecutionAlreadyStarted) Error() string {
	return e.Message
}

func (e *ActivityExecutionAlreadyStarted) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	st := status.New(codes.AlreadyExists, e.Message)
	st, _ = st.WithDetails(
		&errordetails.ActivityExecutionAlreadyStartedFailure{
			StartRequestId: e.StartRequestId,
			RunId:          e.RunId,
		},
	)
	return st
}

func newActivityExecutionAlreadyStarted(st *status.Status, errDetails *errordetails.ActivityExecutionAlreadyStartedFailure) error {
	return &ActivityExecutionAlreadyStarted{
		Message:        st.Message(),
		StartRequestId: errDetails.GetStartRequestId(),
		RunId:          errDetails.GetRunId(),
		st:             st,
	}
}
