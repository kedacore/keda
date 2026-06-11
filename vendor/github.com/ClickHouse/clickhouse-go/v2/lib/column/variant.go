package column

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"

	"github.com/ClickHouse/ch-go/proto"

	"github.com/ClickHouse/clickhouse-go/v2/lib/chcol"
)

const SupportedVariantSerializationVersion = 0
const VariantNullDiscriminator uint8 = 255

type Variant struct {
	chType Type
	name   string

	discriminators []uint8
	offsets        []int

	columns         []Interface
	columnTypeIndex map[string]uint8
}

func (c *Variant) parse(t Type, sc *ServerContext) (_ *Variant, err error) {
	c.chType = t
	var (
		element       []rune
		elements      []Type
		brackets      int
		appendElement = func() {
			if len(element) != 0 {
				cType := strings.TrimSpace(string(element))
				if parts := strings.SplitN(cType, " ", 2); len(parts) == 2 {
					if !strings.Contains(parts[0], "(") {
						cType = parts[1]
					}
				}

				elements = append(elements, Type(strings.TrimSpace(cType)))
			}
		}
	)

	for _, r := range t.params() {
		switch r {
		case '(':
			brackets++
		case ')':
			brackets--
		case ',':
			if brackets == 0 {
				appendElement()
				element = element[:0]
				continue
			}
		}
		element = append(element, r)
	}

	appendElement()

	c.columnTypeIndex = make(map[string]uint8, len(elements))
	for _, columnType := range elements {
		column, err := columnType.Column("", sc)
		if err != nil {
			return nil, err
		}

		c.addColumn(column)
	}

	if len(c.columns) != 0 {
		return c, nil
	}

	return nil, &UnsupportedColumnTypeError{
		t: t,
	}
}

func (c *Variant) addColumn(col Interface) {
	c.columns = append(c.columns, col)
	c.columnTypeIndex[string(col.Type())] = uint8(len(c.columns) - 1)
}

func (c *Variant) appendDiscriminatorRow(d uint8) {
	c.discriminators = append(c.discriminators, d)
}

func (c *Variant) appendNullRow() {
	c.appendDiscriminatorRow(VariantNullDiscriminator)
}

func (c *Variant) Name() string {
	return c.name
}

func (c *Variant) Type() Type {
	return c.chType
}

func (c *Variant) Rows() int {
	return len(c.discriminators)
}

func (c *Variant) Row(i int, ptr bool) any {
	typeIndex := c.discriminators[i]
	offsetIndex := c.offsets[i]
	var value any
	var chType string
	if typeIndex != VariantNullDiscriminator {
		value = c.columns[typeIndex].Row(offsetIndex, ptr)
		chType = string(c.columns[typeIndex].Type())
	}

	vt := chcol.NewVariantWithType(value, chType)
	if ptr {
		return &vt
	}

	return vt
}

func (c *Variant) ScanRow(dest any, row int) error {
	typeIndex := c.discriminators[row]
	offsetIndex := c.offsets[row]
	var value any
	var chType string
	if typeIndex != VariantNullDiscriminator {
		value = c.columns[typeIndex].Row(offsetIndex, false)
		chType = string(c.columns[typeIndex].Type())
	}

	switch v := dest.(type) {
	case *chcol.Variant:
		vt := chcol.NewVariantWithType(value, chType)
		*v = vt
	case **chcol.Variant:
		vt := chcol.NewVariantWithType(value, chType)
		**v = vt
	default:
		if typeIndex == VariantNullDiscriminator {
			return nil
		}

		if err := c.columns[typeIndex].ScanRow(dest, offsetIndex); err != nil {
			return err
		}
	}

	return nil
}

func (c *Variant) Append(v any) (nulls []uint8, err error) {
	switch vv := v.(type) {
	case []chcol.Variant:
		for i, vt := range vv {
			err := c.AppendRow(vt)
			if err != nil {
				return nil, fmt.Errorf("failed to AppendRow at index %d: %w", i, err)
			}
		}

		return nil, nil
	case []*chcol.Variant:
		for i, vt := range vv {
			err := c.AppendRow(vt)
			if err != nil {
				return nil, fmt.Errorf("failed to AppendRow at index %d: %w", i, err)
			}
		}

		return nil, nil
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return nil, &ColumnConverterError{
					Op:   "Append",
					To:   string(c.chType),
					From: fmt.Sprintf("%T", v),
					Hint: "could not get driver.Valuer value",
				}
			}

			return c.Append(val)
		}

		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   string(c.chType),
			From: fmt.Sprintf("%T", v),
		}
	}
}

func (c *Variant) AppendRow(v any) error {
	var requestedType string
	switch vv := v.(type) {
	case nil:
		c.appendNullRow()
		return nil
	case chcol.Variant:
		requestedType = vv.Type()
		v = vv.Any()
		if vv.Nil() {
			c.appendNullRow()
			return nil
		}
	case *chcol.Variant:
		requestedType = vv.Type()
		v = vv.Any()
		if vv.Nil() {
			c.appendNullRow()
			return nil
		}
	}

	if requestedType != "" {
		typeIndex, ok := c.columnTypeIndex[requestedType]
		if !ok {
			return fmt.Errorf("value %v cannot be stored in variant column %s with requested type %s: type not present in variant", v, c.chType, requestedType)
		}

		if err := c.columns[typeIndex].AppendRow(v); err != nil {
			return fmt.Errorf("failed to append row to variant column with requested type %s: %w", requestedType, err)
		}

		c.appendDiscriminatorRow(typeIndex)
		return nil
	}

	// If preferred type wasn't provided, try each column
	var err error
	for i, col := range c.columns {
		if err = col.AppendRow(v); err == nil {
			c.appendDiscriminatorRow(uint8(i))
			return nil
		}
	}

	return fmt.Errorf("value \"%v\" cannot be stored in variant column: no compatible types", v)
}

func (c *Variant) WriteStatePrefix(buffer *proto.Buffer) error {
	buffer.PutUInt64(SupportedVariantSerializationVersion)

	for _, col := range c.columns {
		if serialize, ok := col.(CustomSerialization); ok {
			if err := serialize.WriteStatePrefix(buffer); err != nil {
				return fmt.Errorf("failed to write prefix for type %s in variant: %w", col.Type(), err)
			}
		}
	}

	return nil
}

func (c *Variant) Encode(buffer *proto.Buffer) {
	buffer.PutRaw(c.discriminators)

	for _, col := range c.columns {
		col.Encode(buffer)
	}
}

func (c *Variant) ScanType() reflect.Type {
	return scanTypeVariant
}

func (c *Variant) Reset() {
	c.discriminators = c.discriminators[:0]

	for _, col := range c.columns {
		col.Reset()
	}
}

func (c *Variant) ReadStatePrefix(reader *proto.Reader) error {
	variantSerializationVersion, err := reader.UInt64()
	if err != nil {
		return fmt.Errorf("failed to read variant discriminator version: %w", err)
	}

	if variantSerializationVersion != SupportedVariantSerializationVersion {
		return fmt.Errorf("unsupported variant discriminator version: %d", variantSerializationVersion)
	}

	for _, col := range c.columns {
		if serialize, ok := col.(CustomSerialization); ok {
			if err := serialize.ReadStatePrefix(reader); err != nil {
				return fmt.Errorf("failed to read prefix for type %s in variant: %w", col.Type(), err)
			}
		}
	}

	return nil
}

func (c *Variant) Decode(reader *proto.Reader, rows int) error {
	c.discriminators = make([]uint8, rows)
	c.offsets = make([]int, rows)
	rowCountByType := make(map[uint8]int, len(c.columns))

	for i := 0; i < rows; i++ {
		disc, err := reader.ReadByte()
		if err != nil {
			return fmt.Errorf("failed to read discriminator at index %d: %w", i, err)
		}

		c.discriminators[i] = disc
		if rowCountByType[disc] == 0 {
			rowCountByType[disc] = 1
		} else {
			rowCountByType[disc]++
		}

		c.offsets[i] = rowCountByType[disc] - 1
	}

	for i, col := range c.columns {
		cRows := rowCountByType[uint8(i)]
		if err := col.Decode(reader, cRows); err != nil {
			return fmt.Errorf("failed to decode variant column with %s type: %w", col.Type(), err)
		}
	}

	return nil
}
