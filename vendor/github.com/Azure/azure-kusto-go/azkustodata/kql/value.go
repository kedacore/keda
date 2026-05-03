package kql

import (
	"fmt"
	"github.com/Azure/azure-kusto-go/azkustodata/types"
	"github.com/Azure/azure-kusto-go/azkustodata/value"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"time"
)

func QuoteValue(v value.Kusto) string {
	val := v.GetValue()
	t := v.GetType()
	if val == nil {
		return fmt.Sprintf("%v(null)", t)
	}

	switch t {
	case types.String:
		return QuoteString(v.String(), false)
	case types.DateTime:
		val = FormatDatetime(*val.(*time.Time))
	case types.Timespan:
		val = FormatTimespan(*val.(*time.Duration))
	case types.Dynamic:
		val = string(val.([]byte))
	case types.Bool:
		val = *val.(*bool)
	case types.Int:
		val = *val.(*int32)
	case types.Long:
		val = *val.(*int64)
	case types.Real:
		val = *val.(*float64)
	case types.Decimal:
		val = *val.(*decimal.Decimal)
	case types.GUID:
		val = *val.(*uuid.UUID)
	}

	return fmt.Sprintf("%v(%v)", t, val)
}
