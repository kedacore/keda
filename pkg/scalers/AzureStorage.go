package scalers

import (
	"errors"
	"strings"
)

// ParseAzureStorageConnectionString parses a storage account connection string into (accountName, key)
func ParseAzureStorageConnectionString(connectionString string) (string, string, error) {
	parts := strings.Split(connectionString, ";")

	var name, key string
	for _, v := range parts {
		if strings.HasPrefix(v, "AccountName") {
			accountParts := strings.SplitN(v, "=", 2)
			if len(accountParts) == 2 {
				name = accountParts[1]
			}
		} else if strings.HasPrefix(v, "AccountKey") {
			keyParts := strings.SplitN(v, "=", 2)
			if len(keyParts) == 2 {
				key = keyParts[1]
			}
		}
	}
	if name == "" || key == "" {
		return "", "", errors.New("Can't parse storage connection string")
	}

	return name, key, nil
}
