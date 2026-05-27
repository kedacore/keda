// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.
//
// Modifications Copyright OpenSearch Contributors. See
// GitHub history for details.

// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/opensearch-project/opensearch-go/v4/internal/version"
	"github.com/opensearch-project/opensearch-go/v4/opensearchtransport"
	"github.com/opensearch-project/opensearch-go/v4/signer"
)

const (
	defaultURL         = "http://localhost:9200"
	openSearch         = "opensearch"
	unsupportedProduct = "the client noticed that the server is not a supported distribution"
	envOpenSearchURL   = "OPENSEARCH_URL"
)

// Version returns the package version as a string.
const Version = version.Client

// Error vars
var (
	ErrCreateClient                        = errors.New("cannot create client")
	ErrCreateTransport                     = errors.New("error creating transport")
	ErrParseVersion                        = errors.New("failed to parse opensearch version")
	ErrParseURL                            = errors.New("cannot parse url")
	ErrTransportMissingMethodMetrics       = errors.New("transport is missing method Metrics()")
	ErrTransportMissingMethodDiscoverNodes = errors.New("transport is missing method DiscoverNodes()")
)

// Config represents the client configuration.
type Config struct {
	Addresses []string // A list of nodes to use.
	Username  string   // Username for HTTP Basic Authentication.
	Password  string   // Password for HTTP Basic Authentication.

	Header http.Header // Global HTTP request header.

	Signer signer.Signer

	// PEM-encoded certificate authorities.
	// When set, an empty certificate pool will be created, and the certificates will be appended to it.
	// The option is only valid when the transport is not specified, or when it's http.Transport.
	CACert []byte

	RetryOnStatus        []int // List of status codes for retry. Default: 502, 503, 504.
	DisableRetry         bool  // Default: false.
	EnableRetryOnTimeout bool  // Default: false.
	MaxRetries           int   // Default: 3.

	CompressRequestBody bool // Default: false.

	DiscoverNodesOnStart  bool          // Discover nodes when initializing the client. Default: false.
	DiscoverNodesInterval time.Duration // Discover nodes periodically. Default: disabled.

	EnableMetrics     bool // Enable the metrics collection.
	EnableDebugLogger bool // Enable the debug logging.

	RetryBackoff func(attempt int) time.Duration // Optional backoff duration. Default: nil.

	Transport http.RoundTripper            // The HTTP transport object.
	Logger    opensearchtransport.Logger   // The logger object.
	Selector  opensearchtransport.Selector // The selector object.

	// Optional constructor function for a custom ConnectionPool. Default: nil.
	ConnectionPoolFunc func([]*opensearchtransport.Connection, opensearchtransport.Selector) opensearchtransport.ConnectionPool
}

// Client represents the OpenSearch client.
type Client struct {
	Transport opensearchtransport.Interface
}

// NewDefaultClient creates a new client with default options.
//
// It will use http://localhost:9200 as the default address.
//
// It will use the OPENSEARCH_URL/ELASTICSEARCH_URL environment variable, if set,
// to configure the addresses; use a comma to separate multiple URLs.
//
// It's an error to set both OPENSEARCH_URL and ELASTICSEARCH_URL.
func NewDefaultClient() (*Client, error) {
	return NewClient(Config{})
}

// NewClient creates a new client with configuration from cfg.
//
// It will use http://localhost:9200 as the default address.
//
// It will use the OPENSEARCH_URL/ELASTICSEARCH_URL environment variable, if set,
// to configure the addresses; use a comma to separate multiple URLs.
//
// It's an error to set both OPENSEARCH_URL and ELASTICSEARCH_URL.
func NewClient(cfg Config) (*Client, error) {
	var addrs []string

	if len(cfg.Addresses) == 0 {
		envAddress := getAddressFromEnvironment()
		addrs = envAddress
	} else {
		addrs = append(addrs, cfg.Addresses...)
	}

	urls, err := addrsToURLs(addrs)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateClient, err)
	}

	if len(urls) == 0 {
		//nolint:errcheck // errcheck exclude ???
		u, _ := url.Parse(defaultURL)
		urls = append(urls, u)
	}

	// TODO: Refactor
	if urls[0].User != nil {
		cfg.Username = urls[0].User.Username()
		pw, _ := urls[0].User.Password()
		cfg.Password = pw
	}

	tp, err := opensearchtransport.New(opensearchtransport.Config{
		URLs:     urls,
		Username: cfg.Username,
		Password: cfg.Password,

		Header: cfg.Header,
		CACert: cfg.CACert,

		Signer: cfg.Signer,

		RetryOnStatus:        cfg.RetryOnStatus,
		DisableRetry:         cfg.DisableRetry,
		EnableRetryOnTimeout: cfg.EnableRetryOnTimeout,
		MaxRetries:           cfg.MaxRetries,
		RetryBackoff:         cfg.RetryBackoff,

		CompressRequestBody: cfg.CompressRequestBody,

		EnableMetrics:     cfg.EnableMetrics,
		EnableDebugLogger: cfg.EnableDebugLogger,

		DiscoverNodesInterval: cfg.DiscoverNodesInterval,

		Transport:          cfg.Transport,
		Logger:             cfg.Logger,
		Selector:           cfg.Selector,
		ConnectionPoolFunc: cfg.ConnectionPoolFunc,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateTransport, err)
	}

	client := &Client{Transport: tp}

	if cfg.DiscoverNodesOnStart {
		//nolint:errcheck // goroutine discards return values
		go client.DiscoverNodes()
	}

	return client, err
}

func getAddressFromEnvironment() []string {
	return addrsFromEnvironment(envOpenSearchURL)
}

// ParseVersion returns an int64 representation of version.
func ParseVersion(version string) (int64, int64, int64, error) {
	reVersion := regexp.MustCompile(`^([0-9]+)\.([0-9]+)\.([0-9]+)`)
	matches := reVersion.FindStringSubmatch(version)
	//nolint:gomnd // 4 is the minium regexp match length
	if len(matches) < 4 {
		return 0, 0, 0, fmt.Errorf("%w: regexp does not match on version string", ErrParseVersion)
	}

	major, err := strconv.ParseInt(matches[1], 10, 0)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("%w: %w", ErrParseVersion, err)
	}

	minor, err := strconv.ParseInt(matches[2], 10, 0)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("%w: %w", ErrParseVersion, err)
	}

	patch, err := strconv.ParseInt(matches[3], 10, 0)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("%w: %w", ErrParseVersion, err)
	}

	return major, minor, patch, nil
}

// Perform delegates to Transport to execute a request and return a response.
func (c *Client) Perform(req *http.Request) (*http.Response, error) {
	// Perform the original request.
	return c.Transport.Perform(req)
}

// Do gets and performs the request. It also tries to parse the response into the dataPointer
func (c *Client) Do(ctx context.Context, req Request, dataPointer interface{}) (*Response, error) {
	httpReq, err := req.GetRequest()
	if err != nil {
		return nil, err
	}

	if ctx != nil {
		httpReq = httpReq.WithContext(ctx)
	}

	//nolint:bodyclose // body got already closed by Perform, this is a nopcloser
	resp, err := c.Perform(httpReq)
	if err != nil {
		return nil, err
	}

	response := &Response{
		StatusCode: resp.StatusCode,
		Body:       resp.Body,
		Header:     resp.Header,
	}

	if dataPointer != nil && resp.Body != nil && !response.IsError() {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return response, fmt.Errorf("%w, status: %d, err: %w", ErrReadBody, resp.StatusCode, err)
		}

		response.Body = io.NopCloser(bytes.NewReader(data))

		if err := json.Unmarshal(data, dataPointer); err != nil {
			return response, fmt.Errorf("%w, status: %d, body: %s, err: %w", ErrJSONUnmarshalBody, resp.StatusCode, data, err)
		}
	}

	return response, nil
}

// Metrics returns the client metrics.
func (c *Client) Metrics() (opensearchtransport.Metrics, error) {
	if mt, ok := c.Transport.(opensearchtransport.Measurable); ok {
		return mt.Metrics()
	}

	return opensearchtransport.Metrics{}, ErrTransportMissingMethodMetrics
}

// DiscoverNodes reloads the client connections by fetching information from the cluster.
func (c *Client) DiscoverNodes() error {
	if dt, ok := c.Transport.(opensearchtransport.Discoverable); ok {
		return dt.DiscoverNodes()
	}

	return ErrTransportMissingMethodDiscoverNodes
}

// addrsFromEnvironment returns a list of addresses by splitting
// the given environment variable with comma, or an empty list.
func addrsFromEnvironment(name string) []string {
	var addrs []string

	if envURLs, ok := os.LookupEnv(name); ok && envURLs != "" {
		list := strings.Split(envURLs, ",")
		addrs = make([]string, len(list))

		for idx, u := range list {
			addrs[idx] = strings.TrimSpace(u)
		}
	}

	return addrs
}

// addrsToURLs creates a list of url.URL structures from url list.
func addrsToURLs(addrs []string) ([]*url.URL, error) {
	urls := make([]*url.URL, 0)

	for _, addr := range addrs {
		u, err := url.Parse(strings.TrimRight(addr, "/"))
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrParseURL, err)
		}

		urls = append(urls, u)
	}

	return urls, nil
}

// ToPointer converts any value to a pointer, mainly used for request parameters
func ToPointer[V any](value V) *V {
	return &value
}
