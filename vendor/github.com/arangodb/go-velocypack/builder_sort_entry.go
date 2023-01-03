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
	"bytes"
	"sort"
)

type sortEntry struct {
	Offset ValueLength
	Name   []byte
}

type sortEntries []sortEntry

// Len is the number of elements in the collection.
func (l sortEntries) Len() int { return len(l) }

// Less reports whether the element with
// index i should sort before the element with index j.
func (l sortEntries) Less(i, j int) bool { return bytes.Compare(l[i].Name, l[j].Name) < 0 }

// Swap swaps the elements with indexes i and j.
func (l sortEntries) Swap(i, j int) { l[i], l[j] = l[j], l[i] }

// partition picks the last element as a pivot and reorders the array so that
// all elements with values less than the pivot come before the pivot and all
// elements with values greater than the pivot come after it.
func partition(s sortEntries) int {
	hi := len(s) - 1
	pivot := s[hi]
	i := 0
	for j := 0; j < hi; j++ {
		r := bytes.Compare(s[j].Name, pivot.Name)
		if r <= 0 {
			s[i], s[j] = s[j], s[i]
			i++
		}
	}
	s[i], s[hi] = s[hi], s[i]
	return i
}

// Sort sorts the slice in ascending order.
func (l sortEntries) qSort() {
	if len(l) > 1 {
		p := partition(l)
		l[:p].qSort()
		l[p+1:].qSort()
	}
}

// Sort sorts the slice in ascending order.
func (l sortEntries) Sort() {
	x := len(l)
	if x > 16 {
		sort.Sort(l)
	} else if len(l) > 1 {
		l.qSort()
	}
}
