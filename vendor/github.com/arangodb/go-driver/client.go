//
// DISCLAIMER
//
// Copyright 2017-2025 ArangoDB GmbH, Cologne, Germany
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

package driver

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Client provides access to a single ArangoDB database server, or an entire cluster of ArangoDB servers.
type Client interface {
	// SynchronizeEndpoints fetches all endpoints from an ArangoDB cluster and updates the
	// connection to use those endpoints.
	// When this client is connected to a single server, nothing happens.
	// When this client is connected to a cluster of servers, the connection will be updated to reflect
	// the layout of the cluster.
	// This function requires ArangoDB 3.1.15 or up.
	SynchronizeEndpoints(ctx context.Context) error

	// SynchronizeEndpoints2 fetches all endpoints from an ArangoDB cluster and updates the
	// connection to use those endpoints.
	// When this client is connected to a single server, nothing happens.
	// When this client is connected to a cluster of servers, the connection will be updated to reflect
	// the layout of the cluster.
	// Compared to SynchronizeEndpoints, this function expects a database name as additional parameter.
	// This database name is used to call `_db/<dbname>/_api/cluster/endpoints`. SynchronizeEndpoints uses
	// the default database, i.e. `_system`. In the case the user does not have access to `_system`,
	// SynchronizeEndpoints does not work with earlier versions of arangodb.
	SynchronizeEndpoints2(ctx context.Context, dbname string) error

	// Connection returns the connection used by this client
	Connection() Connection

	// ClientDatabases - Database functions
	ClientDatabases

	// ClientUsers - User functions
	ClientUsers

	// ClientCluster - Cluster functions
	ClientCluster

	// ClientServerInfo - Individual server information functions
	ClientServerInfo

	// ClientServerAdmin - Server/cluster administration functions
	ClientServerAdmin

	// ClientReplication - Replication functions
	ClientReplication

	// ClientAdminBackup - Backup functions
	ClientAdminBackup

	// ClientFoxx - Foxx functions
	ClientFoxx

	// ClientAsyncJob - Asynchronous job functions
	ClientAsyncJob

	ClientLog
}

// LogLevels is a map of topics to log level.
type LogLevels map[string]string

// ClientLog provides access to client logs' wide specific operations.
type ClientLog interface {
	// GetLogLevels returns log levels for topics.
	GetLogLevels(ctx context.Context, opts *LogLevelsGetOptions) (LogLevels, error)
	// SetLogLevels sets log levels for a given topics
	SetLogLevels(ctx context.Context, logLevels LogLevels, opts *LogLevelsSetOptions) error
}

// LogLevelsGetOptions describes log levels get options.
type LogLevelsGetOptions struct {
	// serverID describes log levels for a specific server ID.
	ServerID ServerID
}

// LogLevelsSetOptions describes log levels set options.
type LogLevelsSetOptions struct {
	// serverID describes log levels for a specific server ID.
	ServerID ServerID
}

// ClientConfig contains all settings needed to create a client.
type ClientConfig struct {
	// Connection is the actual server/cluster connection.
	// See http.NewConnection.
	Connection Connection
	// Authentication implements authentication on the server.
	Authentication Authentication

	// Deprecated: using non-zero duration causes routine leak. Please create your own implementation using Client.SynchronizeEndpoints2
	//
	// SynchronizeEndpointsInterval is the interval between automatic synchronization of endpoints.
	// If this value is 0, no automatic synchronization is performed.
	// If this value is > 0, automatic synchronization is started on a go routine.
	// This feature requires ArangoDB 3.1.15 or up.
	SynchronizeEndpointsInterval time.Duration
}

// VersionInfo describes the version of a database server.
type VersionInfo struct {
	// This will always contain "arango"
	Server string `json:"server,omitempty"`
	//  The server version string. The string has the format "major.minor.sub".
	// Major and minor will be numeric, and sub may contain a number or a textual version.
	Version Version `json:"version,omitempty"`
	// Type of license of the server
	License string `json:"license,omitempty"`
	// Optional additional details. This is returned only if the context is configured using WithDetails.
	Details map[string]interface{} `json:"details,omitempty"`
}

func (v *VersionInfo) IsEnterprise() bool {
	return v.License == "enterprise"
}

// String creates a string representation of the given VersionInfo.
func (v VersionInfo) String() string {
	result := fmt.Sprintf("%s, version %s, license %s", v.Server, v.Version, v.License)
	if len(v.Details) > 0 {
		lines := make([]string, 0, len(v.Details))
		for k, v := range v.Details {
			lines = append(lines, fmt.Sprintf("%s: %v", k, v))
		}
		sort.Strings(lines)
		result = result + "\n" + strings.Join(lines, "\n")
	}
	return result
}

// LicenseFeatures describes license's features.
type LicenseFeatures struct {
	// Expires is expiry date as Unix timestamp (seconds since January 1st, 1970 UTC).
	Expires int `json:"expires"`
}

// LicenseStatus describes license's status.
type LicenseStatus string

const (
	// LicenseStatusGood - The license is valid for more than 2 weeks.
	LicenseStatusGood LicenseStatus = "good"
	// LicenseStatusExpired - The license has expired. In this situation, no new Enterprise Edition features can be utilized.
	LicenseStatusExpired LicenseStatus = "expired"
	// LicenseStatusExpiring - The license is valid for less than 2 weeks.
	LicenseStatusExpiring LicenseStatus = "expiring"
	// LicenseStatusReadOnly - The license is expired over 2 weeks. The instance is now restricted to read-only mode.
	LicenseStatusReadOnly LicenseStatus = "read-only"
)

// License describes license information.
type License struct {
	// Features describes properties of the license.
	Features LicenseFeatures `json:"features"`
	// License is an encrypted license key in Base64 encoding.
	License string `json:"license"`
	// Status is a status of a license.
	Status LicenseStatus `json:"status"`
	// Version is a version of a license.
	Version int `json:"version"`
}
