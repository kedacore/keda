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

package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

func (ch *clickhouse) Select(ctx context.Context, dest any, query string, args ...any) error {
	value := reflect.ValueOf(dest)
	if value.Kind() != reflect.Ptr {
		return &OpError{
			Op:  "Select",
			Err: errors.New("must pass a pointer, not a value, to Select destination"),
		}
	}
	if value.IsNil() {
		return &OpError{
			Op:  "Select",
			Err: errors.New("nil pointer passed to Select destination"),
		}
	}
	direct := reflect.Indirect(value)
	if direct.Kind() != reflect.Slice {
		return fmt.Errorf("must pass a slice to Select destination")
	}
	if direct.Len() != 0 {
		// dest should point to empty slice
		// to make select result correct
		direct.Set(reflect.MakeSlice(direct.Type(), 0, direct.Cap()))
	}
	var (
		base      = direct.Type().Elem()
		rows, err = ch.Query(ctx, query, args...)
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		elem := reflect.New(base)
		if err := rows.ScanStruct(elem.Interface()); err != nil {
			return err
		}
		direct.Set(reflect.Append(direct, elem.Elem()))
	}
	if err := rows.Close(); err != nil {
		return err
	}
	return rows.Err()
}

func scan(block *proto.Block, row int, dest ...any) error {
	columns := block.Columns
	if len(columns) != len(dest) {
		return &OpError{
			Op:  "Scan",
			Err: fmt.Errorf("expected %d destination arguments in Scan, not %d", len(columns), len(dest)),
		}
	}
	for i, d := range dest {
		if err := columns[i].ScanRow(d, row-1); err != nil {
			return &OpError{
				Err:        err,
				ColumnName: block.ColumnsNames()[i],
			}
		}
	}
	return nil
}
