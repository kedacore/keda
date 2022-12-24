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

package driver

type Protocol int

const (
	ProtocolHTTP Protocol = iota
	ProtocolVST1_0
	ProtocolVST1_1
)

// ProtocolSet is a set of protocols.
type ProtocolSet []Protocol

// Contains returns true if the given protocol is contained in the given set, false otherwise.
func (ps ProtocolSet) Contains(p Protocol) bool {
	for _, x := range ps {
		if x == p {
			return true
		}
	}
	return false
}

// ContainsAny returns true if any of the given protocols is contained in the given set, false otherwise.
func (ps ProtocolSet) ContainsAny(p ...Protocol) bool {
	for _, x := range ps {
		for _, y := range p {
			if x == y {
				return true
			}
		}
	}
	return false
}
