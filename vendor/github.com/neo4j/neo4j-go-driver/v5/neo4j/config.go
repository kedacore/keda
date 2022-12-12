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

package neo4j

import (
	"crypto/tls"
	"crypto/x509"
	"math"
	"net/url"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j/log"
)

// A Config contains options that can be used to customize certain
// aspects of the driver
type Config struct {
	// RootCAs defines the set of certificate authorities that the driver trusts. If set
	// to nil, the driver uses hosts system certificates.
	//
	// The trusted certificates are used to validate connections for URI schemes 'bolt+s'
	// and 'neo4j+s'.
	//
	// Deprecated: RootCAs will be removed in 6.0.
	// Please rely on TlsConfig's RootCAs attribute instead.
	// RootCAs is ignored if TlsConfig is set.
	RootCAs *x509.CertPool
	// TlsConfig defines the TLS configuration of the driver.
	//
	// The configuration is only used for URI schemes 'bolt+s', 'bolt+ssc',
	// 'neo4j+s' and 'neo4j+ssc'.
	//
	// The MinVersion attribute is always set to VersionTLS11
	// The InsecureSkipVerify attribute of TlsConfig is always derived from the initial URI scheme.
	// The ServerName attribute of TlsConfig is always derived from the initial URI host.
	//
	// This is considered an advanced setting, use it at your own risk.
	// Introduced in 5.0.
	TlsConfig *tls.Config

	// Logging target the driver will send its log outputs
	//
	// Possible to use custom logger (implement log.Logger interface) or
	// use neo4j.ConsoleLogger.
	//
	// default: No Op Logger (log.Void)
	Log log.Logger
	// Resolver that would be used to resolve initial router address. This may
	// be useful if you want to provide more than one URL for initial router.
	// If not specified, the URL provided to NewDriver or NewDriverWithContext
	// is used as the initial router.
	//
	// default: nil
	AddressResolver ServerAddressResolver
	// Maximum amount of time a retryable operation would continue retrying. It
	// cannot be specified as a negative value.
	//
	// default: 30 * time.Second
	MaxTransactionRetryTime time.Duration
	// Maximum number of connections per URL to allow on this driver. It
	// cannot be specified as 0 and negative values are interpreted as
	// math.MaxInt32.
	//
	// default: 100
	MaxConnectionPoolSize int
	// Maximum connection lifetime on pooled connections. Values less than
	// or equal to 0 disables the lifetime check.
	//
	// default: 1 * time.Hour
	MaxConnectionLifetime time.Duration
	// Maximum amount of time to either acquire an idle connection from the pool
	// or create a new connection (when the pool is not full). Negative values
	// result in an infinite wait time, whereas a 0 value results in no timeout.
	// If no timeout is set, an immediate failure follows when there are no
	// available connections.
	//
	// Since 4.3, this setting competes with connection read timeout hints, if
	// the server-side option called
	// "dbms.connector.bolt.connection_keep_alive_for_requests" is enabled.
	// The read timeout is automatically applied and may result in connections
	// being dropped if they are idle beyond the corresponding period.
	//
	// Since 5.0, this setting competes with the context-aware APIs. These APIs
	// are discoverable through NewDriverWithContext.
	// When a connection needs to be acquired from the internal driver
	// connection pool and the user-provided context.Context carries a deadline
	// (through context.WithTimeout or context.WithDeadline), the earliest
	// deadline wins. Connections are still subject to early terminations if a read timeout
	// hint is received.
	//
	// default: 1 * time.Minute
	ConnectionAcquisitionTimeout time.Duration
	// Connect timeout that will be set on underlying sockets. Values less than
	// or equal to 0 results in no timeout being applied.
	//
	// Since 5.0, this setting competes with the context-aware APIs. These APIs
	// are discoverable through NewDriverWithContext.
	// If a connection needs to be created when one of these APIs is called
	// and the user-provided context.Context carries a deadline (through
	// context.WithTimeout or context.WithDeadline), the TCP dialer will pick
	// the earliest between this setting and the context deadline.
	//
	// default: 5 * time.Second
	SocketConnectTimeout time.Duration
	// Whether to enable TCP keep alive on underlying sockets.
	//
	// default: true
	SocketKeepalive bool
	// Optionally override the user agent string sent to Neo4j server.
	//
	// default: neo4j.UserAgent
	UserAgent string
	// FetchSize defines how many records to pull from server in each batch.
	// From Bolt protocol v4 (Neo4j 4+) records can be fetched in batches as
	// compared to fetching all in previous versions.
	//
	// If FetchSize is set to FetchDefault, the driver decides the appropriate
	// size. If set to a positive value that size is used if the underlying
	// protocol supports it otherwise it is ignored.
	//
	// To turn off fetching in batches and always fetch everything, set
	// FetchSize to FetchAll.
	// If a single large result is to be retrieved, this is the most performant
	// setting.
	FetchSize int
}

func defaultConfig() *Config {
	return &Config{
		AddressResolver:              nil,
		MaxTransactionRetryTime:      30 * time.Second,
		MaxConnectionPoolSize:        100,
		MaxConnectionLifetime:        1 * time.Hour,
		ConnectionAcquisitionTimeout: 1 * time.Minute,
		SocketConnectTimeout:         5 * time.Second,
		SocketKeepalive:              true,
		RootCAs:                      nil,
		UserAgent:                    UserAgent,
		FetchSize:                    FetchDefault,
	}
}

func validateAndNormaliseConfig(config *Config) error {
	// Max Transaction Retry Time
	if config.MaxTransactionRetryTime < 0 {
		return &UsageError{Message: "Maximum transaction retry time cannot be smaller than 0"}
	}

	// Max Connection Pool Size
	if config.MaxConnectionPoolSize == 0 {
		return &UsageError{Message: "Maximum connection pool cannot be 0"}
	}

	if config.MaxConnectionPoolSize < 0 {
		config.MaxConnectionPoolSize = math.MaxInt32
	}

	// Max Connection Lifetime
	if config.MaxConnectionLifetime < 0 {
		config.MaxConnectionLifetime = 0
	}

	// Connection Acquisition Timeout
	if config.ConnectionAcquisitionTimeout < 0 {
		config.ConnectionAcquisitionTimeout = -1
	}

	// Socket Connect Timeout
	if config.SocketConnectTimeout < 0 {
		config.SocketConnectTimeout = 0
	}

	return nil
}

// ServerAddress represents a host and port. Host can either be an IP address or a DNS name.
// Both IPv4 and IPv6 hosts are supported.
type ServerAddress interface {
	// Hostname returns the host portion of this ServerAddress.
	Hostname() string
	// Port returns the port portion of this ServerAddress.
	Port() string
}

// ServerAddressResolver is a function type that defines the resolver function used by the routing driver to
// resolve the initial address used to create the driver.
type ServerAddressResolver func(address ServerAddress) []ServerAddress

func newServerAddressURL(hostname string, port string) *url.URL {
	if hostname == "" {
		return nil
	}

	hostAndPort := hostname
	if port != "" {
		hostAndPort = hostAndPort + ":" + port
	}

	return &url.URL{Host: hostAndPort}
}

// NewServerAddress generates a ServerAddress with provided hostname and port information.
func NewServerAddress(hostname string, port string) ServerAddress {
	return newServerAddressURL(hostname, port)
}
