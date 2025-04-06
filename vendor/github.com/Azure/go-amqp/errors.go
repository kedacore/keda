package amqp

import (
	"github.com/Azure/go-amqp/internal/encoding"
)

// ErrCond is an AMQP defined error condition.
// See http://docs.oasis-open.org/amqp/core/v1.0/os/amqp-core-transport-v1.0-os.html#type-amqp-error for info on their meaning.
type ErrCond = encoding.ErrCond

// Error Conditions
const (
	// AMQP Errors
	ErrCondDecodeError           ErrCond = "amqp:decode-error"
	ErrCondFrameSizeTooSmall     ErrCond = "amqp:frame-size-too-small"
	ErrCondIllegalState          ErrCond = "amqp:illegal-state"
	ErrCondInternalError         ErrCond = "amqp:internal-error"
	ErrCondInvalidField          ErrCond = "amqp:invalid-field"
	ErrCondNotAllowed            ErrCond = "amqp:not-allowed"
	ErrCondNotFound              ErrCond = "amqp:not-found"
	ErrCondNotImplemented        ErrCond = "amqp:not-implemented"
	ErrCondPreconditionFailed    ErrCond = "amqp:precondition-failed"
	ErrCondResourceDeleted       ErrCond = "amqp:resource-deleted"
	ErrCondResourceLimitExceeded ErrCond = "amqp:resource-limit-exceeded"
	ErrCondResourceLocked        ErrCond = "amqp:resource-locked"
	ErrCondUnauthorizedAccess    ErrCond = "amqp:unauthorized-access"

	// Connection Errors
	ErrCondConnectionForced   ErrCond = "amqp:connection:forced"
	ErrCondConnectionRedirect ErrCond = "amqp:connection:redirect"
	ErrCondFramingError       ErrCond = "amqp:connection:framing-error"

	// Session Errors
	ErrCondErrantLink       ErrCond = "amqp:session:errant-link"
	ErrCondHandleInUse      ErrCond = "amqp:session:handle-in-use"
	ErrCondUnattachedHandle ErrCond = "amqp:session:unattached-handle"
	ErrCondWindowViolation  ErrCond = "amqp:session:window-violation"

	// Link Errors
	ErrCondDetachForced          ErrCond = "amqp:link:detach-forced"
	ErrCondLinkRedirect          ErrCond = "amqp:link:redirect"
	ErrCondMessageSizeExceeded   ErrCond = "amqp:link:message-size-exceeded"
	ErrCondStolen                ErrCond = "amqp:link:stolen"
	ErrCondTransferLimitExceeded ErrCond = "amqp:link:transfer-limit-exceeded"
)

// Error is an AMQP error.
type Error = encoding.Error

// LinkError is returned by methods on Sender/Receiver when the link has closed.
type LinkError struct {
	// RemoteErr contains any error information provided by the peer if the peer detached the link.
	RemoteErr *Error

	inner error
}

// Error implements the error interface for LinkError.
func (e *LinkError) Error() string {
	if e.RemoteErr == nil && e.inner == nil {
		return "amqp: link closed"
	} else if e.RemoteErr != nil {
		return e.RemoteErr.Error()
	}
	return e.inner.Error()
}

// Unwrap returns the RemoteErr, if any.
func (e *LinkError) Unwrap() error {
	if e.RemoteErr == nil {
		return nil
	}

	return e.RemoteErr
}

// ConnError is returned by methods on Conn and propagated to Session and Senders/Receivers
// when the connection has been closed.
type ConnError struct {
	// RemoteErr contains any error information provided by the peer if the peer closed the AMQP connection.
	RemoteErr *Error

	inner error
}

// Error implements the error interface for ConnError.
func (e *ConnError) Error() string {
	if e.RemoteErr == nil && e.inner == nil {
		return "amqp: connection closed"
	} else if e.RemoteErr != nil {
		return e.RemoteErr.Error()
	}
	return e.inner.Error()
}

// Unwrap returns the RemoteErr, if any.
func (e *ConnError) Unwrap() error {
	if e.RemoteErr == nil {
		return nil
	}

	return e.RemoteErr
}

// SessionError is returned by methods on Session and propagated to Senders/Receivers
// when the session has been closed.
type SessionError struct {
	// RemoteErr contains any error information provided by the peer if the peer closed the session.
	RemoteErr *Error

	inner error
}

// Error implements the error interface for SessionError.
func (e *SessionError) Error() string {
	if e.RemoteErr == nil && e.inner == nil {
		return "amqp: session closed"
	} else if e.RemoteErr != nil {
		return e.RemoteErr.Error()
	}
	return e.inner.Error()
}

// Unwrap returns the RemoteErr, if any.
func (e *SessionError) Unwrap() error {
	if e.RemoteErr == nil {
		return nil
	}

	return e.RemoteErr
}
