package kql

import (
	"github.com/Azure/azure-kusto-go/kusto/data/types"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"strings"
	"time"
)

type Parameters struct {
	parameters map[string]Value
}

func NewParameters() *Parameters {
	return &Parameters{parameters: make(map[string]Value)}
}

func (q *Parameters) Count() int {
	return len(q.parameters)
}
func (q *Parameters) addBase(key string, value Value) *Parameters {
	if RequiresQuoting(key) {
		panic("Invalid parameter values. make sure to adhere to KQL entity name conventions and escaping rules.")
	}
	q.parameters[key] = value
	return q
}

func (q *Parameters) AddBool(key string, value bool) *Parameters {
	return q.addBase(key, newValue(value, types.Bool))
}

func (q *Parameters) AddDateTime(key string, value time.Time) *Parameters {
	return q.addBase(key, newValue(value, types.DateTime))
}

func (q *Parameters) AddDynamic(key string, value interface{}) *Parameters {
	return q.addBase(key, newValue(value, types.Dynamic))
}

func (q *Parameters) AddGUID(key string, value uuid.UUID) *Parameters {
	return q.addBase(key, newValue(value, types.GUID))
}

func (q *Parameters) AddInt(key string, value int32) *Parameters {
	return q.addBase(key, newValue(value, types.Int))
}

func (q *Parameters) AddLong(key string, value int64) *Parameters {
	return q.addBase(key, newValue(value, types.Long))
}

func (q *Parameters) AddReal(key string, value float64) *Parameters {
	return q.addBase(key, newValue(value, types.Real))
}

func (q *Parameters) AddString(key string, value string) *Parameters {
	return q.addBase(key, newValue(value, types.String))
}

func (q *Parameters) AddTimespan(key string, value time.Duration) *Parameters {
	return q.addBase(key, newValue(value, types.Timespan))
}

func (q *Parameters) AddDecimal(key string, value decimal.Decimal) *Parameters {
	return q.addBase(key, newValue(value, types.Decimal))
}

func (q *Parameters) ToDeclarationString() string {
	const (
		declare   = "declare query_parameters("
		closeStmt = ");"
	)
	var build = strings.Builder{}

	if len(q.parameters) == 0 {
		return ""
	}

	build.WriteString(declare)
	comma := len(q.parameters)
	for key, paramVals := range q.parameters {
		build.WriteString(key)
		build.WriteString(":")
		build.WriteString(string(paramVals.Type()))
		if comma > 1 {
			build.WriteString(", ")
		}
		comma--
	}
	build.WriteString(closeStmt)
	return build.String()
}
func (q *Parameters) ToParameterCollection() map[string]string {
	var parameters = make(map[string]string)
	for key, paramVals := range q.parameters {
		parameters[key] = paramVals.String()
	}
	return parameters
}

// Reset resets the parameters map
func (q *Parameters) Reset() {
	q.parameters = make(map[string]Value)
}
