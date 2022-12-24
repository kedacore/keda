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
	"fmt"
	"strconv"
)

type ValueLength uint64

func (s ValueLength) String() string {
	return strconv.FormatInt(int64(s), 10)
}

// getVariableValueLength calculates the length of a variable length integer in unsigned LEB128 format
func getVariableValueLength(value ValueLength) ValueLength {
	l := ValueLength(1)
	for value >= 0x80 {
		value >>= 7
		l++
	}
	return l
}

// check if the length is beyond the size of a SIZE_MAX on this platform
func checkOverflow(length ValueLength) error {
	if length < 0 {
		return fmt.Errorf("Negative length")
	}
	// TODO
	return nil
}
