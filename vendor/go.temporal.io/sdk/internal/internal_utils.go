package internal

// All code in this file is private to the package.

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.temporal.io/sdk/internal/common/metrics"
	"go.temporal.io/sdk/internal/common/retry"
	"google.golang.org/grpc/metadata"
)

const (
	clientNameHeaderName              = "client-name"
	clientNameHeaderValue             = "temporal-go"
	clientVersionHeaderName           = "client-version"
	supportedServerVersionsHeaderName = "supported-server-versions"

	// defaultRPCTimeout is the default gRPC call timeout.
	defaultRPCTimeout = 10 * time.Second
	// minRPCTimeout is minimum gRPC call timeout allowed.
	minRPCTimeout = 1 * time.Second
	// maxRPCTimeout is maximum gRPC call timeout allowed (should not be less than defaultRPCTimeout).
	maxRPCTimeout = 10 * time.Second

	temporalPrefix      = "__temporal_"
	temporalPrefixError = "__temporal_ is a reserved prefix"
)

// grpcContextBuilder stores all gRPC-specific parameters that will
// be stored inside of a context.
type grpcContextBuilder struct {
	Timeout time.Duration

	// ParentContext to build the new context from. If empty, context.Background() is used.
	// The new (child) context inherits a number of properties from the parent context:
	//   - context fields, accessible via `ctx.Value(key)`
	ParentContext context.Context

	MetricsHandler metrics.Handler

	Headers metadata.MD

	IsLongPoll bool
}

func (cb *grpcContextBuilder) Build() (context.Context, context.CancelFunc) {
	ctx := cb.ParentContext
	if ctx == nil {
		ctx = context.Background()
	}
	if cb.Headers != nil {
		ctx = metadata.NewOutgoingContext(ctx, cb.Headers)
	}
	if cb.MetricsHandler != nil {
		ctx = context.WithValue(ctx, metrics.HandlerContextKey{}, cb.MetricsHandler)
	}
	ctx = context.WithValue(ctx, metrics.LongPollContextKey{}, cb.IsLongPoll)
	var cancel context.CancelFunc
	if cb.Timeout != time.Duration(0) {
		ctx, cancel = context.WithTimeout(ctx, cb.Timeout)
	}

	return ctx, cancel
}

func grpcTimeout(timeout time.Duration) func(builder *grpcContextBuilder) {
	return func(b *grpcContextBuilder) {
		b.Timeout = timeout
	}
}

func grpcMetricsHandler(metricsHandler metrics.Handler) func(builder *grpcContextBuilder) {
	return func(b *grpcContextBuilder) {
		b.MetricsHandler = metricsHandler
	}
}

func grpcLongPoll(isLongPoll bool) func(builder *grpcContextBuilder) {
	return func(b *grpcContextBuilder) {
		b.IsLongPoll = isLongPoll
	}
}

func grpcContextValue(key interface{}, val interface{}) func(builder *grpcContextBuilder) {
	return func(b *grpcContextBuilder) {
		b.ParentContext = context.WithValue(b.ParentContext, key, val)
	}
}

func defaultGrpcRetryParameters(ctx context.Context) func(builder *grpcContextBuilder) {
	return grpcContextValue(retry.ConfigKey, createDynamicServiceRetryPolicy(ctx).GrpcRetryConfig())
}

// newGRPCContext - Get context for gRPC calls.
func newGRPCContext(ctx context.Context, options ...func(builder *grpcContextBuilder)) (context.Context, context.CancelFunc) {
	rpcTimeout := defaultRPCTimeout

	// Set rpc timeout less than context timeout to allow for retries when call gets lost
	now := time.Now()
	if deadline, ok := ctx.Deadline(); ok && deadline.After(now) {
		rpcTimeout = deadline.Sub(now) / 2
		// Make sure to not set rpc timeout lower than minRPCTimeout
		if rpcTimeout < minRPCTimeout {
			rpcTimeout = minRPCTimeout
		} else if rpcTimeout > maxRPCTimeout {
			rpcTimeout = maxRPCTimeout
		}
	}

	builder := &grpcContextBuilder{
		ParentContext: ctx,
		Timeout:       rpcTimeout,
		Headers: metadata.New(map[string]string{
			clientNameHeaderName:              clientNameHeaderValue,
			clientVersionHeaderName:           SDKVersion,
			supportedServerVersionsHeaderName: SupportedServerVersions,
		}),
	}

	for _, opt := range options {
		opt(builder)
	}

	return builder.Build()
}

// GetWorkerIdentity gets a default identity for the worker.
func getWorkerIdentity(taskqueueName string) string {
	return fmt.Sprintf("%d@%s@%s", os.Getpid(), getHostName(), taskqueueName)
}

func getHostName() string {
	hostName, err := os.Hostname()
	if err != nil {
		hostName = "Unknown"
	}
	return hostName
}

func getWorkerTaskQueue(stickyUUID string) string {
	// includes hostname for debuggability, stickyUUID guarantees the uniqueness
	return fmt.Sprintf("%s:%s", getHostName(), stickyUUID)
}

// AwaitWaitGroup calls Wait on the given wait
// Returns true if the Wait() call succeeded before the timeout
// Returns false if the Wait() did not return before the timeout
func awaitWaitGroup(wg *sync.WaitGroup, timeout time.Duration) bool {
	doneC := make(chan struct{})

	go func() {
		wg.Wait()
		close(doneC)
	}()

	timer := time.NewTimer(timeout)
	defer func() { timer.Stop() }()

	select {
	case <-doneC:
		return true
	case <-timer.C:
		return false
	}
}

// InterruptCh returns channel which will get data when system receives interrupt signal. Pass it to worker.Run() func to stop worker with Ctrl+C.
func InterruptCh() <-chan interface{} {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	ret := make(chan interface{}, 1)
	go func() {
		s := <-c
		ret <- s
		close(ret)
	}()

	return ret
}

func getStringID(intID int64) string {
	return fmt.Sprintf("%d", intID)
}

type PollerAutoscaleBehavior struct {
	// Minimum is the minimum number of poll calls that will always be attempted (assuming slots are available).
	//
	// Cannot be less than two for workflow tasks, or one for other tasks.
	Minimum int
	// Maximum is the maximum number of poll calls that will ever be open at once. Must be >= `minimum`.
	Maximum int
	// Initial is the number of polls that will be attempted initially before scaling kicks in. Must be between
	// `minimum` and `maximum`.
	Initial int
}
