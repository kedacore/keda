package util

import (
	"net/url"
	"strings"
)

type urlPart string

const (
	// Hostname is a constant refers to the hostname part of the url
	Hostname urlPart = "Hostname"
	// Password is a constant that refers to a password portion of the url if there is one
	Password urlPart = "Password"
)

// NormalizeString will replace all slashes, dots, colons and percent signs with dashes
func NormalizeString(s string) string {
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, ":", "-")
	s = strings.ReplaceAll(s, "%", "-")
	return s
}

// MaskPartOfURL will parse a url and returned a masked version or an error
func MaskPartOfURL(s string, part urlPart) (string, error) {
	url, err := url.Parse(s)
	if err != nil {
		return "", err
	}

	var partOfURL string
	switch part {
	case Hostname:
		hostname := url.Hostname()
		partOfURL = hostname
	case Password:
		password, ok := url.User.Password()
		if ok {
			partOfURL = password
		}
	}

	if partOfURL != "" {
		return strings.ReplaceAll(s, partOfURL, "xxx"), nil
	}

	return s, nil
}
