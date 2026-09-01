package util

import (
	"reflect"
	"testing"
)

func TestGetValueByPath(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		path     string
		expected any
		wantErr  bool
	}{
		{
			name: "Valid path - String value",
			input: map[string]any{
				"some": map[string]any{
					"nested": map[string]any{
						"key": "value",
					},
				},
			},
			path:     "some.nested.key",
			expected: "value",
			wantErr:  false,
		},
		{
			name: "Valid path - Integer value",
			input: map[string]any{
				"another": map[string]any{
					"nested": map[string]any{
						"key": 42,
					},
				},
			},
			path:     "another.nested.key",
			expected: 42,
			wantErr:  false,
		},
		{
			name: "Invalid path - Key not found",
			input: map[string]any{
				"some": map[string]any{
					"nested": map[string]any{
						"key": "value",
					},
				},
			},
			path:     "nonexistent.path",
			expected: nil,
			wantErr:  true,
		},
		{
			name: "Interface slice",
			input: map[string]any{
				"some": []any{
					1, 2, 3,
				},
			},
			path:     "some.0",
			expected: 1,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := GetValueByPath(tt.input, tt.path)

			if (err != nil) != tt.wantErr {
				t.Errorf("Unexpected error status. got %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(actual, tt.expected) {
				t.Errorf("Mismatched result. got %v, want %v", actual, tt.expected)
			}
		})
	}
}
