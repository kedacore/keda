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

	"github.com/google/uuid"
)

type UUID struct {
	col  proto.ColUUID
	name string
}

func (col *UUID) Reset() {
	col.col.Reset()
}

func (col *UUID) Name() string {
	return col.name
}

func (col *UUID) Type() Type {
	return "UUID"
}

func (col *UUID) ScanType() reflect.Type {
	return scanTypeUUID
}

func (col *UUID) Rows() int {
	return col.col.Rows()
}

func (col *UUID) Row(i int, ptr bool) any {
	value := col.row(i)
	if ptr {
		return &value
	}
	return value
}

func (col *UUID) ScanRow(dest any, row int) error {
	switch d := dest.(type) {
	case *string:
		*d = col.row(row).String()
	case **string:
		*d = new(string)
		**d = col.row(row).String()
	case *uuid.UUID:
		*d = col.row(row)
	case **uuid.UUID:
		*d = new(uuid.UUID)
		**d = col.row(row)
	default:
		if scan, ok := dest.(sql.Scanner); ok {
			return scan.Scan(col.row(row).String())
		}
		return &ColumnConverterError{
			Op:   "ScanRow",
			To:   fmt.Sprintf("%T", dest),
			From: "UUID",
			Hint: fmt.Sprintf("try using *%s", col.ScanType()),
		}
	}
	return nil
}

func (col *UUID) Append(v any) (nulls []uint8, err error) {
	switch v := v.(type) {
	case []string:
		nulls = make([]uint8, len(v))
		for _, v := range v {
			var u uuid.UUID
			u, err = uuid.Parse(v)
			if err != nil {
				return
			}
			col.col.Append(u)
		}
	case []*string:
		nulls = make([]uint8, len(v))
		for i, v := range v {
			switch {
			case v != nil:
				var value uuid.UUID
				value, err = uuid.Parse(*v)
				if err != nil {
					return
				}
				col.col.Append(value)
			default:
				nulls[i] = 1
				col.col.Append(uuid.UUID{})
			}
		}
	case []uuid.UUID:
		nulls = make([]uint8, len(v))
		for _, v := range v {
			col.col.Append(v)
		}
	case []*uuid.UUID:
		nulls = make([]uint8, len(v))
		for i, v := range v {
			switch {
			case v != nil:
				col.col.Append(*v)
			default:
				nulls[i] = 1
				col.col.Append(uuid.UUID{})
			}
		}
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return nil, &ColumnConverterError{
					Op:   "Append",
					To:   "UUID",
					From: fmt.Sprintf("%T", v),
					Hint: "could not get driver.Valuer value",
				}
			}
			return col.Append(val)
		}

		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   "UUID",
			From: fmt.Sprintf("%T", v),
		}
	}
	return
}

func (col *UUID) AppendRow(v any) error {
	switch v := v.(type) {
	case string:
		u, err := uuid.Parse(v)
		if err != nil {
			return err
		}
		col.col.Append(u)
	case *string:
		switch {
		case v != nil:
			value, err := uuid.Parse(*v)
			if err != nil {
				return err
			}
			col.col.Append(value)
		default:
			col.col.Append(uuid.UUID{})
		}
	case uuid.UUID:
		col.col.Append(v)
	case *uuid.UUID:
		switch {
		case v != nil:
			col.col.Append(*v)
		default:
			col.col.Append(uuid.UUID{})
		}
	case nil:
		col.col.Append(uuid.UUID{})
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return &ColumnConverterError{
					Op:   "AppendRow",
					To:   "UUID",
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
			To:   "UUID",
			From: fmt.Sprintf("%T", v),
		}
	}
	return nil
}

func (col *UUID) Decode(reader *proto.Reader, rows int) error {
	return col.col.DecodeColumn(reader, rows)
}

func (col *UUID) Encode(buffer *proto.Buffer) {
	col.col.EncodeColumn(buffer)
}

func (col *UUID) row(i int) (uuid uuid.UUID) {
	return col.col.Row(i)
}

var _ Interface = (*UUID)(nil)
