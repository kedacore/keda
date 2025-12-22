// Package nexus provides client and server implementations of the Nexus [HTTP API]
//
// [HTTP API]: https://github.com/nexus-rpc/api
package nexus

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/url"
	"strings"
)

const (
	// HeaderOperationID is the unique ID returned by the StartOperation response for async operations.
	// Must be set on callback headers to support completing operations before the start response is received.
	//
	// Deprecated: Use HeaderOperationToken instead.
	HeaderOperationID = "nexus-operation-id"

	// HeaderOperationToken is the unique token returned by the StartOperation response for async operations.
	// Must be set on callback headers to support completing operations before the start response is received.
	HeaderOperationToken = "nexus-operation-token"

	// HeaderRequestTimeout is the total time to complete a Nexus HTTP request.
	HeaderRequestTimeout = "request-timeout"
	// HeaderOperationTimeout is the total time to complete a Nexus operation.
	// Unlike HeaderRequestTimeout, this applies to the whole operation, not just a single HTTP request.
	HeaderOperationTimeout = "operation-timeout"
)

const StatusUpstreamTimeout = 520

// A Failure represents failed handler invocations as well as `failed` or `canceled` operation results. Failures
// shouldn't typically be constructed directly. The SDK APIs take a [FailureConverter] instance that can translate
// language errors to and from [Failure] instances.
type Failure struct {
	// A simple text message.
	Message string `json:"message"`
	// A key-value mapping for additional context. Useful for decoding the 'details' field, if needed.
	Metadata map[string]string `json:"metadata,omitempty"`
	// Additional JSON serializable structured data.
	Details json.RawMessage `json:"details,omitempty"`
}

// An error that directly represents a wire representation of [Failure].
// The SDK will convert to this error by default unless the [FailureConverter] instance is customized.
type FailureError struct {
	// The underlying Failure object this error represents.
	Failure Failure
}

// Error implements the error interface.
func (e *FailureError) Error() string {
	return e.Failure.Message
}

// OperationError represents "failed" and "canceled" operation results.
type OperationError struct {
	// State of the operation. Only [OperationStateFailed] and [OperationStateCanceled] are valid.
	State OperationState
	// The underlying cause for this error.
	Cause error
}

// UnsuccessfulOperationError represents "failed" and "canceled" operation results.
//
// Deprecated: Use [OperationError] instead.
type UnsuccessfulOperationError = OperationError

// NewFailedOperationError is shorthand for constructing an [OperationError] with state set to
// [OperationStateFailed] and the given err as the cause.
//
// Deprecated: Use [NewOperationFailedError] or construct an [OperationError] directly instead.
func NewFailedOperationError(err error) *OperationError {
	return &OperationError{
		State: OperationStateFailed,
		Cause: err,
	}
}

// NewOperationFailedError is shorthand for constructing an [OperationError] with state set to
// [OperationStateFailed] and the given error message as the cause.
func NewOperationFailedError(message string) *OperationError {
	return &OperationError{
		State: OperationStateFailed,
		Cause: errors.New(message),
	}
}

// OperationFailedErrorf creates an [OperationError] with state set to [OperationStateFailed], using [fmt.Errorf] to
// construct the cause.
func OperationFailedErrorf(format string, args ...any) *OperationError {
	return &OperationError{
		State: OperationStateFailed,
		Cause: fmt.Errorf(format, args...),
	}
}

// NewCanceledOperationError is shorthand for constructing an [OperationError] with state set to
// [OperationStateCanceled] and the given err as the cause.
//
// Deprecated: Use [NewOperationCanceledError] or construct an [OperationError] directly instead.
func NewCanceledOperationError(err error) *OperationError {
	return &OperationError{
		State: OperationStateCanceled,
		Cause: err,
	}
}

// NewOperationCanceledError is shorthand for constructing an [OperationError] with state set to
// [OperationStateCanceled] and the given error message as the cause.
func NewOperationCanceledError(message string) *OperationError {
	return &OperationError{
		State: OperationStateCanceled,
		Cause: errors.New(message),
	}
}

// OperationCanceledErrorf creates an [OperationError] with state set to [OperationStateCanceled], using [fmt.Errorf] to
// construct the cause.
func OperationCanceledErrorf(format string, args ...any) *OperationError {
	return &OperationError{
		State: OperationStateCanceled,
		Cause: fmt.Errorf(format, args...),
	}
}

// OperationErrorf creates an [OperationError] with the given state, using [fmt.Errorf] to construct the cause.
func OperationErrorf(state OperationState, format string, args ...any) *OperationError {
	return &OperationError{
		State: state,
		Cause: fmt.Errorf(format, args...),
	}
}

// Error implements the error interface.
func (e *OperationError) Error() string {
	if e.Cause == nil {
		return fmt.Sprintf("operation %s", e.State)
	}
	return fmt.Sprintf("operation %s: %s", e.State, e.Cause.Error())
}

// Unwrap returns the cause for use with utilities in the errors package.
func (e *OperationError) Unwrap() error {
	return e.Cause
}

// HandlerErrorType is an error type associated with a [HandlerError], defined according to the Nexus specification.
// Only the types defined as consts in this package are valid. Do not use other values.
type HandlerErrorType string

const (
	// The server cannot or will not process the request due to an apparent client error. Clients should not retry
	// this request unless advised otherwise.
	HandlerErrorTypeBadRequest HandlerErrorType = "BAD_REQUEST"
	// The client did not supply valid authentication credentials for this request. Clients should not retry
	// this request unless advised otherwise.
	HandlerErrorTypeUnauthenticated HandlerErrorType = "UNAUTHENTICATED"
	// The caller does not have permission to execute the specified operation. Clients should not retry this
	// request unless advised otherwise.
	HandlerErrorTypeUnauthorized HandlerErrorType = "UNAUTHORIZED"
	// The requested resource could not be found but may be available in the future. Clients should not retry
	// this request unless advised otherwise.
	HandlerErrorTypeNotFound HandlerErrorType = "NOT_FOUND"
	// Returned by the server to when it has given up handling a request. The may occur by enforcing a client
	// provided `Request-Timeout` or for any arbitrary reason such as enforcing some configurable limit. Subsequent
	// requests by the client are permissible.
	HandlerErrorTypeRequestTimeout HandlerErrorType = "REQUEST_TIMEOUT"
	// The request could not be made due to a conflict. The may happen when trying to create an operation that
	// has already been started. Clients should not retry this request unless advised otherwise.
	HandlerErrorTypeConflict HandlerErrorType = "CONFLICT"
	// Some resource has been exhausted, perhaps a per-user quota, or perhaps the entire file system is out of
	// space. Subsequent requests by the client are permissible.
	HandlerErrorTypeResourceExhausted HandlerErrorType = "RESOURCE_EXHAUSTED"
	// An internal error occured. Subsequent requests by the client are permissible.
	HandlerErrorTypeInternal HandlerErrorType = "INTERNAL"
	// The server either does not recognize the request method, or it lacks the ability to fulfill the request.
	// Clients should not retry this request unless advised otherwise.
	HandlerErrorTypeNotImplemented HandlerErrorType = "NOT_IMPLEMENTED"
	// The service is currently unavailable. Subsequent requests by the client are permissible.
	HandlerErrorTypeUnavailable HandlerErrorType = "UNAVAILABLE"
	// Used by gateways to report that a request to an upstream server has timed out. Subsequent requests by the
	// client are permissible.
	HandlerErrorTypeUpstreamTimeout HandlerErrorType = "UPSTREAM_TIMEOUT"
)

// HandlerErrorRetryBehavior allows handlers to explicity set the retry behavior of a [HandlerError]. If not specified,
// retry behavior is determined from the error type. For example [HandlerErrorTypeInternal] is not retryable by default
// unless specified otherwise.
type HandlerErrorRetryBehavior int

const (
	// HandlerErrorRetryBehaviorUnspecified indicates that the retry behavior for a [HandlerError] is determined
	// from the [HandlerErrorType].
	HandlerErrorRetryBehaviorUnspecified HandlerErrorRetryBehavior = iota
	// HandlerErrorRetryBehaviorRetryable explicitly indicates that a [HandlerError] should be retried, overriding
	// the default retry behavior of the [HandlerErrorType].
	HandlerErrorRetryBehaviorRetryable
	// HandlerErrorRetryBehaviorNonRetryable explicitly indicates that a [HandlerError] should not be retried,
	// overriding the default retry behavior of the [HandlerErrorType].
	HandlerErrorRetryBehaviorNonRetryable
)

// HandlerError is a special error that can be returned from [Handler] methods for failing a request with a custom
// status code and failure message.
type HandlerError struct {
	// Error Type. Defaults to HandlerErrorTypeInternal.
	Type HandlerErrorType
	// The underlying cause for this error.
	Cause error
	// RetryBehavior of this error. If not specified, retry behavior is determined from the error type.
	RetryBehavior HandlerErrorRetryBehavior
}

// HandlerErrorf creates a [HandlerError] with the given type, using [fmt.Errorf] to construct the cause.
func HandlerErrorf(typ HandlerErrorType, format string, args ...any) *HandlerError {
	return &HandlerError{
		Type:  typ,
		Cause: fmt.Errorf(format, args...),
	}
}

// Retryable returns a boolean indicating whether or not this error is retryable based on the error's RetryBehavior and
// Type.
func (e *HandlerError) Retryable() bool {
	switch e.RetryBehavior {
	case HandlerErrorRetryBehaviorNonRetryable:
		return false
	case HandlerErrorRetryBehaviorRetryable:
		return true
	}
	switch e.Type {
	case HandlerErrorTypeBadRequest,
		HandlerErrorTypeUnauthenticated,
		HandlerErrorTypeUnauthorized,
		HandlerErrorTypeNotFound,
		HandlerErrorTypeNotImplemented,
		HandlerErrorTypeConflict:
		return false
	case HandlerErrorTypeResourceExhausted,
		HandlerErrorTypeInternal,
		HandlerErrorTypeUnavailable,
		HandlerErrorTypeUpstreamTimeout,
		HandlerErrorTypeRequestTimeout:
		return true
	default:
		return true
	}
}

// Error implements the error interface.
func (e *HandlerError) Error() string {
	typ := e.Type
	if len(typ) == 0 {
		typ = HandlerErrorTypeInternal
	}
	if e.Cause == nil {
		return fmt.Sprintf("handler error (%s)", typ)
	}
	return fmt.Sprintf("handler error (%s): %s", typ, e.Cause.Error())
}

// Unwrap returns the cause for use with utilities in the errors package.
func (e *HandlerError) Unwrap() error {
	return e.Cause
}

// OperationInfo conveys information about an operation.
type OperationInfo struct {
	// ID of the operation.
	//
	// Deprecated: Use Token instead.
	ID string `json:"id"`
	// Token for the operation.
	Token string `json:"token"`
	// State of the operation.
	State OperationState `json:"state"`
}

// OperationState represents the variable states of an operation.
type OperationState string

const (
	// "running" operation state. Indicates an operation is started and not yet completed.
	OperationStateRunning OperationState = "running"
	// "succeeded" operation state. Indicates an operation completed successfully.
	OperationStateSucceeded OperationState = "succeeded"
	// "failed" operation state. Indicates an operation completed as failed.
	OperationStateFailed OperationState = "failed"
	// "canceled" operation state. Indicates an operation completed as canceled.
	OperationStateCanceled OperationState = "canceled"
)

// isMediaTypeJSON returns true if the given content type's media type is application/json.
func isMediaTypeJSON(contentType string) bool {
	if contentType == "" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == "application/json"
}

// isMediaTypeOctetStream returns true if the given content type's media type is application/octet-stream.
func isMediaTypeOctetStream(contentType string) bool {
	if contentType == "" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == "application/octet-stream"
}

// Header is a mapping of string to string.
// It is used throughout the framework to transmit metadata.
// The keys should be in lower case form.
type Header map[string]string

// Get is a case-insensitive key lookup from the header map.
func (h Header) Get(k string) string {
	return h[strings.ToLower(k)]
}

// Set sets the header key to the given value transforming the key to its lower case form.
func (h Header) Set(k, v string) {
	h[strings.ToLower(k)] = v
}

// Link contains an URL and a Type that can be used to decode the URL.
// Links can contain any arbitrary information as a percent-encoded URL.
// It can be used to pass information about the caller to the handler, or vice-versa.
type Link struct {
	// URL information about the link.
	// It must be URL percent-encoded.
	URL *url.URL
	// Type can describe an actual data type for decoding the URL.
	// Valid chars: alphanumeric, '_', '.', '/'
	Type string
}
