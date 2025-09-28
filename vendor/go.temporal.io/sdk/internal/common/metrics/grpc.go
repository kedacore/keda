package metrics

import (
	"context"
	"strings"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// HandlerContextKey is the context key for a MetricHandler value.
type HandlerContextKey struct{}

// LongPollContextKey is the context key for a boolean stating whether the gRPC
// call is a long poll.
type LongPollContextKey struct{}

// NewGRPCInterceptor creates a new gRPC unary interceptor to record metrics.
func NewGRPCInterceptor(defaultHandler Handler, suffix string, disableRequestFailCodes bool) grpc.UnaryClientInterceptor {
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

		// Since this interceptor can be used for clients of different name, we
		// attempt to extract the namespace out of the request. All namespace-based
		// requests have been confirmed to have a top-level namespace field.
		namespace := "_unknown_"
		if nsReq, _ := req.(interface{ GetNamespace() string }); nsReq != nil {
			namespace = nsReq.GetNamespace()
		}

		// Capture time, record start, run, and record end
		tags := map[string]string{OperationTagName: operation, NamespaceTagName: namespace}
		handler = handler.WithTags(tags)
		start := time.Now()
		recordRequestStart(handler, longPoll, suffix)
		err := invoker(ctx, method, req, reply, cc, opts...)
		recordRequestEnd(handler, longPoll, suffix, start, err, disableRequestFailCodes)
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

func recordRequestEnd(handler Handler, longPoll bool, suffix string, start time.Time, err error, disableRequestFailCodes bool) {
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
		errStatus, _ := status.FromError(err)
		if !disableRequestFailCodes {
			handler = handler.WithTags(RequestFailureCodeTags(errStatus.Code()))
		}
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
