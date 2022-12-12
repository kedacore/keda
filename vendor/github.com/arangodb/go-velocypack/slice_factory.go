//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License}
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//
// Author Ewout Prangsma
//

package velocypack

import "encoding/binary"

// NoneSlice creates a slice of type None
func NoneSlice() Slice { return Slice{0x00} }

// IllegalSlice creates a slice of type Illegal
func IllegalSlice() Slice { return Slice{0x17} }

// NullSlice creates a slice of type Null
func NullSlice() Slice { return Slice{0x18} }

// FalseSlice creates a slice of type Boolean with false value
func FalseSlice() Slice { return Slice{0x19} }

// TrueSlice creates a slice of type Boolean with true value
func TrueSlice() Slice { return Slice{0x1a} }

// ZeroSlice creates a slice of type Smallint(0)
func ZeroSlice() Slice { return Slice{0x30} }

// EmptyArraySlice creates a slice of type Array, empty
func EmptyArraySlice() Slice { return Slice{0x01} }

// EmptyObjectSlice creates a slice of type Object, empty
func EmptyObjectSlice() Slice { return Slice{0x0a} }

// MinKeySlice creates a slice of type MinKey
func MinKeySlice() Slice { return Slice{0x1e} }

// MaxKeySlice creates a slice of type MaxKey
func MaxKeySlice() Slice { return Slice{0x1f} }

// StringSlice creates a slice of type String with given string value
func StringSlice(s string) Slice {
	raw := []byte(s)
	l := len(raw)
	if l <= 126 {
		return Slice(append([]byte{byte(0x40 + l)}, raw...))
	}
	buf := make([]byte, 1+8+l)
	buf[0] = 0xbf
	binary.LittleEndian.PutUint64(buf[1:], uint64(l))
	copy(buf[1+8:], raw)
	return buf
}
