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

type graphDefinition struct {
	Name        string `json:"name"`
	IsSmart     bool   `json:"isSmart"`
	IsSatellite bool   `json:"isSatellite"`
	IsDisjoint  bool   `json:"isDisjoint,omitempty"`

	EdgeDefinitions []EdgeDefinition `json:"edgeDefinitions,omitempty"`

	NumberOfShards    int      `json:"numberOfShards,omitempty"`
	OrphanCollections []string `json:"orphanCollections,omitempty"`

	// Deprecated: use 'WriteConcern' instead.
	MinReplicationFactor int `json:"minReplicationFactor,omitempty"`
	WriteConcern         int `json:"writeConcern,omitempty"`

	// ReplicationFactor is the number of replication factor that is used for every collection within this graph.
	// Cannot be modified later.
	ReplicationFactor graphReplicationFactor `json:"replicationFactor,omitempty"`

	// This field must be set to the attribute that will be used for sharding or smart graphs.
	// All vertices are required to have this attribute set. Edges derive the attribute from their connected vertices.
	// This requires ArangoDB Enterprise Edition.
	SmartGraphAttribute string `json:"smartGraphAttribute,omitempty"`

	Initial    *string    `json:"initial,omitempty"`
	InitialCid int        `json:"initialCid,omitempty"`
	ID         string     `json:"_id"`
	Key        DocumentID `json:"_key"`
	Rev        string     `json:"_rev"`
}

type getGraphResponse struct {
	Graph graphDefinition `json:"graph"`
	ArangoError
}

// EdgeCollection opens a connection to an existing edge-collection within the graph.
// If no edge-collection with given name exists, an NotFoundError is returned.
func (g *graph) EdgeCollection(ctx context.Context, name string) (Collection, VertexConstraints, error) {
	req, err := g.conn.NewRequest("GET", g.relPath())
	if err != nil {
		return nil, VertexConstraints{}, WithStack(err)
	}
	resp, err := g.conn.Do(ctx, req)
	if err != nil {
		return nil, VertexConstraints{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, VertexConstraints{}, WithStack(err)
	}
	var data getGraphResponse
	if err := resp.ParseBody("", &data); err != nil {
		return nil, VertexConstraints{}, WithStack(err)
	}
	for _, n := range data.Graph.EdgeDefinitions {
		if n.Collection == name {
			ec, err := newEdgeCollection(name, g)
			if err != nil {
				return nil, VertexConstraints{}, WithStack(err)
			}
			constraints := VertexConstraints{
				From: n.From,
				To:   n.To,
			}
			return ec, constraints, nil
		}
	}
	return nil, VertexConstraints{}, WithStack(newArangoError(404, 0, "not found"))
}

// EdgeCollectionExists returns true if an edge-collection with given name exists within the graph.
func (g *graph) EdgeCollectionExists(ctx context.Context, name string) (bool, error) {
	req, err := g.conn.NewRequest("GET", g.relPath())
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
	var data getGraphResponse
	if err := resp.ParseBody("", &data); err != nil {
		return false, WithStack(err)
	}
	for _, n := range data.Graph.EdgeDefinitions {
		if n.Collection == name {
			return true, nil
		}
	}
	return false, nil
}

// EdgeCollections returns all edge collections of this graph
func (g *graph) EdgeCollections(ctx context.Context) ([]Collection, []VertexConstraints, error) {
	req, err := g.conn.NewRequest("GET", g.relPath())
	if err != nil {
		return nil, nil, WithStack(err)
	}
	resp, err := g.conn.Do(ctx, req)
	if err != nil {
		return nil, nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, nil, WithStack(err)
	}
	var data getGraphResponse
	if err := resp.ParseBody("", &data); err != nil {
		return nil, nil, WithStack(err)
	}
	result := make([]Collection, 0, len(data.Graph.EdgeDefinitions))
	constraints := make([]VertexConstraints, 0, len(data.Graph.EdgeDefinitions))
	for _, n := range data.Graph.EdgeDefinitions {
		ec, err := newEdgeCollection(n.Collection, g)
		if err != nil {
			return nil, nil, WithStack(err)
		}
		result = append(result, ec)
		constraints = append(constraints, VertexConstraints{
			From: n.From,
			To:   n.To,
		})
	}
	return result, constraints, nil
}

// collection: The name of the edge collection to be used.
// from: contains the names of one or more vertex collections that can contain source vertices.
// to: contains the names of one or more edge collections that can contain target vertices.
func (g *graph) CreateEdgeCollection(ctx context.Context, collection string, constraints VertexConstraints) (Collection, error) {
	req, err := g.conn.NewRequest("POST", path.Join(g.relPath(), "edge"))
	if err != nil {
		return nil, WithStack(err)
	}
	input := EdgeDefinition{
		Collection: collection,
		From:       constraints.From,
		To:         constraints.To,
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
	ec, err := newEdgeCollection(collection, g)
	if err != nil {
		return nil, WithStack(err)
	}
	return ec, nil
}

// CreateEdgeCollectionWithOptions creates an edge collection in the graph with additional options
func (g *graph) CreateEdgeCollectionWithOptions(ctx context.Context, collection string, constraints VertexConstraints, options CreateEdgeCollectionOptions) (Collection, error) {
	req, err := g.conn.NewRequest("POST", path.Join(g.relPath(), "edge"))
	if err != nil {
		return nil, WithStack(err)
	}
	input := EdgeDefinition{
		Collection: collection,
		From:       constraints.From,
		To:         constraints.To,
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
	ec, err := newEdgeCollection(collection, g)
	if err != nil {
		return nil, WithStack(err)
	}
	return ec, nil
}

// SetVertexConstraints modifies the vertex constraints of an existing edge collection in the graph.
func (g *graph) SetVertexConstraints(ctx context.Context, collection string, constraints VertexConstraints) error {
	req, err := g.conn.NewRequest("PUT", path.Join(g.relPath(), "edge", collection))
	if err != nil {
		return WithStack(err)
	}
	input := EdgeDefinition{
		Collection: collection,
		From:       constraints.From,
		To:         constraints.To,
	}
	if _, err := req.SetBody(input); err != nil {
		return WithStack(err)
	}
	resp, err := g.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(201, 202); err != nil {
		return WithStack(err)
	}
	return nil
}
