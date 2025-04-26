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

package driver

import (
	"context"
	"net/http"
	"path"
	"time"

	"github.com/arangodb/go-driver/util"
)

// NewClient creates a new Client based on the given config setting.
func NewClient(config ClientConfig) (Client, error) {
	if config.Connection == nil {
		return nil, WithStack(InvalidArgumentError{Message: "Connection is not set"})
	}
	conn := config.Connection
	if config.Authentication != nil {
		var err error
		conn, err = conn.SetAuthentication(config.Authentication)
		if err != nil {
			return nil, WithStack(err)
		}
	}

	c := &client{
		conn: conn,
	}
	if config.SynchronizeEndpointsInterval > 0 {
		go c.autoSynchronizeEndpoints(config.SynchronizeEndpointsInterval)
	}
	return c, nil
}

// client implements the Client interface.
type client struct {
	conn Connection
}

// Connection returns the connection used by this client
func (c *client) Connection() Connection {
	return c.conn
}

// SynchronizeEndpoints fetches all endpoints from an ArangoDB cluster and updates the
// connection to use those endpoints.
// When this client is connected to a single server, nothing happens.
// When this client is connected to a cluster of servers, the connection will be updated to reflect
// the layout of the cluster.
func (c *client) SynchronizeEndpoints(ctx context.Context) error {
	return c.SynchronizeEndpoints2(ctx, "")
}

// SynchronizeEndpoints2 fetches all endpoints from an ArangoDB cluster and updates the
// connection to use those endpoints.
// When this client is connected to a single server, nothing happens.
// When this client is connected to a cluster of servers, the connection will be updated to reflect
// the layout of the cluster.
// Compared to SynchronizeEndpoints, this function expects a database name as additional parameter.
// This database name is used to call `_db/<dbname>/_api/cluster/endpoints`. SynchronizeEndpoints uses
// the default database, i.e. `_system`. In the case the user does not have access to `_system`,
// SynchronizeEndpoints does not work with earlier versions of arangodb.
func (c *client) SynchronizeEndpoints2(ctx context.Context, dbname string) error {
	// Cluster mode, fetch endpoints
	cep, err := c.clusterEndpoints(ctx, dbname)
	if err != nil {
		// ignore Forbidden: automatic failover is not enabled errors
		if !IsArangoErrorWithErrorNum(err, ErrHttpForbidden, ErrHttpInternal, 0, ErrNotImplemented, ErrForbidden) {
			// 3.2 returns no error code, thus check for 0
			// 501 with ErrorNum 9 is in there since 3.7, earlier versions returned 403 and ErrorNum 11.
			return WithStack(err)
		}

		return nil
	}
	var endpoints []string
	for _, ep := range cep.Endpoints {
		endpoints = append(endpoints, util.FixupEndpointURLScheme(ep.Endpoint))
	}

	// Update connection
	if err := c.conn.UpdateEndpoints(endpoints); err != nil {
		return WithStack(err)
	}

	return nil
}

// Deprecated: should not be called in new code.
//
// autoSynchronizeEndpoints performs automatic endpoint synchronization.
func (c *client) autoSynchronizeEndpoints(interval time.Duration) {
	for {
		// SynchronizeEndpoints endpoints
		c.SynchronizeEndpoints(nil)

		// Wait a bit
		time.Sleep(interval)
	}
}

type clusterEndpointsResponse struct {
	Endpoints []clusterEndpoint `json:"endpoints,omitempty"`
}

type clusterEndpoint struct {
	Endpoint string `json:"endpoint,omitempty"`
}

// clusterEndpoints returns the endpoints of a cluster.
func (c *client) clusterEndpoints(ctx context.Context, dbname string) (clusterEndpointsResponse, error) {
	var url string
	if dbname == "" {
		url = "_api/cluster/endpoints"
	} else {
		url = path.Join("_db", pathEscape(dbname), "_api/cluster/endpoints")
	}
	req, err := c.conn.NewRequest("GET", url)
	if err != nil {
		return clusterEndpointsResponse{}, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return clusterEndpointsResponse{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return clusterEndpointsResponse{}, WithStack(err)
	}
	var data clusterEndpointsResponse
	if err := resp.ParseBody("", &data); err != nil {
		return clusterEndpointsResponse{}, WithStack(err)
	}
	return data, nil
}

// GetLogLevels returns log levels for topics.
func (c *client) GetLogLevels(ctx context.Context, opts *LogLevelsGetOptions) (LogLevels, error) {
	req, err := c.conn.NewRequest(http.MethodGet, "_admin/log/level")
	if err != nil {
		return nil, WithStack(err)
	}

	if opts != nil {
		if len(opts.ServerID) > 0 {
			req.SetQuery("serverId", string(opts.ServerID))
		}
	}

	applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(http.StatusOK); err != nil {
		return nil, WithStack(err)
	}

	result := make(LogLevels)
	if err := resp.ParseBody("", &result); err != nil {
		return nil, WithStack(err)
	}

	return result, nil
}

// SetLogLevels sets log levels for a given topics.
func (c *client) SetLogLevels(ctx context.Context, logLevels LogLevels, opts *LogLevelsSetOptions) error {
	req, err := c.conn.NewRequest(http.MethodPut, "_admin/log/level")
	if err != nil {
		return WithStack(err)
	}

	if opts != nil {
		if len(opts.ServerID) > 0 {
			req = req.SetQuery("serverId", string(opts.ServerID))
		}
	}

	if _, err := req.SetBody(logLevels); err != nil {
		return WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}

	if err := resp.CheckStatus(http.StatusOK); err != nil {
		return WithStack(err)
	}

	return nil
}

// GetLicense returns license of an ArangoDB deployment.
func (c *client) GetLicense(ctx context.Context) (License, error) {
	result := License{}
	req, err := c.conn.NewRequest(http.MethodGet, "_admin/license")
	if err != nil {
		return result, WithStack(err)
	}

	applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return result, WithStack(err)
	}
	if err := resp.CheckStatus(http.StatusOK); err != nil {
		return result, WithStack(err)
	}

	if err := resp.ParseBody("", &result); err != nil {
		return result, WithStack(err)
	}

	return result, nil
}
