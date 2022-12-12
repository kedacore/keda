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

type AuthenticationType int

const (
	// AuthenticationTypeBasic uses username+password basic authentication
	AuthenticationTypeBasic AuthenticationType = iota
	// AuthenticationTypeJWT uses username+password JWT token based authentication
	AuthenticationTypeJWT
	// AuthenticationTypeRaw uses a raw value for the Authorization header
	AuthenticationTypeRaw
)

// Authentication implements a kind of authentication.
type Authentication interface {
	// Returns the type of authentication
	Type() AuthenticationType
	// Get returns a configuration property of the authentication.
	// Supported properties depend on type of authentication.
	Get(property string) string
}

// BasicAuthentication creates an authentication implementation based on the given username & password.
func BasicAuthentication(userName, password string) Authentication {
	return &userNameAuthentication{
		authType: AuthenticationTypeBasic,
		userName: userName,
		password: password,
	}
}

// JWTAuthentication creates a JWT token authentication implementation based on the given username & password.
func JWTAuthentication(userName, password string) Authentication {
	return &userNameAuthentication{
		authType: AuthenticationTypeJWT,
		userName: userName,
		password: password,
	}
}

// basicAuthentication implements HTTP Basic authentication.
type userNameAuthentication struct {
	authType AuthenticationType
	userName string
	password string
}

// Returns the type of authentication
func (a *userNameAuthentication) Type() AuthenticationType {
	return a.authType
}

// Get returns a configuration property of the authentication.
// Supported properties depend on type of authentication.
func (a *userNameAuthentication) Get(property string) string {
	switch property {
	case "username":
		return a.userName
	case "password":
		return a.password
	default:
		return ""
	}
}

// RawAuthentication creates a raw authentication implementation based on the given value for the Authorization header.
func RawAuthentication(value string) Authentication {
	return &rawAuthentication{
		value: value,
	}
}

// rawAuthentication implements Raw authentication.
type rawAuthentication struct {
	value string
}

// Returns the type of authentication
func (a *rawAuthentication) Type() AuthenticationType {
	return AuthenticationTypeRaw
}

// Get returns a configuration property of the authentication.
// Supported properties depend on type of authentication.
func (a *rawAuthentication) Get(property string) string {
	switch property {
	case "value":
		return a.value
	default:
		return ""
	}
}
