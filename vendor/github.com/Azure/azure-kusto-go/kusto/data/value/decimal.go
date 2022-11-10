package value

import (
	"fmt"
	"math/big"
	"reflect"
	"regexp"
)

// Decimal represents a Kusto decimal type.  Decimal implements Kusto.
// Because Go does not have a dynamic decimal type that meets all needs, Decimal
// provides the string representation for you to unmarshal into.
type Decimal struct {
	// Value holds the value of the type.
	Value string
	// Valid indicates if this value was set.
	Valid bool
}

func (Decimal) isKustoVal() {}

// String implements fmt.Stringer.
func (d Decimal) String() string {
	if !d.Valid {
		return ""
	}
	return d.Value
}

// ParseFloat provides builtin support for Go's *big.Float conversion where that type meets your needs.
func (d *Decimal) ParseFloat(base int, prec uint, mode big.RoundingMode) (f *big.Float, b int, err error) {
	return big.ParseFloat(d.Value, base, prec, mode)
}

var DecRE = regexp.MustCompile(`^((\d+\.?\d*)|(\d*\.?\d+))$`) // Matches decimal numbers, with or without decimal dot, with optional parts missing.

// Unmarshal unmarshals i into Decimal. i must be a string representing a decimal type or nil.
func (d *Decimal) Unmarshal(i interface{}) error {
	if i == nil {
		d.Value = ""
		d.Valid = false
		return nil
	}

	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("Column with type 'decimal' had type %T", i)
	}

	if !DecRE.MatchString(v) {
		return fmt.Errorf("column with type 'decimal' does not appear to be a decimal number, was %v", v)
	}

	d.Value = v
	d.Valid = true
	return nil
}

// Convert Decimal into reflect value.
func (d Decimal) Convert(v reflect.Value) error {
	t := v.Type()
	switch {
	case t.Kind() == reflect.String:
		if d.Valid {
			v.Set(reflect.ValueOf(d.Value))
		}
		return nil
	case t.ConvertibleTo(reflect.TypeOf(new(string))):
		if d.Valid {
			i := &d.Value
			v.Set(reflect.ValueOf(i))
		}
		return nil
	case t.ConvertibleTo(reflect.TypeOf(Decimal{})):
		v.Set(reflect.ValueOf(d))
		return nil
	case t.ConvertibleTo(reflect.TypeOf(&Decimal{})):
		v.Set(reflect.ValueOf(&d))
		return nil
	}
	return fmt.Errorf("Column was type Kusto.Decimal, receiver had base Kind %s ", t.Kind())
}
