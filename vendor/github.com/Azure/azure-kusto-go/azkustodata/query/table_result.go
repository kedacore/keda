package query

type TableResult interface {
	Table() IterativeTable
	Err() error
}

// TableResult is a structure that holds the result of a table operation.
// It contains a Table and an error, if any occurred during the operation.
type tableResult struct {
	// Table is the result of the operation.
	table IterativeTable
	// Err is the error that occurred during the operation, if any.
	err error
}

// Table returns the table that was the result of the operation.
func (t *tableResult) Table() IterativeTable {
	return t.table
}

// Err returns the error that occurred during the operation, if any.
func (t *tableResult) Err() error {
	return t.err
}

func TableResultSuccess(table IterativeTable) TableResult {
	return &tableResult{
		table: table,
	}
}

func TableResultError(err error) TableResult {
	return &tableResult{
		err: err,
	}
}
