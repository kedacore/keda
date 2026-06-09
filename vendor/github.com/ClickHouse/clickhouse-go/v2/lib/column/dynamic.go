package column

import (
	"database/sql/driver"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/ClickHouse/ch-go/proto"

	"github.com/ClickHouse/clickhouse-go/v2/lib/chcol"
)

const DynamicSerializationVersion = 3
const DynamicDeprecatedSerializationVersion = 1
const DynamicNullDiscriminator = -1 // The Null index changes as data is being built, use -1 as placeholder for writes.
const DefaultMaxDynamicTypes = 32

func supportsFlatDynamicJSON(sc *ServerContext) bool {
	// Any CH version more than 25.6
	return sc.VersionMajor > 25 || (sc.VersionMajor == 25 && sc.VersionMinor >= 6)
}

type Dynamic struct {
	chType Type
	sc     *ServerContext
	name   string

	serializationVersion uint64

	totalTypes     int // Null is last type index + 1, so this doubles as the Null type index for reads.
	discriminators []int
	offsets        []int

	columns           []Interface
	columnIndexByType map[string]int

	deprecated deprecatedDynamic
}

func (c *Dynamic) parse(t Type, sc *ServerContext) (_ *Dynamic, err error) {
	c.chType = t
	c.sc = sc
	tStr := string(t)

	c.columnIndexByType = make(map[string]int)

	if !supportsFlatDynamicJSON(sc) {
		// SharedVariant is special, and does not count against totalTypes
		sv, _ := Type("SharedVariant").Column("", sc)
		c.addColumn(sv)

		c.deprecated.maxTypes = DefaultMaxDynamicTypes
		c.totalTypes = 0 // Reset to 0 after adding SharedVariant
	}

	if tStr == "Dynamic" {
		return c, nil
	}

	if !strings.HasPrefix(tStr, "Dynamic(") || !strings.HasSuffix(tStr, ")") {
		return nil, &UnsupportedColumnTypeError{t: t}
	}

	if !supportsFlatDynamicJSON(sc) {
		typeParamsStr := strings.TrimPrefix(tStr, "Dynamic(")
		typeParamsStr = strings.TrimSuffix(typeParamsStr, ")")

		if strings.HasPrefix(typeParamsStr, "max_types=") {
			v := strings.TrimPrefix(typeParamsStr, "max_types=")
			if maxTypes, err := strconv.Atoi(v); err == nil {
				c.deprecated.maxTypes = maxTypes
			}
		}
	}

	return c, nil
}

func (c *Dynamic) addColumn(col Interface) int {
	typeName := string(col.Type())
	c.deprecated.typeNames = append(c.deprecated.typeNames, typeName)

	colIndex := len(c.deprecated.typeNames) - 1
	c.columns = append(c.columns, col)
	c.columnIndexByType[typeName] = colIndex
	c.totalTypes++

	return colIndex
}

func (c *Dynamic) Name() string {
	return c.name
}

func (c *Dynamic) Type() Type {
	return c.chType
}

func (c *Dynamic) Rows() int {
	return len(c.discriminators)
}

func (c *Dynamic) Row(i int, ptr bool) any {
	typeIndex := c.discriminators[i]
	offsetIndex := c.offsets[i]
	var value any
	var chType string
	if c.serializationVersion == DynamicDeprecatedSerializationVersion {
		if typeIndex != DynamicNullDiscriminator {
			value = c.columns[typeIndex].Row(offsetIndex, ptr)
			chType = string(c.columns[typeIndex].Type())
		}
	} else {
		if typeIndex != c.totalTypes {
			value = c.columns[typeIndex].Row(offsetIndex, ptr)
			chType = string(c.columns[typeIndex].Type())
		}
	}

	dyn := chcol.NewDynamicWithType(value, chType)
	if ptr {
		return &dyn
	}

	return dyn
}

func (c *Dynamic) ScanRow(dest any, row int) error {
	typeIndex := c.discriminators[row]
	offsetIndex := c.offsets[row]
	var value any
	var chType string
	if c.serializationVersion == DynamicDeprecatedSerializationVersion {
		if typeIndex != DynamicNullDiscriminator {
			value = c.columns[typeIndex].Row(offsetIndex, false)
			chType = string(c.columns[typeIndex].Type())
		}
	} else {
		if typeIndex != c.totalTypes {
			value = c.columns[typeIndex].Row(offsetIndex, false)
			chType = string(c.columns[typeIndex].Type())
		}
	}

	switch v := dest.(type) {
	case *chcol.Dynamic:
		dyn := chcol.NewDynamicWithType(value, chType)
		*v = dyn
	case **chcol.Dynamic:
		dyn := chcol.NewDynamicWithType(value, chType)
		**v = dyn
	default:
		if c.serializationVersion == DynamicDeprecatedSerializationVersion {
			if typeIndex == DynamicNullDiscriminator {
				return nil
			}
		} else {
			if typeIndex == c.totalTypes {
				return nil
			}
		}

		if err := c.columns[typeIndex].ScanRow(dest, offsetIndex); err != nil {
			return err
		}
	}

	return nil
}

func (c *Dynamic) appendDiscriminatorRow(d int) {
	c.discriminators = append(c.discriminators, d)
}

func (c *Dynamic) appendNullRow() {
	c.appendDiscriminatorRow(DynamicNullDiscriminator)
}

func (c *Dynamic) Append(v any) (nulls []uint8, err error) {
	switch vv := v.(type) {
	case []chcol.Dynamic:
		for i, dyn := range vv {
			err := c.AppendRow(dyn)
			if err != nil {
				return nil, fmt.Errorf("failed to AppendRow at index %d: %w", i, err)
			}
		}

		return nil, nil
	case []*chcol.Dynamic:
		for i, dyn := range vv {
			err := c.AppendRow(dyn)
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

func (c *Dynamic) AppendRow(v any) error {
	var requestedType string
	switch vv := v.(type) {
	case nil:
		c.appendNullRow()
		return nil
	case chcol.Dynamic:
		requestedType = vv.Type()
		v = vv.Any()
		if vv.Nil() {
			c.appendNullRow()
			return nil
		}
	case *chcol.Dynamic:
		requestedType = vv.Type()
		v = vv.Any()
		if vv.Nil() {
			c.appendNullRow()
			return nil
		}
	}

	if requestedType != "" {
		var col Interface
		colIndex, ok := c.columnIndexByType[requestedType]
		if ok {
			col = c.columns[colIndex]
		} else {
			newCol, err := Type(requestedType).Column("", c.sc)
			if err != nil {
				return fmt.Errorf("value \"%v\" cannot be stored in dynamic column %s with requested type %s: unable to append type: %w", v, c.chType, requestedType, err)
			}

			colIndex = c.addColumn(newCol)
			col = newCol
		}

		if err := col.AppendRow(v); err != nil {
			return fmt.Errorf("value \"%v\" cannot be stored in dynamic column %s with requested type %s: %w", v, c.chType, requestedType, err)
		}

		c.appendDiscriminatorRow(colIndex)
		return nil
	}

	// If preferred type wasn't provided, try each column
	for i, col := range c.columns {
		if c.deprecated.typeNames[i] == "SharedVariant" {
			// Do not try to fit into SharedVariant
			continue
		}

		if err := col.AppendRow(v); err == nil {
			c.appendDiscriminatorRow(i)
			return nil
		}
	}

	// If no existing columns match, try matching a ClickHouse type from common Go types
	inferredTypeName := inferClickHouseTypeFromGoType(v)
	if inferredTypeName != "" {
		return c.AppendRow(chcol.NewDynamicWithType(v, inferredTypeName))
	}

	return fmt.Errorf("value \"%v\" cannot be stored in dynamic column: no compatible types. hint: either use fixed types like int64, int32, etc or use clickhouse.DynamicWithType to wrap the value with concrete ClickHouse column type", v)
}

func (c *Dynamic) encodeHeader(buffer *proto.Buffer) error {
	buffer.PutUInt64(DynamicSerializationVersion)
	buffer.PutUVarInt(uint64(c.totalTypes))

	for _, col := range c.columns {
		buffer.PutString(string(col.Type()))
	}

	for _, col := range c.columns {
		if serialize, ok := col.(CustomSerialization); ok {
			if err := serialize.WriteStatePrefix(buffer); err != nil {
				return fmt.Errorf("failed to write prefix for type %s in dynamic: %w", string(col.Type()), err)
			}
		}
	}

	return nil
}

func discriminatorWriter(totalTypes uint64, buffer *proto.Buffer) func(uint64) {
	switch {
	case totalTypes <= math.MaxUint8:
		return func(d uint64) { buffer.PutUInt8(uint8(d)) }
	case totalTypes <= math.MaxUint16:
		return func(d uint64) { buffer.PutUInt16(uint16(d)) }
	case totalTypes <= math.MaxUint32:
		return func(d uint64) { buffer.PutUInt32(uint32(d)) }
	default:
		return func(d uint64) { buffer.PutUInt64(d) }
	}
}

func (c *Dynamic) encodeData(buffer *proto.Buffer) {
	writeDiscriminator := discriminatorWriter(uint64(c.totalTypes), buffer)
	for _, typeIndex := range c.discriminators {
		if typeIndex == DynamicNullDiscriminator {
			typeIndex = c.totalTypes
		}

		writeDiscriminator(uint64(typeIndex))
	}

	for _, col := range c.columns {
		col.Encode(buffer)
	}
}

func (c *Dynamic) WriteStatePrefix(buffer *proto.Buffer) error {
	if supportsFlatDynamicJSON(c.sc) {
		return c.encodeHeader(buffer)
	}

	return c.encodeHeader_v1(buffer)
}

func (c *Dynamic) Encode(buffer *proto.Buffer) {
	if supportsFlatDynamicJSON(c.sc) {
		c.encodeData(buffer)
		return
	}

	c.encodeData_v1(buffer)
}

func (c *Dynamic) ScanType() reflect.Type {
	return scanTypeDynamic
}

func (c *Dynamic) Reset() {
	c.discriminators = c.discriminators[:0]

	for _, col := range c.columns {
		col.Reset()
	}
}

func (c *Dynamic) decodeHeader(reader *proto.Reader) error {
	totalTypes, err := reader.UVarInt()
	if err != nil {
		return fmt.Errorf("failed to read total types for dynamic column: %w", err)
	}

	c.columns = make([]Interface, 0, totalTypes)
	c.columnIndexByType = make(map[string]int, totalTypes)
	for i := uint64(0); i < totalTypes; i++ {
		typeName, err := reader.Str()
		if err != nil {
			return fmt.Errorf("failed to read type name at index %d for dynamic column: %w", i, err)
		}

		col, err := Type(typeName).Column("", c.sc)
		if err != nil {
			return fmt.Errorf("failed to add dynamic column with type %s: %w", typeName, err)
		}

		c.addColumn(col)
	}

	for _, col := range c.columns {
		if serialize, ok := col.(CustomSerialization); ok {
			if err := serialize.ReadStatePrefix(reader); err != nil {
				return fmt.Errorf("failed to read prefix for type %s in dynamic: %w", col.Type(), err)
			}
		}
	}

	return nil
}

func discriminatorReader(totalTypes uint64, reader *proto.Reader) func() (uint64, error) {
	switch {
	case totalTypes <= math.MaxUint8:
		return func() (uint64, error) {
			v, err := reader.UInt8()
			return uint64(v), err
		}
	case totalTypes <= math.MaxUint16:
		return func() (uint64, error) {
			v, err := reader.UInt16()
			return uint64(v), err
		}
	case totalTypes <= math.MaxUint32:
		return func() (uint64, error) {
			v, err := reader.UInt32()
			return uint64(v), err
		}
	default:
		return func() (uint64, error) {
			return reader.UInt64()
		}
	}
}

func (c *Dynamic) decodeData(reader *proto.Reader, rows int) error {
	c.discriminators = make([]int, rows)
	c.offsets = make([]int, rows)
	rowCountByType := make([]int, c.totalTypes)

	readDiscriminator := discriminatorReader(uint64(c.totalTypes), reader)
	for i := 0; i < rows; i++ {
		disc, err := readDiscriminator()
		if err != nil {
			return fmt.Errorf("failed to read discriminator at index %d: %w", i, err)
		}

		c.discriminators[i] = int(disc)
		if int(disc) != c.totalTypes {
			c.offsets[i] = rowCountByType[disc]
			rowCountByType[disc]++
		}
	}

	for i, col := range c.columns {
		cRows := rowCountByType[i]
		if err := col.Decode(reader, cRows); err != nil {
			return fmt.Errorf("failed to decode dynamic column with %s type: %w", col.Type(), err)
		}
	}

	return nil
}

func (c *Dynamic) ReadStatePrefix(reader *proto.Reader) error {
	dynamicSerializationVersion, err := reader.UInt64()
	if err != nil {
		return fmt.Errorf("failed to read dynamic serialization version: %w", err)
	}
	c.serializationVersion = dynamicSerializationVersion

	switch c.serializationVersion {
	case DynamicSerializationVersion:
		return c.decodeHeader(reader)
	case DynamicDeprecatedSerializationVersion:
		return c.decodeHeader_v1(reader)
	default:
		return fmt.Errorf("unsupported dynamic serialization version: %d", dynamicSerializationVersion)
	}
}

func (c *Dynamic) Decode(reader *proto.Reader, rows int) error {
	switch c.serializationVersion {
	case DynamicSerializationVersion:
		return c.decodeData(reader, rows)
	case DynamicDeprecatedSerializationVersion:
		return c.decodeData_v1(reader, rows)
	default:
		return fmt.Errorf("unsupported dynamic serialization version: %d", c.serializationVersion)
	}
}
