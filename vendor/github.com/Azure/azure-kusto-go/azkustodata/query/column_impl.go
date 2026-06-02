package query

import (
	"github.com/Azure/azure-kusto-go/azkustodata/types"
)

// column is a basic implementation of Column, to be used by specific implementations.
type column struct {
	index     int
	name      string
	kustoType types.Column
}

func (c column) Index() int {
	return c.index
}

func (c column) Name() string {
	return c.name
}

func (c column) Type() types.Column {
	return c.kustoType
}

func NewColumn(ordinal int, name string, kustoType types.Column) Column {
	return &column{
		index:     ordinal,
		name:      name,
		kustoType: kustoType,
	}
}
