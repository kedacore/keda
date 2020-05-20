package azure

import (
	"errors"
	"fmt"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/azure-storage-queue-go/azqueue"
	"net/url"
	"strings"
)

/* ParseAzureStorageConnectionString parses a storage account connection string into (endpointProtocol, accountName, key, endpointSuffix)
   Connection string should be in following format:
   DefaultEndpointsProtocol=https;AccountName=yourStorageAccountName;AccountKey=yourStorageAccountKey;EndpointSuffix=core.windows.net
*/

type AzureStorageEndpointType int

const (
	BlobEndpoint AzureStorageEndpointType = iota
	QueueEndpoint
	TableEndpoint
	FileEndpoint
)

func (e AzureStorageEndpointType) Prefix() string {
	return [...]string{"BlobEndpoint", "QueueEndpoint", "TableEndpoint", "FileEndpoint"}[e]
}

func (e AzureStorageEndpointType) Name() string {
	return [...]string{"blob", "queue", "table", "file"}[e]
}

func ParseAzureStorageQueueConnection(podIdentity, connectionString, accountName string) (azqueue.Credential, *url.URL, error) {
	switch podIdentity {
	case "azure":
		token, err := GetAzureADPodIdentityToken("https://storage.azure.com/")
		if err != nil {
			return nil, nil, err
		}

		if accountName == "" {
			return nil, nil, fmt.Errorf("accountName is required for podIdentity azure")
		}

		credential := azqueue.NewTokenCredential(token.AccessToken, nil)
		endpoint, _ := url.Parse(fmt.Sprintf("https://%s.queue.core.windows.net", accountName))
		return credential, endpoint, nil
	case "", "none":
		endpoint, accountName, accountKey, err := parseAzureStorageConnectionString(connectionString, QueueEndpoint)
		if err != nil {
			return nil, nil, err
		}

		credential, err := azqueue.NewSharedKeyCredential(accountName, accountKey)
		if err != nil {
			return nil, nil, err
		}

		return credential, endpoint, nil
	default:
		return nil, nil, fmt.Errorf("azure queues doesn't support %s pod identity type", podIdentity)
	}
}

func ParseAzureStorageBlobConnection(podIdentity, connectionString, accountName string) (azblob.Credential, *url.URL, error) {
	switch podIdentity {
	case "azure":
		token, err := GetAzureADPodIdentityToken("https://storage.azure.com/")
		if err != nil {
			return nil, nil, err
		}

		if accountName == "" {
			return nil, nil, fmt.Errorf("accountName is required for podIdentity azure")
		}

		credential := azblob.NewTokenCredential(token.AccessToken, nil)
		endpoint, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net", accountName))
		return credential, endpoint, nil
	case "", "none":
		endpoint, accountName, accountKey, err := parseAzureStorageConnectionString(connectionString, BlobEndpoint)
		if err != nil {
			return nil, nil, err
		}

		credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
		if err != nil {
			return nil, nil, err
		}

		return credential, endpoint, nil
	default:
		return nil, nil, fmt.Errorf("azure queues doesn't support %s pod identity type", podIdentity)
	}
}

func parseAzureStorageConnectionString(connectionString string, endpointType AzureStorageEndpointType) (*url.URL, string, string, error) {
	parts := strings.Split(connectionString, ";")

	getValue := func(pair string) string {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			return parts[1]
		}
		return ""
	}

	var endpointProtocol, name, key, endpointSuffix, endpoint string
	for _, v := range parts {
		if strings.HasPrefix(v, "DefaultEndpointsProtocol") {
			endpointProtocol = getValue(v)
		} else if strings.HasPrefix(v, "AccountName") {
			name = getValue(v)
		} else if strings.HasPrefix(v, "AccountKey") {
			key = getValue(v)
		} else if strings.HasPrefix(v, "EndpointSuffix") {
			endpointSuffix = getValue(v)
		} else if endpointType == BlobEndpoint && strings.HasPrefix(v, endpointType.Prefix()) {
			endpoint = getValue(v)
		} else if endpointType == QueueEndpoint && strings.HasPrefix(v, endpointType.Prefix()) {
			endpoint = getValue(v)
		} else if endpointType == TableEndpoint && strings.HasPrefix(v, endpointType.Prefix()) {
			endpoint = getValue(v)
		} else if endpointType == FileEndpoint && strings.HasPrefix(v, endpointType.Prefix()) {
			endpoint = getValue(v)
		}
	}

	if name == "" || key == "" {
		return nil, "", "", errors.New("can't parse storage connection string. Missing key or name")
	}

	if endpoint != "" {
		u, err := url.Parse(endpoint)
		if err != nil {
			return nil, "", "", err
		}
		return u, name, key, nil
	}

	if endpointProtocol == "" || endpointSuffix == "" {
		return nil, "", "", errors.New("can't parse storage connection string. Missing DefaultEndpointsProtocol or EndpointSuffix")
	}

	u, err := url.Parse(fmt.Sprintf("%s://%s.%s.%s", endpointProtocol, name, endpointType.Name(), endpointSuffix))
	if err != nil {
		return nil, "", "", err
	}

	return u, name, key, nil
}
