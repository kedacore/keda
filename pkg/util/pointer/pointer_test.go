/*
Copyright 2026 The KEDA Authors

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

package pointer

import "testing"

func TestIsEmpty(t *testing.T) {
	stringTests := []struct {
		name string
		in   *string
		want bool
	}{
		{name: "nil string pointer is not empty", in: nil, want: false},
		{name: "pointer to empty string is empty", in: strPtr(""), want: true},
		{name: "pointer to non-empty string is not empty", in: strPtr("abc"), want: false},
	}

	intTests := []struct {
		name string
		in   *int
		want bool
	}{
		{name: "nil int pointer is not empty", in: nil, want: false},
		{name: "pointer to zero int is empty", in: intPtr(0), want: true},
		{name: "pointer to non-zero int is not empty", in: intPtr(5), want: false},
	}

	boolTests := []struct {
		name string
		in   *bool
		want bool
	}{
		{name: "nil bool pointer is not empty", in: nil, want: false},
		{name: "pointer to false bool is empty", in: boolPtr(false), want: true},
		{name: "pointer to true bool is not empty", in: boolPtr(true), want: false},
	}

	for _, tc := range stringTests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsEmpty(tc.in); got != tc.want {
				t.Errorf("IsEmpty(%v) = %v, want %v", ptrVal(tc.in), got, tc.want)
			}
		})
	}
	for _, tc := range intTests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsEmpty(tc.in); got != tc.want {
				t.Errorf("IsEmpty(%v) = %v, want %v", ptrVal(tc.in), got, tc.want)
			}
		})
	}
	for _, tc := range boolTests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsEmpty(tc.in); got != tc.want {
				t.Errorf("IsEmpty(%v) = %v, want %v", ptrVal(tc.in), got, tc.want)
			}
		})
	}
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }
func boolPtr(b bool) *bool    { return &b }

func ptrVal[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}
