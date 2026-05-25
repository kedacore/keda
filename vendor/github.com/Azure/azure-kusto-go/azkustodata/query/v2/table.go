package v2

import (
	"github.com/Azure/azure-kusto-go/azkustodata/errors"
	"github.com/Azure/azure-kusto-go/azkustodata/query"
	"strconv"
)

func newBaseTable(dataset query.BaseDataset, id int, name string, kind string, columns []query.Column) (query.BaseTable, error) {
	return query.NewBaseTable(dataset, int64(id), strconv.Itoa(id), name, kind, columns), nil
}

func newBaseTableFromHeader(dataset query.BaseDataset, th TableHeader) (query.BaseTable, error) {
	return newBaseTable(dataset, th.TableId, th.TableName, th.TableKind, th.Columns)
}

func newTable(dataset query.BaseDataset, dt DataTable) (query.Table, error) {
	base, err := newBaseTable(dataset, dt.Header.TableId, dt.Header.TableName, dt.Header.TableKind, dt.Header.Columns)
	if err != nil {
		return nil, err
	}

	return query.NewTable(base, dt.Rows), nil
}

type iterativeWrapper struct {
	table query.Table
}

func (f iterativeWrapper) Id() string { return f.table.Id() }

func (f iterativeWrapper) Index() int64 { return f.table.Index() }

func (f iterativeWrapper) Name() string { return f.table.Name() }

func (f iterativeWrapper) Columns() []query.Column { return f.table.Columns() }

func (f iterativeWrapper) Kind() string { return f.table.Kind() }

func (f iterativeWrapper) ColumnByName(name string) query.Column {
	return f.table.ColumnByName(name)
}

func (f iterativeWrapper) Op() errors.Op { return f.table.Op() }

func (f iterativeWrapper) IsPrimaryResult() bool { return f.table.IsPrimaryResult() }

func (f iterativeWrapper) ToTable() (query.Table, error) { return f.table, nil }

func (f iterativeWrapper) Rows() <-chan query.RowResult {
	ch := make(chan query.RowResult, len(f.table.Rows()))
	go func() {
		defer close(ch)
		for _, row := range f.table.Rows() {
			ch <- query.RowResultSuccess(row)
		}
	}()
	return ch
}
