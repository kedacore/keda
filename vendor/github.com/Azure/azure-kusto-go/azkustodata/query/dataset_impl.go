package query

import (
	"context"
	"github.com/Azure/azure-kusto-go/azkustodata/errors"
)

// baseDataset is a basic implementation of BaseDataset, to be used by specific implementations.
type baseDataset struct {
	ctx                context.Context
	op                 errors.Op
	primaryResultsKind string
}

func (d *baseDataset) Context() context.Context {
	return d.ctx
}

func (d *baseDataset) Op() errors.Op {
	return d.op
}

func (d *baseDataset) PrimaryResultKind() string {
	return d.primaryResultsKind
}

func NewBaseDataset(ctx context.Context, op errors.Op, primaryResultsKind string) BaseDataset {
	return &baseDataset{
		ctx:                ctx,
		op:                 op,
		primaryResultsKind: primaryResultsKind,
	}
}

type dataset struct {
	BaseDataset
	tables []Table
}

func NewDataset(base BaseDataset, tables []Table) Dataset {
	return &dataset{
		BaseDataset: base,
		tables:      tables,
	}
}

func (d *dataset) Tables() []Table {
	return d.tables
}
