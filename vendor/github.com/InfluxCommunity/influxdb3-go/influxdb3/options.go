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
	"github.com/influxdata/line-protocol/v2/lineprotocol"
)

// QueryOptions holds options for query
type QueryOptions struct {
	// Database for querying. Use in `QueryWithOptions` method to override default database in `ClientConfig`.
	Database string

	// Query type.
	QueryType QueryType
}

// WriteOptions holds options for write
type WriteOptions struct {
	// Database for writing. Use in `WriteWithOptions` methods to override default database in `ClientConfig`.
	Database string

	// Precision to use in writes for timestamp.
	// Default `lineprotocol.Nanosecond`
	Precision lineprotocol.Precision

	// Tags added to each point during writing. If a point already has a tag with the same key, it is left unchanged.
	// 
	// Example using WritePointsWithOptions:
	//  c, _ := New(ClientConfig{
	//  	Host:         "host",
	//  	Token:        "my-token",
	//  	Organization: "my-org",
	//  	Database:     "my-database",
	//  })
	//
	//  options := WriteOptions{
	//  	defaultTags: map[string]string{ 
	//  		"rack": "main",
	//  	},
	//  	Precision: lineprotocol.Millisecond,
	//  }
	//
	//  p := NewPointWithMeasurement("measurement")
	//  p.SetField("number", 10)
	//
	//  // Writes with rack=main tag
	//  c.WritePointsWithOptions(context.Background(), &options, p)
	//
	// Example using ClientConfig:
	//  c, _ := New(ClientConfig{
	//  	Host:         "host",
	//  	Token:        "my-token",
	//  	Organization: "my-org",
	//  	Database:     "my-database",
	//  	WriteOptions: &WriteOptions{
	//  		defaultTags: map[string]string{ 
	//  			"rack": "main",
	//  		},
	//  		Precision: lineprotocol.Millisecond,
	//  	},
	//  })
	//
	//  p := NewPointWithMeasurement("measurement")
	//  p.SetField("number", 10)
	//
	//  // Writes with rack=main tag
	//  c.WritePoints(context.Background(), p)
	defaultTags map[string]string

	// Write body larger than the threshold is gzipped. 0 to don't gzip at all
	GzipThreshold int
}

// DefaultQueryOptions specifies default query options
var DefaultQueryOptions = QueryOptions{
	QueryType: FlightSQL,
}

// DefaultWriteOptions specifies default write options
var DefaultWriteOptions = WriteOptions{
	Precision:     lineprotocol.Nanosecond,
	GzipThreshold: 1_000,
}
