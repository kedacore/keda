package nexus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
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
	applyToHTTPResponse(http.ResponseWriter, *httpHandler)
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

func (r *HandlerStartOperationResultSync[T]) applyToHTTPResponse(writer http.ResponseWriter, handler *httpHandler) {
	if err := addLinksToHTTPHeader(r.Links, writer.Header()); err != nil {
		handler.logger.Error("failed to serialize links into header", "error", err)
		// clear any previous links already written to the header
		writer.Header().Del(headerLink)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	handler.writeResult(writer, r.Value)
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

func (r *HandlerStartOperationResultAsync) applyToHTTPResponse(writer http.ResponseWriter, handler *httpHandler) {
	if r.OperationToken == "" && r.OperationID != "" {
		r.OperationToken = r.OperationID
	} else if r.OperationToken != "" && r.OperationID == "" {
		r.OperationID = r.OperationToken
	}
	info := OperationInfo{
		ID:    r.OperationID,
		Token: r.OperationToken,
		State: OperationStateRunning,
	}
	bytes, err := json.Marshal(info)
	if err != nil {
		handler.logger.Error("failed to serialize operation info", "error", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := addLinksToHTTPHeader(r.Links, writer.Header()); err != nil {
		handler.logger.Error("failed to serialize links into header", "error", err)
		// clear any previous links already written to the header
		writer.Header().Del(headerLink)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", contentTypeJSON)
	writer.WriteHeader(http.StatusCreated)

	if _, err := writer.Write(bytes); err != nil {
		handler.logger.Error("failed to write response body", "error", err)
	}
}

// A Handler must implement all of the Nexus service endpoints as defined in the [Nexus HTTP API].
//
// Handler implementations must embed the [UnimplementedHandler].
//
// All Handler methods can return a [HandlerError] to fail requests with a custom [HandlerErrorType] and structured [Failure].
// Arbitrary errors from handler methods are turned into [HandlerErrorTypeInternal],their details are logged and hidden
// from the caller.
//
// [Nexus HTTP API]: https://github.com/nexus-rpc/api
type Handler interface {
	// StartOperation handles requests for starting an operation. Return [HandlerStartOperationResultSync] to
	// respond successfully - inline, or [HandlerStartOperationResultAsync] to indicate that an asynchronous
	// operation was started. Return an [OperationError] to indicate that an operation completed as
	// failed or canceled.
	StartOperation(ctx context.Context, service, operation string, input *LazyValue, options StartOperationOptions) (HandlerStartOperationResult[any], error)
	// GetOperationResult handles requests to get the result of an asynchronous operation. Return non error result
	// to respond successfully - inline, or error with [ErrOperationStillRunning] to indicate that an asynchronous
	// operation is still running. Return an [OperationError] to indicate that an operation completed as
	// failed or canceled.
	//
	// When [GetOperationResultOptions.Wait] is greater than zero, this request should be treated as a long poll.
	// Long poll requests have a server side timeout, configurable via [HandlerOptions.GetResultTimeout], and exposed
	// via context deadline. The context deadline is decoupled from the application level Wait duration.
	//
	// It is the implementor's responsiblity to respect the client's wait duration and return in a timely fashion.
	// Consider using a derived context that enforces the wait timeout when implementing this method and return
	// [ErrOperationStillRunning] when that context expires as shown in the example.
	//
	// NOTE: Experimental
	GetOperationResult(ctx context.Context, service, operation, token string, options GetOperationResultOptions) (any, error)
	// GetOperationInfo handles requests to get information about an asynchronous operation.
	//
	// NOTE: Experimental
	GetOperationInfo(ctx context.Context, service, operation, token string, options GetOperationInfoOptions) (*OperationInfo, error)
	// CancelOperation handles requests to cancel an asynchronous operation.
	// Cancelation in Nexus is:
	//  1. asynchronous - returning from this method only ensures that cancelation is delivered, it may later be
	//  ignored by the underlying operation implemention.
	//  2. idempotent - implementors should ignore duplicate cancelations for the same operation.
	CancelOperation(ctx context.Context, service, operation, token string, options CancelOperationOptions) error
	mustEmbedUnimplementedHandler()
}

type baseHTTPHandler struct {
	logger           *slog.Logger
	failureConverter FailureConverter
}

type httpHandler struct {
	baseHTTPHandler
	options HandlerOptions
}

func (h *httpHandler) writeResult(writer http.ResponseWriter, result any) {
	var reader *Reader
	if r, ok := result.(*Reader); ok {
		// Close the request body in case we error before sending the HTTP request (which may double close but
		// that's fine since we ignore the error).
		defer r.Close()
		reader = r
	} else {
		content, ok := result.(*Content)
		if !ok {
			var err error
			content, err = h.options.Serializer.Serialize(result)
			if err != nil {
				h.writeFailure(writer, fmt.Errorf("failed to serialize handler result: %w", err))
				return
			}
		}
		header := maps.Clone(content.Header)
		header["length"] = strconv.Itoa(len(content.Data))

		reader = &Reader{
			io.NopCloser(bytes.NewReader(content.Data)),
			header,
		}
	}

	header := writer.Header()
	addContentHeaderToHTTPHeader(reader.Header, header)
	if reader.ReadCloser == nil {
		return
	}
	if _, err := io.Copy(writer, reader); err != nil {
		h.logger.Error("failed to write response body", "error", err)
	}
}

func (h *baseHTTPHandler) writeFailure(writer http.ResponseWriter, err error) {
	var failure Failure
	var unsuccessfulError *OperationError
	var handlerError *HandlerError
	var operationState OperationState
	statusCode := http.StatusInternalServerError

	if errors.As(err, &unsuccessfulError) {
		operationState = unsuccessfulError.State
		failure = h.failureConverter.ErrorToFailure(unsuccessfulError.Cause)
		statusCode = statusOperationFailed

		if operationState == OperationStateFailed || operationState == OperationStateCanceled {
			writer.Header().Set(headerOperationState, string(operationState))
		} else {
			h.logger.Error("unexpected operation state", "state", operationState)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else if errors.As(err, &handlerError) {
		failure = h.failureConverter.ErrorToFailure(handlerError.Cause)
		switch handlerError.Type {
		case HandlerErrorTypeBadRequest:
			statusCode = http.StatusBadRequest
		case HandlerErrorTypeUnauthenticated:
			statusCode = http.StatusUnauthorized
		case HandlerErrorTypeUnauthorized:
			statusCode = http.StatusForbidden
		case HandlerErrorTypeNotFound:
			statusCode = http.StatusNotFound
		case HandlerErrorTypeResourceExhausted:
			statusCode = http.StatusTooManyRequests
		case HandlerErrorTypeInternal:
			statusCode = http.StatusInternalServerError
		case HandlerErrorTypeNotImplemented:
			statusCode = http.StatusNotImplemented
		case HandlerErrorTypeUnavailable:
			statusCode = http.StatusServiceUnavailable
		case HandlerErrorTypeUpstreamTimeout:
			statusCode = StatusUpstreamTimeout
		default:
			h.logger.Error("unexpected handler error type", "type", handlerError.Type)
		}
	} else {
		failure = Failure{
			Message: "internal server error",
		}
		h.logger.Error("handler failed", "error", err)
	}

	bytes, err := json.Marshal(failure)
	if err != nil {
		h.logger.Error("failed to marshal failure", "error", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", contentTypeJSON)

	// Set the retry header here after ensuring that we don't fail with internal error due to failed marshaling to
	// preserve the user's intent.
	if handlerError != nil {
		switch handlerError.RetryBehavior {
		case HandlerErrorRetryBehaviorNonRetryable:
			writer.Header().Set(headerRetryable, "false")
		case HandlerErrorRetryBehaviorRetryable:
			writer.Header().Set(headerRetryable, "true")
		}
	}

	writer.WriteHeader(statusCode)

	if _, err := writer.Write(bytes); err != nil {
		h.logger.Error("failed to write response body", "error", err)
	}
}

func (h *httpHandler) startOperation(service, operation string, writer http.ResponseWriter, request *http.Request) {
	links, err := getLinksFromHeader(request.Header)
	if err != nil {
		h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid %q header", headerLink))
		return
	}
	options := StartOperationOptions{
		RequestID:      request.Header.Get(headerRequestID),
		CallbackURL:    request.URL.Query().Get(queryCallbackURL),
		CallbackHeader: prefixStrippedHTTPHeaderToNexusHeader(request.Header, "nexus-callback-"),
		Header:         httpHeaderToNexusHeader(request.Header, "content-", "nexus-callback-"),
		Links:          links,
	}
	value := &LazyValue{
		serializer: h.options.Serializer,
		Reader: &Reader{
			request.Body,
			prefixStrippedHTTPHeaderToNexusHeader(request.Header, "content-"),
		},
	}

	ctx, cancel, ok := h.contextWithTimeoutFromHTTPRequest(writer, request)
	if !ok {
		return
	}
	defer cancel()

	ctx = WithHandlerContext(ctx, HandlerInfo{
		Service:   service,
		Operation: operation,
		Header:    options.Header,
	})
	response, err := h.options.Handler.StartOperation(ctx, service, operation, value, options)
	if err != nil {
		h.writeFailure(writer, err)
	} else {
		if err := addLinksToHTTPHeader(HandlerLinks(ctx), writer.Header()); err != nil {
			h.logger.Error("failed to serialize links into header", "error", err)
			// clear any previous links already written to the header
			writer.Header().Del(headerLink)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		response.applyToHTTPResponse(writer, h)
	}
}

func (h *httpHandler) getOperationResult(service, operation, token string, writer http.ResponseWriter, request *http.Request) {
	options := GetOperationResultOptions{Header: httpHeaderToNexusHeader(request.Header)}
	ctx := request.Context()
	// If both Request-Timeout http header and wait query string are set, the minimum of the Request-Timeout header
	// and h.options.GetResultTimeout will be used.
	requestTimeout, ok := h.parseRequestTimeoutHeader(writer, request)
	if !ok {
		return
	}
	waitStr := request.URL.Query().Get(queryWait)
	if waitStr != "" {
		waitDuration, err := parseDuration(waitStr)
		if err != nil {
			h.logger.Warn("invalid wait duration query parameter", "wait", waitStr)
			h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid wait query parameter"))
			return
		}
		options.Wait = waitDuration
		if requestTimeout > 0 {
			requestTimeout = min(requestTimeout, h.options.GetResultTimeout)
		} else {
			requestTimeout = h.options.GetResultTimeout
		}
	}
	if requestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(request.Context(), requestTimeout)
		defer cancel()
	}

	ctx = WithHandlerContext(ctx, HandlerInfo{
		Service:   service,
		Operation: operation,
		Header:    options.Header,
	})
	result, err := h.options.Handler.GetOperationResult(ctx, service, operation, token, options)
	if err != nil {
		if options.Wait > 0 && ctx.Err() != nil {
			writer.WriteHeader(http.StatusRequestTimeout)
		} else if errors.Is(err, ErrOperationStillRunning) {
			writer.WriteHeader(statusOperationRunning)
		} else {
			h.writeFailure(writer, err)
		}
		return
	}
	h.writeResult(writer, result)
}

func (h *httpHandler) getOperationInfo(service, operation, token string, writer http.ResponseWriter, request *http.Request) {
	options := GetOperationInfoOptions{Header: httpHeaderToNexusHeader(request.Header)}

	ctx, cancel, ok := h.contextWithTimeoutFromHTTPRequest(writer, request)
	if !ok {
		return
	}
	defer cancel()

	ctx = WithHandlerContext(ctx, HandlerInfo{
		Service:   service,
		Operation: operation,
		Header:    options.Header,
	})
	info, err := h.options.Handler.GetOperationInfo(ctx, service, operation, token, options)
	if err != nil {
		h.writeFailure(writer, err)
		return
	}
	if info.ID == "" && info.Token != "" {
		info.ID = info.Token
	} else if info.ID != "" && info.Token == "" {
		info.Token = info.ID
	}

	bytes, err := json.Marshal(info)
	if err != nil {
		h.writeFailure(writer, fmt.Errorf("failed to marshal operation info: %w", err))
		return
	}
	writer.Header().Set("Content-Type", contentTypeJSON)
	if _, err := writer.Write(bytes); err != nil {
		h.logger.Error("failed to write response body", "error", err)
	}
}

func (h *httpHandler) cancelOperation(service, operation, token string, writer http.ResponseWriter, request *http.Request) {
	options := CancelOperationOptions{Header: httpHeaderToNexusHeader(request.Header)}

	ctx, cancel, ok := h.contextWithTimeoutFromHTTPRequest(writer, request)
	if !ok {
		return
	}
	defer cancel()

	ctx = WithHandlerContext(ctx, HandlerInfo{
		Service:   service,
		Operation: operation,
		Header:    options.Header,
	})
	if err := h.options.Handler.CancelOperation(ctx, service, operation, token, options); err != nil {
		h.writeFailure(writer, err)
		return
	}

	writer.WriteHeader(http.StatusAccepted)
}

// parseRequestTimeoutHeader checks if the Request-Timeout HTTP header is set and returns the parsed duration if so.
// Returns (0, true) if unset. Returns ({parsedDuration}, true) if set. If set and there is an error parsing the
// duration, it writes a failure response and returns (0, false).
func (h *httpHandler) parseRequestTimeoutHeader(writer http.ResponseWriter, request *http.Request) (time.Duration, bool) {
	timeoutStr := request.Header.Get(HeaderRequestTimeout)
	if timeoutStr != "" {
		timeoutDuration, err := parseDuration(timeoutStr)
		if err != nil {
			h.logger.Warn("invalid request timeout header", "timeout", timeoutStr)
			h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid request timeout header"))
			return 0, false
		}
		return timeoutDuration, true
	}
	return 0, true
}

// contextWithTimeoutFromHTTPRequest extracts the context from the HTTP request and applies the timeout indicated by
// the Request-Timeout header, if set.
func (h *httpHandler) contextWithTimeoutFromHTTPRequest(writer http.ResponseWriter, request *http.Request) (context.Context, context.CancelFunc, bool) {
	requestTimeout, ok := h.parseRequestTimeoutHeader(writer, request)
	if !ok {
		return nil, nil, false
	}
	if requestTimeout > 0 {
		ctx, cancel := context.WithTimeout(request.Context(), requestTimeout)
		return ctx, cancel, true
	}
	return request.Context(), func() {}, true
}

// HandlerOptions are options for [NewHTTPHandler].
type HandlerOptions struct {
	// Handler for handling service requests.
	Handler Handler
	// A stuctured logger.
	// Defaults to slog.Default().
	Logger *slog.Logger
	// Max duration to allow waiting for a single get result request.
	// Enforced if provided for requests with the wait query parameter set.
	//
	// Defaults to one minute.
	GetResultTimeout time.Duration
	// A [Serializer] to customize handler serialization behavior.
	// By default the handler handles JSONables, byte slices, and nil.
	Serializer Serializer
	// A [FailureConverter] to convert a [Failure] instance to and from an [error].
	// Defaults to [DefaultFailureConverter].
	FailureConverter FailureConverter
}

func (h *httpHandler) handleRequest(writer http.ResponseWriter, request *http.Request) {
	parts := strings.Split(request.URL.EscapedPath(), "/")
	// First part is empty (due to leading /)
	if len(parts) < 3 {
		h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeNotFound, "not found"))
		return
	}
	service, err := url.PathUnescape(parts[1])
	if err != nil {
		h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "failed to parse URL path"))
		return
	}
	operation, err := url.PathUnescape(parts[2])
	if err != nil {
		h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "failed to parse URL path"))
		return
	}

	// First handle StartOperation at /{service}/{operation}
	if len(parts) == 3 && request.Method == "POST" {
		h.startOperation(service, operation, writer, request)
		return
	}

	token := request.Header.Get(HeaderOperationToken)
	if token == "" {
		token = request.URL.Query().Get("token")
	} else {
		// Sanitize this header as it is explicitly passed in as an argument.
		request.Header.Del(HeaderOperationToken)
	}

	if token != "" {
		switch len(parts) {
		case 3: // /{service}/{operation}
			if request.Method != "GET" {
				h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid request method: expected GET, got %q", request.Method))
				return
			}
			h.getOperationInfo(service, operation, token, writer, request)
		case 4:
			switch parts[3] {
			case "result": // /{service}/{operation}/result
				if request.Method != "GET" {
					h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid request method: expected GET, got %q", request.Method))
					return
				}
				h.getOperationResult(service, operation, token, writer, request)
			case "cancel": // /{service}/{operation}/cancel
				if request.Method != "POST" {
					h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid request method: expected POST, got %q", request.Method))
					return
				}
				h.cancelOperation(service, operation, token, writer, request)
			default:
				h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeNotFound, "not found"))
			}
		default:
			h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeNotFound, "not found"))
		}
	} else {
		token, err = url.PathUnescape(parts[3])
		if err != nil {
			h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "failed to parse URL path"))
			return
		}

		switch len(parts) {
		case 4: // /{service}/{operation}/{operation_id}
			if request.Method != "GET" {
				h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid request method: expected GET, got %q", request.Method))
				return
			}
			h.getOperationInfo(service, operation, token, writer, request)
		case 5:
			switch parts[4] {
			case "result": // /{service}/{operation}/{operation_id}/result
				if request.Method != "GET" {
					h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid request method: expected GET, got %q", request.Method))
					return
				}
				h.getOperationResult(service, operation, token, writer, request)
			case "cancel": // /{service}/{operation}/{operation_id}/cancel
				if request.Method != "POST" {
					h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid request method: expected POST, got %q", request.Method))
					return
				}
				h.cancelOperation(service, operation, token, writer, request)
			default:
				h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeNotFound, "not found"))
			}
		default:
			h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeNotFound, "not found"))
		}
	}
}

// NewHTTPHandler constructs an [http.Handler] from given options for handling Nexus service requests.
func NewHTTPHandler(options HandlerOptions) http.Handler {
	if options.Logger == nil {
		options.Logger = slog.Default()
	}
	if options.GetResultTimeout == 0 {
		options.GetResultTimeout = time.Minute
	}
	if options.Serializer == nil {
		options.Serializer = defaultSerializer
	}
	if options.FailureConverter == nil {
		options.FailureConverter = defaultFailureConverter
	}
	handler := &httpHandler{
		baseHTTPHandler: baseHTTPHandler{
			logger:           options.Logger,
			failureConverter: options.FailureConverter,
		},
		options: options,
	}

	return http.HandlerFunc(handler.handleRequest)
}
