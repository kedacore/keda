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

// User opens a connection to an existing user.
// If no user with given name exists, an NotFoundError is returned.
func (c *client) User(ctx context.Context, name string) (User, error) {
	escapedName := pathEscape(name)
	req, err := c.conn.NewRequest("GET", path.Join("_api/user", escapedName))
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
	var data userData
	if err := resp.ParseBody("", &data); err != nil {
		return nil, WithStack(err)
	}
	u, err := newUser(data, c.conn)
	if err != nil {
		return nil, WithStack(err)
	}
	return u, nil
}

// UserExists returns true if a database with given name exists.
func (c *client) UserExists(ctx context.Context, name string) (bool, error) {
	escapedName := pathEscape(name)
	req, err := c.conn.NewRequest("GET", path.Join("_api", "user", escapedName))
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

type listUsersResponse struct {
	Result []userData `json:"result,omitempty"`
	ArangoError
}

// Users returns a list of all users found by the client.
func (c *client) Users(ctx context.Context) ([]User, error) {
	req, err := c.conn.NewRequest("GET", "/_api/user")
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
	var data listUsersResponse
	if err := resp.ParseBody("", &data); err != nil {
		return nil, WithStack(err)
	}
	result := make([]User, 0, len(data.Result))
	for _, userData := range data.Result {
		u, err := newUser(userData, c.conn)
		if err != nil {
			return nil, WithStack(err)
		}
		result = append(result, u)
	}
	return result, nil
}

// CreateUser creates a new user with given name and opens a connection to it.
// If a user with given name already exists, a DuplicateError is returned.
func (c *client) CreateUser(ctx context.Context, name string, options *UserOptions) (User, error) {
	input := struct {
		UserOptions
		Name string `json:"user"`
	}{
		Name: name,
	}
	if options != nil {
		input.UserOptions = *options
	}
	req, err := c.conn.NewRequest("POST", path.Join("_api/user"))
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
	var data userData
	if err := resp.ParseBody("", &data); err != nil {
		return nil, WithStack(err)
	}
	u, err := newUser(data, c.conn)
	if err != nil {
		return nil, WithStack(err)
	}
	return u, nil
}
