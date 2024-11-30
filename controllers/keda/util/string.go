package util

import (
	"strings"
	"unicode"
)

func Truncate(s string, threshold int) string {
	if len(s) > threshold {
		s = s[:threshold]
		s = strings.TrimRightFunc(s, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		})
	}
	return s
}
