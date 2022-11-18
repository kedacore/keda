// Package v2 holds framing information for the v2 REST API.
package v2

import (
	"github.com/Azure/azure-kusto-go/kusto/data/errors"
	"github.com/Azure/azure-kusto-go/kusto/data/table"
	"github.com/Azure/azure-kusto-go/kusto/data/value"
	"github.com/Azure/azure-kusto-go/kusto/internal/frames"
	"github.com/Azure/azure-kusto-go/kusto/internal/frames/unmarshal"
	"github.com/Azure/azure-kusto-go/kusto/internal/frames/unmarshal/json"
)

// Base is information that is encoded in all frames. The fields aren't actually
// in the spec, but are transmitted on the wire.
type Base struct {
	FrameType string
}

// DataSetHeader is the first frame in a response. It implements Frame.
type DataSetHeader struct {
	Base
	// Version is the version of the APi responding. The current version is "v2.0".
	Version string
	// IsProgressive indicates that TableHeader, TableFragment, TableProgress, and TableCompletion
	IsProgressive bool

	Op errors.Op
}

func (DataSetHeader) IsFrame() {}

// DataTable is used report information as a Table with Columns as row headers and Rows as the contained
// data. It implements Frame.
type DataTable struct {
	Base
	// TableID is a numeric representation of the this dataTable in relation to other dataTables returned
	// in numeric order starting at 09.
	TableID int `json:"TableId"`
	// TableKind is a Kusto dataTable sub-type.
	TableKind frames.TableKind
	// TableName is a name for the dataTable.
	TableName frames.TableKind
	// Columns is a list of column names and their Kusto storage types.
	Columns table.Columns
	// Rows contains the table data that was fetched, along with errors.
	Rows      []interface{}
	KustoRows []value.Values
	RowErrors []errors.Error

	Op errors.Op `json:"-"`
}

// UnmarshalRaw unmarshals the raw JSON representing a DataTable.
func (d *DataTable) UnmarshalRaw(raw json.RawMessage) error {
	d.Rows = unmarshal.GetRows()
	defer func() {
		unmarshal.PutRows(d.Rows)
		d.Rows = nil
	}()

	if err := json.Unmarshal(raw, d); err != nil {
		if oe := RawToOneAPIErr(raw, d.Op); oe != nil {
			return oe
		}
		return err
	}

	v, rowErrors, err := unmarshal.Rows(d.Columns, d.Rows, d.Op)
	if err != nil {
		return err
	}
	d.KustoRows = v
	d.RowErrors = rowErrors
	return nil
}

// IsFrame implements frame.Frame.
func (DataTable) IsFrame() {}

// DataSetCompletion indicates the stream id done. It implements Frame.
type DataSetCompletion struct {
	Base
	// HasErrors indicates that their was an error in the stream.
	HasErrors bool
	// Cancelled indicates that the request was cancelled.
	Cancelled bool
	// OneAPIErrors is a list of errors encountered.
	OneAPIErrors []string `json:"OneApiErrors"`

	Op errors.Op `json:"-"`
}

// IsFrame implements frame.Frame.
func (DataSetCompletion) IsFrame() {}

// UnmarshalRaw unmarshals the raw JSON representing a DataSetCompletion.
func (d *DataSetCompletion) UnmarshalRaw(raw json.RawMessage) error {
	return json.Unmarshal(raw, &d)
}

// TableHeader indicates that instead of receiving a dataTable, we will receive a
// stream of table information. This structure holds the base information, but none
// of the row information.
type TableHeader struct {
	Base
	// TableID is a numeric representation of the this TableHeader in the stream.
	TableID int `json:"TableId"`
	// TableKind is a Kusto Table sub-type.
	TableKind frames.TableKind
	// TableName is a name for the Table.
	TableName frames.TableKind
	// Columns is a list of column names and their Kusto storage types.
	Columns table.Columns

	Op errors.Op `json:"-"`
}

// IsFrame implements frame.Frame.
func (TableHeader) IsFrame() {}

// UnmarshalRaw unmarshals the raw JSON representing a TableHeader.
func (t *TableHeader) UnmarshalRaw(raw json.RawMessage) error {
	return json.Unmarshal(raw, &t)
}

// TableFragment details the streaming data passed by server that would normally be the Row data in
type TableFragment struct {
	Base
	// TableID is a numeric representation of the this table in relation to other table parts returned.
	TableID int `json:"TableId"`
	// FieldCount is the number of  fields being returned. This should align with the len(TableHeader.Columns).
	FieldCount int
	// TableFragment type is the type of TFDataAppend or TFDataReplace.
	TableFragmentType string
	// Rows contains the the table data th[at was fetched.
	Rows      []interface{}
	KustoRows []value.Values
	RowErrors []errors.Error

	Columns table.Columns `json:"-"` // Needed for decoding values.

	Op errors.Op `json:"-"`
}

// IsFrame implements frame.Frame.
func (TableFragment) IsFrame() {}

// UnmarshalRaw unmarshals the raw JSON representing a TableFragment.
func (t *TableFragment) UnmarshalRaw(raw json.RawMessage) error {
	t.Rows = unmarshal.GetRows()
	defer func() {
		unmarshal.PutRows(t.Rows)
		t.Rows = nil
	}()

	if err := json.Unmarshal(raw, t); err != nil {
		if oe := RawToOneAPIErr(raw, t.Op); oe != nil {
			return oe
		}
		return err
	}

	v, rowErrors, err := unmarshal.Rows(t.Columns, t.Rows, t.Op)
	if err != nil {
		return err
	}
	t.KustoRows = v
	t.RowErrors = rowErrors

	return nil
}

// TableProgress interleaves with the TableFragment frame described above. It's sole purpose
// is to notify the client about the query progress.
type TableProgress struct {
	Base
	// TableID is a numeric representation of the this table in relation to other table parts returned.
	TableID int `json:"TableId"`
	// TableProgress is the progress in percent (0--100).
	TableProgress float64

	Op errors.Op `json:"-"`
}

// IsFrame implements frame.Frame.
func (TableProgress) IsFrame() {}

// UnmarshalRaw unmarshals the raw JSON representing a TableProgress.
func (t *TableProgress) UnmarshalRaw(raw json.RawMessage) error {
	return json.Unmarshal(raw, &t)
}

// TableCompletion frames marks the end of the table transmission. No more frames related to that table will be sent.
type TableCompletion struct {
	Base
	// TableID is a numeric representation of the this table in relation to other table parts returned.
	TableID int `json:"TableId"`
	// RowCount is the final number of rows in the table.
	RowCount int

	Op errors.Op `json:"-"`
}

// IsFrame implements frame.Frame.
func (TableCompletion) IsFrame() {}

// UnmarshalRaw unmarshals the raw JSON representing a TableCompletion.
func (t *TableCompletion) UnmarshalRaw(raw json.RawMessage) error {
	return json.Unmarshal(raw, &t)
}

// RawToOneAPIErr returns a OneAPI error if it is buried where the "Row" should be. Otherwise it returns nil.
func RawToOneAPIErr(raw json.RawMessage, op errors.Op) error {
	m := map[string]interface{}{}

	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}

	if oe, ok := m[frames.FieldRows]; ok {
		entireErr, ok := oe.(map[string]interface{})
		if ok {
			return errors.OneToErr(entireErr, op)
		}
	}
	return nil
}
