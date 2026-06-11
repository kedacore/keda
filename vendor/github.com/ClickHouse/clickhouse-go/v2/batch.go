package clickhouse

import (
	"fmt"
	"regexp"
	"strings"
)

var normalizeInsertQueryMatch = regexp.MustCompile(`(?i)(?:(?:--[^\n]*|#![^\n]*|#\s[^\n]*)\n\s*)*(INSERT\s+INTO\s+([^(]+)(?:\s*\([^()]*(?:\([^()]*\)[^()]*)*\))?)(?:\s*VALUES)?`)
var truncateFormat = regexp.MustCompile(`(?i)\sFORMAT\s+[^\s]+`)
var truncateValues = regexp.MustCompile(`\sVALUES\s.*$`)
var extractInsertColumnsMatch = regexp.MustCompile(`(?si)INSERT INTO .+\s\((?P<Columns>.+)\)$`)

func extractNormalizedInsertQueryAndColumns(query string) (normalizedQuery string, tableName string, columns []string, err error) {
	query = truncateFormat.ReplaceAllString(query, "")
	query = truncateValues.ReplaceAllString(query, "")

	matches := normalizeInsertQueryMatch.FindStringSubmatch(query)
	if len(matches) == 0 {
		err = fmt.Errorf("invalid INSERT query: %s", query)
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
			columns[i] = strings.ReplaceAll(strings.Trim(strings.TrimSpace(columns[i]), "\""), "`", "")
		}
	}

	return
}
