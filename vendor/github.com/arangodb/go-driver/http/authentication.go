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

package http

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	driver "github.com/arangodb/go-driver"
)

// ErrAuthenticationNotChanged is returned when authentication is not changed.
var ErrAuthenticationNotChanged = errors.New("authentication not changed")

// Authentication implements a kind of authentication.
type httpAuthentication interface {
	// Prepare is called before the first request of the given connection is made.
	Prepare(ctx context.Context, conn driver.Connection) error

	// Configure is called for every request made on a connection.
	Configure(req driver.Request) error
}

// IsAuthenticationTheSame checks whether two authentications are the same.
func IsAuthenticationTheSame(auth1, auth2 driver.Authentication) bool {

	if auth1 == nil && auth2 == nil {
		return true
	}

	if auth1 == nil || auth2 == nil {
		return false
	}

	if auth1.Type() != auth2.Type() {
		return false
	}

	if auth1.Type() == driver.AuthenticationTypeRaw {
		if auth1.Get("value") != auth2.Get("value") {
			return false
		}
	} else {
		if auth1.Get("username") != auth2.Get("username") ||
			auth1.Get("password") != auth2.Get("password") {
			return false
		}
	}

	return true
}

// newBasicAuthentication creates an authentication implementation based on the given username & password.
func newBasicAuthentication(userName, password string) httpAuthentication {
	auth := fmt.Sprintf("%s:%s", userName, password)
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))
	return &basicAuthentication{
		authorizationValue: "Basic " + encoded,
	}
}

// newJWTAuthentication creates a JWT token authentication implementation based on the given username & password.
func newJWTAuthentication(userName, password string) httpAuthentication {
	return &jwtAuthentication{
		userName: userName,
		password: password,
	}
}

// newRawAuthentication creates a Raw authentication implementation based on the given value.
func newRawAuthentication(value string) httpAuthentication {
	return &basicAuthentication{
		authorizationValue: value,
	}
}

// basicAuthentication implements HTTP Basic authentication.
type basicAuthentication struct {
	authorizationValue string
}

// Prepare is called before the first request of the given connection is made.
func (a *basicAuthentication) Prepare(ctx context.Context, conn driver.Connection) error {
	// No need to do anything here
	return nil
}

// Configure is called for every request made on a connection.
func (a *basicAuthentication) Configure(req driver.Request) error {
	req.SetHeader("Authorization", a.authorizationValue)
	return nil
}

// jwtAuthentication implements JWT token authentication.
type jwtAuthentication struct {
	userName string
	password string
	token    string
}

type jwtOpenRequest struct {
	UserName string `json:"username"`
	Password string `json:"password"`
}

type jwtOpenResponse struct {
	Token              string `json:"jwt"`
	MustChangePassword bool   `json:"must_change_password,omitempty"`
}

// Prepare is called before the first request of the given connection is made.
func (a *jwtAuthentication) Prepare(ctx context.Context, conn driver.Connection) error {
	// Prepare request
	r, err := conn.NewRequest("POST", "/_open/auth")
	if err != nil {
		return driver.WithStack(err)
	}
	r.SetBody(jwtOpenRequest{
		UserName: a.userName,
		Password: a.password,
	})

	// Perform request
	resp, err := conn.Do(ctx, r)
	if err != nil {
		return driver.WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return driver.WithStack(err)
	}

	// Parse response
	var data jwtOpenResponse
	if err := resp.ParseBody("", &data); err != nil {
		return driver.WithStack(err)
	}

	// Store token
	a.token = data.Token

	// Ok
	return nil
}

// Configure is called for every request made on a connection.
func (a *jwtAuthentication) Configure(req driver.Request) error {
	req.SetHeader("Authorization", "bearer "+a.token)
	return nil
}

// newAuthenticatedConnection creates a Connection that applies the given connection on the given underlying connection.
func newAuthenticatedConnection(conn driver.Connection, auth httpAuthentication) (driver.Connection, error) {
	if conn == nil {
		return nil, driver.WithStack(driver.InvalidArgumentError{Message: "conn is nil"})
	}
	if auth == nil {
		return nil, driver.WithStack(driver.InvalidArgumentError{Message: "auth is nil"})
	}
	return &authenticatedConnection{
		conn: conn,
		auth: auth,
	}, nil
}

// authenticatedConnection implements authentication behavior for connections.
type authenticatedConnection struct {
	conn         driver.Connection // Un-authenticated connection
	auth         httpAuthentication
	prepareMutex sync.Mutex
	prepared     int32
}

// NewRequest creates a new request with given method and path.
func (c *authenticatedConnection) NewRequest(method, path string) (driver.Request, error) {
	r, err := c.conn.NewRequest(method, path)
	if err != nil {
		return nil, driver.WithStack(err)
	}
	return r, nil
}

// Do performs a given request, returning its response.
func (c *authenticatedConnection) Do(ctx context.Context, req driver.Request) (driver.Response, error) {
	if atomic.LoadInt32(&c.prepared) == 0 {
		// Probably we're not yet prepared
		if err := c.prepare(ctx); err != nil {
			// Authentication failed
			return nil, driver.WithStack(err)
		}
	}
	// Configure the request for authentication.
	if err := c.auth.Configure(req); err != nil {
		// Failed to configure request for authentication
		return nil, driver.WithStack(err)
	}
	// Do the authenticated request
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return nil, driver.WithStack(err)
	}
	return resp, nil
}

// Unmarshal unmarshals the given raw object into the given result interface.
func (c *authenticatedConnection) Unmarshal(data driver.RawObject, result interface{}) error {
	if err := c.conn.Unmarshal(data, result); err != nil {
		return driver.WithStack(err)
	}
	return nil
}

// Endpoints returns the endpoints used by this connection.
func (c *authenticatedConnection) Endpoints() []string {
	return c.conn.Endpoints()
}

// UpdateEndpoints reconfigures the connection to use the given endpoints.
func (c *authenticatedConnection) UpdateEndpoints(endpoints []string) error {
	if err := c.conn.UpdateEndpoints(endpoints); err != nil {
		return driver.WithStack(err)
	}
	return nil
}

// Configure the authentication used for this connection.
func (c *authenticatedConnection) SetAuthentication(auth driver.Authentication) (driver.Connection, error) {
	result, err := c.conn.SetAuthentication(auth)
	if err != nil {
		return nil, driver.WithStack(err)
	}
	return result, nil
}

// Protocols returns all protocols used by this connection.
func (c *authenticatedConnection) Protocols() driver.ProtocolSet {
	return c.conn.Protocols()
}

// prepare calls Authentication.Prepare if needed.
func (c *authenticatedConnection) prepare(ctx context.Context) error {
	c.prepareMutex.Lock()
	defer c.prepareMutex.Unlock()
	if c.prepared == 0 {
		// We need to prepare first
		if err := c.auth.Prepare(ctx, c.conn); err != nil {
			// Authentication failed
			return driver.WithStack(err)
		}
		// We're now prepared
		atomic.StoreInt32(&c.prepared, 1)
	} else {
		// We're already prepared, do nothing
	}
	return nil
}
