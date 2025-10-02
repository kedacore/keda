package lineprotocol

import "fmt"

// ValueKind represents the type of a field value.
type ValueKind uint8

const (
	Unknown ValueKind = iota
	String
	Int
	Uint
	Float
	Bool
)

var kinds = []string{
	Unknown: "unknown",
	String:  "string",
	Int:     "int",
	Uint:    "uint",
	Float:   "float",
	Bool:    "bool",
}

// String returns k as a string. It panics if k isn't one of the
// enumerated ValueKind constants. The string form is
// the lower-case form of the constant.
func (k ValueKind) String() string {
	return kinds[k]
}

// MarshalText implements encoding.TextMarshaler for ValueKind.
// It returns an error if k is Unknown.
func (k ValueKind) MarshalText() ([]byte, error) {
	if k == Unknown {
		return nil, fmt.Errorf("cannot marshal 'unknown' value type")
	}
	return []byte(k.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler for ValueKind.
func (k *ValueKind) UnmarshalText(data []byte) error {
	s := string(data)
	for i, kstr := range kinds {
		if i > 0 && kstr == s {
			*k = ValueKind(i)
			return nil
		}
	}
	return fmt.Errorf("unknown Value kind %q", s)
}
