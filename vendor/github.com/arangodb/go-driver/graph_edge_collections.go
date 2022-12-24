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

// GraphEdgeCollections provides access to all edge collections of a single graph in a database.
type GraphEdgeCollections interface {
	// EdgeCollection opens a connection to an existing edge-collection within the graph.
	// If no edge-collection with given name exists, an NotFoundError is returned.
	// Note: When calling Remove on the returned Collection, the collection is removed from the graph. Not from the database.
	EdgeCollection(ctx context.Context, name string) (Collection, VertexConstraints, error)

	// EdgeCollectionExists returns true if an edge-collection with given name exists within the graph.
	EdgeCollectionExists(ctx context.Context, name string) (bool, error)

	// EdgeCollections returns all edge collections of this graph
	// Note: When calling Remove on any of the returned Collection's, the collection is removed from the graph. Not from the database.
	EdgeCollections(ctx context.Context) ([]Collection, []VertexConstraints, error)

	// CreateEdgeCollection creates an edge collection in the graph.
	// collection: The name of the edge collection to be used.
	// constraints.From: contains the names of one or more vertex collections that can contain source vertices.
	// constraints.To: contains the names of one or more edge collections that can contain target vertices.
	CreateEdgeCollection(ctx context.Context, collection string, constraints VertexConstraints) (Collection, error)

	// CreateEdgeCollectionWithOptions creates an edge collection in the graph with additional options
	CreateEdgeCollectionWithOptions(ctx context.Context, collection string, constraints VertexConstraints, options CreateEdgeCollectionOptions) (Collection, error)

	// SetVertexConstraints modifies the vertex constraints of an existing edge collection in the graph.
	SetVertexConstraints(ctx context.Context, collection string, constraints VertexConstraints) error
}

// VertexConstraints limit the vertex collection you can use in an edge.
type VertexConstraints struct {
	// From contains names of vertex collection that are allowed to be used in the From part of an edge.
	From []string
	// To contains names of vertex collection that are allowed to be used in the To part of an edge.
	To []string
}

// CreateEdgeCollectionOptions contains optional parameters for creating a new edge collection
type CreateEdgeCollectionOptions struct {
	// Satellites contains an array of collection names that will be used to create SatelliteCollections for a Hybrid (Disjoint) SmartGraph (Enterprise Edition only)
	Satellites []string `json:"satellites,omitempty"`
}
