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
)

// Transaction represents a transaction in the Neo4j database
// Deprecated: use ExplicitTransaction instead.
// ExplicitTransaction is available via SessionWithContext.
// SessionWithContext is available via the context-aware driver/returned
// by NewDriverWithContext.
// Transaction will be removed in 6.0.
type Transaction interface {
	// Run executes a statement on this transaction and returns a result
	Run(cypher string, params map[string]any) (Result, error)
	// Commit commits the transaction
	Commit() error
	// Rollback rolls back the transaction
	Rollback() error
	// Close rolls back the actual transaction if it's not already committed/rolled back
	// and closes all resources associated with this transaction
	Close() error
}

// Transaction implementation when explicit transaction started
type transaction struct {
	delegate ExplicitTransaction
}

func (tx *transaction) Run(cypher string, params map[string]any) (Result, error) {
	result, err := tx.delegate.Run(context.Background(), cypher, params)
	if err != nil {
		return nil, err
	}
	return result.legacy(), nil
}

func (tx *transaction) Commit() error {
	return tx.delegate.Commit(context.Background())
}

func (tx *transaction) Rollback() error {
	return tx.delegate.Rollback(context.Background())
}

func (tx *transaction) Close() error {
	return tx.delegate.Close(context.Background())
}
