// The MIT License
//
// Copyright (c) 2022 Temporal Technologies Inc.  All rights reserved.
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

	"github.com/gogo/protobuf/proto"
	commonpb "go.temporal.io/api/common/v1"
	failurepb "go.temporal.io/api/failure/v1"
	"go.temporal.io/sdk/converter"
)

var defaultFailureConverter = NewDefaultFailureConverter(DefaultFailureConverterOptions{})

// GetDefaultFailureConverter returns the default failure converter used by Temporal.
func GetDefaultFailureConverter() converter.FailureConverter {
	return defaultFailureConverter
}

// DefaultFailureConverterOptions are optional parameters for DefaultFailureConverter creation.
type DefaultFailureConverterOptions struct {
	// Optional: Sets DataConverter to customize serialization/deserialization of fields.
	// default: Default data converter
	DataConverter converter.DataConverter

	// Optional: Whether to encode error messages and stack traces.
	// default: false
	EncodeCommonAttributes bool
}

// DefaultFailureConverter seralizes errors with the option to encode common parameters under Failure.EncodedAttributes
type DefaultFailureConverter struct {
	dataConverter          converter.DataConverter
	encodeCommonAttributes bool
}

// NewDefaultFailureConverter creates new instance of DefaultFailureConverter.
func NewDefaultFailureConverter(opt DefaultFailureConverterOptions) *DefaultFailureConverter {
	if opt.DataConverter == nil {
		opt.DataConverter = converter.GetDefaultDataConverter()
	}
	return &DefaultFailureConverter{
		dataConverter:          opt.DataConverter,
		encodeCommonAttributes: opt.EncodeCommonAttributes,
	}
}

// ErrorToFailure converts an error to a Failure
func (dfc *DefaultFailureConverter) ErrorToFailure(err error) *failurepb.Failure {
	if err == nil {
		return nil
	}

	if fh, ok := err.(failureHolder); ok {
		if fh.failure() != nil {
			return fh.failure()
		}
	}

	failure := &failurepb.Failure{
		Source: "GoSDK",
	}

	if m, ok := err.(messenger); ok && m != nil {
		failure.Message = m.message()
	} else {
		failure.Message = err.Error()
	}

	switch err := err.(type) {
	case *ApplicationError:
		failureInfo := &failurepb.ApplicationFailureInfo{
			Type:         err.errType,
			NonRetryable: err.nonRetryable,
			Details:      convertErrDetailsToPayloads(err.details, dfc.dataConverter),
		}
		failure.FailureInfo = &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: failureInfo}
	case *CanceledError:
		failureInfo := &failurepb.CanceledFailureInfo{
			Details: convertErrDetailsToPayloads(err.details, dfc.dataConverter),
		}
		failure.FailureInfo = &failurepb.Failure_CanceledFailureInfo{CanceledFailureInfo: failureInfo}
	case *PanicError:
		failureInfo := &failurepb.ApplicationFailureInfo{
			Type: getErrType(err),
		}
		failure.FailureInfo = &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: failureInfo}
		failure.StackTrace = err.StackTrace()
	case *workflowPanicError:
		failureInfo := &failurepb.ApplicationFailureInfo{
			Type:         getErrType(&PanicError{}),
			NonRetryable: true,
		}
		failure.FailureInfo = &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: failureInfo}
		failure.StackTrace = err.StackTrace()
	case *TimeoutError:
		failureInfo := &failurepb.TimeoutFailureInfo{
			TimeoutType:          err.timeoutType,
			LastHeartbeatDetails: convertErrDetailsToPayloads(err.lastHeartbeatDetails, dfc.dataConverter),
		}
		failure.FailureInfo = &failurepb.Failure_TimeoutFailureInfo{TimeoutFailureInfo: failureInfo}
	case *TerminatedError:
		failureInfo := &failurepb.TerminatedFailureInfo{}
		failure.FailureInfo = &failurepb.Failure_TerminatedFailureInfo{TerminatedFailureInfo: failureInfo}
	case *ServerError:
		failureInfo := &failurepb.ServerFailureInfo{
			NonRetryable: err.nonRetryable,
		}
		failure.FailureInfo = &failurepb.Failure_ServerFailureInfo{ServerFailureInfo: failureInfo}
	case *ActivityError:
		failureInfo := &failurepb.ActivityFailureInfo{
			ScheduledEventId: err.scheduledEventID,
			StartedEventId:   err.startedEventID,
			Identity:         err.identity,
			ActivityType:     err.activityType,
			ActivityId:       err.activityID,
			RetryState:       err.retryState,
		}
		failure.FailureInfo = &failurepb.Failure_ActivityFailureInfo{ActivityFailureInfo: failureInfo}
	case *ChildWorkflowExecutionError:
		failureInfo := &failurepb.ChildWorkflowExecutionFailureInfo{
			Namespace: err.namespace,
			WorkflowExecution: &commonpb.WorkflowExecution{
				WorkflowId: err.workflowID,
				RunId:      err.runID,
			},
			WorkflowType:     &commonpb.WorkflowType{Name: err.workflowType},
			InitiatedEventId: err.initiatedEventID,
			StartedEventId:   err.startedEventID,
			RetryState:       err.retryState,
		}
		failure.FailureInfo = &failurepb.Failure_ChildWorkflowExecutionFailureInfo{ChildWorkflowExecutionFailureInfo: failureInfo}
	default: // All unknown errors are considered to be retryable ApplicationFailureInfo.
		failureInfo := &failurepb.ApplicationFailureInfo{
			Type:         getErrType(err),
			NonRetryable: false,
		}
		failure.FailureInfo = &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: failureInfo}
	}

	failure.Cause = dfc.ErrorToFailure(errors.Unwrap(err))

	if dfc.encodeCommonAttributes {
		err := converter.EncodeCommonFailureAttributes(dfc.dataConverter, failure)
		if err != nil {
			panic(err)
		}
	}
	return failure
}

// FailureToError converts an Failure to an error
func (dfc *DefaultFailureConverter) FailureToError(failure *failurepb.Failure) error {
	if failure == nil {
		return nil
	}
	// Copy the original future to pass to the failureHolder
	originalFailure := proto.Clone(failure).(*failurepb.Failure)
	converter.DecodeCommonFailureAttributes(dfc.dataConverter, failure)

	message := failure.GetMessage()
	stackTrace := failure.GetStackTrace()
	var err error

	if failure.GetApplicationFailureInfo() != nil {
		applicationFailureInfo := failure.GetApplicationFailureInfo()
		details := newEncodedValues(applicationFailureInfo.GetDetails(), dfc.dataConverter)
		switch applicationFailureInfo.GetType() {
		case getErrType(&PanicError{}):
			err = newPanicError(message, stackTrace)
		default:
			err = NewApplicationError(
				message,
				applicationFailureInfo.GetType(),
				applicationFailureInfo.GetNonRetryable(),
				dfc.FailureToError(failure.GetCause()),
				details)
		}
	} else if failure.GetCanceledFailureInfo() != nil {
		details := newEncodedValues(failure.GetCanceledFailureInfo().GetDetails(), dfc.dataConverter)
		err = NewCanceledError(details)
	} else if failure.GetTimeoutFailureInfo() != nil {
		timeoutFailureInfo := failure.GetTimeoutFailureInfo()
		lastHeartbeatDetails := newEncodedValues(timeoutFailureInfo.GetLastHeartbeatDetails(), dfc.dataConverter)
		err = NewTimeoutError(
			message,
			timeoutFailureInfo.GetTimeoutType(),
			dfc.FailureToError(failure.GetCause()),
			lastHeartbeatDetails)
	} else if failure.GetTerminatedFailureInfo() != nil {
		err = newTerminatedError()
	} else if failure.GetServerFailureInfo() != nil {
		err = NewServerError(message, failure.GetServerFailureInfo().GetNonRetryable(), dfc.FailureToError(failure.GetCause()))
	} else if failure.GetResetWorkflowFailureInfo() != nil {
		err = NewApplicationError(message, "", true, dfc.FailureToError(failure.GetCause()), failure.GetResetWorkflowFailureInfo().GetLastHeartbeatDetails())
	} else if failure.GetActivityFailureInfo() != nil {
		activityTaskInfoFailure := failure.GetActivityFailureInfo()
		err = NewActivityError(
			activityTaskInfoFailure.GetScheduledEventId(),
			activityTaskInfoFailure.GetStartedEventId(),
			activityTaskInfoFailure.GetIdentity(),
			activityTaskInfoFailure.GetActivityType(),
			activityTaskInfoFailure.GetActivityId(),
			activityTaskInfoFailure.GetRetryState(),
			dfc.FailureToError(failure.GetCause()),
		)
	} else if failure.GetChildWorkflowExecutionFailureInfo() != nil {
		childWorkflowExecutionFailureInfo := failure.GetChildWorkflowExecutionFailureInfo()
		err = NewChildWorkflowExecutionError(
			childWorkflowExecutionFailureInfo.GetNamespace(),
			childWorkflowExecutionFailureInfo.GetWorkflowExecution().GetWorkflowId(),
			childWorkflowExecutionFailureInfo.GetWorkflowExecution().GetRunId(),
			childWorkflowExecutionFailureInfo.GetWorkflowType().GetName(),
			childWorkflowExecutionFailureInfo.GetInitiatedEventId(),
			childWorkflowExecutionFailureInfo.GetStartedEventId(),
			childWorkflowExecutionFailureInfo.GetRetryState(),
			dfc.FailureToError(failure.GetCause()),
		)
	}

	if err == nil {
		// All unknown types are considered to be retryable ApplicationError.
		err = NewApplicationError(message, "", false, dfc.FailureToError(failure.GetCause()))
	}

	if fh, ok := err.(failureHolder); ok {
		fh.setFailure(originalFailure)
	}

	return err
}
