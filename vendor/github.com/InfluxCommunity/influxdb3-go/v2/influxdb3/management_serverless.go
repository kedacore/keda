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
)

type (
	// ServerlessClient represents a client for InfluxDB Serverless administration operations.
	ServerlessClient struct {
		client *Client
	}

	Bucket struct {
		Name           string                `json:"name"`
		OrgID          string                `json:"orgID,omitempty"`
		Description    string                `json:"description,omitempty"`
		RetentionRules []BucketRetentionRule `json:"retentionRules"`
	}

	BucketRetentionRule struct {
		Type               string `json:"type,omitempty"`
		EverySeconds       int    `json:"everySeconds,omitempty"`
		ShardGroupDuration int    `json:"shardGroupDuration,omitempty"`
	}
)

// NewServerlessClient creates new ServerlessClient with given InfluxDB client.
func NewServerlessClient(client *Client) *ServerlessClient {
	return &ServerlessClient{client: client}
}

// CreateBucket creates a new bucket
func (c *ServerlessClient) CreateBucket(ctx context.Context, bucket *Bucket) error {
	if bucket == nil {
		return errors.New("bucket must not be nil")
	}

	if bucket.OrgID == "" {
		bucket.OrgID = c.client.config.Organization
	}

	if bucket.Name == "" {
		bucket.Name = c.client.config.Database
	}

	return c.createBucket(ctx, "v2/buckets", bucket)
}

// createBucket is a helper function for CreateBucket to enhance test coverage.
func (c *ServerlessClient) createBucket(ctx context.Context, path string, bucket any) error {
	u, err := c.client.apiURL.Parse(path)
	if err != nil {
		return fmt.Errorf("failed to parth bucket creation path: %w", err)
	}

	body, err := json.Marshal(bucket)
	if err != nil {
		return fmt.Errorf("failed to marshal bucket creation request body: %w", err)
	}

	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	param := httpParams{
		endpointURL: u,
		queryParams: nil,
		httpMethod:  "POST",
		headers:     headers,
		body:        bytes.NewReader(body),
	}

	resp, err := c.client.makeAPICall(ctx, param)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}
