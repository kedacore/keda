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

package cluster

import (
	"context"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	driver "github.com/arangodb/go-driver"
)

const (
	keyFollowLeaderRedirect driver.ContextKey = "arangodb-followLeaderRedirect"
)

// ConnectionConfig provides all configuration options for a cluster connection.
type ConnectionConfig struct {
	// DefaultTimeout is the timeout used by requests that have no timeout set in the given context.
	DefaultTimeout time.Duration
}

// ServerConnectionBuilder specifies a function called by the cluster connection when it
// needs to create an underlying connection to a specific endpoint.
type ServerConnectionBuilder func(endpoint string) (driver.Connection, error)

// NewConnection creates a new cluster connection to a cluster of servers.
// The given connections are existing connections to each of the servers.
func NewConnection(config ConnectionConfig, connectionBuilder ServerConnectionBuilder, endpoints []string) (driver.Connection, error) {
	if connectionBuilder == nil {
		return nil, driver.WithStack(driver.InvalidArgumentError{Message: "Must a connection builder"})
	}
	if len(endpoints) == 0 {
		return nil, driver.WithStack(driver.InvalidArgumentError{Message: "Must provide at least 1 endpoint"})
	}
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = defaultTimeout
	}
	cConn := &clusterConnection{
		connectionBuilder: connectionBuilder,
		defaultTimeout:    config.DefaultTimeout,
	}
	// Initialize endpoints
	if err := cConn.UpdateEndpoints(endpoints); err != nil {
		return nil, driver.WithStack(err)
	}
	return cConn, nil
}

const (
	defaultTimeout                   = 9 * time.Minute
	keyEndpoint    driver.ContextKey = "arangodb-endpoint"
)

type clusterConnection struct {
	connectionBuilder ServerConnectionBuilder
	servers           []driver.Connection
	endpoints         []string
	current           int
	mutex             sync.RWMutex
	defaultTimeout    time.Duration
	auth              driver.Authentication
}

// NewRequest creates a new request with given method and path.
func (c *clusterConnection) NewRequest(method, path string) (driver.Request, error) {
	c.mutex.RLock()
	servers := c.servers
	c.mutex.RUnlock()

	// It is assumed that all servers used the same protocol.
	if len(servers) > 0 {
		return servers[0].NewRequest(method, path)
	}
	return nil, driver.WithStack(driver.ArangoError{
		HasError:     true,
		Code:         http.StatusServiceUnavailable,
		ErrorMessage: "no servers available",
	})
}

// Do performs a given request, returning its response.
func (c *clusterConnection) Do(ctx context.Context, req driver.Request) (driver.Response, error) {
	followLeaderRedirect := true
	if ctx == nil {
		ctx = context.Background()
	} else {
		if v := ctx.Value(keyFollowLeaderRedirect); v != nil {
			if on, ok := v.(bool); ok {
				followLeaderRedirect = on
			}
		}
	}
	// Timeout management.
	// We take the given timeout and divide it in 3 so we allow for other servers
	// to give it a try if an earlier server fails.
	deadline, hasDeadline := ctx.Deadline()
	var timeout time.Duration
	if hasDeadline {
		timeout = deadline.Sub(time.Now())
	} else {
		timeout = c.defaultTimeout
	}

	var server driver.Connection
	var serverCount int
	var durationPerRequest time.Duration

	if v := ctx.Value(keyEndpoint); v != nil {
		if endpoint, ok := v.(string); ok {
			// Override pool to only specific server if it is found
			if s, ok := c.getSpecificServer(endpoint); ok {
				server = s
				durationPerRequest = timeout
				serverCount = 1
			}
		}
	}

	if server == nil {
		server, serverCount = c.getCurrentServer()
		timeoutDivider := math.Max(1.0, math.Min(3.0, float64(serverCount)))
		durationPerRequest = time.Duration(float64(timeout) / timeoutDivider)
	}

	attempt := 1
	for {
		// Send request to specific endpoint with a 1/3 timeout (so we get 3 attempts)
		serverCtx, cancel := context.WithTimeout(ctx, durationPerRequest)
		resp, err := server.Do(serverCtx, req)
		cancel()

		isNoLeaderResponse := false
		if err == nil && resp.StatusCode() == 503 {
			// Service unavailable, parse the body, perhaps this is a "no leader"
			// case where we have to failover.
			var aerr driver.ArangoError
			if perr := resp.ParseBody("", &aerr); perr == nil && aerr.HasError {
				if driver.IsNoLeader(aerr) {
					isNoLeaderResponse = true
					// Save error in case we have no more servers
					err = aerr
				}
			}
		}

		if !isNoLeaderResponse || !followLeaderRedirect {
			if err == nil {
				// We're done
				return resp, nil
			}
			// No success yet
			if driver.IsCanceled(err) {
				// Request was cancelled, we return directly.
				return nil, driver.WithStack(err)
			}
			// If we've completely written the request, we return the error,
			// otherwise we'll failover to a new server.
			if req.Written() {
				// Request has been written to network, do not failover
				if driver.IsArangoError(err) {
					// ArangoError, so we got an error response from server.
					return nil, driver.WithStack(err)
				}
				// Not an ArangoError, so it must be some kind of timeout, network ... error.
				return nil, driver.WithStack(&driver.ResponseError{Err: err})
			}
		}

		// Failed, try next server
		attempt++
		if attempt > serverCount {
			// A specific server was specified, no failover.
			// or
			// We've tried all servers. Giving up.
			return nil, driver.WithStack(err)
		}
		server = c.getNextServer()
	}
}

/*func printError(err error, indent string) {
	if err == nil {
		return
	}
	fmt.Printf("%sGot %T %+v\n", indent, err, err)
	if xerr, ok := err.(*os.SyscallError); ok {
		printError(xerr.Err, indent+"  ")
	} else if xerr, ok := err.(*net.OpError); ok {
		printError(xerr.Err, indent+"  ")
	} else if xerr, ok := err.(*url.Error); ok {
		printError(xerr.Err, indent+"  ")
	}
}*/

// Unmarshal unmarshals the given raw object into the given result interface.
func (c *clusterConnection) Unmarshal(data driver.RawObject, result interface{}) error {
	c.mutex.RLock()
	servers := c.servers
	c.mutex.RUnlock()

	if len(servers) > 0 {
		if err := c.servers[0].Unmarshal(data, result); err != nil {
			return driver.WithStack(err)
		}
		return nil
	}
	return driver.WithStack(driver.ArangoError{
		HasError:     true,
		Code:         http.StatusServiceUnavailable,
		ErrorMessage: "no servers available",
	})
}

// Endpoints returns the endpoints used by this connection.
func (c *clusterConnection) Endpoints() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var result []string
	for _, s := range c.servers {
		result = append(result, s.Endpoints()...)
	}
	return result
}

// UpdateEndpoints reconfigures the connection to use the given endpoints.
func (c *clusterConnection) UpdateEndpoints(endpoints []string) error {
	if len(endpoints) == 0 {
		return driver.WithStack(driver.InvalidArgumentError{Message: "Must provide at least 1 endpoint"})
	}
	sort.Strings(endpoints)
	if strings.Join(endpoints, ",") == strings.Join(c.endpoints, ",") {
		// No changes
		return nil
	}

	// Create new connections
	servers := make([]driver.Connection, 0, len(endpoints))
	for _, ep := range endpoints {
		conn, err := c.connectionBuilder(ep)
		if err != nil {
			return driver.WithStack(err)
		}
		if c.auth != nil {
			conn, err = conn.SetAuthentication(c.auth)
			if err != nil {
				return driver.WithStack(err)
			}
		}
		servers = append(servers, conn)
	}

	// Swap connections
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.servers = servers
	c.endpoints = endpoints
	c.current = 0

	return nil
}

// Configure the authentication used for this connection.
func (c *clusterConnection) SetAuthentication(auth driver.Authentication) (driver.Connection, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Configure underlying servers
	newServerConnections := make([]driver.Connection, len(c.servers))
	for i, s := range c.servers {
		authConn, err := s.SetAuthentication(auth)
		if err != nil {
			return nil, driver.WithStack(err)
		}
		newServerConnections[i] = authConn
	}

	// Save authentication
	c.auth = auth
	c.servers = newServerConnections

	return c, nil
}

// Protocols returns all protocols used by this connection.
func (c *clusterConnection) Protocols() driver.ProtocolSet {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var result driver.ProtocolSet
	for _, s := range c.servers {
		for _, p := range s.Protocols() {
			if !result.Contains(p) {
				result = append(result, p)
			}
		}
	}
	return result
}

// getCurrentServer returns the currently used server and number of servers.
func (c *clusterConnection) getCurrentServer() (driver.Connection, int) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.servers[c.current], len(c.servers)
}

// getSpecificServer returns the server with the given endpoint.
func (c *clusterConnection) getSpecificServer(endpoint string) (driver.Connection, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	for _, s := range c.servers {
		for _, x := range s.Endpoints() {
			if x == endpoint {
				return s, true
			}
		}
	}

	// If endpoint is not found allow to use default connection pool - request will be routed thru coordinators
	return nil, false
}

// getNextServer changes the currently used server and returns the new server.
func (c *clusterConnection) getNextServer() driver.Connection {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.current = (c.current + 1) % len(c.servers)
	return c.servers[c.current]
}
