// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
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
	"errors"
	"fmt"
	"reflect"

	"github.com/gogo/protobuf/types"
	commonpb "go.temporal.io/api/common/v1"
	historypb "go.temporal.io/api/history/v1"
	protocolpb "go.temporal.io/api/protocol/v1"
	updatepb "go.temporal.io/api/update/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/internal/protocol"
)

type updateState string

const (
	updateStateNew              updateState = "New"
	updateStateRequestInitiated updateState = "RequestScheduled"
	updateStateAccepted         updateState = "Accepted"
	updateStateCompleted        updateState = "Completed"

	updateProtocolV1 = "temporal.api.update.v1"
)

type (
	// UpdateCallbacks supplies callbacks for the different stages of processing
	// a workflow update.
	UpdateCallbacks interface {
		// Accept is called for an update after it has passed validation an
		// before execution has started.
		Accept()

		// Reject is called for an update if validation fails.
		Reject(err error)

		// Complete is called for an update with the result of executing the
		// update function. If the provided error is non-nil then the overall
		// outcome is understood to be a failure.
		Complete(success interface{}, err error)
	}

	// UpdateScheduluer allows an update state machine to spawn coroutines and
	// yield itself as necessary.
	UpdateScheduler interface {
		// Spawn starts a new named coroutine, executing the given function f.
		Spawn(ctx Context, name string, f func(ctx Context)) Context

		// Yield returns control to the scheduler.
		Yield(ctx Context, status string)
	}

	// updateEnv encapsulates the utility functions needed by update protocol
	// instance in order to implement the UpdateCallbacks interface. This
	// interface is conveniently implemented by
	// *workflowExecutionEventHandlerImpl.
	updateEnv interface {
		GetFailureConverter() converter.FailureConverter
		GetDataConverter() converter.DataConverter
		Send(*protocolpb.Message, ...msgSendOpt)
	}

	// updateProtocol wraps an updateEnv and some protocol metadata to
	// implement the UpdateCallbacks abstraction. It handles callbacks by
	// sending protocol lmessages.
	updateProtocol struct {
		protoInstanceID string
		requestMsgID    string
		requestSeqID    int64
		initialRequest  updatepb.Request
		scheduleUpdate  func(name string, args *commonpb.Payloads, header *commonpb.Header, callbacks UpdateCallbacks)
		env             updateEnv
		state           updateState
	}

	// updateHandler is the underlying type that is registered into a workflow
	// environment when the user-code in a workflow registers an update callback
	// for a given name. It offers the ability to invoke the associated
	// execution and validation functions.
	updateHandler struct {
		fn         interface{}
		validateFn interface{}
		name       string
	}
)

// newUpdateResponder constructs an updateProtocolResponder instance to handle
// update callbacks.
func newUpdateProtocol(
	protoInstanceID string,
	scheduleUpdate func(name string, args *commonpb.Payloads, header *commonpb.Header, callbacks UpdateCallbacks),
	env updateEnv,
) *updateProtocol {
	return &updateProtocol{
		protoInstanceID: protoInstanceID,
		env:             env,
		scheduleUpdate:  scheduleUpdate,
		state:           updateStateNew,
	}
}

func (up *updateProtocol) requireState(action string, valid ...updateState) {
	for _, validState := range valid {
		if up.state == validState {
			return
		}
	}
	panic(fmt.Sprintf("invalid action %q in update protocol from state %s", action, up.state))
}

func (up *updateProtocol) HandleMessage(msg *protocolpb.Message) error {
	if err := types.UnmarshalAny(msg.Body, &up.initialRequest); err != nil {
		return err
	}
	up.requireState("update request", updateStateNew)
	up.requestMsgID = msg.GetId()
	up.requestSeqID = msg.GetEventId()
	input := up.initialRequest.GetInput()
	up.scheduleUpdate(input.GetName(), input.GetArgs(), input.GetHeader(), up)
	up.state = updateStateRequestInitiated
	return nil
}

// Accept is called for an update after it has passed validation and
// before execution has started.
func (up *updateProtocol) Accept() {
	up.requireState("accept", updateStateRequestInitiated)
	up.env.Send(&protocolpb.Message{
		Id:                 up.protoInstanceID + "/accept",
		ProtocolInstanceId: up.protoInstanceID,
		Body: protocol.MustMarshalAny(&updatepb.Acceptance{
			AcceptedRequestMessageId:         up.requestMsgID,
			AcceptedRequestSequencingEventId: up.requestSeqID,
			AcceptedRequest:                  &up.initialRequest,
		}),
	}, withExpectedEventPredicate(up.checkAcceptedEvent))
	up.state = updateStateAccepted
}

// Reject is called for an update if validation fails.
func (up *updateProtocol) Reject(err error) {
	up.requireState("reject", updateStateNew, updateStateRequestInitiated)
	up.env.Send(&protocolpb.Message{
		Id:                 up.protoInstanceID + "/reject",
		ProtocolInstanceId: up.protoInstanceID,
		Body: protocol.MustMarshalAny(&updatepb.Rejection{
			RejectedRequestMessageId:         up.requestMsgID,
			RejectedRequestSequencingEventId: up.requestSeqID,
			RejectedRequest:                  &up.initialRequest,
			Failure:                          up.env.GetFailureConverter().ErrorToFailure(err),
		}),
	})
	up.state = updateStateCompleted
}

// Complete is called for an update with the result of executing the
// update function.
func (up *updateProtocol) Complete(success interface{}, outcomeErr error) {
	up.requireState("complete", updateStateAccepted)
	outcome := &updatepb.Outcome{}
	if outcomeErr != nil {
		outcome.Value = &updatepb.Outcome_Failure{
			Failure: up.env.GetFailureConverter().ErrorToFailure(outcomeErr),
		}
	} else {
		success, err := up.env.GetDataConverter().ToPayloads(success)
		if err != nil {
			panic(err)
		}
		outcome.Value = &updatepb.Outcome_Success{
			Success: success,
		}
	}
	up.env.Send(&protocolpb.Message{
		Id:                 up.protoInstanceID + "/complete",
		ProtocolInstanceId: up.protoInstanceID,
		Body: protocol.MustMarshalAny(&updatepb.Response{
			Meta:    up.initialRequest.GetMeta(),
			Outcome: outcome,
		}),
	}, withExpectedEventPredicate(up.checkCompletedEvent))
	up.state = updateStateCompleted
}

func (up *updateProtocol) checkCompletedEvent(e *historypb.HistoryEvent) bool {
	attrs := e.GetWorkflowExecutionUpdateCompletedEventAttributes()
	if attrs == nil {
		return false
	}
	return attrs.Meta.GetUpdateId() == up.protoInstanceID
}

func (up *updateProtocol) checkAcceptedEvent(e *historypb.HistoryEvent) bool {
	attrs := e.GetWorkflowExecutionUpdateAcceptedEventAttributes()
	if attrs == nil {
		return false
	}
	return attrs.AcceptedRequest.GetMeta().GetUpdateId() == up.protoInstanceID &&
		attrs.AcceptedRequestMessageId == up.requestMsgID &&
		attrs.AcceptedRequestSequencingEventId == up.requestSeqID
}

// defaultHandler receives the initial invocation of an upate during WFT
// processing. The implementation will verify that an updateHandler exists for
// the supplied name (rejecting the update otherwise) and use the provided spawn
// function to create a new coroutine that will execute in the workflow context.
// The spawned coroutine is what will actually invoke the user-supplied callback
// functions for validation and execution. Update progress is emitted via calls
// into the UpdateCallbacks parameter.
func defaultUpdateHandler(
	rootCtx Context,
	name string,
	serializedArgs *commonpb.Payloads,
	header *commonpb.Header,
	callbacks UpdateCallbacks,
	scheduler UpdateScheduler,
) {
	env := getWorkflowEnvironment(rootCtx)
	ctx, err := workflowContextWithHeaderPropagated(rootCtx, header, env.GetContextPropagators())
	if err != nil {
		callbacks.Reject(err)
		return
	}
	scheduler.Spawn(ctx, name, func(ctx Context) {
		eo := getWorkflowEnvOptions(ctx)

		// If we suspect that handler registration has not occurred (e.g.
		// because this update is part of the first workflow task and is being
		// delivered before the workflow function itself has run and had a
		// chance to register update handlers) then we yield control back to the
		// scheduler to allow handler registration to occur. The scheduler will
		// resume this coroutine after others have run to a blocking point.
		if len(eo.updateHandlers) == 0 {
			scheduler.Yield(ctx, "yielding for initial handler registration")
		}
		handler, ok := eo.updateHandlers[name]
		if !ok {
			keys := make([]string, 0, len(eo.updateHandlers))
			for k := range eo.updateHandlers {
				keys = append(keys, k)
			}
			callbacks.Reject(fmt.Errorf("unknown update %v. KnownUpdates=%v", name, keys))
			return
		}

		args, err := decodeArgsToRawValues(
			env.GetDataConverter(),
			reflect.TypeOf(handler.fn),
			serializedArgs,
		)
		if err != nil {
			callbacks.Reject(fmt.Errorf("unable to decode the input for update %q: %w", name, err))
			return
		}
		input := UpdateInput{Name: name, Args: args}

		envInterceptor := getWorkflowEnvironmentInterceptor(ctx)
		if !IsReplaying(ctx) {
			// we don't execute update validation during replay so that
			// validation routines can change across versions
			if err := envInterceptor.inboundInterceptor.ValidateUpdate(ctx, &input); err != nil {
				callbacks.Reject(err)
				return
			}
		}
		callbacks.Accept()
		success, err := envInterceptor.inboundInterceptor.ExecuteUpdate(ctx, &input)
		callbacks.Complete(success, err)
	})
}

// newUpdateHandler instantiates a new updateHandler if the supplied handler and
// opts.Validator functions pass validation of their respective interfaces and
// that the two interfaces are themselves equivalent (allowing for them to
// differ by the presence/absence of a leading Context parameter).
func newUpdateHandler(
	updateName string,
	handler interface{},
	opts UpdateHandlerOptions,
) (*updateHandler, error) {
	if err := validateUpdateHandlerFn(handler); err != nil {
		return nil, err
	}
	var validateFn interface{} = func(...interface{}) error { return nil }
	if opts.Validator != nil {
		if err := validateValidatorFn(opts.Validator); err != nil {
			return nil, err
		}
		if err := validateEquivalentParams(handler, opts.Validator); err != nil {
			return nil, err
		}
		validateFn = opts.Validator
	}
	return &updateHandler{
		fn:         handler,
		validateFn: validateFn,
		name:       updateName,
	}, nil
}

// validate invokes the update's validation function.
func (h *updateHandler) validate(ctx Context, input []interface{}) (err error) {
	defer func() {
		if p := recover(); p != nil {
			st := getStackTraceRaw("update validator [panic]:", 7, 0)
			err = newPanicError(fmt.Sprintf("update validator panic: %v", p), st)
		}
	}()
	_, err = executeFunctionWithWorkflowContext(ctx, h.validateFn, input)
	return err
}

// execute executes the update itself.
func (h *updateHandler) execute(ctx Context, input []interface{}) (result interface{}, err error) {
	return executeFunctionWithWorkflowContext(ctx, h.fn, input)
}

// HasCompleted allows the completion status of the update protocol to be
// observed externally.
func (up *updateProtocol) HasCompleted() bool {
	return up.state == updateStateCompleted
}

// validateValidatorFn validates that the supplied interface
//
// 1. is a function
// 2. has exactly one return parameter
// 3. the one return prarmeter is of type `error`
func validateValidatorFn(fn interface{}) error {
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		return fmt.Errorf("validator must be function but was %s", fnType.Kind())
	}

	if fnType.NumOut() != 1 {
		return fmt.Errorf(
			"validator must return exactly 1 value (an error), but found %d return values",
			fnType.NumOut(),
		)
	}

	if !isError(fnType.Out(0)) {
		return fmt.Errorf(
			"return value of validator must be error but found %v",
			fnType.Out(fnType.NumOut()-1).Kind(),
		)
	}
	return nil
}

// validateUpdateHandlerFn validates that the supplied interface
//
// 1. is a function
// 2. has one or two return parameters, the last of which is of type `error`
// 3. if there are two return parameters, the first is a serializable type
func validateUpdateHandlerFn(fn interface{}) error {
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		return fmt.Errorf("handler must be function but was %s", fnType.Kind())
	}
	switch fnType.NumOut() {
	case 1:
		if !isError(fnType.Out(0)) {
			return fmt.Errorf(
				"last return value of handler must be error but found %v",
				fnType.Out(0).Kind(),
			)
		}
	case 2:
		if !isValidResultType(fnType.Out(0)) {
			return fmt.Errorf(
				"first return value of handler must be serializable but found: %v",
				fnType.Out(0).Kind(),
			)
		}
		if !isError(fnType.Out(1)) {
			return fmt.Errorf(
				"last return value of handler must be error but found %v",
				fnType.Out(1).Kind(),
			)
		}
	default:
		return errors.New("update handler return signature must be a single " +
			"error or a serializable result and error (i.e. (ResultType, error))")
	}
	return nil
}
