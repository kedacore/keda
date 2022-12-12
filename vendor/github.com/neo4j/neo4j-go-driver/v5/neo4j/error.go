/*
 * Copyright (c) "Neo4j"
 * Neo4j Sweden AB [https://neo4j.com]
 *
 * This file is part of Neo4j.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package neo4j

import (
	"context"
	"errors"
	"fmt"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/db"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/bolt"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/connector"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/errorutil"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/pool"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/retry"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/router"
	"io"
	"net"
)

// IsRetryable determines whether an operation can be retried based on the error
// it triggered. This API is meant for use in scenarios where users want to
// implement their own retry mechanism.
// A similar logic is used by the driver for transaction functions.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	var connectivityErr *ConnectivityError
	var commitFailedError *retry.CommitFailedDeadError
	if errors.As(err, &connectivityErr) && !errors.As(connectivityErr.inner, &commitFailedError) {
		// all connectivity errors are safe to retry except during transaction commit
		return true
	}
	return retry.IsRetryable(err)
}

// Neo4jError represents errors originating from Neo4j service.
// Alias for convenience. This error is defined in db package and
// used internally.
type Neo4jError = db.Neo4jError

// UsageError represents errors caused by incorrect usage of the driver API.
// This does not include Cypher syntax (those errors will be Neo4jError).
type UsageError struct {
	Message string
}

func (e *UsageError) Error() string {
	return e.Message
}

// TransactionExecutionLimit error indicates that a retryable transaction has
// failed due to reaching a limit like a timeout or maximum number of attempts.
type TransactionExecutionLimit struct {
	Errors []error
	Causes []string
}

func newTransactionExecutionLimit(errors []error, causes []string) *TransactionExecutionLimit {
	tel := &TransactionExecutionLimit{Errors: make([]error, len(errors)), Causes: causes}
	for i, err := range errors {
		tel.Errors[i] = wrapError(err)
	}

	return tel
}

func (e *TransactionExecutionLimit) Error() string {
	cause := "Unknown cause"
	l := len(e.Causes)
	if l > 0 {
		cause = e.Causes[l-1]
	}
	var err error
	l = len(e.Errors)
	if l > 0 {
		err = e.Errors[l-1]
	}
	return fmt.Sprintf("TransactionExecutionLimit: %s after %d attempts, last error: %s", cause, len(e.Errors), err)
}

// ConnectivityError represent errors caused by the driver not being able to connect to Neo4j services,
// or lost connections.
type ConnectivityError struct {
	inner error
}

func (e *ConnectivityError) Error() string {
	return fmt.Sprintf("ConnectivityError: %s", e.inner.Error())
}

// IsNeo4jError returns true if the provided error is an instance of Neo4jError.
func IsNeo4jError(err error) bool {
	_, is := err.(*Neo4jError)
	return is
}

// IsUsageError returns true if the provided error is an instance of UsageError.
func IsUsageError(err error) bool {
	_, is := err.(*UsageError)
	return is
}

// IsConnectivityError returns true if the provided error is an instance of ConnectivityError.
func IsConnectivityError(err error) bool {
	_, is := err.(*ConnectivityError)
	return is
}

// IsTransactionExecutionLimit returns true if the provided error is an instance of TransactionExecutionLimit.
func IsTransactionExecutionLimit(err error) bool {
	_, is := err.(*TransactionExecutionLimit)
	return is
}

// TokenExpiredError represent errors caused by the driver not being able to connect to Neo4j services,
// or lost connections.
type TokenExpiredError struct {
	Code    string
	Message string
}

func (e *TokenExpiredError) Error() string {
	return fmt.Sprintf("TokenExpiredError: %s (%s)", e.Code, e.Message)
}

func wrapError(err error) error {
	if err == nil {
		return nil
	}
	if err == io.EOF {
		return &ConnectivityError{inner: err}
	}
	switch e := err.(type) {
	case *db.UnsupportedTypeError, *db.FeatureNotSupportedError:
		// Usage of a type not supported by database network protocol or feature
		// not supported by current version or edition.
		return &UsageError{Message: err.Error()}
	case *pool.PoolClosed:
		return &UsageError{Message: err.Error()}
	case *connector.TlsError, net.Error:
		return &ConnectivityError{inner: err}
	case *pool.PoolTimeout, *pool.PoolFull:
		return &ConnectivityError{inner: err}
	case *router.ReadRoutingTableError:
		return &ConnectivityError{inner: err}
	case *retry.CommitFailedDeadError:
		return &ConnectivityError{inner: err}
	case *bolt.ConnectionReadTimeout:
		return &ConnectivityError{inner: err}
	case *bolt.ConnectionWriteTimeout:
		return &ConnectivityError{inner: err}
	case *db.Neo4jError:
		if e.Code == "Neo.ClientError.Security.TokenExpired" {
			return &TokenExpiredError{Code: e.Code, Message: e.Msg}
		}
	}
	if err != nil && err.Error() == bolt.InvalidTransactionError {
		return &UsageError{Message: bolt.InvalidTransactionError}
	}
	return err
}

type ctxCloser interface {
	Close(ctx context.Context) error
}

func deferredClose(ctx context.Context, closer ctxCloser, prevErr error) error {
	return errorutil.CombineErrors(prevErr, closer.Close(ctx))
}
