package util

import (
	"net/url"
	"strings"
)

// NormalizeString will replace all slashes, dots, colons and percent signs with dashes
func NormalizeString(s string) string {
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, ":", "-")
	s = strings.ReplaceAll(s, "%", "-")
	return s
}

// MaskPassword will parse a url and returned a masked version or an error
func MaskPassword(s string) (string, error) {
	url, err := url.Parse(s)
	if err != nil {
		return "", err
	}

	if password, ok := url.User.Password(); ok {
		return strings.ReplaceAll(s, password, "xxx"), nil
	}

	return s, nil
}
