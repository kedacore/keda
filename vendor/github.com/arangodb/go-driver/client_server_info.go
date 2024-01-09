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

import "context"

// ClientServerInfo provides access to information about a single ArangoDB server.
// When your client uses multiple endpoints, it is undefined which server
// will respond to requests of this interface.
type ClientServerInfo interface {
	// Version returns version information from the connected database server.
	// Use WithDetails to configure a context that will include additional details in the return VersionInfo.
	Version(ctx context.Context) (VersionInfo, error)

	// ServerRole returns the role of the server that answers the request.
	ServerRole(ctx context.Context) (ServerRole, error)

	// Gets the ID of this server in the cluster.
	// An error is returned when calling this to a server that is not part of a cluster.
	ServerID(ctx context.Context) (string, error)
}

// ServerRole is the role of an arangod server
type ServerRole string

const (
	// ServerRoleSingle indicates that the server is a single-server instance
	ServerRoleSingle ServerRole = "Single"
	// ServerRoleSingleActive indicates that the server is a the leader of a single-server resilient pair
	ServerRoleSingleActive ServerRole = "SingleActive"
	// ServerRoleSinglePassive indicates that the server is a a follower of a single-server resilient pair
	ServerRoleSinglePassive ServerRole = "SinglePassive"
	// ServerRoleDBServer indicates that the server is a dbserver within a cluster
	ServerRoleDBServer ServerRole = "DBServer"
	// ServerRoleCoordinator indicates that the server is a coordinator within a cluster
	ServerRoleCoordinator ServerRole = "Coordinator"
	// ServerRoleAgent indicates that the server is an agent within a cluster
	ServerRoleAgent ServerRole = "Agent"
	// ServerRoleUndefined indicates that the role of the server cannot be determined
	ServerRoleUndefined ServerRole = "Undefined"
)
