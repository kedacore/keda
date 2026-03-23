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
	"github.com/ClickHouse/ch-go/proto"
	"reflect"
	"time"
)

type Nullable struct {
	base     Interface
	nulls    proto.ColUInt8
	enable   bool
	scanType reflect.Type
	name     string
}

func (col *Nullable) Reset() {
	col.base.Reset()
	col.nulls.Reset()
}

func (col *Nullable) Name() string {
	return col.name
}

func (col *Nullable) parse(t Type, tz *time.Location) (_ *Nullable, err error) {
	col.enable = true
	if col.base, err = Type(t.params()).Column(col.name, tz); err != nil {
		return nil, err
	}
	switch base := col.base.ScanType(); {
	case base == nil:
		col.scanType = reflect.TypeOf(nil)
	case base.Kind() == reflect.Ptr:
		col.scanType = base
	default:
		col.scanType = reflect.New(base).Type()
	}
	return col, nil
}

func (col *Nullable) Base() Interface {
	return col.base
}

func (col *Nullable) Type() Type {
	return "Nullable(" + col.base.Type() + ")"
}

func (col *Nullable) ScanType() reflect.Type {
	return col.scanType
}

func (col *Nullable) Rows() int {
	if !col.enable {
		return col.base.Rows()
	}
	return col.nulls.Rows()
}

func (col *Nullable) Row(i int, ptr bool) any {
	if col.enable {
		if col.nulls.Row(i) == 1 {
			return nil
		}
	}
	return col.base.Row(i, true)
}

func (col *Nullable) ScanRow(dest any, row int) error {
	if col.enable {
		switch col.nulls.Row(row) {
		case 1:
			switch v := dest.(type) {
			case **uint64:
				*v = nil
			case **int64:
				*v = nil
			case **uint32:
				*v = nil
			case **int32:
				*v = nil
			case **uint16:
				*v = nil
			case **int16:
				*v = nil
			case **uint8:
				*v = nil
			case **int8:
				*v = nil
			case **string:
				*v = nil
			case **float32:
				*v = nil
			case **float64:
				*v = nil
			case **time.Time:
				*v = nil
			}
			if scan, ok := dest.(sql.Scanner); ok {
				return scan.Scan(nil)
			}
			return nil
		}
	}
	return col.base.ScanRow(dest, row)
}

func (col *Nullable) Append(v any) ([]uint8, error) {
	nulls, err := col.base.Append(v)
	if err != nil {
		return nil, err
	}
	for i := range nulls {
		col.nulls.Append(nulls[i])
	}
	return nulls, nil
}

func (col *Nullable) AppendRow(v any) error {
	// Might receive double pointers like **String, because of how Nullable columns are read
	// Unpack because we can't write double pointers
	rv := reflect.ValueOf(v)
	if v != nil && rv.Kind() == reflect.Pointer && !rv.IsNil() && rv.Elem().Kind() == reflect.Pointer {
		v = rv.Elem().Interface()
		rv = reflect.ValueOf(v)
	}

	if v == nil || (rv.Kind() == reflect.Pointer && rv.IsNil()) {
		col.nulls.Append(1)
		// used to detect sql.Null* types
	} else if val, ok := v.(driver.Valuer); ok {
		val, err := val.Value()
		if err != nil {
			return err
		}
		if val == nil {
			col.nulls.Append(1)
		} else {
			col.nulls.Append(0)
		}
	} else {
		col.nulls.Append(0)
	}
	return col.base.AppendRow(v)
}

func (col *Nullable) Decode(reader *proto.Reader, rows int) error {
	if col.enable {
		if err := col.nulls.DecodeColumn(reader, rows); err != nil {
			return err
		}
	}
	if err := col.base.Decode(reader, rows); err != nil {
		return err
	}
	return nil
}

func (col *Nullable) Encode(buffer *proto.Buffer) {
	if col.enable {
		col.nulls.EncodeColumn(buffer)
	}
	col.base.Encode(buffer)
}

var _ Interface = (*Nullable)(nil)
