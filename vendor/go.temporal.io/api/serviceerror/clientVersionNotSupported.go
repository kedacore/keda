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
	// ClientVersionNotSupported represents client version is not supported error.
	ClientVersionNotSupported struct {
		Message           string
		ClientVersion     string
		ClientName        string
		SupportedVersions string
		st                *status.Status
	}
)

// NewClientVersionNotSupported returns new ClientVersionNotSupported error.
func NewClientVersionNotSupported(clientVersion, clientName, supportedVersions string) error {
	return &ClientVersionNotSupported{
		Message:           fmt.Sprintf("Client version %s is not supported. Server supports %s versions: %s", clientVersion, clientName, supportedVersions),
		ClientVersion:     clientVersion,
		ClientName:        clientName,
		SupportedVersions: supportedVersions,
	}
}

// Error returns string message.
func (e *ClientVersionNotSupported) Error() string {
	return e.Message
}

func (e *ClientVersionNotSupported) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	st := status.New(codes.FailedPrecondition, e.Message)
	st, _ = st.WithDetails(
		&errordetails.ClientVersionNotSupportedFailure{
			ClientVersion:     e.ClientVersion,
			ClientName:        e.ClientName,
			SupportedVersions: e.SupportedVersions,
		},
	)
	return st
}

func newClientVersionNotSupported(st *status.Status, errDetails *errordetails.ClientVersionNotSupportedFailure) error {
	return &ClientVersionNotSupported{
		Message:           st.Message(),
		ClientVersion:     errDetails.GetClientVersion(),
		ClientName:        errDetails.GetClientName(),
		SupportedVersions: errDetails.GetSupportedVersions(),
		st:                st,
	}
}
