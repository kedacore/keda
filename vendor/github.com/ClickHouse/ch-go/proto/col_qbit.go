package proto

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/go-faster/errors"
)

// ColQBit represents QBit(T, N) column where T is BFloat16/Float32/Float64.
// QBit stores vectors in a bit-sliced format where each bit plane is stored
// as a separate FixedString column. This enables runtime precision tuning
// for vector similarity searches.
//
// Internally, QBit is represented as a Tuple of FixedString columns where:
//   - Number of columns = bit width of element type (16/32/64)
//   - Each FixedString length = dimension (stores one bit from each element)
//   - Column i stores the i-th bit from all elements of all vectors
type ColQBit struct {
	// elementType can be BFloat16, Float32, or Float64
	elementType ColumnType

	// dimension is the Vector dimension (length of the single vector)
	// in QBit(T, N), N represents the dimension
	dimension int

	// bitWidth is the number of bits per element (16, 32, or 64). Depending on ColumnType
	bitWidth int

	// bytesPerRow represents bytes needed to store dimension bits = ceil(dimension/8)
	// example: QBit(Float32, 100)
	// Here
	bytesPerRow int

	// bitPlanes are how internally QBit columns are stored.
	// [bitWidth][bytesPerRow * rows]
	bitPlanes [][]byte

	// rows is number of vector rows in QBit column.
	rows int
}

// Make ColQBit always satisfies required Column related interfaces.
var (
	_ Column    = (*ColQBit)(nil)
	_ ColInput  = (*ColQBit)(nil)
	_ ColResult = (*ColQBit)(nil)
)

// NewColQBit creates a new QBit column with the specified element type and dimension.
func NewColQBit(elementType ColumnType, dimension int) (*ColQBit, error) {
	bitWidth, err := qbitBitWidth(elementType)
	if err != nil {
		return nil, err
	}
	bitsPerByte := 8

	// ceil(dimension / 8) without needing floating point.
	// e.g: dimension = 9
	// (9 + 8 - 1)/8 = 2 (bytes)
	bytesPerRow := (dimension + bitsPerByte - 1) / bitsPerByte
	bitPlanes := make([][]byte, bitWidth)
	for i := range bitPlanes {
		bitPlanes[i] = make([]byte, 0)
	}

	return &ColQBit{
		elementType: elementType,
		dimension:   dimension,
		bitWidth:    bitWidth,
		bytesPerRow: bytesPerRow,
		bitPlanes:   bitPlanes,
		rows:        0,
	}, nil
}

// qbitBitWidth returns the bit width for the given element type.
func qbitBitWidth(elementType ColumnType) (int, error) {
	switch elementType {
	case ColumnTypeBFloat16:
		return 16, nil
	case ColumnTypeFloat32:
		return 32, nil
	case ColumnTypeFloat64:
		return 64, nil
	default:
		return 0, fmt.Errorf("unsupported QBit element type: %s", elementType)
	}
}

// ParseQBitType parses a QBit type string like "QBit(Float32, 1024)".
func ParseQBitType(t ColumnType) (elementType ColumnType, dimension int, err error) {
	base := t.Base()
	if base != "QBit" {
		return "", 0, fmt.Errorf("not a QBit type: %s", t)
	}

	elem := string(t.Elem())
	parts := strings.Split(elem, ",")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid QBit format (expected 2 parameters): %s", t)
	}

	elementTypeStr := strings.TrimSpace(parts[0])
	dimensionStr := strings.TrimSpace(parts[1])

	elementType = ColumnType(elementTypeStr)
	if elementType != ColumnTypeBFloat16 && elementType != ColumnTypeFloat32 && elementType != ColumnTypeFloat64 {
		return "", 0, fmt.Errorf("invalid QBit element type: %s", elementType)
	}

	dimension, err = strconv.Atoi(dimensionStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid QBit dimension: %s", err)
	}
	if dimension <= 0 {
		return "", 0, fmt.Errorf("QBit dimension must be positive: %d", dimension)
	}

	return elementType, dimension, nil
}

func (c *ColQBit) Type() ColumnType {
	return ColumnType(fmt.Sprintf("QBit(%s, %d)", c.elementType, c.dimension))
}

func (c *ColQBit) Rows() int {
	return c.rows
}

func (c *ColQBit) Reset() {
	for i := range c.bitPlanes {
		c.bitPlanes[i] = c.bitPlanes[i][:0]
	}
	c.rows = 0
}

// Row reconstructs a vector from the bit-plane representation.
func (c *ColQBit) Row(i int) []float32 {
	if i < 0 || i >= c.rows {
		return nil
	}

	result := make([]float32, c.dimension)

	// Reconstruct each element from its bits across all bit planes
	for elemIdx := range c.dimension {
		byteIdx := elemIdx / 8
		bitIdx := uint(elemIdx % 8)

		var bits uint64
		// Collect bits from all bit planes (MSB first)
		for planeIdx := range c.bitWidth {
			offset := i*c.bytesPerRow + byteIdx
			if offset < len(c.bitPlanes[planeIdx]) {
				bit := (c.bitPlanes[planeIdx][offset] >> bitIdx) & 1
				bits |= uint64(bit) << uint(c.bitWidth-1-planeIdx)
			}
		}

		// Convert bits to float based on element type
		switch c.elementType {
		case ColumnTypeBFloat16:
			result[elemIdx] = BFloat16ToFloat32(uint16(bits))
		case ColumnTypeFloat32:
			result[elemIdx] = math.Float32frombits(uint32(bits))
		case ColumnTypeFloat64:
			f64 := math.Float64frombits(bits)
			result[elemIdx] = float32(f64)
		}
	}

	return result
}

// Append adds a vector to the QBit column.
func (c *ColQBit) Append(vector []float32) error {
	if len(vector) != c.dimension {
		return fmt.Errorf("vector dimension mismatch: expected %d, got %d", c.dimension, len(vector))
	}

	// Convert vector elements to bit representation
	bits := make([]uint64, c.dimension)
	for i, v := range vector {
		switch c.elementType {
		case ColumnTypeBFloat16:
			bits[i] = uint64(Float32ToBFloat16(v))
		case ColumnTypeFloat32:
			bits[i] = uint64(math.Float32bits(v))
		case ColumnTypeFloat64:
			bits[i] = math.Float64bits(float64(v))
		}
	}

	// Transpose bits to bit planes
	for planeIdx := range c.bitWidth {
		// Allocate space for this row in the bit plane
		startLen := len(c.bitPlanes[planeIdx])
		c.bitPlanes[planeIdx] = append(c.bitPlanes[planeIdx], make([]byte, c.bytesPerRow)...)

		// Extract the planeIdx-th bit from each element (MSB first)
		bitMask := uint64(1) << uint(c.bitWidth-1-planeIdx)
		for elemIdx := range c.dimension {
			if (bits[elemIdx] & bitMask) != 0 {
				byteIdx := elemIdx / 8
				bitIdx := uint(elemIdx % 8)
				c.bitPlanes[planeIdx][startLen+byteIdx] |= 1 << bitIdx
			}
		}
	}

	c.rows++
	return nil
}

// DecodeColumn decodes QBit column from reader.
// QBit is internally stored as a Tuple of FixedString columns.
func (c *ColQBit) DecodeColumn(r *Reader, rows int) error {
	if rows == 0 {
		return nil
	}

	// Read each bit plane as a FixedString column
	for planeIdx := range c.bitWidth {
		// Read rows * bytesPerRow bytes for this bit plane
		data, err := r.ReadRaw(rows * c.bytesPerRow)
		if err != nil {
			return errors.Wrapf(err, "decode bit plane %d", planeIdx)
		}
		c.bitPlanes[planeIdx] = append(c.bitPlanes[planeIdx], data...)
	}

	c.rows += rows
	return nil
}

// EncodeColumn encodes QBit column to buffer.
// QBit is internally stored as a Tuple of FixedString columns.
func (c *ColQBit) EncodeColumn(b *Buffer) {
	// Write each bit plane as a FixedString column
	for planeIdx := range c.bitWidth {
		b.Buf = append(b.Buf, c.bitPlanes[planeIdx]...)
	}
}

// WriteColumn encodes the column data and chains it to w for later writing.
func (c *ColQBit) WriteColumn(w *Writer) {
	w.ChainBuffer(c.EncodeColumn)
}

// Infer implements the Inferable interface for QBit columns.
func (c *ColQBit) Infer(t ColumnType) error {
	elementType, dimension, err := ParseQBitType(t)
	if err != nil {
		return err
	}

	bitWidth, err := qbitBitWidth(elementType)
	if err != nil {
		return err
	}

	c.elementType = elementType
	c.dimension = dimension
	c.bitWidth = bitWidth
	c.bytesPerRow = (dimension + 7) / 8

	// Initialize bit planes
	c.bitPlanes = make([][]byte, bitWidth)
	for i := range c.bitPlanes {
		c.bitPlanes[i] = make([]byte, 0)
	}
	c.rows = 0

	return nil
}
