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

// Package pool handles the database connection pool.
package pool

// Thread safe

import (
	"container/list"
	"context"
	"fmt"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/bolt"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/db"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/racing"
	"math"
	"sort"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j/log"
)

// DefaultLivenessCheckThreshold disables the liveness check of connections
// Liveness checks are performed before a connection is deemed idle enough to be reset
const DefaultLivenessCheckThreshold = math.MaxInt64

type Connect func(context.Context, string, log.BoltLogger) (db.Connection, error)

type qitem struct {
	servers []string
	wakeup  chan bool
	conn    db.Connection
}

type Pool struct {
	maxSize    int
	maxAge     time.Duration
	connect    Connect
	servers    map[string]*server
	serversMut racing.Mutex
	queueMut   racing.Mutex
	queue      list.List
	now        func() time.Time
	closed     bool
	log        log.Logger
	logId      string
}

type serverPenalty struct {
	name    string
	penalty uint32
}

func New(maxSize int, maxAge time.Duration, connect Connect, logger log.Logger, logId string) *Pool {
	// Means infinite life, simplifies checking later on
	if maxAge <= 0 {
		maxAge = 1<<63 - 1
	}

	p := &Pool{
		maxSize:    maxSize,
		maxAge:     maxAge,
		connect:    connect,
		servers:    make(map[string]*server),
		serversMut: racing.NewMutex(),
		queueMut:   racing.NewMutex(),
		now:        time.Now,
		logId:      logId,
		log:        logger,
	}
	p.log.Infof(log.Pool, p.logId, "Created")
	return p
}

func (p *Pool) Close(ctx context.Context) error {
	p.closed = true
	// Cancel everything in the queue by just emptying at and let all callers timeout
	if !p.queueMut.TryLock(ctx) {
		return racing.LockTimeoutError("could not acquire queue lock in time when closing pool")
	}
	p.queue.Init()
	p.queueMut.Unlock()
	// Go through each server and close all connections to it
	if !p.serversMut.TryLock(ctx) {
		return racing.LockTimeoutError("could not acquire server lock in time when closing pool")
	}
	for n, s := range p.servers {
		s.closeAll(ctx)
		delete(p.servers, n)
	}
	p.serversMut.Unlock()
	p.log.Infof(log.Pool, p.logId, "Closed")
	return nil
}

func (p *Pool) anyExistingConnectionsOnServers(ctx context.Context, serverNames []string) (bool, error) {
	if !p.serversMut.TryLock(ctx) {
		return false, fmt.Errorf("could not acquire server lock in time when checking server connection")
	}
	defer p.serversMut.Unlock()
	for _, s := range serverNames {
		b := p.servers[s]
		if b != nil {
			if b.size() > 0 {
				return true, nil
			}
		}
	}
	return false, nil
}

// For testing
func (p *Pool) queueSize(ctx context.Context) (int, error) {
	if !p.queueMut.TryLock(ctx) {
		return -1, fmt.Errorf("could not acquire queue lock in time when checking queue size")
	}
	defer p.queueMut.Unlock()
	return p.queue.Len(), nil
}

// For testing
func (p *Pool) getServers(ctx context.Context) (map[string]*server, error) {
	if !p.serversMut.TryLock(ctx) {
		return nil, fmt.Errorf("could not acquire server lock in time when getting servers")
	}
	defer p.serversMut.Unlock()
	servers := make(map[string]*server)
	for k, v := range p.servers {
		servers[k] = v
	}
	return servers, nil
}

// CleanUp prunes all old connection on all the servers, this makes sure that servers
// gets removed from the map at some point in time. If there is a noticed
// failed connect still active  we should wait a while with removal to get
// prioritization right.
func (p *Pool) CleanUp(ctx context.Context) error {
	if !p.serversMut.TryLock(ctx) {
		return fmt.Errorf("could not acquire server lock in time when cleaning up pool")
	}
	defer p.serversMut.Unlock()
	now := p.now()
	for n, s := range p.servers {
		s.removeIdleOlderThan(ctx, now, p.maxAge)
		if s.size() == 0 && !s.hasFailedConnect(now) {
			delete(p.servers, n)
		}
	}
	return nil
}

func (p *Pool) getPenaltiesForServers(ctx context.Context, serverNames []string) ([]serverPenalty, error) {
	if !p.serversMut.TryLock(ctx) {
		return nil, fmt.Errorf("could not acquire server lock in time when computing server penalties")
	}
	defer p.serversMut.Unlock()

	// Retrieve penalty for each server
	penalties := make([]serverPenalty, len(serverNames))
	now := p.now()
	for i, n := range serverNames {
		s := p.servers[n]
		penalties[i].name = n
		if s != nil {
			// Make sure that we don't get a too old connection
			s.removeIdleOlderThan(ctx, now, p.maxAge)
			penalties[i].penalty = s.calculatePenalty(now)
		} else {
			penalties[i].penalty = newConnectionPenalty
		}
	}
	return penalties, nil
}

func (p *Pool) tryAnyIdle(ctx context.Context, serverNames []string, idlenessThreshold time.Duration) (db.Connection, error) {
	if !p.serversMut.TryLock(ctx) {
		return nil, racing.LockTimeoutError("could not acquire server lock in time when getting idle connection")
	}
	defer p.serversMut.Unlock()
	for _, serverName := range serverNames {
		srv := p.servers[serverName]
		if srv != nil {
			// Try to get an existing idle connection
			conn, _ := srv.getIdle(ctx, idlenessThreshold)
			if conn != nil {
				return conn, nil
			}
		}
	}
	return nil, nil
}

func (p *Pool) Borrow(ctx context.Context, serverNames []string, wait bool, boltLogger log.BoltLogger, idlenessThreshold time.Duration) (db.Connection, error) {
	if p.closed {
		return nil, &PoolClosed{}
	}
	p.log.Debugf(log.Pool, p.logId, "Trying to borrow connection from %s", serverNames)

	// Retrieve penalty for each server
	penalties, err := p.getPenaltiesForServers(ctx, serverNames)
	if err != nil {
		return nil, err
	}
	// Sort server penalties by lowest penalty
	sort.Slice(penalties, func(i, j int) bool {
		return penalties[i].penalty < penalties[j].penalty
	})

	var conn db.Connection
	for _, s := range penalties {
		conn, err = p.tryBorrow(ctx, s.name, boltLogger, idlenessThreshold)
		if err == nil {
			return conn, nil
		}

		if bolt.IsTimeoutError(err) {
			p.log.Warnf(log.Pool, p.logId, "Borrow time-out")
			return nil, &PoolTimeout{servers: serverNames, err: err}
		}
	}

	anyConnection, anyConnectionErr := p.anyExistingConnectionsOnServers(ctx, serverNames)
	if anyConnectionErr != nil {
		return nil, err
	}
	// If there are no connections for any of the servers, there is no point in waiting for anything
	// to be returned.
	if !anyConnection {
		p.log.Warnf(log.Pool, p.logId, "No server connection available to any of %v", serverNames)
		if err == nil {
			err = fmt.Errorf("no server connection available to any of %v", serverNames)
		}
		// Intentionally return last error from last connection attempt to make it easier to
		// see connection errors for users.
		return nil, err
	}

	if !wait {
		return nil, &PoolFull{servers: serverNames}
	}

	// Wait for a matching connection to be returned from another thread.
	if !p.queueMut.TryLock(ctx) {
		return nil, racing.LockTimeoutError("could not acquire lock in time when trying to get an idle connection")
	}
	// Ok, now that we own the queue we can add the item there but between getting the lock
	// and above check for an existing connection another thread might have returned a connection
	// so check again to avoid potentially starving this thread.
	conn, err = p.tryAnyIdle(ctx, serverNames, idlenessThreshold)
	if err != nil {
		p.queueMut.Unlock()
		return nil, err
	}
	if conn != nil {
		p.queueMut.Unlock()
		return conn, nil
	}
	// Add a waiting request to the queue and unlock the queue to let other threads that return
	// their connections access the queue.
	q := &qitem{
		servers: serverNames,
		wakeup:  make(chan bool),
	}
	e := p.queue.PushBack(q)
	p.queueMut.Unlock()

	p.log.Warnf(log.Pool, p.logId, "Borrow queued")
	// Wait for either a wake-up signal that indicates that we got a connection or a timeout.
	select {
	case <-q.wakeup:
		return q.conn, nil
	case <-ctx.Done():
		// TODO: provided ctx has reached deadline already - set some hardcoded timeout instead?
		if !p.queueMut.TryLock(context.Background()) {
			return nil, racing.LockTimeoutError("could not acquire lock in time when removing server wait request")
		}
		p.queue.Remove(e)
		p.queueMut.Unlock()
		if q.conn != nil {
			return q.conn, nil
		}
		p.log.Warnf(log.Pool, p.logId, "Borrow time-out")
		return nil, &PoolTimeout{err: ctx.Err(), servers: serverNames}
	}
}

func (p *Pool) tryBorrow(ctx context.Context, serverName string, boltLogger log.BoltLogger, idlenessThreshold time.Duration) (db.Connection, error) {
	// For now, lock complete servers map to avoid over connecting but with the downside
	// that long connect times will block connects to other servers as well. To fix this
	// we would need to add a pending connect to the server and lock per server.
	if !p.serversMut.TryLock(ctx) {
		return nil, racing.LockTimeoutError("could not acquire lock in time when borrowing a connection")
	}
	defer p.serversMut.Unlock()

	srv := p.servers[serverName]
	if srv != nil {
		for {
			connection, found := srv.getIdle(ctx, idlenessThreshold)
			if connection == nil && found {
				continue
			}
			if connection != nil {
				connection.SetBoltLogger(boltLogger)
				return connection, nil
			}
			if srv.size() >= p.maxSize {
				return nil, &PoolFull{servers: []string{serverName}}
			}
			break
		}
	} else {
		// Make sure that there is a server in the map
		srv = NewServer()
		p.servers[serverName] = srv
	}

	// No idle connection, try to connect
	p.log.Infof(log.Pool, p.logId, "Connecting to %s", serverName)
	c, err := p.connect(ctx, serverName, boltLogger)
	if err != nil {
		// Failed to connect, keep track that it was bad for a while
		srv.notifyFailedConnect(p.now())
		p.log.Warnf(log.Pool, p.logId, "Failed to connect to %s: %s", serverName, err)
		return nil, err
	}

	// Ok, got a connection, register the connection
	srv.registerBusy(c)
	srv.notifySuccessfulConnect()
	return c, nil
}

func (p *Pool) unreg(ctx context.Context, serverName string, c db.Connection, now time.Time) error {
	if !p.serversMut.TryLock(ctx) {
		return racing.LockTimeoutError("could not acquire server lock in time when unregistering server")
	}
	defer p.serversMut.Unlock()

	defer func() {
		// Close connection in another thread to avoid potential long blocking operation during close.
		go c.Close(ctx)
	}()

	server := p.servers[serverName]
	// Check for strange condition of not finding the server.
	if server == nil {
		p.log.Warnf(log.Pool, p.logId, "Server %s not found", serverName)
		return nil
	}

	server.unregisterBusy(c)
	if server.size() == 0 && !server.hasFailedConnect(now) {
		delete(p.servers, serverName)
	}
	return nil
}

func (p *Pool) removeIdleOlderThanOnServer(ctx context.Context, serverName string, now time.Time, maxAge time.Duration) error {
	if !p.serversMut.TryLock(ctx) {
		return racing.LockTimeoutError("could not acquire server lock in time before removing old idle connections")
	}
	defer p.serversMut.Unlock()
	server := p.servers[serverName]
	if server == nil {
		return nil
	}
	server.removeIdleOlderThan(ctx, now, maxAge)
	return nil
}

func (p *Pool) Return(ctx context.Context, c db.Connection) error {
	if p.closed {
		p.log.Warnf(log.Pool, p.logId, "Trying to return connection to closed pool")
		return nil
	}

	// Get the name of the server that the connection belongs to.
	serverName := c.ServerName()
	isAlive := c.IsAlive()
	p.log.Debugf(log.Pool, p.logId, "Returning connection to %s {alive:%t}", serverName, isAlive)

	// If the connection is dead, remove all other idle connections on the same server that older
	// or of the same age as the dead connection, otherwise perform normal cleanup of old connections
	maxAge := p.maxAge
	now := p.now()
	age := now.Sub(c.Birthdate())
	if !isAlive {
		// Since this connection has died all other connections that connected before this one
		// might also be bad, remove the idle ones.
		if age < maxAge {
			maxAge = age
		}
	}
	if err := p.removeIdleOlderThanOnServer(ctx, serverName, now, maxAge); err != nil {
		return err
	}

	// Prepare connection for being used by someone else if is alive.
	// Since reset could find the connection to be in a bad state or non-recoverable state,
	// make sure again that it really is alive.
	if isAlive {
		c.Reset(ctx)
		isAlive = c.IsAlive()
	}

	c.SetBoltLogger(nil)

	// Shouldn't return a too old or dead connection back to the pool
	if !isAlive || age >= p.maxAge {
		if err := p.unreg(ctx, serverName, c, now); err != nil {
			return err
		}
		p.log.Infof(log.Pool, p.logId, "Unregistering dead or too old connection to %s", serverName)
		// Returning here could cause a waiting thread to wait until it times out, to do it
		// properly we could wake up threads that waits on the server and wake them up if there
		// are no more connections to wait for.
		return nil
	}

	// Check if there is anyone in the queue waiting for a connection to this server.
	if !p.queueMut.TryLock(ctx) {
		return racing.LockTimeoutError("could not acquire queue lock when checking connection requests")
	}
	for e := p.queue.Front(); e != nil; e = e.Next() {
		queuedRequest := e.Value.(*qitem)
		// Check requested servers
		for _, rserver := range queuedRequest.servers {
			if rserver == serverName {
				queuedRequest.conn = c
				p.queue.Remove(e)
				p.queueMut.Unlock()
				queuedRequest.wakeup <- true
				return nil
			}
		}
	}
	p.queueMut.Unlock()

	// Just put it back in the list of idle connections for this server
	if !p.serversMut.TryLock(ctx) {
		return racing.LockTimeoutError("could not acquire server lock when putting connection back to idle")
	}
	defer p.serversMut.Unlock()
	server := p.servers[serverName]
	if server != nil { // Strange when server not found
		server.returnBusy(c)
	} else {
		p.log.Warnf(log.Pool, p.logId, "Server %s not found", serverName)
	}
	return nil
}
