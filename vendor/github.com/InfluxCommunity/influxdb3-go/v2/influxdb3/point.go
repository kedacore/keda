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
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/influxdata/line-protocol/v2/lineprotocol"
)

// Point represents InfluxDB time series point, holding tags and fields
type Point struct {
	Values *PointValues

	// fieldConverter this converter function must return one of these types supported by InfluxDB int64, uint64, float64, bool, string, []byte.
	fieldConverter func(interface{}) interface{}
}

// NewPointWithPointValues returns a new Point with given PointValues.
func NewPointWithPointValues(values *PointValues) *Point {
	return &Point{Values: values}
}

// NewPoint is a convenient function for creating a Point with the given measurement name, tags, fields, and timestamp.
//
// Parameters:
//   - measurement: The measurement name for the Point.
//   - tags: The tags for the Point.
//   - fields: The fields for the Point.
//   - ts: The timestamp for the Point.
//
// Returns:
//   - The created Point.
func NewPoint(measurement string, tags map[string]string, fields map[string]interface{}, ts time.Time) *Point {
	m := NewPointWithMeasurement(measurement)
	if len(tags) > 0 {
		for k, v := range tags {
			m.SetTag(k, v)
		}
	}
	if len(fields) > 0 {
		for k, v := range fields {
			m.SetField(k, v)
		}
	}
	m.SetTimestamp(ts)
	return m
}

// NewPointWithMeasurement creates a new PointData with specified measurement name.
func NewPointWithMeasurement(name string) *Point {
	return NewPointWithPointValues(NewPointValues(name))
}

// FromValues creates a new PointData with given Values.
func FromValues(values *PointValues) (*Point, error) {
	if values.GetMeasurement() == "" {
		return nil, errors.New("missing measurement")
	}
	return NewPointWithPointValues(values), nil
}

// GetMeasurement returns the measurement name.
// It will return null when querying with SQL Query.
func (p *Point) GetMeasurement() string {
	return p.Values.GetMeasurement()
}

// SetMeasurement sets the measurement name and returns the modified PointValues
func (p *Point) SetMeasurement(measurementName string) *Point {
	if measurementName != "" {
		p.Values.SetMeasurement(measurementName)
	}
	return p
}

// SetTimestampWithEpoch sets the timestamp using an int64 which represents start of epoch in nanoseconds and returns the modified PointDataValues
func (p *Point) SetTimestampWithEpoch(timestamp int64) *Point {
	p.Values.SetTimestampWithEpoch(timestamp)
	return p
}

// SetTimestamp sets the timestamp using a time.Time and returns the modified PointValues
func (p *Point) SetTimestamp(timestamp time.Time) *Point {
	p.Values.SetTimestamp(timestamp)
	return p
}

// SetTag sets a tag value and returns the modified PointValues
func (p *Point) SetTag(name string, value string) *Point {
	p.Values.SetTag(name, value)
	return p
}

// GetTag retrieves a tag value
func (p *Point) GetTag(name string) (string, bool) {
	return p.Values.GetTag(name)
}

// RemoveTag Removes a tag with the specified name if it exists; otherwise, it does nothing.
func (p *Point) RemoveTag(name string) *Point {
	p.Values.RemoveTag(name)
	return p
}

// GetTagNames retrieves all tag names
func (p *Point) GetTagNames() []string {
	return p.Values.GetTagNames()
}

// GetDoubleField gets the double field value associated with the specified name.
// If the field is not present, returns nil.
func (p *Point) GetDoubleField(name string) *float64 {
	return p.Values.GetDoubleField(name)
}

// SetDoubleField adds or replaces a double field.
func (p *Point) SetDoubleField(name string, value float64) *Point {
	p.Values.SetDoubleField(name, value)
	return p
}

// GetIntegerField gets the integer field value associated with the specified name.
// If the field is not present, returns nil.
func (p *Point) GetIntegerField(name string) *int64 {
	return p.Values.GetIntegerField(name)
}

// SetIntegerField adds or replaces an integer field.
func (p *Point) SetIntegerField(name string, value int64) *Point {
	p.Values.SetIntegerField(name, value)
	return p
}

// GetUIntegerField gets the uinteger field value associated with the specified name.
// If the field is not present, returns nil.
func (p *Point) GetUIntegerField(name string) *uint64 {
	return p.Values.GetUIntegerField(name)
}

// SetUIntegerField adds or replaces an unsigned integer field.
func (p *Point) SetUIntegerField(name string, value uint64) *Point {
	p.Values.SetUIntegerField(name, value)
	return p
}

// GetStringField gets the string field value associated with the specified name.
// If the field is not present, returns nil.
func (p *Point) GetStringField(name string) *string {
	return p.Values.GetStringField(name)
}

// SetStringField adds or replaces a string field.
func (p *Point) SetStringField(name string, value string) *Point {
	p.Values.SetStringField(name, value)
	return p
}

// GetBooleanField gets the bool field value associated with the specified name.
// If the field is not present, returns nil.
func (p *Point) GetBooleanField(name string) *bool {
	return p.Values.GetBooleanField(name)
}

// SetBooleanField adds or replaces a bool field.
func (p *Point) SetBooleanField(name string, value bool) *Point {
	p.Values.SetBooleanField(name, value)
	return p
}

// GetField gets field of given name. Can be nil if field doesn't exist.
func (p *Point) GetField(name string) interface{} {
	return p.Values.GetField(name)
}

// SetField adds or replaces a field with an interface{} value.
func (p *Point) SetField(name string, value interface{}) *Point {
	p.Values.SetField(name, value)
	return p
}

// RemoveField removes a field with the specified name if it exists; otherwise, it does nothing.
func (p *Point) RemoveField(name string) *Point {
	p.Values.RemoveField(name)
	return p
}

// GetFieldNames gets an array of field names associated with this object.
func (p *Point) GetFieldNames() []string {
	return p.Values.GetFieldNames()
}

// HasFields checks if the point contains any fields.
func (p *Point) HasFields() bool {
	return p.Values.HasFields()
}

// Copy returns a copy of the Point.
func (p *Point) Copy() *Point {
	return &Point{
		Values: p.Values.Copy(),
	}
}

// MarshalBinary converts the Point to its binary representation in line protocol format.
//
// Parameters:
//   - precision: The precision to use for timestamp encoding in line protocol format.
//
// Returns:
//   - The binary representation of the Point in line protocol format.
//   - An error, if any.
func (p *Point) MarshalBinary(precision lineprotocol.Precision) ([]byte, error) {
	return p.MarshalBinaryWithDefaultTags(precision, nil)
}

// MarshalBinaryWithDefaultTags converts the Point to its binary representation in line protocol format with default tags.
//
// Parameters:
//   - precision: The precision to use for timestamp encoding in line protocol format.
//   - DefaultTags: Tags added to each point during writing. If a point already has a tag with the same key, it is left unchanged.
//
// Returns:
//   - The binary representation of the Point in line protocol format.
//   - An error, if any.
func (p *Point) MarshalBinaryWithDefaultTags(precision lineprotocol.Precision, defaultTags map[string]string) ([]byte, error) {
	var enc lineprotocol.Encoder
	enc.SetPrecision(precision)
	enc.StartLine(p.Values.MeasurementName)

	// N.B. Some customers have requested support for newline and tab chars in tag values (EAR 5476)
	// Though this is outside the lineprotocol specification, it was supported in
	// previous GO client versions.
	replacer := strings.NewReplacer(
		"\n", "\\n",
		"\t", "\\t",
	)

	// sort Tags
	tagKeys := make([]string, 0, len(p.Values.Tags)+len(defaultTags))
	for k := range p.Values.Tags {
		tagKeys = append(tagKeys, k)
	}
	for k := range defaultTags {
		tagKeys = append(tagKeys, k)
	}

	sort.Strings(tagKeys)
	lastKey := ""
	// ensure empty string key is written too
	if len(tagKeys) > 0 && tagKeys[0] == "" {
		lastKey = "_"
	}
	for _, tagKey := range tagKeys {
		if lastKey == tagKey {
			continue
		}
		lastKey = tagKey

		// N.B. Some customers have requested support for newline and tab chars in tag values (EAR 5476)
		if value, ok := p.Values.Tags[tagKey]; ok {
			enc.AddTag(tagKey, replacer.Replace(value))
		} else {
			enc.AddTag(tagKey, replacer.Replace(defaultTags[tagKey]))
		}
	}

	// sort Fields
	fieldKeys := make([]string, 0, len(p.Values.Fields))
	for k := range p.Values.Fields {
		fieldKeys = append(fieldKeys, k)
	}
	sort.Strings(fieldKeys)
	converter := p.fieldConverter
	if converter == nil {
		converter = convertField
	}
	for _, fieldKey := range fieldKeys {
		fieldValue := converter(p.Values.Fields[fieldKey])
		value, ok := lineprotocol.NewValue(fieldValue)
		if !ok {
			return nil, fmt.Errorf("invalid value for field %s: %v", fieldKey, fieldValue)
		}
		enc.AddField(fieldKey, value)
	}

	enc.EndLine(p.Values.Timestamp)
	if err := enc.Err(); err != nil {
		return nil, fmt.Errorf("encoding error: %w", err)
	}
	return enc.Bytes(), nil
}

// WithFieldConverter sets a custom field converter function for transforming field values when used.
func (p *Point) WithFieldConverter(converter func(interface{}) interface{}) {
	p.fieldConverter = converter
}

// convertField converts any primitive type to types supported by line protocol
func convertField(v interface{}) interface{} {
	switch v := v.(type) {
	case bool, int64, uint64, string, float64:
		return v
	case int:
		return int64(v)
	case uint:
		return uint64(v)
	case []byte:
		return string(v)
	case int32:
		return int64(v)
	case int16:
		return int64(v)
	case int8:
		return int64(v)
	case uint32:
		return uint64(v)
	case uint16:
		return uint64(v)
	case uint8:
		return uint64(v)
	case float32:
		return float64(v)
	case time.Time:
		return v.Format(time.RFC3339Nano)
	case time.Duration:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}
