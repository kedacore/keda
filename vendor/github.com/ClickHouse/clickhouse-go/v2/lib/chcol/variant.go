package chcol

import (
	"database/sql/driver"
	"encoding/json"
)

// Variant represents a ClickHouse Variant type that can hold multiple possible types
type Variant struct {
	value  any
	chType string
}

// NewVariant creates a new Variant with the given value
func NewVariant(v any) Variant {
	return Variant{
		value:  v,
		chType: "",
	}
}

// NewVariantWithType creates a new Variant with the given value and ClickHouse type
func NewVariantWithType(v any, chType string) Variant {
	return Variant{
		value:  v,
		chType: chType,
	}
}

// WithType creates a new Variant with the current value and given ClickHouse type
func (v Variant) WithType(chType string) Variant {
	return Variant{
		value:  v.value,
		chType: chType,
	}
}

// Type returns the ClickHouse type as a string.
func (v Variant) Type() string {
	return v.chType
}

// HasType returns true if the value has a type ClickHouse included.
func (v Variant) HasType() bool {
	return v.chType == ""
}

// Nil returns true if the underlying value is nil.
func (v Variant) Nil() bool {
	return v.value == nil
}

// Any returns the underlying value as any.
func (v Variant) Any() any {
	return v.value
}

// Scan implements the sql.Scanner interface
func (v *Variant) Scan(value any) error {
	switch vv := value.(type) {
	case Variant:
		v.value = vv.value
		v.chType = vv.chType
	case *Variant:
		v.value = vv.value
		v.chType = vv.chType
	default:
		v.value = value
	}

	return nil
}

// Value implements the driver.Valuer interface
func (v Variant) Value() (driver.Value, error) {
	return v, nil
}

// MarshalJSON implements the json.Marshaler interface
func (v Variant) MarshalJSON() ([]byte, error) {
	if v.Nil() {
		return []byte("null"), nil
	}

	return json.Marshal(v.value)
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (v *Variant) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		v.value = nil
		return nil
	}

	if err := json.Unmarshal(data, &v.value); err != nil {
		return err
	}

	return nil
}

// MarshalText implements the encoding.TextMarshaler interface
func (v Variant) MarshalText() ([]byte, error) {
	if v.Nil() {
		return []byte(""), nil
	}

	switch vv := v.value.(type) {
	case string:
		return []byte(vv), nil
	case []byte:
		return vv, nil
	case json.RawMessage:
		return vv, nil
	}

	return json.Marshal(v.value)
}

// UnmarshalText implements the encoding.TextUnmarshaler interface
func (v *Variant) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		v.value = nil
		return nil
	}

	v.value = string(text)
	return nil
}
