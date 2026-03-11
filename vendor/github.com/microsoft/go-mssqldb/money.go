package mssql

import (
	"database/sql"
	"database/sql/driver"

	"github.com/shopspring/decimal"
)

type Money[D decimal.Decimal | decimal.NullDecimal] struct {
	Decimal D
}

func (m Money[D]) Value() (driver.Value, error) {
	valuer, _ := any(m.Decimal).(driver.Valuer)

	return valuer.Value()
}

func (m *Money[D]) Scan(v any) error {
	scanner, _ := any(&m.Decimal).(sql.Scanner)

	return scanner.Scan(v)
}
