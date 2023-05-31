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
	// ServerVersionNotSupported represents client version is not supported error.
	ServerVersionNotSupported struct {
		Message                       string
		ServerVersion                 string
		ClientSupportedServerVersions string
		st                            *status.Status
	}
)

// NewServerVersionNotSupported returns new ServerVersionNotSupported error.
func NewServerVersionNotSupported(serverVersion, supportedVersions string) error {
	return &ServerVersionNotSupported{
		Message:                       fmt.Sprintf("Server version %s is not supported. Client supports server versions: %s", serverVersion, supportedVersions),
		ServerVersion:                 serverVersion,
		ClientSupportedServerVersions: supportedVersions,
	}
}

// Error returns string message.
func (e *ServerVersionNotSupported) Error() string {
	return e.Message
}

func (e *ServerVersionNotSupported) Status() *status.Status {
	if e.st != nil {
		return e.st
	}

	st := status.New(codes.FailedPrecondition, e.Message)
	st, _ = st.WithDetails(
		&errordetails.ServerVersionNotSupportedFailure{
			ServerVersion:                 e.ServerVersion,
			ClientSupportedServerVersions: e.ClientSupportedServerVersions,
		},
	)
	return st
}

func newServerVersionNotSupported(st *status.Status, errDetails *errordetails.ServerVersionNotSupportedFailure) error {
	return &ServerVersionNotSupported{
		Message:                       st.Message(),
		ServerVersion:                 errDetails.GetServerVersion(),
		ClientSupportedServerVersions: errDetails.GetClientSupportedServerVersions(),
		st:                            st,
	}
}
