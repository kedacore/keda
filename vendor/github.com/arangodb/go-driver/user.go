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

// User provides access to a single user of a single server / cluster of servers.
type User interface {
	// Name returns the name of the user.
	Name() string

	//  Is this an active user?
	IsActive() bool

	// Is a password change for this user needed?
	IsPasswordChangeNeeded() bool

	// Get extra information about this user that was passed during its creation/update/replacement
	Extra(result interface{}) error

	// Remove removes the user.
	// If the user does not exist, a NotFoundError is returned.
	Remove(ctx context.Context) error

	// Update updates individual properties of the user.
	// If the user does not exist, a NotFoundError is returned.
	Update(ctx context.Context, options UserOptions) error

	// Replace replaces all properties of the user.
	// If the user does not exist, a NotFoundError is returned.
	Replace(ctx context.Context, options UserOptions) error

	// AccessibleDatabases returns a list of all databases that can be accessed (read/write or read-only) by this user.
	AccessibleDatabases(ctx context.Context) ([]Database, error)

	// SetDatabaseAccess sets the access this user has to the given database.
	// Pass a `nil` database to set the default access this user has to any new database.
	// This function requires ArangoDB 3.2 and up for access value `GrantReadOnly`.
	SetDatabaseAccess(ctx context.Context, db Database, access Grant) error

	// GetDatabaseAccess gets the access rights for this user to the given database.
	// Pass a `nil` database to get the default access this user has to any new database.
	// This function requires ArangoDB 3.2 and up.
	// By default this function returns the "effective" grant.
	// To return the "configured" grant, pass a context configured with `WithConfigured`.
	// This distinction is only relevant in ArangoDB 3.3 in the context of a readonly database.
	GetDatabaseAccess(ctx context.Context, db Database) (Grant, error)

	// RemoveDatabaseAccess removes the access this user has to the given database.
	// As a result the users access falls back to its default access.
	// If you remove default access (db==`nil`) for a user (and there are no specific access
	// rules for a database), the user's access falls back to no-access.
	// Pass a `nil` database to set the default access this user has to any new database.
	// This function requires ArangoDB 3.2 and up.
	RemoveDatabaseAccess(ctx context.Context, db Database) error

	// SetCollectionAccess sets the access this user has to a collection.
	// If you pass a `Collection`, it will set access for that collection.
	// If you pass a `Database`, it will set the default collection access for that database.
	// If you pass `nil`, it will set the default collection access for the default database.
	// This function requires ArangoDB 3.2 and up.
	SetCollectionAccess(ctx context.Context, col AccessTarget, access Grant) error

	// GetCollectionAccess gets the access rights for this user to the given collection.
	// If you pass a `Collection`, it will get access for that collection.
	// If you pass a `Database`, it will get the default collection access for that database.
	// If you pass `nil`, it will get the default collection access for the default database.
	// By default this function returns the "effective" grant.
	// To return the "configured" grant, pass a context configured with `WithConfigured`.
	// This distinction is only relevant in ArangoDB 3.3 in the context of a readonly database.
	GetCollectionAccess(ctx context.Context, col AccessTarget) (Grant, error)

	// RemoveCollectionAccess removes the access this user has to a collection.
	// If you pass a `Collection`, it will removes access for that collection.
	// If you pass a `Database`, it will removes the default collection access for that database.
	// If you pass `nil`, it will removes the default collection access for the default database.
	// This function requires ArangoDB 3.2 and up.
	RemoveCollectionAccess(ctx context.Context, col AccessTarget) error

	// GrantReadWriteAccess grants this user read/write access to the given database.
	//
	// Deprecated: use GrantDatabaseReadWriteAccess instead.
	GrantReadWriteAccess(ctx context.Context, db Database) error

	// RevokeAccess revokes this user access to the given database.
	//
	// Deprecated: use `SetDatabaseAccess(ctx, db, GrantNone)` instead.
	RevokeAccess(ctx context.Context, db Database) error
}

// Grant specifies access rights for an object
type Grant string

const (
	// GrantReadWrite indicates read/write access to an object
	GrantReadWrite Grant = "rw"
	// GrantReadOnly indicates read-only access to an object
	GrantReadOnly Grant = "ro"
	// GrantNone indicates no access to an object
	GrantNone Grant = "none"
)

// AccessTarget is implemented by Database & Collection and it used to
// get/set/remove collection permissions.
type AccessTarget interface {
	// Name returns the name of the database/collection.
	Name() string
}
