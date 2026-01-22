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

package ext

import (
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2/lib/column"
	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
	"strings"
)

func NewTable(name string, columns ...func(t *Table) error) (*Table, error) {
	table := &Table{
		name:  name,
		block: &proto.Block{},
	}
	for _, column := range columns {
		if err := column(table); err != nil {
			return nil, err
		}
	}
	return table, nil
}

type Table struct {
	name  string
	block *proto.Block
}

func (tbl *Table) Name() string {
	return tbl.name
}

func (tbl *Table) Structure() string {
	columnStructure := make([]string, 0, len(tbl.block.Columns))
	for _, c := range tbl.block.Columns {
		columnStructure = append(columnStructure, fmt.Sprintf("%v %v", c.Name(), c.Type()))
	}
	return strings.Join(columnStructure, ", ")
}

func (tbl *Table) Block() *proto.Block {
	return tbl.block
}

func (tbl *Table) Append(v ...any) error {
	return tbl.block.Append(v...)
}

func Column(name string, ct column.Type) func(t *Table) error {
	return func(tbl *Table) error {
		return tbl.block.AddColumn(name, ct)
	}
}
