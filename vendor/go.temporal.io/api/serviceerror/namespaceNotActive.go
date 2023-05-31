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
	// NamespaceNotActive represents namespace not active error.
	NamespaceNotActive struct {
		Message        string
		Namespace      string
		CurrentCluster string
		ActiveCluster  string
		st             *status.Status
	}
)

// NewNamespaceNotActive returns new NamespaceNotActive error.
func NewNamespaceNotActive(namespace, currentCluster, activeCluster string) error {
	return &NamespaceNotActive{
		Message: fmt.Sprintf(
			"Namespace: %s is active in cluster: %s, while current cluster %s is a standby cluster.",
			namespace,
			activeCluster,
			currentCluster,
		),
		Namespace:      namespace,
		CurrentCluster: currentCluster,
		ActiveCluster:  activeCluster,
	}
}

// Error returns string message.
func (e *NamespaceNotActive) Error() string {
	return e.Message
}

func (e *NamespaceNotActive) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	st := status.New(codes.FailedPrecondition, e.Message)
	st, _ = st.WithDetails(
		&errordetails.NamespaceNotActiveFailure{
			Namespace:      e.Namespace,
			CurrentCluster: e.CurrentCluster,
			ActiveCluster:  e.ActiveCluster,
		},
	)
	return st
}

func newNamespaceNotActive(st *status.Status, errDetails *errordetails.NamespaceNotActiveFailure) error {
	return &NamespaceNotActive{
		Message:        st.Message(),
		Namespace:      errDetails.GetNamespace(),
		CurrentCluster: errDetails.GetCurrentCluster(),
		ActiveCluster:  errDetails.GetActiveCluster(),
		st:             st,
	}
}
