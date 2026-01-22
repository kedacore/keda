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

type Enum16 struct {
	iv     map[string]proto.Enum16
	vi     map[proto.Enum16]string
	chType Type
	col    proto.ColEnum16
	name   string
}

func (col *Enum16) Reset() {
	col.col.Reset()
}

func (col *Enum16) Name() string {
	return col.name
}

func (col *Enum16) Type() Type {
	return col.chType
}

func (col *Enum16) ScanType() reflect.Type {
	return scanTypeString
}

func (col *Enum16) Rows() int {
	return col.col.Rows()
}

func (col *Enum16) Row(i int, ptr bool) any {
	value := col.vi[col.col.Row(i)]
	if ptr {
		return &value
	}
	return value
}

func (col *Enum16) ScanRow(dest any, row int) error {
	value := col.col.Row(row)
	switch d := dest.(type) {
	case *string:
		*d = col.vi[value]
	case **string:
		*d = new(string)
		**d = col.vi[value]
	default:
		if scan, ok := dest.(sql.Scanner); ok {
			return scan.Scan(col.vi[value])
		}
		return &ColumnConverterError{
			Op:   "ScanRow",
			To:   fmt.Sprintf("%T", dest),
			From: "Enum16",
		}
	}
	return nil
}

func (col *Enum16) Append(v any) (nulls []uint8, err error) {
	switch v := v.(type) {
	case []int16:
		nulls = make([]uint8, len(v))
		for _, elem := range v {
			if err = col.AppendRow(elem); err != nil {
				return nil, err
			}
		}
	case []*int16:
		nulls = make([]uint8, len(v))
		for i, elem := range v {
			switch {
			case elem != nil:
				if err = col.AppendRow(elem); err != nil {
					return nil, err
				}
			default:
				col.col.Append(0)
				nulls[i] = 1
			}
		}
	case []int:
		nulls = make([]uint8, len(v))
		for _, elem := range v {
			if err = col.AppendRow(elem); err != nil {
				return nil, err
			}
		}
	case []*int:
		nulls = make([]uint8, len(v))
		for i, elem := range v {
			switch {
			case elem != nil:
				if err = col.AppendRow(elem); err != nil {
					return nil, err
				}
			default:
				col.col.Append(0)
				nulls[i] = 1
			}
		}
	case []string:
		nulls = make([]uint8, len(v))
		for _, elem := range v {
			v, ok := col.iv[elem]
			if !ok {
				return nil, &Error{
					Err:        fmt.Errorf("unknown element %q", elem),
					ColumnType: string(col.chType),
				}
			}
			col.col.Append(v)
		}
	case []*string:
		nulls = make([]uint8, len(v))
		for i, elem := range v {
			switch {
			case elem != nil:
				v, ok := col.iv[*elem]
				if !ok {
					return nil, &Error{
						Err:        fmt.Errorf("unknown element %q", *elem),
						ColumnType: string(col.chType),
					}
				}
				col.col.Append(v)
			default:
				col.col.Append(0)
				nulls[i] = 1
			}
		}
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return nil, &ColumnConverterError{
					Op:   "Append",
					To:   "Enum16",
					From: fmt.Sprintf("%T", v),
					Hint: "could not get driver.Valuer value",
				}
			}
			return col.Append(val)
		}
		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   "Enum16",
			From: fmt.Sprintf("%T", v),
		}
	}
	return
}

func (col *Enum16) AppendRow(elem any) error {
	switch elem := elem.(type) {
	case int16:
		return col.AppendRow(int(elem))
	case *int16:
		return col.AppendRow(int(*elem))
	case int:
		v := proto.Enum16(elem)
		_, ok := col.vi[v]
		if !ok {
			return &Error{
				Err:        fmt.Errorf("unknown element %v", elem),
				ColumnType: string(col.chType),
			}
		}
		col.col.Append(v)
	case *int:
		switch {
		case elem != nil:
			v := proto.Enum16(*elem)
			_, ok := col.vi[v]
			if !ok {
				return &Error{
					Err:        fmt.Errorf("unknown element %v", *elem),
					ColumnType: string(col.chType),
				}
			}
			col.col.Append(v)
		default:
			col.col.Append(0)
		}
	case string:
		v, ok := col.iv[elem]
		if !ok {
			return &Error{
				Err:        fmt.Errorf("unknown element %q", elem),
				ColumnType: string(col.chType),
			}
		}
		col.col.Append(v)
	case *string:
		switch {
		case elem != nil:
			v, ok := col.iv[*elem]
			if !ok {
				return &Error{
					Err:        fmt.Errorf("unknown element %q", *elem),
					ColumnType: string(col.chType),
				}
			}
			col.col.Append(v)
		default:
			col.col.Append(0)
		}
	case nil:
		col.col.Append(0)
	default:
		if valuer, ok := elem.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return &ColumnConverterError{
					Op:   "AppendRow",
					To:   "Enum16",
					From: fmt.Sprintf("%T", elem),
					Hint: "could not get driver.Valuer value",
				}
			}
			return col.AppendRow(val)
		}
		if s, ok := elem.(fmt.Stringer); ok {
			return col.AppendRow(s.String())
		} else {
			return &ColumnConverterError{
				Op:   "AppendRow",
				To:   "Enum16",
				From: fmt.Sprintf("%T", elem),
			}
		}
	}
	return nil
}

func (col *Enum16) Decode(reader *proto.Reader, rows int) error {
	return col.col.DecodeColumn(reader, rows)
}

func (col *Enum16) Encode(buffer *proto.Buffer) {
	col.col.EncodeColumn(buffer)
}

var _ Interface = (*Enum16)(nil)
