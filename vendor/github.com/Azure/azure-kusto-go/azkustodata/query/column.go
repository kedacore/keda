package query

import "github.com/Azure/azure-kusto-go/azkustodata/types"

// Column represents a column in a table.
type Column interface {
	// Ordinal returns the column's index in the table.
	Index() int
	// Name returns the column's name.
	Name() string
	// Type returns the column's kusto data type.
	Type() types.Column
}

type Columns []Column
