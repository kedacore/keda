// The MIT License
//
// Copyright (c) 2022 Temporal Technologies Inc.  All rights reserved.
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
	"errors"

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
