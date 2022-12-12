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

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

// Cluster provides access to cluster wide specific operations.
// To use this interface, an ArangoDB cluster is required.
type Cluster interface {
	// Get the cluster configuration & health
	Health(ctx context.Context) (ClusterHealth, error)

	// Get the inventory of the cluster containing all collections (with entire details) of a database.
	DatabaseInventory(ctx context.Context, db Database) (DatabaseInventory, error)

	// MoveShard moves a single shard of the given collection from server `fromServer` to
	// server `toServer`.
	MoveShard(ctx context.Context, col Collection, shard ShardID, fromServer, toServer ServerID) error

	// CleanOutServer triggers activities to clean out a DBServer.
	CleanOutServer(ctx context.Context, serverID string) error

	// ResignServer triggers activities to let a DBServer resign for all shards.
	ResignServer(ctx context.Context, serverID string) error

	// IsCleanedOut checks if the dbserver with given ID has been cleaned out.
	IsCleanedOut(ctx context.Context, serverID string) (bool, error)

	// RemoveServer is a low-level option to remove a server from a cluster.
	// This function is suitable for servers of type coordinator or dbserver.
	// The use of `ClientServerAdmin.Shutdown` is highly recommended above this function.
	RemoveServer(ctx context.Context, serverID ServerID) error
}

// ServerID identifies an arangod server in a cluster.
type ServerID string

// ClusterHealth contains health information for all servers in a cluster.
type ClusterHealth struct {
	// Unique identifier of the entire cluster.
	// This ID is created when the cluster was first created.
	ID string `json:"ClusterId"`
	// Health per server
	Health map[ServerID]ServerHealth `json:"Health"`
}

// ServerSyncStatus describes the servers sync status
type ServerSyncStatus string

const (
	ServerSyncStatusUnknown   ServerSyncStatus = "UNKNOWN"
	ServerSyncStatusUndefined ServerSyncStatus = "UNDEFINED"
	ServerSyncStatusStartup   ServerSyncStatus = "STARTUP"
	ServerSyncStatusStopping  ServerSyncStatus = "STOPPING"
	ServerSyncStatusStopped   ServerSyncStatus = "STOPPED"
	ServerSyncStatusServing   ServerSyncStatus = "SERVING"
	ServerSyncStatusShutdown  ServerSyncStatus = "SHUTDOWN"
)

// ServerHealth contains health information of a single server in a cluster.
type ServerHealth struct {
	Endpoint            string           `json:"Endpoint"`
	LastHeartbeatAcked  time.Time        `json:"LastHeartbeatAcked"`
	LastHeartbeatSent   time.Time        `json:"LastHeartbeatSent"`
	LastHeartbeatStatus string           `json:"LastHeartbeatStatus"`
	Role                ServerRole       `json:"Role"`
	ShortName           string           `json:"ShortName"`
	Status              ServerStatus     `json:"Status"`
	CanBeDeleted        bool             `json:"CanBeDeleted"`
	HostID              string           `json:"Host,omitempty"`
	Version             Version          `json:"Version,omitempty"`
	Engine              EngineType       `json:"Engine,omitempty"`
	SyncStatus          ServerSyncStatus `json:"SyncStatus,omitempty"`

	// Only for Coordinators
	AdvertisedEndpoint *string `json:"AdvertisedEndpoint,omitempty"`

	// Only for Agents
	Leader  *string `json:"Leader,omitempty"`
	Leading *bool   `json:"Leading,omitempty"`
}

// ServerStatus describes the health status of a server
type ServerStatus string

const (
	// ServerStatusGood indicates server is in good state
	ServerStatusGood ServerStatus = "GOOD"
	// ServerStatusBad indicates server has missed 1 heartbeat
	ServerStatusBad ServerStatus = "BAD"
	// ServerStatusFailed indicates server has been declared failed by the supervision, this happens after about 15s being bad.
	ServerStatusFailed ServerStatus = "FAILED"
)

// DatabaseInventory describes a detailed state of the collections & shards of a specific database within a cluster.
type DatabaseInventory struct {
	// Details of database, this is present since ArangoDB 3.6
	Info DatabaseInfo `json:"properties,omitempty"`
	// Details of all collections
	Collections []InventoryCollection `json:"collections,omitempty"`
	// Details of all views
	Views []InventoryView `json:"views,omitempty"`
	State State           `json:"state,omitempty"`
	Tick  string          `json:"tick,omitempty"`
}

type State struct {
	Running                bool      `json:"running,omitempty"`
	LastLogTick            string    `json:"lastLogTick,omitempty"`
	LastUncommittedLogTick string    `json:"lastUncommittedLogTick,omitempty"`
	TotalEvents            int64     `json:"totalEvents,omitempty"`
	Time                   time.Time `json:"time,omitempty"`
}

// UnmarshalJSON marshals State to arangodb json representation
func (s *State) UnmarshalJSON(d []byte) error {
	var internal interface{}

	if err := json.Unmarshal(d, &internal); err != nil {
		return err
	}

	if val, ok := internal.(string); ok {
		if val != "unused" {
			fmt.Printf("unrecognized State value: %s\n", val)
		}
		*s = State{}
		return nil
	} else {
		type Alias State
		out := Alias{}

		if err := json.Unmarshal(d, &out); err != nil {
			return &json.UnmarshalTypeError{
				Value: string(d),
				Type:  reflect.TypeOf(s).Elem(),
			}
		}
		*s = State(out)
	}

	return nil
}

// IsReady returns true if the IsReady flag of all collections is set.
func (i DatabaseInventory) IsReady() bool {
	for _, c := range i.Collections {
		if !c.IsReady {
			return false
		}
	}
	return true
}

// PlanVersion returns the plan version of the first collection in the given inventory.
func (i DatabaseInventory) PlanVersion() int64 {
	if len(i.Collections) == 0 {
		return 0
	}
	return i.Collections[0].PlanVersion
}

// CollectionByName returns the InventoryCollection with given name.
// Return false if not found.
func (i DatabaseInventory) CollectionByName(name string) (InventoryCollection, bool) {
	for _, c := range i.Collections {
		if c.Parameters.Name == name {
			return c, true
		}
	}
	return InventoryCollection{}, false
}

// ViewByName returns the InventoryView with given name.
// Return false if not found.
func (i DatabaseInventory) ViewByName(name string) (InventoryView, bool) {
	for _, v := range i.Views {
		if v.Name == name {
			return v, true
		}
	}
	return InventoryView{}, false
}

// InventoryCollection is a single element of a DatabaseInventory, containing all information
// of a specific collection.
type InventoryCollection struct {
	Parameters  InventoryCollectionParameters `json:"parameters"`
	Indexes     []InventoryIndex              `json:"indexes,omitempty"`
	PlanVersion int64                         `json:"planVersion,omitempty"`
	IsReady     bool                          `json:"isReady,omitempty"`
	AllInSync   bool                          `json:"allInSync,omitempty"`
}

// IndexByFieldsAndType returns the InventoryIndex with given fields & type.
// Return false if not found.
func (i InventoryCollection) IndexByFieldsAndType(fields []string, indexType string) (InventoryIndex, bool) {
	for _, idx := range i.Indexes {
		if idx.Type == indexType && idx.FieldsEqual(fields) {
			return idx, true
		}
	}
	return InventoryIndex{}, false
}

// InventoryCollectionParameters contains all configuration parameters of a collection in a database inventory.
type InventoryCollectionParameters struct {
	// Available from 3.7 ArangoD version.
	CacheEnabled         bool   `json:"cacheEnabled,omitempty"`
	Deleted              bool   `json:"deleted,omitempty"`
	DistributeShardsLike string `json:"distributeShardsLike,omitempty"`
	// Deprecated: since 3.7 version. It is related only to MMFiles.
	DoCompact bool `json:"doCompact,omitempty"`
	// Available from 3.7 ArangoD version.
	GloballyUniqueId string `json:"globallyUniqueId,omitempty"`
	ID               string `json:"id,omitempty"`
	// Deprecated: since 3.7 version. It is related only to MMFiles.
	IndexBuckets int              `json:"indexBuckets,omitempty"`
	Indexes      []InventoryIndex `json:"indexes,omitempty"`
	// Available from 3.9 ArangoD version.
	InternalValidatorType int `json:"internalValidatorType,omitempty"`
	// Available from 3.7 ArangoD version.
	IsDisjoint bool `json:"isDisjoint,omitempty"`
	IsSmart    bool `json:"isSmart,omitempty"`
	// Available from 3.7 ArangoD version.
	IsSmartChild bool `json:"isSmartChild,omitempty"`
	IsSystem     bool `json:"isSystem,omitempty"`
	// Deprecated: since 3.7 version. It is related only to MMFiles.
	IsVolatile bool `json:"isVolatile,omitempty"`
	// Deprecated: since 3.7 version. It is related only to MMFiles.
	JournalSize int64 `json:"journalSize,omitempty"`
	KeyOptions  struct {
		AllowUserKeys bool `json:"allowUserKeys,omitempty"`
		// Deprecated: this field has wrong type and will be removed in the future. It is not used anymore since it can cause parsing issues.
		LastValue   int64  `json:"-"`
		LastValueV2 uint64 `json:"lastValue,omitempty"`
		Type        string `json:"type,omitempty"`
	} `json:"keyOptions"`
	// Deprecated: use 'WriteConcern' instead.
	MinReplicationFactor int    `json:"minReplicationFactor,omitempty"`
	Name                 string `json:"name,omitempty"`
	NumberOfShards       int    `json:"numberOfShards,omitempty"`
	// Deprecated: since 3.7 ArangoD version.
	Path              string `json:"path,omitempty"`
	PlanID            string `json:"planId,omitempty"`
	ReplicationFactor int    `json:"replicationFactor,omitempty"`
	// Schema for collection validation.
	Schema            *CollectionSchemaOptions `json:"schema,omitempty"`
	ShadowCollections []int                    `json:"shadowCollections,omitempty"`
	ShardingStrategy  ShardingStrategy         `json:"shardingStrategy,omitempty"`
	ShardKeys         []string                 `json:"shardKeys,omitempty"`
	Shards            map[ShardID][]ServerID   `json:"shards,omitempty"`
	// Optional only for some collections.
	SmartGraphAttribute string `json:"smartGraphAttribute,omitempty"`
	// Optional only for some collections.
	SmartJoinAttribute string           `json:"smartJoinAttribute,omitempty"`
	Status             CollectionStatus `json:"status,omitempty"`
	// Available from 3.7 ArangoD version.
	SyncByRevision bool           `json:"syncByRevision,omitempty"`
	Type           CollectionType `json:"type,omitempty"`
	// Available from 3.7 ArangoD version.
	UsesRevisionsAsDocumentIds bool `json:"usesRevisionsAsDocumentIds,omitempty"`
	WaitForSync                bool `json:"waitForSync,omitempty"`
	// Available from 3.6 ArangoD version.
	WriteConcern int `json:"writeConcern,omitempty"`
	// Available from 3.10 ArangoD version.
	ComputedValues []ComputedValue `json:"computedValues,omitempty"`
}

// IsSatellite returns true if the collection is a satellite collection
func (icp *InventoryCollectionParameters) IsSatellite() bool {
	return icp.ReplicationFactor == ReplicationFactorSatellite
}

// ShardID is an internal identifier of a specific shard
type ShardID string

// InventoryIndex contains all configuration parameters of a single index of a collection in a database inventory.
type InventoryIndex struct {
	ID              string   `json:"id,omitempty"`
	Type            string   `json:"type,omitempty"`
	Fields          []string `json:"fields,omitempty"`
	Unique          bool     `json:"unique"`
	Sparse          bool     `json:"sparse"`
	Deduplicate     bool     `json:"deduplicate"`
	MinLength       int      `json:"minLength,omitempty"`
	GeoJSON         bool     `json:"geoJson,omitempty"`
	Name            string   `json:"name,omitempty"`
	ExpireAfter     int      `json:"expireAfter,omitempty"`
	Estimates       bool     `json:"estimates,omitempty"`
	FieldValueTypes string   `json:"fieldValueTypes,omitempty"`
	CacheEnabled    *bool    `json:"cacheEnabled,omitempty"`
}

// FieldsEqual returns true when the given fields list equals the
// Fields list in the InventoryIndex.
// The order of fields is irrelevant.
func (i InventoryIndex) FieldsEqual(fields []string) bool {
	return stringSliceEqualsIgnoreOrder(i.Fields, fields)
}

// InventoryView is a single element of a DatabaseInventory, containing all information
// of a specific view.
type InventoryView struct {
	Name     string   `json:"name,omitempty"`
	Deleted  bool     `json:"deleted,omitempty"`
	ID       string   `json:"id,omitempty"`
	IsSystem bool     `json:"isSystem,omitempty"`
	PlanID   string   `json:"planId,omitempty"`
	Type     ViewType `json:"type,omitempty"`
	// Include all properties from an arangosearch view.
	ArangoSearchViewProperties
}

// stringSliceEqualsIgnoreOrder returns true when the given lists contain the same elements.
// The order of elements is irrelevant.
func stringSliceEqualsIgnoreOrder(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	bMap := make(map[string]struct{})
	for _, x := range b {
		bMap[x] = struct{}{}
	}
	for _, x := range a {
		if _, found := bMap[x]; !found {
			return false
		}
	}
	return true
}
