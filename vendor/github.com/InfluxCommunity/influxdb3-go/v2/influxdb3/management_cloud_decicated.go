/*
 The MIT License

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE.
*/

package influxdb3

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

type (
	// CloudDedicatedClient represents a client for InfluxDB Cloud Dedicated administration operations.
	// https://docs.influxdata.com/influxdb/cloud-dedicated/admin/databases/create/?t=Management+API
	CloudDedicatedClient struct {
		client *Client
	}

	CloudDedicatedClientConfig struct {
		AccountID        string
		ClusterID        string
		ManagementToken  string
		ManagementAPIURL *url.URL // default is https://console.influxdata.com
	}

	Database struct {
		ClusterDatabaseName               string              `json:"name"`
		ClusterDatabaseMaxTables          uint64              `json:"maxTables"`          // default 500
		ClusterDatabaseMaxColumnsPerTable uint64              `json:"maxColumnsPerTable"` // default 250
		ClusterDatabaseRetentionPeriod    uint64              `json:"retentionPeriod"`    // nanoseconds default 0 is infinite
		ClusterDatabasePartitionTemplate  []PartitionTemplate `json:"partitionTemplate"`  // Tag or TagBucket, limit is total of 7
	}

	PartitionTemplate interface {
		isPartitionTemplate()
	}

	Tag struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}

	TagBucket struct {
		Type  string         `json:"type"`
		Value TagBucketValue `json:"value"`
	}

	TagBucketValue struct {
		TagName         string `json:"tagName"`
		NumberOfBuckets uint64 `json:"numberOfBuckets"`
	}
)

var (
	MaxPartitions    = 7
	ManagementAPIURL = "https://console.influxdata.com"
)

func (t Tag) isPartitionTemplate()        {}
func (tb TagBucket) isPartitionTemplate() {}

// NewCloudDedicatedClient creates new CloudDedicatedClient with given InfluxDB client.
func NewCloudDedicatedClient(client *Client) *CloudDedicatedClient {
	return &CloudDedicatedClient{client: client}
}

// CreateDatabase creates a new database. If Database.ClusterDatabaseName is
// not provided, it defaults to the database name configured in Client.
func (d *CloudDedicatedClient) CreateDatabase(ctx context.Context, config *CloudDedicatedClientConfig, db *Database) error {
	if db == nil {
		return errors.New("database must not nil")
	}

	if db.ClusterDatabaseName == "" {
		if d.client.config.Database == "" {
			return errors.New("database name must not be empty")
		}
		db.ClusterDatabaseName = d.client.config.Database
	}

	if len(db.ClusterDatabasePartitionTemplate) > MaxPartitions {
		return fmt.Errorf("partition template should not have more than %d tags or tag buckets", MaxPartitions)
	}

	if db.ClusterDatabaseMaxTables == 0 {
		db.ClusterDatabaseMaxTables = uint64(500)
	}

	if db.ClusterDatabaseMaxColumnsPerTable == 0 {
		db.ClusterDatabaseMaxColumnsPerTable = uint64(250)
	}

	path := fmt.Sprintf("/api/v0/accounts/%s/clusters/%s/databases", config.AccountID, config.ClusterID)

	return d.createDatabase(ctx, path, db, config)
}

// createDatabase is a helper function for CreateDatabase to enhance test coverage.
func (d *CloudDedicatedClient) createDatabase(ctx context.Context, path string, db any, config *CloudDedicatedClientConfig) error {
	if config.ManagementAPIURL == nil {
		config.ManagementAPIURL, _ = url.Parse(ManagementAPIURL)
	}
	u, err := config.ManagementAPIURL.Parse(path)
	if err != nil {
		return fmt.Errorf("failed to parse database creation path: %w", err)
	}

	body, err := json.Marshal(db)
	if err != nil {
		return fmt.Errorf("failed to marshal database creation request body: %w", err)
	}

	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("Accept", "application/json")
	headers.Set("Authorization", "Bearer "+config.ManagementToken)

	param := httpParams{
		endpointURL: u,
		queryParams: nil,
		httpMethod:  "POST",
		headers:     headers,
		body:        bytes.NewReader(body),
	}

	resp, err := d.client.makeAPICall(ctx, param)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}
