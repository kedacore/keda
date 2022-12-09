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
	"reflect"
	"testing"
)

func TestParseRange(t *testing.T) {
	testData := []struct {
		name    string
		from    string
		to      string
		exp     []int32
		isError bool
	}{
		{"success", "3", "10", []int32{3, 4, 5, 6, 7, 8, 9, 10}, false},
		{"failure, from not an int", "a", "10", nil, true},
		{"failure, to not an int", "3", "a", nil, true},
	}

	for _, tt := range testData {
		got, err := ParseRange(tt.from, tt.to)

		if err != nil && !tt.isError {
			t.Errorf("Expected no error but got %s\n", err)
		}

		if err == nil && tt.isError {
			t.Errorf("Expected error but got %s\n", err)
		}

		if !reflect.DeepEqual(tt.exp, got) {
			t.Errorf("Expected %v but got %v\n", tt.exp, got)
		}
	}
}

func TestParseint32List(t *testing.T) {
	testData := []struct {
		name    string
		pattern string
		exp     []int32
		isError bool
	}{
		{"success_single", "100", []int32{100}, false},
		{"success_list", "1,2,3,4,5,6,10", []int32{1, 2, 3, 4, 5, 6, 10}, false},
		{"success_list, range, list", "1,2,4-10", []int32{1, 2, 4, 5, 6, 7, 8, 9, 10}, false},
		{"success_range", "1-10", []int32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, false},
		{"failure_list", "a,2,3", nil, true},
		{"failure_range", "a-3", nil, true},
		{"failure_not_a_range", "a-3-", nil, true},
	}

	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseInt32List(tt.pattern)

			if err != nil && !tt.isError {
				t.Errorf("Expected no error but got %s\n", err)
			}

			if err == nil && tt.isError {
				t.Errorf("Expected error but got %s\n", err)
			}

			if !reflect.DeepEqual(tt.exp, got) {
				t.Errorf("Expected %v but got %v\n", tt.exp, got)
			}
		})
	}
}
