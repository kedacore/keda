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
	"encoding"
	"fmt"
	"github.com/ClickHouse/ch-go/proto"
	"reflect"

	"github.com/ClickHouse/clickhouse-go/v2/lib/binary"
)

type String struct {
	name string
	col  proto.ColStr
}

func (col *String) Reset() {
	col.col.Reset()
}

func (col String) Name() string {
	return col.name
}

func (String) Type() Type {
	return "String"
}

func (String) ScanType() reflect.Type {
	return scanTypeString
}

func (col *String) Rows() int {
	return col.col.Rows()
}

func (col *String) Row(i int, ptr bool) any {
	val := col.col.Row(i)
	if ptr {
		return &val
	}
	return val
}

func (col *String) ScanRow(dest any, row int) error {
	val := col.Row(row, false).(string)
	switch d := dest.(type) {
	case *string:
		*d = val
	case **string:
		*d = new(string)
		**d = val
	case *sql.NullString:
		return d.Scan(val)
	case encoding.BinaryUnmarshaler:
		return d.UnmarshalBinary(binary.Str2Bytes(val, len(val)))
	default:
		if scan, ok := dest.(sql.Scanner); ok {
			return scan.Scan(val)
		}
		return &ColumnConverterError{
			Op:   "ScanRow",
			To:   fmt.Sprintf("%T", dest),
			From: "String",
		}
	}
	return nil
}

func (col *String) AppendRow(v any) error {
	switch v := v.(type) {
	case string:
		col.col.Append(v)
	case *string:
		switch {
		case v != nil:
			col.col.Append(*v)
		default:
			col.col.Append("")
		}
	case sql.NullString:
		switch v.Valid {
		case true:
			col.col.Append(v.String)
		default:
			col.col.Append("")
		}
	case *sql.NullString:
		switch v.Valid {
		case true:
			col.col.Append(v.String)
		default:
			col.col.Append("")
		}
	case []byte:
		col.col.AppendBytes(v)
	case nil:
		col.col.Append("")
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return &ColumnConverterError{
					Op:   "AppendRow",
					To:   "String",
					From: fmt.Sprintf("%T", v),
					Hint: "could not get driver.Valuer value",
				}
			}
			return col.AppendRow(val)
		}

		if s, ok := v.(fmt.Stringer); ok {
			return col.AppendRow(s.String())
		}

		return &ColumnConverterError{
			Op:   "AppendRow",
			To:   "String",
			From: fmt.Sprintf("%T", v),
		}
	}
	return nil
}

func (col *String) Append(v any) (nulls []uint8, err error) {
	switch v := v.(type) {
	case []string:
		col.col.AppendArr(v)
		nulls = make([]uint8, len(v))
	case []*string:
		nulls = make([]uint8, len(v))
		for i := range v {
			switch {
			case v[i] != nil:
				col.col.Append(*v[i])
			default:
				col.col.Append("")
				nulls[i] = 1
			}
		}
	case []sql.NullString:
		nulls = make([]uint8, len(v))
		for i := range v {
			col.AppendRow(v[i])
		}
	case []*sql.NullString:
		nulls = make([]uint8, len(v))
		for i := range v {
			if v[i] == nil {
				nulls[i] = 1
			}
			col.AppendRow(v[i])
		}
	case [][]byte:
		nulls = make([]uint8, len(v))
		for i := range v {
			col.col.Append(string(v[i]))
		}
	default:

		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return nil, &ColumnConverterError{
					Op:   "Append",
					To:   "String",
					From: fmt.Sprintf("%T", v),
					Hint: "could not get driver.Valuer value",
				}
			}
			return col.Append(val)
		}
		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   "String",
			From: fmt.Sprintf("%T", v),
		}
	}
	return
}

func (col *String) Decode(reader *proto.Reader, rows int) error {
	return col.col.DecodeColumn(reader, rows)
}

func (col *String) Encode(buffer *proto.Buffer) {
	col.col.EncodeColumn(buffer)
}

var _ Interface = (*String)(nil)
