package proto

import (
	"strconv"
	"strings"

	"github.com/go-faster/errors"
)

var (
	_ Column           = (*ColEnum)(nil)
	_ ColumnOf[string] = (*ColEnum)(nil)
	_ Inferable        = (*ColEnum)(nil)
	_ Preparable       = (*ColEnum)(nil)
)

// ColEnum is inference helper for enums.
//
// You can set Values and actual enum mapping will be inferred during query
// execution.
type ColEnum struct {
	t    ColumnType
	base ColumnType

	rawToStr map[int]string
	strToRaw map[string]int
	raw8     ColEnum8
	raw16    ColEnum16

	// Values of ColEnum.
	Values []string
}

func (e *ColEnum) raw() Column {
	if e.t.Base() == ColumnTypeEnum8 {
		return &e.raw8
	}
	return &e.raw16
}

func (e ColEnum) Row(i int) string {
	return e.Values[i]
}

// Append value to Enum8 column.
func (e *ColEnum) Append(v string) {
	e.Values = append(e.Values, v)
}

func (e *ColEnum) AppendArr(vs []string) {
	e.Values = append(e.Values, vs...)
}

func (e *ColEnum) parse(t ColumnType) error {
	if e.rawToStr == nil {
		e.rawToStr = map[int]string{}
	}
	if e.strToRaw == nil {
		e.strToRaw = map[string]int{}
	}

	elements := t.Elem().String()
	for _, elem := range strings.Split(elements, ",") {
		def := strings.TrimSpace(elem)
		// 'hello' = 1
		parts := strings.SplitN(def, "=", 2)
		if len(parts) != 2 {
			return errors.Errorf("bad enum definition %q", def)
		}
		var (
			left  = strings.TrimSpace(parts[0]) // 'hello'
			right = strings.TrimSpace(parts[1]) // 1
		)
		idx, err := strconv.Atoi(right)
		if err != nil {
			return errors.Errorf("bad right side of definition %q", right)
		}
		left = strings.TrimFunc(left, func(c rune) bool {
			return c == '\''
		})
		e.strToRaw[left] = idx
		e.rawToStr[idx] = left
	}
	return nil
}

func (e *ColEnum) Infer(t ColumnType) error {
	if !strings.HasPrefix(t.Base().String(), "Enum") {
		return errors.Errorf("invalid base %q to infer enum", t.Base())
	}
	if err := e.parse(t); err != nil {
		return errors.Wrap(err, "parse type")
	}
	base := t.Base()
	switch base {
	case ColumnTypeEnum8, ColumnTypeEnum16:
		e.base = base
	default:
		return errors.Errorf("invalid base %q", base)
	}
	e.t = t
	return nil
}

func (e *ColEnum) Rows() int {
	return len(e.Values)
}

func appendEnum[E Enum8 | Enum16](c []E, mapping map[int]string, values []string) ([]string, error) {
	for _, v := range c {
		s, ok := mapping[int(v)]
		if !ok {
			return nil, errors.Errorf("unknown enum value %d", v)
		}
		values = append(values, s)
	}
	return values, nil
}

func (e *ColEnum) DecodeColumn(r *Reader, rows int) error {
	if err := e.raw().DecodeColumn(r, rows); err != nil {
		return errors.Wrap(err, "raw")
	}
	var (
		err error
		v   []string
	)
	switch e.base {
	case ColumnTypeEnum8:
		v, err = appendEnum[Enum8](e.raw8, e.rawToStr, e.Values[:0])
	case ColumnTypeEnum16:
		v, err = appendEnum[Enum16](e.raw16, e.rawToStr, e.Values[:0])
	default:
		return errors.Errorf("invalid enum base %q", e.base)
	}
	if err != nil {
		return errors.Wrap(err, "map values")
	}
	e.Values = v
	return nil
}

func (e *ColEnum) Reset() {
	e.raw().Reset()
	e.Values = e.Values[:0]
}

func (e *ColEnum) Prepare() error {
	e.raw8 = e.raw8[:0]
	e.raw16 = e.raw16[:0]
	for _, v := range e.Values {
		raw, ok := e.strToRaw[v]
		if !ok {
			return errors.Errorf("unknown enum value %q", v)
		}
		switch e.base {
		case ColumnTypeEnum8:
			e.raw8.Append(Enum8(raw))
		case ColumnTypeEnum16:
			e.raw16.Append(Enum16(raw))
		default:
			return errors.Errorf("invalid base %q", e.base)
		}
	}
	return nil
}

func (e *ColEnum) EncodeColumn(b *Buffer) {
	e.raw().EncodeColumn(b)
}

func (e *ColEnum) Type() ColumnType { return e.t }
