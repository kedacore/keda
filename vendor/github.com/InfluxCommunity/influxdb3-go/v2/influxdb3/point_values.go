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
	"fmt"
	"log/slog"
	"maps"
	"time"
)

// PointValues is representing InfluxDB time series point, holding tags and fields
type PointValues struct {
	MeasurementName string
	Tags            map[string]string
	Fields          map[string]any
	Timestamp       time.Time
}

// NewPointValues returns a new PointValues
func NewPointValues(measurementName string) *PointValues {
	return &PointValues{
		MeasurementName: measurementName,
		Tags:            make(map[string]string),
		Fields:          make(map[string]any),
	}
}

// GetMeasurement returns the measurement name
// It will return null when querying with SQL Query.
func (pv *PointValues) GetMeasurement() string {
	return pv.MeasurementName
}

// SetMeasurement sets the measurement name and returns the modified PointValues
func (pv *PointValues) SetMeasurement(measurementName string) *PointValues {
	pv.MeasurementName = measurementName
	return pv
}

// SetTimestampWithEpoch sets the timestamp using an int64 which represents start of epoch in nanoseconds and returns the modified PointDataValues
func (pv *PointValues) SetTimestampWithEpoch(timestamp int64) *PointValues {
	pv.Timestamp = time.Unix(0, timestamp)
	return pv
}

// SetTimestamp sets the timestamp using a time.Time and returns the modified PointValues
func (pv *PointValues) SetTimestamp(timestamp time.Time) *PointValues {
	pv.Timestamp = timestamp
	return pv
}

// SetTag sets a tag value and returns the modified PointValues
func (pv *PointValues) SetTag(name string, value string) *PointValues {
	if value == "" {
		slog.Debug(fmt.Sprintf("Empty tags has no effect, tag [%s], measurement [%s]", name, pv.MeasurementName))
	} else {
		pv.Tags[name] = value
	}
	return pv
}

// GetTag retrieves a tag value
func (pv *PointValues) GetTag(name string) (string, bool) {
	val, ok := pv.Tags[name]
	return val, ok
}

// RemoveTag Removes a tag with the specified name if it exists; otherwise, it does nothing.
func (pv *PointValues) RemoveTag(name string) *PointValues {
	delete(pv.Tags, name)
	return pv
}

// GetTagNames retrieves all tag names
func (pv *PointValues) GetTagNames() []string {
	tagNames := make([]string, 0, len(pv.Tags))
	for key := range pv.Tags {
		tagNames = append(tagNames, key)
	}
	return tagNames
}

// GetDoubleField gets the double field value associated with the specified name.
// If the field is not present, returns nil.
func (pv *PointValues) GetDoubleField(name string) *float64 {
	value, ok := pv.Fields[name]
	if !ok {
		return nil // field not found
	}

	if doubleValue, ok := value.(float64); ok {
		return &doubleValue
	}

	return nil // field is not a double
}

// SetDoubleField adds or replaces a double field.
func (pv *PointValues) SetDoubleField(name string, value float64) *PointValues {
	pv.Fields[name] = value
	return pv
}

// GetIntegerField gets the integer field value associated with the specified name.
// If the field is not present, returns nil.
func (pv *PointValues) GetIntegerField(name string) *int64 {
	value, ok := pv.Fields[name]
	if !ok {
		return nil // field not found
	}

	if intValue, ok := value.(int64); ok {
		return &intValue
	}

	return nil // field is not an int
}

// SetIntegerField adds or replaces an integer field.
func (pv *PointValues) SetIntegerField(name string, value int64) *PointValues {
	pv.Fields[name] = value
	return pv
}

// GetUIntegerField gets the uint field value associated with the specified name.
// If the field is not present, returns nil.
func (pv *PointValues) GetUIntegerField(name string) *uint64 {
	value, ok := pv.Fields[name]
	if !ok {
		return nil // field not found
	}

	if uintValue, ok := value.(uint64); ok {
		return &uintValue
	}

	return nil // field is not an int
}

// SetUIntegerField adds or replaces an unsigned integer field.
func (pv *PointValues) SetUIntegerField(name string, value uint64) *PointValues {
	pv.Fields[name] = value
	return pv
}

// GetStringField gets the string field value associated with the specified name.
// If the field is not present, returns nil.
func (pv *PointValues) GetStringField(name string) *string {
	value, ok := pv.Fields[name]
	if !ok {
		return nil // field not found
	}

	if strValue, ok := value.(string); ok {
		return &strValue
	}

	return nil // field is not a string
}

// SetStringField adds or replaces a string field.
func (pv *PointValues) SetStringField(name string, value string) *PointValues {
	pv.Fields[name] = value
	return pv
}

// GetBooleanField gets the bool field value associated with the specified name.
// If the field is not present, returns nil.
func (pv *PointValues) GetBooleanField(name string) *bool {
	value, ok := pv.Fields[name]
	if !ok {
		return nil // field not found
	}

	if boolValue, ok := value.(bool); ok {
		return &boolValue
	}

	return nil // field is not a bool
}

// SetBooleanField adds or replaces a bool field.
func (pv *PointValues) SetBooleanField(name string, value bool) *PointValues {
	pv.Fields[name] = value
	return pv
}

// GetField gets field of given name. Can be nil if field doesn't exist.
func (pv *PointValues) GetField(name string) any {
	value, exists := pv.Fields[name]
	if !exists {
		return nil
	}
	return value
}

// SetField adds or replaces a field with an any value.
func (pv *PointValues) SetField(name string, value any) *PointValues {
	pv.Fields[name] = value
	return pv
}

// RemoveField removes a field with the specified name if it exists; otherwise, it does nothing.
func (pv *PointValues) RemoveField(name string) *PointValues {
	delete(pv.Fields, name)
	return pv
}

// GetFieldNames gets an array of field names associated with this object.
func (pv *PointValues) GetFieldNames() []string {
	names := make([]string, 0, len(pv.Fields))
	for name := range pv.Fields {
		names = append(names, name)
	}
	return names
}

// HasFields checks if the point contains any fields.
func (pv *PointValues) HasFields() bool {
	return len(pv.Fields) > 0
}

// Copy returns a copy of the PointValues
func (pv *PointValues) Copy() *PointValues {
	newPDV := &PointValues{
		MeasurementName: pv.MeasurementName,
		Tags:            make(map[string]string, len(pv.Tags)),
		Fields:          make(map[string]any, len(pv.Fields)),
		Timestamp:       pv.Timestamp,
	}

	maps.Copy(newPDV.Tags, pv.Tags)

	maps.Copy(newPDV.Fields, pv.Fields)

	return newPDV
}

// AsPointWithMeasurement returns a Point with the specified measurement name
func (pv *PointValues) AsPointWithMeasurement(measurement string) (*Point, error) {
	pv.SetMeasurement(measurement)
	return pv.AsPoint()
}

// AsPoint returns a Point. If the PointValues does not have a measurement name, an error is returned.
func (pv *PointValues) AsPoint() (*Point, error) {
	return FromValues(pv)
}
