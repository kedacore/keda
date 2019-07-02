package scalers

import (
	"errors"
	"strings"
)

/* ParseAzureStorageConnectionString parses a storage account connection string into (endpointProtocol, accountName, key, endpointSuffix)
   Connection string should be in following format:
   DefaultEndpointsProtocol=https;AccountName=yourStorageAccountName;AccountKey=yourStorageAccountKey;EndpointSuffix=core.windows.net
*/
func ParseAzureStorageConnectionString(connectionString string) (string, string, string, string, error) {
	parts := strings.Split(connectionString, ";")

	var endpointProtocol, name, key, endpointSuffix string
	for _, v := range parts {
		if strings.HasPrefix(v, "DefaultEndpointsProtocol") {
			protocolParts := strings.SplitN(v, "=", 2)
			if len(protocolParts) == 2 {
				endpointProtocol = protocolParts[1]
			}
		} else if strings.HasPrefix(v, "AccountName") {
			accountParts := strings.SplitN(v, "=", 2)
			if len(accountParts) == 2 {
				name = accountParts[1]
			}
		} else if strings.HasPrefix(v, "AccountKey") {
			keyParts := strings.SplitN(v, "=", 2)
			if len(keyParts) == 2 {
				key = keyParts[1]
			}
		} else if strings.HasPrefix(v, "EndpointSuffix") {
			suffixParts := strings.SplitN(v, "=", 2)
			if len(suffixParts) == 2 {
				endpointSuffix = suffixParts[1]
			}
		}
	}
	if name == "" || key == "" || endpointProtocol == "" || endpointSuffix == "" {
		return "", "", "", "", errors.New("Can't parse storage connection string")
	}

	return endpointProtocol, name, key, endpointSuffix, nil
}
