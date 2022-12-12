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

// Package neo4j provides required functionality to connect and execute statements against a Neo4j Database.
package neo4j

import (
	"context"
	"fmt"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/db"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/racing"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/log"
	"net/url"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/connector"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/pool"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/router"
)

// AccessMode defines modes that routing driver decides to which cluster member
// a connection should be opened.
type AccessMode int

const (
	// AccessModeWrite tells the driver to use a connection to 'Leader'
	AccessModeWrite AccessMode = 0
	// AccessModeRead tells the driver to use a connection to one of the 'Follower' or 'Read Replica'.
	AccessModeRead AccessMode = 1
)

// DriverWithContext represents a pool(s) of connections to a neo4j server or cluster. It's
// safe for concurrent use.
type DriverWithContext interface {
	// Target returns the url this driver is bootstrapped
	Target() url.URL
	// NewSession creates a new session based on the specified session configuration.
	NewSession(ctx context.Context, config SessionConfig) SessionWithContext
	// VerifyConnectivity checks that the driver can connect to a remote server or cluster by
	// establishing a network connection with the remote. Returns nil if successful
	// or error describing the problem.
	VerifyConnectivity(ctx context.Context) error
	// Close the driver and all underlying connections
	Close(ctx context.Context) error
	// IsEncrypted determines whether the driver communication with the server
	// is encrypted. This is a static check. The function can also be called on
	// a closed Driver.
	IsEncrypted() bool
	// GetServerInfo attempts to obtain server information from the target Neo4j
	// deployment
	GetServerInfo(ctx context.Context) (ServerInfo, error)
}

// NewDriverWithContext is the entry point to the neo4j driver to create an instance of a Driver. It is the first function to
// be called in order to establish a connection to a neo4j database. It requires a Bolt URI and an authentication
// token as parameters and can also take optional configuration function(s) as variadic parameters.
//
// In order to connect to a single instance database, you need to pass a URI with scheme 'bolt', 'bolt+s' or 'bolt+ssc'.
//
//	driver, err = NewDriverWithContext("bolt://db.server:7687", BasicAuth(username, password))
//
// In order to connect to a causal cluster database, you need to pass a URI with scheme 'neo4j', 'neo4j+s' or 'neo4j+ssc'
// and its host part set to be one of the core cluster members.
//
//	driver, err = NewDriverWithContext("neo4j://core.db.server:7687", BasicAuth(username, password))
//
// You can override default configuration options by providing a configuration function(s)
//
//	driver, err = NewDriverWithContext(uri, BasicAuth(username, password), function (config *Config) {
//		config.MaxConnectionPoolSize = 10
//	})
func NewDriverWithContext(target string, auth AuthToken, configurers ...func(*Config)) (DriverWithContext, error) {
	parsed, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	d := driverWithContext{target: parsed, mut: racing.NewMutex()}

	routing := true
	d.connector.Network = "tcp"
	address := parsed.Host
	switch parsed.Scheme {
	case "bolt":
		routing = false
		d.connector.SkipEncryption = true
	case "bolt+unix":
		// bolt+unix://<path to socket>
		routing = false
		d.connector.SkipEncryption = true
		d.connector.Network = "unix"
		if parsed.Host != "" {
			return nil, &UsageError{
				Message: fmt.Sprintf("Host part should be empty for scheme %s", parsed.Scheme),
			}
		}
		address = parsed.Path
	case "bolt+s":
		routing = false
	case "bolt+ssc":
		d.connector.SkipVerify = true
		routing = false
	case "neo4j":
		d.connector.SkipEncryption = true
	case "neo4j+ssc":
		d.connector.SkipVerify = true
	case "neo4j+s":
	default:
		return nil, &UsageError{
			Message: fmt.Sprintf("URI scheme %s is not supported", parsed.Scheme),
		}
	}

	if parsed.Host != "" && parsed.Port() == "" {
		address += ":7687"
		parsed.Host = address
	}

	if !routing && len(parsed.RawQuery) > 0 {
		return nil, &UsageError{
			Message: fmt.Sprintf("Routing context is not supported for URL scheme %s", parsed.Scheme),
		}
	}

	// Apply client hooks for setting up configuration
	d.config = defaultConfig()
	for _, configurer := range configurers {
		configurer(d.config)
	}
	if err := validateAndNormaliseConfig(d.config); err != nil {
		return nil, err
	}

	// Setup logging
	d.log = d.config.Log
	if d.log == nil {
		// Default to void logger
		d.log = &log.Void{}
	}
	d.logId = log.NewId()

	routingContext, err := routingContextFromUrl(routing, parsed)
	if err != nil {
		return nil, err
	}

	// Continue to setup connector
	d.connector.DialTimeout = d.config.SocketConnectTimeout
	d.connector.SocketKeepAlive = d.config.SocketKeepalive
	d.connector.UserAgent = d.config.UserAgent
	//lint:ignore SA1019 RootCAs is still supported until 6.0
	d.connector.RootCAs = d.config.RootCAs
	d.connector.TlsConfig = d.config.TlsConfig
	d.connector.Log = d.log
	d.connector.Auth = auth.tokens
	d.connector.RoutingContext = routingContext

	// Let the pool use the same log ID as the driver to simplify log reading.
	d.pool = pool.New(d.config.MaxConnectionPoolSize, d.config.MaxConnectionLifetime, d.connector.Connect, d.log, d.logId)

	if !routing {
		d.router = &directRouter{address: address}
	} else {
		var routersResolver func() []string
		addressResolverHook := d.config.AddressResolver
		if addressResolverHook != nil {
			routersResolver = func() []string {
				addresses := addressResolverHook(parsed)
				servers := make([]string, len(addresses))
				for i, a := range addresses {
					servers[i] = fmt.Sprintf("%s:%s", a.Hostname(), a.Port())
				}
				return servers
			}
		}
		// Let the router use the same log ID as the driver to simplify log reading.
		d.router = router.New(address, routersResolver, routingContext, d.pool, d.log, d.logId)
	}

	d.log.Infof(log.Driver, d.logId, "Created { target: %s }", address)
	return &d, nil
}

const routingContextAddressKey = "address"

func routingContextFromUrl(useRouting bool, u *url.URL) (map[string]string, error) {
	if !useRouting {
		return nil, nil
	}
	queryValues := u.Query()
	routingContext := make(map[string]string, len(queryValues)+1 /*For address*/)
	for k, vs := range queryValues {
		if len(vs) > 1 {
			return nil, &UsageError{
				Message: fmt.Sprintf("Duplicated routing context key '%s'", k),
			}
		}
		if len(vs) == 0 {
			return nil, &UsageError{
				Message: fmt.Sprintf("Empty routing context key '%s'", k),
			}
		}
		v := vs[0]
		v = strings.TrimSpace(v)
		if len(v) == 0 {
			return nil, &UsageError{
				Message: fmt.Sprintf("Empty routing context value for key '%s'", k),
			}
		}
		if k == routingContextAddressKey {
			return nil, &UsageError{Message: fmt.Sprintf("Illegal key '%s' for routing context", k)}
		}
		routingContext[k] = v
	}
	routingContext[routingContextAddressKey] = u.Host
	return routingContext, nil
}

type sessionRouter interface {
	// Readers returns the list of servers that can serve reads on the requested database.
	// note: bookmarks are lazily supplied, only when a new routing table needs to be fetched
	// this is needed because custom bookmark managers may provide bookmarks from external systems
	// they should not be called when it is not needed (e.g. when a routing table is cached)
	Readers(ctx context.Context, bookmarks func(context.Context) ([]string, error), database string, boltLogger log.BoltLogger) ([]string, error)
	// Writers returns the list of servers that can serve writes on the requested database.
	// note: bookmarks are lazily supplied, see Readers documentation to learn why
	Writers(ctx context.Context, bookmarks func(context.Context) ([]string, error), database string, boltLogger log.BoltLogger) ([]string, error)
	// GetNameOfDefaultDatabase returns the name of the default database for the specified user.
	// The correct database name is needed when requesting readers or writers.
	// the bookmarks are eagerly provided since this method always fetches a new routing table
	GetNameOfDefaultDatabase(ctx context.Context, bookmarks []string, user string, boltLogger log.BoltLogger) (string, error)
	Invalidate(ctx context.Context, database string) error
	CleanUp(ctx context.Context) error
	InvalidateWriter(ctx context.Context, name string, server string) error
	InvalidateReader(ctx context.Context, name string, server string) error
}

type driverWithContext struct {
	target    *url.URL
	config    *Config
	pool      *pool.Pool
	mut       racing.Mutex
	connector connector.Connector
	router    sessionRouter
	logId     string
	log       log.Logger
}

func (d *driverWithContext) Target() url.URL {
	return *d.target
}

func (d *driverWithContext) NewSession(ctx context.Context, config SessionConfig) SessionWithContext {
	if config.DatabaseName == "" {
		config.DatabaseName = db.DefaultDatabase
	}

	if !d.mut.TryLock(ctx) {
		return &erroredSessionWithContext{
			err: racing.LockTimeoutError("could not acquire lock in time when creating session")}
	}
	defer d.mut.Unlock()
	if d.pool == nil {
		return &erroredSessionWithContext{
			err: &UsageError{Message: "Trying to create session on closed driver"}}
	}
	return newSessionWithContext(d.config, config, d.router, d.pool, d.log)
}

func (d *driverWithContext) VerifyConnectivity(ctx context.Context) error {
	_, err := d.GetServerInfo(ctx)
	return err
}

func (d *driverWithContext) IsEncrypted() bool {
	return !d.connector.SkipEncryption
}

func (d *driverWithContext) GetServerInfo(ctx context.Context) (_ ServerInfo, err error) {
	session := d.NewSession(ctx, SessionConfig{})
	defer func() {
		err = deferredClose(ctx, session, err)
	}()
	return session.getServerInfo(ctx)
}

func (d *driverWithContext) Close(ctx context.Context) error {
	if !d.mut.TryLock(ctx) {
		return racing.LockTimeoutError("could not acquire lock in time when closing driver")
	}
	defer d.mut.Unlock()
	// Safeguard against closing more than once
	if d.pool != nil {
		if err := d.pool.Close(ctx); err != nil {
			return err
		}
	}
	d.pool = nil
	d.log.Infof(log.Driver, d.logId, "Closed")
	return nil
}
