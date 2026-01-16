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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3/gzip"
	"github.com/influxdata/line-protocol/v2/lineprotocol"
)

// timeType is the exact type for the Time
var timeType = reflect.TypeFor[time.Time]()

// WritePoints writes all the given points to the server into the given database.
// The data is written synchronously. Empty batch is skipped.
//
// Parameters:
//   - ctx: The context.Context to use for the request.
//   - points: The points to write.
//   - options: Optional write options. See WriteOption for available options.
//
// Returns:
//   - An error, if any.
func (c *Client) WritePoints(ctx context.Context, points []*Point, options ...WriteOption) error {
	return c.writePoints(ctx, points, newWriteOptions(c.config.WriteOptions, options))
}

// WritePointsWithOptions writes all the given points to the server into the given database.
// The data is written synchronously. Empty batch is skipped.
//
// Parameters:
//   - ctx: The context.Context to use for the request.
//   - points: The points to write.
//   - options: Write options.
//
// Returns:
//   - An error, if any.
//
// Deprecated: use WritePoints with variadic WriteOption options.
func (c *Client) WritePointsWithOptions(ctx context.Context, options *WriteOptions, points ...*Point) error {
	if options == nil {
		return errors.New("options not set")
	}

	return c.writePoints(ctx, points, options)
}

func (c *Client) writePoints(ctx context.Context, points []*Point, options *WriteOptions) error {
	var buff []byte
	var precision lineprotocol.Precision
	if options != nil {
		precision = options.Precision
	} else {
		precision = c.config.WriteOptions.Precision
	}
	var defaultTags map[string]string
	if options != nil && options.DefaultTags != nil {
		defaultTags = options.DefaultTags
	} else {
		defaultTags = c.config.WriteOptions.DefaultTags
	}

	for _, p := range points {
		bts, err := p.MarshalBinaryWithDefaultTags(precision, defaultTags)
		if err != nil {
			return err
		}
		buff = append(buff, bts...)
	}

	return c.write(ctx, buff, options)
}

// Write writes line protocol record(s) to the server into the given database.
// Multiple records must be separated by the new line character (\n).
// The data is written synchronously. Empty buff is skipped.
//
// Parameters:
//   - ctx: The context.Context to use for the request.
//   - buff: The line protocol record(s) to write.
//   - options: Optional write options. See WriteOption for available options.
//
// Returns:
//   - An error, if any.
func (c *Client) Write(ctx context.Context, buff []byte, options ...WriteOption) error {
	return c.write(ctx, buff, newWriteOptions(c.config.WriteOptions, options))
}

// WriteWithOptions writes line protocol record(s) to the server into the given database.
// Multiple records must be separated by the new line character (\n).
// The data is written synchronously. Empty buff is skipped.
//
// Parameters:
//   - ctx: The context.Context to use for the request.
//   - buff: The line protocol record(s) to write.
//   - options: Write options.
//
// Returns:
//   - An error, if any.
//
// Deprecated: use Write with variadic WriteOption option
func (c *Client) WriteWithOptions(ctx context.Context, options *WriteOptions, buff []byte) error {
	if options == nil {
		return errors.New("options not set")
	}

	return c.write(ctx, buff, options)
}

func (c *Client) makeHTTPParams(buff []byte, options *WriteOptions) (*httpParams, error) {
	var database string
	if options.Database != "" {
		database = options.Database
	} else {
		database = c.config.Database
	}
	if database == "" {
		return nil, errors.New("database not specified")
	}

	var precision = options.Precision

	var gzipThreshold = options.GzipThreshold

	var body io.Reader
	var u *url.URL
	var params url.Values
	if options.NoSync {
		// Setting no_sync=true is supported only in the v3 API.
		u, _ = c.apiURL.Parse("v3/write_lp")
		params = u.Query()
		params.Set("org", c.config.Organization)
		params.Set("db", database)
		params.Set("precision", toV3PrecisionString(precision))
		params.Set("no_sync", "true")
	} else {
		// By default, use the v2 API.
		u, _ = c.apiURL.Parse("v2/write")
		params = u.Query()
		params.Set("org", c.config.Organization)
		params.Set("bucket", database)
		params.Set("precision", precision.String())
	}
	u.RawQuery = params.Encode()
	body = bytes.NewReader(buff)
	headers := http.Header{"Content-Type": {"application/json"}}
	if gzipThreshold > 0 && len(buff) >= gzipThreshold {
		r, err := gzip.CompressWithGzip(body)
		if err != nil {
			return nil, fmt.Errorf("unable to compress body: %w", err)
		}

		// This is necessary for Request.GetBody to be set by NewRequest, ensuring that
		// the Transport can retry the request when a network error occurs.
		// See: https://github.com/golang/go/blob/726d898c92ed0159f283f324478d00f15419f476/src/net/http/request.go#L884
		// See: https://github.com/golang/go/blob/726d898c92ed0159f283f324478d00f15419f476/src/net/http/transport.go#L89-L92
		//
		// It is particularly useful for handling transient errors in HTTP/2 and persistent
		// connections in standard HTTP.
		// Additionally, it helps manage graceful HTTP/2 shutdowns (e.g. GOAWAY frames).
		b, err := io.ReadAll(r)
		if err != nil {
			return nil, fmt.Errorf("unable to read compressed body: %w", err)
		}
		body = bytes.NewReader(b)

		headers["Content-Encoding"] = []string{"gzip"}
	}

	return &httpParams{
		endpointURL: u,
		httpMethod:  "POST",
		headers:     headers,
		queryParams: u.Query(),
		body:        body,
	}, nil
}

func (c *Client) write(ctx context.Context, buff []byte, options *WriteOptions) error {
	// Skip zero size batch
	if len(buff) == 0 {
		return nil
	}

	params, err := c.makeHTTPParams(buff, options)
	if err != nil {
		return err
	}

	resp, err := c.makeAPICall(ctx, *params)
	if err != nil {
		var svErr *ServerError
		if options.NoSync && errors.As(err, &svErr) && svErr.StatusCode == http.StatusMethodNotAllowed &&
			strings.HasSuffix(params.endpointURL.Path, "/api/v3/write_lp") {
			// Server does not support the v3 write API, can't use the NoSync option.
			return errors.New("server doesn't support write with NoSync=true (supported by InfluxDB 3 Core/Enterprise servers only)")
		}
		return err
	}
	return resp.Body.Close()
}

// WriteData encodes fields of custom points into line protocol
// and writes line protocol record(s) to the server into the given database.
// Each custom point must be annotated with 'lp' prefix and Values measurement, tag, field, or timestamp.
// A valid point must contain a measurement and at least one field.
// The points are written synchronously. Empty batch is skipped.
//
// A field with a timestamp must be of type time.Time.
//
// Parameters:
//   - ctx: The context.Context to use for the request.
//   - points: The custom points to encode and write.
//   - options: Optional write options. See WriteOption for available options.
//
// Returns:
//   - An error, if any.
func (c *Client) WriteData(ctx context.Context, points []any, options ...WriteOption) error {
	return c.writeData(ctx, points, newWriteOptions(c.config.WriteOptions, options))
}

// WriteDataWithOptions encodes fields of custom points into line protocol
// and writes line protocol record(s) to the server into the given database.
// Each custom point must be annotated with 'lp' prefix and Values measurement, tag, field, or timestamp.
// A valid point must contain a measurement and at least one field.
// The points are written synchronously. Empty batch is skipped.
//
// A field with a timestamp must be of type time.Time.
//
// Parameters:
//   - ctx: The context.Context to use for the request.
//   - points: The custom points to encode and write.
//   - options: Write options.
//
// Returns:
//   - An error, if any.
//
// Deprecated: use WriteData with variadic WriteOption option
func (c *Client) WriteDataWithOptions(ctx context.Context, options *WriteOptions, points ...any) error {
	if options == nil {
		return errors.New("options not set")
	}

	return c.writeData(ctx, points, options)
}

func (c *Client) writeData(ctx context.Context, points []any, options *WriteOptions) error {
	var buff []byte
	for _, p := range points {
		b, err := encode(p, options)
		if err != nil {
			return fmt.Errorf("error encoding point: %w", err)
		}
		buff = append(buff, b...)
	}

	return c.write(ctx, buff, options)
}

func encode(x any, options *WriteOptions) ([]byte, error) {
	t := reflect.TypeOf(x)
	v := reflect.ValueOf(x)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("cannot use %v as point", t)
	}

	fields := reflect.VisibleFields(t)

	var point = &Point{
		Values: &PointValues{
			Tags:   make(map[string]string),
			Fields: make(map[string]any),
		},
	}

	for _, f := range fields {
		tag, ok := f.Tag.Lookup("lp")
		if !ok || tag == "-" {
			continue
		}
		parts := strings.Split(tag, ",")
		if len(parts) > 2 {
			return nil, errors.New("multiple tag attributes are not supported")
		}
		typ, name := parts[0], f.Name
		if len(parts) == 2 {
			name = parts[1]
		}
		field := v.FieldByIndex(f.Index)
		switch typ {
		case "measurement":
			if point.GetMeasurement() != "" {
				return nil, errors.New("multiple measurement fields")
			}
			point.SetMeasurement(field.String())
		case "tag":
			point.SetTag(name, field.String())
		case "field":
			fieldVal, err := fieldValue(name, f, field, t)
			if err != nil {
				return nil, err
			}
			point.SetField(name, fieldVal)
		case "timestamp":
			if f.Type != timeType {
				return nil, fmt.Errorf("cannot use field '%s' as a timestamp", f.Name)
			}
			point.SetTimestamp(field.Interface().(time.Time))
		default:
			return nil, fmt.Errorf("invalid tag %s", typ)
		}
	}
	if point.GetMeasurement() == "" {
		return nil, errors.New("no struct field with tag 'measurement'")
	}
	if !point.HasFields() {
		return nil, errors.New("no struct field with tag 'field'")
	}

	return point.MarshalBinaryWithDefaultTags(options.Precision, options.DefaultTags)
}

func fieldValue(name string, f reflect.StructField, v reflect.Value, t reflect.Type) (any, error) {
	var fieldVal any
	if f.IsExported() {
		fieldVal = v.Interface()
	} else {
		switch v.Kind() {
		case reflect.Bool:
			fieldVal = v.Bool()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			fieldVal = v.Int()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			fieldVal = v.Uint()
		case reflect.Float32, reflect.Float64:
			fieldVal = v.Float()
		case reflect.String:
			fieldVal = v.String()
		default:
			return nil, fmt.Errorf("cannot use field '%s' of type '%v' as a field", name, t)
		}
	}
	return fieldVal, nil
}

func toV3PrecisionString(precision lineprotocol.Precision) string {
	switch precision {
	case lineprotocol.Nanosecond:
		return "nanosecond"
	case lineprotocol.Microsecond:
		return "microsecond"
	case lineprotocol.Millisecond:
		return "millisecond"
	case lineprotocol.Second:
		return "second"
	}
	panic(fmt.Errorf("unknown precision: %v", precision))
}
