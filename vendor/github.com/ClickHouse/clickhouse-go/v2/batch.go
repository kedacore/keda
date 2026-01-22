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
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

var normalizeInsertQueryMatch = regexp.MustCompile(`(?i)(INSERT\s+INTO\s+([^(]+)(?:\s*\([^()]*(?:\([^()]*\)[^()]*)*\))?)(?:\s*VALUES)?`)
var truncateFormat = regexp.MustCompile(`(?i)\sFORMAT\s+[^\s]+`)
var truncateValues = regexp.MustCompile(`\sVALUES\s.*$`)
var extractInsertColumnsMatch = regexp.MustCompile(`(?si)INSERT INTO .+\s\((?P<Columns>.+)\)$`)

func extractNormalizedInsertQueryAndColumns(query string) (normalizedQuery string, tableName string, columns []string, err error) {
	query = truncateFormat.ReplaceAllString(query, "")
	query = truncateValues.ReplaceAllString(query, "")

	matches := normalizeInsertQueryMatch.FindStringSubmatch(query)
	if len(matches) == 0 {
		err = errors.Errorf("invalid INSERT query: %s", query)
		return
	}

	normalizedQuery = fmt.Sprintf("%s FORMAT Native", matches[1])
	tableName = strings.TrimSpace(matches[2])

	columns = make([]string, 0)
	matches = extractInsertColumnsMatch.FindStringSubmatch(matches[1])
	if len(matches) == 2 {
		columns = strings.Split(matches[1], ",")
		for i := range columns {
			// refers to https://clickhouse.com/docs/en/sql-reference/syntax#identifiers
			// we can use identifiers with double quotes or backticks, for example: "id", `id`, but not both, like `"id"`.
			columns[i] = strings.Trim(strings.Trim(strings.TrimSpace(columns[i]), "\""), "`")
		}
	}

	return
}
