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
	"net/http"

	"github.com/influxdata/line-protocol/v2/lineprotocol"
	"google.golang.org/grpc"
)

// QueryOptions holds options for query
type QueryOptions struct {
	// Database for querying. Use to override default database in `ClientConfig`.
	Database string

	// Query type.
	QueryType QueryType

	// Headers to be included in requests. Use to add or override headers in `ClientConfig`.
	Headers http.Header

	// GRPC call options to be added
	GrpcCallOptions []grpc.CallOption
}

// WriteOptions holds options for write
type WriteOptions struct {
	// Database for writing. Use to override default database in `ClientConfig`.
	Database string

	// Precision of timestamp to use when writing data.
	// Default value: lineprotocol.Nanosecond
	Precision lineprotocol.Precision

	// Tags added to each point during writing. If a point already has a tag with the same key, it is left unchanged.
	DefaultTags map[string]string

	// Write body larger than the threshold is gzipped. 0 for no compression.
	GzipThreshold int

	// Instructs the server whether to wait with the response until WAL persistence completes.
	// NoSync=true means faster write but without the confirmation that the data was persisted.
	//
	// Note: This option is supported by InfluxDB 3 Core and Enterprise servers only.
	// For other InfluxDB 3 server types (InfluxDB Clustered, InfluxDB Clould Serverless/Dedicated)
	// the write operation will fail with an error.
	//
	// Default value: false.
	NoSync bool
}

// DefaultQueryOptions specifies default query options
var DefaultQueryOptions = QueryOptions{
	QueryType: SQL,
}

// DefaultWriteOptions specifies default write options
var DefaultWriteOptions = WriteOptions{
	Precision:     lineprotocol.Nanosecond,
	GzipThreshold: 1_000,
	NoSync:        false,
}

// Option is a functional option type that can be passed to Client.Query and Client.Write methods.
type Option func(o *options)

// QueryOption is a functional option type that can be passed to Client.Query.
// Available options:
//   - WithDatabase
//   - WithQueryType
//   - WithHeader
//   - WithGrpcCallOption
type QueryOption = Option

// WriteOption is a functional option type that can be passed to Client.Write methods.
// Available options:
//   - WithDatabase
//   - WithPrecision
//   - WithGzipThreshold
//   - WithDefaultTags
//   - WithNoSync
type WriteOption = Option

// WithDatabase is used to override default database in Client.Query and Client.Write methods.
func WithDatabase(database string) Option {
	return func(o *options) {
		o.QueryOptions.Database = database
		o.WriteOptions.Database = database
	}
}

// WithQueryType is used to override default query type in Client.Query method.
func WithQueryType(queryType QueryType) Option {
	return func(o *options) {
		o.QueryType = queryType
	}
}

// WithHeader is used to add or override default header in Client.Query method.
func WithHeader(key, value string) Option {
	return func(o *options) {
		if o.Headers == nil {
			o.Headers = make(http.Header, 0)
		}
		o.Headers[key] = []string{value}
	}
}

// WithPrecision is used to override default precision in Client.Write methods.
func WithPrecision(precision lineprotocol.Precision) Option {
	return func(o *options) {
		o.Precision = precision
	}
}

// WithGzipThreshold is used to override default GZIP threshold in Client.Write methods.
func WithGzipThreshold(gzipThreshold int) Option {
	return func(o *options) {
		o.GzipThreshold = gzipThreshold
	}
}

// WithDefaultTags is used to override default tags in Client.Write methods.
func WithDefaultTags(tags map[string]string) Option {
	return func(o *options) {
		o.DefaultTags = tags
	}
}

// WithNoSync is used to override default tags in Client.Write methods.
func WithNoSync(noSync bool) Option {
	return func(o *options) {
		o.NoSync = noSync
	}
}

// WithGrpcCallOption is used to send GRPC call options to the underlying Flight client
//
// Example:
//
//	qIter, qErr := client.Query(context.Background(),
//	    "SELECT * FROM examples",
//	    WithGrpcCallOption(grpc.MaxCallRecvMsgSize(5_000_000)),
//	   )
//
// For more information see https://pkg.go.dev/google.golang.org/grpc#CallOption
func WithGrpcCallOption(grpcCallOption grpc.CallOption) Option {
	return func(o *options) {
		o.QueryOptions.GrpcCallOptions = append(o.QueryOptions.GrpcCallOptions, grpcCallOption)
	}
}

type options struct {
	QueryOptions
	WriteOptions
}

func newQueryOptions(defaults *QueryOptions, opts []Option) *QueryOptions {
	return &(newOptions(defaults, nil, opts).QueryOptions)
}

func newWriteOptions(defaults *WriteOptions, opts []Option) *WriteOptions {
	return &(newOptions(nil, defaults, opts).WriteOptions)
}

func newOptions(defaultQueryOptions *QueryOptions, defaultWriteOptions *WriteOptions, opts []Option) *options {
	o := &options{}

	if defaultQueryOptions != nil {
		o.QueryOptions = *defaultQueryOptions
	}
	if defaultWriteOptions != nil {
		o.WriteOptions = *defaultWriteOptions
	}

	for _, opt := range opts {
		opt(o)
	}

	return o
}
