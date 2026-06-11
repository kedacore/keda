package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

type scanSelectQueryFunc func(ctx context.Context, query string, args ...any) (driver.Rows, error)

func scanSelect(queryFunc scanSelectQueryFunc, ctx context.Context, dest any, query string, args ...any) error {
	value := reflect.ValueOf(dest)
	if value.Kind() != reflect.Ptr {
		return &OpError{
			Op:  "Select",
			Err: errors.New("must pass a pointer, not a value, to Select destination"),
		}
	}
	if value.IsNil() {
		return &OpError{
			Op:  "Select",
			Err: errors.New("nil pointer passed to Select destination"),
		}
	}
	direct := reflect.Indirect(value)
	if direct.Kind() != reflect.Slice {
		return fmt.Errorf("must pass a slice to Select destination")
	}
	if direct.Len() != 0 {
		// dest should point to empty slice
		// to make select result correct
		direct.Set(reflect.MakeSlice(direct.Type(), 0, direct.Cap()))
	}
	var (
		base      = direct.Type().Elem()
		rows, err = queryFunc(ctx, query, args...)
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		elem := reflect.New(base)
		if err := rows.ScanStruct(elem.Interface()); err != nil {
			return err
		}
		direct.Set(reflect.Append(direct, elem.Elem()))
	}
	if err := rows.Close(); err != nil {
		return err
	}

	return rows.Err()
}

func (ch *clickhouse) Select(ctx context.Context, dest any, query string, args ...any) error {
	return scanSelect(ch.Query, ctx, dest, query, args...)
}

func scan(block *proto.Block, row int, dest ...any) error {
	columns := block.Columns
	if len(columns) != len(dest) {
		return &OpError{
			Op:  "Scan",
			Err: fmt.Errorf("expected %d destination arguments in Scan, not %d", len(columns), len(dest)),
		}
	}
	for i, d := range dest {
		if err := columns[i].ScanRow(d, row-1); err != nil {
			return &OpError{
				Err:        err,
				ColumnName: block.ColumnsNames()[i],
			}
		}
	}
	return nil
}
