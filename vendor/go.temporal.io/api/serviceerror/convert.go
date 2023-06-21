// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
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

package serviceerror

import (
	"context"
	"errors"

	"github.com/gogo/status"
	"google.golang.org/grpc/codes"

	"go.temporal.io/api/errordetails/v1"
)

// ToStatus converts service error to gogo gRPC Status.
// If error is not a service error it returns status with code Unknown.
func ToStatus(err error) *status.Status {
	if err == nil {
		return status.New(codes.OK, "")
	}

	if svcerr, ok := err.(ServiceError); ok {
		return svcerr.Status()
	}

	// Special case for context.DeadlineExceeded and context.Canceled because they can happen in unpredictable places.
	if errors.Is(err, context.DeadlineExceeded) {
		return status.New(codes.DeadlineExceeded, err.Error())
	}
	if errors.Is(err, context.Canceled) {
		return status.New(codes.Canceled, err.Error())
	}

	// Internal logic of status.Convert is:
	//   - if err is already gogo Status or gRPC Status, then just return it (this should never happen though).
	//   - otherwise returns codes.Unknown with message from err.Error() (this might happen if some generic go error reach to this point).
	return status.Convert(err)
}

// FromStatus converts gogo gRPC Status to service error.
func FromStatus(st *status.Status) error {
	if st == nil || st.Code() == codes.OK {
		return nil
	}

	// Simple case. `st.Code()` to `serviceerror` is one to one mapping and there are no error details.
	switch st.Code() {
	case codes.DataLoss:
		return newDataLoss(st)
	case codes.DeadlineExceeded:
		return newDeadlineExceeded(st)
	case codes.Canceled:
		return newCanceled(st)
	case codes.Unavailable:
		return newUnavailable(st)
	case codes.Unimplemented:
		return newUnimplemented(st)
	case codes.Unknown:
		// Unwrap error message from unknown error.
		return errors.New(st.Message())

	// Unsupported codes.
	case codes.Aborted,
		codes.Unauthenticated:
		// Use standard gRPC error representation for unsupported codes ("rpc error: code = %s desc = %s").
		return st.Err()
	}

	errDetails := extractErrorDetails(st)
	// If there was an error during details extraction, for example unknown message type,
	// which can happen when new error details are added and getting read by old clients,
	// then errDetails will be of type `error` with corresponding error inside.
	// This error is ignored and `serviceerror` is built using `st.Code()` only.

	switch st.Code() {
	case codes.Internal:
		switch errDetails := errDetails.(type) {
		case *errordetails.SystemWorkflowFailure:
			return newSystemWorkflow(st, errDetails)
		default:
			return newInternal(st)
		}
	case codes.NotFound:
		switch errDetails := errDetails.(type) {
		case *errordetails.NotFoundFailure:
			return newNotFound(st, errDetails)
		case *errordetails.NamespaceNotFoundFailure:
			return newNamespaceNotFound(st, errDetails)
		default:
			return newNotFound(st, nil)
		}
	case codes.InvalidArgument:
		switch errDetails.(type) {
		case *errordetails.QueryFailedFailure:
			return newQueryFailed(st)
		default:
			return newInvalidArgument(st)
		}
	case codes.ResourceExhausted:
		switch errDetails := errDetails.(type) {
		case *errordetails.ResourceExhaustedFailure:
			return newResourceExhausted(st, errDetails)
		default:
			return newResourceExhausted(st, nil)
		}
	case codes.AlreadyExists:
		switch errDetails := errDetails.(type) {
		case *errordetails.NamespaceAlreadyExistsFailure:
			return newNamespaceAlreadyExists(st)
		case *errordetails.WorkflowExecutionAlreadyStartedFailure:
			return newWorkflowExecutionAlreadyStarted(st, errDetails)
		case *errordetails.CancellationAlreadyRequestedFailure:
			return newCancellationAlreadyRequested(st)
		default:
			return newAlreadyExists(st)
		}
	case codes.FailedPrecondition:
		switch errDetails := errDetails.(type) {
		case *errordetails.NamespaceNotActiveFailure:
			return newNamespaceNotActive(st, errDetails)
		case *errordetails.NamespaceInvalidStateFailure:
			return newNamespaceInvalidState(st, errDetails)
		case *errordetails.ClientVersionNotSupportedFailure:
			return newClientVersionNotSupported(st, errDetails)
		case *errordetails.ServerVersionNotSupportedFailure:
			return newServerVersionNotSupported(st, errDetails)
		case *errordetails.WorkflowNotReadyFailure:
			return newWorkflowNotReady(st)
		default:
			return newFailedPrecondition(st)
		}
	case codes.PermissionDenied:
		switch errDetails := errDetails.(type) {
		case *errordetails.PermissionDeniedFailure:
			return newPermissionDenied(st, errDetails)
		default:
			return newPermissionDenied(st, nil)
		}
	case codes.OutOfRange:
		switch errDetails := errDetails.(type) {
		case *errordetails.NewerBuildExistsFailure:
			return newNewerBuildExists(st, errDetails)
		default:
			// fall through to st.Err()
		}
	}

	// `st.Code()` has unknown value (should never happen).
	// Use standard gRPC error representation "rpc error: code = %s desc = %s".
	return st.Err()
}

func extractErrorDetails(st *status.Status) interface{} {
	details := st.Details()
	if len(details) > 0 {
		return details[0]
	}

	return nil
}
