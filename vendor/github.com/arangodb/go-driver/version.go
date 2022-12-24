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

import (
	"strconv"
	"strings"
)

// Version holds a server version string. The string has the format "major.minor.sub".
// Major and minor will be numeric, and sub may contain a number or a textual version.
type Version string

// Major returns the major part of the version
// E.g. "3.1.7" -> 3
func (v Version) Major() int {
	parts := strings.Split(string(v), ".")
	result, _ := strconv.Atoi(parts[0])
	return result
}

// Minor returns the minor part of the version.
// E.g. "3.1.7" -> 1
func (v Version) Minor() int {
	parts := strings.Split(string(v), ".")
	if len(parts) >= 2 {
		result, _ := strconv.Atoi(parts[1])
		return result
	}
	return 0
}

// Sub returns the sub part of the version.
// E.g. "3.1.7" -> "7"
func (v Version) Sub() string {
	parts := strings.SplitN(string(v), ".", 3)
	if len(parts) == 3 {
		return parts[2]
	}
	return ""
}

// SubInt returns the sub part of the version as integer.
// The bool return value indicates if the sub part is indeed a number.
// E.g. "3.1.7" -> 7, true
// E.g. "3.1.foo" -> 0, false
func (v Version) SubInt() (int, bool) {
	result, err := strconv.Atoi(v.Sub())
	return result, err == nil
}

// CompareTo returns an integer comparing two version.
// The result will be 0 if v==other, -1 if v < other, and +1 if v > other.
// If major & minor parts are equal and sub part is not a number,
// the sub part will be compared using lexicographical string comparison.
func (v Version) CompareTo(other Version) int {
	a := v.Major()
	b := other.Major()
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}

	a = v.Minor()
	b = other.Minor()
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}

	a, aIsInt := v.SubInt()
	b, bIsInt := other.SubInt()

	if !aIsInt || !bIsInt {
		// Do a string comparison
		return strings.Compare(v.Sub(), other.Sub())
	}
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
