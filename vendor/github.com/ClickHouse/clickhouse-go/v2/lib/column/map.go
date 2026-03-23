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
	"reflect"
	"strings"
	"time"

	"github.com/ClickHouse/ch-go/proto"
)

// https://github.com/ClickHouse/ClickHouse/blob/master/src/Columns/ColumnMap.cpp
type Map struct {
	keys     Interface
	values   Interface
	chType   Type
	offsets  Int64
	scanType reflect.Type
	name     string
}

type OrderedMap interface {
	Get(key any) (any, bool)
	Put(key any, value any)
	Keys() <-chan any
}

type MapIterator interface {
	Next() bool
	Key() any
	Value() any
}

type IterableOrderedMap interface {
	Put(key any, value any)
	Iterator() MapIterator
}

func (col *Map) Reset() {
	col.keys.Reset()
	col.values.Reset()
	col.offsets.Reset()
}

func (col *Map) Name() string {
	return col.name
}

func (col *Map) parse(t Type, tz *time.Location) (_ Interface, err error) {
	col.chType = t
	types := make([]string, 2, 2)
	typeParams := t.params()
	idx := strings.Index(typeParams, ",")
	if strings.HasPrefix(typeParams, "Enum") {
		idx = strings.Index(typeParams, "),") + 1
	}
	if idx > 0 {
		types[0] = typeParams[:idx]
		types[1] = typeParams[idx+1:]
	}
	if types[0] != "" && types[1] != "" {
		if col.keys, err = Type(strings.TrimSpace(types[0])).Column(col.name, tz); err != nil {
			return nil, err
		}
		if col.values, err = Type(strings.TrimSpace(types[1])).Column(col.name, tz); err != nil {
			return nil, err
		}
		col.scanType = reflect.MapOf(
			col.keys.ScanType(),
			col.values.ScanType(),
		)
		return col, nil
	}
	return nil, &UnsupportedColumnTypeError{
		t: t,
	}
}

func (col *Map) Type() Type {
	return col.chType
}

func (col *Map) ScanType() reflect.Type {
	return col.scanType
}

func (col *Map) Rows() int {
	return col.offsets.col.Rows()
}

func (col *Map) Row(i int, ptr bool) any {
	return col.row(i).Interface()
}

func (col *Map) ScanRow(dest any, i int) error {
	value := reflect.Indirect(reflect.ValueOf(dest))
	if value.Type() == col.scanType {
		value.Set(col.row(i))
		return nil
	}
	if om, ok := dest.(IterableOrderedMap); ok {
		keys, values := col.orderedRow(i)
		for i := range keys {
			om.Put(keys[i], values[i])
		}
		return nil
	}
	if om, ok := dest.(OrderedMap); ok {
		keys, values := col.orderedRow(i)
		for i := range keys {
			om.Put(keys[i], values[i])
		}
		return nil
	}
	return &ColumnConverterError{
		Op:   "ScanRow",
		To:   fmt.Sprintf("%T", dest),
		From: string(col.chType),
		Hint: fmt.Sprintf("try using %s", col.scanType),
	}
}

func (col *Map) Append(v any) (nulls []uint8, err error) {
	value := reflect.Indirect(reflect.ValueOf(v))
	if value.Kind() != reflect.Slice {
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return nil, &ColumnConverterError{
					Op:   "Append",
					To:   string(col.chType),
					From: fmt.Sprintf("%T", v),
					Hint: fmt.Sprintf("could not get driver.Valuer value, try using %s", col.scanType),
				}
			}
			return col.Append(val)
		}
		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   string(col.chType),
			From: fmt.Sprintf("%T", v),
			Hint: fmt.Sprintf("try using %s", col.scanType),
		}
	}
	for i := 0; i < value.Len(); i++ {
		if err := col.AppendRow(value.Index(i).Interface()); err != nil {
			return nil, err
		}
	}
	return
}

func (col *Map) AppendRow(v any) error {
	if v == nil {
		return &ColumnConverterError{
			Op:   "Append",
			To:   string(col.chType),
			From: fmt.Sprintf("%T", v),
			Hint: fmt.Sprintf("try using %s", col.scanType),
		}
	}

	value := reflect.Indirect(reflect.ValueOf(v))
	if value.Type() == col.scanType {
		var (
			size int64
			iter = value.MapRange()
		)
		for iter.Next() {
			size++
			if err := col.keys.AppendRow(iter.Key().Interface()); err != nil {
				return err
			}
			if err := col.values.AppendRow(iter.Value().Interface()); err != nil {
				return err
			}
		}
		var prev int64
		if n := col.offsets.Rows(); n != 0 {
			prev = col.offsets.col.Row(n - 1)
		}
		col.offsets.col.Append(prev + size)
		return nil
	}

	if orderedMap, ok := v.(IterableOrderedMap); ok {
		var size int64
		iter := orderedMap.Iterator()
		for iter.Next() {
			key, value := iter.Key(), iter.Value()
			size++
			if err := col.keys.AppendRow(key); err != nil {
				return err
			}
			if err := col.values.AppendRow(value); err != nil {
				return err
			}
		}
		var prev int64
		if n := col.offsets.Rows(); n != 0 {
			prev = col.offsets.col.Row(n - 1)
		}
		col.offsets.col.Append(prev + size)
		return nil
	}

	if orderedMap, ok := v.(OrderedMap); ok {
		var size int64
		for key := range orderedMap.Keys() {
			value, ok := orderedMap.Get(key)
			if !ok {
				return fmt.Errorf("ordered map has key %v but no corresponding value", key)
			}
			size++
			if err := col.keys.AppendRow(key); err != nil {
				return err
			}
			if err := col.values.AppendRow(value); err != nil {
				return err
			}
		}
		var prev int64
		if n := col.offsets.Rows(); n != 0 {
			prev = col.offsets.col.Row(n - 1)
		}
		col.offsets.col.Append(prev + size)
		return nil
	}

	if valuer, ok := v.(driver.Valuer); ok {
		val, err := valuer.Value()
		if err != nil {
			return &ColumnConverterError{
				Op:   "AppendRow",
				To:   string(col.chType),
				From: fmt.Sprintf("%T", v),
				Hint: fmt.Sprintf("could not get driver.Valuer value, try using %s", col.scanType),
			}
		}
		return col.AppendRow(val)
	}

	return &ColumnConverterError{
		Op:   "AppendRow",
		To:   string(col.chType),
		From: fmt.Sprintf("%T", v),
		Hint: fmt.Sprintf("try using %s", col.scanType),
	}

}

func (col *Map) Decode(reader *proto.Reader, rows int) error {
	if err := col.offsets.col.DecodeColumn(reader, rows); err != nil {
		return err
	}
	if i := col.offsets.Rows(); i != 0 {
		size := int(col.offsets.col.Row(i - 1))
		if err := col.keys.Decode(reader, size); err != nil {
			return err
		}
		return col.values.Decode(reader, size)
	}
	return nil
}

func (col *Map) Encode(buffer *proto.Buffer) {
	col.offsets.col.EncodeColumn(buffer)
	col.keys.Encode(buffer)
	col.values.Encode(buffer)
}

func (col *Map) ReadStatePrefix(reader *proto.Reader) error {
	if serialize, ok := col.keys.(CustomSerialization); ok {
		if err := serialize.ReadStatePrefix(reader); err != nil {
			return err
		}
	}
	if serialize, ok := col.values.(CustomSerialization); ok {
		if err := serialize.ReadStatePrefix(reader); err != nil {
			return err
		}
	}
	return nil
}

func (col *Map) WriteStatePrefix(encoder *proto.Buffer) error {
	if serialize, ok := col.keys.(CustomSerialization); ok {
		if err := serialize.WriteStatePrefix(encoder); err != nil {
			return err
		}
	}
	if serialize, ok := col.values.(CustomSerialization); ok {
		if err := serialize.WriteStatePrefix(encoder); err != nil {
			return err
		}
	}
	return nil
}

func (col *Map) row(n int) reflect.Value {
	var (
		prev  int64
		value = reflect.MakeMap(col.scanType)
	)
	if n != 0 {
		prev = col.offsets.col.Row(n - 1)
	}
	var (
		size = int(col.offsets.col.Row(n) - prev)
		from = int(prev)
	)
	for next := 0; next < size; next++ {
		value.SetMapIndex(
			reflect.ValueOf(col.keys.Row(from+next, false)),
			reflect.ValueOf(col.values.Row(from+next, false)),
		)
	}
	return value
}

func (col *Map) orderedRow(n int) ([]any, []any) {
	var prev int64
	if n != 0 {
		prev = col.offsets.col.Row(n - 1)
	}
	var (
		size = int(col.offsets.col.Row(n) - prev)
		from = int(prev)
	)
	keys := make([]any, size)
	values := make([]any, size)
	for next := 0; next < size; next++ {
		keys[next] = col.keys.Row(from+next, false)
		values[next] = col.values.Row(from+next, false)
	}
	return keys, values
}

var (
	_ Interface           = (*Map)(nil)
	_ CustomSerialization = (*Map)(nil)
)
