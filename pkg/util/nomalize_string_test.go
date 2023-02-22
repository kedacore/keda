package util

import (
	"testing"
)

func TestNormalizeString(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "replaceAllSlash",
			input:    "/input/",
			expected: "-input-",
		},
		{
			name:     "replaceAllDot",
			input:    ".input.",
			expected: "-input-",
		},
		{
			name:     "replaceAllSemiColon",
			input:    ":input:",
			expected: "-input-",
		},
		{
			name:     "replaceAllPercentage",
			input:    "%input%",
			expected: "-input-",
		},
		{
			name:     "replaceAllOpenedBracket",
			input:    "(input(",
			expected: "-input-",
		},
		{
			name:     "replaceAllClosedBracket",
			input:    ")input)",
			expected: "-input-",
		},
		{
			name:     "replaceCombinedString",
			input:    "/.:%()input/.:%()",
			expected: "------input------",
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			outputString := NormalizeString(test.input)
			if outputString != test.expected {
				t.Errorf("Expected %s but got %s", test.expected, outputString)
			}
		})
	}
}
