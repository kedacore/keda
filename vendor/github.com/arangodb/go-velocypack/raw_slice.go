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

import "errors"

// RawSlice is a raw encoded Velocypack value.
// It implements Marshaler and Unmarshaler and can
// be used to delay Velocypack decoding or precompute a Velocypack encoding.
type RawSlice []byte

// MarshalVPack returns m as the Velocypack encoding of m.
func (m RawSlice) MarshalVPack() (Slice, error) {
	if m == nil {
		return NullSlice(), nil
	}
	return Slice(m), nil
}

// UnmarshalVPack sets *m to a copy of data.
func (m *RawSlice) UnmarshalVPack(data Slice) error {
	if m == nil {
		return errors.New("velocypack.RawSlice: UnmarshalVPack on nil pointer")
	}
	*m = append((*m)[0:0], data...)
	return nil
}

var _ Marshaler = (*RawSlice)(nil)
var _ Unmarshaler = (*RawSlice)(nil)
