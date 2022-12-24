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
)

type serverModeResponse struct {
	Mode ServerMode `json:"mode"`
	ArangoError
}

type serverModeRequest struct {
	Mode ServerMode `json:"mode"`
}

// ShutdownInfo stores information about shutdown of the coordinator.
type ShutdownInfo struct {
	// AQLCursors stores a number of AQL cursors that are still active.
	AQLCursors int `json:"AQLcursors"`
	// Transactions stores a number of ongoing transactions.
	Transactions int `json:"transactions"`
	// PendingJobs stores a number of ongoing asynchronous requests.
	PendingJobs int `json:"pendingJobs"`
	// DoneJobs stores a number of finished asynchronous requests, whose result has not yet been collected.
	DoneJobs int `json:"doneJobs"`
	// PregelConductors stores a number of ongoing Pregel jobs.
	PregelConductors int `json:"pregelConductors"`
	// LowPrioOngoingRequests stores a number of ongoing low priority requests.
	LowPrioOngoingRequests int `json:"lowPrioOngoingRequests"`
	// LowPrioQueuedRequests stores a number of queued low priority requests.
	LowPrioQueuedRequests int `json:"lowPrioQueuedRequests"`
	// AllClear is set if all operations are closed.
	AllClear bool `json:"allClear"`
	// SoftShutdownOngoing describes whether a soft shutdown of the Coordinator is in progress.
	SoftShutdownOngoing bool `json:"softShutdownOngoing"`
}

// ServerMode returns the current mode in which the server/cluster is operating.
// This call needs ArangoDB 3.3 and up.
func (c *client) ServerMode(ctx context.Context) (ServerMode, error) {
	req, err := c.conn.NewRequest("GET", "_admin/server/mode")
	if err != nil {
		return "", WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return "", WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return "", WithStack(err)
	}
	var result serverModeResponse
	if err := resp.ParseBody("", &result); err != nil {
		return "", WithStack(err)
	}
	return result.Mode, nil
}

// SetServerMode changes the current mode in which the server/cluster is operating.
// This call needs a client that uses JWT authentication.
// This call needs ArangoDB 3.3 and up.
func (c *client) SetServerMode(ctx context.Context, mode ServerMode) error {
	req, err := c.conn.NewRequest("PUT", "_admin/server/mode")
	if err != nil {
		return WithStack(err)
	}
	input := serverModeRequest{
		Mode: mode,
	}
	req, err = req.SetBody(input)
	if err != nil {
		return WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return WithStack(err)
	}
	return nil
}

// Logs retrieve logs from server in ArangoDB 3.8.0+ format
func (c *client) Logs(ctx context.Context) (ServerLogs, error) {
	req, err := c.conn.NewRequest("GET", "_admin/log/entries")
	if err != nil {
		return ServerLogs{}, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return ServerLogs{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return ServerLogs{}, WithStack(err)
	}
	var data ServerLogs
	if err := resp.ParseBody("", &data); err != nil {
		return ServerLogs{}, WithStack(err)
	}
	return data, nil
}

// Shutdown a specific server, optionally removing it from its cluster.
func (c *client) Shutdown(ctx context.Context, removeFromCluster bool) error {
	req, err := c.conn.NewRequest("DELETE", "_admin/shutdown")
	if err != nil {
		return WithStack(err)
	}
	if removeFromCluster {
		req.SetQuery("remove_from_cluster", "1")
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return WithStack(err)
	}
	return nil
}

// Metrics returns the metrics of the server in Prometheus format.
func (c *client) Metrics(ctx context.Context) ([]byte, error) {
	return c.getMetrics(ctx, "")
}

// MetricsForSingleServer returns the metrics of the specific server in Prometheus format.
// This parameter 'serverID' is only meaningful on Coordinators.
func (c *client) MetricsForSingleServer(ctx context.Context, serverID string) ([]byte, error) {
	return c.getMetrics(ctx, serverID)
}

// Metrics returns the metrics of the server in Prometheus format.
func (c *client) getMetrics(ctx context.Context, serverID string) ([]byte, error) {
	var rawResponse []byte
	ctx = WithRawResponse(ctx, &rawResponse)

	req, err := c.conn.NewRequest("GET", "_admin/metrics/v2")
	if err != nil {
		return rawResponse, WithStack(err)
	}

	if serverID != "" {
		req.SetQuery("serverId", serverID)
	}

	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return rawResponse, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return rawResponse, WithStack(err)
	}
	return rawResponse, nil
}

// Statistics queries statistics from a specific server.
func (c *client) Statistics(ctx context.Context) (ServerStatistics, error) {
	req, err := c.conn.NewRequest("GET", "_admin/statistics")
	if err != nil {
		return ServerStatistics{}, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return ServerStatistics{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return ServerStatistics{}, WithStack(err)
	}
	var data ServerStatistics
	if err := resp.ParseBody("", &data); err != nil {
		return ServerStatistics{}, WithStack(err)
	}
	return data, nil
}

// ShutdownV2 shuts down a specific coordinator, optionally removing it from the cluster with a graceful manner.
// When `graceful` is true then run soft shutdown process and the `ShutdownInfoV2` can be used to check the progress.
// It is available since versions: v3.7.12, v3.8.1, v3.9.0.
func (c *client) ShutdownV2(ctx context.Context, removeFromCluster, graceful bool) error {
	req, err := c.conn.NewRequest("DELETE", "_admin/shutdown")
	if err != nil {
		return WithStack(err)
	}
	if removeFromCluster {
		req.SetQuery("remove_from_cluster", "1")
	}
	if graceful {
		req.SetQuery("soft", "true")
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return WithStack(err)
	}
	return nil
}

// ShutdownInfoV2 returns information about shutdown progress.
// It is available since versions: v3.7.12, v3.8.1, v3.9.0.
func (c *client) ShutdownInfoV2(ctx context.Context) (ShutdownInfo, error) {
	req, err := c.conn.NewRequest("GET", "_admin/shutdown")
	if err != nil {
		return ShutdownInfo{}, WithStack(err)
	}
	resp, err := c.conn.Do(ctx, req)
	if err != nil {
		return ShutdownInfo{}, WithStack(err)
	}
	if err := resp.CheckStatus(200); err != nil {
		return ShutdownInfo{}, WithStack(err)
	}
	data := ShutdownInfo{}
	if err := resp.ParseBody("", &data); err != nil {
		return ShutdownInfo{}, WithStack(err)
	}
	return data, nil
}
