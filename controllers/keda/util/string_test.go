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
	"testing"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		threshold int
		want      string
	}{
		{
			name:      "string shorter than threshold",
			input:     "hello",
			threshold: 10,
			want:      "hello",
		},
		{
			name:      "string longer than threshold ending with special chars",
			input:     "abc---def---ghi---",
			threshold: 5,
			want:      "abc",
		},
		{
			name:      "63 character limit case",
			input:     "this-is-64-characters-name-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			threshold: 63,
			want:      "this-is-64-characters-name-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"[:63],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.input, tt.threshold)
			if got != tt.want {
				t.Errorf("Truncuate() = %v, want %v", got, tt.want)
			}
		})
	}
}
