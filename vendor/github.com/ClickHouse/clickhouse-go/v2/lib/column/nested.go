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
	"strings"
	"time"

	"github.com/ClickHouse/ch-go/proto"
)

type Nested struct {
	Interface
	name string
}

func (col *Nested) Reset() {
	col.Interface.Reset()
}

func asDDL(cols []namedCol) string {
	sCols := make([]string, len(cols), len(cols))
	for i := range cols {
		sCols[i] = fmt.Sprintf("%s %s", cols[i].name, cols[i].colType)
	}
	return strings.Join(sCols, ", ")
}

func (col *Nested) parse(t Type, tz *time.Location) (_ Interface, err error) {
	columns := fmt.Sprintf("Array(Tuple(%s))", asDDL(nestedColumns(t.params())))
	if col.Interface, err = (&Array{name: col.name}).parse(Type(columns), tz); err != nil {
		return nil, err
	}
	return col, nil
}

func nestedColumns(raw string) (columns []namedCol) {
	var (
		nBegin   int
		begin    int
		brackets int
	)
	for i, r := range raw + "," {
		switch r {
		case '(':
			brackets++
		case ')':
			brackets--
		case ' ':
			if brackets == 0 {
				begin = i + 1
			}
		case ',':
			if brackets == 0 {
				columns, begin = append(columns, namedCol{
					name:    strings.TrimSpace(raw[nBegin:begin]),
					colType: Type(raw[begin:i]),
				}), i+1
				nBegin = i + 1
				continue
			}
		}
	}
	for i, column := range columns {
		if strings.HasPrefix(string(column.colType), "Nested(") {
			columns[i] = namedCol{
				colType: Type(fmt.Sprintf("Array(Tuple(%s))", asDDL(nestedColumns(column.colType.params())))),
				name:    column.name,
			}
		}
	}
	return
}

func (col *Nested) ReadStatePrefix(reader *proto.Reader) error {
	if serialize, ok := col.Interface.(CustomSerialization); ok {
		if err := serialize.ReadStatePrefix(reader); err != nil {
			return err
		}
	}
	return nil
}

func (col *Nested) WriteStatePrefix(buffer *proto.Buffer) error {
	if serialize, ok := col.Interface.(CustomSerialization); ok {
		if err := serialize.WriteStatePrefix(buffer); err != nil {
			return err
		}
	}
	return nil
}

var _ Interface = (*Nested)(nil)
