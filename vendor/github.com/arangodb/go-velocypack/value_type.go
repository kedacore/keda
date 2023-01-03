//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
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

type ValueType int

const (
	None    ValueType = iota // not yet initialized
	Illegal                  // illegal value
	Null                     // JSON null
	Bool
	Array
	Object
	Double
	UTCDate
	External
	MinKey
	MaxKey
	Int
	UInt
	SmallInt
	String
	Binary
	BCD
	Custom
)

// String returns a string representation of the given type.
func (vt ValueType) String() string {
	return typeNames[vt]
}

var typeNames = [...]string{
	"None",
	"Illegal",
	"Null",
	"Bool",
	"Array",
	"Object",
	"Double",
	"UTCDate",
	"External",
	"MinKey",
	"MaxKey",
	"Int",
	"UInt",
	"SmallInt",
	"String",
	"Binary",
	"BCD",
	"Custom",
}

var typeMap = [256]ValueType{
	/* 0x00 */ None /* 0x01 */, Array,
	/* 0x02 */ Array /* 0x03 */, Array,
	/* 0x04 */ Array /* 0x05 */, Array,
	/* 0x06 */ Array /* 0x07 */, Array,
	/* 0x08 */ Array /* 0x09 */, Array,
	/* 0x0a */ Object /* 0x0b */, Object,
	/* 0x0c */ Object /* 0x0d */, Object,
	/* 0x0e */ Object /* 0x0f */, Object,
	/* 0x10 */ Object /* 0x11 */, Object,
	/* 0x12 */ Object /* 0x13 */, Array,
	/* 0x14 */ Object /* 0x15 */, None,
	/* 0x16 */ None /* 0x17 */, Illegal,
	/* 0x18 */ Null /* 0x19 */, Bool,
	/* 0x1a */ Bool /* 0x1b */, Double,
	/* 0x1c */ UTCDate /* 0x1d */, External,
	/* 0x1e */ MinKey /* 0x1f */, MaxKey,
	/* 0x20 */ Int /* 0x21 */, Int,
	/* 0x22 */ Int /* 0x23 */, Int,
	/* 0x24 */ Int /* 0x25 */, Int,
	/* 0x26 */ Int /* 0x27 */, Int,
	/* 0x28 */ UInt /* 0x29 */, UInt,
	/* 0x2a */ UInt /* 0x2b */, UInt,
	/* 0x2c */ UInt /* 0x2d */, UInt,
	/* 0x2e */ UInt /* 0x2f */, UInt,
	/* 0x30 */ SmallInt /* 0x31 */, SmallInt,
	/* 0x32 */ SmallInt /* 0x33 */, SmallInt,
	/* 0x34 */ SmallInt /* 0x35 */, SmallInt,
	/* 0x36 */ SmallInt /* 0x37 */, SmallInt,
	/* 0x38 */ SmallInt /* 0x39 */, SmallInt,
	/* 0x3a */ SmallInt /* 0x3b */, SmallInt,
	/* 0x3c */ SmallInt /* 0x3d */, SmallInt,
	/* 0x3e */ SmallInt /* 0x3f */, SmallInt,
	/* 0x40 */ String /* 0x41 */, String,
	/* 0x42 */ String /* 0x43 */, String,
	/* 0x44 */ String /* 0x45 */, String,
	/* 0x46 */ String /* 0x47 */, String,
	/* 0x48 */ String /* 0x49 */, String,
	/* 0x4a */ String /* 0x4b */, String,
	/* 0x4c */ String /* 0x4d */, String,
	/* 0x4e */ String /* 0x4f */, String,
	/* 0x50 */ String /* 0x51 */, String,
	/* 0x52 */ String /* 0x53 */, String,
	/* 0x54 */ String /* 0x55 */, String,
	/* 0x56 */ String /* 0x57 */, String,
	/* 0x58 */ String /* 0x59 */, String,
	/* 0x5a */ String /* 0x5b */, String,
	/* 0x5c */ String /* 0x5d */, String,
	/* 0x5e */ String /* 0x5f */, String,
	/* 0x60 */ String /* 0x61 */, String,
	/* 0x62 */ String /* 0x63 */, String,
	/* 0x64 */ String /* 0x65 */, String,
	/* 0x66 */ String /* 0x67 */, String,
	/* 0x68 */ String /* 0x69 */, String,
	/* 0x6a */ String /* 0x6b */, String,
	/* 0x6c */ String /* 0x6d */, String,
	/* 0x6e */ String /* 0x6f */, String,
	/* 0x70 */ String /* 0x71 */, String,
	/* 0x72 */ String /* 0x73 */, String,
	/* 0x74 */ String /* 0x75 */, String,
	/* 0x76 */ String /* 0x77 */, String,
	/* 0x78 */ String /* 0x79 */, String,
	/* 0x7a */ String /* 0x7b */, String,
	/* 0x7c */ String /* 0x7d */, String,
	/* 0x7e */ String /* 0x7f */, String,
	/* 0x80 */ String /* 0x81 */, String,
	/* 0x82 */ String /* 0x83 */, String,
	/* 0x84 */ String /* 0x85 */, String,
	/* 0x86 */ String /* 0x87 */, String,
	/* 0x88 */ String /* 0x89 */, String,
	/* 0x8a */ String /* 0x8b */, String,
	/* 0x8c */ String /* 0x8d */, String,
	/* 0x8e */ String /* 0x8f */, String,
	/* 0x90 */ String /* 0x91 */, String,
	/* 0x92 */ String /* 0x93 */, String,
	/* 0x94 */ String /* 0x95 */, String,
	/* 0x96 */ String /* 0x97 */, String,
	/* 0x98 */ String /* 0x99 */, String,
	/* 0x9a */ String /* 0x9b */, String,
	/* 0x9c */ String /* 0x9d */, String,
	/* 0x9e */ String /* 0x9f */, String,
	/* 0xa0 */ String /* 0xa1 */, String,
	/* 0xa2 */ String /* 0xa3 */, String,
	/* 0xa4 */ String /* 0xa5 */, String,
	/* 0xa6 */ String /* 0xa7 */, String,
	/* 0xa8 */ String /* 0xa9 */, String,
	/* 0xaa */ String /* 0xab */, String,
	/* 0xac */ String /* 0xad */, String,
	/* 0xae */ String /* 0xaf */, String,
	/* 0xb0 */ String /* 0xb1 */, String,
	/* 0xb2 */ String /* 0xb3 */, String,
	/* 0xb4 */ String /* 0xb5 */, String,
	/* 0xb6 */ String /* 0xb7 */, String,
	/* 0xb8 */ String /* 0xb9 */, String,
	/* 0xba */ String /* 0xbb */, String,
	/* 0xbc */ String /* 0xbd */, String,
	/* 0xbe */ String /* 0xbf */, String,
	/* 0xc0 */ Binary /* 0xc1 */, Binary,
	/* 0xc2 */ Binary /* 0xc3 */, Binary,
	/* 0xc4 */ Binary /* 0xc5 */, Binary,
	/* 0xc6 */ Binary /* 0xc7 */, Binary,
	/* 0xc8 */ BCD /* 0xc9 */, BCD,
	/* 0xca */ BCD /* 0xcb */, BCD,
	/* 0xcc */ BCD /* 0xcd */, BCD,
	/* 0xce */ BCD /* 0xcf */, BCD,
	/* 0xd0 */ BCD /* 0xd1 */, BCD,
	/* 0xd2 */ BCD /* 0xd3 */, BCD,
	/* 0xd4 */ BCD /* 0xd5 */, BCD,
	/* 0xd6 */ BCD /* 0xd7 */, BCD,
	/* 0xd8 */ None /* 0xd9 */, None,
	/* 0xda */ None /* 0xdb */, None,
	/* 0xdc */ None /* 0xdd */, None,
	/* 0xde */ None /* 0xdf */, None,
	/* 0xe0 */ None /* 0xe1 */, None,
	/* 0xe2 */ None /* 0xe3 */, None,
	/* 0xe4 */ None /* 0xe5 */, None,
	/* 0xe6 */ None /* 0xe7 */, None,
	/* 0xe8 */ None /* 0xe9 */, None,
	/* 0xea */ None /* 0xeb */, None,
	/* 0xec */ None /* 0xed */, None,
	/* 0xee */ None /* 0xef */, None,
	/* 0xf0 */ Custom /* 0xf1 */, Custom,
	/* 0xf2 */ Custom /* 0xf3 */, Custom,
	/* 0xf4 */ Custom /* 0xf5 */, Custom,
	/* 0xf6 */ Custom /* 0xf7 */, Custom,
	/* 0xf8 */ Custom /* 0xf9 */, Custom,
	/* 0xfa */ Custom /* 0xfb */, Custom,
	/* 0xfc */ Custom /* 0xfd */, Custom,
	/* 0xfe */ Custom /* 0xff */, Custom}

const (
	doubleLength  = 8
	int64Length   = 8
	charPtrLength = 8
)

var fixedTypeLengths = [256]int{
	/* 0x00 */ 1 /* 0x01 */, 1,
	/* 0x02 */ 0 /* 0x03 */, 0,
	/* 0x04 */ 0 /* 0x05 */, 0,
	/* 0x06 */ 0 /* 0x07 */, 0,
	/* 0x08 */ 0 /* 0x09 */, 0,
	/* 0x0a */ 1 /* 0x0b */, 0,
	/* 0x0c */ 0 /* 0x0d */, 0,
	/* 0x0e */ 0 /* 0x0f */, 0,
	/* 0x10 */ 0 /* 0x11 */, 0,
	/* 0x12 */ 0 /* 0x13 */, 0,
	/* 0x14 */ 0 /* 0x15 */, 0,
	/* 0x16 */ 0 /* 0x17 */, 1,
	/* 0x18 */ 1 /* 0x19 */, 1,
	/* 0x1a */ 1 /* 0x1b */, 1 + doubleLength, /*sizeof(double)*/
	/* 0x1c */ 1 + int64Length /*sizeof(int64_t)*/ /* 0x1d */, 1 + charPtrLength, /* sizeof(char*)*/
	/* 0x1e */ 1 /* 0x1f */, 1,
	/* 0x20 */ 2 /* 0x21 */, 3,
	/* 0x22 */ 4 /* 0x23 */, 5,
	/* 0x24 */ 6 /* 0x25 */, 7,
	/* 0x26 */ 8 /* 0x27 */, 9,
	/* 0x28 */ 2 /* 0x29 */, 3,
	/* 0x2a */ 4 /* 0x2b */, 5,
	/* 0x2c */ 6 /* 0x2d */, 7,
	/* 0x2e */ 8 /* 0x2f */, 9,
	/* 0x30 */ 1 /* 0x31 */, 1,
	/* 0x32 */ 1 /* 0x33 */, 1,
	/* 0x34 */ 1 /* 0x35 */, 1,
	/* 0x36 */ 1 /* 0x37 */, 1,
	/* 0x38 */ 1 /* 0x39 */, 1,
	/* 0x3a */ 1 /* 0x3b */, 1,
	/* 0x3c */ 1 /* 0x3d */, 1,
	/* 0x3e */ 1 /* 0x3f */, 1,
	/* 0x40 */ 1 /* 0x41 */, 2,
	/* 0x42 */ 3 /* 0x43 */, 4,
	/* 0x44 */ 5 /* 0x45 */, 6,
	/* 0x46 */ 7 /* 0x47 */, 8,
	/* 0x48 */ 9 /* 0x49 */, 10,
	/* 0x4a */ 11 /* 0x4b */, 12,
	/* 0x4c */ 13 /* 0x4d */, 14,
	/* 0x4e */ 15 /* 0x4f */, 16,
	/* 0x50 */ 17 /* 0x51 */, 18,
	/* 0x52 */ 19 /* 0x53 */, 20,
	/* 0x54 */ 21 /* 0x55 */, 22,
	/* 0x56 */ 23 /* 0x57 */, 24,
	/* 0x58 */ 25 /* 0x59 */, 26,
	/* 0x5a */ 27 /* 0x5b */, 28,
	/* 0x5c */ 29 /* 0x5d */, 30,
	/* 0x5e */ 31 /* 0x5f */, 32,
	/* 0x60 */ 33 /* 0x61 */, 34,
	/* 0x62 */ 35 /* 0x63 */, 36,
	/* 0x64 */ 37 /* 0x65 */, 38,
	/* 0x66 */ 39 /* 0x67 */, 40,
	/* 0x68 */ 41 /* 0x69 */, 42,
	/* 0x6a */ 43 /* 0x6b */, 44,
	/* 0x6c */ 45 /* 0x6d */, 46,
	/* 0x6e */ 47 /* 0x6f */, 48,
	/* 0x70 */ 49 /* 0x71 */, 50,
	/* 0x72 */ 51 /* 0x73 */, 52,
	/* 0x74 */ 53 /* 0x75 */, 54,
	/* 0x76 */ 55 /* 0x77 */, 56,
	/* 0x78 */ 57 /* 0x79 */, 58,
	/* 0x7a */ 59 /* 0x7b */, 60,
	/* 0x7c */ 61 /* 0x7d */, 62,
	/* 0x7e */ 63 /* 0x7f */, 64,
	/* 0x80 */ 65 /* 0x81 */, 66,
	/* 0x82 */ 67 /* 0x83 */, 68,
	/* 0x84 */ 69 /* 0x85 */, 70,
	/* 0x86 */ 71 /* 0x87 */, 72,
	/* 0x88 */ 73 /* 0x89 */, 74,
	/* 0x8a */ 75 /* 0x8b */, 76,
	/* 0x8c */ 77 /* 0x8d */, 78,
	/* 0x8e */ 79 /* 0x8f */, 80,
	/* 0x90 */ 81 /* 0x91 */, 82,
	/* 0x92 */ 83 /* 0x93 */, 84,
	/* 0x94 */ 85 /* 0x95 */, 86,
	/* 0x96 */ 87 /* 0x97 */, 88,
	/* 0x98 */ 89 /* 0x99 */, 90,
	/* 0x9a */ 91 /* 0x9b */, 92,
	/* 0x9c */ 93 /* 0x9d */, 94,
	/* 0x9e */ 95 /* 0x9f */, 96,
	/* 0xa0 */ 97 /* 0xa1 */, 98,
	/* 0xa2 */ 99 /* 0xa3 */, 100,
	/* 0xa4 */ 101 /* 0xa5 */, 102,
	/* 0xa6 */ 103 /* 0xa7 */, 104,
	/* 0xa8 */ 105 /* 0xa9 */, 106,
	/* 0xaa */ 107 /* 0xab */, 108,
	/* 0xac */ 109 /* 0xad */, 110,
	/* 0xae */ 111 /* 0xaf */, 112,
	/* 0xb0 */ 113 /* 0xb1 */, 114,
	/* 0xb2 */ 115 /* 0xb3 */, 116,
	/* 0xb4 */ 117 /* 0xb5 */, 118,
	/* 0xb6 */ 119 /* 0xb7 */, 120,
	/* 0xb8 */ 121 /* 0xb9 */, 122,
	/* 0xba */ 123 /* 0xbb */, 124,
	/* 0xbc */ 125 /* 0xbd */, 126,
	/* 0xbe */ 127 /* 0xbf */, 0,
	/* 0xc0 */ 0 /* 0xc1 */, 0,
	/* 0xc2 */ 0 /* 0xc3 */, 0,
	/* 0xc4 */ 0 /* 0xc5 */, 0,
	/* 0xc6 */ 0 /* 0xc7 */, 0,
	/* 0xc8 */ 0 /* 0xc9 */, 0,
	/* 0xca */ 0 /* 0xcb */, 0,
	/* 0xcc */ 0 /* 0xcd */, 0,
	/* 0xce */ 0 /* 0xcf */, 0,
	/* 0xd0 */ 0 /* 0xd1 */, 0,
	/* 0xd2 */ 0 /* 0xd3 */, 0,
	/* 0xd4 */ 0 /* 0xd5 */, 0,
	/* 0xd6 */ 0 /* 0xd7 */, 0,
	/* 0xd8 */ 0 /* 0xd9 */, 0,
	/* 0xda */ 0 /* 0xdb */, 0,
	/* 0xdc */ 0 /* 0xdd */, 0,
	/* 0xde */ 0 /* 0xdf */, 0,
	/* 0xe0 */ 0 /* 0xe1 */, 0,
	/* 0xe2 */ 0 /* 0xe3 */, 0,
	/* 0xe4 */ 0 /* 0xe5 */, 0,
	/* 0xe6 */ 0 /* 0xe7 */, 0,
	/* 0xe8 */ 0 /* 0xe9 */, 0,
	/* 0xea */ 0 /* 0xeb */, 0,
	/* 0xec */ 0 /* 0xed */, 0,
	/* 0xee */ 0 /* 0xef */, 0,
	/* 0xf0 */ 2 /* 0xf1 */, 3,
	/* 0xf2 */ 5 /* 0xf3 */, 9,
	/* 0xf4 */ 0 /* 0xf5 */, 0,
	/* 0xf6 */ 0 /* 0xf7 */, 0,
	/* 0xf8 */ 0 /* 0xf9 */, 0,
	/* 0xfa */ 0 /* 0xfb */, 0,
	/* 0xfc */ 0 /* 0xfd */, 0,
	/* 0xfe */ 0 /* 0xff */, 0}

var widthMap = [32]uint{
	0, // 0x00, None
	1, // 0x01, empty array
	1, // 0x02, array without index table
	2, // 0x03, array without index table
	4, // 0x04, array without index table
	8, // 0x05, array without index table
	1, // 0x06, array with index table
	2, // 0x07, array with index table
	4, // 0x08, array with index table
	8, // 0x09, array with index table
	1, // 0x0a, empty object
	1, // 0x0b, object with sorted index table
	2, // 0x0c, object with sorted index table
	4, // 0x0d, object with sorted index table
	8, // 0x0e, object with sorted index table
	1, // 0x0f, object with unsorted index table
	2, // 0x10, object with unsorted index table
	4, // 0x11, object with unsorted index table
	8, // 0x12, object with unsorted index table
	0}

var firstSubMap = [32]int{
	0, // 0x00, None
	1, // 0x01, empty array
	2, // 0x02, array without index table
	3, // 0x03, array without index table
	5, // 0x04, array without index table
	9, // 0x05, array without index table
	3, // 0x06, array with index table
	5, // 0x07, array with index table
	9, // 0x08, array with index table
	9, // 0x09, array with index table
	1, // 0x0a, empty object
	3, // 0x0b, object with sorted index table
	5, // 0x0c, object with sorted index table
	9, // 0x0d, object with sorted index table
	9, // 0x0e, object with sorted index table
	3, // 0x0f, object with unsorted index table
	5, // 0x10, object with unsorted index table
	9, // 0x11, object with unsorted index table
	9, // 0x12, object with unsorted index table
	0}
