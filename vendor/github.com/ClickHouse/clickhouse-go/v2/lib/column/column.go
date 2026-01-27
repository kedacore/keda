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
	"regexp"
	"strings"
)

// column names which match this must be escaped - see https://clickhouse.com/docs/en/sql-reference/syntax/#identifiers
var escapeColRegex = regexp.MustCompile("^[a-zA-Z_][0-9a-zA-Z_]*$")

// to escape and unescape special chars
var colEscape = strings.NewReplacer("`", "\\`", "\\", "\\\\")
var colUnEscape = strings.NewReplacer("\\`", "`", "\\\\", "\\")

type Type string

func (t Type) params() string {
	switch start, end := strings.Index(string(t), "("), strings.LastIndex(string(t), ")"); {
	case len(t) == 0, start <= 0, end <= 0, end < start:
		return ""
	default:
		return string(t[start+1 : end])
	}
}

type Error struct {
	ColumnType string
	Err        error
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.ColumnType, e.Err)
}

type ColumnConverterError struct {
	Op       string
	Hint     string
	From, To string
}

func (e *ColumnConverterError) Error() string {
	var hint string
	if len(e.Hint) != 0 {
		hint += ". " + e.Hint
	}
	return fmt.Sprintf("clickhouse [%s]: converting %s to %s is unsupported%s", e.Op, e.From, e.To, hint)
}

type UnsupportedColumnTypeError struct {
	t Type
}

func (e *UnsupportedColumnTypeError) Error() string {
	return fmt.Sprintf("clickhouse: unsupported column type %q", e.t)
}

type Interface interface {
	Name() string
	Type() Type
	Rows() int
	Row(i int, ptr bool) any
	ScanRow(dest any, row int) error
	Append(v any) (nulls []uint8, err error)
	AppendRow(v any) error
	Decode(reader *proto.Reader, rows int) error
	Encode(buffer *proto.Buffer)
	ScanType() reflect.Type
	Reset()
}

type CustomSerialization interface {
	ReadStatePrefix(*proto.Reader) error
	WriteStatePrefix(*proto.Buffer) error
}
