// The MIT License
//
// Copyright (c) 2024 Temporal Technologies Inc.  All rights reserved.
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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.temporal.io/api/errordetails/v1"
)

type (
	// NamespaceUnavailable is returned by the service when a request addresses a namespace that is unavailable. For
	// example, when a namespace is in the process of failing over between clusters. This is a transient error that
	// should be automatically retried by clients.
	NamespaceUnavailable struct {
		Namespace string
		st        *status.Status
	}
)

// NewNamespaceUnavailable returns new NamespaceUnavailable error.
func NewNamespaceUnavailable(namespace string) error {
	return &NamespaceUnavailable{
		Namespace: namespace,
	}
}

// Error returns string message.
func (e *NamespaceUnavailable) Error() string {
	// No need to do a nil check, that's handled in Message().
	if e.st.Message() != "" {
		return e.st.Message()
	}
	// Continuing the practice of starting errors with upper case and ending with periods even if it's not
	// idiomatic.
	return fmt.Sprintf("Namespace unavailable: %q.", e.Namespace)
}

func (e *NamespaceUnavailable) Status() *status.Status {
	if e.st != nil {
		return e.st
	}
	st := status.New(codes.Unavailable, e.Error())
	// We seem to be okay ignoring these errors everywhere else, doing this here too.
	st, _ = st.WithDetails(
		&errordetails.NamespaceUnavailableFailure{
			Namespace: e.Namespace,
		},
	)
	return st
}

func newNamespaceUnavailable(st *status.Status, errDetails *errordetails.NamespaceUnavailableFailure) error {
	return &NamespaceUnavailable{
		st:        st,
		Namespace: errDetails.GetNamespace(),
	}
}
