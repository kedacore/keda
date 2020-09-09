package util

import (
	"strings"
)

// NormalizeString will replace all slashes, dots and colons with dashes
func NormalizeString(s string) string {
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, ":", "-")
	return s
}
