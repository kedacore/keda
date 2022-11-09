// Copyright (C) 2017 Kale Blankenship
// Portions Copyright (c) Microsoft Corporation
package amqp

import (
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/go-amqp/internal/encoding"
)

// Error Conditions
const (
	// AMQP Errors
	ErrorInternalError         ErrorCondition = "amqp:internal-error"
	ErrorNotFound              ErrorCondition = "amqp:not-found"
	ErrorUnauthorizedAccess    ErrorCondition = "amqp:unauthorized-access"
	ErrorDecodeError           ErrorCondition = "amqp:decode-error"
	ErrorResourceLimitExceeded ErrorCondition = "amqp:resource-limit-exceeded"
	ErrorNotAllowed            ErrorCondition = "amqp:not-allowed"
	ErrorInvalidField          ErrorCondition = "amqp:invalid-field"
	ErrorNotImplemented        ErrorCondition = "amqp:not-implemented"
	ErrorResourceLocked        ErrorCondition = "amqp:resource-locked"
	ErrorPreconditionFailed    ErrorCondition = "amqp:precondition-failed"
	ErrorResourceDeleted       ErrorCondition = "amqp:resource-deleted"
	ErrorIllegalState          ErrorCondition = "amqp:illegal-state"
	ErrorFrameSizeTooSmall     ErrorCondition = "amqp:frame-size-too-small"

	// Connection Errors
	ErrorConnectionForced   ErrorCondition = "amqp:connection:forced"
	ErrorFramingError       ErrorCondition = "amqp:connection:framing-error"
	ErrorConnectionRedirect ErrorCondition = "amqp:connection:redirect"

	// Session Errors
	ErrorWindowViolation  ErrorCondition = "amqp:session:window-violation"
	ErrorErrantLink       ErrorCondition = "amqp:session:errant-link"
	ErrorHandleInUse      ErrorCondition = "amqp:session:handle-in-use"
	ErrorUnattachedHandle ErrorCondition = "amqp:session:unattached-handle"

	// Link Errors
	ErrorDetachForced          ErrorCondition = "amqp:link:detach-forced"
	ErrorTransferLimitExceeded ErrorCondition = "amqp:link:transfer-limit-exceeded"
	ErrorMessageSizeExceeded   ErrorCondition = "amqp:link:message-size-exceeded"
	ErrorLinkRedirect          ErrorCondition = "amqp:link:redirect"
	ErrorStolen                ErrorCondition = "amqp:link:stolen"
)

type Error = encoding.Error

type ErrorCondition = encoding.ErrorCondition

// DetachError is returned by a link (Receiver/Sender) when a detach frame is received.
//
// RemoteError will be nil if the link was detached gracefully.
type DetachError struct {
	RemoteError *Error
}

func (e *DetachError) Error() string {
	return fmt.Sprintf("link detached, reason: %+v", e.RemoteError)
}

// Errors
var (
	// ErrSessionClosed is propagated to Sender/Receivers
	// when Session.Close() is called.
	ErrSessionClosed = errors.New("amqp: session closed")

	// ErrLinkClosed is returned by send and receive operations when
	// Sender.Close() or Receiver.Close() are called.
	ErrLinkClosed = errors.New("amqp: link closed")
)

// ConnectionError is propagated to Session and Senders/Receivers
// when the connection has been closed or is no longer functional.
type ConnectionError struct {
	inner error
}

func (c *ConnectionError) Error() string {
	if c.inner == nil {
		return "amqp: connection closed"
	}
	return c.inner.Error()
}
