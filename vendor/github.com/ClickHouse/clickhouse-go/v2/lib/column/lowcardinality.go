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
	"math"
	"reflect"
	"time"
)

const indexTypeMask = 0b11111111

const (
	keyUInt8  = 0
	keyUInt16 = 1
	keyUInt32 = 2
	keyUInt64 = 3
)

const (
	/// Need to read dictionary if it wasn't.
	needGlobalDictionaryBit = 1 << 8
	/// Need to read additional keys. Additional keys are stored before indexes as value N and N keys after them.
	hasAdditionalKeysBit = 1 << 9
	/// Need to update dictionary. It means that previous granule has different dictionary.
	needUpdateDictionary = 1 << 10

	updateAll = hasAdditionalKeysBit | needUpdateDictionary
)

const sharedDictionariesWithAdditionalKeys = 1

// https://github.com/ClickHouse/ClickHouse/blob/master/src/Columns/ColumnLowCardinality.cpp
// https://github.com/ClickHouse/clickhouse-cpp/blob/master/clickhouse/columns/lowcardinality.cpp
type LowCardinality struct {
	key      byte
	rows     int
	index    Interface
	chType   Type
	nullable bool

	keys8  UInt8
	keys16 UInt16
	keys32 UInt32
	keys64 UInt64

	append struct {
		keys  []int
		index map[any]int
	}
	name string
}

func (col *LowCardinality) Reset() {
	col.rows = 0
	col.index.Reset()
	col.keys8.Reset()
	col.keys16.Reset()
	col.keys32.Reset()
	col.keys64.Reset()
	col.append.index = make(map[any]int)
	col.append.keys = col.append.keys[:0]
}

func (col *LowCardinality) Name() string {
	return col.name
}

func (col *LowCardinality) parse(t Type, tz *time.Location) (_ *LowCardinality, err error) {
	col.chType = t
	col.append.index = make(map[any]int)
	if col.index, err = Type(t.params()).Column(col.name, tz); err != nil {
		return nil, err
	}
	if nullable, ok := col.index.(*Nullable); ok {
		col.nullable, nullable.enable = true, false
	}
	return col, nil
}

func (col *LowCardinality) Type() Type {
	return col.chType
}

func (col *LowCardinality) ScanType() reflect.Type {
	return col.index.ScanType()
}

func (col *LowCardinality) Rows() int {
	return col.rows
}

func (col *LowCardinality) Row(i int, ptr bool) any {
	idx := col.indexRowNum(i)
	if idx == 0 && col.nullable {
		return nil
	}
	return col.index.Row(idx, ptr)
}

func (col *LowCardinality) ScanRow(dest any, row int) error {
	idx := col.indexRowNum(row)
	if idx == 0 && col.nullable {
		return nil
	}
	return col.index.ScanRow(dest, idx)
}

func (col *LowCardinality) Append(v any) (nulls []uint8, err error) {
	value := reflect.Indirect(reflect.ValueOf(v))
	if value.Kind() != reflect.Slice {
		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   string(col.chType),
			From: fmt.Sprintf("%T", v),
		}
	}
	for i := 0; i < value.Len(); i++ {
		if err := col.AppendRow(value.Index(i).Interface()); err != nil {
			return nil, err
		}
	}
	return
}

func (col *LowCardinality) AppendRow(v any) error {
	col.rows++
	if col.index.Rows() == 0 { // init
		if col.index.AppendRow(nil); col.nullable {
			col.index.AppendRow(nil)
		}
	}
	// second check is unfortunate - but we could be passed a *type(nil) e.g. via LowCardinality(Nullable(String))
	if v == nil || (reflect.ValueOf(v).Kind() == reflect.Ptr && reflect.ValueOf(v).IsNil()) {
		col.append.keys = append(col.append.keys, 0)
		return nil
	}
	switch x := v.(type) {
	case time.Time:
		v = x.Truncate(time.Second)
	}
	if _, found := col.append.index[v]; !found {
		if err := col.index.AppendRow(v); err != nil {
			return err
		}
		col.append.index[v] = col.index.Rows() - 1
	}
	col.append.keys = append(col.append.keys, col.append.index[v])
	return nil
}

func (col *LowCardinality) Decode(reader *proto.Reader, rows int) error {
	if rows == 0 {
		return nil
	}
	indexSerializationType, err := reader.UInt64()
	if err != nil {
		return err
	}
	col.key = byte(indexSerializationType & indexTypeMask)
	switch col.key {
	case keyUInt8, keyUInt16, keyUInt32, keyUInt64:
	default:
		return &Error{
			ColumnType: "LowCardinality",
			Err:        errors.New("invalid index serialization version value"),
		}
	}
	switch {
	case indexSerializationType&needGlobalDictionaryBit == 1:
		return &Error{
			ColumnType: "LowCardinality",
			Err:        errors.New("global dictionary is not supported"),
		}
	case indexSerializationType&hasAdditionalKeysBit == 0:
		return &Error{
			ColumnType: "LowCardinality",
			Err:        errors.New("additional keys bit is missing"),
		}
	}
	indexRows, err := reader.Int64()
	if err != nil {
		return err
	}
	if err := col.index.Decode(reader, int(indexRows)); err != nil {
		return err
	}
	keysRows, err := reader.Int64()
	if err != nil {
		return err
	}
	col.rows = int(keysRows)
	return col.keys().Decode(reader, col.rows)
}

func (col *LowCardinality) Encode(buffer *proto.Buffer) {
	if col.rows == 0 {
		return
	}
	defer func() {
		col.append.keys, col.append.index = nil, nil
	}()
	ixLen := uint64(len(col.append.index))
	switch {
	case col.keys().Rows() > 0:
		// We already have keys, so this column is probably in a block directly decoded from the server, and we should
		// not reset them
	case ixLen < math.MaxUint8:
		col.key = keyUInt8
		for _, v := range col.append.keys {
			col.keys8.AppendRow(uint8(v))
		}
	case ixLen < math.MaxUint16:
		col.key = keyUInt16
		for _, v := range col.append.keys {
			col.keys16.AppendRow(uint16(v))
		}
	case ixLen < math.MaxUint32:
		col.key = keyUInt32
		for _, v := range col.append.keys {
			col.keys32.AppendRow(uint32(v))
		}
	default:
		col.key = keyUInt64
		for _, v := range col.append.keys {
			col.keys64.AppendRow(uint64(v))
		}
	}
	buffer.PutUInt64(updateAll | uint64(col.key))
	buffer.PutInt64(int64(col.index.Rows()))
	col.index.Encode(buffer)
	keys := col.keys()
	buffer.PutInt64(int64(keys.Rows()))
	keys.Encode(buffer)
}

func (col *LowCardinality) ReadStatePrefix(reader *proto.Reader) error {
	keyVersion, err := reader.UInt64()
	if err != nil {
		return err
	}
	if keyVersion != sharedDictionariesWithAdditionalKeys {
		return &Error{
			ColumnType: "LowCardinality",
			Err:        errors.New("invalid key serialization version value"),
		}
	}
	return nil
}

func (col *LowCardinality) WriteStatePrefix(buffer *proto.Buffer) error {
	buffer.PutUInt64(sharedDictionariesWithAdditionalKeys)
	return nil
}

func (col *LowCardinality) keys() Interface {
	switch col.key {
	case keyUInt8:
		return &col.keys8
	case keyUInt16:
		return &col.keys16
	case keyUInt32:
		return &col.keys32
	}
	return &col.keys64
}

func (col *LowCardinality) indexRowNum(row int) int {
	switch v := col.keys().Row(row, false).(type) {
	case uint8:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		return int(v)
	}
	return 0
}

var (
	_ Interface           = (*LowCardinality)(nil)
	_ CustomSerialization = (*LowCardinality)(nil)
)
