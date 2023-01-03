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

type listVertexCollectionResponse struct {
	Collections []string `json:"collections,omitempty"`
	ArangoError
}

// VertexCollection opens a connection to an existing edge-collection within the graph.
// If no edge-collection with given name exists, an NotFoundError is returned.
func (g *graph) VertexCollection(ctx context.Context, name string) (Collection, error) {
	req, err := g.conn.NewRequest("GET", path.Join(g.relPath(), "vertex"))
	if err != nil {
		return nil, WithStack(err)
	}
	resp, err := g.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	var data listVertexCollectionResponse
	if err := resp.ParseBody("", &data); err != nil {
		return nil, WithStack(err)
	}
	for _, n := range data.Collections {
		if n == name {
			ec, err := newVertexCollection(name, g)
			if err != nil {
				return nil, WithStack(err)
			}
			return ec, nil
		}
	}
	return nil, WithStack(newArangoError(404, 0, "not found"))
}

// VertexCollectionExists returns true if an edge-collection with given name exists within the graph.
func (g *graph) VertexCollectionExists(ctx context.Context, name string) (bool, error) {
	req, err := g.conn.NewRequest("GET", path.Join(g.relPath(), "vertex"))
	if err != nil {
		return false, WithStack(err)
	}
	resp, err := g.conn.Do(ctx, req)
	if err != nil {
		return false, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return false, WithStack(err)
	}
	var data listVertexCollectionResponse
	if err := resp.ParseBody("", &data); err != nil {
		return false, WithStack(err)
	}
	for _, n := range data.Collections {
		if n == name {
			return true, nil
		}
	}
	return false, nil
}

// VertexCollections returns all edge collections of this graph
func (g *graph) VertexCollections(ctx context.Context) ([]Collection, error) {
	req, err := g.conn.NewRequest("GET", path.Join(g.relPath(), "vertex"))
	if err != nil {
		return nil, WithStack(err)
	}
	resp, err := g.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	var data listVertexCollectionResponse
	if err := resp.ParseBody("", &data); err != nil {
		return nil, WithStack(err)
	}
	result := make([]Collection, 0, len(data.Collections))
	for _, name := range data.Collections {
		ec, err := newVertexCollection(name, g)
		if err != nil {
			return nil, WithStack(err)
		}
		result = append(result, ec)
	}
	return result, nil
}

// collection: The name of the edge collection to be used.
// from: contains the names of one or more vertex collections that can contain source vertices.
// to: contains the names of one or more edge collections that can contain target vertices.
func (g *graph) CreateVertexCollection(ctx context.Context, collection string) (Collection, error) {
	req, err := g.conn.NewRequest("POST", path.Join(g.relPath(), "vertex"))
	if err != nil {
		return nil, WithStack(err)
	}
	input := struct {
		Collection string `json:"collection,omitempty"`
	}{
		Collection: collection,
	}
	if _, err := req.SetBody(input); err != nil {
		return nil, WithStack(err)
	}
	resp, err := g.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(201, 202); err != nil {
		return nil, WithStack(err)
	}
	ec, err := newVertexCollection(collection, g)
	if err != nil {
		return nil, WithStack(err)
	}
	return ec, nil
}

// CreateVertexCollectionWithOptions creates a vertex collection in the graph
func (g *graph) CreateVertexCollectionWithOptions(ctx context.Context, collection string, options CreateVertexCollectionOptions) (Collection, error) {
	req, err := g.conn.NewRequest("POST", path.Join(g.relPath(), "vertex"))
	if err != nil {
		return nil, WithStack(err)
	}
	input := struct {
		Collection string                        `json:"collection,omitempty"`
		Options    CreateVertexCollectionOptions `json:"options,omitempty"`
	}{
		Collection: collection,
		Options:    options,
	}
	if _, err := req.SetBody(input); err != nil {
		return nil, WithStack(err)
	}
	resp, err := g.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(201, 202); err != nil {
		return nil, WithStack(err)
	}
	ec, err := newVertexCollection(collection, g)
	if err != nil {
		return nil, WithStack(err)
	}
	return ec, nil
}
