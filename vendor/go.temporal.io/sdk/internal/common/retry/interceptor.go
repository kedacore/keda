package retry

import (
	"context"
	"math"
	"strings"
	"sync/atomic"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/util/backoffutils"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/errordetails/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// UnlimitedMaximumAttempts when maximum attempts is set to this special value, then the number of attempts is unlimited.
	UnlimitedMaximumAttempts = 0
	// UnlimitedInterval when maximum interval is set to this special value, then there is no upper bound on the retry delay.
	// Should not be used together with unlimited attempts as resulting retry interval can grow to unreasonable values.
	UnlimitedInterval = 0
	// DefaultBackoffCoefficient is default backOffCoefficient for retryPolicy
	DefaultBackoffCoefficient = 2.0
	// DefaultMaximumInterval is default maximum amount of time for an individual retry.
	DefaultMaximumInterval = 10 * time.Second
	// DefaultExpirationInterval is default expiration time for all retry attempts.
	DefaultExpirationInterval = time.Minute
	// DefaultMaximumAttempts is default maximum number of attempts.
	DefaultMaximumAttempts = UnlimitedMaximumAttempts
	// DefaultJitter is a default jitter applied on the backoff interval for delay randomization.
	DefaultJitter = 0.2
)

type (
	// GrpcRetryConfig defines required configuration for exponential backoff function that is supplied to gRPC retrier.
	GrpcRetryConfig struct {
		initialInterval    time.Duration
		backoffCoefficient float64
		maximumInterval    time.Duration
		expirationInterval time.Duration
		jitter             float64
		maximumAttempts    int
	}

	contextKey struct{}
)

func (ck contextKey) String() string {
	return "RetryConfig"
}

// SetBackoffCoefficient sets rate at which backoff coefficient will change.
func (g *GrpcRetryConfig) SetBackoffCoefficient(backoffCoefficient float64) {
	g.backoffCoefficient = backoffCoefficient
}

// SetMaximumInterval defines maximum amount of time between attempts.
func (g *GrpcRetryConfig) SetMaximumInterval(maximumInterval time.Duration) {
	g.maximumInterval = maximumInterval
}

// SetExpirationInterval defines total amount of time that can be used for all retry attempts.
// Note that this value is ignored if deadline is set on the context.
func (g *GrpcRetryConfig) SetExpirationInterval(expirationInterval time.Duration) {
	g.expirationInterval = expirationInterval
}

// SetJitter defines level of randomization for each delay interval. For example 0.2 would mex target +- 20%
func (g *GrpcRetryConfig) SetJitter(jitter float64) {
	g.jitter = jitter
}

// SetMaximumAttempts defines maximum total number of retry attempts.
func (g *GrpcRetryConfig) SetMaximumAttempts(maximumAttempts int) {
	g.maximumAttempts = maximumAttempts
}

// NewGrpcRetryConfig creates new retry config with specified initial interval and defaults for other parameters.
// Use SetXXX functions on this config in order to customize values.
func NewGrpcRetryConfig(initialInterval time.Duration) *GrpcRetryConfig {
	return &GrpcRetryConfig{
		initialInterval:    initialInterval,
		backoffCoefficient: DefaultBackoffCoefficient,
		maximumInterval:    DefaultMaximumInterval,
		expirationInterval: DefaultExpirationInterval,
		jitter:             DefaultJitter,
		maximumAttempts:    DefaultMaximumAttempts,
	}
}

var (
	// ConfigKey context key for GrpcRetryConfig
	ConfigKey = contextKey{}
	// gRPC response codes that represent unconditionally retryable errors.
	// The following status codes are never retried by the library:
	//    INVALID_ARGUMENT, NOT_FOUND, ALREADY_EXISTS, FAILED_PRECONDITION, ABORTED, OUT_OF_RANGE, DATA_LOSS
	// codes.DeadlineExceeded and codes.Canceled are not here (and shouldn't be here!)
	// because they are coming from go context and "context errors are not retriable based on user settings"
	// by gRPC library.
	// codes.Internal and codes.ResourceExhausted have special logic for whether they are retryable or not,
	// and so they're not included in this list.
	alwaysRetryableCodes = []codes.Code{codes.Aborted, codes.Unavailable, codes.Unknown}
)

// NewRetryOptionsInterceptor creates a new gRPC interceptor that populates retry options for each call based on values
// provided in the context. The atomic bool is checked each call to determine whether internals are included in retry.
// If not present or false, internals are assumed to be included.
func NewRetryOptionsInterceptor(excludeInternal *atomic.Bool) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if rc, ok := ctx.Value(ConfigKey).(*GrpcRetryConfig); ok {
			if _, ok := ctx.Deadline(); !ok {
				deadlineCtx, cancel := context.WithDeadline(ctx, time.Now().Add(rc.expirationInterval))
				defer cancel()
				ctx = deadlineCtx
			}
			// Populate backoff function, which provides retrier with the delay for each attempt.
			opts = append(opts, grpc_retry.WithBackoff(func(_ context.Context, attempt uint) time.Duration {
				next := float64(rc.initialInterval) * math.Pow(rc.backoffCoefficient, float64(attempt))
				if rc.maximumInterval != UnlimitedInterval {
					next = math.Min(next, float64(rc.maximumInterval))
				}
				return backoffutils.JitterUp(time.Duration(next), rc.jitter)
			}))
			// Max attempts is a required parameter in grpc retry interceptor,
			// if it's set to zero then no retries will be made.
			if rc.maximumAttempts != UnlimitedMaximumAttempts {
				opts = append(opts, grpc_retry.WithMax(uint(rc.maximumAttempts)))
			} else {
				opts = append(opts, grpc_retry.WithMax(math.MaxUint32))
			}
			opts = append(opts, grpc_retry.WithRetriable(func(err error) bool {
				return IsRetryable(status.Convert(err), excludeInternal)
			}))
		} else {
			// Do not retry if retry config is not set.
			opts = append(opts, grpc_retry.Disable())
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func IsRetryable(status *status.Status, excludeInternalFromRetry *atomic.Bool) bool {
	if status == nil {
		return false
	}
	errCode := status.Code()
	for _, retryable := range alwaysRetryableCodes {
		if errCode == retryable {
			return true
		}
	}
	if errCode == codes.Internal {
		return !excludeInternalFromRetry.Load()
	}
	if errCode == codes.ResourceExhausted {
		if details := status.Details(); len(details) > 0 {
			if failure, ok := details[0].(*errordetails.ResourceExhaustedFailure); ok {
				return failure.Cause != RESOURCE_EXHAUSTED_CAUSE_EXT_GRPC_MESSAGE_TOO_LARGE
			}
		}
	}
	return false
}

// RESOURCE_EXHAUSTED_CAUSE_EXT_GRPC_MESSAGE_TOO_LARGE is an extension to the ResourceExhaustedCause enum to mark gRPC message too large errors.
const RESOURCE_EXHAUSTED_CAUSE_EXT_GRPC_MESSAGE_TOO_LARGE enumspb.ResourceExhaustedCause = 101 // TODO: add the cause to the upstream API repo and remove this (see https://github.com/temporalio/sdk-go/issues/2030)

// SetGrpcMessageTooLargeErrorCauseInterceptor adds appropriate error details if the error cause is gRPC message being too large.
func SetGrpcMessageTooLargeErrorCauseInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	err := invoker(ctx, method, req, reply, cc, opts...)
	if grpcStatus := status.Convert(err); isGrpcMessageTooLargeError(grpcStatus) {
		// Copying code and message but ignoring original details
		newStatus := status.New(grpcStatus.Code(), grpcStatus.Message())
		newStatus, detailsErr := newStatus.WithDetails(&errordetails.ResourceExhaustedFailure{
			Cause: RESOURCE_EXHAUSTED_CAUSE_EXT_GRPC_MESSAGE_TOO_LARGE,
		})
		if detailsErr == nil {
			if newErr := newStatus.Err(); newErr != nil {
				err = newErr
			}
		}
	}
	return err
}

func isGrpcMessageTooLargeError(status *status.Status) bool {
	if status == nil {
		return false
	}
	message := status.Message()
	return strings.HasPrefix(message, "grpc: received message larger than max") ||
		strings.HasPrefix(message, "grpc: message after decompression larger than max") ||
		strings.HasPrefix(message, "grpc: received message after decompression larger than max")
}
