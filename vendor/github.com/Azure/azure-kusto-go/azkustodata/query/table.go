package query

import "github.com/Azure/azure-kusto-go/azkustodata/errors"

type BaseTable interface {
	Id() string
	Index() int64
	Name() string
	Columns() []Column
	Kind() string
	ColumnByName(name string) Column
	Op() errors.Op
	IsPrimaryResult() bool
}

type Table interface {
	BaseTable
	Rows() []Row
}

// IterativeTable is a table that returns rows one at a time.
type IterativeTable interface {
	BaseTable
	// Rows returns a channel that will be populated with rows as they are read.
	Rows() <-chan RowResult
	ToTable() (Table, error)
}
