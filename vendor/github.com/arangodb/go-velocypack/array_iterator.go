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

type ArrayIterator struct {
	s        Slice
	position ValueLength
	size     ValueLength
	current  Slice
}

// NewArrayIterator initializes an iterator at position 0 of the given object slice.
func NewArrayIterator(s Slice) (*ArrayIterator, error) {
	if !s.IsArray() {
		return nil, InvalidTypeError{"Expected Array slice"}
	}
	size, err := s.Length()
	if err != nil {
		return nil, WithStack(err)
	}
	i := &ArrayIterator{
		s:        s,
		position: 0,
		size:     size,
	}
	if size > 0 {
		i.current, err = s.At(0)
		if err != nil {
			return nil, WithStack(err)
		}
	}
	return i, nil
}

// IsValid returns true if the given position of the iterator is valid.
func (i *ArrayIterator) IsValid() bool {
	return i.position < i.size
}

// IsFirst returns true if the current position is 0.
func (i *ArrayIterator) IsFirst() bool {
	return i.position == 0
}

// Value returns the value of the current position of the iterator
func (i *ArrayIterator) Value() (Slice, error) {
	if i.position >= i.size {
		return nil, WithStack(IndexOutOfBoundsError)
	}
	if current := i.current; current != nil {
		return current, nil
	}
	value, err := i.s.At(i.position)
	return value, WithStack(err)
}

// Next moves to the next position.
func (i *ArrayIterator) Next() error {
	i.position++
	if i.position < i.size && i.current != nil {
		var err error
		// skip over entry
		i.current, err = i.current.Next()
		if err != nil {
			return WithStack(err)
		}
	} else {
		i.current = nil
	}
	return nil
}
