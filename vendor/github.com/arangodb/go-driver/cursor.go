//
// DISCLAIMER
//
// Copyright 2023 ArangoDB GmbH, Cologne, Germany
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

package driver

import (
	"context"
	"io"
	"time"
)

// QueryExtra holds Query extra information
type QueryExtra interface {
	// GetStatistics returns Query statistics
	GetStatistics() QueryStatistics

	// GetProfileRaw returns raw profile information in json
	GetProfileRaw() ([]byte, bool, error)

	// GetPlanRaw returns raw plan
	GetPlanRaw() ([]byte, bool, error)
}

// QueryStatistics Statistics returned with the query cursor
type QueryStatistics interface {
	// WritesExecuted the total number of data-modification operations successfully executed.
	WritesExecuted() int64

	// WritesIgnored The total number of data-modification operations that were unsuccessful
	WritesIgnored() int64

	// ScannedFull The total number of documents iterated over when scanning a collection without an index.
	ScannedFull() int64

	// ScannedIndex The total number of documents iterated over when scanning a collection using an index.
	ScannedIndex() int64

	// Filtered the total number of documents that were removed after executing a filter condition in a FilterNode
	Filtered() int64

	// FullCount Returns the number of results before the last LIMIT in the query was applied.
	// A valid return value is only available when the has been created with a context that was
	// prepared with `WithFullCount`. Additionally, this will also not return a valid value if
	// the context was prepared with `WithStream`.
	FullCount() int64

	// ExecutionTime of the query (wall-clock time). value will be set from the outside
	ExecutionTime() time.Duration
}

// Cursor is returned from a query, used to iterate over a list of documents.
// Note that a Cursor must always be closed to avoid holding on to resources in the server while they are no longer needed.
type Cursor interface {
	io.Closer

	// HasMore returns true if the next call to ReadDocument does not return a NoMoreDocuments error.
	HasMore() bool

	// ReadDocument reads the next document from the cursor.
	// The document data is stored into result, the document metadata is returned.
	// If the cursor has no more documents, a NoMoreDocuments error is returned.
	// Note: If the query (resulting in this cursor) does not return documents,
	//       then the returned DocumentMeta will be empty.
	ReadDocument(ctx context.Context, result interface{}) (DocumentMeta, error)

	// RetryReadDocument reads the last document from the cursor once more time
	// It can be used e.g., in case of network error during ReadDocument
	// It requires 'driver.WithQueryAllowRetry' to be set to true on the Context during Cursor creation.
	RetryReadDocument(ctx context.Context, result interface{}) (DocumentMeta, error)

	// Count returns the total number of result documents available.
	// A valid return value is only available when the cursor has been created with a context that was
	// prepared with `WithQueryCount` and not with `WithQueryStream`.
	Count() int64

	// Statistics returns the query execution statistics for this cursor.
	// This might not be valid if the cursor has been created with a context that was
	// prepared with `WithQueryStream`
	Statistics() QueryStatistics

	// Extra returns the query extras for this cursor.
	Extra() QueryExtra
}
