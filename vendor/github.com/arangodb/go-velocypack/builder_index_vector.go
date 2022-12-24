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

const (
	minIndexVectorGrowDelta = 32
	maxIndexVectorGrowDelta = 1024
)

// indexVector is a list of index of positions.
type indexVector []ValueLength

// Add an index position to the end of the list.
func (iv *indexVector) Add(v ValueLength) {
	*iv = append(*iv, v)
}

// RemoveLast removes the last index position from the end of the list.
func (iv *indexVector) RemoveLast() {
	l := len(*iv)
	if l > 0 {
		*iv = (*iv)[:l-1]
	}
}

// Clear removes all entries
func (iv *indexVector) Clear() {
	if len(*iv) > 0 {
		*iv = (*iv)[0:0]
	}
}

// IsEmpty returns true if there are no values on the vector.
func (iv indexVector) IsEmpty() bool {
	l := len(iv)
	return l == 0
}
