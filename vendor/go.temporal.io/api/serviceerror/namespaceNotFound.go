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

	"go.temporal.io/api/errordetails/v1"
)

type (
	// NamespaceNotFound represents namespace not found error.
	NamespaceNotFound struct {
		Message   string
		Namespace string
		st        *status.Status
	}
)

// NewNamespaceNotFound returns new NamespaceNotFound error.
func NewNamespaceNotFound(namespace string) error {
	return &NamespaceNotFound{
		Message: fmt.Sprintf(
			"Namespace %s is not found.",
			namespace,
		),
		Namespace: namespace,
	}
}

// Error returns string message.
func (e *NamespaceNotFound) Error() string {
	return e.Message
}

func (e *NamespaceNotFound) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	st := status.New(codes.NotFound, e.Message)
	st, _ = st.WithDetails(
		&errordetails.NamespaceNotFoundFailure{
			Namespace: e.Namespace,
		},
	)
	return st
}

func newNamespaceNotFound(st *status.Status, errDetails *errordetails.NamespaceNotFoundFailure) error {
	return &NamespaceNotFound{
		Message:   st.Message(),
		Namespace: errDetails.GetNamespace(),
		st:        st,
	}
}
