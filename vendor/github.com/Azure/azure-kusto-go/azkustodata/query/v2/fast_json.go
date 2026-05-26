package v2

import (
	"bytes"
	"encoding/json"
	"github.com/Azure/azure-kusto-go/azkustodata/errors"
	"github.com/Azure/azure-kusto-go/azkustodata/query"
	"github.com/Azure/azure-kusto-go/azkustodata/types"
	"github.com/Azure/azure-kusto-go/azkustodata/value"
	"io"
)

func newDecoder(r io.Reader) *json.Decoder {
	dec := json.NewDecoder(r)
	// This option uses the json.Number type for all numbers, instead of float64.
	// This allows us to parse numbers that are too large for a float64, like uint64 or decimal.
	dec.UseNumber()
	return dec
}

// UnmarshalJSON implements the json.Unmarshaler interface for TableFragment.
// See decodeTableFragment for further explanation.
func (t *TableFragment) UnmarshalJSON(b []byte) error {
	decoder := newDecoder(bytes.NewReader(b))

	rows, err := decodeTableFragment(b, decoder, t.Columns, t.PreviousIndex)
	if err != nil {
		return err
	}
	t.Rows = rows

	return nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for DataTable.
// A DataTable is "just" a TableHeader and TableFragment, so we can reuse the existing functions.
func (q *DataTable) UnmarshalJSON(b []byte) error {
	decoder := newDecoder(bytes.NewReader(b))

	err := decodeHeader(decoder, &q.Header, DataTableFrameType)
	if err != nil {
		return err
	}

	rows, err := decodeTableFragment(b, decoder, q.Header.Columns, 0)
	if err != nil {
		return err
	}
	q.Rows = rows

	return nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for DataSetHeader.
// We need to decode this manually to set the correct Columns, in order to save on allocations later on.
func (t *TableHeader) UnmarshalJSON(b []byte) error {
	decoder := newDecoder(bytes.NewReader(b))

	err := decodeHeader(decoder, t, TableHeaderFrameType)
	if err != nil {
		return err
	}

	return nil
}

// decodeHeader decodes the header of a table, which is the same for TableHeader and DataTable.
// It assumes the order of the properties in the JSON is fixed.
func decodeHeader(decoder *json.Decoder, t *TableHeader, frameType FrameType) error {
	err := assertToken(decoder, json.Delim('{'))
	if err != nil {
		return err
	}

	err = assertStringProperty(decoder, "FrameType", string(frameType))
	if err != nil {
		return err
	}

	t.TableId, err = getIntProperty(decoder, "TableId")
	if err != nil {
		return err
	}

	t.TableKind, err = getStringProperty(decoder, "TableKind")
	if err != nil {
		return err
	}

	t.TableName, err = getStringProperty(decoder, "TableName")
	if err != nil {
		return err
	}

	err = assertToken(decoder, json.Token("Columns"))
	if err != nil {
		return err
	}

	t.Columns, err = decodeColumns(decoder)
	if err != nil {
		return err
	}
	return nil
}

// decodeTableFragment decodes the common part of a TableFragment and DataTable - the rows.
func decodeTableFragment(b []byte, decoder *json.Decoder, columns []query.Column, previousIndex int) ([]query.Row, error) {

	// skip properties until we reach the Rows property (guaranteed to be the last one)
	for {
		tok, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		if tok == json.Token("Rows") {
			break
		}
	}

	rows, err := decodeRows(b, decoder, columns, previousIndex)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

// decodeColumns decodes the columns of a table from the JSON.
// Columns is an array of the form [ { "ColumnName": "name", "ColumnType": "type" }, ... ]
// 1. We need to set the ColumnIndex, which is not present in the JSON
// 2. We need to normalize the column type - in rare cases, kusto has type aliases like "date" instead of "datetime", and we need to normalize them
// 3. We need to validate the column type - if it's not a valid type, we should return an error
func decodeColumns(decoder *json.Decoder) ([]query.Column, error) {
	cols := make([]query.Column, 0)

	err := assertToken(decoder, json.Delim('['))
	if err != nil {
		return nil, err
	}

	for i := 0; decoder.More(); i++ {
		col := FrameColumn{
			ColumnIndex: i,
		}
		decoder.Decode(&col)
		// Normalize the column type - error is an empty string
		col.ColumnType = string(types.NormalizeColumn(col.ColumnType))
		if col.ColumnType == "" {
			return nil, errors.ES(errors.OpTableAccess, errors.KClientArgs, "column[%d] is of type %s, which is not valid", i, col.ColumnType)
		}
		cols = append(cols, col)
	}

	if err := assertToken(decoder, json.Delim(']')); err != nil {
		return nil, err
	}

	return cols, nil
}

// decodeRows decodes the rows of a table from the JSON.
// Rows is an array of the form [ [value1, value2, ...], ... ]
// In V2 Fragmented, it's guaranteed that no errors will appear in the middle of the array, only at the end of the table.
// This function:
// 1. Creates a cached map of column names to columns for faster lookup
// 2. Decodes the rows into a slice of query.Rows
func decodeRows(b []byte, decoder *json.Decoder, cols []query.Column, startIndex int) ([]query.Row, error) {
	const RowArrayAllocSize = 10
	var rows = make([]query.Row, 0, RowArrayAllocSize)

	columnsByName := make(map[string]query.Column, len(cols))
	for _, c := range cols {
		columnsByName[c.Name()] = c
	}

	err := assertToken(decoder, json.Delim('['))
	if err != nil {
		return nil, err
	}

	for i := startIndex; decoder.More(); i++ {
		rowValues, err := decodeRow(b, decoder, cols)
		if err != nil {
			return nil, err
		}

		row := query.NewRowFromParts(cols, func(name string) query.Column { return columnsByName[name] }, i, rowValues)
		rows = append(rows, row)
	}

	if err := assertToken(decoder, json.Delim(']')); err != nil {
		return nil, err
	}
	return rows, nil
}

// decodeRow decodes a single row from the JSON.
// A row is an array of values of the types from kusto, as indicated by the columns.
// For dynamic values, they can appear as nested arrays or objects, so we need to handle them.
// Otherwise, we just unmarshal the value into the correct type.
func decodeRow(
	buffer []byte,
	decoder *json.Decoder,
	cols []query.Column) (value.Values, error) {

	err := assertToken(decoder, json.Delim('['))
	if err != nil {
		return nil, err
	}

	values := make([]value.Kusto, 0, len(cols))

	field := 0

	for ; decoder.More(); field++ {
		t, err := decoder.Token()
		if err != nil {
			return nil, err
		}

		// Handle nested values
		if t == json.Delim('[') || t == json.Delim('{') {
			t, err = decodeNestedValue(decoder, buffer)
			if err != nil {
				return nil, err
			}
		}

		// Create a new value of the correct type
		kustoValue := value.Default(cols[field].Type())

		// Unmarshal the value
		err = kustoValue.Unmarshal(t)
		if err != nil {
			return nil, err
		}

		values = append(values, kustoValue)
	}

	err = assertToken(decoder, json.Delim(']'))
	if err != nil {
		return nil, err
	}

	return values, nil
}

// decodeNestedValue decodes a nested value from the JSON into a byte array inside a json.Token.
// How it works:
// 1. We need the original buffer to be able to extract the nested value from the offsets.
// 2. We get the starting offset of the nested value.
// 3. We get the next tokens, we ignore all of them unless they start a new nested value.
// 4. If we find a nested value, we increase the nesting level, and decrease it when we find the closing token.
// 5. At the end, we're guaranteed to be at the end of original the nested value.
// 6. We get the final offset of the nested value.
// 7. We return a json.Token that points to the entire byte range of the nested value.
func decodeNestedValue(decoder *json.Decoder, buffer []byte) (json.Token, error) {
	nest := 1
	initialOffset := decoder.InputOffset() - 1
	for {
		for decoder.More() {
			t, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			if t == json.Delim('[') || t == json.Delim('{') {
				nest++
			}
		}
		t, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		if t == json.Delim(']') || t == json.Delim('}') {
			nest--
		}
		if nest == 0 {
			break
		}
	}
	finalOffset := decoder.InputOffset()

	return json.Token(buffer[initialOffset:finalOffset]), nil
}

// validateDataSetHeader makes sure the dataset header is valid for V2 Fragmented Query.
func validateDataSetHeader(dec *json.Decoder) error {
	const HeaderVersion = "v2.0"
	const NotProgressive = false
	const IsFragmented = true
	const ErrorReportingEndOfTable = "EndOfTable"

	if err := assertToken(dec, json.Delim('{')); err != nil {
		return err
	}

	if err := assertStringProperty(dec, "FrameType", json.Token(string(DataSetHeaderFrameType))); err != nil {
		return err
	}

	if err := assertStringProperty(dec, "IsProgressive", json.Token(NotProgressive)); err != nil {
		return err
	}

	if err := assertStringProperty(dec, "Version", json.Token(HeaderVersion)); err != nil {
		return err
	}

	if err := assertStringProperty(dec, "IsFragmented", json.Token(IsFragmented)); err != nil {
		return err
	}

	if err := assertStringProperty(dec, "ErrorReportingPlacement", json.Token(ErrorReportingEndOfTable)); err != nil {
		return err
	}

	return nil
}
