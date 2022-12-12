package bolt

import (
	"context"
	"fmt"
	"time"
)

const InvalidTransactionError = "invalid transaction handle"

type ConnectionReadTimeout struct {
	userContext context.Context
	readTimeout time.Duration
	err         error
}

func (crt *ConnectionReadTimeout) Error() string {
	userDeadline := "N/A"
	if deadline, ok := crt.userContext.Deadline(); ok {
		userDeadline = deadline.String()
	}
	return fmt.Sprintf(
		"Timeout while reading from connection [server-side timeout hint: %s, user-provided context deadline: %s]: %s",
		crt.readTimeout.String(),
		userDeadline,
		crt.err)
}

type ConnectionWriteTimeout struct {
	userContext context.Context
	err         error
}

func (cwt *ConnectionWriteTimeout) Error() string {
	userDeadline := "N/A"
	if deadline, ok := cwt.userContext.Deadline(); ok {
		userDeadline = deadline.String()
	}
	return fmt.Sprintf("Timeout while writing to connection [user-provided context deadline: %s]: %s", userDeadline, cwt.err)
}

type ConnectionReadCanceled struct {
	err error
}

func (crc *ConnectionReadCanceled) Error() string {
	return fmt.Sprintf("Reading from connection has been canceled: %s", crc.err)
}

type ConnectionWriteCanceled struct {
	err error
}

func (cwc *ConnectionWriteCanceled) Error() string {
	return fmt.Sprintf("Writing to connection has been canceled: %s", cwc.err)
}

type timeout interface {
	Timeout() bool
}

func IsTimeoutError(err error) bool {
	if err == context.DeadlineExceeded {
		return true
	}
	timeoutErr, ok := err.(timeout)
	return ok && timeoutErr.Timeout()
}
