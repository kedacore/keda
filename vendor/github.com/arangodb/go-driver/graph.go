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

// Graph provides access to all edge & vertex collections of a single graph in a database.
type Graph interface {
	// Name returns the name of the graph.
	Name() string

	// Remove removes the entire graph.
	// If the graph does not exist, a NotFoundError is returned.
	Remove(ctx context.Context) error

	// IsSmart returns true of smart is smart. In case of Community Edition it is always false
	IsSmart() bool

	// IsSatellite returns true of smart is satellite. In case of Community Edition it is always false
	IsSatellite() bool

	// IsDisjoint return information if graph have isDisjoint flag set to true
	IsDisjoint() bool

	// GraphEdgeCollections Edge collection functions
	GraphEdgeCollections

	// GraphVertexCollections Vertex collection functions
	GraphVertexCollections

	// ID returns the id of the graph.
	ID() string

	// Key returns the key of the graph.
	Key() DocumentID

	// Rev returns the revision of the graph.
	Rev() string

	// EdgeDefinitions returns the edge definitions of the graph.
	EdgeDefinitions() []EdgeDefinition

	// SmartGraphAttribute returns the attributes of a smart graph if there are any.
	SmartGraphAttribute() string

	// MinReplicationFactor returns the minimum replication factor for the graph.
	MinReplicationFactor() int

	// NumberOfShards returns the number of shards for the graph.
	NumberOfShards() int

	// OrphanCollections returns the orphan collcetions of the graph.
	OrphanCollections() []string

	// ReplicationFactor returns the current replication factor.
	ReplicationFactor() int

	// WriteConcern returns the write concern setting of the graph.
	WriteConcern() int
}
