// The MIT License
//
// Copyright (c) 2021 Temporal Technologies Inc.  All rights reserved.
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

package metrics

import (
	"context"
	"strings"
	"time"

	"github.com/gogo/status"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// HandlerContextKey is the context key for a MetricHandler value.
type HandlerContextKey struct{}

// LongPollContextKey is the context key for a boolean stating whether the gRPC
// call is a long poll.
type LongPollContextKey struct{}

// NewGRPCInterceptor creates a new gRPC unary interceptor to record metrics.
func NewGRPCInterceptor(defaultHandler Handler, suffix string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req interface{},
		reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		handler, _ := ctx.Value(HandlerContextKey{}).(Handler)
		if handler == nil {
			handler = defaultHandler
		}
		longPoll, ok := ctx.Value(LongPollContextKey{}).(bool)
		if !ok {
			longPoll = false
		}

		// Only take method name after the last slash
		operation := method[strings.LastIndex(method, "/")+1:]
		tags := map[string]string{OperationTagName: operation}

		// Since this interceptor can be used for clients of different name, we
		// attempt to extract the namespace out of the request. All namespace-based
		// requests have been confirmed to have a top-level namespace field.
		if nsReq, _ := req.(interface{ GetNamespace() string }); nsReq != nil {
			tags[NamespaceTagName] = nsReq.GetNamespace()
		}

		// Capture time, record start, run, and record end
		handler = handler.WithTags(tags)
		start := time.Now()
		recordRequestStart(handler, longPoll, suffix)
		err := invoker(ctx, method, req, reply, cc, opts...)
		recordRequestEnd(handler, longPoll, suffix, start, err)
		return err
	}
}

func recordRequestStart(handler Handler, longPoll bool, suffix string) {
	// Count request
	metric := TemporalRequest
	if longPoll {
		metric = TemporalLongRequest
	}
	metric += suffix
	handler.Counter(metric).Inc(1)
}

func recordRequestEnd(handler Handler, longPoll bool, suffix string, start time.Time, err error) {
	// Record latency
	timerMetric := TemporalRequestLatency
	if longPoll {
		timerMetric = TemporalLongRequestLatency
	}
	timerMetric += suffix
	handler.Timer(timerMetric).Record(time.Since(start))

	// Count failure
	if err != nil {
		failureMetric := TemporalRequestFailure
		if longPoll {
			failureMetric = TemporalLongRequestFailure
		}
		failureMetric += suffix
		handler.Counter(failureMetric).Inc(1)

		// If it's a resource exhausted, extract cause if present and increment
		if s := status.Convert(err); s.Code() == codes.ResourceExhausted {
			resMetric := TemporalRequestResourceExhausted
			if longPoll {
				resMetric = TemporalLongRequestResourceExhausted
			}
			var cause enumspb.ResourceExhaustedCause
			if resErr, _ := serviceerror.FromStatus(s).(*serviceerror.ResourceExhausted); resErr != nil {
				cause = resErr.Cause
			}
			handler.WithTags(map[string]string{CauseTagName: cause.String()}).Counter(resMetric).Inc(1)
		}
	}
}
