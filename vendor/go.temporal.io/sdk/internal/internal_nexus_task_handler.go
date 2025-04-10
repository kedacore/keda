// The MIT License
//
// Copyright (c) 2024 Temporal Technologies Inc.  All rights reserved.
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

package internal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"runtime/debug"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	"go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	nexuspb "go.temporal.io/api/nexus/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/internal/common/metrics"
	"go.temporal.io/sdk/log"
)

// errNexusTaskTimeout is returned when the Nexus task handler times out.
// It is used instead of context.DeadlineExceeded to allow the poller to differentiate between Nexus task handler
// timeout and other errors.
var errNexusTaskTimeout = errors.New("nexus task timeout")

func nexusHandlerError(t nexus.HandlerErrorType, message string) *nexuspb.HandlerError {
	return &nexuspb.HandlerError{
		ErrorType: string(t),
		Failure: &nexuspb.Failure{
			Message: message,
		},
	}
}

type nexusTaskHandler struct {
	nexusHandler     nexus.Handler
	identity         string
	namespace        string
	taskQueueName    string
	client           Client
	dataConverter    converter.DataConverter
	failureConverter converter.FailureConverter
	logger           log.Logger
	metricsHandler   metrics.Handler
	registry         *registry
}

func newNexusTaskHandler(
	nexusHandler nexus.Handler,
	identity string,
	namespace string,
	taskQueueName string,
	client Client,
	dataConverter converter.DataConverter,
	failureConverter converter.FailureConverter,
	logger log.Logger,
	metricsHandler metrics.Handler,
	registry *registry,
) *nexusTaskHandler {
	return &nexusTaskHandler{
		nexusHandler:     nexusHandler,
		logger:           logger,
		dataConverter:    dataConverter,
		failureConverter: failureConverter,
		identity:         identity,
		namespace:        namespace,
		taskQueueName:    taskQueueName,
		client:           client,
		metricsHandler:   metricsHandler,
		registry:         registry,
	}
}

func (h *nexusTaskHandler) Execute(task *workflowservice.PollNexusTaskQueueResponse) (*workflowservice.RespondNexusTaskCompletedRequest, *workflowservice.RespondNexusTaskFailedRequest, error) {
	nctx, handlerErr := h.newNexusOperationContext(task)
	if handlerErr != nil {
		return nil, h.fillInFailure(task.TaskToken, handlerErr), nil
	}
	res, handlerErr, err := h.execute(nctx, task)
	if err != nil {
		return nil, nil, err
	}
	if handlerErr != nil {
		return nil, h.fillInFailure(task.TaskToken, handlerErr), nil
	}
	return h.fillInCompletion(task.TaskToken, res), nil, nil
}

func (h *nexusTaskHandler) ExecuteContext(nctx *NexusOperationContext, task *workflowservice.PollNexusTaskQueueResponse) (*workflowservice.RespondNexusTaskCompletedRequest, *workflowservice.RespondNexusTaskFailedRequest, error) {
	res, handlerErr, err := h.execute(nctx, task)
	if err != nil {
		return nil, nil, err
	}
	if handlerErr != nil {
		return nil, h.fillInFailure(task.TaskToken, handlerErr), nil
	}
	return h.fillInCompletion(task.TaskToken, res), nil, nil
}

func (h *nexusTaskHandler) execute(nctx *NexusOperationContext, task *workflowservice.PollNexusTaskQueueResponse) (*nexuspb.Response, *nexuspb.HandlerError, error) {
	header := nexus.Header(task.GetRequest().GetHeader())
	if header == nil {
		header = nexus.Header{}
	}

	ctx, cancel, handlerErr := h.goContextForTask(nctx, header)
	if handlerErr != nil {
		return nil, handlerErr, nil
	}
	defer cancel()

	switch req := task.GetRequest().GetVariant().(type) {
	case *nexuspb.Request_StartOperation:
		return h.handleStartOperation(ctx, nctx, req.StartOperation, header)
	case *nexuspb.Request_CancelOperation:
		return h.handleCancelOperation(ctx, nctx, req.CancelOperation, header)
	default:
		return nil, nexusHandlerError(nexus.HandlerErrorTypeNotImplemented, "unknown request type"), nil
	}
}

func (h *nexusTaskHandler) handleStartOperation(
	ctx context.Context,
	nctx *NexusOperationContext,
	req *nexuspb.StartOperationRequest,
	header nexus.Header,
) (*nexuspb.Response, *nexuspb.HandlerError, error) {
	serializer := &payloadSerializer{
		converter: h.dataConverter,
		payload:   req.GetPayload(),
	}
	// Create a fake lazy value, Temporal server already converts Nexus content into payloads.
	input := nexus.NewLazyValue(
		serializer,
		&nexus.Reader{
			ReadCloser: emptyReaderNopCloser,
		},
	)
	// Ensure we don't pass nil values to handlers.
	callbackHeader := req.GetCallbackHeader()
	if callbackHeader == nil {
		callbackHeader = make(map[string]string)
	}
	nexusLinks := make([]nexus.Link, 0, len(req.GetLinks()))
	for _, link := range req.GetLinks() {
		if link == nil {
			continue
		}
		linkURL, err := url.Parse(link.GetUrl())
		if err != nil {
			nctx.log.Error("Failed to parse link url: %s", link.GetUrl(), tagError, err)
			return nil, nexusHandlerError(nexus.HandlerErrorTypeBadRequest, "failed to parse link url"), nil
		}
		nexusLinks = append(nexusLinks, nexus.Link{
			URL:  linkURL,
			Type: link.GetType(),
		})
	}
	startOptions := nexus.StartOperationOptions{
		RequestID:      req.RequestId,
		CallbackURL:    req.Callback,
		Header:         header,
		CallbackHeader: callbackHeader,
		Links:          nexusLinks,
	}
	ctx = nexus.WithHandlerContext(ctx, nexus.HandlerInfo{
		Service:   req.GetService(),
		Operation: req.GetOperation(),
		Header:    header,
	})
	var opres nexus.HandlerStartOperationResult[any]
	var err error
	var panic bool
	func() {
		defer func() {
			recovered := recover()
			if recovered != nil {
				panic = true
				var ok bool
				err, ok = recovered.(error)
				if !ok {
					err = fmt.Errorf("panic: %v", recovered)
				}

				nctx.log.Error("Panic captured while handling Nexus task", tagStackTrace, string(debug.Stack()), tagError, err)
			}
		}()
		opres, err = h.nexusHandler.StartOperation(ctx, req.GetService(), req.GetOperation(), input, startOptions)
	}()
	if ctx.Err() != nil {
		if !panic {
			nctx.log.Error("Context error while processing Nexus task", tagError, ctx.Err())
		}
		return nil, nil, errNexusTaskTimeout
	}
	if err != nil {
		if !panic {
			nctx.log.Error("Handler returned error while processing Nexus task", tagError, err)
		}
		var unsuccessfulOperationErr *nexus.OperationError
		err = convertKnownErrors(err)
		if errors.As(err, &unsuccessfulOperationErr) {
			failure, err := h.errorToFailure(unsuccessfulOperationErr.Cause)
			if err != nil {
				return nil, nil, err
			}

			return &nexuspb.Response{
				Variant: &nexuspb.Response_StartOperation{
					StartOperation: &nexuspb.StartOperationResponse{
						Variant: &nexuspb.StartOperationResponse_OperationError{
							OperationError: &nexuspb.UnsuccessfulOperationError{
								OperationState: string(unsuccessfulOperationErr.State),
								Failure:        failure,
							},
						},
					},
				},
			}, nil, nil
		}
		var handlerErr *nexus.HandlerError
		if errors.As(err, &handlerErr) {
			protoErr, err := h.nexusHandlerErrorToProto(handlerErr)
			return nil, protoErr, err
		}
		// Default to internal error.
		protoErr, err := h.internalError(err)
		return nil, protoErr, err
	}
	switch t := opres.(type) {
	case *nexus.HandlerStartOperationResultAsync:
		nexusLinks := nexus.HandlerLinks(ctx)
		links := make([]*nexuspb.Link, len(nexusLinks))
		for i, nexusLink := range nexusLinks {
			links[i] = &nexuspb.Link{
				Url:  nexusLink.URL.String(),
				Type: nexusLink.Type,
			}
		}
		token := t.OperationToken
		//lint:ignore SA1019 this field might be set by users of older SDKs.
		if t.OperationID != "" {
			token = t.OperationID //lint:ignore SA1019 this field might be set by users of older SDKs.
		}
		return &nexuspb.Response{
			Variant: &nexuspb.Response_StartOperation{
				StartOperation: &nexuspb.StartOperationResponse{
					Variant: &nexuspb.StartOperationResponse_AsyncSuccess{
						AsyncSuccess: &nexuspb.StartOperationResponse_Async{
							OperationToken: token,
							OperationId:    token,
							Links:          links,
						},
					},
				},
			},
		}, nil, nil
	default:
		nexusLinks := nexus.HandlerLinks(ctx)
		links := make([]*nexuspb.Link, len(nexusLinks))
		for i, nexusLink := range nexusLinks {
			links[i] = &nexuspb.Link{
				Url:  nexusLink.URL.String(),
				Type: nexusLink.Type,
			}
		}
		// *nexus.HandlerStartOperationResultSync is generic, we can't type switch unfortunately.
		value := reflect.ValueOf(t).Elem().FieldByName("Value").Interface()
		payload, err := h.dataConverter.ToPayload(value)
		if err != nil {
			nctx.log.Error("Cannot convert Nexus sync result", tagError, err)
			protoErr, err := h.internalError(fmt.Errorf("cannot convert nexus sync result: %w", err))
			return nil, protoErr, err
		}
		return &nexuspb.Response{
			Variant: &nexuspb.Response_StartOperation{
				StartOperation: &nexuspb.StartOperationResponse{
					Variant: &nexuspb.StartOperationResponse_SyncSuccess{
						SyncSuccess: &nexuspb.StartOperationResponse_Sync{
							Payload: payload,
							Links:   links,
						},
					},
				},
			},
		}, nil, nil
	}
}

func (h *nexusTaskHandler) handleCancelOperation(ctx context.Context, nctx *NexusOperationContext, req *nexuspb.CancelOperationRequest, header nexus.Header) (*nexuspb.Response, *nexuspb.HandlerError, error) {
	cancelOptions := nexus.CancelOperationOptions{Header: header}
	ctx = nexus.WithHandlerContext(ctx, nexus.HandlerInfo{
		Service:   req.GetService(),
		Operation: req.GetOperation(),
		Header:    header,
	})
	var err error
	var panic bool
	func() {
		defer func() {
			recovered := recover()
			if recovered != nil {
				panic = true
				var ok bool
				err, ok = recovered.(error)
				if !ok {
					err = fmt.Errorf("panic: %v", recovered)
				}

				nctx.log.Error("Panic captured while handling Nexus task", tagStackTrace, string(debug.Stack()), tagError, err)
			}
		}()
		token := req.GetOperationToken()
		if token == "" {
			// Support servers older than 1.27.0.
			token = req.GetOperationId()
		}
		err = h.nexusHandler.CancelOperation(ctx, req.GetService(), req.GetOperation(), token, cancelOptions)
	}()
	if ctx.Err() != nil {
		if !panic {
			nctx.log.Error("Context error while processing Nexus task", tagError, ctx.Err())
		}
		return nil, nil, errNexusTaskTimeout
	}
	if err != nil {
		if !panic {
			nctx.log.Error("Handler returned error while processing Nexus task", tagError, err)
		}
		err = convertKnownErrors(err)
		var handlerErr *nexus.HandlerError
		if errors.As(err, &handlerErr) {
			protoErr, err := h.nexusHandlerErrorToProto(handlerErr)
			return nil, protoErr, err
		}
		// Default to internal error.
		protoErr, err := h.internalError(err)
		return nil, protoErr, err
	}

	return &nexuspb.Response{
		Variant: &nexuspb.Response_CancelOperation{
			CancelOperation: &nexuspb.CancelOperationResponse{},
		},
	}, nil, nil
}

func (h *nexusTaskHandler) internalError(err error) (*nexuspb.HandlerError, error) {
	failure, err := h.errorToFailure(err)
	if err != nil {
		return nil, err
	}
	return &nexuspb.HandlerError{ErrorType: string(nexus.HandlerErrorTypeInternal), Failure: failure}, nil
}

func (h *nexusTaskHandler) goContextForTask(nctx *NexusOperationContext, header nexus.Header) (context.Context, context.CancelFunc, *nexuspb.HandlerError) {
	// Associate the NexusOperationContext with the context.Context used to invoke operations.
	ctx := context.WithValue(context.Background(), nexusOperationContextKey, nctx)

	timeoutStr := header.Get(nexus.HeaderRequestTimeout)
	if timeoutStr != "" {
		timeout, err := time.ParseDuration(timeoutStr)
		if err != nil {
			return nil, nil, nexusHandlerError(nexus.HandlerErrorTypeBadRequest, "cannot parse request timeout")
		}

		ctx, cancel := context.WithTimeout(ctx, timeout)
		return ctx, cancel, nil
	}

	return ctx, func() {}, nil
}

func (h *nexusTaskHandler) newNexusOperationContext(response *workflowservice.PollNexusTaskQueueResponse) (*NexusOperationContext, *nexuspb.HandlerError) {
	var service, operation string

	switch req := response.GetRequest().GetVariant().(type) {
	case *nexuspb.Request_StartOperation:
		service = req.StartOperation.Service
		operation = req.StartOperation.Operation
	case *nexuspb.Request_CancelOperation:
		service = req.CancelOperation.Service
		operation = req.CancelOperation.Operation
	default:
		return nil, nexusHandlerError(nexus.HandlerErrorTypeNotImplemented, "unknown request type")
	}

	logger := log.With(h.logger,
		tagNexusService, service,
		tagNexusOperation, operation,
		tagTaskQueue, h.taskQueueName,
	)
	metricsHandler := h.metricsHandler.WithTags(metrics.NexusTags(service, operation, h.taskQueueName))

	return &NexusOperationContext{
		client:         h.client,
		Namespace:      h.namespace,
		TaskQueue:      h.taskQueueName,
		metricsHandler: metricsHandler,
		log:            logger,
		registry:       h.registry,
	}, nil
}

func (h *nexusTaskHandler) fillInCompletion(taskToken []byte, res *nexuspb.Response) *workflowservice.RespondNexusTaskCompletedRequest {
	return &workflowservice.RespondNexusTaskCompletedRequest{
		Identity:  h.identity,
		Namespace: h.namespace,
		TaskToken: taskToken,
		Response:  res,
	}
}

func (h *nexusTaskHandler) fillInFailure(taskToken []byte, err *nexuspb.HandlerError) *workflowservice.RespondNexusTaskFailedRequest {
	return &workflowservice.RespondNexusTaskFailedRequest{
		Identity:  h.identity,
		Namespace: h.namespace,
		TaskToken: taskToken,
		Error:     err,
	}
}

var nexusFailureTypeString = string((&failurepb.Failure{}).ProtoReflect().Descriptor().FullName())
var nexusFailureMetadata = map[string]string{"type": nexusFailureTypeString}

func (h *nexusTaskHandler) errorToFailure(err error) (*nexuspb.Failure, error) {
	failure := h.failureConverter.ErrorToFailure(err)
	if failure == nil {
		return nil, nil
	}
	message := failure.Message
	failure.Message = ""
	b, err := protojson.Marshal(failure)
	if err != nil {
		return nil, err
	}
	return &nexuspb.Failure{
		Message:  message,
		Metadata: nexusFailureMetadata,
		Details:  b,
	}, nil
}

func (h *nexusTaskHandler) nexusHandlerErrorToProto(handlerErr *nexus.HandlerError) (*nexuspb.HandlerError, error) {
	failure, err := h.errorToFailure(handlerErr.Cause)
	if err != nil {
		return nil, err
	}
	var retryBehavior enumspb.NexusHandlerErrorRetryBehavior
	switch handlerErr.RetryBehavior {
	case nexus.HandlerErrorRetryBehaviorRetryable:
		retryBehavior = enumspb.NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_RETRYABLE
	case nexus.HandlerErrorRetryBehaviorNonRetryable:
		retryBehavior = enumspb.NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_NON_RETRYABLE
	}
	return &nexuspb.HandlerError{
		ErrorType:     string(handlerErr.Type),
		Failure:       failure,
		RetryBehavior: retryBehavior,
	}, nil
}

// payloadSerializer is a fake nexus Serializer that uses a data converter to read from an embedded payload instead of
// using the given nexus.Context. Supports only Deserialize.
type payloadSerializer struct {
	converter converter.DataConverter
	payload   *common.Payload
}

func (p *payloadSerializer) Deserialize(_ *nexus.Content, v any) error {
	return p.converter.FromPayload(p.payload, v)
}

func (p *payloadSerializer) Serialize(v any) (*nexus.Content, error) {
	panic("unimplemented") // not used - operation outputs are directly serialized to payload.
}

var emptyReaderNopCloser = io.NopCloser(bytes.NewReader([]byte{}))

// convertKnownErrors converts known errors to corresponding Nexus HandlerError.
func convertKnownErrors(err error) error {
	// Not using errors.As to be consistent ApplicationError checking with the rest of the SDK.
	if appErr, ok := err.(*ApplicationError); ok && appErr.NonRetryable() {
		return &nexus.HandlerError{
			Type:          nexus.HandlerErrorTypeInternal,
			Cause:         appErr,
			RetryBehavior: nexus.HandlerErrorRetryBehaviorNonRetryable,
		}
	}
	return convertServiceError(err)
}

// convertServiceError converts a serviceerror into a Nexus HandlerError if possible.
// If exposeDetails is true, the error message from the given error is exposed in the converted HandlerError, otherwise,
// a default message with minimal information is attached to the returned error.
// Roughly taken from https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto
// and
// https://github.com/grpc-ecosystem/grpc-gateway/blob/a7cf811e6ffabeaddcfb4ff65602c12671ff326e/runtime/errors.go#L56.
func convertServiceError(err error) error {
	var st *status.Status

	// Temporal serviceerrors have a Status() method.
	stGetter, ok := err.(interface{ Status() *status.Status })
	if !ok {
		// Not a serviceerror, passthrough.
		return err
	}

	st = stGetter.Status()

	switch st.Code() {
	case codes.InvalidArgument:
		return &nexus.HandlerError{Type: nexus.HandlerErrorTypeBadRequest, Cause: err}
	case codes.AlreadyExists, codes.FailedPrecondition, codes.OutOfRange:
		return &nexus.HandlerError{Type: nexus.HandlerErrorTypeInternal, Cause: err, RetryBehavior: nexus.HandlerErrorRetryBehaviorNonRetryable}
	case codes.Aborted, codes.Unavailable:
		return &nexus.HandlerError{Type: nexus.HandlerErrorTypeUnavailable, Cause: err}
	case codes.Canceled, codes.DataLoss, codes.Internal, codes.Unknown, codes.Unauthenticated, codes.PermissionDenied:
		// Note that codes.Unauthenticated, codes.PermissionDenied have Nexus error types but we convert to internal
		// because this is not a client auth error and happens when the handler fails to auth with Temporal and should
		// be considered retryable.
		return &nexus.HandlerError{Type: nexus.HandlerErrorTypeInternal, Cause: err}
	case codes.NotFound:
		return &nexus.HandlerError{Type: nexus.HandlerErrorTypeNotFound, Cause: err}
	case codes.ResourceExhausted:
		return &nexus.HandlerError{Type: nexus.HandlerErrorTypeResourceExhausted, Cause: err}
	case codes.Unimplemented:
		return &nexus.HandlerError{Type: nexus.HandlerErrorTypeNotImplemented, Cause: err}
	case codes.DeadlineExceeded:
		return &nexus.HandlerError{Type: nexus.HandlerErrorTypeUpstreamTimeout, Cause: err}
	}

	return err
}
