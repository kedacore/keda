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
	"bytes"
	"errors"
	"math"
	"strconv"

	"github.com/ClickHouse/ch-go/proto"
)

func Enum(chType Type, name string) (Interface, error) {
	enumType, values, indexes, valid := extractEnumNamedValues(chType)
	if !valid {
		return nil, &Error{
			ColumnType: string(chType),
			Err:        errors.New("invalid Enum"),
		}
	}

	if enumType == enum8Type {
		enum := Enum8{
			iv:     make(map[string]proto.Enum8, len(values)),
			vi:     make(map[proto.Enum8]string, len(values)),
			chType: chType,
			name:   name,
		}
		for i := range values {
			v := int8(indexes[i])
			enum.iv[values[i]] = proto.Enum8(v)
			enum.vi[proto.Enum8(v)] = values[i]
		}
		return &enum, nil
	}
	enum := Enum16{
		iv:     make(map[string]proto.Enum16, len(values)),
		vi:     make(map[proto.Enum16]string, len(values)),
		chType: chType,
		name:   name,
	}

	for i := range values {
		enum.iv[values[i]] = proto.Enum16(indexes[i])
		enum.vi[proto.Enum16(indexes[i])] = values[i]
	}
	return &enum, nil
}

const (
	enum8Type = "Enum8"
	E
	enum16Type = "Enum16"
)

func extractEnumNamedValues(chType Type) (typ string, values []string, indexes []int, valid bool) {
	src := []byte(chType)

	var bracketOpen, stringOpen bool

	var foundValueOffset int
	var foundValueLen int
	var skippedValueTokens []int
	var indexFound bool
	var valueFound bool
	var valueIndex = 0

	for c := 0; c < len(src); c++ {
		token := src[c]

		switch {
		// open bracket found, capture the type
		case token == '(' && !stringOpen:
			typ = string(src[:c])

			// Ignore everything captured as non-enum type
			if typ != enum8Type && typ != enum16Type {
				return
			}

			bracketOpen = true
			break
		// when inside a bracket, we can start capture value inside single quotes
		case bracketOpen && token == '\'' && !stringOpen:
			foundValueOffset = c + 1
			stringOpen = true
			break
		// close the string and capture the value
		case token == '\'' && stringOpen:
			stringOpen = false
			foundValueLen = c - foundValueOffset
			valueFound = true
			break
		// escape character, skip the next character
		case token == '\\' && stringOpen:
			skippedValueTokens = append(skippedValueTokens, c-foundValueOffset)
			c++
			break
		// capture optional index. `=` token is followed with an integer index
		case token == '=' && !stringOpen:
			if !valueFound {
				return
			}
			indexStart := c + 1
			// find the end of the index, it's either a comma or a closing bracket
			for _, token := range src[indexStart:] {
				if token == ',' || token == ')' {
					break
				}
				c++
			}

			idx, err := strconv.Atoi(string(bytes.TrimSpace(src[indexStart : c+1])))
			if err != nil {
				return
			}
			valueIndex = idx
			indexFound = true
			break
		// capture the value and index when a comma or closing bracket is found
		case (token == ',' || token == ')') && !stringOpen:
			if !valueFound {
				return
			}
			// if no index was found for current value, increment the value index
			// e.g. Enum8('a','b') is equivalent to Enum8('a'=1,'b'=2)
			// or Enum8('a'=3,'b') is equivalent to Enum8('a'=3,'b'=4)
			// so if no index is provided, we increment the value index
			if !indexFound {
				valueIndex++
			}

			// if the index is out of range, return
			if (typ == enum8Type && valueIndex > math.MaxUint8) ||
				(typ == enum16Type && valueIndex > math.MaxUint16) {
				return
			}

			foundName := src[foundValueOffset : foundValueOffset+foundValueLen]
			for _, skipped := range skippedValueTokens {
				foundName = append(foundName[:skipped], foundName[skipped+1:]...)
			}

			indexes = append(indexes, valueIndex)
			values = append(values, string(foundName))
			indexFound = false
			valueFound = false
			break
		}
	}

	// Enum type must have at least one value
	if valueIndex == 0 {
		return
	}

	valid = true
	return
}
