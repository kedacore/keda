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

// This code is (mostly) taken for the Go sources.
// See https://golang.org/src/encoding/json/
//
// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package velocypack

import (
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
)

// A field represents a single field found in a struct.
type field struct {
	name      string
	nameBytes []byte                 // []byte(name)
	equalFold func(s, t []byte) bool // bytes.EqualFold or equivalent

	tag       bool
	index     []int
	typ       reflect.Type
	omitEmpty bool
	quoted    bool
}

func fieldByIndex(v reflect.Value, index []int) reflect.Value {
	for _, i := range index {
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return reflect.Value{}
			}
			v = v.Elem()
		}
		v = v.Field(i)
	}
	return v
}

func typeByIndex(t reflect.Type, index []int) reflect.Type {
	for _, i := range index {
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		t = t.Field(i).Type
	}
	return t
}

func fillField(f field) field {
	f.nameBytes = []byte(f.name)
	f.equalFold = foldFunc(f.nameBytes)
	return f
}

// byIndex sorts field by index sequence.
type byIndex []field

func (x byIndex) Len() int { return len(x) }

func (x byIndex) Swap(i, j int) { x[i], x[j] = x[j], x[i] }

func (x byIndex) Less(i, j int) bool {
	for k, xik := range x[i].index {
		if k >= len(x[j].index) {
			return false
		}
		if xik != x[j].index[k] {
			return xik < x[j].index[k]
		}
	}
	return len(x[i].index) < len(x[j].index)
}

// sort field by name, breaking ties with depth, then
// breaking ties with "name came from json tag", then
// breaking ties with index sequence.
type byNameIndexlenTag []field

func (x byNameIndexlenTag) Len() int { return len(x) }

func (x byNameIndexlenTag) Swap(i, j int) { x[i], x[j] = x[j], x[i] }

func (x byNameIndexlenTag) Less(i, j int) bool {
	if x[i].name != x[j].name {
		return x[i].name < x[j].name
	}
	if len(x[i].index) != len(x[j].index) {
		return len(x[i].index) < len(x[j].index)
	}
	if x[i].tag != x[j].tag {
		return x[i].tag
	}
	return byIndex(x).Less(i, j)
}

// typeFields returns a list of fields that JSON should recognize for the given type.
// The algorithm is breadth-first search over the set of structs to include - the top struct
// and then any reachable anonymous structs.
func typeFields(t reflect.Type) []field {
	// Anonymous fields to explore at the current level and the next.
	current := []field{}
	next := []field{{typ: t}}

	// Count of queued names for current level and the next.
	count := map[reflect.Type]int{}
	nextCount := map[reflect.Type]int{}

	// Types already visited at an earlier level.
	visited := map[reflect.Type]bool{}

	// Fields found.
	var fields []field

	for len(next) > 0 {
		current, next = next, current[:0]
		count, nextCount = nextCount, map[reflect.Type]int{}

		for _, f := range current {
			if visited[f.typ] {
				continue
			}
			visited[f.typ] = true

			// Scan f.typ for fields to include.
			for i := 0; i < f.typ.NumField(); i++ {
				sf := f.typ.Field(i)
				if sf.PkgPath != "" && !sf.Anonymous { // unexported
					continue
				}

				tag := sf.Tag.Get("velocypack")
				if len(tag) == 0 {
					tag = sf.Tag.Get("json")
				}

				if tag == "-" {
					continue
				}

				name, opts := parseTag(tag)
				if !isValidTag(name) {
					name = ""
				}
				index := make([]int, len(f.index)+1)
				copy(index, f.index)
				index[len(f.index)] = i

				ft := sf.Type
				if ft.Name() == "" && ft.Kind() == reflect.Ptr {
					// Follow pointer.
					ft = ft.Elem()
				}

				// Only strings, floats, integers, and booleans can be quoted.
				quoted := false
				if opts.Contains("string") {
					switch ft.Kind() {
					case reflect.Bool,
						reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
						reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
						reflect.Float32, reflect.Float64,
						reflect.String:
						quoted = true
					}
				}

				// Record found field and index sequence.
				if name != "" || !sf.Anonymous || ft.Kind() != reflect.Struct {
					tagged := name != ""
					if name == "" {
						name = sf.Name
					}
					fields = append(fields, fillField(field{
						name:      name,
						tag:       tagged,
						index:     index,
						typ:       ft,
						omitEmpty: opts.Contains("omitempty"),
						quoted:    quoted,
					}))
					if count[f.typ] > 1 {
						// If there were multiple instances, add a second,
						// so that the annihilation code will see a duplicate.
						// It only cares about the distinction between 1 or 2,
						// so don't bother generating any more copies.
						fields = append(fields, fields[len(fields)-1])
					}
					continue
				}

				// Record new anonymous struct to explore in next round.
				nextCount[ft]++
				if nextCount[ft] == 1 {
					next = append(next, fillField(field{name: ft.Name(), index: index, typ: ft}))
				}
			}
		}
	}

	sort.Sort(byNameIndexlenTag(fields))

	// Delete all fields that are hidden by the Go rules for embedded fields,
	// except that fields with JSON tags are promoted.

	// The fields are sorted in primary order of name, secondary order
	// of field index length. Loop over names; for each name, delete
	// hidden fields by choosing the one dominant field that survives.
	out := fields[:0]
	for advance, i := 0, 0; i < len(fields); i += advance {
		// One iteration per name.
		// Find the sequence of fields with the name of this first field.
		fi := fields[i]
		name := fi.name
		for advance = 1; i+advance < len(fields); advance++ {
			fj := fields[i+advance]
			if fj.name != name {
				break
			}
		}
		if advance == 1 { // Only one field with this name
			out = append(out, fi)
			continue
		}
		dominant, ok := dominantField(fields[i : i+advance])
		if ok {
			out = append(out, dominant)
		}
	}

	fields = out
	sort.Sort(byIndex(fields))

	return fields
}

// dominantField looks through the fields, all of which are known to
// have the same name, to find the single field that dominates the
// others using Go's embedding rules, modified by the presence of
// JSON tags. If there are multiple top-level fields, the boolean
// will be false: This condition is an error in Go and we skip all
// the fields.
func dominantField(fields []field) (field, bool) {
	// The fields are sorted in increasing index-length order. The winner
	// must therefore be one with the shortest index length. Drop all
	// longer entries, which is easy: just truncate the slice.
	length := len(fields[0].index)
	tagged := -1 // Index of first tagged field.
	for i, f := range fields {
		if len(f.index) > length {
			fields = fields[:i]
			break
		}
		if f.tag {
			if tagged >= 0 {
				// Multiple tagged fields at the same level: conflict.
				// Return no field.
				return field{}, false
			}
			tagged = i
		}
	}
	if tagged >= 0 {
		return fields[tagged], true
	}
	// All remaining fields have the same length. If there's more than one,
	// we have a conflict (two fields named "X" at the same level) and we
	// return no field.
	if len(fields) > 1 {
		return field{}, false
	}
	return fields[0], true
}

var fieldCache struct {
	value atomic.Value // map[reflect.Type][]field
	mu    sync.Mutex   // used only by writers
}

// cachedTypeFields is like typeFields but uses a cache to avoid repeated work.
func cachedTypeFields(t reflect.Type) []field {
	m, _ := fieldCache.value.Load().(map[reflect.Type][]field)
	f := m[t]
	if f != nil {
		return f
	}

	// Compute fields without lock.
	// Might duplicate effort but won't hold other computations back.
	f = typeFields(t)
	if f == nil {
		f = []field{}
	}

	fieldCache.mu.Lock()
	m, _ = fieldCache.value.Load().(map[reflect.Type][]field)
	newM := make(map[reflect.Type][]field, len(m)+1)
	for k, v := range m {
		newM[k] = v
	}
	newM[t] = f
	fieldCache.value.Store(newM)
	fieldCache.mu.Unlock()
	return f
}
