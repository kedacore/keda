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

import (
	"context"
	"path"
)

// Database opens a connection to an existing database.
// If no database with given name exists, an NotFoundError is returned.
func (c *client) Database(ctx context.Context, name string) (Database, error) {
	escapedName := pathEscape(name)
	req, err := c.conn.NewRequest("GET", path.Join("_db", escapedName, "_api/database/current"))
	if err != nil {
		return nil, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	db, err := newDatabase(name, c.conn)
	if err != nil {
		return nil, WithStack(err)
	}
	return db, nil
}

// DatabaseExists returns true if a database with given name exists.
func (c *client) DatabaseExists(ctx context.Context, name string) (bool, error) {
	escapedName := pathEscape(name)
	req, err := c.conn.NewRequest("GET", path.Join("_db", escapedName, "_api/database/current"))
	if err != nil {
		return false, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return false, WithStack(err)
	}
	if err := resp.CheckStatus(200); err == nil {
		return true, nil
	} else if IsNotFound(err) {
		return false, nil
	} else {
		return false, WithStack(err)
	}
}

type getDatabaseResponse struct {
	Result []string `json:"result,omitempty"`
	ArangoError
}

// Databases returns a list of all databases found by the client.
func (c *client) Databases(ctx context.Context) ([]Database, error) {
	result, err := listDatabases(ctx, c.conn, path.Join("/_db/_system/_api/database"))
	if err != nil {
		return nil, WithStack(err)
	}
	return result, nil
}

// AccessibleDatabases returns a list of all databases that can be accessed by the authenticated user.
func (c *client) AccessibleDatabases(ctx context.Context) ([]Database, error) {
	result, err := listDatabases(ctx, c.conn, path.Join("/_db/_system/_api/database/user"))
	if err != nil {
		return nil, WithStack(err)
	}
	return result, nil
}

// listDatabases returns a list of databases using a GET to the given path.
func listDatabases(ctx context.Context, conn Connection, path string) ([]Database, error) {
	req, err := conn.NewRequest("GET", path)
	if err != nil {
		return nil, WithStack(err)
	}
	resp, err := conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return nil, WithStack(err)
	}
	var data getDatabaseResponse
	if err := resp.ParseBody("", &data); err != nil {
		return nil, WithStack(err)
	}
	result := make([]Database, 0, len(data.Result))
	for _, name := range data.Result {
		db, err := newDatabase(name, conn)
		if err != nil {
			return nil, WithStack(err)
		}
		result = append(result, db)
	}
	return result, nil
}

// CreateDatabase creates a new database with given name and opens a connection to it.
// If the a database with given name already exists, a DuplicateError is returned.
func (c *client) CreateDatabase(ctx context.Context, name string, options *CreateDatabaseOptions) (Database, error) {
	input := struct {
		CreateDatabaseOptions
		Name string `json:"name"`
	}{
		Name: name,
	}
	if options != nil {
		input.CreateDatabaseOptions = *options
	}
	req, err := c.conn.NewRequest("POST", path.Join("_db/_system/_api/database"))
	if err != nil {
		return nil, WithStack(err)
	}
	if _, err := req.SetBody(input); err != nil {
		return nil, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, WithStack(err)
	}
	if err := resp.CheckStatus(201); err != nil {
		return nil, WithStack(err)
	}
	db, err := newDatabase(name, c.conn)
	if err != nil {
		return nil, WithStack(err)
	}
	return db, nil
}
