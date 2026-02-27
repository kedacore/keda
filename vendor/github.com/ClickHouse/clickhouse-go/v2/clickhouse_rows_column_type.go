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
	"reflect"

	"github.com/ClickHouse/clickhouse-go/v2/lib/column"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type columnType struct {
	name     string
	chType   string
	nullable bool
	scanType reflect.Type
}

func (c *columnType) Name() string {
	return c.name
}

func (c *columnType) Nullable() bool {
	return c.nullable
}

func (c *columnType) ScanType() reflect.Type {
	return c.scanType
}

func (c *columnType) DatabaseTypeName() string {
	return c.chType
}

func (r *rows) ColumnTypes() []driver.ColumnType {
	types := make([]driver.ColumnType, 0, len(r.columns))
	for i, c := range r.block.Columns {
		_, nullable := c.(*column.Nullable)
		types = append(types, &columnType{
			name:     r.columns[i],
			chType:   string(c.Type()),
			nullable: nullable,
			scanType: c.ScanType(),
		})
	}
	return types
}
