//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//
// Author Lars Maier
//

package driver

import (
	"context"
	"time"
)

// BeginTransactionOptions provides options for BeginTransaction call
type BeginTransactionOptions struct {
	WaitForSync        bool
	AllowImplicit      bool
	LockTimeout        time.Duration
	MaxTransactionSize uint64
}

// TransactionCollections is used to specify which collections are accessed by
// a transaction and how
type TransactionCollections struct {
	Read      []string `json:"read,omitempty"`
	Write     []string `json:"write,omitempty"`
	Exclusive []string `json:"exclusive,omitempty"`
}

// CommitTransactionOptions provides options for CommitTransaction. Currently unused
type CommitTransactionOptions struct{}

// AbortTransactionOptions provides options for CommitTransaction. Currently unused
type AbortTransactionOptions struct{}

// TransactionID identifies a transaction
type TransactionID string

// TransactionStatus describes the status of an transaction
type TransactionStatus string

const (
	TransactionRunning   TransactionStatus = "running"
	TransactionCommitted TransactionStatus = "committed"
	TransactionAborted   TransactionStatus = "aborted"
)

// TransactionStatusRecord provides insight about the status of transaction
type TransactionStatusRecord struct {
	Status TransactionStatus
}

// DatabaseStreamingTransactions provides access to the Streaming Transactions API
type DatabaseStreamingTransactions interface {
	BeginTransaction(ctx context.Context, cols TransactionCollections, opts *BeginTransactionOptions) (TransactionID, error)
	CommitTransaction(ctx context.Context, tid TransactionID, opts *CommitTransactionOptions) error
	AbortTransaction(ctx context.Context, tid TransactionID, opts *AbortTransactionOptions) error

	TransactionStatus(ctx context.Context, tid TransactionID) (TransactionStatusRecord, error)
}
