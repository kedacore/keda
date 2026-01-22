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

package proto

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ClickHouse/clickhouse-go/v2/lib/column"
)

type Block struct {
	names    []string
	Packet   byte
	Columns  []column.Interface
	Timezone *time.Location
}

func (b *Block) Rows() int {
	if len(b.Columns) == 0 {
		return 0
	}
	return b.Columns[0].Rows()
}

func (b *Block) AddColumn(name string, ct column.Type) error {
	column, err := ct.Column(name, b.Timezone)
	if err != nil {
		return err
	}
	b.names, b.Columns = append(b.names, name), append(b.Columns, column)
	return nil
}

func (b *Block) Append(v ...any) (err error) {
	columns := b.Columns
	if len(columns) != len(v) {
		return &BlockError{
			Op:  "Append",
			Err: fmt.Errorf("clickhouse: expected %d arguments, got %d", len(columns), len(v)),
		}
	}
	for i, v := range v {
		if err := b.Columns[i].AppendRow(v); err != nil {
			return &BlockError{
				Op:         "AppendRow",
				Err:        err,
				ColumnName: columns[i].Name(),
			}
		}
	}
	return nil
}

func (b *Block) ColumnsNames() []string {
	return b.names
}

// SortColumns sorts our block according to the requested order - a slice of column names. Names must be identical in requested order and block.
func (b *Block) SortColumns(columns []string) error {
	if len(columns) == 0 {
		// no preferred sort order
		return nil
	}
	if len(columns) != len(b.Columns) {
		return fmt.Errorf("requested column order is incorrect length to sort block - expected %d, got %d", len(b.Columns), len(columns))
	}
	missing := difference(b.names, columns)
	if len(missing) > 0 {
		return fmt.Errorf("block cannot be sorted - missing columns in requested order: %v", missing)
	}
	lookup := make(map[string]int)
	for i, col := range columns {
		lookup[col] = i
	}
	// we assume both lists have the same
	sort.Slice(b.Columns, func(i, j int) bool {
		iRank, jRank := lookup[b.Columns[i].Name()], lookup[b.Columns[j].Name()]
		return iRank < jRank
	})
	sort.Slice(b.names, func(i, j int) bool {
		iRank, jRank := lookup[b.names[i]], lookup[b.names[j]]
		return iRank < jRank
	})
	return nil
}

func difference(a, b []string) []string {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}
	var diff []string
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}

func (b *Block) EncodeHeader(buffer *proto.Buffer, revision uint64) (err error) {
	if revision > 0 {
		encodeBlockInfo(buffer)
	}
	var rows int
	if len(b.Columns) != 0 {
		rows = b.Columns[0].Rows()
		for _, c := range b.Columns[1:] {
			cRows := c.Rows()
			if rows != cRows {
				return &BlockError{
					Op:  "Encode",
					Err: fmt.Errorf("mismatched len of columns - expected %d, received %d for col %s", rows, cRows, c.Name()),
				}
			}
		}
	}
	buffer.PutUVarInt(uint64(len(b.Columns)))
	buffer.PutUVarInt(uint64(rows))
	return nil
}

func (b *Block) EncodeColumn(buffer *proto.Buffer, revision uint64, i int) (err error) {
	if i >= 0 && i < len(b.Columns) {
		c := b.Columns[i]
		buffer.PutString(c.Name())
		buffer.PutString(string(c.Type()))

		if revision >= DBMS_MIN_REVISION_WITH_CUSTOM_SERIALIZATION {
			buffer.PutBool(false)
		}

		if serialize, ok := c.(column.CustomSerialization); ok {
			if err := serialize.WriteStatePrefix(buffer); err != nil {
				return &BlockError{
					Op:         "Encode",
					Err:        err,
					ColumnName: c.Name(),
				}
			}
		}
		c.Encode(buffer)
		return nil
	}
	return &BlockError{
		Op:  "Encode",
		Err: fmt.Errorf("%d is out of range of %d columns", i, len(b.Columns)),
	}
}

func (b *Block) Encode(buffer *proto.Buffer, revision uint64) (err error) {
	if err := b.EncodeHeader(buffer, revision); err != nil {
		return err
	}
	for i := range b.Columns {
		if err := b.EncodeColumn(buffer, revision, i); err != nil {
			return err
		}
	}
	return nil
}

func (b *Block) Decode(reader *proto.Reader, revision uint64) (err error) {
	if revision > 0 {
		if err := decodeBlockInfo(reader); err != nil {
			return err
		}
	}
	var (
		numRows uint64
		numCols uint64
	)
	if numCols, err = reader.UVarInt(); err != nil {
		return err
	}
	if numRows, err = reader.UVarInt(); err != nil {
		return err
	}
	if numRows > 1_000_000_000 {
		return &BlockError{
			Op:  "Decode",
			Err: errors.New("more then 1 billion rows in block - suspiciously big - preventing OOM"),
		}
	}
	b.Columns = make([]column.Interface, numCols, numCols)
	b.names = make([]string, numCols, numCols)
	for i := 0; i < int(numCols); i++ {
		var (
			columnName string
			columnType string
		)
		if columnName, err = reader.Str(); err != nil {
			return err
		}
		if columnType, err = reader.Str(); err != nil {
			return err
		}
		c, err := column.Type(columnType).Column(columnName, b.Timezone)
		if err != nil {
			return err
		}

		if revision >= DBMS_MIN_REVISION_WITH_CUSTOM_SERIALIZATION {
			hasCustom, err := reader.Bool()
			if err != nil {
				return err
			}
			if hasCustom {
				return &BlockError{
					Op:  "Decode",
					Err: errors.New(fmt.Sprintf("custom serialization for column %s. not supported by clickhouse-go driver", columnName)),
				}
			}
		}

		if numRows != 0 {
			if serialize, ok := c.(column.CustomSerialization); ok {
				if err := serialize.ReadStatePrefix(reader); err != nil {
					return &BlockError{
						Op:         "Decode",
						Err:        err,
						ColumnName: columnName,
					}
				}
			}
			if err := c.Decode(reader, int(numRows)); err != nil {
				return &BlockError{
					Op:         "Decode",
					Err:        err,
					ColumnName: columnName,
				}
			}
		}
		b.names[i] = columnName
		b.Columns[i] = c
	}
	return nil
}

func (b *Block) Reset() {
	for i := range b.Columns {
		b.Columns[i].Reset()
	}
}

func encodeBlockInfo(buffer *proto.Buffer) {
	buffer.PutUVarInt(1)
	buffer.PutBool(false)
	buffer.PutUVarInt(2)
	buffer.PutInt32(-1)
	buffer.PutUVarInt(0)
}

func decodeBlockInfo(reader *proto.Reader) error {
	{
		if _, err := reader.UVarInt(); err != nil {
			return err
		}
		if _, err := reader.Bool(); err != nil {
			return err
		}
		if _, err := reader.UVarInt(); err != nil {
			return err
		}
		if _, err := reader.Int32(); err != nil {
			return err
		}
	}
	if _, err := reader.UVarInt(); err != nil {
		return err
	}
	return nil
}

type BlockError struct {
	Op         string
	Err        error
	ColumnName string
}

func (e *BlockError) Error() string {
	switch err := e.Err.(type) {
	case *column.Error:
		return fmt.Sprintf("clickhouse [%s]: (%s %s) %s", e.Op, e.ColumnName, err.ColumnType, err.Err)
	}
	return fmt.Sprintf("clickhouse [%s]: %s %s", e.Op, e.ColumnName, e.Err)
}
