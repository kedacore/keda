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

import (
	"fmt"
	"io"
	"strconv"
)

type DumperOptions struct {
	// EscapeUnicode turns on escapping multi-byte Unicode characters when dumping them to JSON (creates \uxxxx sequences).
	EscapeUnicode bool
	// EscapeForwardSlashes turns on escapping forward slashes when serializing VPack values into JSON.
	EscapeForwardSlashes    bool
	UnsupportedTypeBehavior UnsupportedTypeBehavior
}

type UnsupportedTypeBehavior int

const (
	NullifyUnsupportedType UnsupportedTypeBehavior = iota
	ConvertUnsupportedType
	FailOnUnsupportedType
)

type Dumper struct {
	w           io.Writer
	indentation uint
	options     DumperOptions
}

// NewDumper creates a new dumper around the given writer, with an optional options.
func NewDumper(w io.Writer, options *DumperOptions) *Dumper {
	d := &Dumper{
		w: w,
	}
	if options != nil {
		d.options = *options
	}
	return d
}

func (d *Dumper) Append(s Slice) error {
	w := d.w
	switch s.Type() {
	case Null:
		if _, err := w.Write([]byte("null")); err != nil {
			return WithStack(err)
		}
		return nil
	case Bool:
		if v, err := s.GetBool(); err != nil {
			return WithStack(err)
		} else if v {
			if _, err := w.Write([]byte("true")); err != nil {
				return WithStack(err)
			}
		} else {
			if _, err := w.Write([]byte("false")); err != nil {
				return WithStack(err)
			}
		}
		return nil
	case Double:
		if v, err := s.GetDouble(); err != nil {
			return WithStack(err)
		} else if err := d.appendDouble(v); err != nil {
			return WithStack(err)
		}
		return nil
	case Int, SmallInt:
		if v, err := s.GetInt(); err != nil {
			return WithStack(err)
		} else if err := d.appendInt(v); err != nil {
			return WithStack(err)
		}
		return nil
	case UInt:
		if v, err := s.GetUInt(); err != nil {
			return WithStack(err)
		} else if err := d.appendUInt(v); err != nil {
			return WithStack(err)
		}
		return nil
	case String:
		if v, err := s.GetString(); err != nil {
			return WithStack(err)
		} else if err := d.appendString(v); err != nil {
			return WithStack(err)
		}
		return nil
	case Array:
		if err := d.appendArray(s); err != nil {
			return WithStack(err)
		}
		return nil
	case Object:
		if err := d.appendObject(s); err != nil {
			return WithStack(err)
		}
		return nil
	default:
		switch d.options.UnsupportedTypeBehavior {
		case NullifyUnsupportedType:
			if _, err := w.Write([]byte("null")); err != nil {
				return WithStack(err)
			}
		case ConvertUnsupportedType:
			msg := fmt.Sprintf("(non-representable type %s)", s.Type().String())
			if err := d.appendString(msg); err != nil {
				return WithStack(err)
			}
		default:
			return WithStack(NoJSONEquivalentError)
		}
	}

	return nil
}

var (
	doubleQuoteSeq = []byte{'"'}
	escapeTable    = [256]byte{
		// 0    1    2    3    4    5    6    7    8    9    A    B    C    D    E
		// F
		'u', 'u', 'u', 'u', 'u', 'u', 'u', 'u', 'b', 't', 'n', 'u', 'f', 'r',
		'u',
		'u', // 00
		'u', 'u', 'u', 'u', 'u', 'u', 'u', 'u', 'u', 'u', 'u', 'u', 'u', 'u',
		'u',
		'u', // 10
		0, 0, '"', 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0,
		'/', // 20
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0,
		0, // 30~4F
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		'\\', 0, 0, 0, // 50
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0,
		0, // 60~FF
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0}
)

func (d *Dumper) appendUInt(v uint64) error {
	s := strconv.FormatUint(v, 10)
	if _, err := d.w.Write([]byte(s)); err != nil {
		return WithStack(err)
	}
	return nil
}

func (d *Dumper) appendInt(v int64) error {
	s := strconv.FormatInt(v, 10)
	if _, err := d.w.Write([]byte(s)); err != nil {
		return WithStack(err)
	}
	return nil
}

func formatDouble(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 64)
}

func (d *Dumper) appendDouble(v float64) error {
	s := formatDouble(v)
	if _, err := d.w.Write([]byte(s)); err != nil {
		return WithStack(err)
	}
	return nil
}

func (d *Dumper) appendString(v string) error {
	p := []byte(v)
	e := len(p)
	buf := make([]byte, 0, 16)
	if _, err := d.w.Write(doubleQuoteSeq); err != nil {
		return WithStack(err)
	}
	for i := 0; i < e; i++ {
		buf = buf[0:0]
		c := p[i]
		if (c & 0x80) == 0 {
			// check for control characters
			esc := escapeTable[c]

			if esc != 0 {
				if c != '/' || d.options.EscapeForwardSlashes {
					// escape forward slashes only when requested
					buf = append(buf, '\\')
				}
				buf = append(buf, esc)

				if esc == 'u' {
					i1 := ((uint(c)) & 0xf0) >> 4
					i2 := ((uint(c)) & 0x0f)

					buf = append(buf, '0', '0', hexChar(i1), hexChar(i2))
				}
			} else {
				buf = append(buf, c)
			}
		} else if (c & 0xe0) == 0xc0 {
			// two-byte sequence
			if i+1 >= e {
				return WithStack(InvalidUtf8SequenceError)
			}

			if d.options.EscapeUnicode {
				value := ((uint(p[i]) & 0x1f) << 6) | (uint(p[i+1]) & 0x3f)
				buf = dumpUnicodeCharacter(buf, value)
			} else {
				buf = append(buf, p[i:i+2]...)
			}
			i++
		} else if (c & 0xf0) == 0xe0 {
			// three-byte sequence
			if i+2 >= e {
				return WithStack(InvalidUtf8SequenceError)
			}

			if d.options.EscapeUnicode {
				value := (((uint(p[i]) & 0x0f) << 12) | ((uint(p[i+1]) & 0x3f) << 6) | (uint(p[i + +2]) & 0x3f))
				buf = dumpUnicodeCharacter(buf, value)
			} else {
				buf = append(buf, p[i:i+3]...)
			}
			i += 2
		} else if (c & 0xf8) == 0xf0 {
			// four-byte sequence
			if i+3 >= e {
				return WithStack(InvalidUtf8SequenceError)
			}

			if d.options.EscapeUnicode {
				value := (((uint(p[i]) & 0x0f) << 18) | ((uint(p[i+1]) & 0x3f) << 12) | ((uint(p[i+2]) & 0x3f) << 6) | (uint(p[i+3]) & 0x3f))
				// construct the surrogate pairs
				value -= 0x10000
				high := (((value & 0xffc00) >> 10) + 0xd800)
				buf = dumpUnicodeCharacter(buf, high)
				low := (value & 0x3ff) + 0xdc00
				buf = dumpUnicodeCharacter(buf, low)
			} else {
				buf = append(buf, p[i:i+4]...)
			}
			i += 3
		}
		if _, err := d.w.Write(buf); err != nil {
			return WithStack(err)
		}
	}
	if _, err := d.w.Write(doubleQuoteSeq); err != nil {
		return WithStack(err)
	}
	return nil
}

func (d *Dumper) appendArray(v Slice) error {
	w := d.w
	it, err := NewArrayIterator(v)
	if err != nil {
		return WithStack(err)
	}
	if _, err := w.Write([]byte{'['}); err != nil {
		return WithStack(err)
	}
	for it.IsValid() {
		if !it.IsFirst() {
			if _, err := w.Write([]byte{','}); err != nil {
				return WithStack(err)
			}
		}
		if value, err := it.Value(); err != nil {
			return WithStack(err)
		} else if err := d.Append(value); err != nil {
			return WithStack(err)
		}
		if err := it.Next(); err != nil {
			return WithStack(err)
		}
	}
	if _, err := w.Write([]byte{']'}); err != nil {
		return WithStack(err)
	}
	return nil
}

func (d *Dumper) appendObject(v Slice) error {
	w := d.w
	it, err := NewObjectIterator(v)
	if err != nil {
		return WithStack(err)
	}
	if _, err := w.Write([]byte{'{'}); err != nil {
		return WithStack(err)
	}
	for it.IsValid() {
		if !it.IsFirst() {
			if _, err := w.Write([]byte{','}); err != nil {
				return WithStack(err)
			}
		}
		if key, err := it.Key(true); err != nil {
			return WithStack(err)
		} else if err := d.Append(key); err != nil {
			return WithStack(err)
		}
		if _, err := w.Write([]byte{':'}); err != nil {
			return WithStack(err)
		}
		if value, err := it.Value(); err != nil {
			return WithStack(err)
		} else if err := d.Append(value); err != nil {
			return WithStack(err)
		}
		if err := it.Next(); err != nil {
			return WithStack(err)
		}
	}
	if _, err := w.Write([]byte{'}'}); err != nil {
		return WithStack(err)
	}
	return nil
}

func dumpUnicodeCharacter(dst []byte, value uint) []byte {
	dst = append(dst, '\\', 'u')

	mask := uint(0xf000)
	shift := uint(12)
	for i := 3; i >= 0; i-- {
		p := (value & mask) >> shift
		dst = append(dst, hexChar(p))
		if i > 0 {
			mask = mask >> 4
			shift -= 4
		}
	}
	return dst
}

func hexChar(v uint) byte {
	v = v & uint(0x0f)
	if v < 10 {
		return byte('0' + v)
	}
	return byte('A' + v - 10)
}
