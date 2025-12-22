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
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"github.com/apache/arrow-go/v18/arrow/flight"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
)

func (c *Client) initializeQueryClient(hostPortURL string, certPool *x509.CertPool, proxyURL *url.URL) error {
	var transport grpc.DialOption

	if certPool != nil {
		transport = grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(certPool, ""))
	} else {
		transport = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	opts := []grpc.DialOption{
		transport,
	}

	if proxyURL != nil {
		// Configure the grpc-go client to use a proxy by setting the HTTPS_PROXY environment variable.
		// This approach is generally safer than implementing a custom Dialer because it leverages built-in
		// proxy handling, reducing the risk of introducing vulnerabilities or misconfigurations.
		// More info: https://github.com/grpc/grpc-go/blob/master/Documentation/proxy.md
		prevHTTPSProxy := os.Getenv("HTTPS_PROXY")
		if prevHTTPSProxy != "" && prevHTTPSProxy != proxyURL.String() {
			slog.Warn(
				fmt.Sprintf("Environment variable HTTPS_PROXY is already set, "+
					"it's value will be overridden with: %s", proxyURL.String()),
			)
		}
		err := os.Setenv("HTTPS_PROXY", proxyURL.String())
		if err != nil {
			return fmt.Errorf("setenv HTTPS_PROXY: %w", err)
		}
	}

	client, err := flight.NewClientWithMiddleware(hostPortURL, nil, nil, opts...)
	if err != nil {
		return fmt.Errorf("flight: %w", err)
	}
	c.queryClient = client

	return nil
}

func (c *Client) setQueryClient(flightClient flight.Client) {
	c.queryClient = flightClient
}

// QueryParameters is a type for query parameters.
type QueryParameters = map[string]any

// Query queries data from InfluxDB v3.
// Parameters:
//   - ctx: The context.Context to use for the request.
//   - query: The query string to execute.
//   - options: The optional query options. See QueryOption for available options.
//
// Returns:
//   - A result iterator (*QueryIterator).
//   - An error, if any.
func (c *Client) Query(ctx context.Context, query string, options ...QueryOption) (*QueryIterator, error) {
	return c.query(ctx, query, nil, newQueryOptions(&DefaultQueryOptions, options))
}

// QueryPointValue queries data from InfluxDB v3.
// Parameters:
//   - ctx: The context.Context to use for the request.
//   - query: The query string to execute.
//   - options: The optional query options. See QueryOption for available options.
//
// Returns:
//   - A result iterator (*PointValueIterator).
//   - An error, if any.
func (c *Client) QueryPointValue(ctx context.Context, query string, options ...QueryOption) (*PointValueIterator, error) {
	return c.queryPointValue(ctx, query, nil, newQueryOptions(&DefaultQueryOptions, options))
}

// QueryWithParameters queries data from InfluxDB v3 with parameterized query.
// Parameters:
//   - ctx: The context.Context to use for the request.
//   - query: The query string to execute.
//   - parameters: The query parameters.
//   - options: The optional query options. See QueryOption for available options.
//
// Returns:
//   - A result iterator (*QueryIterator).
//   - An error, if any.
func (c *Client) QueryWithParameters(ctx context.Context, query string, parameters QueryParameters,
	options ...QueryOption) (*QueryIterator, error) {
	return c.query(ctx, query, parameters, newQueryOptions(&DefaultQueryOptions, options))
}

// QueryPointValueWithParameters queries data from InfluxDB v3 with parameterized query.
// Parameters:
//   - ctx: The context.Context to use for the request.
//   - query: The query string to execute.
//   - parameters: The query parameters.
//   - options: The optional query options. See QueryOption for available options.
//
// Returns:
//   - A result iterator (*PointValueIterator).
//   - An error, if any.
func (c *Client) QueryPointValueWithParameters(ctx context.Context, query string, parameters QueryParameters,
	options ...QueryOption) (*PointValueIterator, error) {
	return c.queryPointValue(ctx, query, parameters, newQueryOptions(&DefaultQueryOptions, options))
}

// QueryWithOptions Query data from InfluxDB v3 with query options.
// Parameters:
//   - ctx: The context.Context to use for the request.
//   - options: Query options (query type, optional database).
//   - query: The query string to execute.
//
// Returns:
//   - A result iterator (*QueryIterator).
//   - An error, if any.
//
// Deprecated: use Query with variadic QueryOption options.
func (c *Client) QueryWithOptions(ctx context.Context, options *QueryOptions, query string) (*QueryIterator, error) {
	if options == nil {
		return nil, errors.New("options not set")
	}

	return c.query(ctx, query, nil, options)
}

func (c *Client) query(ctx context.Context, query string, parameters QueryParameters, options *QueryOptions) (*QueryIterator, error) {
	reader, err := c.getReader(ctx, query, parameters, options)
	if err != nil {
		return nil, err
	}

	return NewQueryIteratorFromReader(reader), nil
}

func (c *Client) queryPointValue(ctx context.Context, query string, parameters QueryParameters, options *QueryOptions) (*PointValueIterator, error) {
	reader, err := c.getReader(ctx, query, parameters, options)
	if err != nil {
		return nil, err
	}

	return NewPointValueIteratorFomReader(reader), nil
}

func (c *Client) getReader(ctx context.Context, query string, parameters QueryParameters, options *QueryOptions) (RecordReader, error) { //nolint:ireturn
	var database string
	if options.Database != "" {
		database = options.Database
	} else {
		database = c.config.Database
	}
	if database == "" {
		return nil, errors.New("database not specified")
	}

	var queryType = options.QueryType

	md := make(metadata.MD, 0)
	for k, v := range c.config.Headers {
		for _, value := range v {
			md.Append(k, value)
		}
	}
	for k, v := range options.Headers {
		for _, value := range v {
			md.Append(k, value)
		}
	}
	md.Set("authorization", "Bearer "+c.config.Token)
	md.Set("User-Agent", userAgent)
	ctx = metadata.NewOutgoingContext(ctx, md)

	ticketData := map[string]any{
		"database":   database,
		"sql_query":  query,
		"query_type": strings.ToLower(queryType.String()),
	}

	if len(parameters) > 0 {
		ticketData["params"] = parameters
	}

	ticketJSON, err := json.Marshal(ticketData)
	if err != nil {
		return nil, fmt.Errorf("serialize: %w", err)
	}

	ticket := &flight.Ticket{Ticket: ticketJSON}

	grpcCallOptions := make([]grpc.CallOption, 0)
	if options.GrpcCallOptions != nil {
		grpcCallOptions = append(grpcCallOptions, options.GrpcCallOptions...)
	}

	var _ctx context.Context
	var cancel context.CancelFunc

	if c.config.QueryTimeout > 0 {
		_ctx, cancel = context.WithTimeout(ctx, c.config.QueryTimeout)
	} else {
		_ctx = ctx
	}

	stream, err := c.queryClient.DoGet(_ctx, ticket, grpcCallOptions...)
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return nil, fmt.Errorf("flight do get: %w", err)
	}

	reader, err := flight.NewRecordReader(stream, ipc.WithAllocator(memory.DefaultAllocator))
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return nil, fmt.Errorf("flight reader: %w", err)
	}

	if cancel == nil {
		return reader, nil
	}
	return &cancelingRecordReader{reader: reader, cancel: cancel}, nil
}
