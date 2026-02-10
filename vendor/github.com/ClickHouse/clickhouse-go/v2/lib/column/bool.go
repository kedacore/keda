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
	"database/sql"
	"database/sql/driver"
	"fmt"
	"github.com/ClickHouse/ch-go/proto"
	"reflect"
)

type Bool struct {
	col  proto.ColBool
	name string
}

func (col *Bool) Reset() {
	col.col.Reset()
}

func (col *Bool) Name() string {
	return col.name
}

func (col *Bool) Type() Type {
	return "Bool"
}

func (col *Bool) ScanType() reflect.Type {
	return scanTypeBool
}

func (col *Bool) Rows() int {
	return col.col.Rows()
}

func (col *Bool) Row(i int, ptr bool) any {
	val := col.row(i)
	if ptr {
		return &val
	}
	return val
}

func (col *Bool) ScanRow(dest any, row int) error {
	switch d := dest.(type) {
	case *bool:
		*d = col.row(row)
	case **bool:
		*d = new(bool)
		**d = col.row(row)
	case sql.Scanner:
		return d.Scan(col.row(row))
	default:
		return &ColumnConverterError{
			Op:   "ScanRow",
			To:   fmt.Sprintf("%T", dest),
			From: "Bool",
		}
	}
	return nil
}

func (col *Bool) Append(v any) (nulls []uint8, err error) {
	switch v := v.(type) {
	case []bool:
		for _, v := range v {
			col.col.Append(v)
		}
	case []*bool:
		nulls = make([]uint8, len(v))
		for i, v := range v {
			var value bool
			switch {
			case v != nil:
				if *v {
					value = true
				}
			default:
				nulls[i] = 1
			}
			col.col.Append(value)
		}
	case []sql.NullBool:
		nulls = make([]uint8, len(v))
		for i := range v {
			col.Append(v[i])
		}
	case []*sql.NullBool:
		nulls = make([]uint8, len(v))
		for i := range v {
			if v[i] == nil {
				nulls[i] = 1
			}
			col.Append(v[i])
		}
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return nil, &ColumnConverterError{
					Op:   "Append",
					To:   "Bool",
					From: fmt.Sprintf("%T", v),
					Hint: "could not get driver.Valuer value",
				}
			}
			return col.Append(val)
		}
		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   "Bool",
			From: fmt.Sprintf("%T", v),
		}
	}
	return
}

func (col *Bool) AppendRow(v any) error {
	var value bool
	switch v := v.(type) {
	case bool:
		value = v
	case *bool:
		if v != nil {
			value = *v
		}
	case sql.NullBool:
		switch v.Valid {
		case true:
			value = v.Bool
		}
	case *sql.NullBool:
		switch v.Valid {
		case true:
			value = v.Bool
		}
	case nil:
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return &ColumnConverterError{
					Op:   "AppendRow",
					To:   "Bool",
					From: fmt.Sprintf("%T", v),
					Hint: "could not get driver.Valuer value",
				}
			}
			return col.AppendRow(val)
		}
		return &ColumnConverterError{
			Op:   "AppendRow",
			To:   "Bool",
			From: fmt.Sprintf("%T", v),
		}
	}
	col.col.Append(value)
	return nil
}

func (col *Bool) Decode(reader *proto.Reader, rows int) error {
	return col.col.DecodeColumn(reader, rows)
}

func (col *Bool) Encode(buffer *proto.Buffer) {
	col.col.EncodeColumn(buffer)
}

func (col *Bool) row(i int) bool {
	return col.col.Row(i)
}

var _ Interface = (*Bool)(nil)
