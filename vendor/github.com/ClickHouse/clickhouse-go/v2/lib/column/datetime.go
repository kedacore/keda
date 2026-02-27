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
	"reflect"
	"strings"
	"time"

	"github.com/ClickHouse/ch-go/proto"

	"github.com/ClickHouse/clickhouse-go/v2/lib/timezone"
)

var (
	minDateTime, _ = time.Parse("2006-01-02 15:04:05", "1970-01-01 00:00:00")
	maxDateTime, _ = time.Parse("2006-01-02 15:04:05", "2105-12-31 23:59:59")
)

const (
	defaultDateTimeFormatNoZone   = "2006-01-02 15:04:05"
	defaultDateTimeFormatWithZone = "2006-01-02 15:04:05 -07:00"
)

type DateTime struct {
	chType Type
	name   string
	col    proto.ColDateTime
}

func (col *DateTime) Reset() {
	col.col.Reset()
}

func (col *DateTime) Name() string {
	return col.name
}

func (col *DateTime) parse(t Type, tz *time.Location) (_ *DateTime, err error) {
	if col.chType = t; col.chType == "DateTime" {
		col.col.Location = tz
		return col, nil
	}
	var name = strings.TrimSuffix(strings.TrimPrefix(string(t), "DateTime('"), "')")
	timezone, err := timezone.Load(name)
	if err != nil {
		return nil, err
	}
	col.col.Location = timezone
	return col, nil
}

func (col *DateTime) Type() Type {
	return col.chType
}

func (col *DateTime) ScanType() reflect.Type {
	return scanTypeTime
}

func (col *DateTime) Rows() int {
	return col.col.Rows()
}

func (col *DateTime) Row(i int, ptr bool) any {
	value := col.row(i)
	if ptr {
		return &value
	}
	return value
}

func (col *DateTime) ScanRow(dest any, row int) error {
	switch d := dest.(type) {
	case *time.Time:
		*d = col.row(row)
	case **time.Time:
		*d = new(time.Time)
		**d = col.row(row)
	case *sql.NullTime:
		return d.Scan(col.row(row))
	default:
		if scan, ok := dest.(sql.Scanner); ok {
			return scan.Scan(col.row(row))
		}
		return &ColumnConverterError{
			Op:   "ScanRow",
			To:   fmt.Sprintf("%T", dest),
			From: "DateTime",
		}
	}
	return nil
}

func (col *DateTime) Append(v any) (nulls []uint8, err error) {
	switch v := v.(type) {
	// we assume int64 is in seconds and don't currently scale to the precision
	case []int64:
		nulls = make([]uint8, len(v))
		for i := range v {
			col.col.Append(time.Unix(v[i], 0))
		}
	case []*int64:
		nulls = make([]uint8, len(v))
		for i := range v {
			switch {
			case v[i] != nil:
				col.col.Append(time.Unix(*v[i], 0))
			default:
				col.col.Append(time.Time{})
				nulls[i] = 1
			}
		}
	case []time.Time:
		nulls = make([]uint8, len(v))
		for i := range v {
			col.col.Append(v[i])
		}

	case []*time.Time:
		nulls = make([]uint8, len(v))
		for i := range v {
			switch {
			case v[i] != nil:
				col.col.Append(*v[i])
			default:
				nulls[i] = 1
				col.col.Append(time.Time{})
			}
		}
	case []sql.NullTime:
		nulls = make([]uint8, len(v))
		for i := range v {
			col.AppendRow(v[i])
		}
	case []*sql.NullTime:
		nulls = make([]uint8, len(v))
		for i := range v {
			if v[i] == nil {
				nulls[i] = 1
			}
			col.AppendRow(v[i])
		}
	case []string:
		nulls = make([]uint8, len(v))
		for i := range v {
			value, err := col.parseDateTime(v[i])
			if err != nil {
				return nil, err
			}
			col.col.Append(value)
		}
	case []*string:
		nulls = make([]uint8, len(v))
		for i := range v {
			switch {
			case v[i] == nil || *v[i] == "":
				nulls[i] = 1
				col.col.Append(time.Time{})
			default:
				value, err := col.parseDateTime(*v[i])
				if err != nil {
					return nil, err
				}
				col.col.Append(value)
			}
		}
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return nil, &ColumnConverterError{
					Op:   "Append",
					To:   "DateTime",
					From: fmt.Sprintf("%T", v),
					Hint: "could not get driver.Valuer value",
				}
			}
			return col.Append(val)
		}
		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   "DateTime",
			From: fmt.Sprintf("%T", v),
		}
	}
	return
}

func (col *DateTime) AppendRow(v any) error {
	switch v := v.(type) {
	// we assume int64 is in seconds and don't currently scale to the precision
	case int64:
		col.col.Append(time.Unix(v, 0))
	case *int64:
		switch {
		case v != nil:
			col.col.Append(time.Unix(*v, 0))
		default:
			col.col.Append(time.Time{})
		}
	case time.Time:
		col.col.Append(v)
	case *time.Time:
		switch {
		case v != nil:
			col.col.Append(*v)
		default:
			col.col.Append(time.Time{})
		}
	case sql.NullTime:
		switch v.Valid {
		case true:
			col.col.Append(v.Time)
		default:
			col.col.Append(time.Time{})
		}
	case *sql.NullTime:
		switch v.Valid {
		case true:
			col.col.Append(v.Time)
		default:
			col.col.Append(time.Time{})
		}
	case nil:
		col.col.Append(time.Time{})
	case string:
		dateTime, err := col.parseDateTime(v)
		if err != nil {
			return err
		}
		col.col.Append(dateTime)
	case *string:
		if v == nil || *v == "" {
			col.col.Append(time.Time{})
		} else {
			dateTime, err := col.parseDateTime(*v)
			if err != nil {
				return err
			}
			col.col.Append(dateTime)
		}
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return &ColumnConverterError{
					Op:   "AppendRow",
					To:   "DateTime",
					From: fmt.Sprintf("%T", v),
					Hint: "could not get driver.Valuer value",
				}
			}
			return col.AppendRow(val)
		}
		s, ok := v.(fmt.Stringer)
		if ok {
			return col.AppendRow(s.String())
		}
		return &ColumnConverterError{
			Op:   "AppendRow",
			To:   "DateTime",
			From: fmt.Sprintf("%T", v),
		}
	}
	return nil
}

func (col *DateTime) Decode(reader *proto.Reader, rows int) error {
	return col.col.DecodeColumn(reader, rows)
}

func (col *DateTime) Encode(buffer *proto.Buffer) {
	col.col.EncodeColumn(buffer)
}

func (col *DateTime) row(i int) time.Time {
	v := col.col.Row(i)
	return v
}

func (col *DateTime) parseDateTime(value string) (tv time.Time, err error) {
	if tv, err = time.Parse(defaultDateTimeFormatWithZone, value); err == nil {
		return tv, nil
	}
	if tv, err = time.Parse(defaultDateTimeFormatNoZone, value); err == nil {
		return getTimeWithDifferentLocation(tv, time.Local), nil
	}
	return time.Time{}, err
}

var _ Interface = (*DateTime)(nil)
