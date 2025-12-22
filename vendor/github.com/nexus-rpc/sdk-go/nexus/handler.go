package nexus

import (
	"context"
	"sync"
)

type handlerCtxKeyType struct{}

var handlerCtxKey = handlerCtxKeyType{}

// HandlerInfo contains the general information for an operation invocation, across different handler methods.
//
// NOTE: Experimental
type HandlerInfo struct {
	// Service is the name of the service that contains the operation.
	Service string
	// Operation is the name of the operation.
	Operation string
	// Header contains the request header fields received by the server.
	Header Header
}

type handlerCtx struct {
	mu    sync.Mutex
	links []Link
	info  HandlerInfo
}

// WithHandlerContext returns a new context from a given context setting it up for being used for handler methods.
// Meant to be used by frameworks, not directly by applications.
//
// NOTE: Experimental
func WithHandlerContext(ctx context.Context, info HandlerInfo) context.Context {
	return context.WithValue(ctx, handlerCtxKey, &handlerCtx{info: info})
}

// IsHandlerContext returns true if the given context is a handler context where [ExtractHandlerInfo], [AddHandlerLinks]
// and [HandlerLinks] can be called. It returns true when called from any [OperationHandler] or [Handler] method or a
// [MiddlewareFunc].
//
// NOTE: Experimental
func IsHandlerContext(ctx context.Context) bool {
	return ctx.Value(handlerCtxKey) != nil
}

// HandlerLinks retrieves the attached links on the given handler context. The returned slice should not be mutated.
// Links are only attached on successful responses to the StartOperation [Handler] and Start [OperationHandler] methods.
// The context provided must be the context passed to any [OperationHandler] or [Handler] method or a [MiddlewareFunc]
// or this method will panic, [IsHandlerContext] can be used to verify the context is valid.
//
// NOTE: Experimental
func HandlerLinks(ctx context.Context) []Link {
	hctx := ctx.Value(handlerCtxKey).(*handlerCtx)
	hctx.mu.Lock()
	cpy := make([]Link, len(hctx.links))
	copy(cpy, hctx.links)
	hctx.mu.Unlock()
	return cpy
}

// AddHandlerLinks associates links with the current operation to be propagated back to the caller. This method may be
// called multiple times for a given handler, each call appending additional links. Links are only attached on
// successful responses to the StartOperation [Handler] and Start [OperationHandler] methods. The context provided must
// be the context passed to any [OperationHandler] or [Handler] method or a [MiddlewareFunc] or this method will panic,
// [IsHandlerContext] can be used to verify the context is valid.
//
// NOTE: Experimental
func AddHandlerLinks(ctx context.Context, links ...Link) {
	hctx := ctx.Value(handlerCtxKey).(*handlerCtx)
	hctx.mu.Lock()
	hctx.links = append(hctx.links, links...)
	hctx.mu.Unlock()
}

// SetHandlerLinks associates links with the current operation to be propagated back to the caller. This method replaces
// any previously associated links, it is recommended to use [AddHandlerLinks] to avoid accidental override. Links are
// only attached on successful responses to the StartOperation [Handler] and Start [OperationHandler] methods. The
// context provided must be the context passed to any [OperationHandler] or [Handler] method or a [MiddlewareFunc] or
// this method will panic, [IsHandlerContext] can be used to verify the context is valid.
//
// NOTE: Experimental
func SetHandlerLinks(ctx context.Context, links ...Link) {
	hctx := ctx.Value(handlerCtxKey).(*handlerCtx)
	hctx.mu.Lock()
	hctx.links = links
	hctx.mu.Unlock()
}

// ExtractHandlerInfo extracts the [HandlerInfo] from a given context. The context provided must be the context passed
// to any [OperationHandler] or [Handler] method or a [MiddlewareFunc] or this method will panic, [IsHandlerContext] can
// be used to verify the context is valid.
//
// NOTE: Experimental
func ExtractHandlerInfo(ctx context.Context) HandlerInfo {
	hctx := ctx.Value(handlerCtxKey).(*handlerCtx)
	return hctx.info
}

// An HandlerStartOperationResult is the return type from the [Handler] StartOperation and [Operation] Start methods. It
// has two implementations: [HandlerStartOperationResultSync] and [HandlerStartOperationResultAsync].
type HandlerStartOperationResult[T any] interface {
	mustImplementHandlerStartOperationResult()
}

// HandlerStartOperationResultSync indicates that an operation completed successfully.
type HandlerStartOperationResultSync[T any] struct {
	// Value is the output of the operation.
	Value T
	// Links to be associated with the operation.
	//
	// Deprecated: Use AddHandlerLinks instead.
	Links []Link
}

func (*HandlerStartOperationResultSync[T]) mustImplementHandlerStartOperationResult() {}

// ValueAsAny returns the generic value out of the result.
func (r *HandlerStartOperationResultSync[T]) ValueAsAny() any {
	return r.Value
}

// HandlerStartOperationResultAsync indicates that an operation has been accepted and will complete asynchronously.
type HandlerStartOperationResultAsync struct {
	// OperationID is a unique ID to identify the operation.
	//
	// Deprecated: Use OperationToken instead.
	OperationID string
	// OperationToken is a unique token to identify the operation.
	OperationToken string
	// Links to be associated with the operation.
	//
	// Deprecated: Use AddHandlerLinks instead.
	Links []Link
}

func (*HandlerStartOperationResultAsync) mustImplementHandlerStartOperationResult() {}

// A Handler must implement all of the Nexus service endpoints.
//
// Handler implementations must embed the [UnimplementedHandler].
//
// All Handler methods can return a [HandlerError] to fail requests with a custom [HandlerErrorType] and structured [Failure].
// Arbitrary errors from handler methods are turned into [HandlerErrorTypeInternal], their details are logged and hidden
// from the caller.
type Handler interface {
	// StartOperation handles requests for starting an operation. Return [HandlerStartOperationResultSync] to
	// respond successfully - inline, or [HandlerStartOperationResultAsync] to indicate that an asynchronous
	// operation was started. Return an [OperationError] to indicate that an operation completed as
	// failed or canceled.
	StartOperation(ctx context.Context, service, operation string, input *LazyValue, options StartOperationOptions) (HandlerStartOperationResult[any], error)
	// CancelOperation handles requests to cancel an asynchronous operation.
	// Cancelation in Nexus is:
	//  1. asynchronous - returning from this method only ensures that cancelation is delivered, it may later be
	//  ignored by the underlying operation implemention.
	//  2. idempotent - implementors should ignore duplicate cancelations for the same operation.
	CancelOperation(ctx context.Context, service, operation, token string, options CancelOperationOptions) error
	// GetOperationResult handles requests to get the result of an asynchronous operation. Return non error result
	// to respond successfully - inline, or error with [ErrOperationStillRunning] to indicate that an asynchronous
	// operation is still running. Return an [OperationError] to indicate that an operation completed as
	// failed or canceled.
	//
	// When [GetOperationResultOptions.Wait] is greater than zero, this request should be treated as a long poll.
	// Long poll requests have a server side timeout, configurable via [HandlerOptions.GetResultTimeout], and exposed
	// via context deadline. The context deadline is decoupled from the application level Wait duration.
	//
	// Deprecated: Getting a result directly from a handler is no longer supported.
	GetOperationResult(ctx context.Context, service, operation, token string, options GetOperationResultOptions) (any, error)
	// GetOperationInfo handles requests to get information about an asynchronous operation.
	//
	// Deprecated: Getting info directly from a handler is no longer supported.
	GetOperationInfo(ctx context.Context, service, operation, token string, options GetOperationInfoOptions) (*OperationInfo, error)
	mustEmbedUnimplementedHandler()
}
