package kql

import (
	"errors"
	"fmt"
	"github.com/Azure/azure-kusto-go/kusto/data/types"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"strings"
	"time"
)

// stringConstant is an internal type that cannot be created outside the package.  The only two ways to build
// a stringConstant is to pass a string constant or use a local function to build the stringConstant.
// This allows us to enforce the use of constants or strings built with injection protection.
type stringConstant string

// String implements fmt.Stringer.
func (s stringConstant) String() string {
	return string(s)
}

type Builder struct {
	builder strings.Builder
}

func New(value stringConstant) *Builder {
	return (&Builder{
		builder: strings.Builder{},
	}).AddLiteral(value)
}

func FromBuilder(builder *Builder) *Builder {
	return New(stringConstant(builder.String()))
}

// String implements fmt.Stringer.
func (b *Builder) String() string {
	return b.builder.String()
}
func (b *Builder) addBase(value fmt.Stringer) *Builder {
	b.builder.WriteString(value.String())
	return b
}

// AddUnsafe enables unsafe actions on a Builder - adds a string as is, no validation checking or escaping.
// This turns off safety features that could allow a service client to compromise your data store.
// USE AT YOUR OWN RISK!
func (b *Builder) AddUnsafe(value string) *Builder {
	b.builder.WriteString(value)
	return b
}

func (b *Builder) AddLiteral(value stringConstant) *Builder {
	return b.addBase(value)
}

func (b *Builder) AddBool(value bool) *Builder {
	return b.addBase(newValue(value, types.Bool))
}

func (b *Builder) AddDateTime(value time.Time) *Builder {
	return b.addBase(newValue(value, types.DateTime))
}

func (b *Builder) AddDynamic(value interface{}) *Builder {
	return b.addBase(newValue(value, types.Dynamic))
}

func (b *Builder) AddGUID(value uuid.UUID) *Builder {
	return b.addBase(newValue(value, types.GUID))
}

func (b *Builder) AddInt(value int32) *Builder {
	return b.addBase(newValue(value, types.Int))
}

func (b *Builder) AddLong(value int64) *Builder {
	return b.addBase(newValue(value, types.Long))
}

func (b *Builder) AddReal(value float64) *Builder {
	return b.addBase(newValue(value, types.Real))
}

func (b *Builder) AddString(value string) *Builder {
	return b.addBase(newValue(value, types.String))
}

func (b *Builder) AddTimespan(value time.Duration) *Builder {
	return b.addBase(newValue(value, types.Timespan))
}

func (b *Builder) AddDecimal(value decimal.Decimal) *Builder {
	return b.addBase(newValue(value, types.Decimal))
}

func (b *Builder) GetParameters() (map[string]string, error) {
	return nil, errors.New("this option does not support Parameters")
}
func (b *Builder) SupportsInlineParameters() bool {
	return false
}

// Reset resets the stringBuilder
func (b *Builder) Reset() {
	b.builder.Reset()
}
