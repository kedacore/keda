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
	"go.temporal.io/api/errordetails/v1"
	"google.golang.org/grpc/codes"
)

type (
	// PermissionDenied represents permission denied error.
	PermissionDenied struct {
		Message string
		Reason  string
		st      *status.Status
	}
)

// NewPermissionDenied returns new PermissionDenied error.
func NewPermissionDenied(message, reason string) error {
	return &PermissionDenied{
		Message: message,
		Reason:  reason,
	}
}

// Error returns string message.
func (e *PermissionDenied) Error() string {
	return e.Message
}

func (e *PermissionDenied) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	st := status.New(codes.PermissionDenied, e.Message)
	st, _ = st.WithDetails(
		&errordetails.PermissionDeniedFailure{
			Reason: e.Reason,
		},
	)
	return st
}

func newPermissionDenied(st *status.Status, errDetails *errordetails.PermissionDeniedFailure) error {
	return &PermissionDenied{
		Message: st.Message(),
		Reason:  errDetails.GetReason(),
		st:      st,
	}
}
