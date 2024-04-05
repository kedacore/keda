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
	"strings"

	"github.com/apache/arrow/go/v13/arrow"
	"github.com/apache/arrow/go/v13/arrow/array"
	"github.com/apache/arrow/go/v13/arrow/flight"
)

// QueryIterator is a custom query iterator that encapsulates and simplifies the logic for
// the flight reader. It provides methods such as Next, Value, and Index to consume the flight reader,
// or users can use the underlying reader directly with the Raw method.
type QueryIterator struct {
	reader *flight.Reader
	// Current record
	record arrow.Record
	// Index of row of current object in current record
	indexInRecord int
	// Total index of current object
	i int64
	// Current object
	current map[string]interface{}
	// Done
	done bool
}

func newQueryIterator(reader *flight.Reader) *QueryIterator {
	return &QueryIterator{
		reader:        reader,
		record:        nil,
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
			i.done = true
			return false
		}
		i.record = i.reader.Record()
		i.indexInRecord = 0
	}

	schema := i.reader.Schema()
	obj := make(map[string]interface{}, len(i.record.Columns()))

	for ci, col := range i.record.Columns() {
		name := schema.Field(ci).Name
		value, err := getArrowValue(col, i.indexInRecord)

		if err != nil {
			panic(err)
		}
		obj[name] = value
	}

	i.current = obj

	return true
}

// AsPoints return data from InfluxDB IOx into PointValues structure.
func (i *QueryIterator) AsPoints() *PointValues {
	readerSchema := i.reader.Schema()
	p := NewPointValues("")

	for ci, col := range i.record.Columns() {
		schema := readerSchema.Field(ci)
		name := schema.Name
		value, err := getArrowValue(col, i.indexInRecord)
		if err != nil {
			panic(err)
		}
		if value == nil {
			continue
		}

		metadataType, hasMetadataType := schema.Metadata.GetValue("iox::column::type")

		if stringValue, isString := value.(string); ((name == "measurement") || (name == "iox::measurement")) && isString {
			p.SetMeasurement(stringValue)
			continue
		}

		if !hasMetadataType {
			if timestampValue, isTimestamp := value.(arrow.Timestamp); isTimestamp && name == "time" {
				p.SetTimestamp(timestampValue.ToTime(arrow.Nanosecond))
			} else {
				p.SetField(name, value)
			}
			continue
		}

		parts := strings.Split(metadataType, "::")
		_, _, valueType := parts[0], parts[1], parts[2]

		if valueType == "field" {
			p.SetField(name, value)
		} else if stringValue, isString := value.(string); isString && valueType == "tag" {
			p.SetTag(name, stringValue)
		} else if timestampValue, isTimestamp := value.(arrow.Timestamp); isTimestamp && valueType == "timestamp" {
			p.SetTimestamp(timestampValue.ToTime(arrow.Nanosecond))
		}
	}

	return p
}

// Value returns the current value from the flight reader as a map object.
// The map contains the fields and tags as key-value pairs.
//
// Returns:
//   - A map[string]interface{} object representing the current value.
func (i *QueryIterator) Value() map[string]interface{} {
	return i.current
}

// Index returns the current index of Value.
//
// Returns:
//   - The current index value.
func (i *QueryIterator) Index() interface{} {
	return i.i
}

// Done returns a boolean value indicating whether the iteration is complete or not.
//
// Returns:
//   - true if the iteration is complete, false otherwise.
func (i *QueryIterator) Done() bool {
	return i.done
}

// Raw returns the underlying flight.Reader associated with the QueryIterator.
// WARNING: It is imperative to use either the Raw method or the Value and Next functions, but not both at the same time,
// as it can lead to unpredictable behavior.
//
// Returns:
//   - The underlying flight.Reader.
func (i *QueryIterator) Raw() *flight.Reader {
	return i.reader
}

func getArrowValue(arrayNoType arrow.Array, i int) (interface{}, error) {
	if arrayNoType.IsNull(i) {
		return nil, nil
	}
	switch arrayNoType.DataType().ID() {
	case arrow.NULL:
		return nil, nil
	case arrow.BOOL:
		return arrayNoType.(*array.Boolean).Value(i), nil
	case arrow.UINT8:
		return arrayNoType.(*array.Uint8).Value(i), nil
	case arrow.INT8:
		return arrayNoType.(*array.Int8).Value(i), nil
	case arrow.UINT16:
		return arrayNoType.(*array.Uint16).Value(i), nil
	case arrow.INT16:
		return arrayNoType.(*array.Int16).Value(i), nil
	case arrow.UINT32:
		return arrayNoType.(*array.Uint32).Value(i), nil
	case arrow.INT32:
		return arrayNoType.(*array.Int32).Value(i), nil
	case arrow.UINT64:
		return arrayNoType.(*array.Uint64).Value(i), nil
	case arrow.INT64:
		return arrayNoType.(*array.Int64).Value(i), nil
	case arrow.FLOAT16:
		return arrayNoType.(*array.Float16).Value(i), nil
	case arrow.FLOAT32:
		return arrayNoType.(*array.Float32).Value(i), nil
	case arrow.FLOAT64:
		return arrayNoType.(*array.Float64).Value(i), nil
	case arrow.STRING:
		return arrayNoType.(*array.String).Value(i), nil
	case arrow.BINARY:
		return arrayNoType.(*array.Binary).Value(i), nil
	case arrow.FIXED_SIZE_BINARY:
		return arrayNoType.(*array.FixedSizeBinary).Value(i), nil
	case arrow.DATE32:
		return arrayNoType.(*array.Date32).Value(i), nil
	case arrow.DATE64:
		return arrayNoType.(*array.Date64).Value(i), nil
	case arrow.TIMESTAMP:
		return arrayNoType.(*array.Timestamp).Value(i), nil
	case arrow.TIME32:
		return arrayNoType.(*array.Time32).Value(i), nil
	case arrow.TIME64:
		return arrayNoType.(*array.Time64).Value(i), nil
	case arrow.INTERVAL_MONTHS:
		return arrayNoType.(*array.MonthInterval).Value(i), nil
	case arrow.INTERVAL_DAY_TIME:
		return arrayNoType.(*array.DayTimeInterval).Value(i), nil
	case arrow.DECIMAL128:
		return arrayNoType.(*array.Decimal128).Value(i), nil
	case arrow.DECIMAL256:
		return arrayNoType.(*array.Decimal256).Value(i), nil
	// case arrow.LIST:
	// 	return arrayNoType.(*array.List).Value(i), nil
	// case arrow.STRUCT:
	// 	return arrayNoType.(*array.Struct).Value(i), nil
	// case arrow.SPARSE_UNION:
	// 	return arrayNoType.(*array.SparseUnion).Value(i), nil
	// case arrow.DENSE_UNION:
	// 	return arrayNoType.(*array.DenseUnion).Value(i), nil
	// case arrow.DICTIONARY:
	// 	return arrayNoType.(*array.Dictionary).Value(i), nil
	// case arrow.MAP:
	// 	return arrayNoType.(*array.Map).Value(i), nil
	// case arrow.EXTENSION:
	// 	return arrayNoType.(*array.ExtensionArrayBase).Value(i), nil
	// case arrow.FIXED_SIZE_LIST:
	// 	return arrayNoType.(*array.FixedSizeList).Value(i), nil
	case arrow.DURATION:
		return arrayNoType.(*array.Duration).Value(i), nil
	case arrow.LARGE_STRING:
		return arrayNoType.(*array.LargeString).Value(i), nil
	case arrow.LARGE_BINARY:
		return arrayNoType.(*array.LargeBinary).Value(i), nil
	// case arrow.LARGE_LIST:
	// 	return arrayNoType.(*array.LargeList).Value(i), nil
	case arrow.INTERVAL_MONTH_DAY_NANO:
		return arrayNoType.(*array.MonthDayNanoInterval).Value(i), nil
	// case arrow.RUN_END_ENCODED:
	// 	return arrayNoType.(*array.RunEndEncoded).Value(i), nil

	default:
		return nil, fmt.Errorf("not supported data type: %s", arrayNoType.DataType().ID().String())

	}
}
