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

/*
Package jwt provides a helper function used to access ArangoDB
servers using a JWT secret.

Authenticating with a JWT secret results in "super-user" access
to the database.

To use a JWT secret to access your database, use code like this:

	// Create an HTTP connection to the database
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{"http://localhost:8529"},
	})
	if err != nil {
		// Handle error
	}

	// Prepare authentication
	hdr, err := CreateArangodJwtAuthorizationHeader("yourJWTSecret", "yourUniqueServerID")
	if err != nil {
		// Handle error
	}
	auth := driver.RawAuthentication(hdr)

	// Create a client
	c, err := driver.NewClient(driver.ClientConfig{
		Connection:     conn,
		Authentication: auth,
	})
	if err != nil {
		// Handle error
	}
*/
package jwt
