// The MIT License
//
// Copyright (c) 2022 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
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

package client

import (
	"go.temporal.io/api/proxy"
	"go.temporal.io/api/workflowservice/v1"
)

// WorkflowServiceProxyOptions provides options for configuring a WorkflowServiceProxyServer.
// Client is a WorkflowServiceClient used to forward requests received by the server to the
// Temporal Frontend.
type WorkflowServiceProxyOptions struct {
	Client workflowservice.WorkflowServiceClient
}

// NewWorkflowServiceProxyServer creates a WorkflowServiceServer suitable for registering with a GRPC Server. Requests will
// be forwarded to the passed in WorkflowService Client. GRPC interceptors can be added on the Server or Client to adjust
// requests and responses.
func NewWorkflowServiceProxyServer(options WorkflowServiceProxyOptions) (workflowservice.WorkflowServiceServer, error) {
	return proxy.NewWorkflowServiceProxyServer(proxy.WorkflowServiceProxyOptions(options))
}
