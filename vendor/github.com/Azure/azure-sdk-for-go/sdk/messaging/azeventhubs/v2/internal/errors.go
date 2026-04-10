// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package internal

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/amqpwrap"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/eh"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/exported"
	"github.com/Azure/go-amqp"
)

type errNonRetriable struct {
	Message string
}

func NewErrNonRetriable(message string) error {
	return errNonRetriable{Message: message}
}

func (e errNonRetriable) Error() string { return e.Message }

// RecoveryKind dictates what kind of recovery is possible. Used with
// GetRecoveryKind().
type RecoveryKind string

const (
	RecoveryKindNone  RecoveryKind = ""
	RecoveryKindFatal RecoveryKind = "fatal"
	RecoveryKindLink  RecoveryKind = "link"
	RecoveryKindConn  RecoveryKind = "connection"
)

func IsFatalEHError(err error) bool {
	return GetRecoveryKind(err) == RecoveryKindFatal
}

// TransformError will create a proper error type that users
// can potentially inspect.
// If the error is actionable then it'll be of type exported.Error which
// has a 'Code' field that can be used programatically.
// If it's not actionable or if it's nil it'll just be returned.
func TransformError(err error) error {
	if err == nil {
		return nil
	}

	_, ok := err.(*exported.Error)

	if ok {
		// it's already been wrapped.
		return err
	}

	if IsOwnershipLostError(err) {
		return exported.NewError(exported.ErrorCodeOwnershipLost, err)
	}

	// there are a few errors that all boil down to "bad creds or unauthorized"
	var amqpErr *amqp.Error

	if errors.As(err, &amqpErr) && amqpErr.Condition == amqp.ErrCondUnauthorizedAccess {
		return exported.NewError(exported.ErrorCodeUnauthorizedAccess, err)
	}

	var rpcErr RPCError
	if errors.As(err, &rpcErr) && rpcErr.Resp.Code == http.StatusUnauthorized {
		return exported.NewError(exported.ErrorCodeUnauthorizedAccess, err)
	}

	rk := GetRecoveryKind(err)

	switch rk {
	case RecoveryKindLink:
		// note that we could give back a more differentiated error code
		// here but it's probably best to just give the customer the simplest
		// recovery mechanism possible.
		return exported.NewError(exported.ErrorCodeConnectionLost, err)
	case RecoveryKindConn:
		return exported.NewError(exported.ErrorCodeConnectionLost, err)
	default:
		// isn't one of our specifically called out cases so we'll just return it.
		return err
	}
}

func IsQuickRecoveryError(err error) bool {
	if IsOwnershipLostError(err) {
		return false
	}

	var de *amqp.LinkError
	return errors.As(err, &de)
}

func IsCancelError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	if err.Error() == "context canceled" { // go-amqp is returning this when I cancel
		return true
	}

	return false
}

const errorConditionLockLost = amqp.ErrCond("com.microsoft:message-lock-lost")

var amqpConditionsToRecoveryKind = map[amqp.ErrCond]RecoveryKind{
	// no recovery needed, these are temporary errors.
	amqp.ErrCond("com.microsoft:server-busy"):         RecoveryKindNone,
	amqp.ErrCond("com.microsoft:timeout"):             RecoveryKindNone,
	amqp.ErrCond("com.microsoft:operation-cancelled"): RecoveryKindNone,

	// Link recovery needed
	amqp.ErrCondDetachForced:          RecoveryKindLink, // "amqp:link:detach-forced"
	amqp.ErrCondTransferLimitExceeded: RecoveryKindLink, // "amqp:link:transfer-limit-exceeded"

	// Connection recovery needed
	amqp.ErrCondConnectionForced: RecoveryKindConn, // "amqp:connection:forced"
	amqp.ErrCondInternalError:    RecoveryKindConn, // "amqp:internal-error"

	// No recovery possible - this operation is non retriable.

	// ErrCondResourceLimitExceeded comes back if the entity is actually full.
	amqp.ErrCondResourceLimitExceeded:                      RecoveryKindFatal, // "amqp:resource-limit-exceeded"
	amqp.ErrCondMessageSizeExceeded:                        RecoveryKindFatal, // "amqp:link:message-size-exceeded"
	amqp.ErrCondUnauthorizedAccess:                         RecoveryKindFatal, // creds are bad
	amqp.ErrCondNotFound:                                   RecoveryKindFatal, // "amqp:not-found"
	amqp.ErrCondNotAllowed:                                 RecoveryKindFatal, // "amqp:not-allowed"
	amqp.ErrCond("com.microsoft:entity-disabled"):          RecoveryKindFatal, // entity is disabled in the portal
	amqp.ErrCond("com.microsoft:session-cannot-be-locked"): RecoveryKindFatal,
	amqp.ErrCond("com.microsoft:argument-out-of-range"):    RecoveryKindFatal, // asked for a partition ID that doesn't exist
	errorConditionLockLost:                                 RecoveryKindFatal,
	eh.ErrCondGeoReplicationOffset:                         RecoveryKindFatal,
}

// GetRecoveryKind determines the recovery type for non-session based links.
func GetRecoveryKind(err error) RecoveryKind {
	if err == nil {
		return RecoveryKindNone
	}

	if errors.Is(err, ErrRPCLinkClosed) {
		return RecoveryKindFatal
	}

	if IsCancelError(err) {
		return RecoveryKindFatal
	}

	if errors.Is(err, amqpwrap.ErrConnResetNeeded) {
		return RecoveryKindConn
	}

	var netErr net.Error

	// these are errors that can flow from the go-amqp connection to
	// us. There's work underway to improve this but for now we can handle
	// these as "catastrophic" errors and reset everything.
	if errors.Is(err, io.EOF) || errors.As(err, &netErr) {
		return RecoveryKindConn
	}

	var errNonRetriable errNonRetriable

	if errors.As(err, &errNonRetriable) {
		return RecoveryKindFatal
	}

	// azidentity returns errors that match this for auth failures.
	var errNonRetriableMarker interface {
		NonRetriable()
		error
	}

	if errors.As(err, &errNonRetriableMarker) {
		return RecoveryKindFatal
	}

	if IsOwnershipLostError(err) {
		return RecoveryKindFatal
	}

	// check the "special" AMQP errors that aren't condition-based.
	if IsQuickRecoveryError(err) {
		return RecoveryKindLink
	}

	var connErr *amqp.ConnError
	var sessionErr *amqp.SessionError

	if errors.As(err, &connErr) ||
		// session closures appear to leak through when the connection itself is going down.
		errors.As(err, &sessionErr) {
		return RecoveryKindConn
	}

	// then it's _probably_ an actual *amqp.Error, in which case we bucket it by
	// the 'condition'.
	var amqpError *amqp.Error

	if errors.As(err, &amqpError) {
		recoveryKind, ok := amqpConditionsToRecoveryKind[amqpError.Condition]

		if ok {
			return recoveryKind
		}
	}

	var rpcErr RPCError

	if errors.As(err, &rpcErr) {
		// Described more here:
		// https://www.oasis-open.org/committees/download.php/54441/AMQP%20Management%20v1.0%20WD09
		// > Unsuccessful operations MUST NOT result in a statusCode in the 2xx range as defined in Section 10.2 of [RFC2616]
		// RFC2616 is the specification for HTTP.
		code := rpcErr.RPCCode()

		if code == http.StatusNotFound ||
			code == http.StatusUnauthorized {
			return RecoveryKindFatal
		}

		// simple timeouts
		if rpcErr.Resp.Code == http.StatusRequestTimeout || rpcErr.Resp.Code == http.StatusServiceUnavailable ||
			// internal server errors are worth retrying (they will typically lead
			// to a more actionable error). A simple example of this is when you're
			// in the middle of an operation and the link is detached. Sometimes you'll get
			// the detached event immediately, but sometimes you'll get an intermediate 500
			// indicating your original operation was cancelled.
			rpcErr.Resp.Code == http.StatusInternalServerError {
			return RecoveryKindNone
		}
	}

	// this is some error type we've never seen - recover the entire connection.
	return RecoveryKindConn
}

func IsNotAllowedError(err error) bool {
	var e *amqp.Error

	return errors.As(err, &e) &&
		e.Condition == amqp.ErrCondNotAllowed
}

func IsOwnershipLostError(err error) bool {
	var de *amqp.LinkError

	if errors.As(err, &de) {
		return de.RemoteErr != nil && de.RemoteErr.Condition == "amqp:link:stolen"
	}

	return false
}
