package nexus

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"strconv"
	"time"
)

// NewCompletionHTTPRequest creates an HTTP request deliver an operation completion to a given URL.
//
// NOTE: Experimental
func NewCompletionHTTPRequest(ctx context.Context, url string, completion OperationCompletion) (*http.Request, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, err
	}
	if err := completion.applyToHTTPRequest(httpReq); err != nil {
		return nil, err
	}

	httpReq.Header.Set(headerUserAgent, userAgent)
	return httpReq, nil
}

// OperationCompletion is input for [NewCompletionHTTPRequest].
// It has two implementations: [OperationCompletionSuccessful] and [OperationCompletionUnsuccessful].
//
// NOTE: Experimental
type OperationCompletion interface {
	applyToHTTPRequest(*http.Request) error
}

// OperationCompletionSuccessful is input for [NewCompletionHTTPRequest], used to deliver successful operation results.
//
// NOTE: Experimental
type OperationCompletionSuccessful struct {
	// Header to send in the completion request.
	// Note that this is a Nexus header, not an HTTP header.
	Header Header

	// A [Reader] that may be directly set on the completion or constructed when instantiating via
	// [NewOperationCompletionSuccessful].
	// Automatically closed when the completion is delivered.
	Reader *Reader
	// OperationID is the unique ID for this operation. Used when a completion callback is received before a started response.
	//
	// Deprecated: Use OperatonToken instead.
	OperationID string
	// OperationToken is the unique token for this operation. Used when a completion callback is received before a
	// started response.
	OperationToken string
	// StartTime is the time the operation started. Used when a completion callback is received before a started response.
	StartTime time.Time
	// Links are used to link back to the operation when a completion callback is received before a started response.
	Links []Link
}

// OperationCompletionSuccessfulOptions are options for [NewOperationCompletionSuccessful].
//
// NOTE: Experimental
type OperationCompletionSuccessfulOptions struct {
	// Optional serializer for the result. Defaults to the SDK's default Serializer, which handles JSONables, byte
	// slices and nils.
	Serializer Serializer
	// OperationID is the unique ID for this operation. Used when a completion callback is received before a started response.
	//
	// Deprecated: Use OperatonToken instead.
	OperationID string
	// OperationToken is the unique token for this operation. Used when a completion callback is received before a
	// started response.
	OperationToken string
	// StartTime is the time the operation started. Used when a completion callback is received before a started response.
	StartTime time.Time
	// Links are used to link back to the operation when a completion callback is received before a started response.
	Links []Link
}

// NewOperationCompletionSuccessful constructs an [OperationCompletionSuccessful] from a given result.
//
// NOTE: Experimental
func NewOperationCompletionSuccessful(result any, options OperationCompletionSuccessfulOptions) (*OperationCompletionSuccessful, error) {
	reader, ok := result.(*Reader)
	if !ok {
		content, ok := result.(*Content)
		if !ok {
			serializer := options.Serializer
			if serializer == nil {
				serializer = defaultSerializer
			}
			var err error
			content, err = serializer.Serialize(result)
			if err != nil {
				return nil, err
			}
		}
		header := maps.Clone(content.Header)
		if header == nil {
			header = make(Header, 1)
		}
		header["length"] = strconv.Itoa(len(content.Data))

		reader = &Reader{
			Header:     header,
			ReadCloser: io.NopCloser(bytes.NewReader(content.Data)),
		}
	}

	return &OperationCompletionSuccessful{
		Header:         make(Header),
		Reader:         reader,
		OperationID:    options.OperationID,
		OperationToken: options.OperationToken,
		StartTime:      options.StartTime,
		Links:          options.Links,
	}, nil
}

func (c *OperationCompletionSuccessful) applyToHTTPRequest(request *http.Request) error {
	if request.Header == nil {
		request.Header = make(http.Header, len(c.Header)+len(c.Reader.Header)+1) // +1 for headerOperationState
	}
	if c.Reader.Header != nil {
		addContentHeaderToHTTPHeader(c.Reader.Header, request.Header)
	}
	if c.Header != nil {
		addNexusHeaderToHTTPHeader(c.Header, request.Header)
	}
	request.Header.Set(headerOperationState, string(OperationStateSucceeded))

	if c.OperationID == "" && c.OperationToken != "" {
		c.OperationID = c.OperationToken
	} else if c.OperationToken == "" && c.OperationID != "" {
		c.OperationToken = c.OperationID
	}
	if c.Header.Get(HeaderOperationID) == "" && c.OperationID != "" {
		request.Header.Set(HeaderOperationID, c.OperationID)
	}
	if c.Header.Get(HeaderOperationToken) == "" && c.OperationToken != "" {
		request.Header.Set(HeaderOperationToken, c.OperationToken)
	}
	if c.Header.Get(headerOperationStartTime) == "" && !c.StartTime.IsZero() {
		request.Header.Set(headerOperationStartTime, c.StartTime.Format(http.TimeFormat))
	}
	if c.Header.Get(headerLink) == "" {
		if err := addLinksToHTTPHeader(c.Links, request.Header); err != nil {
			return err
		}
	}

	request.Body = c.Reader.ReadCloser
	return nil
}

// OperationCompletionUnsuccessful is input for [NewCompletionHTTPRequest], used to deliver unsuccessful operation
// results.
//
// NOTE: Experimental
type OperationCompletionUnsuccessful struct {
	// Header to send in the completion request.
	// Note that this is a Nexus header, not an HTTP header.
	Header Header
	// State of the operation, should be failed or canceled.
	State OperationState
	// OperationID is the unique ID for this operation. Used when a completion callback is received before a started response.
	//
	// Deprecated: Use OperatonToken instead.
	OperationID string
	// OperationToken is the unique token for this operation. Used when a completion callback is received before a
	// started response.
	OperationToken string
	// StartTime is the time the operation started. Used when a completion callback is received before a started response.
	StartTime time.Time
	// Links are used to link back to the operation when a completion callback is received before a started response.
	Links []Link
	// Failure object to send with the completion.
	Failure Failure
}

// OperationCompletionUnsuccessfulOptions are options for [NewOperationCompletionUnsuccessful].
//
// NOTE: Experimental
type OperationCompletionUnsuccessfulOptions struct {
	// A [FailureConverter] to convert a [Failure] instance to and from an [error]. Defaults to
	// [DefaultFailureConverter].
	FailureConverter FailureConverter
	// OperationID is the unique ID for this operation. Used when a completion callback is received before a started response.
	//
	// Deprecated: Use OperatonToken instead.
	OperationID string
	// OperationToken is the unique token for this operation. Used when a completion callback is received before a
	// started response.
	OperationToken string
	// StartTime is the time the operation started. Used when a completion callback is received before a started response.
	StartTime time.Time
	// Links are used to link back to the operation when a completion callback is received before a started response.
	Links []Link
}

// NewOperationCompletionUnsuccessful constructs an [OperationCompletionUnsuccessful] from a given error.
//
// NOTE: Experimental
func NewOperationCompletionUnsuccessful(error *OperationError, options OperationCompletionUnsuccessfulOptions) (*OperationCompletionUnsuccessful, error) {
	if options.FailureConverter == nil {
		options.FailureConverter = defaultFailureConverter
	}

	return &OperationCompletionUnsuccessful{
		Header:         make(Header),
		State:          error.State,
		Failure:        options.FailureConverter.ErrorToFailure(error.Cause),
		OperationID:    options.OperationID,
		OperationToken: options.OperationToken,
		StartTime:      options.StartTime,
		Links:          options.Links,
	}, nil
}

func (c *OperationCompletionUnsuccessful) applyToHTTPRequest(request *http.Request) error {
	if request.Header == nil {
		request.Header = make(http.Header, len(c.Header)+2) // +2 for headerOperationState and content-type
	}
	if c.Header != nil {
		addNexusHeaderToHTTPHeader(c.Header, request.Header)
	}
	request.Header.Set(headerOperationState, string(c.State))
	request.Header.Set("Content-Type", contentTypeJSON)

	if c.OperationID == "" && c.OperationToken != "" {
		c.OperationID = c.OperationToken
	}
	if c.OperationToken == "" && c.OperationID != "" {
		c.OperationToken = c.OperationID
	}
	if c.Header.Get(HeaderOperationID) == "" && c.OperationID != "" {
		request.Header.Set(HeaderOperationID, c.OperationID)
	} else if c.Header.Get(HeaderOperationToken) == "" && c.OperationToken != "" {
		request.Header.Set(HeaderOperationToken, c.OperationToken)
	}
	if c.Header.Get(headerOperationStartTime) == "" && !c.StartTime.IsZero() {
		request.Header.Set(headerOperationStartTime, c.StartTime.Format(http.TimeFormat))
	}
	if c.Header.Get(headerLink) == "" {
		if err := addLinksToHTTPHeader(c.Links, request.Header); err != nil {
			return err
		}
	}

	b, err := json.Marshal(c.Failure)
	if err != nil {
		return err
	}

	request.Body = io.NopCloser(bytes.NewReader(b))
	return nil
}

// CompletionRequest is input for CompletionHandler.CompleteOperation.
//
// NOTE: Experimental
type CompletionRequest struct {
	// The original HTTP request.
	HTTPRequest *http.Request
	// State of the operation.
	State OperationState
	// OperationID is the unique ID for this operation. Used when a completion callback is received before a started response.
	//
	// Deprecated: Use OperatonToken instead.
	OperationID string
	// OperationToken is the unique token for this operation. Used when a completion callback is received before a
	// started response.
	OperationToken string
	// StartTime is the time the operation started. Used when a completion callback is received before a started response.
	StartTime time.Time
	// Links are used to link back to the operation when a completion callback is received before a started response.
	Links []Link
	// Parsed from request and set if State is failed or canceled.
	Error error
	// Extracted from request and set if State is succeeded.
	Result *LazyValue
}

// A CompletionHandler can receive operation completion requests as delivered via the callback URL provided in
// start-operation requests.
//
// NOTE: Experimental
type CompletionHandler interface {
	CompleteOperation(context.Context, *CompletionRequest) error
}

// CompletionHandlerOptions are options for [NewCompletionHTTPHandler].
//
// NOTE: Experimental
type CompletionHandlerOptions struct {
	// Handler for completion requests.
	Handler CompletionHandler
	// A stuctured logging handler.
	// Defaults to slog.Default().
	Logger *slog.Logger
	// A [Serializer] to customize handler serialization behavior.
	// By default the handler handles, JSONables, byte slices, and nil.
	Serializer Serializer
	// A [FailureConverter] to convert a [Failure] instance to and from an [error]. Defaults to
	// [DefaultFailureConverter].
	FailureConverter FailureConverter
}

type completionHTTPHandler struct {
	baseHTTPHandler
	options CompletionHandlerOptions
}

func (h *completionHTTPHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	completion := CompletionRequest{
		State:          OperationState(request.Header.Get(headerOperationState)),
		OperationID:    request.Header.Get(HeaderOperationID),
		OperationToken: request.Header.Get(HeaderOperationToken),
		HTTPRequest:    request,
	}
	if completion.OperationID == "" && completion.OperationToken != "" {
		completion.OperationID = completion.OperationToken
	} else if completion.OperationToken == "" && completion.OperationID != "" {
		completion.OperationToken = completion.OperationID
	}
	if startTimeHeader := request.Header.Get(headerOperationStartTime); startTimeHeader != "" {
		var parseTimeErr error
		if completion.StartTime, parseTimeErr = http.ParseTime(startTimeHeader); parseTimeErr != nil {
			h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "failed to parse operation start time header"))
			return
		}
	}
	var decodeErr error
	if completion.Links, decodeErr = getLinksFromHeader(request.Header); decodeErr != nil {
		h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "failed to decode links from request headers"))
		return
	}
	switch completion.State {
	case OperationStateFailed, OperationStateCanceled:
		if !isMediaTypeJSON(request.Header.Get("Content-Type")) {
			h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid request content type: %q", request.Header.Get("Content-Type")))
			return
		}
		var failure Failure
		b, err := io.ReadAll(request.Body)
		if err != nil {
			h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "failed to read Failure from request body"))
			return
		}
		if err := json.Unmarshal(b, &failure); err != nil {
			h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "failed to read Failure from request body"))
			return
		}
		completion.Error = h.failureConverter.FailureToError(failure)
	case OperationStateSucceeded:
		completion.Result = &LazyValue{
			serializer: h.options.Serializer,
			Reader: &Reader{
				request.Body,
				prefixStrippedHTTPHeaderToNexusHeader(request.Header, "content-"),
			},
		}
	default:
		h.writeFailure(writer, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid request operation state: %q", completion.State))
		return
	}
	if err := h.options.Handler.CompleteOperation(ctx, &completion); err != nil {
		h.writeFailure(writer, err)
	}
}

// NewCompletionHTTPHandler constructs an [http.Handler] from given options for handling operation completion requests.
//
// NOTE: Experimental
func NewCompletionHTTPHandler(options CompletionHandlerOptions) http.Handler {
	if options.Logger == nil {
		options.Logger = slog.Default()
	}
	if options.Serializer == nil {
		options.Serializer = defaultSerializer
	}
	if options.FailureConverter == nil {
		options.FailureConverter = defaultFailureConverter
	}
	return &completionHTTPHandler{
		options: options,
		baseHTTPHandler: baseHTTPHandler{
			logger:           options.Logger,
			failureConverter: options.FailureConverter,
		},
	}
}
