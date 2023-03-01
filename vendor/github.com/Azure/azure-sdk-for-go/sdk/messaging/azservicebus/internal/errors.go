// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package internal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/amqpwrap"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/exported"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/go-amqp"
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

func IsFatalSBError(err error) bool {
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

	if isLockLostError(err) {
		return exported.NewError(exported.CodeLockLost, err)
	}

	if isMicrosoftTimeoutError(err) {
		// one scenario where this error pops up is if you're waiting for an available
		// session and there are none available. It waits, up to one minute, and then
		// returns this error.
		return exported.NewError(exported.CodeTimeout, err)
	}

	rk := GetRecoveryKind(err)

	switch rk {
	case RecoveryKindLink:
		// note that we could give back a more differentiated error code
		// here but it's probably best to just give the customer the simplest
		// recovery mechanism possible.
		return exported.NewError(exported.CodeConnectionLost, err)
	case RecoveryKindConn:
		return exported.NewError(exported.CodeConnectionLost, err)
	default:
		// isn't one of our specifically called out cases so we'll just return it.
		return err
	}
}

func isMicrosoftTimeoutError(err error) bool {
	var amqpErr *amqp.Error

	if errors.As(err, &amqpErr) && amqpErr.Condition == amqp.ErrorCondition("com.microsoft:timeout") {
		return true
	}

	return false
}

func IsDetachError(err error) bool {
	var de *amqp.DetachError
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

func IsDrainingError(err error) bool {
	// TODO: we should be able to identify these errors programatically
	return strings.Contains(err.Error(), "link is currently draining")
}

const errorConditionLockLost = amqp.ErrorCondition("com.microsoft:message-lock-lost")

var amqpConditionsToRecoveryKind = map[amqp.ErrorCondition]RecoveryKind{
	// no recovery needed, these are temporary errors.
	amqp.ErrorCondition("com.microsoft:server-busy"):         RecoveryKindNone,
	amqp.ErrorCondition("com.microsoft:timeout"):             RecoveryKindNone,
	amqp.ErrorCondition("com.microsoft:operation-cancelled"): RecoveryKindNone,

	// Link recovery needed
	amqp.ErrorDetachForced:          RecoveryKindLink, // "amqp:link:detach-forced"
	amqp.ErrorTransferLimitExceeded: RecoveryKindLink, // "amqp:link:transfer-limit-exceeded"

	// Connection recovery needed
	amqp.ErrorConnectionForced: RecoveryKindConn, // "amqp:connection:forced"
	amqp.ErrorInternalError:    RecoveryKindConn, // "amqp:internal-error"

	// No recovery possible - this operation is non retriable.
	amqp.ErrorMessageSizeExceeded:                                 RecoveryKindFatal, // "amqp:link:message-size-exceeded"
	amqp.ErrorUnauthorizedAccess:                                  RecoveryKindFatal, // creds are bad
	amqp.ErrorNotFound:                                            RecoveryKindFatal, // "amqp:not-found"
	amqp.ErrorNotAllowed:                                          RecoveryKindFatal, // "amqp:not-allowed"
	amqp.ErrorCondition("com.microsoft:entity-disabled"):          RecoveryKindFatal, // entity is disabled in the portal
	amqp.ErrorCondition("com.microsoft:session-cannot-be-locked"): RecoveryKindFatal,
	errorConditionLockLost:                                        RecoveryKindFatal,
}

// GetRecoveryKindForSession determines the recovery type for session-based links.
func GetRecoveryKindForSession(err error) RecoveryKind {
	// when a session is detached there's a delay before we can reacquire the
	// lock. So a lock lost on a session _is_ retryable.
	if isLockLostError(err) {
		return RecoveryKindLink
	}

	return GetRecoveryKind(err)
}

// GetRecoveryKind determines the recovery type for non-session based links.
func GetRecoveryKind(err error) RecoveryKind {
	if err == nil {
		return RecoveryKindNone
	}

	if IsCancelError(err) {
		return RecoveryKindFatal
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

	// check the "special" AMQP errors that aren't condition-based.
	if errors.Is(err, amqp.ErrLinkClosed) ||
		IsDetachError(err) {
		return RecoveryKindLink
	}

	var connErr *amqp.ConnectionError

	if errors.As(err, &connErr) ||
		// session closures appear to leak through when the connection itself is going down.
		errors.Is(err, amqp.ErrSessionClosed) {
		return RecoveryKindConn
	}

	if IsDrainingError(err) {
		// temporary, operation should just be retryable since drain will
		// eventually complete.
		return RecoveryKindNone
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

		if code == http.StatusNotFound || code == RPCResponseCodeLockLost {
			return RecoveryKindFatal
		}

		// this can happen when we're recovering the link - the client gets closed and the old link is still being
		// used by this instance of the client. It needs to recover and attempt it again.
		if code == http.StatusUnauthorized {
			return RecoveryKindLink
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

type (
	// ErrMissingField indicates that an expected property was missing from an AMQP message. This should only be
	// encountered when there is an error with this library, or the server has altered its behavior unexpectedly.
	ErrMissingField string

	// ErrMalformedMessage indicates that a message was expected in the form of []byte was not a []byte. This is likely
	// a bug and should be reported.
	ErrMalformedMessage string

	// ErrIncorrectType indicates that type assertion failed. This should only be encountered when there is an error
	// with this library, or the server has altered its behavior unexpectedly.
	ErrIncorrectType struct {
		Key          string
		ExpectedType reflect.Type
		ActualValue  interface{}
	}

	// ErrAMQP indicates that the server communicated an AMQP error with a particular
	ErrAMQP amqpwrap.RPCResponse

	// ErrNoMessages is returned when an operation returned no messages. It is not indicative that there will not be
	// more messages in the future.
	ErrNoMessages struct{}

	// ErrNotFound is returned when an entity is not found (404)
	ErrNotFound struct {
		EntityPath string
	}

	// ErrConnectionClosed indicates that the connection has been closed.
	ErrConnectionClosed string
)

func (e ErrMissingField) Error() string {
	return fmt.Sprintf("missing value %q", string(e))
}

func (e ErrMalformedMessage) Error() string {
	return "message was expected in the form of []byte was not a []byte"
}

// NewErrIncorrectType lets you skip using the `reflect` package. Just provide a variable of the desired type as
// 'expected'.
func NewErrIncorrectType(key string, expected, actual interface{}) ErrIncorrectType {
	return ErrIncorrectType{
		Key:          key,
		ExpectedType: reflect.TypeOf(expected),
		ActualValue:  actual,
	}
}

func (e ErrIncorrectType) Error() string {
	return fmt.Sprintf(
		"value at %q was expected to be of type %q but was actually of type %q",
		e.Key,
		e.ExpectedType,
		reflect.TypeOf(e.ActualValue))
}

func (e ErrAMQP) Error() string {
	return fmt.Sprintf("server says (%d) %s", e.Code, e.Description)
}

func (e ErrNoMessages) Error() string {
	return "no messages available"
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("entity at %s not found", e.EntityPath)
}

// IsErrNotFound returns true if the error argument is an ErrNotFound type
func IsErrNotFound(err error) bool {
	_, ok := err.(ErrNotFound)
	return ok
}

func (e ErrConnectionClosed) Error() string {
	return fmt.Sprintf("the connection has been closed: %s", string(e))
}

func isLockLostError(err error) bool {
	var rpcErr RPCError

	// this is the error you get if you settle on the management$ link
	// with an expired locktoken.
	if errors.As(err, &rpcErr) && rpcErr.Resp.Code == RPCResponseCodeLockLost {
		return true
	}

	var amqpErr *amqp.Error

	// this is the error you get if you settle on the actual receiver link you
	// got the message on with an expired locktoken.
	if errors.As(err, &amqpErr) && amqpErr.Condition == errorConditionLockLost {
		return true
	}

	return false
}

var errConnResetNeeded = errors.New("connection must be reset, link/connection state may be inconsistent")
