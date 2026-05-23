package v2

import (
	"context"
	"github.com/Azure/azure-kusto-go/azkustodata/query"
	"sync/atomic"
)

// iterativeTable represents a table that is streamed from the service.
// It is used by the iterative dataset.
// The rows are received from the service via the rawRows channel, and are parsed and sent to the rows channel.
type iterativeTable struct {
	query.BaseTable
	// a channel of rows and errors, exposed to the user
	rows chan query.RowResult
	// the number of rows in the table, updated as rows are received
	rowCount atomic.Uint32
	// a context for the table
	ctx context.Context
}

// addRawRows is called by the dataset to add rows to the table.
// It will add the rows to the table, unless the table is already skipped.
func (t *iterativeTable) addRawRows(rows []query.Row) {
	for _, row := range rows {
		if !t.reportRow(row) {
			return
		}
		t.rowCount.Add(1)
	}
}

// RowCount returns the current number of rows in the table.
func (t *iterativeTable) RowCount() int {
	return int(t.rowCount.Load())
}

func (t *iterativeTable) setRowCount(rowCount int) {
	t.rowCount.Store(uint32(rowCount))
}

func NewIterativeTable(dataset *iterativeDataset, th TableHeader) (query.IterativeTable, error) {
	baseTable, err := newBaseTableFromHeader(dataset, th)
	if err != nil {
		return nil, err
	}

	t := &iterativeTable{
		BaseTable: baseTable,
		ctx:       dataset.Context(),
		rows:      make(chan query.RowResult, dataset.rowCapacity),
	}

	return t, nil
}

func (t *iterativeTable) finishTable(errs []OneApiError, cancelError error) {
	if cancelError != nil {
		t.reportError(cancelError)
	} else if errs != nil {
		t.reportError(combineOneApiErrors(errs))
	}
	close(t.rows)
}

func (t *iterativeTable) reportRow(row query.Row) bool {
	select {
	case t.rows <- query.RowResultSuccess(row):
		return true
	case <-t.ctx.Done():
		return false
	}
}

func (t *iterativeTable) reportError(err error) bool {
	select {
	case t.rows <- query.RowResultError(err):
		return true
	case <-t.ctx.Done():
		return false
	}
}

// Rows returns a channel of rows and errors.
func (t *iterativeTable) Rows() <-chan query.RowResult {
	return t.rows
}

// ToTable reads the entire table, converting it from an iterative table to a regular table.
func (t *iterativeTable) ToTable() (query.Table, error) {
	var rows []query.Row
	for r := range t.rows {
		if r.Err() != nil {
			return nil, r.Err()
		} else {
			rows = append(rows, r.Row())
		}
	}

	return query.NewTable(t.BaseTable, rows), nil
}
