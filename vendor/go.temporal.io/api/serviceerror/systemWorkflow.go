// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package serviceerror

import (
	"fmt"

	"github.com/gogo/status"
	"google.golang.org/grpc/codes"

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
