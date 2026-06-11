package column

import (
	"fmt"
	"slices"

	"github.com/ClickHouse/ch-go/proto"
)

type deprecatedDynamic struct {
	maxTypes  int
	typeNames []string
}

func (c *Dynamic) sortColumnsForEncoding() {
	previousTypeNames := make([]string, 0, len(c.deprecated.typeNames))
	previousTypeNames = append(previousTypeNames, c.deprecated.typeNames...)
	slices.Sort(c.deprecated.typeNames)

	for i, typeName := range c.deprecated.typeNames {
		c.columnIndexByType[typeName] = i
	}

	sortedDiscriminatorMap := make([]int, len(c.columns))
	sortedColumns := make([]Interface, len(c.columns))
	for i, typeName := range previousTypeNames {
		correctIndex := c.columnIndexByType[typeName]

		sortedDiscriminatorMap[i] = correctIndex
		sortedColumns[correctIndex] = c.columns[i]
	}
	c.columns = sortedColumns

	for i := range c.discriminators {
		if c.discriminators[i] == DynamicNullDiscriminator {
			continue
		}

		c.discriminators[i] = sortedDiscriminatorMap[c.discriminators[i]]
	}
}

func (c *Dynamic) encodeHeader_v1(buffer *proto.Buffer) error {
	c.sortColumnsForEncoding()

	buffer.PutUInt64(DynamicDeprecatedSerializationVersion)
	buffer.PutUVarInt(uint64(c.deprecated.maxTypes))
	buffer.PutUVarInt(uint64(c.totalTypes))

	for _, typeName := range c.deprecated.typeNames {
		if typeName == "SharedVariant" {
			// SharedVariant is implicitly present in Dynamic, do not append to type names
			continue
		}

		buffer.PutString(typeName)
	}

	buffer.PutUInt64(SupportedVariantSerializationVersion)

	for _, col := range c.columns {
		if serialize, ok := col.(CustomSerialization); ok {
			if err := serialize.WriteStatePrefix(buffer); err != nil {
				return fmt.Errorf("failed to write prefix for type %s in dynamic: %w", col.Type(), err)
			}
		}
	}

	return nil
}

func (c *Dynamic) encodeData_v1(buffer *proto.Buffer) {
	for _, disc := range c.discriminators {
		if disc == DynamicNullDiscriminator {
			disc = int(VariantNullDiscriminator)
		}

		buffer.PutUInt8(uint8(disc))
	}

	for _, col := range c.columns {
		col.Encode(buffer)
	}
}

func (c *Dynamic) decodeHeader_v1(reader *proto.Reader) error {
	maxTypes, err := reader.UVarInt()
	if err != nil {
		return fmt.Errorf("failed to read max types for dynamic column: %w", err)
	}
	c.deprecated.maxTypes = int(maxTypes)

	totalTypes, err := reader.UVarInt()
	if err != nil {
		return fmt.Errorf("failed to read total types for dynamic column: %w", err)
	}

	sortedTypeNames := make([]string, 0, totalTypes+1)
	for i := uint64(0); i < totalTypes; i++ {
		typeName, err := reader.Str()
		if err != nil {
			return fmt.Errorf("failed to read type name at index %d for dynamic column: %w", i, err)
		}

		sortedTypeNames = append(sortedTypeNames, typeName)
	}

	sortedTypeNames = append(sortedTypeNames, "SharedVariant")
	slices.Sort(sortedTypeNames) // Re-sort after adding SharedVariant

	c.deprecated.typeNames = make([]string, 0, len(sortedTypeNames))
	c.columns = make([]Interface, 0, len(sortedTypeNames))
	c.columnIndexByType = make(map[string]int, len(sortedTypeNames))

	for _, typeName := range sortedTypeNames {
		col, err := Type(typeName).Column("", c.sc)
		if err != nil {
			return fmt.Errorf("failed to add dynamic column with type %s: %w", typeName, err)
		}

		c.addColumn(col)
	}

	c.totalTypes = int(totalTypes) // Reset to server's totalTypes

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
				return fmt.Errorf("failed to read prefix for type %s in dynamic: %w", col.Type(), err)
			}
		}
	}

	return nil
}

func (c *Dynamic) decodeData_v1(reader *proto.Reader, rows int) error {
	c.discriminators = make([]int, rows)
	c.offsets = make([]int, rows)
	rowCountByType := make(map[int]int, len(c.columns))

	for i := 0; i < rows; i++ {
		discByte, err := reader.ReadByte()
		if err != nil {
			return fmt.Errorf("failed to read discriminator at index %d: %w", i, err)
		}

		disc := int(discByte)
		if disc == int(VariantNullDiscriminator) {
			disc = DynamicNullDiscriminator
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
		cRows := rowCountByType[i]
		if err := col.Decode(reader, cRows); err != nil {
			return fmt.Errorf("failed to decode dynamic column with %s type: %w", col.Type(), err)
		}
	}

	return nil
}
