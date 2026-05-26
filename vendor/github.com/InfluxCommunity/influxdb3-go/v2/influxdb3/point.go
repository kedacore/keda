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
	"errors"
	"fmt"
	"maps"
	"math"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Point represents InfluxDB time series point, holding tags and fields
type Point struct {
	Values *PointValues

	// fieldConverter this converter function must return one of these types supported by InfluxDB int64, uint64, float64, bool, string, []byte.
	fieldConverter func(any) any
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
func NewPoint(measurement string, tags map[string]string, fields map[string]any, ts time.Time) *Point {
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
func (p *Point) GetField(name string) any {
	return p.Values.GetField(name)
}

// SetField adds or replaces a field with an any value.
func (p *Point) SetField(name string, value any) *Point {
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
//
// Notes:
//   - nil, NaN, +Inf, and -Inf field values are omitted from line protocol.
//   - If no fields remain after filtering, MarshalBinary returns an empty byte slice and nil error.
func (p *Point) MarshalBinary(precision Precision) ([]byte, error) {
	return p.marshalBinaryWithOptions(precision, nil, nil)
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
//
// Field filtering behavior is the same as MarshalBinary:
// nil, NaN, +Inf, and -Inf field values are omitted, and points with no
// remaining fields serialize to an empty byte slice.
func (p *Point) MarshalBinaryWithDefaultTags(precision Precision, defaultTags map[string]string) ([]byte, error) {
	return p.marshalBinaryWithOptions(precision, defaultTags, nil)
}

// WithFieldConverter sets a custom field converter function for transforming field values when used.
func (p *Point) WithFieldConverter(converter func(any) any) {
	p.fieldConverter = converter
}

func (p *Point) marshalBinaryWithOptions(precision Precision, defaultTags map[string]string, tagOrder []string) ([]byte, error) {
	if p == nil || p.Values == nil || p.Values.MeasurementName == "" {
		return nil, errors.New("encoding error: missing measurement")
	}

	var sb bytes.Buffer

	escapeKey(&sb, p.Values.MeasurementName, false)

	if err := p.appendTags(&sb, defaultTags, tagOrder); err != nil {
		return nil, err
	}

	appendedFields, err := p.appendFields(&sb)
	if err != nil {
		return nil, err
	}
	if !appendedFields {
		return []byte{}, nil
	}

	p.appendTime(&sb, precision)
	sb.WriteByte('\n')
	return sb.Bytes(), nil
}

func (p *Point) appendTags(sb *bytes.Buffer, defaultTags map[string]string, tagOrder []string) error {
	tagKeys, err := p.collectOrderedTagKeys(defaultTags, tagOrder)
	if err != nil {
		return err
	}

	for _, tagKey := range tagKeys {
		tagValue, ok := p.Values.Tags[tagKey]
		if !ok {
			tagValue = defaultTags[tagKey]
		}

		if tagValue == "" {
			continue
		}

		sb.WriteByte(',')
		escapeKey(sb, tagKey, true)
		sb.WriteByte('=')
		escapeKey(sb, tagValue, true)
	}

	sb.WriteByte(' ')
	return nil
}

func (p *Point) collectOrderedTagKeys(defaultTags map[string]string, tagOrder []string) ([]string, error) {
	tags := p.Values.Tags

	// Keep strict validation for point tags (explicit user point data),
	// while preserving backward-compatible behavior for default tags where
	// empty keys are ignored (not treated as hard errors).
	if _, exists := tags[""]; exists {
		return nil, fmt.Errorf("encoding error: invalid tag key %q", "")
	}

	tagKeySet := make(map[string]struct{}, len(tags)+len(defaultTags))
	for k := range tags {
		if strings.ContainsAny(k, "\n\r\t") {
			return nil, fmt.Errorf("encoding error: invalid tag key %q", k)
		}
		if k != "" {
			tagKeySet[k] = struct{}{}
		}
	}
	for k := range defaultTags {
		if strings.ContainsAny(k, "\n\r\t") {
			return nil, fmt.Errorf("encoding error: invalid tag key %q", k)
		}
		if k != "" {
			tagKeySet[k] = struct{}{}
		}
	}
	tagKeys := make([]string, 0, len(tagKeySet))
	if len(tagOrder) == 0 {
		tagKeys = slices.Collect(maps.Keys(tagKeySet))
		slices.Sort(tagKeys)
		return tagKeys, nil
	}

	seenOrderKeys := make(map[string]struct{}, len(tagOrder))
	for _, tagKey := range tagOrder {
		if tagKey == "" {
			continue
		}
		if _, seen := seenOrderKeys[tagKey]; seen {
			continue
		}
		seenOrderKeys[tagKey] = struct{}{}
		if _, exists := tagKeySet[tagKey]; !exists {
			continue
		}
		tagKeys = append(tagKeys, tagKey)
		delete(tagKeySet, tagKey)
	}

	remainingKeys := slices.Collect(maps.Keys(tagKeySet))
	slices.Sort(remainingKeys)
	return append(tagKeys, remainingKeys...), nil
}

func (p *Point) appendFields(sb *bytes.Buffer) (bool, error) {
	fieldKeys := make([]string, 0, len(p.Values.Fields))
	for k := range p.Values.Fields {
		fieldKeys = append(fieldKeys, k)
	}
	sort.Strings(fieldKeys)

	converter := p.fieldConverter
	if converter == nil {
		converter = convertField
	}

	appended := false
	for _, fieldKey := range fieldKeys {
		if fieldKey == "" || strings.ContainsAny(fieldKey, "\n\r\t") {
			return false, fmt.Errorf("encoding error: invalid field key %q", fieldKey)
		}

		fieldValue := converter(p.Values.Fields[fieldKey])
		if isNotDefined(fieldValue) {
			continue
		}

		if appended {
			sb.WriteByte(',')
		}
		escapeKey(sb, fieldKey, true)
		sb.WriteByte('=')

		if err := appendFieldValue(sb, fieldKey, fieldValue); err != nil {
			return false, err
		}
		appended = true
	}

	return appended, nil
}

func appendFieldValue(sb *bytes.Buffer, fieldKey string, fieldValue any) error {
	switch value := fieldValue.(type) {
	case float64:
		sb.WriteString(strconv.FormatFloat(value, 'g', -1, 64))
	case float32:
		sb.WriteString(strconv.FormatFloat(float64(value), 'g', -1, 32))
	case int:
		sb.WriteString(strconv.FormatInt(int64(value), 10))
		sb.WriteByte('i')
	case int8:
		sb.WriteString(strconv.FormatInt(int64(value), 10))
		sb.WriteByte('i')
	case int16:
		sb.WriteString(strconv.FormatInt(int64(value), 10))
		sb.WriteByte('i')
	case int32:
		sb.WriteString(strconv.FormatInt(int64(value), 10))
		sb.WriteByte('i')
	case int64:
		sb.WriteString(strconv.FormatInt(value, 10))
		sb.WriteByte('i')
	case uint:
		sb.WriteString(strconv.FormatUint(uint64(value), 10))
		sb.WriteByte('u')
	case uint8:
		sb.WriteString(strconv.FormatUint(uint64(value), 10))
		sb.WriteByte('u')
	case uint16:
		sb.WriteString(strconv.FormatUint(uint64(value), 10))
		sb.WriteByte('u')
	case uint32:
		sb.WriteString(strconv.FormatUint(uint64(value), 10))
		sb.WriteByte('u')
	case uint64:
		sb.WriteString(strconv.FormatUint(value, 10))
		sb.WriteByte('u')
	case bool:
		if value {
			sb.WriteString("true")
		} else {
			sb.WriteString("false")
		}
	case string:
		sb.WriteByte('"')
		escapeValue(sb, value)
		sb.WriteByte('"')
	case []byte:
		sb.WriteByte('"')
		escapeValue(sb, string(value))
		sb.WriteByte('"')
	default:
		return fmt.Errorf("invalid value for field %s: %v", fieldKey, fieldValue)
	}
	return nil
}

func (p *Point) appendTime(sb *bytes.Buffer, precision Precision) {
	timestamp := p.Values.Timestamp

	if timestamp.IsZero() {
		return
	}

	ts := timestamp.UnixNano()
	switch precision {
	case Nanosecond:
		// no-op
	case Microsecond:
		ts /= int64(time.Microsecond)
	case Millisecond:
		ts /= int64(time.Millisecond)
	case Second:
		ts /= int64(time.Second)
	default:
		panic(fmt.Errorf("unknown precision value %d", precision))
	}

	sb.WriteByte(' ')
	sb.WriteString(strconv.FormatInt(ts, 10))
}

func escapeKey(sb *bytes.Buffer, key string, escapeEqual bool) {
	for i := range len(key) {
		switch key[i] {
		case '\n':
			sb.WriteString("\\n")
			continue
		case '\r':
			sb.WriteString("\\r")
			continue
		case '\t':
			sb.WriteString("\\t")
			continue
		case ' ', ',':
			sb.WriteByte('\\')
		case '=':
			if escapeEqual {
				sb.WriteByte('\\')
			}
		}
		sb.WriteByte(key[i])
	}
}

func escapeValue(sb *bytes.Buffer, value string) {
	for i := range len(value) {
		switch value[i] {
		case '\n':
			sb.WriteString("\\n")
			continue
		case '\r':
			sb.WriteString("\\r")
			continue
		case '\t':
			sb.WriteString("\\t")
			continue
		case '\\', '"':
			sb.WriteByte('\\')
		}
		sb.WriteByte(value[i])
	}
}

func isNotDefined(value any) bool {
	if value == nil {
		return true
	}
	if v, ok := value.(float64); ok {
		return math.IsNaN(v) || math.IsInf(v, 0)
	}
	if v, ok := value.(float32); ok {
		return math.IsNaN(float64(v)) || math.IsInf(float64(v), 0)
	}
	return false
}

// convertField converts any primitive type to types supported by line protocol
func convertField(v any) any {
	if isNilLike(v) {
		return nil
	}

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

func isNilLike(v any) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}
