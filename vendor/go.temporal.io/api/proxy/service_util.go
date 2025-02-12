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

package proxy

import (
	"context"

	"google.golang.org/grpc/metadata"
)

func (s *workflowServiceProxyServer) reqCtx(ctx context.Context) context.Context {
	if s.disableHeaderForwarding {
		return ctx
	}

	// Copy incoming header to outgoing if not already present in outgoing. We
	// have confirmed in gRPC that incoming is a copy so we can mutate it.
	incoming, _ := metadata.FromIncomingContext(ctx)

	// Remove common headers and if there's nothing left, return early
	incoming.Delete("user-agent")
	incoming.Delete(":authority")
	incoming.Delete("content-type")
	if len(incoming) == 0 {
		return ctx
	}

	// Put all incoming on outgoing if they are not already there. We have
	// confirmed in gRPC that outgoing is a copy so we can mutate it.
	outgoing, _ := metadata.FromOutgoingContext(ctx)
	if outgoing == nil {
		outgoing = metadata.MD{}
	}
	for k, v := range incoming {
		if len(outgoing.Get(k)) == 0 {
			outgoing.Set(k, v...)
		}
	}
	return metadata.NewOutgoingContext(ctx, outgoing)
}
