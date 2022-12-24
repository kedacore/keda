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

// builderStack is a stack of positions.
type builderStack struct {
	stack     []ValueLength
	bootstrap [4]ValueLength
}

// Push the given value on top of the stack
func (s *builderStack) Push(v ValueLength) {
	if s.stack == nil {
		s.stack = s.bootstrap[0:1]
		s.stack[0] = v
	} else {
		s.stack = append(s.stack, v)
	}
}

// Pop removes the top of the stack.
func (s *builderStack) Pop() {
	l := len(s.stack)
	if l > 0 {
		s.stack = s.stack[:l-1]
	}
}

func (s *builderStack) Clear() {
	s.stack = nil
}

// Tos returns the value at the top of the stack.
// Returns <value at top of stack>, <stack length>
func (s builderStack) Tos() (ValueLength, int) {
	//	_s := *s
	l := len(s.stack)
	if l > 0 {
		return (s.stack)[l-1], l
	}
	return 0, 0
}

// IsEmpty returns true if there are no values on the stack.
func (s builderStack) IsEmpty() bool {
	l := len(s.stack)
	return l == 0
}

// Len returns the number of elements of the stack.
func (s builderStack) Len() int {
	return len(s.stack)
}
