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
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

// Package connector is responsible for connecting to a database server.
package connector

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/db"
	"io"
	"net"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/bolt"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/log"
)

type Connector struct {
	SkipEncryption bool
	SkipVerify     bool
	// Deprecated: RootCAs will be removed in 6.0. Configure TlsConfig directly instead.
	RootCAs         *x509.CertPool
	DialTimeout     time.Duration
	SocketKeepAlive bool
	Auth            map[string]any
	Log             log.Logger
	UserAgent       string
	RoutingContext  map[string]string
	Network         string
	TlsConfig       *tls.Config
}

func (c Connector) Connect(ctx context.Context, address string, boltLogger log.BoltLogger) (db.Connection, error) {
	dialer := net.Dialer{Timeout: c.DialTimeout}
	if !c.SocketKeepAlive {
		dialer.KeepAlive = -1 * time.Second // Turns keep-alive off
	}

	conn, err := dialer.DialContext(ctx, c.Network, address)
	if err != nil {
		return nil, err
	}

	// TLS not requested, perform Bolt handshake
	if c.SkipEncryption {
		return bolt.Connect(ctx, address, conn, c.Auth, c.UserAgent, c.RoutingContext, c.Log, boltLogger)
	}

	// TLS requested, continue with handshake
	serverName, _, err := net.SplitHostPort(address)
	if err != nil {
		conn.Close()
		return nil, err
	}
	tlsConn := tls.Client(conn, c.tlsConfig(serverName))
	err = tlsConn.HandshakeContext(ctx)
	if err != nil {
		if err == io.EOF {
			// Give a bit nicer error message
			err = errors.New("remote end closed the connection, check that TLS is enabled on the server")
		}
		conn.Close()
		return nil, &TlsError{inner: err}
	}
	// Perform Bolt handshake
	return bolt.Connect(ctx, address, tlsConn, c.Auth, c.UserAgent, c.RoutingContext, c.Log, boltLogger)
}

func (c Connector) tlsConfig(serverName string) *tls.Config {
	var config *tls.Config
	if c.TlsConfig == nil {
		config = &tls.Config{RootCAs: c.RootCAs}
	} else {
		config = c.TlsConfig
	}
	config.MinVersion = tls.VersionTLS11
	config.InsecureSkipVerify = c.SkipVerify
	config.ServerName = serverName
	return config
}

// TlsError encapsulates all errors related to TLS connection creation
// This is needed since the tls package does not provide a common error type
// à la net.Error, and a common type is needed to properly classify the error
// for Testkit
type TlsError struct {
	inner error
}

func (e *TlsError) Error() string {
	return e.inner.Error()
}
