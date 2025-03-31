//
// DISCLAIMER
//
// Copyright 2020-2025 ArangoDB GmbH, Cologne, Germany
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

package driver

// TransactionOptions contains options that customize the transaction.
type TransactionOptions struct {
	// Transaction size limit in bytes. Honored by the RocksDB storage engine only.
	MaxTransactionSize int

	// An optional numeric value that can be used to set a timeout for waiting on collection
	// locks. If not specified, a default value will be used.
	// Setting lockTimeout to 0 will make ArangoDB not time out waiting for a lock.
	LockTimeout *int

	// An optional boolean flag that, if set, will force the transaction to write
	// all data to disk before returning.
	WaitForSync bool

	// Deprecated
	//
	// Maximum number of operations after which an intermediate commit is performed
	// automatically. Honored by the RocksDB storage engine only.
	IntermediateCommitCount *int

	// Optional arguments passed to action.
	Params []interface{}

	// Deprecated:
	//
	// Maximum total size of operations after which an intermediate commit is
	// performed automatically. Honored by the RocksDB storage engine only.
	IntermediateCommitSize *int

	// ReadCollections Collections that the transaction reads from.
	ReadCollections []string

	// WriteCollections Collections that the transaction writes to.
	WriteCollections []string

	// ExclusiveCollections Collections that the transaction writes exclusively to.
	ExclusiveCollections []string
}

type transactionRequest struct {
	MaxTransactionSize      int                           `json:"maxTransactionSize"`
	LockTimeout             *int                          `json:"lockTimeout,omitempty"`
	WaitForSync             bool                          `json:"waitForSync"`
	IntermediateCommitCount *int                          `json:"intermediateCommitCount,omitempty"`
	Params                  []interface{}                 `json:"params"`
	IntermediateCommitSize  *int                          `json:"intermediateCommitSize,omitempty"`
	Action                  string                        `json:"action"`
	Collections             transactionCollectionsRequest `json:"collections"`
}

type transactionCollectionsRequest struct {
	Read      []string `json:"read,omitempty"`
	Write     []string `json:"write,omitempty"`
	Exclusive []string `json:"exclusive,omitempty"`
}

type transactionResponse struct {
	ArangoError
	Result interface{} `json:"result"`
}
