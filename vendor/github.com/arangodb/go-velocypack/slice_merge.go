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

// Merge creates a slice that contains all fields from all given slices.
// When a field exists (with same name) in an earlier slice, it is ignored.
// All slices must be objects.
func Merge(slices ...Slice) (Slice, error) {
	// Calculate overall length
	l := ValueLength(0)
	for _, s := range slices {
		if err := s.AssertType(Object); err != nil {
			return nil, WithStack(err)
		}
		byteSize, err := s.ByteSize()
		if err != nil {
			return nil, WithStack(err)
		}
		l += byteSize
	}

	if len(slices) == 1 {
		// Fast path, only 1 slice
		return slices[0], nil
	}

	// Create a buffer to hold all slices.
	b := NewBuilder(uint(l))
	keys := make(map[string]struct{})
	if err := b.OpenObject(); err != nil {
		return nil, WithStack(err)
	}
	for _, s := range slices {
		it, err := NewObjectIterator(s, true)
		if err != nil {
			return nil, WithStack(err)
		}
		for it.IsValid() {
			keySlice, err := it.Key(true)
			if err != nil {
				return nil, WithStack(err)
			}
			key, err := keySlice.GetString()
			if err != nil {
				return nil, WithStack(err)
			}
			if _, found := keys[key]; !found {
				// Record key
				keys[key] = struct{}{}

				// Fetch value
				value, err := it.Value()
				if err != nil {
					return nil, WithStack(err)
				}

				// Add key,value
				if err := b.addInternalKeyValue(key, NewSliceValue(value)); err != nil {
					return nil, WithStack(err)
				}
			}

			// Move to next field
			if err := it.Next(); err != nil {
				return nil, WithStack(err)
			}
		}
	}
	if err := b.Close(); err != nil {
		return nil, WithStack(err)
	}

	// Return slice
	result, err := b.Slice()
	if err != nil {
		return nil, WithStack(err)
	}
	return result, nil
}
