package kql

import (
	"github.com/Azure/azure-kusto-go/azkustodata/value"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"sort"
	"strings"
	"time"
)

type Parameters struct {
	parameters map[string]value.Kusto
}

func NewParameters() *Parameters {
	return &Parameters{parameters: make(map[string]value.Kusto)}
}

func (q *Parameters) Count() int {
	return len(q.parameters)
}
func (q *Parameters) AddValue(key string, v value.Kusto) *Parameters {
	if RequiresQuoting(key) {
		panic("Invalid parameter values. make sure to adhere to KQL entity name conventions and escaping rules.")
	}
	q.parameters[key] = v
	return q
}

func (q *Parameters) AddBool(key string, v bool) *Parameters {
	return q.AddValue(key, value.NewBool(v))
}

func (q *Parameters) AddDateTime(key string, v time.Time) *Parameters {
	return q.AddValue(key, value.NewDateTime(v))
}

func (q *Parameters) AddDynamic(key string, v interface{}) *Parameters {
	return q.AddValue(key, value.DynamicFromInterface(v))
}

func (q *Parameters) AddSerializedDynamic(key string, v []byte) *Parameters {
	return q.AddValue(key, value.NewDynamic(v))
}

func (q *Parameters) AddGUID(key string, v uuid.UUID) *Parameters {
	return q.AddValue(key, value.NewGUID(v))
}

func (q *Parameters) AddInt(key string, v int32) *Parameters {
	return q.AddValue(key, value.NewInt(v))
}

func (q *Parameters) AddLong(key string, v int64) *Parameters {
	return q.AddValue(key, value.NewLong(v))
}

func (q *Parameters) AddReal(key string, v float64) *Parameters {
	return q.AddValue(key, value.NewReal(v))
}

func (q *Parameters) AddString(key string, v string) *Parameters {
	return q.AddValue(key, value.NewString(v))
}

func (q *Parameters) AddTimespan(key string, v time.Duration) *Parameters {
	return q.AddValue(key, value.NewTimespan(v))
}

func (q *Parameters) AddDecimal(key string, v decimal.Decimal) *Parameters {
	return q.AddValue(key, value.NewDecimal(v))
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

	keys := make([]string, 0, len(q.parameters))
	for k := range q.parameters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, key := range keys {
		build.WriteString(key)
		build.WriteString(":")
		build.WriteString(string(q.parameters[key].GetType()))
		if i < len(keys)-1 {
			build.WriteString(", ")
		}
	}
	build.WriteString(closeStmt)
	return build.String()
}
func (q *Parameters) ToParameterCollection() map[string]string {
	var parameters = make(map[string]string)
	for key, paramVals := range q.parameters {
		parameters[key] = QuoteValue(paramVals)
	}
	return parameters
}

// Reset resets the parameters map
func (q *Parameters) Reset() {
	q.parameters = make(map[string]value.Kusto)
}
