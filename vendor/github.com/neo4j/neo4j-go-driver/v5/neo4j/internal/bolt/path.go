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
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package bolt

import (
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
)

// Intermediate representation of part of path
type relNode struct {
	// Deprecated: id is deprecated and will be removed in 6.0. Use elementId instead.
	id        int64
	elementId string
	name      string
	props     map[string]any
}

// buildPath builds a path from Bolt representation
func buildPath(nodes []dbtype.Node, relNodes []*relNode, indexes []int) dbtype.Path {
	num := len(indexes) / 2
	if num == 0 {
		var path dbtype.Path
		if len(nodes) > 0 {
			// there could be a single, disconnected node
			path.Nodes = nodes
		}
		return path
	}
	rels := make([]dbtype.Relationship, 0, num)

	i := 0
	n1 := nodes[0]
	for num > 0 {
		relni := indexes[i]
		i++
		n2i := indexes[i]
		i++
		num--
		var reln *relNode
		var n1start bool
		if relni < 0 {
			reln = relNodes[(relni*-1)-1]
		} else {
			reln = relNodes[relni-1]
			n1start = true
		}
		n2 := nodes[n2i]

		rel := dbtype.Relationship{
			Id:        reln.id,
			ElementId: reln.elementId,
			Type:      reln.name,
			Props:     reln.props,
		}
		if n1start {
			//lint:ignore SA1019 Id, StartId and EndId are supported at least until 6.0
			rel.StartId, rel.EndId = n1.Id, n2.Id
			rel.StartElementId, rel.EndElementId = n1.ElementId, n2.ElementId
		} else {
			//lint:ignore SA1019 Id, StartId and EndId are supported at least until 6.0
			rel.StartId, rel.EndId = n2.Id, n1.Id
			rel.StartElementId, rel.EndElementId = n2.ElementId, n1.ElementId
		}
		rels = append(rels, rel)
		n1 = n2
	}

	return dbtype.Path{Nodes: nodes, Relationships: rels}
}
