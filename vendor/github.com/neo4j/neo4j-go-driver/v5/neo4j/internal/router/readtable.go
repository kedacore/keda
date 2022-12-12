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

package router

import (
	"context"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/db"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/pool"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/log"
)

// Tries to read routing table from any of the specified routers using new or existing connection
// from the supplied pool.
func readTable(ctx context.Context, connectionPool Pool, routers []string, routerContext map[string]string, bookmarks []string,
	database, impersonatedUser string, boltLogger log.BoltLogger) (*db.RoutingTable, error) {
	// Preserve last error to be returned, set a default for case of no routers
	var err error = &ReadRoutingTableError{}

	// Try the routers one at the time since some of them might no longer support routing and we
	// can't force the pool to not re-use these when putting them back in the pool and retrieving
	// another db.
	for _, router := range routers {
		var conn db.Connection
		if conn, err = connectionPool.Borrow(ctx, []string{router}, true, boltLogger, pool.DefaultLivenessCheckThreshold); err != nil {
			// Check if failed due to context timing out
			if ctx.Err() != nil {
				return nil, wrapError(router, ctx.Err())
			}
			err = wrapError(router, err)
			continue
		}

		// We have a connection to the "router"
		var table *db.RoutingTable
		table, err = conn.GetRoutingTable(ctx, routerContext, bookmarks, database, impersonatedUser)
		connectionPool.Return(ctx, conn)
		if err == nil {
			return table, nil
		}
		err = wrapError(router, err)
	}
	return nil, err
}
