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

// GraphVertexCollections provides access to all vertex collections of a single graph in a database.
type GraphVertexCollections interface {
	// VertexCollection opens a connection to an existing vertex-collection within the graph.
	// If no vertex-collection with given name exists, an NotFoundError is returned.
	// Note: When calling Remove on the returned Collection, the collection is removed from the graph. Not from the database.
	VertexCollection(ctx context.Context, name string) (Collection, error)

	// VertexCollectionExists returns true if an vertex-collection with given name exists within the graph.
	VertexCollectionExists(ctx context.Context, name string) (bool, error)

	// VertexCollections returns all vertex collections of this graph
	// Note: When calling Remove on any of the returned Collection's, the collection is removed from the graph. Not from the database.
	VertexCollections(ctx context.Context) ([]Collection, error)

	// CreateVertexCollection creates a vertex collection in the graph.
	// collection: The name of the vertex collection to be used.
	CreateVertexCollection(ctx context.Context, collection string) (Collection, error)

	// CreateVertexCollectionWithOptions creates a vertex collection in the graph
	CreateVertexCollectionWithOptions(ctx context.Context, collection string, options CreateVertexCollectionOptions) (Collection, error)
}

// CreateVertexCollectionOptions contains optional parameters for creating a new vertex collection
type CreateVertexCollectionOptions struct {
	// Satellites contains an array of collection names that will be used to create SatelliteCollections for a Hybrid (Disjoint) SmartGraph (Enterprise Edition only)
	Satellites []string `json:"satellites,omitempty"`
}
