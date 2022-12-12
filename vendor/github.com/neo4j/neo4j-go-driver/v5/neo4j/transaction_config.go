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

import "time"

// TransactionConfig holds the settings for explicit and auto-commit transactions. Actual configuration is expected
// to be done using configuration functions that are predefined, i.e. 'WithTxTimeout' and 'WithTxMetadata', or one
// that you could write by your own.
type TransactionConfig struct {
	// Timeout is the configured transaction timeout.
	Timeout time.Duration
	// Metadata is the configured transaction metadata that will be attached to the underlying transaction.
	Metadata map[string]any
}

// WithTxTimeout returns a transaction configuration function that applies a timeout to a transaction.
//
// To apply a transaction timeout to an explicit transaction:
//	session.BeginTransaction(WithTxTimeout(5*time.Second))
//
// To apply a transaction timeout to an auto-commit transaction:
//	session.Run("RETURN 1", nil, WithTxTimeout(5*time.Second))
//
// To apply a transaction timeout to a read transaction function:
//	session.ExecuteRead(DoWork, WithTxTimeout(5*time.Second))
//
// To apply a transaction timeout to a write transaction function:
//	session.ExecuteWrite(DoWork, WithTxTimeout(5*time.Second))
func WithTxTimeout(timeout time.Duration) func(*TransactionConfig) {
	return func(config *TransactionConfig) {
		config.Timeout = timeout
	}
}

// WithTxMetadata returns a transaction configuration function that attaches metadata to a transaction.
//
// To attach a metadata to an explicit transaction:
//	session.BeginTransaction(WithTxMetadata(map[string)any{"work-id": 1}))
//
// To attach a metadata to an auto-commit transaction:
//	session.Run("RETURN 1", nil, WithTxMetadata(map[string)any{"work-id": 1}))
//
// To attach a metadata to a read transaction function:
//	session.ExecuteRead(DoWork, WithTxMetadata(map[string)any{"work-id": 1}))
//
// To attach a metadata to a write transaction function:
//	session.ExecuteWrite(DoWork, WithTxMetadata(map[string)any{"work-id": 1}))
func WithTxMetadata(metadata map[string]any) func(*TransactionConfig) {
	return func(config *TransactionConfig) {
		config.Metadata = metadata
	}
}
