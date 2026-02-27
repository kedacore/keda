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

type Ring struct {
	set  *Array
	name string
}

func (col *Ring) Reset() {
	col.set.Reset()
}

func (col *Ring) Name() string {
	return col.name
}

func (col *Ring) Type() Type {
	return "Ring"
}

func (col *Ring) ScanType() reflect.Type {
	return scanTypeRing
}

func (col *Ring) Rows() int {
	return col.set.Rows()
}

func (col *Ring) Row(i int, ptr bool) any {
	value := col.row(i)
	if ptr {
		return &value
	}
	return value
}

func (col *Ring) ScanRow(dest any, row int) error {
	switch d := dest.(type) {
	case *orb.Ring:
		*d = col.row(row)
	case **orb.Ring:
		*d = new(orb.Ring)
		**d = col.row(row)
	default:
		return &ColumnConverterError{
			Op:   "ScanRow",
			To:   fmt.Sprintf("%T", dest),
			From: "Ring",
			Hint: fmt.Sprintf("try using *%s", col.ScanType()),
		}
	}
	return nil
}

func (col *Ring) Append(v any) (nulls []uint8, err error) {
	switch v := v.(type) {
	case []orb.Ring:
		values := make([][]orb.Point, 0, len(v))
		for _, v := range v {
			values = append(values, v)
		}
		return col.set.Append(values)
	case []*orb.Ring:
		nulls = make([]uint8, len(v))
		values := make([][]orb.Point, 0, len(v))
		for i, v := range v {
			if v == nil {
				nulls[i] = 1
				values = append(values, orb.Ring{})
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
					To:   "Ring",
					From: fmt.Sprintf("%T", v),
					Hint: fmt.Sprintf("could not get driver.Valuer value, try using %s", col.Type()),
				}
			}
			return col.Append(val)
		}
		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   "Ring",
			From: fmt.Sprintf("%T", v),
		}
	}
}

func (col *Ring) AppendRow(v any) error {
	switch v := v.(type) {
	case orb.Ring:
		return col.set.AppendRow([]orb.Point(v))
	case *orb.Ring:
		return col.set.AppendRow([]orb.Point(*v))
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return &ColumnConverterError{
					Op:   "AppendRow",
					To:   "Ring",
					From: fmt.Sprintf("%T", v),
					Hint: fmt.Sprintf("could not get driver.Valuer value, try using %s", col.Type()),
				}
			}
			return col.AppendRow(val)
		}
		return &ColumnConverterError{
			Op:   "AppendRow",
			To:   "Ring",
			From: fmt.Sprintf("%T", v),
		}
	}
}

func (col *Ring) Decode(reader *proto.Reader, rows int) error {
	return col.set.Decode(reader, rows)
}

func (col *Ring) Encode(buffer *proto.Buffer) {
	col.set.Encode(buffer)
}

func (col *Ring) row(i int) orb.Ring {
	var value []orb.Point
	{
		col.set.ScanRow(&value, i)
	}
	return value
}

var _ Interface = (*Ring)(nil)
