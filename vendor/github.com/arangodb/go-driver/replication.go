//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
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
// Author Ewout Prangsma
//

package driver

import (
	"context"
	"time"
)

// Tick is represent a place in either the Write-Ahead Log,
// journals and datafiles value reported by the server
type Tick string

// Batch represents state on the server used during
// certain replication operations to keep state required
// by the client (such as Write-Ahead Log, inventory and data-files)
type Batch interface {
	// id of this batch
	BatchID() string
	// LastTick reported by the server for this batch
	LastTick() Tick
	// Extend the lifetime of an existing batch on the server
	Extend(ctx context.Context, ttl time.Duration) error
	// DeleteBatch deletes an existing batch on the server
	Delete(ctx context.Context) error
}

// Replication provides access to replication related operations.
type Replication interface {
	// CreateBatch creates a "batch" to prevent removal of state required for replication
	CreateBatch(ctx context.Context, db Database, serverID int64, ttl time.Duration) (Batch, error)

	// Get the inventory of the server containing all collections (with entire details) of a database.
	// When this function is called on a coordinator is a cluster, an ID of a DBServer must be provided
	// using a context that is prepare with `WithDBServerID`.
	DatabaseInventory(ctx context.Context, db Database) (DatabaseInventory, error)

	// GetRevisionTree retrieves the Revision tree (Merkel tree) associated with the collection.
	GetRevisionTree(ctx context.Context, db Database, batchId, collection string) (RevisionTree, error)

	// GetRevisionsByRanges retrieves the revision IDs of documents within requested ranges.
	GetRevisionsByRanges(ctx context.Context, db Database, batchId, collection string, minMaxRevision []RevisionMinMax,
		resume RevisionUInt64) (RevisionRanges, error)

	// GetRevisionDocuments retrieves documents by revision.
	GetRevisionDocuments(ctx context.Context, db Database, batchId, collection string,
		revisions Revisions) ([]map[string]interface{}, error)
}
