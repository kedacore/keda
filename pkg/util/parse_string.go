/*
Copyright 2021 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"fmt"
	"strconv"
	"strings"
)

func ParseRange(from, to string) ([]int32, error) {
	f, err := strconv.ParseInt(from, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("parse error for '%s': %s", from, err)
	}
	t, err := strconv.ParseInt(to, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("parse error for '%s': %s", to, err)
	}
	var parsed []int32
	for i := int32(f); i <= int32(t); i++ {
		parsed = append(parsed, i)
	}
	return parsed, nil
}

func ParseInt32List(pattern string) ([]int32, error) {
	var parsed []int32
	terms := strings.Split(pattern, ",")
	for _, term := range terms {
		literals := strings.Split(term, "-")
		switch {
		case len(literals) == 1:
			i, err := strconv.ParseInt(literals[0], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("parse error: %s", err)
			}
			parsed = append(parsed, int32(i))
		case len(literals) == 2:
			r, err := ParseRange(literals[0], literals[1])
			if err != nil {
				return nil, fmt.Errorf("error in range: %s", err)
			}
			parsed = append(parsed, r...)

		default:
			return nil, fmt.Errorf("error in range syntax, got '%s'", term)
		}
	}
	return parsed, nil
}
