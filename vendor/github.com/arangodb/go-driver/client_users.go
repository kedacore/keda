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

// ClientUsers provides access to the users in a single arangodb database server, or an entire cluster of arangodb servers.
type ClientUsers interface {
	// User opens a connection to an existing user.
	// If no user with given name exists, an NotFoundError is returned.
	User(ctx context.Context, name string) (User, error)

	// UserExists returns true if a user with given name exists.
	UserExists(ctx context.Context, name string) (bool, error)

	// Users returns a list of all users found by the client.
	Users(ctx context.Context) ([]User, error)

	// CreateUser creates a new user with given name and opens a connection to it.
	// If a user with given name already exists, a Conflict error is returned.
	CreateUser(ctx context.Context, name string, options *UserOptions) (User, error)
}

// UserOptions contains options for creating a new user, updating or replacing a user.
type UserOptions struct {
	// The user password as a string. If not specified, it will default to an empty string.
	Password string `json:"passwd,omitempty"`
	// A flag indicating whether the user account should be activated or not. The default value is true. If set to false, the user won't be able to log into the database.
	Active *bool `json:"active,omitempty"`
	// A JSON object with extra user information. The data contained in extra will be stored for the user but not be interpreted further by ArangoDB.
	Extra interface{} `json:"extra,omitempty"`
}
