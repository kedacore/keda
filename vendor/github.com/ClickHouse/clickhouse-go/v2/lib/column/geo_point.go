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

type Point struct {
	name string
	col  proto.ColPoint
}

func (col *Point) Reset() {
	col.col.Reset()
}

func (col *Point) Name() string {
	return col.name
}

func (col *Point) Type() Type {
	return "Point"
}

func (col *Point) ScanType() reflect.Type {
	return scanTypePoint
}

func (col *Point) Rows() int {
	return col.col.Rows()
}

func (col *Point) Row(i int, ptr bool) any {
	value := col.row(i)
	if ptr {
		return &value
	}
	return value
}

func (col *Point) ScanRow(dest any, row int) error {
	switch d := dest.(type) {
	case *orb.Point:
		*d = col.row(row)
	case **orb.Point:
		*d = new(orb.Point)
		**d = col.row(row)
	default:
		return &ColumnConverterError{
			Op:   "ScanRow",
			To:   fmt.Sprintf("%T", dest),
			From: "Point",
			Hint: fmt.Sprintf("try using *%s", col.ScanType()),
		}
	}
	return nil
}

func (col *Point) Append(v any) (nulls []uint8, err error) {
	switch v := v.(type) {
	case []orb.Point:
		nulls = make([]uint8, len(v))
		for _, v := range v {
			col.col.Append(proto.Point{
				X: v.Lon(),
				Y: v.Lat(),
			})
		}
	case []*orb.Point:
		nulls = make([]uint8, len(v))
		for i, v := range v {
			if v == nil {
				nulls[i] = 1
				col.col.Append(proto.Point{})
			} else {
				col.col.Append(proto.Point{
					X: v.Lon(),
					Y: v.Lat(),
				})
			}
		}
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return nil, &ColumnConverterError{
					Op:   "Append",
					To:   "Point",
					From: fmt.Sprintf("%T", v),
					Hint: fmt.Sprintf("could not get driver.Valuer value, try using %s", col.Type()),
				}
			}
			return col.Append(val)
		}
		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   "Point",
			From: fmt.Sprintf("%T", v),
		}
	}
	return
}
func (col *Point) AppendRow(v any) error {
	switch v := v.(type) {
	case orb.Point:
		col.col.Append(proto.Point{
			X: v.Lon(),
			Y: v.Lat(),
		})
	case *orb.Point:
		col.col.Append(proto.Point{
			X: v.Lon(),
			Y: v.Lat(),
		})
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return &ColumnConverterError{
					Op:   "AppendRow",
					To:   "Point",
					From: fmt.Sprintf("%T", v),
					Hint: fmt.Sprintf("could not get driver.Valuer value, try using %s", col.Type()),
				}
			}
			return col.AppendRow(val)
		}
		return &ColumnConverterError{
			Op:   "AppendRow",
			To:   "Point",
			From: fmt.Sprintf("%T", v),
		}
	}
	return nil
}

func (col *Point) Decode(reader *proto.Reader, rows int) error {
	return col.col.DecodeColumn(reader, rows)
}

func (col *Point) Encode(buffer *proto.Buffer) {
	col.col.EncodeColumn(buffer)
}

func (col *Point) row(i int) orb.Point {
	p := col.col.Row(i)
	return orb.Point{
		p.X,
		p.Y,
	}
}

var _ Interface = (*Point)(nil)
