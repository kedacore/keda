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
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package neo4j

import (
	"context"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/db"
)

// ManagedTransaction represents a transaction managed by the driver and operated on by the user, via transaction functions
type ManagedTransaction interface {
	// Run executes a statement on this transaction and returns a result
	Run(ctx context.Context, cypher string, params map[string]any) (ResultWithContext, error)

	legacy() Transaction
}

// ExplicitTransaction represents a transaction in the Neo4j database
type ExplicitTransaction interface {
	// Run executes a statement on this transaction and returns a result
	Run(ctx context.Context, cypher string, params map[string]any) (ResultWithContext, error)
	// Commit commits the transaction
	Commit(ctx context.Context) error
	// Rollback rolls back the transaction
	Rollback(ctx context.Context) error
	// Close rolls back the actual transaction if it's not already committed/rolled back
	// and closes all resources associated with this transaction
	Close(ctx context.Context) error

	// legacy returns the non-cancelling, legacy variant of this ExplicitTransaction type
	// This is used so that legacy transaction functions can delegate work to their newer, context-aware variants
	legacy() Transaction
}

// Transaction implementation when explicit transaction started
type explicitTransaction struct {
	conn      db.Connection
	fetchSize int
	txHandle  db.TxHandle
	done      bool
	runFailed bool
	err       error
	onClosed  func(*explicitTransaction)
}

func (tx *explicitTransaction) Run(ctx context.Context, cypher string,
	params map[string]any) (ResultWithContext, error) {
	stream, err := tx.conn.RunTx(ctx, tx.txHandle, db.Command{Cypher: cypher, Params: params, FetchSize: tx.fetchSize})
	if err != nil {
		tx.err = err
		tx.runFailed = true
		tx.onClosed(tx)
		return nil, wrapError(tx.err)
	}
	// no result consumption hook here since bookmarks are sent after commit, not after pulling results
	return newResultWithContext(tx.conn, stream, cypher, params, nil), nil
}

func (tx *explicitTransaction) Commit(ctx context.Context) error {
	if tx.runFailed {
		tx.runFailed, tx.done = false, true
		return tx.err
	}
	if tx.done {
		return transactionAlreadyCompletedError()
	}
	tx.err = tx.conn.TxCommit(ctx, tx.txHandle)
	tx.done = true
	tx.onClosed(tx)
	return wrapError(tx.err)
}

func (tx *explicitTransaction) Close(ctx context.Context) error {
	if tx.done {
		// repeated calls to Close => NOOP
		return nil
	}
	return tx.Rollback(ctx)
}

func (tx *explicitTransaction) Rollback(ctx context.Context) error {
	if tx.runFailed {
		tx.done, tx.runFailed = true, false
		return nil
	}
	if tx.done {
		return transactionAlreadyCompletedError()
	}
	if !tx.conn.IsAlive() || tx.conn.HasFailed() {
		// tx implicitly rolled back by having failed
		tx.err = nil
	} else {
		tx.err = tx.conn.TxRollback(ctx, tx.txHandle)
	}
	tx.done = true
	tx.onClosed(tx)
	return wrapError(tx.err)
}

func (tx *explicitTransaction) legacy() Transaction {
	return &transaction{
		delegate: tx,
	}
}

// ManagedTransaction implementation used as parameter to transactional functions
type managedTransaction struct {
	conn      db.Connection
	fetchSize int
	txHandle  db.TxHandle
}

func (tx *managedTransaction) Run(ctx context.Context, cypher string, params map[string]any) (ResultWithContext, error) {
	stream, err := tx.conn.RunTx(ctx, tx.txHandle, db.Command{Cypher: cypher, Params: params, FetchSize: tx.fetchSize})
	if err != nil {
		return nil, wrapError(err)
	}
	// no result consumption hook here since bookmarks are sent after commit, not after pulling results
	return newResultWithContext(tx.conn, stream, cypher, params, nil), nil
}

// legacy interop only - remove in 6.0
func (tx *managedTransaction) Commit(context.Context) error {
	return &UsageError{Message: "Commit not allowed on retryable transaction"}
}

// legacy interop only - remove in 6.0
func (tx *managedTransaction) Rollback(context.Context) error {
	return &UsageError{Message: "Rollback not allowed on retryable transaction"}
}

// legacy interop only - remove in 6.0
func (tx *managedTransaction) Close(context.Context) error {
	return &UsageError{Message: "Close not allowed on retryable transaction"}
}

// legacy interop only - remove in 6.0
func (tx *managedTransaction) legacy() Transaction {
	return &transaction{
		delegate: tx,
	}
}

// Represents an auto commit transaction.
// Does not implement the ExplicitTransaction nor the ManagedTransaction interface.
type autocommitTransaction struct {
	conn     db.Connection
	res      ResultWithContext
	closed   bool
	onClosed func()
}

func (tx *autocommitTransaction) done(ctx context.Context) {
	if !tx.closed {
		tx.res.buffer(ctx)
		tx.closed = true
		tx.onClosed()
	}
}

func (tx *autocommitTransaction) discard(ctx context.Context) {
	if !tx.closed {
		tx.res.Consume(ctx)
		tx.closed = true
		tx.onClosed()
	}
}

func transactionAlreadyCompletedError() *UsageError {
	return &UsageError{Message: "commit or rollback already called once on this transaction"}
}
