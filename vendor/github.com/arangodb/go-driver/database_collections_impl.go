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
	"path"
)

// Collection opens a connection to an existing collection within the database.
// If no collection with given name exists, an NotFoundError is returned.
func (d *database) Collection(ctx context.Context, name string) (Collection, error) {
	escapedName := pathEscape(name)
	req, err := d.conn.NewRequest("GET", path.Join(d.relPath(), "_api/collection", escapedName))
	if err != nil {
		return nil, WithStack(err)
	}
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	coll, err := newCollection(name, d)
	if err != nil {
		return nil, WithStack(err)
	}
	return coll, nil
}

// CollectionExists returns true if a collection with given name exists within the database.
func (d *database) CollectionExists(ctx context.Context, name string) (bool, error) {
	escapedName := pathEscape(name)
	req, err := d.conn.NewRequest("GET", path.Join(d.relPath(), "_api/collection", escapedName))
	if err != nil {
		return false, WithStack(err)
	}
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return false, WithStack(err)
	}
	if err := resp.CheckStatus(200); err == nil {
		return true, nil
	} else if IsNotFound(err) {
		return false, nil
	} else {
		return false, WithStack(err)
	}
}

type getCollectionResponse struct {
	Result []CollectionInfo `json:"result,omitempty"`
}

// Collections returns a list of all collections in the database.
func (d *database) Collections(ctx context.Context) ([]Collection, error) {
	req, err := d.conn.NewRequest("GET", path.Join(d.relPath(), "_api/collection"))
	if err != nil {
		return nil, WithStack(err)
	}
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	var data getCollectionResponse
	if err := resp.ParseBody("", &data); err != nil {
		return nil, WithStack(err)
	}
	result := make([]Collection, 0, len(data.Result))
	for _, info := range data.Result {
		col, err := newCollection(info.Name, d)
		if err != nil {
			return nil, WithStack(err)
		}
		result = append(result, col)
	}
	return result, nil
}

type createCollectionOptionsInternal struct {
	CacheEnabled          *bool                 `json:"cacheEnabled,omitempty"`
	ComputedValues        []ComputedValue       `json:"computedValues,omitempty"`
	DistributeShardsLike  string                `json:"distributeShardsLike,omitempty"`
	DoCompact             *bool                 `json:"doCompact,omitempty"`
	IndexBuckets          int                   `json:"indexBuckets,omitempty"`
	InternalValidatorType int                   `json:"internalValidatorType,omitempty"`
	IsDisjoint            bool                  `json:"isDisjoint,omitempty"`
	IsSmart               bool                  `json:"isSmart,omitempty"`
	IsSystem              bool                  `json:"isSystem,omitempty"`
	IsVolatile            bool                  `json:"isVolatile,omitempty"`
	JournalSize           int                   `json:"journalSize,omitempty"`
	KeyOptions            *CollectionKeyOptions `json:"keyOptions,omitempty"`
	// Deprecated: use 'WriteConcern' instead
	MinReplicationFactor int                      `json:"minReplicationFactor,omitempty"`
	Name                 string                   `json:"name"`
	NumberOfShards       int                      `json:"numberOfShards,omitempty"`
	ReplicationFactor    replicationFactor        `json:"replicationFactor,omitempty"`
	Schema               *CollectionSchemaOptions `json:"schema,omitempty"`
	ShardingStrategy     ShardingStrategy         `json:"shardingStrategy,omitempty"`
	ShardKeys            []string                 `json:"shardKeys,omitempty"`
	SmartGraphAttribute  string                   `json:"smartGraphAttribute,omitempty"`
	SmartJoinAttribute   string                   `json:"smartJoinAttribute,omitempty"`
	SyncByRevision       bool                     `json:"syncByRevision,omitempty"`
	Type                 CollectionType           `json:"type,omitempty"`
	WaitForSync          bool                     `json:"waitForSync,omitempty"`
	WriteConcern         int                      `json:"writeConcern,omitempty"`
}

// CreateCollection creates a new collection with given name and options, and opens a connection to it.
// If a collection with given name already exists within the database, a DuplicateError is returned.
func (d *database) CreateCollection(ctx context.Context, name string, options *CreateCollectionOptions) (Collection, error) {
	options.Init()
	input := createCollectionOptionsInternal{
		Name: name,
	}
	if options != nil {
		input.fromExternal(options)
	}
	req, err := d.conn.NewRequest("POST", path.Join(d.relPath(), "_api/collection"))
	if err != nil {
		return nil, WithStack(err)
	}
	if _, err := req.SetBody(input); err != nil {
		return nil, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := d.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	col, err := newCollection(name, d)
	if err != nil {
		return nil, WithStack(err)
	}
	return col, nil
}

func (p *createCollectionOptionsInternal) fromExternal(i *CreateCollectionOptions) {
	p.CacheEnabled = i.CacheEnabled
	p.ComputedValues = i.ComputedValues
	p.DistributeShardsLike = i.DistributeShardsLike
	p.DoCompact = i.DoCompact
	p.IndexBuckets = i.IndexBuckets
	p.InternalValidatorType = i.InternalValidatorType
	p.IsDisjoint = i.IsDisjoint
	p.IsSmart = i.IsSmart
	p.IsSystem = i.IsSystem
	p.IsVolatile = i.IsVolatile
	p.JournalSize = i.JournalSize
	p.KeyOptions = i.KeyOptions
	p.MinReplicationFactor = i.MinReplicationFactor
	p.NumberOfShards = i.NumberOfShards
	p.ReplicationFactor = replicationFactor(i.ReplicationFactor)
	p.Schema = i.Schema
	p.ShardingStrategy = i.ShardingStrategy
	p.ShardKeys = i.ShardKeys
	p.SmartGraphAttribute = i.SmartGraphAttribute
	p.SmartJoinAttribute = i.SmartJoinAttribute
	p.SyncByRevision = i.SyncByRevision
	p.Type = i.Type
	p.WaitForSync = i.WaitForSync
	p.WriteConcern = i.WriteConcern
}
