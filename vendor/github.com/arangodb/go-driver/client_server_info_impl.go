//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
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

import (
	"context"
)

// Version returns version information from the connected database server.
func (c *client) Version(ctx context.Context) (VersionInfo, error) {
	req, err := c.conn.NewRequest("GET", "_api/version")
	if err != nil {
		return VersionInfo{}, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return VersionInfo{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return VersionInfo{}, WithStack(err)
	}
	var data VersionInfo
	if err := resp.ParseBody("", &data); err != nil {
		return VersionInfo{}, WithStack(err)
	}
	return data, nil
}

// roleResponse contains the response body of the `/admin/server/role` api.
type roleResponse struct {
	// Role of the server within a cluster
	Role string `json:"role,omitempty"`
	Mode string `json:"mode,omitempty"`
	ArangoError
}

// asServerRole converts the response into a ServerRole
func (r roleResponse) asServerRole(ctx context.Context, c *client) (ServerRole, error) {
	switch r.Role {
	case "SINGLE":
		switch r.Mode {
		case "resilient":
			if err := c.echo(ctx); IsNoLeader(err) {
				return ServerRoleSinglePassive, nil
			} else if err != nil {
				return ServerRoleUndefined, WithStack(err)
			}
			return ServerRoleSingleActive, nil
		default:
			return ServerRoleSingle, nil
		}
	case "PRIMARY":
		return ServerRoleDBServer, nil
	case "COORDINATOR":
		return ServerRoleCoordinator, nil
	case "AGENT":
		return ServerRoleAgent, nil
	case "UNDEFINED":
		return ServerRoleUndefined, nil
	default:
		return ServerRoleUndefined, nil
	}
}

// ServerRole returns the role of the server that answers the request.
func (c *client) ServerRole(ctx context.Context) (ServerRole, error) {
	req, err := c.conn.NewRequest("GET", "_admin/server/role")
	if err != nil {
		return ServerRoleUndefined, WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return ServerRoleUndefined, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return ServerRoleUndefined, WithStack(err)
	}
	var data roleResponse
	if err := resp.ParseBody("", &data); err != nil {
		return ServerRoleUndefined, WithStack(err)
	}
	role, err := data.asServerRole(ctx, c)
	if err != nil {
		return ServerRoleUndefined, WithStack(err)
	}
	return role, nil
}

type idResponse struct {
	ID string `json:"id,omitempty"`
}

// Gets the ID of this server in the cluster.
// An error is returned when calling this to a server that is not part of a cluster.
func (c *client) ServerID(ctx context.Context) (string, error) {
	req, err := c.conn.NewRequest("GET", "_admin/server/id")
	if err != nil {
		return "", WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return "", WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return "", WithStack(err)
	}
	var data idResponse
	if err := resp.ParseBody("", &data); err != nil {
		return "", WithStack(err)
	}
	return data.ID, nil
}

// clusterEndpoints returns the endpoints of a cluster.
func (c *client) echo(ctx context.Context) error {
	req, err := c.conn.NewRequest("GET", "_admin/echo")
	if err != nil {
		return WithStack(err)
	}
	applyContextSettings(ctx, req)
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return WithStack(err)
	}
	return nil
}
