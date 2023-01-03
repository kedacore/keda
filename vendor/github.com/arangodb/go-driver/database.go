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
// Author Ewout Prangsma
//

package driver

import "context"

// Database provides access to all collections & graphs in a single database.
type Database interface {
	// Name returns the name of the database.
	Name() string

	// Info fetches information about the database.
	Info(ctx context.Context) (DatabaseInfo, error)

	// EngineInfo returns information about the database engine being used.
	// Note: When your cluster has multiple endpoints (cluster), you will get information
	// from the server that is currently being used.
	// If you want to know exactly which server the information is from, use a client
	// with only a single endpoint and avoid automatic synchronization of endpoints.
	EngineInfo(ctx context.Context) (EngineInfo, error)

	// Remove removes the entire database.
	// If the database does not exist, a NotFoundError is returned.
	Remove(ctx context.Context) error

	// Collection functions
	DatabaseCollections

	// View functions
	DatabaseViews

	// Graph functions
	DatabaseGraphs

	// Pregel functions
	DatabasePregels

	// Streaming Transactions functions
	DatabaseStreamingTransactions

	// ArangoSearch Analyzers API
	DatabaseArangoSearchAnalyzers

	// Query performs an AQL query, returning a cursor used to iterate over the returned documents.
	// Note that the returned Cursor must always be closed to avoid holding on to resources in the server while they are no longer needed.
	Query(ctx context.Context, query string, bindVars map[string]interface{}) (Cursor, error)

	// ValidateQuery validates an AQL query.
	// When the query is valid, nil returned, otherwise an error is returned.
	// The query is not executed.
	ValidateQuery(ctx context.Context, query string) error

	// OptimizerRulesForQueries returns the available optimizer rules for AQL queries
	// returns an array of objects that contain the name of each available rule and its respective flags.
	OptimizerRulesForQueries(ctx context.Context) ([]QueryRule, error)

	// Transaction performs a javascript transaction. The result of the transaction function is returned.
	Transaction(ctx context.Context, action string, options *TransactionOptions) (interface{}, error)
}

// DatabaseInfo contains information about a database
type DatabaseInfo struct {
	// The identifier of the database.
	ID string `json:"id,omitempty"`
	// The name of the database.
	Name string `json:"name,omitempty"`
	// The filesystem path of the database.
	Path string `json:"path,omitempty"`
	// If true then the database is the _system database.
	IsSystem bool `json:"isSystem,omitempty"`
	// Default replication factor for collections in database
	ReplicationFactor int `json:"replicationFactor,omitempty"`
	// Default write concern for collections in database
	WriteConcern int `json:"writeConcern,omitempty"`
	// Default sharding for collections in database
	Sharding DatabaseSharding `json:"sharding,omitempty"`
}

// EngineType indicates type of database engine being used.
type EngineType string

const (
	EngineTypeMMFiles = EngineType("mmfiles")
	EngineTypeRocksDB = EngineType("rocksdb")
)

func (t EngineType) String() string {
	return string(t)
}

// EngineInfo contains information about the database engine being used.
type EngineInfo struct {
	Type     EngineType             `json:"name"`
	Supports map[string]interface{} `json:"supports,omitempty"`
}

type QueryRule struct {
	Name  string     `json:"name"`
	Flags QueryFlags `json:"flags,omitempty"`
}

type QueryFlags struct {
	Hidden                   bool `json:"hidden,omitempty"`
	ClusterOnly              bool `json:"clusterOnly,omitempty"`
	CanBeDisabled            bool `json:"canBeDisabled,omitempty"`
	CanCreateAdditionalPlans bool `json:"canCreateAdditionalPlans,omitempty"`
	DisabledByDefault        bool `json:"disabledByDefault,omitempty"`
	EnterpriseOnly           bool `json:"enterpriseOnly,omitempty"`
}
