package v1

import (
	"bufio"
	"bytes"
	"encoding/json"
	"github.com/Azure/azure-kusto-go/azkustodata/errors"
	"io"
)

type RawRow struct {
	Row    []interface{}
	Errors []string
}

func newDecoder(r io.Reader) *json.Decoder {
	dec := json.NewDecoder(r)
	// This option uses the json.Number type for all numbers, instead of float64.
	// This allows us to parse numbers that are too large for a float64, like uint64 or decimal.
	dec.UseNumber()
	return dec
}

// UnmarshalJSON implements the json.Unmarshaler interface, to decode a RawRow from JSON.
// It needs special handling, because the field may be a Row or a list of Errors.
func (r *RawRow) UnmarshalJSON(data []byte) error {
	var row []interface{}
	var errs struct {
		Errors []string `json:"Exceptions"`
	}

	var err error

	reader := bytes.NewReader(data)
	dec := newDecoder(reader)

	if err = dec.Decode(&row); err != nil {
		_, err := reader.Seek(0, io.SeekStart)
		if err != nil {
			return err
		}

		if err = dec.Decode(&errs); err != nil {
			return err
		}
		r.Errors = errs.Errors
		r.Row = nil
		return nil
	}
	r.Row = row
	r.Errors = nil
	return nil
}

type RawColumn struct {
	ColumnName string `json:"ColumnName"`
	DataType   string `json:"DataType"`
	ColumnType string `json:"ColumnType"`
}

type RawTable struct {
	TableName string      `json:"TableName"`
	Columns   []RawColumn `json:"Columns"`
	Rows      []RawRow    `json:"Rows"`
}

type V1 struct {
	Tables     []RawTable `json:"Tables"`
	Exceptions []string   `json:"Exceptions"`
}

func decodeV1(data io.ReadCloser) (*V1, error) {
	var v1 V1
	br := bufio.NewReader(data)
	peek, err := br.Peek(1)
	if err != nil {
		return nil, err
	}
	if peek[0] != '{' {
		all, err := io.ReadAll(br)
		if err != nil {
			return nil, err
		}
		return nil, errors.ES(errors.OpUnknown, errors.KInternal, "Got error: %v", string(all))
	}

	dec := newDecoder(br)
	err = dec.Decode(&v1)
	if err != nil {
		return nil, err
	}

	return &v1, nil
}
