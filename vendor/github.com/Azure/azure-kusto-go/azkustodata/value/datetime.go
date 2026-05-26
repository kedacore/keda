package value

import (
	"fmt"
	"github.com/Azure/azure-kusto-go/azkustodata/types"
	"reflect"
	"time"
)

// DateTime represents a Kusto datetime type.  DateTime implements Kusto.
type DateTime struct {
	pointerValue[time.Time]
}

// NewDateTime creates a new DateTime.
func NewDateTime(v time.Time) *DateTime {
	return &DateTime{newPointerValue[time.Time](&v)}
}

// NewNullDateTime creates a new null DateTime.
func NewNullDateTime() *DateTime {
	return &DateTime{newPointerValue[time.Time](nil)}
}

// String implements fmt.Stringer.
func (d *DateTime) String() string {
	if d.value == nil {
		return ""
	}
	return fmt.Sprint(d.value.Format(time.RFC3339Nano))
}

// Marshal marshals the DateTime into a Kusto compatible string.
func (d *DateTime) Marshal() string {
	if d.value == nil {
		return time.Time{}.Format(time.RFC3339Nano)
	}

	return d.value.Format(time.RFC3339Nano)
}

// Unmarshal unmarshals i into DateTime. i must be a string representing RFC3339Nano or nil.
func (d *DateTime) Unmarshal(i interface{}) error {
	if i == nil {
		d.value = nil
		return nil
	}

	str, ok := i.(string)
	if !ok {
		return convertError(d, i)
	}

	t, err := time.Parse(time.RFC3339Nano, str)
	if err != nil {
		return parseError(d, i, err)
	}
	d.value = &t
	return nil
}

// Convert DateTime into reflect value.
func (d *DateTime) Convert(v reflect.Value) error {
	return Convert[time.Time](*d, &d.pointerValue, v)
}

// GetType returns the type of the value.
func (d *DateTime) GetType() types.Column {
	return types.DateTime
}
