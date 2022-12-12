/*
 * Copyright (c) "Neo4j"
 * Neo4j Sweden AB [https://neo4j.com]
 *
 * This file is part of Neo4j.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package neo4j provides required functionality to connect and execute statements against a Neo4j Database.
package neo4j

import (
	"context"
	"net/url"
)

// Driver represents a pool(s) of connections to a neo4j server or cluster. It's
// safe for concurrent use.
// Deprecated: please use DriverWithContext instead. This interface will be removed in 6.0.
type Driver interface {
	// Target returns the url this driver is bootstrapped
	Target() url.URL
	// NewSession creates a new session based on the specified session configuration.
	NewSession(config SessionConfig) Session
	// VerifyConnectivity checks that the driver can connect to a remote server or cluster by
	// establishing a network connection with the remote. Returns nil if succesful
	// or error describing the problem.
	VerifyConnectivity() error
	// Close the driver and all underlying connections
	Close() error
	// IsEncrypted determines whether the driver communication with the server
	// is encrypted. This is a static check. The function can also be called on
	// a closed Driver.
	IsEncrypted() bool
}

// NewDriver is the entry point to the neo4j driver to create an instance of a Driver. It is the first function to
// be called in order to establish a connection to a neo4j database. It requires a Bolt URI and an authentication
// token as parameters and can also take optional configuration function(s) as variadic parameters.
//
// In order to connect to a single instance database, you need to pass a URI with scheme 'bolt', 'bolt+s' or 'bolt+ssc'.
//	driver, err = NewDriver("bolt://db.server:7687", BasicAuth(username, password))
//
// In order to connect to a causal cluster database, you need to pass a URI with scheme 'neo4j', 'neo4j+s' or 'neo4j+ssc'
// and its host part set to be one of the core cluster members.
//	driver, err = NewDriver("neo4j://core.db.server:7687", BasicAuth(username, password))
//
// You can override default configuration options by providing a configuration function(s)
//	driver, err = NewDriver(uri, BasicAuth(username, password), function (config *Config) {
// 		config.MaxConnectionPoolSize = 10
// 	})
//
// Deprecated: please use NewDriverWithContext instead. This function will be removed in 6.0.
func NewDriver(target string, auth AuthToken, configurers ...func(*Config)) (Driver, error) {
	delegate, err := NewDriverWithContext(target, auth, configurers...)
	if err != nil {
		return nil, err
	}
	return &driver{delegate: delegate}, nil
}

type driver struct {
	delegate DriverWithContext
}

func (d *driver) Target() url.URL {
	return d.delegate.Target()
}

func (d *driver) NewSession(config SessionConfig) Session {
	return d.delegate.NewSession(context.Background(), config).legacy()
}

func (d *driver) VerifyConnectivity() error {
	return d.delegate.VerifyConnectivity(context.Background())
}

func (d *driver) Close() error {
	return d.delegate.Close(context.Background())
}

func (d *driver) IsEncrypted() bool {
	return d.delegate.IsEncrypted()
}
