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

// Session represents a logical connection (which is not tied to a physical connection)
// to the server
// Deprecated: use SessionWithContext instead.
// SessionWithContext is created via the context-aware driver returned
// by NewDriverWithContext.
// Session will be removed in 6.0.
type Session interface {
	// LastBookmarks returns the bookmark received following the last successfully completed transaction.
	// If no bookmark was received or if this transaction was rolled back, the initial set of bookmarks will be
	// returned.
	LastBookmarks() Bookmarks
	// LastBookmark returns the bookmark received following the last successfully completed transaction.
	// If no bookmark was received or if this transaction was rolled back, the bookmark value will not be changed.
	// Deprecated: since version 5.0. Will be removed in 6.0. Use LastBookmarks instead.
	// Warning: this method can lead to unexpected behaviour if the session has not yet successfully completed a
	// transaction.
	LastBookmark() string
	// BeginTransaction starts a new explicit transaction on this session
	BeginTransaction(configurers ...func(*TransactionConfig)) (Transaction, error)
	// ReadTransaction executes the given unit of work in a AccessModeRead transaction with
	// retry logic in place
	ReadTransaction(work TransactionWork, configurers ...func(*TransactionConfig)) (any, error)
	// WriteTransaction executes the given unit of work in a AccessModeWrite transaction with
	// retry logic in place
	WriteTransaction(work TransactionWork, configurers ...func(*TransactionConfig)) (any, error)
	// Run executes an auto-commit statement and returns a result
	Run(cypher string, params map[string]any, configurers ...func(*TransactionConfig)) (Result, error)
	// Close closes any open resources and marks this session as unusable
	Close() error
}

type session struct {
	delegate *sessionWithContext
}

func (s *session) LastBookmarks() Bookmarks {
	return s.delegate.LastBookmarks()
}

func (s *session) LastBookmark() string {
	return s.delegate.lastBookmark()
}

func (s *session) BeginTransaction(configurers ...func(*TransactionConfig)) (Transaction, error) {
	tx, err := s.delegate.BeginTransaction(context.Background(), configurers...)
	if err != nil {
		return nil, err
	}
	return tx.legacy(), nil
}

func (s *session) ReadTransaction(
	work TransactionWork, configurers ...func(*TransactionConfig)) (any, error) {

	return s.delegate.ExecuteRead(
		context.Background(),
		transactionWorkBridge(work),
		configurers...,
	)
}

func (s *session) WriteTransaction(
	work TransactionWork, configurers ...func(*TransactionConfig)) (any, error) {

	return s.delegate.ExecuteWrite(
		context.Background(),
		transactionWorkBridge(work),
		configurers...,
	)
}

func (s *session) Run(
	cypher string, params map[string]any, configurers ...func(*TransactionConfig)) (Result, error) {

	result, err := s.delegate.Run(context.Background(), cypher, params, configurers...)
	if err != nil {
		return nil, err
	}
	return result.legacy(), nil
}

func (s *session) Close() error {
	return s.delegate.Close(context.Background())
}

func transactionWorkBridge(work TransactionWork) ManagedTransactionWork {
	return func(txc ManagedTransaction) (any, error) {
		return work(txc.legacy())
	}
}

type erroredSession struct {
	err error
}

func (s *erroredSession) LastBookmark() string {
	return ""
}

func (s *erroredSession) LastBookmarks() Bookmarks {
	return []string{}
}

func (s *erroredSession) BeginTransaction(...func(*TransactionConfig)) (Transaction, error) {
	return nil, s.err
}
func (s *erroredSession) ReadTransaction(TransactionWork, ...func(*TransactionConfig)) (any, error) {
	return nil, s.err
}
func (s *erroredSession) WriteTransaction(TransactionWork, ...func(*TransactionConfig)) (any, error) {
	return nil, s.err
}
func (s *erroredSession) Run(string, map[string]any, ...func(*TransactionConfig)) (Result, error) {
	return nil, s.err
}
func (s *erroredSession) Close() error {
	return s.err
}
