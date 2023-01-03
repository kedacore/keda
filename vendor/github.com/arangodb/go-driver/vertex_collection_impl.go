//
// DISCLAIMER
//
// Copyright 2017-2021 ArangoDB GmbH, Cologne, Germany
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
// Author Tomasz Mielech
//

package driver

import (
	"context"
	"path"
)

// newVertexCollection creates a new Vertex Collection implementation.
func newVertexCollection(name string, g *graph) (Collection, error) {
	if name == "" {
		return nil, WithStack(InvalidArgumentError{Message: "name is empty"})
	}
	if g == nil {
		return nil, WithStack(InvalidArgumentError{Message: "g is nil"})
	}
	return &vertexCollection{
		name: name,
		g:    g,
		conn: g.db.conn,
	}, nil
}

type vertexCollection struct {
	name string
	g    *graph
	conn Connection
}

// relPath creates the relative path to this edge collection (`_db/<db-name>/_api/gharial/<graph-name>/vertex/<collection-name>`)
func (c *vertexCollection) relPath() string {
	escapedName := pathEscape(c.name)
	return path.Join(c.g.relPath(), "vertex", escapedName)
}

// Name returns the name of the edge collection.
func (c *vertexCollection) Name() string {
	return c.name
}

// Database returns the database containing the collection.
func (c *vertexCollection) Database() Database {
	return c.g.db
}

// rawCollection returns a standard document implementation of Collection
// for this vertex collection.
func (c *vertexCollection) rawCollection() Collection {
	result, _ := newCollection(c.name, c.g.db)
	return result
}

// Status fetches the current status of the collection.
func (c *vertexCollection) Status(ctx context.Context) (CollectionStatus, error) {
	result, err := c.rawCollection().Status(ctx)
	if err != nil {
		return CollectionStatus(0), WithStack(err)
	}
	return result, nil
}

// Count fetches the number of document in the collection.
func (c *vertexCollection) Count(ctx context.Context) (int64, error) {
	result, err := c.rawCollection().Count(ctx)
	if err != nil {
		return 0, WithStack(err)
	}
	return result, nil
}

// Statistics returns the number of documents and additional statistical information about the collection.
func (c *vertexCollection) Statistics(ctx context.Context) (CollectionStatistics, error) {
	result, err := c.rawCollection().Statistics(ctx)
	if err != nil {
		return CollectionStatistics{}, WithStack(err)
	}
	return result, nil
}

// Revision fetches the revision ID of the collection.
// The revision ID is a server-generated string that clients can use to check whether data
// in a collection has changed since the last revision check.
func (c *vertexCollection) Revision(ctx context.Context) (string, error) {
	result, err := c.rawCollection().Revision(ctx)
	if err != nil {
		return "", WithStack(err)
	}
	return result, nil
}

// Properties fetches extended information about the collection.
func (c *vertexCollection) Properties(ctx context.Context) (CollectionProperties, error) {
	result, err := c.rawCollection().Properties(ctx)
	if err != nil {
		return CollectionProperties{}, WithStack(err)
	}
	return result, nil
}

// SetProperties changes properties of the collection.
func (c *vertexCollection) SetProperties(ctx context.Context, options SetCollectionPropertiesOptions) error {
	if err := c.rawCollection().SetProperties(ctx, options); err != nil {
		return WithStack(err)
	}
	return nil
}

// Shards fetches shards information of the collection.
func (c *vertexCollection) Shards(ctx context.Context, details bool) (CollectionShards, error) {
	result, err := c.rawCollection().Shards(ctx, details)
	if err != nil {
		return result, WithStack(err)
	}
	return result, nil
}

// Load the collection into memory.
func (c *vertexCollection) Load(ctx context.Context) error {
	if err := c.rawCollection().Load(ctx); err != nil {
		return WithStack(err)
	}
	return nil
}

// Unload unloads the collection from memory.
func (c *vertexCollection) Unload(ctx context.Context) error {
	if err := c.rawCollection().Unload(ctx); err != nil {
		return WithStack(err)
	}
	return nil
}

// Remove removes the entire collection.
// If the collection does not exist, a NotFoundError is returned.
func (c *vertexCollection) Remove(ctx context.Context) error {
	req, err := c.conn.NewRequest("DELETE", c.relPath())
	if err != nil {
		return WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(201, 202); err != nil {
		return WithStack(err)
	}
	return nil
}

// Truncate removes all documents from the collection, but leaves the indexes intact.
func (c *vertexCollection) Truncate(ctx context.Context) error {
	if err := c.rawCollection().Truncate(ctx); err != nil {
		return WithStack(err)
	}
	return nil
}
