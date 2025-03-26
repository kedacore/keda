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

const (
	SatelliteGraph = -100
)

// DatabaseGraphs provides access to all graphs in a single database.
type DatabaseGraphs interface {
	// Graph opens a connection to an existing graph within the database.
	// If no graph with given name exists, an NotFoundError is returned.
	Graph(ctx context.Context, name string) (Graph, error)

	// GraphExists returns true if a graph with given name exists within the database.
	GraphExists(ctx context.Context, name string) (bool, error)

	// Graphs returns a list of all graphs in the database.
	Graphs(ctx context.Context) ([]Graph, error)

	// CreateGraph creates a new graph with given name and options, and opens a connection to it.
	// If a graph with given name already exists within the database, a DuplicateError is returned.
	//
	// Deprecated: since ArangoDB 3.9 - please use CreateGraphV2 instead
	CreateGraph(ctx context.Context, name string, options *CreateGraphOptions) (Graph, error)

	// CreateGraphV2 creates a new graph with given name and options, and opens a connection to it.
	// If a graph with given name already exists within the database, a DuplicateError is returned.
	CreateGraphV2(ctx context.Context, name string, options *CreateGraphOptions) (Graph, error)
}

// CreateGraphOptions contains options that customize the creating of a graph.
type CreateGraphOptions struct {
	// OrphanVertexCollections is an array of additional vertex collections used in the graph.
	// These are vertices for which there are no edges linking these vertices with anything.
	OrphanVertexCollections []string
	// EdgeDefinitions is an array of edge definitions for the graph.
	EdgeDefinitions []EdgeDefinition
	// IsSmart defines if the created graph should be smart.
	// This only has effect in Enterprise Edition.
	IsSmart bool
	// SmartGraphAttribute is the attribute name that is used to smartly shard the vertices of a graph.
	// Every vertex in this Graph has to have this attribute.
	// Cannot be modified later.
	SmartGraphAttribute string
	// NumberOfShards is the number of shards that is used for every collection within this graph.
	// Cannot be modified later.
	NumberOfShards int
	// ReplicationFactor is the number of replication factor that is used for every collection within this graph.
	// Cannot be modified later.
	ReplicationFactor int
	// WriteConcern is the number of min replication factor that is used for every collection within this graph.
	// Cannot be modified later.
	WriteConcern int
	// IsDisjoint set isDisjoint flag for Graph. Required ArangoDB 3.7+
	IsDisjoint bool
	// Satellites contains an array of collection names that will be used to create SatelliteCollections for a Hybrid (Disjoint) SmartGraph (Enterprise Edition only)
	// Requires ArangoDB 3.9+
	Satellites []string `json:"satellites,omitempty"`
}

// EdgeDefinition contains all information needed to define a single edge in a graph.
type EdgeDefinition struct {
	// The name of the edge collection to be used.
	Collection string `json:"collection"`
	// To contains the names of one or more vertex collections that can contain target vertices.
	To []string `json:"to"`
	// From contains the names of one or more vertex collections that can contain source vertices.
	From []string `json:"from"`
	// Options contains optional parameters
	Options CreateEdgeCollectionOptions `json:"options,omitempty"`
}
