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
	"fmt"
	"github.com/ClickHouse/ch-go/proto"
	"reflect"
	"strings"
	"time"
)

type offset struct {
	values   UInt64
	scanType reflect.Type
}

type Array struct {
	depth    int
	chType   Type
	values   Interface
	offsets  []*offset
	scanType reflect.Type
	name     string
}

func (col *Array) Reset() {
	col.values.Reset()
	for i := range col.offsets {
		col.offsets[i].values.Reset()
	}
}

func (col *Array) Name() string {
	return col.name
}

func (col *Array) parse(t Type, tz *time.Location) (_ *Array, err error) {
	col.chType = t
	var typeStr = string(t)

parse:
	for {
		switch {
		case strings.HasPrefix(typeStr, "Array("):
			col.depth++
			typeStr = strings.TrimPrefix(typeStr, "Array(")
			typeStr = strings.TrimSuffix(typeStr, ")")
		default:
			break parse
		}
	}
	if col.depth != 0 {
		if col.values, err = Type(typeStr).Column(col.name, tz); err != nil {
			return nil, err
		}
		offsetScanTypes := make([]reflect.Type, 0, col.depth)
		col.offsets, col.scanType = make([]*offset, 0, col.depth), col.values.ScanType()
		for i := 0; i < col.depth; i++ {
			col.scanType = reflect.SliceOf(col.scanType)
			offsetScanTypes = append(offsetScanTypes, col.scanType)
		}
		for i := len(offsetScanTypes) - 1; i >= 0; i-- {
			col.offsets = append(col.offsets, &offset{
				scanType: offsetScanTypes[i],
			})
		}
		return col, nil
	}
	return nil, &UnsupportedColumnTypeError{
		t: t,
	}
}

func (col *Array) Base() Interface {
	return col.values
}

func (col *Array) Type() Type {
	return col.chType
}

func (col *Array) ScanType() reflect.Type {
	return col.scanType
}

func (col *Array) Rows() int {
	if len(col.offsets) != 0 {
		return col.offsets[0].values.Rows()
	}
	return 0
}

func (col *Array) Row(i int, ptr bool) any {
	value, err := col.scan(col.ScanType(), i)
	if err != nil {
		fmt.Println(err)
	}
	return value.Interface()
}

func (col *Array) Append(v any) (nulls []uint8, err error) {
	value := reflect.Indirect(reflect.ValueOf(v))
	if value.Kind() != reflect.Slice {
		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   string(col.chType),
			From: fmt.Sprintf("%T", v),
			Hint: "value must be a slice",
		}
	}
	for i := 0; i < value.Len(); i++ {
		if err := col.AppendRow(value.Index(i)); err != nil {
			return nil, err
		}
	}
	return
}

func (col *Array) AppendRow(v any) error {
	if col.depth == 1 {
		// try to use reflection-free method.
		return col.appendRowPlain(v)
	}
	return col.appendRowDefault(v)
}

func (col *Array) appendRowDefault(v any) error {
	var elem reflect.Value
	switch v := v.(type) {
	case reflect.Value:
		elem = reflect.Indirect(v)
	default:
		elem = reflect.Indirect(reflect.ValueOf(v))
	}
	if !elem.IsValid() {
		from := fmt.Sprintf("%T", v)
		if !elem.IsValid() {
			from = fmt.Sprintf("%v", v)
		}
		return &ColumnConverterError{
			Op:   "AppendRow",
			To:   string(col.chType),
			From: from,
			Hint: fmt.Sprintf("try using %s", col.scanType),
		}
	}
	return col.append(elem, 0)
}

func appendRowPlain[T any](col *Array, arr []T) error {
	col.appendOffset(0, uint64(len(arr)))
	for _, item := range arr {
		if err := col.values.AppendRow(item); err != nil {
			return err
		}
	}
	return nil
}

func appendNullableRowPlain[T any](col *Array, arr []*T) error {
	col.appendOffset(0, uint64(len(arr)))
	for _, item := range arr {
		var err error
		if item == nil {
			err = col.values.AppendRow(nil)
		} else {
			err = col.values.AppendRow(item)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (col *Array) append(elem reflect.Value, level int) error {
	if level < col.depth {
		switch elem.Kind() {
		// allows to traverse pointers to slices and slices cast to `any`
		case reflect.Interface, reflect.Ptr:
			if !elem.IsNil() {
				return col.append(elem.Elem(), level)
			}
		// reflect.Value.Len() & reflect.Value.Index() is called in `append` method which is only valid for
		// Slice, Array and String that make sense here.
		case reflect.Slice, reflect.Array, reflect.String:
			col.appendOffset(level, uint64(elem.Len()))
			for i := 0; i < elem.Len(); i++ {
				if err := col.append(elem.Index(i), level+1); err != nil {
					return err
				}
			}
			return nil
		}
		return &ColumnConverterError{
			Op:   "AppendRow",
			To:   "Array",
			From: fmt.Sprintf("%T", elem),
		}
	}
	if elem.Kind() == reflect.Ptr && elem.IsNil() {
		return col.values.AppendRow(nil)
	}
	return col.values.AppendRow(elem.Interface())
}

func (col *Array) appendOffset(level int, offset uint64) {
	if ln := col.offsets[level].values.Rows(); ln != 0 {
		offset += col.offsets[level].values.col.Row(ln - 1)
	}
	col.offsets[level].values.col.Append(offset)
}

func (col *Array) Decode(reader *proto.Reader, rows int) error {
	for _, offset := range col.offsets {
		if err := offset.values.col.DecodeColumn(reader, rows); err != nil {
			return err
		}
		switch {
		case offset.values.Rows() > 0:
			rows = int(offset.values.col.Row(offset.values.col.Rows() - 1))
		default:
			rows = 0
		}
	}
	return col.values.Decode(reader, rows)
}

func (col *Array) Encode(buffer *proto.Buffer) {
	for _, offset := range col.offsets {
		offset.values.col.EncodeColumn(buffer)
	}
	col.values.Encode(buffer)
}

func (col *Array) ReadStatePrefix(reader *proto.Reader) error {
	if serialize, ok := col.values.(CustomSerialization); ok {
		if err := serialize.ReadStatePrefix(reader); err != nil {
			return err
		}
	}
	return nil
}

func (col *Array) WriteStatePrefix(buffer *proto.Buffer) error {
	if serialize, ok := col.values.(CustomSerialization); ok {
		if err := serialize.WriteStatePrefix(buffer); err != nil {
			return err
		}
	}
	return nil
}

func (col *Array) ScanRow(dest any, row int) error {
	elem := reflect.Indirect(reflect.ValueOf(dest))
	value, err := col.scan(elem.Type(), row)
	if err != nil {
		return err
	}
	elem.Set(value)
	return nil
}

func (col *Array) scan(sliceType reflect.Type, row int) (reflect.Value, error) {
	switch col.values.(type) {
	case *Tuple:
		subSlice, err := col.scanSliceOfObjects(sliceType, row)
		if err != nil {
			return reflect.Value{}, err
		}
		return subSlice, nil
	default:
		subSlice, err := col.scanSlice(sliceType, row, 0)
		if err != nil {
			return reflect.Value{}, err
		}
		return subSlice, nil
	}
}

func (col *Array) scanSlice(sliceType reflect.Type, row int, level int) (reflect.Value, error) {
	// We could try and set - if it exceeds just return immediately
	offset := col.offsets[level]
	var (
		end   = offset.values.col.Row(row)
		start = uint64(0)
	)
	if row > 0 {
		start = offset.values.col.Row(row - 1)
	}
	base := offset.scanType.Elem()
	isPtr := base.Kind() == reflect.Ptr

	var rSlice reflect.Value
	switch sliceType.Kind() {
	case reflect.Interface:
		sliceType = offset.scanType
		rSlice = reflect.MakeSlice(sliceType, 0, int(end-start))
	case reflect.Slice:
		rSlice = reflect.MakeSlice(sliceType, 0, int(end-start))
	default:
		return reflect.Value{}, &Error{
			ColumnType: fmt.Sprint(sliceType.Kind()),
			Err:        fmt.Errorf("column %s - needs a slice or any", col.Name()),
		}
	}

	for i := start; i < end; i++ {
		var value reflect.Value
		var err error
		switch {
		case level == len(col.offsets)-1:
			switch dcol := col.values.(type) {
			case *Nested:
				//Array(Nested
				aCol := dcol.Interface.(*Array)
				value, err = aCol.scanSliceOfObjects(sliceType.Elem(), int(i))
				if err != nil {
					return reflect.Value{}, err
				}
			case *Array:
				//Array(Array
				value, err = dcol.scanSlice(sliceType.Elem(), int(i), 0)
				if err != nil {
					return reflect.Value{}, err
				}
			case *Tuple:
				// Array(Tuple possible outside JSON object cases e.g. if the user defines a  Array(Array( Tuple(String, Int64) ))
				value, err = dcol.scan(sliceType.Elem(), int(i))
				if err != nil {
					return reflect.Value{}, err
				}
			default:
				v := col.values.Row(int(i), isPtr)
				val := reflect.ValueOf(v)
				if v == nil {
					val = reflect.Zero(base)
				}
				if sliceType.Kind() == reflect.Interface {
					value = reflect.New(sliceType).Elem()
					if err := setJSONFieldValue(value, val); err != nil {
						return reflect.Value{}, err
					}
				} else {
					value = reflect.New(sliceType.Elem()).Elem()
					if err := setJSONFieldValue(value, val); err != nil {
						return reflect.Value{}, err
					}
				}
			}
		default:
			value, err = col.scanSlice(sliceType.Elem(), int(i), level+1)
			if err != nil {
				return reflect.Value{}, err
			}
		}
		rSlice = reflect.Append(rSlice, value)
	}
	return rSlice, nil
}

func (col *Array) scanSliceOfObjects(sliceType reflect.Type, row int) (reflect.Value, error) {
	if sliceType.Kind() == reflect.Interface {
		// catches any - Note this swallows custom interfaces to which maps couldn't conform
		subMap := make(map[string]any)
		return col.scanSliceOfMaps(reflect.SliceOf(reflect.TypeOf(subMap)), row)
	} else if sliceType.Kind() == reflect.Slice {
		// make a slice of the right type - we need this to be a slice of a type capable of taking an object as nested
		switch sliceType.Elem().Kind() {
		case reflect.Struct:
			return col.scanSliceOfStructs(sliceType, row)
		case reflect.Map:
			return col.scanSliceOfMaps(sliceType, row)
		case reflect.Slice:
			// tuples can be read as arrays
			return col.scanSlice(sliceType, row, 0)
		case reflect.Interface:
			// catches []any - Note this swallows custom interfaces to which maps could never conform
			subMap := make(map[string]any)
			return col.scanSliceOfMaps(reflect.SliceOf(reflect.TypeOf(subMap)), row)
		default:
			return reflect.Value{}, &Error{
				ColumnType: fmt.Sprint(sliceType.Elem().Kind()),
				Err:        fmt.Errorf("column %s - needs a slice of objects or an any", col.Name()),
			}
		}
	}
	return reflect.Value{}, &Error{
		ColumnType: fmt.Sprint(sliceType.Kind()),
		Err:        fmt.Errorf("column %s - needs a slice or any", col.Name()),
	}
}

// the following 2 functions can probably be refactored - the share alot of common code for structs and maps
func (col *Array) scanSliceOfMaps(sliceType reflect.Type, row int) (reflect.Value, error) {
	if sliceType.Kind() != reflect.Slice {
		return reflect.Value{}, &ColumnConverterError{
			Op:   "ScanRow",
			To:   sliceType.String(),
			From: string(col.Type()),
		}
	}
	tCol, ok := col.values.(*Tuple)
	if !ok {
		return reflect.Value{}, &Error{
			ColumnType: fmt.Sprint(col.values.Type()),
			Err:        fmt.Errorf("column %s - must be a tuple", col.Name()),
		}
	}
	// Array(Tuple so depth 1 for JSON
	offset := col.offsets[0]
	var (
		end   = offset.values.col.Row(row)
		start = uint64(0)
	)
	if row > 0 {
		start = offset.values.col.Row(row - 1)
	}
	if end-start > 0 {
		rSlice := reflect.MakeSlice(sliceType, 0, int(end-start))
		for i := start; i < end; i++ {
			sMap := reflect.MakeMap(sliceType.Elem())
			if err := tCol.scanMap(sMap, int(i)); err != nil {
				return reflect.Value{}, err
			}
			rSlice = reflect.Append(rSlice, sMap)
		}
		return rSlice, nil
	}
	return reflect.MakeSlice(sliceType, 0, 0), nil
}

func (col *Array) scanSliceOfStructs(sliceType reflect.Type, row int) (reflect.Value, error) {
	if sliceType.Kind() != reflect.Slice {
		return reflect.Value{}, &ColumnConverterError{
			Op:   "ScanRow",
			To:   sliceType.String(),
			From: string(col.Type()),
		}
	}
	tCol, ok := col.values.(*Tuple)
	if !ok {
		return reflect.Value{}, &Error{
			ColumnType: fmt.Sprint(col.values.Type()),
			Err:        fmt.Errorf("column %s - must be a tuple", col.Name()),
		}
	}
	// Array(Tuple so depth 1 for JSON
	offset := col.offsets[0]
	var (
		end   = offset.values.col.Row(row)
		start = uint64(0)
	)
	if row > 0 {
		start = offset.values.col.Row(row - 1)
	}
	if end-start > 0 {
		// create a slice of the type from the sliceType - if this might be any as its driven by the target datastructure
		rSlice := reflect.MakeSlice(sliceType, 0, int(end-start))
		for i := start; i < end; i++ {
			sStruct := reflect.New(sliceType.Elem()).Elem()
			if err := tCol.scanStruct(sStruct, int(i)); err != nil {
				return reflect.Value{}, err
			}
			rSlice = reflect.Append(rSlice, sStruct)
		}
		return rSlice, nil
	}
	return reflect.MakeSlice(sliceType, 0, 0), nil
}

var (
	_ Interface           = (*Array)(nil)
	_ CustomSerialization = (*Array)(nil)
)
