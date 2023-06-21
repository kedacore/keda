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
	"strings"

	"github.com/gogo/status"
	"google.golang.org/grpc/codes"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/errordetails/v1"
)

type (
	// NamespaceInvalidState represents namespace not active error.
	NamespaceInvalidState struct {
		Message       string
		Namespace     string
		State         enumspb.NamespaceState
		AllowedStates []enumspb.NamespaceState
		st            *status.Status
	}
)

// NewNamespaceInvalidState returns new NamespaceInvalidState error.
func NewNamespaceInvalidState(namespace string, state enumspb.NamespaceState, allowedStates []enumspb.NamespaceState) error {
	var allowedStatesStr []string
	for _, allowedState := range allowedStates {
		allowedStatesStr = append(allowedStatesStr, allowedState.String())
	}
	return &NamespaceInvalidState{
		Message: fmt.Sprintf(
			"Namespace has invalid state: %s. Must be %s.",
			state,
			strings.Join(allowedStatesStr, " or "),
		),
		Namespace:     namespace,
		State:         state,
		AllowedStates: allowedStates,
	}
}

// Error returns string message.
func (e *NamespaceInvalidState) Error() string {
	return e.Message
}

func (e *NamespaceInvalidState) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	st := status.New(codes.FailedPrecondition, e.Message)
	st, _ = st.WithDetails(
		&errordetails.NamespaceInvalidStateFailure{
			Namespace:     e.Namespace,
			State:         e.State,
			AllowedStates: e.AllowedStates,
		},
	)
	return st
}

func newNamespaceInvalidState(st *status.Status, errDetails *errordetails.NamespaceInvalidStateFailure) error {
	return &NamespaceInvalidState{
		Message:       st.Message(),
		Namespace:     errDetails.GetNamespace(),
		State:         errDetails.GetState(),
		AllowedStates: errDetails.GetAllowedStates(),
		st:            st,
	}
}
