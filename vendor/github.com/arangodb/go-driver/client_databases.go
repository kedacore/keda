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

package driver

import "context"

type DatabaseSharding string

const (
	DatabaseShardingSingle DatabaseSharding = "single"
	DatabaseShardingNone   DatabaseSharding = ""
)

// ClientDatabases provides access to the databases in a single arangodb database server, or an entire cluster of arangodb servers.
type ClientDatabases interface {
	// Database opens a connection to an existing database.
	// If no database with given name exists, an NotFoundError is returned.
	Database(ctx context.Context, name string) (Database, error)

	// DatabaseExists returns true if a database with given name exists.
	DatabaseExists(ctx context.Context, name string) (bool, error)

	// Databases returns a list of all databases found by the client.
	Databases(ctx context.Context) ([]Database, error)

	// AccessibleDatabases returns a list of all databases that can be accessed by the authenticated user.
	AccessibleDatabases(ctx context.Context) ([]Database, error)

	// CreateDatabase creates a new database with given name and opens a connection to it.
	// If the a database with given name already exists, a DuplicateError is returned.
	CreateDatabase(ctx context.Context, name string, options *CreateDatabaseOptions) (Database, error)
}

// CreateDatabaseOptions contains options that customize the creating of a database.
type CreateDatabaseOptions struct {
	// List of users to initially create for the new database. User information will not be changed for users that already exist.
	// If users is not specified or does not contain any users, a default user root will be created with an empty string password.
	// This ensures that the new database will be accessible after it is created.
	Users []CreateDatabaseUserOptions `json:"users,omitempty"`

	// Options database defaults
	Options CreateDatabaseDefaultOptions `json:"options,omitempty"`
}

// CreateDatabaseDefaultOptions contains options that change defaults for collections
type CreateDatabaseDefaultOptions struct {
	// Default replication factor for collections in database
	ReplicationFactor int `json:"replicationFactor,omitempty"`
	// Default write concern for collections in database
	WriteConcern int `json:"writeConcern,omitempty"`
	// Default sharding for collections in database
	Sharding DatabaseSharding `json:"sharding,omitempty"`
}

// CreateDatabaseUserOptions contains options for creating a single user for a database.
type CreateDatabaseUserOptions struct {
	// Loginname of the user to be created
	UserName string `json:"user,omitempty"`
	// The user password as a string. If not specified, it will default to an empty string.
	Password string `json:"passwd,omitempty"`
	// A flag indicating whether the user account should be activated or not. The default value is true. If set to false, the user won't be able to log into the database.
	Active *bool `json:"active,omitempty"`
	// A JSON object with extra user information. The data contained in extra will be stored for the user but not be interpreted further by ArangoDB.
	Extra interface{} `json:"extra,omitempty"`
}
