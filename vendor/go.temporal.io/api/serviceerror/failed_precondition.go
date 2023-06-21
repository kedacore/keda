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
	"github.com/gogo/status"
	"google.golang.org/grpc/codes"
)

type (
	// FailedPrecondition represents failed precondition error.
	FailedPrecondition struct {
		Message string
		st      *status.Status
	}
)

// NewFailedPrecondition returns new FailedPrecondition error.
func NewFailedPrecondition(message string) error {
	return &FailedPrecondition{
		Message: message,
	}
}

// Error returns string message.
func (e *FailedPrecondition) Error() string {
	return e.Message
}

func (e *FailedPrecondition) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	return status.New(codes.FailedPrecondition, e.Message)
}

func newFailedPrecondition(st *status.Status) error {
	return &FailedPrecondition{
		Message: st.Message(),
		st:      st,
	}
}
