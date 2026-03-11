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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/apache/arrow-go/v18/arrow/flight"
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
	queryClient flight.Client
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
	c.apiURL.Path = path.Join(c.apiURL.Path, "api") + "/"

	// Prepare authorization header value
	authScheme := c.config.AuthScheme
	if authScheme == "" {
		authScheme = "Token"
	}
	c.authorization = fmt.Sprintf("%s %s", authScheme, c.config.Token)

	// Prepare SSL certificate pool (if host URL is secure)
	var certPool *x509.CertPool
	hostPortURL, secure := ReplaceURLProtocolWithPort(c.config.Host)
	if config.SSLRootsFilePath != "" && secure {
		// Use the system certificate pool
		certPool, err = x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("x509: %w", err)
		}

		certs, err := os.ReadFile(config.SSLRootsFilePath)
		if err != nil {
			return nil, fmt.Errorf("error reading %s: %w", config.SSLRootsFilePath, err)
		}
		ok := certPool.AppendCertsFromPEM(certs)
		if !ok {
			slog.Warn("No valid certificates found in " + config.SSLRootsFilePath)
		}
	}

	// Prepare proxy (if configured)
	var proxyURL *url.URL
	if config.Proxy != "" {
		proxyURL, err = url.Parse(config.Proxy)
		if err != nil {
			return nil, fmt.Errorf("parsing proxy URL: %w", err)
		}
	}

	// Prepare HTTP client
	if c.config.HTTPClient == nil {
		c.config.HTTPClient = newHTTPClient(config)
	} else {
		// HTTPClient provided by the user.
		configureHTTPClient(c.config.HTTPClient, config)
	}
	if certPool != nil {
		setHTTPClientCertPool(c.config.HTTPClient, certPool, config)
	}
	if proxyURL != nil {
		setHTTPClientProxy(c.config.HTTPClient, proxyURL, config)
	}

	// Use default write option if not set
	if config.WriteOptions == nil {
		options := DefaultWriteOptions
		c.config.WriteOptions = &options
	}

	// Init FlightSQL client
	err = c.initializeQueryClient(hostPortURL, secure, proxyURL)
	if err != nil {
		return nil, fmt.Errorf("flight client: %w", err)
	}

	return c, nil
}

// newHTTPTransport creates a new http.Transport based on ClientConfig.
func newHTTPTransport(config ClientConfig) *http.Transport {
	timeout := config.getTimeoutOrDefault()
	idleConnectionTimeout := config.getIdleConnectionTimeoutOrDefault()
	maxIdleConnections := config.getMaxIdleConnectionsOrDefault()

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.IdleConnTimeout = idleConnectionTimeout
	transport.MaxIdleConns = maxIdleConnections
	transport.MaxIdleConnsPerHost = maxIdleConnections

	dialer := &net.Dialer{
		Timeout:   timeout,          // connection timeout
		KeepAlive: 30 * time.Second, // unchanged default from http.DefaultTransport
	}
	transport.DialContext = dialer.DialContext

	return transport
}

// newHTTPClient creates a new HTTPClient based on ClientConfig.
func newHTTPClient(config ClientConfig) *http.Client {
	timeout := config.getTimeoutOrDefault()

	return &http.Client{
		Timeout:   timeout,
		Transport: newHTTPTransport(config),
	}
}

// configureHTTPClient configures existing HTTPClient based on ClientConfig.
func configureHTTPClient(httpClient *http.Client, config ClientConfig) {
	if config.isTimeoutSet() {
		httpClient.Timeout = config.getTimeoutOrDefault()
	}
	if config.isIdleConnectionTimeoutSet() {
		ensureTransportSet(httpClient, config)
		if transport, ok := httpClient.Transport.(*http.Transport); ok {
			transport.IdleConnTimeout = config.getIdleConnectionTimeoutOrDefault()
		}
	}
	if config.isMaxIdleConnectionsSet() {
		ensureTransportSet(httpClient, config)
		if transport, ok := httpClient.Transport.(*http.Transport); ok {
			maxIdleConnections := config.getMaxIdleConnectionsOrDefault()
			transport.MaxIdleConns = maxIdleConnections
			transport.MaxIdleConnsPerHost = maxIdleConnections
		}
	}
}

func ensureTransportSet(httpClient *http.Client, config ClientConfig) {
	if httpClient.Transport == nil {
		httpClient.Transport = newHTTPTransport(config)
	}
}

func setHTTPClientProxy(httpClient *http.Client, proxyURL *url.URL, config ClientConfig) {
	ensureTransportSet(httpClient, config)
	if transport, ok := httpClient.Transport.(*http.Transport); ok {
		transport.Proxy = http.ProxyURL(proxyURL)
	}
}

func setHTTPClientCertPool(httpClient *http.Client, certPool *x509.CertPool, config ClientConfig) {
	ensureTransportSet(httpClient, config)
	if transport, ok := httpClient.Transport.(*http.Transport); ok {
		transport.TLSClientConfig = &tls.Config{
			RootCAs:    certPool,
			MinVersion: tls.VersionTLS12,
		}
	}
}

// NewFromConnectionString creates new Client from the specified connection string.
// Parameters:
//   - connectionString: connection string in URL format.
//
// Supported query parameters:
//   - token - authentication token (required)
//   - authScheme - authentication scheme
//   - org - organization name
//   - database - database (bucket) name
//   - precision - timestamp precision when writing data
//   - gzipThreshold - payload size threshold for gzipping data
//   - writeNoSync - bool value whether to skip waiting for WAL persistence on write.
//     (See WriteOptions.NoSync for more details)
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
//   - INFLUX_AUTH_SCHEME - authentication scheme
//   - INFLUX_ORG - organization name
//   - INFLUX_DATABASE - database (bucket) name
//   - INFLUX_PRECISION - timestamp precision when writing data
//   - INFLUX_GZIP_THRESHOLD - payload size threshold for gzipping data
//   - INFLUX_WRITE_NO_SYNC - bool value whether to skip waiting for WAL persistence on write
//     (See WriteOptions.NoSync for more details)
//   - INFLUX_WRITE_TIMEOUT - duration value (e.g. 10s) to determine how long to wait for a write response
//   - INFLUX_QUERY_TIMEOUT - duration value (e.g. 10s) applied to queries for calculating a context response Deadline
func NewFromEnv() (*Client, error) {
	cfg := ClientConfig{}
	err := cfg.env()
	if err != nil {
		return nil, err
	}
	return New(cfg)
}

// GetServerVersion fetches the version of the server by parsing the response headers or body of the "/ping" endpoint.
func (c *Client) GetServerVersion() (string, error) {
	parse, _ := c.apiURL.Parse("/ping")
	r, err := c.makeAPICall(context.Background(), httpParams{
		endpointURL: parse,
		httpMethod:  "GET",
		headers:     http.Header{},
		body:        nil,
	})
	if err != nil {
		return "", err
	}
	defer func() {
		_ = r.Body.Close()
	}()

	v := r.Header.Get(strings.ToLower("X-Influxdb-Version"))
	if v == "" {
		var body []byte
		body, _ = io.ReadAll(r.Body)
		var versionResp struct {
			Version string `json:"version"`
		}
		err = json.Unmarshal(body, &versionResp)
		if err != nil {
			return v, err
		}
		v = versionResp.Version
	}

	return v, nil
}

// Close closes all idle connections.
func (c *Client) Close() error {
	c.config.HTTPClient.CloseIdleConnections()
	err := c.queryClient.Close()
	return err
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
		return nil, fmt.Errorf("error calling %s: %w", fullURL, err)
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
	if c.authorization != "" && req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", c.authorization)
	}

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error calling %s: %w", fullURL, err)
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
	defer func() {
		_ = r.Body.Close()
	}()

	var httpError struct {
		ServerError
		// InfluxDB V3 Core/Ent V3 write error message fields
		Error string `json:"error"`
		Data  []struct {
			ErrorMessage string `json:"error_message"`
			LineNumber   int    `json:"line_number"`
			OriginalLine string `json:"original_line"`
		} `json:"data"`
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
	if ctype == "application/json" || ctype == "" {
		err := json.Unmarshal(body, &httpError)
		if err != nil && ctype != "" {
			httpError.Message = fmt.Sprintf("cannot decode error response: %v", err)
		}
		if httpError.Error != "" {
			httpError.Message = httpError.Error
			for a, b := range httpError.Data {
				if a == 0 {
					httpError.Message += ":"
				}
				httpError.Message += fmt.Sprintf("\n\tline %d: %s (%s)", b.LineNumber, b.ErrorMessage, b.OriginalLine)
			}
		}
	}
	if httpError.Message == "" {
		if len(body) > 0 {
			httpError.Message = string(body)
		} else {
			httpError.Message = r.Status
		}
	}

	httpError.Headers = r.Header

	return &httpError.ServerError
}
