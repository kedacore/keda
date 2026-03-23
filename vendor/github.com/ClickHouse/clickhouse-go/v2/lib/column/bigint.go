// Licensed to ClickHouse, Inc. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. ClickHouse, Inc. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package column

import (
	"database/sql/driver"
	"encoding/binary"
	"fmt"
	"github.com/ClickHouse/ch-go/proto"
	"math/big"
	"reflect"
)

type BigInt struct {
	size   int
	chType Type
	name   string
	signed bool
	col    proto.Column
}

func (col *BigInt) Reset() {
	col.col.Reset()
}

func (col *BigInt) Name() string {
	return col.name
}

func (col *BigInt) Type() Type {
	return col.chType
}

func (col *BigInt) ScanType() reflect.Type {
	return scanTypeBigInt
}

func (col *BigInt) Rows() int {
	return col.col.Rows()
}

func (col *BigInt) Row(i int, ptr bool) any {
	value := col.row(i)
	if ptr {
		return value
	}
	return *value
}

func (col *BigInt) ScanRow(dest any, row int) error {
	switch d := dest.(type) {
	case *big.Int:
		*d = *col.row(row)
	case **big.Int:
		*d = new(big.Int)
		**d = *col.row(row)
	default:
		return &ColumnConverterError{
			Op:   "ScanRow",
			To:   fmt.Sprintf("%T", dest),
			From: string(col.chType),
		}
	}
	return nil
}

func (col *BigInt) Append(v any) (nulls []uint8, err error) {
	switch v := v.(type) {
	case []big.Int:
		nulls = make([]uint8, len(v))
		for i := range v {
			col.append(&v[i])
		}
	case []*big.Int:
		nulls = make([]uint8, len(v))
		for i := range v {
			switch {
			case v[i] != nil:
				col.append(v[i])
			default:
				nulls[i] = 1
				col.append(big.NewInt(0))
			}
		}
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
		}
	}
	return
}

func (col *BigInt) AppendRow(v any) error {
	switch v := v.(type) {
	case big.Int:
		col.append(&v)
	case *big.Int:
		switch {
		case v != nil:
			col.append(v)
		default:
			col.append(big.NewInt(0))
		}
	case nil:
		col.append(big.NewInt(0))
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
		}
	}
	return nil
}

func (col *BigInt) Decode(reader *proto.Reader, rows int) error {
	return col.col.DecodeColumn(reader, rows)
}

func (col *BigInt) Encode(buffer *proto.Buffer) {
	col.col.EncodeColumn(buffer)
}

func (col *BigInt) row(i int) *big.Int {
	b := make([]byte, col.size)
	switch vCol := col.col.(type) {
	case *proto.ColInt128:
		v := vCol.Row(i)
		binary.LittleEndian.PutUint64(b[0:64/8], v.Low)
		binary.LittleEndian.PutUint64(b[64/8:128/8], v.High)
		return rawToBigInt(b, true)
	case *proto.ColUInt128:
		v := vCol.Row(i)
		binary.LittleEndian.PutUint64(b[0:64/8], v.Low)
		binary.LittleEndian.PutUint64(b[64/8:128/8], v.High)
		return rawToBigInt(b, false)
	case *proto.ColInt256:
		v := vCol.Row(i)
		binary.LittleEndian.PutUint64(b[0:64/8], v.Low.Low)
		binary.LittleEndian.PutUint64(b[64/8:128/8], v.Low.High)
		binary.LittleEndian.PutUint64(b[128/8:192/8], v.High.Low)
		binary.LittleEndian.PutUint64(b[192/8:256/8], v.High.High)
		return rawToBigInt(b, true)
	case *proto.ColUInt256:
		v := vCol.Row(i)
		binary.LittleEndian.PutUint64(b[0:64/8], v.Low.Low)
		binary.LittleEndian.PutUint64(b[64/8:128/8], v.Low.High)
		binary.LittleEndian.PutUint64(b[128/8:192/8], v.High.Low)
		binary.LittleEndian.PutUint64(b[192/8:256/8], v.High.High)
		return rawToBigInt(b, false)
	}
	return big.NewInt(0)
}

func (col *BigInt) append(v *big.Int) {
	dest := make([]byte, col.size)
	bigIntToRaw(dest, new(big.Int).Set(v))
	switch v := col.col.(type) {
	case *proto.ColInt128:
		v.Append(proto.Int128{
			Low:  binary.LittleEndian.Uint64(dest[0 : 64/8]),
			High: binary.LittleEndian.Uint64(dest[64/8 : 128/8]),
		})
	case *proto.ColUInt128:
		v.Append(proto.UInt128{
			Low:  binary.LittleEndian.Uint64(dest[0 : 64/8]),
			High: binary.LittleEndian.Uint64(dest[64/8 : 128/8]),
		})
	case *proto.ColInt256:
		v.Append(proto.Int256{
			Low: proto.UInt128{
				Low:  binary.LittleEndian.Uint64(dest[0 : 64/8]),
				High: binary.LittleEndian.Uint64(dest[64/8 : 128/8]),
			},
			High: proto.UInt128{
				Low:  binary.LittleEndian.Uint64(dest[128/8 : 192/8]),
				High: binary.LittleEndian.Uint64(dest[192/8 : 256/8]),
			},
		})
	case *proto.ColUInt256:
		v.Append(proto.UInt256{
			Low: proto.UInt128{
				Low:  binary.LittleEndian.Uint64(dest[0 : 64/8]),
				High: binary.LittleEndian.Uint64(dest[64/8 : 128/8]),
			},
			High: proto.UInt128{
				Low:  binary.LittleEndian.Uint64(dest[128/8 : 192/8]),
				High: binary.LittleEndian.Uint64(dest[192/8 : 256/8]),
			},
		})
	}
}

func bigIntToRaw(dest []byte, v *big.Int) {
	var sign int
	if v.Sign() < 0 {
		v.Not(v).FillBytes(dest)
		sign = -1
	} else {
		v.FillBytes(dest)
	}
	endianSwap(dest, sign < 0)
}

func rawToBigInt(v []byte, signed bool) *big.Int {
	// LittleEndian to BigEndian
	endianSwap(v, false)
	var lt = new(big.Int)
	if signed && len(v) > 0 && v[0]&0x80 != 0 {
		// [0] ^ will +1
		for i := 0; i < len(v); i++ {
			v[i] = ^v[i]
		}
		lt.SetBytes(v)
		// neg ^ will -1
		lt.Not(lt)
	} else {
		lt.SetBytes(v)
	}
	return lt
}

func endianSwap(src []byte, not bool) {
	for i := 0; i < len(src)/2; i++ {
		if not {
			src[i], src[len(src)-i-1] = ^src[len(src)-i-1], ^src[i]
		} else {
			src[i], src[len(src)-i-1] = src[len(src)-i-1], src[i]
		}
	}
}

var _ Interface = (*BigInt)(nil)
