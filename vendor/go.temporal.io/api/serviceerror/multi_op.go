package serviceerror

import (
	"errors"
	"fmt"

	"go.temporal.io/api/errordetails/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MultiOperationExecution represents a MultiOperationExecution error.
type MultiOperationExecution struct {
	Message string
	errs    []error
	st      *status.Status
}

// NewMultiOperationExecution returns a new MultiOperationExecution error.
func NewMultiOperationExecution(message string, errs []error) error {
	return &MultiOperationExecution{Message: message, errs: errs}
}

// NewMultiOperationExecutionf returns a new MultiOperationExecution error with formatted message.
func NewMultiOperationExecutionf(errs []error, format string, args ...any) error {
	return &MultiOperationExecution{Message: fmt.Sprintf(format, args...), errs: errs}
}

// Error returns string message.
func (e *MultiOperationExecution) Error() string {
	return e.Message
}

func (e *MultiOperationExecution) OperationErrors() []error {
	return e.errs
}

func (e *MultiOperationExecution) Status() *status.Status {
	var code *codes.Code
	failure := &errordetails.MultiOperationExecutionFailure{
		Statuses: make([]*errordetails.MultiOperationExecutionFailure_OperationStatus, len(e.errs)),
	}

	var abortedErr *MultiOperationAborted
	for i, err := range e.errs {
		st := ToStatus(err)

		// the first non-OK and non-Aborted code becomes the code for the entire Status
		if code == nil && st.Code() != codes.OK && !errors.As(err, &abortedErr) {
			c := st.Code()
			code = &c
		}

		failure.Statuses[i] = &errordetails.MultiOperationExecutionFailure_OperationStatus{
			Code:    int32(st.Code()),
			Message: st.Message(),
			Details: st.Proto().Details,
		}
	}

	// this should never happen, but it's better to set it to `Aborted` than to panic
	if code == nil {
		c := codes.Aborted
		code = &c
	}

	st := status.New(*code, e.Error())
	st, _ = st.WithDetails(failure)
	return st
}

func newMultiOperationExecution(st *status.Status, errs []error) error {
	return &MultiOperationExecution{
		Message: st.Message(),
		errs:    errs,
		st:      st,
	}
}
