// Licensed to ClickHouse, Inc. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. ClickHouse, Inc. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package column

import (
	"database/sql/driver"
	"fmt"
	"github.com/ClickHouse/ch-go/proto"
	"reflect"

	"github.com/paulmach/orb"
)

type MultiPolygon struct {
	set  *Array
	name string
}

func (col *MultiPolygon) Reset() {
	col.set.Reset()
}

func (col *MultiPolygon) Name() string {
	return col.name
}

func (col *MultiPolygon) Type() Type {
	return "MultiPolygon"
}

func (col *MultiPolygon) ScanType() reflect.Type {
	return scanTypeMultiPolygon
}

func (col *MultiPolygon) Rows() int {
	return col.set.Rows()
}

func (col *MultiPolygon) Row(i int, ptr bool) any {
	value := col.row(i)
	if ptr {
		return &value
	}
	return value
}

func (col *MultiPolygon) ScanRow(dest any, row int) error {
	switch d := dest.(type) {
	case *orb.MultiPolygon:
		*d = col.row(row)
	case **orb.MultiPolygon:
		*d = new(orb.MultiPolygon)
		**d = col.row(row)
	default:
		return &ColumnConverterError{
			Op:   "ScanRow",
			To:   fmt.Sprintf("%T", dest),
			From: "MultiPolygon",
			Hint: fmt.Sprintf("try using *%s", col.ScanType()),
		}
	}
	return nil
}

func (col *MultiPolygon) Append(v any) (nulls []uint8, err error) {
	switch v := v.(type) {
	case []orb.MultiPolygon:
		values := make([][]orb.Polygon, 0, len(v))
		for _, v := range v {
			values = append(values, v)
		}
		return col.set.Append(values)
	case []*orb.MultiPolygon:
		nulls = make([]uint8, len(v))
		values := make([][]orb.Polygon, 0, len(v))
		for i, v := range v {
			if v == nil {
				nulls[i] = 1
				values = append(values, orb.MultiPolygon{})
			} else {
				values = append(values, *v)
			}
		}
		return col.set.Append(values)
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return nil, &ColumnConverterError{
					Op:   "Append",
					To:   "MultiPolygon",
					From: fmt.Sprintf("%T", v),
					Hint: fmt.Sprintf("could not get driver.Valuer value, try using %s", col.Type()),
				}
			}
			return col.Append(val)
		}
		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   "MultiPolygon",
			From: fmt.Sprintf("%T", v),
		}
	}
}

func (col *MultiPolygon) AppendRow(v any) error {
	switch v := v.(type) {
	case orb.MultiPolygon:
		return col.set.AppendRow([]orb.Polygon(v))
	case *orb.MultiPolygon:
		return col.set.AppendRow([]orb.Polygon(*v))
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return &ColumnConverterError{
					Op:   "AppendRow",
					To:   "MultiPolygon",
					From: fmt.Sprintf("%T", v),
					Hint: fmt.Sprintf("could not get driver.Valuer value, try using %s", col.Type()),
				}
			}
			return col.AppendRow(val)
		}
		return &ColumnConverterError{
			Op:   "AppendRow",
			To:   "MultiPolygon",
			From: fmt.Sprintf("%T", v),
		}
	}
}

func (col *MultiPolygon) Decode(reader *proto.Reader, rows int) error {
	return col.set.Decode(reader, rows)
}

func (col *MultiPolygon) Encode(buffer *proto.Buffer) {
	col.set.Encode(buffer)
}

func (col *MultiPolygon) row(i int) orb.MultiPolygon {
	var value []orb.Polygon
	{
		col.set.ScanRow(&value, i)
	}
	return value
}

var _ Interface = (*MultiPolygon)(nil)
