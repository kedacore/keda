/*
Copyright 2023 The KEDA Authors

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

import "testing"

func TestContains(t *testing.T) {
	type args[TP comparable] struct {
		s []TP
		e TP
	}

	type person struct {
		Name string
		Age  int
	}

	intTests := []struct {
		name string
		args args[int]
		want bool
	}{
		{
			name: "int slice contains element",
			args: args[int]{
				s: []int{1, 2, 3, 4, 5},
				e: 3,
			},
			want: true,
		},
		{
			name: "int slice does not contain element",
			args: args[int]{
				s: []int{1, 2, 3, 4, 5},
				e: 6,
			},
			want: false,
		},
		{
			name: "empty int slice does not contain element",
			args: args[int]{
				s: []int{},
				e: 6,
			},
			want: false,
		},
	}

	stringTests := []struct {
		name string
		args args[string]
		want bool
	}{
		{
			name: "string slice contains element",
			args: args[string]{
				s: []string{"a", "b", "c", "d", "e"},
				e: "c",
			},
			want: true,
		},
		{
			name: "string slice does not contain element",
			args: args[string]{
				s: []string{"a", "b", "c", "d", "e"},
				e: "f",
			},
			want: false,
		},
		{
			name: "empty string slice does not contain element",
			args: args[string]{
				s: []string{},
				e: "f",
			},
			want: false,
		},
		{
			name: "string slice contains empty string",
			args: args[string]{
				s: []string{"a", "b", "c", "d", "e"},
				e: "",
			},
			want: false,
		},
	}

	personTests := []struct {
		name string
		args args[person]
		want bool
	}{
		{
			name: "person slice contains element",
			args: args[person]{
				s: []person{
					{
						Name: "John",
						Age:  30,
					},
					{
						Name: "Jane",
						Age:  25,
					},
					{
						Name: "Bob",
						Age:  40,
					},
				},
				e: person{
					Name: "Jane",
					Age:  25,
				},
			},
			want: true,
		},
		{
			name: "person slice does not contain element",
			args: args[person]{
				s: []person{
					{
						Name: "John",
						Age:  30,
					},
					{
						Name: "Jane",
						Age:  25,
					},
					{
						Name: "Bob",
						Age:  40,
					},
				},
				e: person{
					Name: "Alice",
					Age:  20,
				},
			},
			want: false,
		},
		{
			name: "slice does not fully match",
			args: args[person]{
				s: []person{
					{
						Name: "John",
						Age:  30,
					},
					{
						Name: "Jane",
						Age:  25,
					},
					{
						Name: "Bob",
						Age:  40,
					},
				},
				e: person{
					Name: "Jane",
					Age:  30,
				},
			},
			want: false,
		},
	}

	for _, tt := range intTests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Contains(tt.args.s, tt.args.e); got != tt.want {
				t.Errorf("Contains() = %v, want %v", got, tt.want)
			}
		})
	}

	for _, tt := range stringTests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Contains(tt.args.s, tt.args.e); got != tt.want {
				t.Errorf("Contains() = %v, want %v", got, tt.want)
			}
		})
	}

	for _, tt := range personTests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Contains(tt.args.s, tt.args.e); got != tt.want {
				t.Errorf("Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}
