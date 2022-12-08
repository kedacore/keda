package frames

import (
	"context"
	"fmt"
	"io"

	"github.com/Azure/azure-kusto-go/kusto/data/errors"
)

const (
	// TypeDataTable is the .FrameType that indicates a Kusto DataTable.
	TypeDataTable = "DataTable"
	// TypeDataSetCompletion is the .FrameType that indicates a Kusto DataSetCompletion.
	TypeDataSetCompletion = "DataSetCompletion"
	// TypeDataSetHeader is the .FrameType that indicates a Kusto DataSetHeader.
	TypeDataSetHeader = "DataSetHeader"
	// TypeTableHeader is the .FrameType that indicates a Kusto TableHeader.
	TypeTableHeader = "TableHeader"
	// TypeTableFragment is the .FrameType that indicates a Kusto TableFragment.
	TypeTableFragment = "TableFragment"
	// TypeTableProgress is the .FrameType that indicates a Kusto TableProgress.
	TypeTableProgress = "TableProgress"
	// TypeTableCompletion is the .FrameType that indicates a Kusto TableCompletion.
	TypeTableCompletion = "TableCompletion"
)

// These constants represent keys for fields when unmarshalling various JSON dicts representing Kusto frames.
const (
	FieldFrameType         = "FrameType"
	FieldTableID           = "TableId"
	FieldTableKind         = "TableKind"
	FieldTableName         = "TableName"
	FieldColumns           = "Columns"
	FieldRows              = "Rows"
	FieldColumnName        = "ColumnName"
	FieldColumnType        = "ColumnType"
	FieldCount             = "FieldCount"
	FieldTableFragmentType = "TableFragmentType"
	FieldTableProgress     = "TableProgress"
	FieldRowCount          = "RowCount"
)

// TableKind describes the kind of table.
type TableKind string

const (
	// QueryProperties is a dataTable.TableKind that contains properties about the query itself.
	// The dataTable.TableName is usually ExtendedProperties.
	QueryProperties TableKind = "QueryProperties"
	// PrimaryResult is a dataTable.TableKind that contains the query information the user wants.
	// The dataTable.TableName is PrimaryResult.
	PrimaryResult TableKind = "PrimaryResult"
	// QueryCompletionInformation contains information on how long the query took.
	// The dataTable.TableName is QueryCompletionInformation.
	QueryCompletionInformation TableKind = "QueryCompletionInformation"
	QueryTraceLog              TableKind = "QueryTraceLog"
	QueryPerfLog               TableKind = "QueryPerfLog"
	QueryResult                TableKind = "QueryResult"
	TableOfContents            TableKind = "TableOfContents"
	QueryPlan                  TableKind = "QueryPlan"
	ExtendedProperties         TableKind = "@ExtendedProperties"
	UnknownTableKind           TableKind = "Unknown"
)

// Decoder provides a function that will decode an incoming data stream and return a channel of Frame objects.
type Decoder interface {
	// Decode decodes an io.Reader representing a stream of Kusto frames into our Frame representation.
	// The type and order of frames is dependent on the REST interface version and the progressive frame settings.
	Decode(ctx context.Context, r io.ReadCloser, op errors.Op) chan Frame
}

// Frame is a type of Kusto frame as defined in the reference document.
type Frame interface {
	IsFrame()
}

// Error is not actually a Kusto frame, but is used to signal the end of a stream
// where we encountered an error. Error implements error.
type Error struct {
	Msg string
}

// Error implements error.Error().
func (e Error) Error() string {
	return e.Msg
}

// IsFrame implements Frame.IsFrame().
func (Error) IsFrame() {}

// Errorf write a frames.Error to ch with fmt.Sprint(s, a...).
func Errorf(ctx context.Context, ch chan Frame, s string, a ...interface{}) {
	select {
	case <-ctx.Done():
	case ch <- Error{Msg: fmt.Sprintf(s, a...)}:
	}
}
