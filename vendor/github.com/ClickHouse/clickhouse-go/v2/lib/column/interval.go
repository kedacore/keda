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
	"errors"
	"fmt"
	"github.com/ClickHouse/ch-go/proto"
	"reflect"
	"strings"
)

type Interval struct {
	chType Type
	name   string
	col    proto.ColInt64
}

func (col *Interval) Reset() {
	col.col.Reset()
}

func (col *Interval) Name() string {
	return col.name
}

func (col *Interval) parse(t Type) (Interface, error) {
	switch col.chType = t; col.chType {
	case "IntervalSecond", "IntervalMinute", "IntervalHour", "IntervalDay", "IntervalWeek", "IntervalMonth", "IntervalYear":
		return col, nil
	}
	return nil, &UnsupportedColumnTypeError{
		t: t,
	}
}

func (col *Interval) Type() Type             { return col.chType }
func (col *Interval) ScanType() reflect.Type { return scanTypeString }
func (col *Interval) Rows() int              { return col.col.Rows() }
func (col *Interval) Row(i int, ptr bool) any {
	val := col.row(i)
	if ptr {
		return &val
	}
	return val
}
func (col *Interval) ScanRow(dest any, row int) error {
	switch d := dest.(type) {
	case *string:
		*d = col.row(row)
	case **string:
		*d = new(string)
		**d = col.row(row)
	default:
		return &ColumnConverterError{
			Op:   "ScanRow",
			To:   fmt.Sprintf("%T", dest),
			From: "Interval",
		}
	}
	return nil
}

func (Interval) Append(any) ([]uint8, error) {
	return nil, &Error{
		ColumnType: "Interval",
		Err:        errors.New("data type values can't be stored in tables"),
	}
}

func (Interval) AppendRow(any) error {
	return &Error{
		ColumnType: "Interval",
		Err:        errors.New("data type values can't be stored in tables"),
	}
}

func (col *Interval) Decode(reader *proto.Reader, rows int) error {
	return col.col.DecodeColumn(reader, rows)
}

func (Interval) Encode(buffer *proto.Buffer) {
}

func (col *Interval) row(i int) string {
	val := col.col.Row(i)
	v := fmt.Sprintf("%d %s", val, strings.TrimPrefix(string(col.chType), "Interval"))
	if val > 1 {
		v += "s"
	}
	return v
}

var _ Interface = (*Interval)(nil)
