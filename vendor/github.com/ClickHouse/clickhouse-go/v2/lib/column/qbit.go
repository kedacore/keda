package column

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"

	"github.com/ClickHouse/ch-go/proto"
)

// QBit represents the QBit(T, N) column type for vector embeddings.
// QBit stores vectors in a bit-sliced format enabling runtime precision tuning
// for vector similarity searches.
//
// Supported element types: BFloat16, Float32, Float64
//
// Example usage:
//
//	vectors := [][]float32{
//	    {1.0, 2.0, 3.0, 4.0},
//	    {5.0, 6.0, 7.0, 8.0},
//	}
//	batch.Append(vectors)
type QBit struct {
	name        string
	chType      Type
	elementType string // "BFloat16", "Float32", or "Float64"
	dimension   int
	col         *proto.ColQBit
}

func (col *QBit) parse(t Type) (*QBit, error) {
	elementType, dimension, err := proto.ParseQBitType(proto.ColumnType(t))
	if err != nil {
		return nil, err
	}

	qbitCol, err := proto.NewColQBit(elementType, dimension)
	if err != nil {
		return nil, err
	}

	col.elementType = string(elementType)
	col.dimension = dimension
	col.chType = t
	col.col = qbitCol

	return col, nil
}

func (col *QBit) Name() string {
	return col.name
}

func (col *QBit) Type() Type {
	return col.chType
}

func (col *QBit) Reset() {
	col.col.Reset()
}

func (col *QBit) Rows() int {
	return col.col.Rows()
}

func (col *QBit) ScanType() reflect.Type {
	// Return slice of float32 slice (vector)
	return reflect.TypeOf([]float32{})
}

func (col *QBit) Row(i int, ptr bool) any {
	vec := col.row(i)
	if ptr {
		return &vec
	}
	return vec
}

func (col *QBit) row(i int) []float32 {
	return col.col.Row(i)
}

func (col *QBit) ScanRow(dest any, row int) error {
	vec := col.row(row)

	switch d := dest.(type) {
	case *[]float32:
		*d = vec
	case **[]float32:
		*d = new([]float32)
		**d = vec
	case *[]float64:
		// Convert float32 to float64
		*d = make([]float64, len(vec))
		for i, v := range vec {
			(*d)[i] = float64(v)
		}
	case **[]float64:
		*d = new([]float64)
		**d = make([]float64, len(vec))
		for i, v := range vec {
			(**d)[i] = float64(v)
		}
	case sql.Scanner:
		return d.Scan(vec)
	default:
		return &ColumnConverterError{
			Op:   "ScanRow",
			To:   fmt.Sprintf("%T", dest),
			From: string(col.chType),
			Hint: "QBit columns scan to []float32 or []float64",
		}
	}
	return nil
}

func (col *QBit) Append(v any) ([]uint8, error) {
	switch v := v.(type) {
	case [][]float32:
		for _, vec := range v {
			if err := col.col.Append(vec); err != nil {
				return nil, err
			}
		}
	case [][]float64:
		// Convert float64 to float32
		for _, vec := range v {
			f32vec := make([]float32, len(vec))
			for i, val := range vec {
				f32vec[i] = float32(val)
			}
			if err := col.col.Append(f32vec); err != nil {
				return nil, err
			}
		}
	case [][]*float32:
		// Support nullable vectors
		nulls := make([]uint8, len(v))
		for i, vec := range v {
			if vec == nil {
				nulls[i] = 1
				// Append zero vector
				zeroVec := make([]float32, col.dimension)
				if err := col.col.Append(zeroVec); err != nil {
					return nil, err
				}
			} else {
				// Dereference pointers
				derefVec := make([]float32, len(vec))
				for j, ptr := range vec {
					if ptr != nil {
						derefVec[j] = *ptr
					}
				}
				if err := col.col.Append(derefVec); err != nil {
					return nil, err
				}
			}
		}
		return nulls, nil
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return nil, &ColumnConverterError{
					Op:   "Append",
					To:   string(col.chType),
					From: fmt.Sprintf("%T", v),
					Hint: "could not get driver.Valuer value",
				}
			}
			return col.Append(val)
		}
		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   string(col.chType),
			From: fmt.Sprintf("%T", v),
			Hint: "QBit columns accept [][]float32 or [][]float64",
		}
	}
	return nil, nil
}

func (col *QBit) AppendRow(v any) error {
	switch v := v.(type) {
	case []float32:
		return col.col.Append(v)
	case []float64:
		// Convert float64 to float32
		f32vec := make([]float32, len(v))
		for i, val := range v {
			f32vec[i] = float32(val)
		}
		return col.col.Append(f32vec)
	case *[]float32:
		if v != nil {
			return col.col.Append(*v)
		}
		// Append zero vector for nil
		zeroVec := make([]float32, col.dimension)
		return col.col.Append(zeroVec)
	case *[]float64:
		if v != nil {
			f32vec := make([]float32, len(*v))
			for i, val := range *v {
				f32vec[i] = float32(val)
			}
			return col.col.Append(f32vec)
		}
		// Append zero vector for nil
		zeroVec := make([]float32, col.dimension)
		return col.col.Append(zeroVec)
	case []*float32:
		// Vector with potentially nil elements
		vec := make([]float32, len(v))
		for i, ptr := range v {
			if ptr != nil {
				vec[i] = *ptr
			}
		}
		return col.col.Append(vec)
	case []*float64:
		// Vector with potentially nil elements
		vec := make([]float32, len(v))
		for i, ptr := range v {
			if ptr != nil {
				vec[i] = float32(*ptr)
			}
		}
		return col.col.Append(vec)
	case nil:
		// Append zero vector for nil
		zeroVec := make([]float32, col.dimension)
		return col.col.Append(zeroVec)
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return &ColumnConverterError{
					Op:   "AppendRow",
					To:   string(col.chType),
					From: fmt.Sprintf("%T", v),
					Hint: "could not get driver.Valuer value",
				}
			}
			return col.AppendRow(val)
		}
		return &ColumnConverterError{
			Op:   "AppendRow",
			To:   string(col.chType),
			From: fmt.Sprintf("%T", v),
			Hint: "QBit columns accept []float32, []float64, or []*float32/64",
		}
	}
}

func (col *QBit) Decode(reader *proto.Reader, rows int) error {
	return col.col.DecodeColumn(reader, rows)
}

func (col *QBit) Encode(buffer *proto.Buffer) {
	col.col.EncodeColumn(buffer)
}

var _ Interface = (*QBit)(nil)
