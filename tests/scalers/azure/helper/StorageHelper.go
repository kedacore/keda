//go:build e2e
// +build e2e

package helper

import (
	"strings"
)

func GetAccountFromStorageConnectionString(connection string) string {
	parts := strings.Split(connection, ";")

	getValue := func(pair string) string {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			return parts[1]
		}
		return ""
	}
	for _, v := range parts {
		if strings.HasPrefix(v, "AccountName") {
			return getValue(v)
		}
	}
	return ""
}
