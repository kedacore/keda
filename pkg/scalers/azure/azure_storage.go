package azure

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/azure-storage-queue-go/azqueue"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/util"
)

/* ParseAzureStorageConnectionString parses a storage account connection string into (endpointProtocol, accountName, key, endpointSuffix)
   Connection string should be in following format:
   DefaultEndpointsProtocol=https;AccountName=yourStorageAccountName;AccountKey=yourStorageAccountKey;EndpointSuffix=core.windows.net
*/

// StorageEndpointType for different types of storage provided by Azure
type StorageEndpointType int

const (
	// BlobEndpoint storage type
	BlobEndpoint StorageEndpointType = iota
	// QueueEndpoint storage type
	QueueEndpoint
	// TableEndpoint storage type
	TableEndpoint
	// FileEndpoint storage type
	FileEndpoint
)

// Prefix returns prefix for a StorageEndpointType
func (e StorageEndpointType) Prefix() string {
	return [...]string{"BlobEndpoint", "QueueEndpoint", "TableEndpoint", "FileEndpoint"}[e]
}

// Name returns resource name for StorageEndpointType
func (e StorageEndpointType) Name() string {
	return [...]string{"blob", "queue", "table", "file"}[e]
}

// ParseAzureStorageQueueConnection parses queue connection string and returns credential and resource url
func ParseAzureStorageQueueConnection(httpClient util.HTTPDoer, podIdentity kedav1alpha1.PodIdentityProvider, connectionString, accountName string) (azqueue.Credential, *url.URL, error) {
	switch podIdentity {
	case kedav1alpha1.PodIdentityProviderAzure:
		token, err := GetAzureADPodIdentityToken(httpClient, "https://storage.azure.com/")
		if err != nil {
			return nil, nil, err
		}

		if accountName == "" {
			return nil, nil, fmt.Errorf("accountName is required for podIdentity azure")
		}

		credential := azqueue.NewTokenCredential(token.AccessToken, nil)
		endpoint, _ := url.Parse(fmt.Sprintf("https://%s.queue.core.windows.net", accountName))
		return credential, endpoint, nil
	case "", kedav1alpha1.PodIdentityProviderNone:
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

// ParseAzureStorageBlobConnection parses blob connection string and returns credential and resource url
func ParseAzureStorageBlobConnection(httpClient util.HTTPDoer, podIdentity kedav1alpha1.PodIdentityProvider, connectionString, accountName string) (azblob.Credential, *url.URL, error) {
	switch podIdentity {
	case kedav1alpha1.PodIdentityProviderAzure:
		token, err := GetAzureADPodIdentityToken(httpClient, "https://storage.azure.com/")
		if err != nil {
			return nil, nil, err
		}

		if accountName == "" {
			return nil, nil, fmt.Errorf("accountName is required for podIdentity azure")
		}

		credential := azblob.NewTokenCredential(token.AccessToken, nil)
		endpoint, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net", accountName))
		return credential, endpoint, nil
	case "", kedav1alpha1.PodIdentityProviderNone:
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

func parseAzureStorageConnectionString(connectionString string, endpointType StorageEndpointType) (*url.URL, string, string, error) {
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
		switch {
		case strings.HasPrefix(v, "DefaultEndpointsProtocol"):
			endpointProtocol = getValue(v)
		case strings.HasPrefix(v, "AccountName"):
			name = getValue(v)
		case strings.HasPrefix(v, "AccountKey"):
			key = getValue(v)
		case strings.HasPrefix(v, "EndpointSuffix"):
			endpointSuffix = getValue(v)
		case endpointType == BlobEndpoint && strings.HasPrefix(v, endpointType.Prefix()):
			endpoint = getValue(v)
		case endpointType == QueueEndpoint && strings.HasPrefix(v, endpointType.Prefix()):
			endpoint = getValue(v)
		case endpointType == TableEndpoint && strings.HasPrefix(v, endpointType.Prefix()):
			endpoint = getValue(v)
		case endpointType == FileEndpoint && strings.HasPrefix(v, endpointType.Prefix()):
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
