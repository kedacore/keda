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
