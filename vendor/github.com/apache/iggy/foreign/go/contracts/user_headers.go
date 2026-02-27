// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package iggcon

import (
	"encoding/binary"
	"errors"
)

type HeaderValue struct {
	Kind  HeaderKind
	Value []byte
}

type HeaderKey struct {
	Value string
}

func NewHeaderKey(val string) (HeaderKey, error) {
	if len(val) == 0 || len(val) > 255 {
		return HeaderKey{}, errors.New("value has incorrect size, must be between 1 and 255")
	}
	return HeaderKey{Value: val}, nil
}

type HeaderKind int

const (
	Raw     HeaderKind = 1
	String  HeaderKind = 2
	Bool    HeaderKind = 3
	Int8    HeaderKind = 4
	Int16   HeaderKind = 5
	Int32   HeaderKind = 6
	Int64   HeaderKind = 7
	Int128  HeaderKind = 8
	Uint8   HeaderKind = 9
	Uint16  HeaderKind = 10
	Uint32  HeaderKind = 11
	Uint64  HeaderKind = 12
	Uint128 HeaderKind = 13
	Float   HeaderKind = 14
	Double  HeaderKind = 15
)

func GetHeadersBytes(headers map[HeaderKey]HeaderValue) []byte {
	headersLength := 0
	for key, header := range headers {
		headersLength += 4 + len(key.Value) + 1 + 4 + len(header.Value)
	}
	headersBytes := make([]byte, headersLength)
	position := 0
	for key, value := range headers {
		headerBytes := getBytesFromHeader(key, value)
		copy(headersBytes[position:position+len(headerBytes)], headerBytes)
		position += len(headerBytes)
	}
	return headersBytes
}

func getBytesFromHeader(key HeaderKey, value HeaderValue) []byte {
	headerBytesLength := 4 + len(key.Value) + 1 + 4 + len(value.Value)
	headerBytes := make([]byte, headerBytesLength)

	binary.LittleEndian.PutUint32(headerBytes[:4], uint32(len(key.Value)))
	copy(headerBytes[4:4+len(key.Value)], key.Value)

	headerBytes[4+len(key.Value)] = byte(value.Kind)

	binary.LittleEndian.PutUint32(headerBytes[4+len(key.Value)+1:4+len(key.Value)+1+4], uint32(len(value.Value)))
	copy(headerBytes[4+len(key.Value)+1+4:], value.Value)

	return headerBytes
}

func DeserializeHeaders(userHeadersBytes []byte) (map[HeaderKey]HeaderValue, error) {
	headers := make(map[HeaderKey]HeaderValue)
	position := 0

	for position < len(userHeadersBytes) {
		if len(userHeadersBytes) <= position+4 {
			return nil, errors.New("invalid header key length")
		}

		keyLength := binary.LittleEndian.Uint32(userHeadersBytes[position : position+4])
		if keyLength == 0 || 255 < keyLength {
			return nil, errors.New("key has incorrect size, must be between 1 and 255")
		}
		position += 4

		if len(userHeadersBytes) < position+int(keyLength) {
			return nil, errors.New("invalid header key")
		}

		key := string(userHeadersBytes[position : position+int(keyLength)])
		position += int(keyLength)

		headerKind, err := deserializeHeaderKind(userHeadersBytes, position)
		if err != nil {
			return nil, err
		}
		position++

		if len(userHeadersBytes) <= position+4 {
			return nil, errors.New("invalid header value length")
		}

		valueLength := binary.LittleEndian.Uint32(userHeadersBytes[position : position+4])
		position += 4

		if valueLength == 0 || 255 < valueLength {
			return nil, errors.New("value has incorrect size, must be between 1 and 255")
		}

		if len(userHeadersBytes) < position+int(valueLength) {
			return nil, errors.New("invalid header value")
		}

		value := userHeadersBytes[position : position+int(valueLength)]
		position += int(valueLength)

		headers[HeaderKey{Value: key}] = HeaderValue{
			Kind:  headerKind,
			Value: value,
		}
	}

	return headers, nil
}

func deserializeHeaderKind(payload []byte, position int) (HeaderKind, error) {
	if position >= len(payload) {
		return 0, errors.New("invalid header kind position")
	}

	return HeaderKind(payload[position]), nil
}
