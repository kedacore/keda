package v2

import (
	"github.com/Azure/azure-kusto-go/azkustodata/query"
	"github.com/Azure/azure-kusto-go/azkustodata/types"
)

type FrameColumn struct {
	ColumnIndex int    `json:"-"`
	ColumnName  string `json:"ColumnName"`
	ColumnType  string `json:"ColumnType"`
}

func (f FrameColumn) Index() int {
	return f.ColumnIndex
}

func (f FrameColumn) Name() string {
	return f.ColumnName
}

func (f FrameColumn) Type() types.Column {
	return types.Column(f.ColumnType)
}

type DataTable struct {
	Header TableHeader
	Rows   []query.Row
}

type FrameType string

const (
	DataSetHeaderFrameType     FrameType = "DataSetHeader"
	DataTableFrameType         FrameType = "DataTable"
	TableHeaderFrameType       FrameType = "TableHeader"
	TableFragmentFrameType     FrameType = "TableFragment"
	TableCompletionFrameType   FrameType = "TableCompletion"
	DataSetCompletionFrameType FrameType = "DataSetCompletion"
)

type DataSetHeader struct {
	IsProgressive           bool
	Version                 string
	IsFragmented            bool
	ErrorReportingPlacement string
}

type TableHeader struct {
	TableId   int
	TableKind string
	TableName string
	Columns   []query.Column
}

type TableFragment struct {
	Columns       []query.Column
	Rows          []query.Row
	PreviousIndex int
}

type TableCompletion struct {
	TableId      int
	RowCount     int
	OneApiErrors []OneApiError
}

type DataSetCompletion struct {
	HasErrors    bool
	Cancelled    bool
	OneApiErrors []OneApiError
}
