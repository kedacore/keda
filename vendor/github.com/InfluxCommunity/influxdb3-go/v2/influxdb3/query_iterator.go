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
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/flight"
)

type responseColumnType byte

const (
	responseColumnTypeUnknown responseColumnType = iota
	responseColumnTypeTimestamp
	responseColumnTypeField
	responseColumnTypeTag
)

// QueryIterator is a custom query iterator that encapsulates and simplifies the logic for
// the flight reader. It provides methods such as Next, Value, and Index to consume the flight reader,
// or users can use the underlying reader directly with the Raw method.
//
// The QueryIterator can return responses as one of the following data types:
//   - iterator.Value() returns map[string]any object representing the current row
//   - iterator.AsPoints() returns *PointValues object representing the current row
//   - iterator.Raw() returns the underlying *flight.Reader object
type QueryIterator struct {
	reader RecordReader
	// Current record
	record arrow.RecordBatch
	// The first err that might occur
	err error
	// Index of row of current object in current record
	indexInRecord int
	// Total index of current object
	i int64
	// Current object
	current map[string]any
	// Done
	done bool
}

// NewQueryIterator creates a new QueryIterator instance with the provided flight.Reader.
func NewQueryIterator(reader *flight.Reader) *QueryIterator {
	return NewQueryIteratorFromReader(reader)
}

// NewQueryIteratorFromReader creates a new QueryIterator instance with the provided RecordReader.
func NewQueryIteratorFromReader(reader RecordReader) *QueryIterator {
	return &QueryIterator{
		reader:        reader,
		record:        nil,
		err:           nil,
		indexInRecord: -1,
		i:             -1,
		current:       nil,
	}
}

// Next reads the next value of the flight reader and returns true if a value is present.
//
// Returns:
//   - true if a value is present, false otherwise.
func (i *QueryIterator) Next() bool {
	if i.done {
		return false
	}
	i.indexInRecord++
	i.i++
	for i.record == nil || i.indexInRecord >= int(i.record.NumRows()) {
		if !i.reader.Next() {
			if readError := i.reader.Err(); readError != nil && i.err == nil {
				i.err = i.reader.Err()
			}
			i.done = true
			return false
		}
		i.record = i.reader.RecordBatch()
		i.indexInRecord = 0
	}

	readerSchema := i.reader.Schema()
	obj := make(map[string]any, len(i.record.Columns()))

	for ci, col := range i.record.Columns() {
		field := readerSchema.Field(ci)
		name := field.Name
		value, _, err := getArrowValue(col, field, i.indexInRecord)
		if err != nil {
			panic(err)
		}
		obj[name] = value
	}

	i.current = obj

	return true
}

// AsPoints return data from InfluxDB v3 into PointValues structure.
func (i *QueryIterator) AsPoints() *PointValues {
	return rowToPointValue(i.record, i.indexInRecord)
}

func rowToPointValue(record arrow.RecordBatch, rowIndex int) *PointValues {
	readerSchema := record.Schema()
	p := NewPointValues("")

	for ci, col := range record.Columns() {
		field := readerSchema.Field(ci)
		name := field.Name
		value, columnType, err := getArrowValue(col, field, rowIndex)
		if err != nil {
			panic(err)
		}
		if value == nil {
			continue
		}

		if stringValue, isString := value.(string); ((name == "measurement") || (name == "iox::measurement")) && isString {
			p.SetMeasurement(stringValue)
			continue
		}

		switch {
		case columnType == responseColumnTypeUnknown:
			if timestampValue, isTimestamp := value.(arrow.Timestamp); isTimestamp && name == "time" {
				p.SetTimestamp(timestampValue.ToTime(arrow.Nanosecond))
			} else {
				p.SetField(name, value)
			}
		case columnType == responseColumnTypeField:
			p.SetField(name, value)
		case columnType == responseColumnTypeTag:
			p.SetTag(name, value.(string))
		case columnType == responseColumnTypeTimestamp:
			p.SetTimestamp(value.(time.Time))
		}
	}

	return p
}

// Value returns the current value from the flight reader as a map object.
// The map contains the fields and tags as key-value pairs.
//
// The current value types respect metadata provided by InfluxDB v3 metadata query response.
// Tags are mapped as a "string", timestamp as "time.Time", and fields as their respective types.
//
// Field are mapped to the following types:
//   - iox::column_type::field::integer: => int64
//   - iox::column_type::field::uinteger: => uint64
//   - iox::column_type::field::float: => float64
//   - iox::column_type::field::string: => string
//   - iox::column_type::field::boolean: => bool
//
// Returns:
//   - A map[string]any object representing the current value.
func (i *QueryIterator) Value() map[string]any {
	return i.current
}

// Index returns the current index of Value.
//
// Returns:
//   - The current index value.
func (i *QueryIterator) Index() any {
	return i.i
}

// Done returns a boolean value indicating whether the iteration is complete or not.
//
// Returns:
//   - true if the iteration is complete, false otherwise.
func (i *QueryIterator) Done() bool {
	return i.done
}

// Err returns the first err that might have occurred during iteration
//
// Returns:
//   - the err or nil if no err occurred
func (i *QueryIterator) Err() error { return i.err }

// Raw returns the underlying flight.Reader associated with the QueryIterator.
// WARNING: It is imperative to use either the Raw method or the Value and Next functions, but not both at the same time,
// as it can lead to unpredictable behavior.
//
// Returns:
//   - The underlying flight.Reader.
func (i *QueryIterator) Raw() *flight.Reader {
	if r, ok := i.reader.(*cancelingRecordReader); ok {
		return r.Reader()
	} else if f, ok := i.reader.(*flight.Reader); ok {
		return f
	}
	return nil
}

func getArrowValue(arrayNoType arrow.Array, field arrow.Field, i int) (any, responseColumnType, error) {
	var columnType = responseColumnTypeUnknown
	if arrayNoType.IsNull(i) {
		return nil, columnType, nil
	}
	typeExtractor := map[arrow.Type]func(arrow.Array, int) any{
		arrow.BOOL:                    func(arr arrow.Array, i int) any { return arr.(*array.Boolean).Value(i) },
		arrow.UINT8:                   func(arr arrow.Array, i int) any { return arr.(*array.Uint8).Value(i) },
		arrow.INT8:                    func(arr arrow.Array, i int) any { return arr.(*array.Int8).Value(i) },
		arrow.UINT16:                  func(arr arrow.Array, i int) any { return arr.(*array.Uint16).Value(i) },
		arrow.INT16:                   func(arr arrow.Array, i int) any { return arr.(*array.Int16).Value(i) },
		arrow.UINT32:                  func(arr arrow.Array, i int) any { return arr.(*array.Uint32).Value(i) },
		arrow.INT32:                   func(arr arrow.Array, i int) any { return arr.(*array.Int32).Value(i) },
		arrow.UINT64:                  func(arr arrow.Array, i int) any { return arr.(*array.Uint64).Value(i) },
		arrow.INT64:                   func(arr arrow.Array, i int) any { return arr.(*array.Int64).Value(i) },
		arrow.FLOAT16:                 func(arr arrow.Array, i int) any { return arr.(*array.Float16).Value(i) },
		arrow.FLOAT32:                 func(arr arrow.Array, i int) any { return arr.(*array.Float32).Value(i) },
		arrow.FLOAT64:                 func(arr arrow.Array, i int) any { return arr.(*array.Float64).Value(i) },
		arrow.STRING:                  func(arr arrow.Array, i int) any { return arr.(*array.String).Value(i) },
		arrow.BINARY:                  func(arr arrow.Array, i int) any { return arr.(*array.Binary).Value(i) },
		arrow.FIXED_SIZE_BINARY:       func(arr arrow.Array, i int) any { return arr.(*array.FixedSizeBinary).Value(i) },
		arrow.DATE32:                  func(arr arrow.Array, i int) any { return arr.(*array.Date32).Value(i) },
		arrow.DATE64:                  func(arr arrow.Array, i int) any { return arr.(*array.Date64).Value(i) },
		arrow.TIMESTAMP:               func(arr arrow.Array, i int) any { return arr.(*array.Timestamp).Value(i) },
		arrow.TIME32:                  func(arr arrow.Array, i int) any { return arr.(*array.Time32).Value(i) },
		arrow.TIME64:                  func(arr arrow.Array, i int) any { return arr.(*array.Time64).Value(i) },
		arrow.INTERVAL_MONTHS:         func(arr arrow.Array, i int) any { return arr.(*array.MonthInterval).Value(i) },
		arrow.INTERVAL_DAY_TIME:       func(arr arrow.Array, i int) any { return arr.(*array.DayTimeInterval).Value(i) },
		arrow.DECIMAL128:              func(arr arrow.Array, i int) any { return arr.(*array.Decimal128).Value(i) },
		arrow.DECIMAL256:              func(arr arrow.Array, i int) any { return arr.(*array.Decimal256).Value(i) },
		arrow.DURATION:                func(arr arrow.Array, i int) any { return arr.(*array.Duration).Value(i) },
		arrow.LARGE_STRING:            func(arr arrow.Array, i int) any { return arr.(*array.LargeString).Value(i) },
		arrow.LARGE_BINARY:            func(arr arrow.Array, i int) any { return arr.(*array.LargeBinary).Value(i) },
		arrow.INTERVAL_MONTH_DAY_NANO: func(arr arrow.Array, i int) any { return arr.(*array.MonthDayNanoInterval).Value(i) },
	}

	dataType := arrayNoType.DataType().ID()
	if extractor, exists := typeExtractor[dataType]; exists {
		value := extractor(arrayNoType, i)

		if metadata, hasMetadata := field.Metadata.GetValue("iox::column::type"); hasMetadata {
			value, columnType = getMetadataType(metadata, value, columnType)
		}
		return value, columnType, nil
	}

	return nil, columnType, fmt.Errorf("not supported data type: %s", dataType.String())
}

func getMetadataType(metadata string, value any, columnType responseColumnType) (any, responseColumnType) {
	switch metadata {
	case "iox::column_type::field::integer":
		if intValue, ok := value.(int64); ok {
			value = intValue
			columnType = responseColumnTypeField
		}
	case "iox::column_type::field::uinteger":
		if uintValue, ok := value.(uint64); ok {
			value = uintValue
			columnType = responseColumnTypeField
		}
	case "iox::column_type::field::float":
		if floatValue, ok := value.(float64); ok {
			value = floatValue
			columnType = responseColumnTypeField
		}
	case "iox::column_type::field::string":
		if stringValue, ok := value.(string); ok {
			value = stringValue
			columnType = responseColumnTypeField
		}
	case "iox::column_type::field::boolean":
		if boolValue, ok := value.(bool); ok {
			value = boolValue
			columnType = responseColumnTypeField
		}
	case "iox::column_type::tag":
		if stringValue, ok := value.(string); ok {
			value = stringValue
			columnType = responseColumnTypeTag
		}
	case "iox::column_type::timestamp":
		if timestampValue, ok := value.(arrow.Timestamp); ok {
			value = timestampValue.ToTime(arrow.Nanosecond)
			columnType = responseColumnTypeTimestamp
		}
	}
	return value, columnType
}
