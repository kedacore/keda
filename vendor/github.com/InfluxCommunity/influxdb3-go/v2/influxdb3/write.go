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
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

// timeType is the exact type for the Time
var timeType = reflect.TypeFor[time.Time]()

// WritePoints writes all the given points to the server into the given database.
// The data is written synchronously. Empty batch is skipped.
// Points that serialize to an empty line (for example, all fields are nil/NaN/Inf)
// are skipped. If all points are skipped, no request is sent and no error is returned.
//
// Warning: If the provided slice contains only one Point, and that Point
// contains fields with nil/NaN/Inf values, those fields are not written to InfluxDB.
// If such fields are later queried explicitly, for example,
// "SELECT field_with_value, field_with_null_value FROM my_table" an error will be thrown.
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
	var precision Precision
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
	var tagOrder []string
	if options != nil && options.TagOrder != nil {
		tagOrder = options.TagOrder
	} else {
		tagOrder = c.config.WriteOptions.TagOrder
	}

	for _, p := range points {
		bts, err := p.marshalBinaryWithOptions(precision, defaultTags, tagOrder)
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
	if err := options.validate(); err != nil {
		return nil, err
	}

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
	if options.UseV2Api {
		u, _ = c.apiURL.Parse("v2/write")
		params = u.Query()
		params.Set("org", c.config.Organization)
		params.Set("bucket", database)
		params.Set("precision", precision.String())
	} else {
		u, _ = c.apiURL.Parse("v3/write_lp")
		params = u.Query()
		params.Set("org", c.config.Organization)
		params.Set("db", database)
		params.Set("precision", toV3PrecisionString(precision))
		if options.NoSync {
			params.Set("no_sync", "true")
		}
		if !options.AcceptPartial {
			params.Set("accept_partial", "false")
		}
	}
	u.RawQuery = params.Encode()
	body = bytes.NewReader(buff)
	headers := http.Header{"Content-Type": {"text/plain; charset=utf-8"}}
	if gzipThreshold > 0 && len(buff) >= gzipThreshold {
		r, err := compressWithGzip(buff)
		if err != nil {
			return nil, fmt.Errorf("unable to compress body: %w", err)
		}

		// The request body must be replayable so NewRequest can set GetBody.
		// compressWithGzip returns a replayable reader.
		// This is particularly useful for transient HTTP/2 errors and persistent connections.
		// Additionally, it helps manage graceful HTTP/2 shutdowns (e.g. GOAWAY frames).
		body = r

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
		if !options.UseV2Api && errors.As(err, &svErr) && svErr.StatusCode == http.StatusMethodNotAllowed &&
			strings.HasSuffix(params.endpointURL.Path, "/api/v3/write_lp") {
			return fmt.Errorf(
				"server doesn't support v3 write API (set WithUseV2Api(true); write options: {UseV2Api:%t,NoSync:%t,AcceptPartial:%t})",
				options.UseV2Api,
				options.NoSync,
				options.AcceptPartial,
			)
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
// During Point serialization, nil, NaN, +Inf, and -Inf field values are omitted.
// If a point has no remaining fields after filtering, it is skipped.
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

	return point.marshalBinaryWithOptions(options.Precision, options.DefaultTags, options.TagOrder)
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

func toV3PrecisionString(precision Precision) string {
	switch precision {
	case Nanosecond:
		return "nanosecond"
	case Microsecond:
		return "microsecond"
	case Millisecond:
		return "millisecond"
	case Second:
		return "second"
	}
	panic(fmt.Errorf("unknown precision value %d", precision))
}

// compressWithGzip compresses data and returns it as a replayable bytes.Reader.
func compressWithGzip(data []byte) (*bytes.Reader, error) {
	var compressed bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressed)
	if _, err := gzipWriter.Write(data); err != nil {
		_ = gzipWriter.Close()
		return nil, err
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, err
	}
	return bytes.NewReader(compressed.Bytes()), nil
}
