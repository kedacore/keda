//
// DISCLAIMER
//
// Copyright 2017-2025 ArangoDB GmbH, Cologne, Germany
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

// DatabaseCollections provides access to all collections in a single database.
type DatabaseCollections interface {
	// Collection opens a connection to an existing collection within the database.
	// If no collection with given name exists, an NotFoundError is returned.
	Collection(ctx context.Context, name string) (Collection, error)

	// CollectionExists returns true if a collection with given name exists within the database.
	CollectionExists(ctx context.Context, name string) (bool, error)

	// Collections returns a list of all collections in the database.
	Collections(ctx context.Context) ([]Collection, error)

	// CreateCollection creates a new collection with given name and options, and opens a connection to it.
	// If a collection with given name already exists within the database, a DuplicateError is returned.
	CreateCollection(ctx context.Context, name string, options *CreateCollectionOptions) (Collection, error)
}

// CreateCollectionOptions contains options that customize the creating of a collection.
type CreateCollectionOptions struct {
	// CacheEnabled set cacheEnabled option in collection properties
	CacheEnabled *bool `json:"cacheEnabled,omitempty"`
	// ComputedValues let configure collections to generate document attributes when documents are created or modified, using an AQL expression
	ComputedValues []ComputedValue `json:"computedValues,omitempty"`
	// This field is used for internal purposes only. DO NOT USE.
	DistributeShardsLike string `json:"distributeShardsLike,omitempty"`
	// DoCompact checks if the collection will be compacted (default is true)
	DoCompact *bool `json:"doCompact,omitempty"`
	// The number of buckets into which indexes using a hash table are split. The default is 16 and this number has to be a power
	// of 2 and less than or equal to 1024. For very large collections one should increase this to avoid long pauses when the hash
	// table has to be initially built or resized, since buckets are resized individually and can be initially built in parallel.
	// For example, 64 might be a sensible value for a collection with 100 000 000 documents.
	// Currently, only the edge index respects this value, but other index types might follow in future ArangoDB versions.
	// Changes are applied when the collection is loaded the next time.
	IndexBuckets int `json:"indexBuckets,omitempty"`
	// Available from 3.9 ArangoD version.
	InternalValidatorType int `json:"internalValidatorType,omitempty"`
	// IsDisjoint set isDisjoint flag for Graph. Required ArangoDB 3.7+
	IsDisjoint bool `json:"isDisjoint,omitempty"`
	// Set to create a smart edge or vertex collection.
	// This requires ArangoDB Enterprise Edition.
	IsSmart bool `json:"isSmart,omitempty"`
	// If true, create a system collection. In this case collection-name should start with an underscore.
	// End users should normally create non-system collections only. API implementors may be required to create system
	// collections in very special occasions, but normally a regular collection will do. (The default is false)
	IsSystem bool `json:"isSystem,omitempty"`
	// If true then the collection data is kept in-memory only and not made persistent.
	// Unloading the collection will cause the collection data to be discarded. Stopping or re-starting the server will also
	// cause full loss of data in the collection. Setting this option will make the resulting collection be slightly faster
	// than regular collections because ArangoDB does not enforce any synchronization to disk and does not calculate any
	// CRC checksums for datafiles (as there are no datafiles). This option should therefore be used for cache-type collections only,
	// and not for data that cannot be re-created otherwise. (The default is false)
	IsVolatile bool `json:"isVolatile,omitempty"`
	// The maximal size of a journal or datafile in bytes. The value must be at least 1048576 (1 MiB). (The default is a configuration parameter)
	JournalSize int `json:"journalSize,omitempty"`
	// Specifies how keys in the collection are created.
	KeyOptions *CollectionKeyOptions `json:"keyOptions,omitempty"`
	// Deprecated: use 'WriteConcern' instead
	MinReplicationFactor int `json:"minReplicationFactor,omitempty"`
	// In a cluster, this value determines the number of shards to create for the collection. In a single server setup, this option is meaningless. (default is 1)
	NumberOfShards int `json:"numberOfShards,omitempty"`
	// ReplicationFactor in a cluster (default is 1), this attribute determines how many copies of each shard are kept on different DBServers.
	// The value 1 means that only one copy (no synchronous replication) is kept.
	// A value of k means that k-1 replicas are kept. Any two copies reside on different DBServers.
	// Replication between them is synchronous, that is, every write operation to the "leader" copy will be replicated to all "follower" replicas,
	// before the write operation is reported successful. If a server fails, this is detected automatically
	// and one of the servers holding copies take over, usually without an error being reported.
	ReplicationFactor int `json:"replicationFactor,omitempty"`
	// Schema for collection validation
	Schema *CollectionSchemaOptions `json:"schema,omitempty"`
	// This attribute specifies the name of the sharding strategy to use for the collection.
	// Must be one of ShardingStrategy* values.
	ShardingStrategy ShardingStrategy `json:"shardingStrategy,omitempty"`
	// In a cluster, this attribute determines which document attributes are used to
	// determine the target shard for documents. Documents are sent to shards based on the values of their shard key attributes.
	// The values of all shard key attributes in a document are hashed, and the hash value is used to determine the target shard.
	// Note: Values of shard key attributes cannot be changed once set. This option is meaningless in a single server setup.
	// The default is []string{"_key"}.
	ShardKeys []string `json:"shardKeys,omitempty"`
	// This field must be set to the attribute that will be used for sharding or smart graphs.
	// All vertices are required to have this attribute set. Edges derive the attribute from their connected vertices.
	// This requires ArangoDB Enterprise Edition.
	SmartGraphAttribute string `json:"smartGraphAttribute,omitempty"`
	// SmartJoinAttribute
	// In the specific case that the two collections have the same number of shards, the data of the two collections can
	// be co-located on the same server for the same shard key values. In this case the extra hop via the coordinator will not be necessary.
	// See documentation for smart joins.
	// This requires ArangoDB Enterprise Edition.
	SmartJoinAttribute string `json:"smartJoinAttribute,omitempty"`
	// Available from 3.7 ArangoDB version
	SyncByRevision bool `json:"syncByRevision,omitempty"`
	// The type of the collection to create. (default is CollectionTypeDocument)
	Type CollectionType `json:"type,omitempty"`
	// If true then the data is synchronized to disk before returning from a document create, update, replace or removal operation. (default: false)
	WaitForSync bool `json:"waitForSync,omitempty"`
	// WriteConcern contains how many copies must be available before a collection can be written.
	// It is required that 1 <= WriteConcern <= ReplicationFactor.
	// Default is 1. Not available for satellite collections.
	// Available from 3.6 ArangoDB version.
	WriteConcern int `json:"writeConcern,omitempty"`
}

// Init translate deprecated fields into current one for backward compatibility
func (c *CreateCollectionOptions) Init() {
	if c == nil {
		return
	}

	c.KeyOptions.Init()
}

// CollectionType is the type of a collection.
type CollectionType int

const (
	// CollectionTypeDocument specifies a document collection
	CollectionTypeDocument = CollectionType(2)
	// CollectionTypeEdge specifies an edges collection
	CollectionTypeEdge = CollectionType(3)
)

type ComputedValue struct {
	//  The name of the target attribute. Can only be a top-level attribute, but you
	//   may return a nested object. Cannot be `_key`, `_id`, `_rev`, `_from`, `_to`,
	//   or a shard key attribute.
	Name string `json:"name"`
	// An AQL `RETURN` operation with an expression that computes the desired value.
	Expression string `json:"expression"`
	// An array of strings to define on which write operations the value shall be
	// computed. The possible values are `"insert"`, `"update"`, and `"replace"`.
	// The default is `["insert", "update", "replace"]`.
	ComputeOn []ComputeOn `json:"computeOn,omitempty"`
	// Whether the computed value shall take precedence over a user-provided or existing attribute.
	Overwrite bool `json:"overwrite"`
	// Whether to let the write operation fail if the expression produces a warning. The default is false.
	FailOnWarning *bool `json:"failOnWarning,omitempty"`
	// Whether the result of the expression shall be stored if it evaluates to `null`.
	// This can be used to skip the value computation if any pre-conditions are not met.
	KeepNull *bool `json:"keepNull,omitempty"`
}

type ComputeOn string

const (
	ComputeOnInsert  ComputeOn = "insert"
	ComputeOnUpdate  ComputeOn = "update"
	ComputeOnReplace ComputeOn = "replace"
)

// CollectionKeyOptions specifies ways for creating keys of a collection.
type CollectionKeyOptions struct {
	// If set to true, then it is allowed to supply own key values in the _key attribute of a document.
	// If set to false, then the key generator will solely be responsible for generating keys and supplying own
	// key values in the _key attribute of documents is considered an error.
	//
	// Deprecated: Use AllowUserKeysPtr instead
	AllowUserKeys bool `json:"-"`
	// If set to true, then it is allowed to supply own key values in the _key attribute of a document.
	// If set to false, then the key generator will solely be responsible for generating keys and supplying own
	// key values in the _key attribute of documents is considered an error.
	AllowUserKeysPtr *bool `json:"allowUserKeys,omitempty"`
	// Specifies the type of the key generator. The currently available generators are traditional and autoincrement.
	Type KeyGeneratorType `json:"type,omitempty"`
	// increment value for autoincrement key generator. Not used for other key generator types.
	Increment int `json:"increment,omitempty"`
	// Initial offset value for autoincrement key generator. Not used for other key generator types.
	Offset int `json:"offset,omitempty"`
}

// Init translate deprecated fields into current one for backward compatibility
func (c *CollectionKeyOptions) Init() {
	if c == nil {
		return
	}

	if c.AllowUserKeysPtr == nil {
		if c.AllowUserKeys {
			c.AllowUserKeysPtr = &c.AllowUserKeys
		}
	}
}

// KeyGeneratorType is a type of key generated, used in `CollectionKeyOptions`.
type KeyGeneratorType string

const (
	KeyGeneratorTraditional   = KeyGeneratorType("traditional")
	KeyGeneratorAutoIncrement = KeyGeneratorType("autoincrement")
)

// ShardingStrategy describes the sharding strategy of a collection
type ShardingStrategy string

const (
	ShardingStrategyCommunityCompat           ShardingStrategy = "community-compat"
	ShardingStrategyEnterpriseCompat          ShardingStrategy = "enterprise-compat"
	ShardingStrategyEnterpriseSmartEdgeCompat ShardingStrategy = "enterprise-smart-edge-compat"
	ShardingStrategyHash                      ShardingStrategy = "hash"
	ShardingStrategyEnterpriseHashSmartEdge   ShardingStrategy = "enterprise-hash-smart-edge"
)
