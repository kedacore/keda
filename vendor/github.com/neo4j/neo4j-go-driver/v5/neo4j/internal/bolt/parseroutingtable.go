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

package bolt

import (
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/db"
	idb "github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/db"
)

// Parses a record assumed to contain a routing table into common DB API routing table struct
// Returns nil if error while parsing
func parseRoutingTableRecord(rec *db.Record) *idb.RoutingTable {
	ttl, ok := rec.Values[0].(int64)
	if !ok {
		return nil
	}
	listOfX, ok := rec.Values[1].([]any)
	if !ok {
		return nil
	}

	table := &idb.RoutingTable{
		TimeToLive: int(ttl),
	}

	for _, x := range listOfX {
		// Each x should be a map consisting of addresses and the role
		m, ok := x.(map[string]any)
		if !ok {
			return nil
		}
		addressesX, ok := m["addresses"].([]any)
		if !ok {
			return nil
		}
		addresses := make([]string, len(addressesX))
		for i, addrX := range addressesX {
			addr, ok := addrX.(string)
			if !ok {
				return nil
			}
			addresses[i] = addr
		}
		role, ok := m["role"].(string)
		if !ok {
			return nil
		}
		switch role {
		case "READ":
			table.Readers = addresses
		case "WRITE":
			table.Writers = addresses
		case "ROUTE":
			table.Routers = addresses
		}
	}
	return table
}
