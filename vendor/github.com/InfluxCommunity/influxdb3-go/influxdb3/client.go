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

// Package influxdb3 provides client for InfluxDB server.
package influxdb3

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/apache/arrow/go/v13/arrow/flight"
)

// Client implements an InfluxDB client.
type Client struct {
	// Configuration options.
	config ClientConfig
	// Pre-created Authorization HTTP header value.
	authorization string
	// Cached base server API URL.
	apiURL *url.URL
	// Flight client for executing queries
	queryClient *flight.Client
}

// httpParams holds parameters for creating an HTTP request
type httpParams struct {
	// URL of server endpoint
	endpointURL *url.URL
	// Params to be added to URL
	queryParams url.Values
	// HTTP request method, eg. POST
	httpMethod string
	// HTTP request headers
	headers http.Header
	// HTTP POST/PUT body
	body io.Reader
}

// New creates new Client with given config, where `Host` and `Token` are mandatory.
func New(config ClientConfig) (*Client, error) {
	// Validate the config
	err := config.validate()
	if err != nil {
		return nil, err
	}

	// Create client instance
	c := &Client{config: config}

	// Prepare host API URL
	hostAddress := config.Host
	if !strings.HasSuffix(hostAddress, "/") {
		// For subsequent path parts concatenation, url has to end with '/'
		hostAddress = config.Host + "/"
	}
	c.apiURL, err = url.Parse(hostAddress)
	if err != nil {
		return nil, fmt.Errorf("parsing host URL: %w", err)
	}
	c.apiURL.Path = path.Join(c.apiURL.Path, "api/v2") + "/"

	// Prepare auth header value
	c.authorization = "Token " + c.config.Token

	// Prepare HTTP client
	if c.config.HTTPClient == nil {
		c.config.HTTPClient = http.DefaultClient
	}

	// Use default write option if not set
	if config.WriteOptions == nil {
		options := DefaultWriteOptions
		c.config.WriteOptions = &options
	}

	// Init FlightSQL client
	err = c.initializeQueryClient()
	if err != nil {
		return nil, fmt.Errorf("flight client: %w", err)
	}

	return c, nil
}

// NewFromConnectionString creates new Client from the specified connection string.
// Parameters:
//   - connectionString: connection string in URL format.
//
// Supported query parameters:
//   - token - authentication token (required)
//   - org - organization name
//   - database - database (bucket) name
//   - precision - timestamp precision when writing data
//   - gzipThreshold - payload size threshold for gzipping data
//
// Example: https://us-east-1-1.aws.cloud2.influxdata.com/?token=my-token&database=my-database
func NewFromConnectionString(connectionString string) (*Client, error) {
	cfg := ClientConfig{}
	err := cfg.parse(connectionString)
	if err != nil {
		return nil, err
	}
	return New(cfg)
}

// NewFromEnv creates new Client instance from environment variables.
// Supported variables:
//   - INFLUX_HOST - cloud/server URL (required)
//   - INFLUX_TOKEN - authentication token (required)
//   - INFLUX_ORG - organization name
//   - INFLUX_DATABASE - database (bucket) name
//   - INFLUX_PRECISION - timestamp precision when writing data
//   - INFLUX_GZIP_THRESHOLD - payload size threshold for gzipping data
func NewFromEnv() (*Client, error) {
	cfg := ClientConfig{}
	err := cfg.env()
	if err != nil {
		return nil, err
	}
	return New(cfg)
}

// makeAPICall issues an HTTP request to InfluxDB host API url according to parameters.
// Additionally, sets Authorization header and User-Agent.
// It returns http.Response or error. Error can be a *hostError if host responded with error.
func (c *Client) makeAPICall(ctx context.Context, params httpParams) (*http.Response, error) {
	// copy URL
	urlObj := *params.endpointURL
	urlObj.RawQuery = params.queryParams.Encode()

	fullURL := urlObj.String()

	req, err := http.NewRequestWithContext(ctx, params.httpMethod, fullURL, params.body)
	if err != nil {
		return nil, fmt.Errorf("error calling %s: %v", fullURL, err)
	}
	for k, v := range c.config.Headers {
		for _, i := range v {
			req.Header.Add(k, i)
		}
	}
	for k, v := range params.headers {
		for _, i := range v {
			req.Header.Add(k, i)
		}
	}
	req.Header.Set("User-Agent", userAgent)
	if c.authorization != "" {
		req.Header.Add("Authorization", c.authorization)
	}

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error calling %s: %v", fullURL, err)
	}
	err = c.resolveHTTPError(resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// resolveHTTPError parses host error response and returns error with human-readable message
func (c *Client) resolveHTTPError(r *http.Response) error {
	// successful status code range
	if r.StatusCode >= 200 && r.StatusCode < 300 {
		return nil
	}

	var httpError struct {
		ServerError
		// Error message of InfluxDB 1 error
		Error string `json:"error"`
	}

	httpError.StatusCode = r.StatusCode
	if v := r.Header.Get("Retry-After"); v != "" {
		r, err := strconv.ParseInt(v, 10, 32)
		if err == nil {
			httpError.RetryAfter = int(r)
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpError.Message = fmt.Sprintf("cannot read error response:: %v", err)
	}
	ctype, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if ctype == "application/json" {
		err := json.Unmarshal(body, &httpError)
		if err != nil {
			httpError.Message = fmt.Sprintf("cannot decode error response: %v", err)
		}
		if httpError.Message == "" && httpError.Code == "" {
			httpError.Message = httpError.Error
		}
	}
	if httpError.Message == "" {
		if len(body) > 0 {
			httpError.Message = string(body)
		} else {
			httpError.Message = r.Status
		}
	}

	return &httpError.ServerError
}

// Close closes all idle connections.
func (c *Client) Close() error {
	c.config.HTTPClient.CloseIdleConnections()
	err := (*c.queryClient).Close()
	return err
}
