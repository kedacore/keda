/*
 * Copyright (c) "Neo4j"
 * Neo4j Sweden AB [https://neo4j.com]
 *
 * This file is part of Neo4j.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

// Package dbtype contains definitions of supported database types.
package dbtype

// Node represents a node in the neo4j graph database
type Node struct {
	// Deprecated: Id is deprecated and will be removed in 6.0. Use ElementId instead.
	Id        int64          // Id of this Node.
	ElementId string         // ElementId of this Node.
	Labels    []string       // Labels attached to this Node.
	Props     map[string]any // Properties of this Node.
}

// Relationship represents a relationship in the neo4j graph database
type Relationship struct {
	// Deprecated: Id is deprecated and will be removed in 6.0. Use ElementId instead.
	Id        int64  // Id of this Relationship.
	ElementId string // ElementId of this Relationship.
	// Deprecated: StartId is deprecated and will be removed in 6.0. Use StartElementId instead.
	StartId        int64  // Id of the start Node of this Relationship.
	StartElementId string // ElementId of the start Node of this Relationship.
	// Deprecated: EndId is deprecated and will be removed in 6.0. Use EndElementId instead.
	EndId        int64          // Id of the end Node of this Relationship.
	EndElementId string         // ElementId of the end Node of this Relationship.
	Type         string         // Type of this Relationship.
	Props        map[string]any // Properties of this Relationship.
}

// Path represents a directed sequence of relationships between two nodes.
// This generally represents a traversal or walk through a graph and maintains a direction separate from that of any
// relationships traversed. It is allowed to be of size 0, meaning there are no relationships in it. In this case,
// it contains only a single node which is both the start and the end of the path.
type Path struct {
	Nodes         []Node // All the nodes in the path.
	Relationships []Relationship
}
