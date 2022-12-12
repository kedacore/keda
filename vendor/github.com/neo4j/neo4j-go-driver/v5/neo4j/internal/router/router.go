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
	"errors"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/db"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/racing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j/log"
)

const missingWriterRetries = 100
const missingReaderRetries = 100

type databaseRouter struct {
	dueUnix int64
	table   *db.RoutingTable
}

// Router is thread safe
type Router struct {
	routerContext map[string]string
	pool          Pool
	dbRouters     map[string]*databaseRouter
	dbRoutersMut  racing.Mutex
	now           func() time.Time
	sleep         func(time.Duration)
	rootRouter    string
	getRouters    func() []string
	log           log.Logger
	logId         string
}

type Pool interface {
	// Borrow acquires a connection from the provided list of servers
	// If all connections are busy and the pool is full, calls to Borrow may wait for a connection to become idle
	// If a connection has been idle for longer than idlenessThreshold, it will be reset
	// to check if it's still alive.
	Borrow(ctx context.Context, servers []string, wait bool, boltLogger log.BoltLogger, idlenessThreshold time.Duration) (db.Connection, error)
	Return(ctx context.Context, c db.Connection) error
}

func New(rootRouter string, getRouters func() []string, routerContext map[string]string, pool Pool, logger log.Logger, logId string) *Router {
	r := &Router{
		rootRouter:    rootRouter,
		getRouters:    getRouters,
		routerContext: routerContext,
		pool:          pool,
		dbRouters:     make(map[string]*databaseRouter),
		dbRoutersMut:  racing.NewMutex(),
		now:           time.Now,
		sleep:         time.Sleep,
		log:           logger,
		logId:         logId,
	}
	r.log.Infof(log.Router, r.logId, "Created {context: %v}", routerContext)
	return r
}

func (r *Router) readTable(ctx context.Context, dbRouter *databaseRouter, bookmarks []string, database, impersonatedUser string, boltLogger log.BoltLogger) (*db.RoutingTable, error) {
	var (
		table *db.RoutingTable
		err   error
	)

	// Try last known set of routers if there are any
	if dbRouter != nil && len(dbRouter.table.Routers) > 0 {
		routers := dbRouter.table.Routers
		r.log.Infof(log.Router, r.logId, "Reading routing table for '%s' from previously known routers: %v", database, routers)
		table, err = readTable(ctx, r.pool, routers, r.routerContext, bookmarks, database, impersonatedUser, boltLogger)
	}

	// Try initial router if no routers or failed
	if table == nil {
		r.log.Infof(log.Router, r.logId, "Reading routing table from initial router: %s", r.rootRouter)
		table, err = readTable(ctx, r.pool, []string{r.rootRouter}, r.routerContext, bookmarks, database, impersonatedUser, boltLogger)
	}

	// Use hook to retrieve possibly different set of routers and retry
	if table == nil && r.getRouters != nil {
		routers := r.getRouters()
		r.log.Infof(log.Router, r.logId, "Reading routing table for '%s' from custom routers: %v", routers)
		table, err = readTable(ctx, r.pool, routers, r.routerContext, bookmarks, database, impersonatedUser, boltLogger)
	}

	if err != nil {
		r.log.Error(log.Router, r.logId, err)
		return nil, err
	}

	if table == nil {
		// Safeguard for logical error somewhere else
		err = errors.New("no error and no table")
		r.log.Error(log.Router, r.logId, err)
		return nil, err
	}
	return table, nil
}

func (r *Router) getOrReadTable(ctx context.Context, bookmarksFn func(context.Context) ([]string, error), database string, boltLogger log.BoltLogger) (*db.RoutingTable, error) {
	now := r.now()

	if !r.dbRoutersMut.TryLock(ctx) {
		return nil, racing.LockTimeoutError("could not acquire router lock in time when getting routing table")
	}
	defer r.dbRoutersMut.Unlock()

	dbRouter := r.dbRouters[database]
	if dbRouter != nil && now.Unix() < dbRouter.dueUnix {
		return dbRouter.table, nil
	}

	bookmarks, err := bookmarksFn(ctx)
	if err != nil {
		return nil, err
	}
	table, err := r.readTable(ctx, dbRouter, bookmarks, database, "", boltLogger)
	if err != nil {
		return nil, err
	}

	r.storeRoutingTable(database, table, now)

	return table, nil
}

func (r *Router) Readers(ctx context.Context, bookmarks func(context.Context) ([]string, error), database string, boltLogger log.BoltLogger) ([]string, error) {
	table, err := r.getOrReadTable(ctx, bookmarks, database, boltLogger)
	if err != nil {
		return nil, err
	}

	// During startup, we can get tables without any readers
	retries := missingReaderRetries
	for len(table.Readers) == 0 {
		retries--
		if retries == 0 {
			break
		}
		r.log.Infof(log.Router, r.logId, "Invalidating routing table, no readers")
		if err := r.Invalidate(ctx, table.DatabaseName); err != nil {
			return nil, err
		}
		r.sleep(100 * time.Millisecond)
		table, err = r.getOrReadTable(ctx, bookmarks, database, boltLogger)
		if err != nil {
			return nil, err
		}
	}
	if len(table.Readers) == 0 {
		return nil, wrapError(r.rootRouter, errors.New("no readers"))
	}

	return table.Readers, nil
}

func (r *Router) Writers(ctx context.Context, bookmarks func(context.Context) ([]string, error), database string, boltLogger log.BoltLogger) ([]string, error) {
	table, err := r.getOrReadTable(ctx, bookmarks, database, boltLogger)
	if err != nil {
		return nil, err
	}

	// During election, we can get tables without any writers
	retries := missingWriterRetries
	for len(table.Writers) == 0 {
		retries--
		if retries == 0 {
			break
		}
		r.log.Infof(log.Router, r.logId, "Invalidating routing table, no writers")
		if err := r.Invalidate(ctx, database); err != nil {
			return nil, err
		}
		r.sleep(100 * time.Millisecond)
		table, err = r.getOrReadTable(ctx, bookmarks, database, boltLogger)
		if err != nil {
			return nil, err
		}
	}
	if len(table.Writers) == 0 {
		return nil, wrapError(r.rootRouter, errors.New("no writers"))
	}

	return table.Writers, nil
}

func (r *Router) GetNameOfDefaultDatabase(ctx context.Context, bookmarks []string, user string, boltLogger log.BoltLogger) (string, error) {
	table, err := r.readTable(ctx, nil, bookmarks, db.DefaultDatabase, user, boltLogger)
	if err != nil {
		return "", err
	}
	// Store the fresh routing table as well to avoid another roundtrip to receive servers from session.
	now := r.now()
	if !r.dbRoutersMut.TryLock(ctx) {
		return "", racing.LockTimeoutError("could not acquire router lock in time when resolving home database")
	}
	defer r.dbRoutersMut.Unlock()
	r.storeRoutingTable(table.DatabaseName, table, now)
	return table.DatabaseName, err
}

func (r *Router) Context() map[string]string {
	return r.routerContext
}

func (r *Router) Invalidate(ctx context.Context, database string) error {
	r.log.Infof(log.Router, r.logId, "Invalidating routing table for '%s'", database)
	if !r.dbRoutersMut.TryLock(ctx) {
		return racing.LockTimeoutError("could not acquire router lock in time when invalidating database router")
	}
	defer r.dbRoutersMut.Unlock()
	// Reset due time to the 70s, this will make next access refresh the routing table using
	// last set of routers instead of the original one.
	dbRouter := r.dbRouters[database]
	if dbRouter != nil {
		dbRouter.dueUnix = 0
	}
	return nil
}

func (r *Router) InvalidateWriter(ctx context.Context, db string, server string) error {
	if !r.dbRoutersMut.TryLock(ctx) {
		return racing.LockTimeoutError("could not acquire router lock in time when getting routing table")
	}
	defer r.dbRoutersMut.Unlock()

	router := r.dbRouters[db]
	if router == nil {
		return nil
	}
	writers := router.table.Writers
	for i, writer := range writers {
		if writer == server {
			router.table.Writers = append(writers[0:i], writers[i+1:]...)
			return nil
		}
	}
	return nil
}

func (r *Router) InvalidateReader(ctx context.Context, db string, server string) error {
	if !r.dbRoutersMut.TryLock(ctx) {
		return racing.LockTimeoutError("could not acquire router lock in time when invalidating reader")
	}
	defer r.dbRoutersMut.Unlock()

	router := r.dbRouters[db]
	if router == nil {
		return nil
	}
	readers := router.table.Readers
	for i, reader := range readers {
		if reader == server {
			router.table.Readers = append(readers[0:i], readers[i+1:]...)
			return nil
		}
	}
	return nil
}

func (r *Router) CleanUp(ctx context.Context) error {
	r.log.Debugf(log.Router, r.logId, "Cleaning up")
	now := r.now().Unix()
	if !r.dbRoutersMut.TryLock(ctx) {
		return racing.LockTimeoutError("could not acquire router lock in time when invalidating reader")
	}
	defer r.dbRoutersMut.Unlock()

	for dbName, dbRouter := range r.dbRouters {
		if now > dbRouter.dueUnix {
			delete(r.dbRouters, dbName)
		}
	}
	return nil
}

func (r *Router) storeRoutingTable(database string, table *db.RoutingTable, now time.Time) {
	r.dbRouters[database] = &databaseRouter{
		table:   table,
		dueUnix: now.Add(time.Duration(table.TimeToLive) * time.Second).Unix(),
	}
	r.log.Debugf(log.Router, r.logId, "New routing table for '%s', TTL %d", database, table.TimeToLive)
}
