/*
Copyright 2021 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package azure

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/azure-storage-queue-go/azqueue"
	az "github.com/Azure/go-autorest/autorest/azure"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
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

const (
	// Azure storage resource is "https://storage.azure.com/" in all cloud environments
	storageResource = "https://storage.azure.com/"
)

var (
	// ErrAzureConnectionStringKeyName indicates an error in the connection string AccountKey or AccountName.
	ErrAzureConnectionStringKeyName = errors.New("can't parse storage connection string. Missing key or name")

	// ErrAzureConnectionStringEndpoint indicates an error in the connection string DefaultEndpointsProtocol or EndpointSuffix.
	ErrAzureConnectionStringEndpoint = errors.New("can't parse storage connection string. Missing DefaultEndpointsProtocol or EndpointSuffix")
)

// Prefix returns prefix for a StorageEndpointType
func (e StorageEndpointType) Prefix() string {
	return [...]string{"BlobEndpoint", "QueueEndpoint", "TableEndpoint", "FileEndpoint"}[e]
}

// Name returns resource name for StorageEndpointType
func (e StorageEndpointType) Name() string {
	return [...]string{"blob", "queue", "table", "file"}[e]
}

// GetEndpointSuffix returns the endpoint suffix for a StorageEndpointType based on the specified environment
func (e StorageEndpointType) GetEndpointSuffix(environment az.Environment) string {
	return fmt.Sprintf("%s.%s", e.Name(), environment.StorageEndpointSuffix)
}

// ParseAzureStorageEndpointSuffix parses cloud and endpointSuffix metadata and returns endpoint suffix
func ParseAzureStorageEndpointSuffix(metadata map[string]string, endpointType StorageEndpointType) (string, error) {
	envSuffixProvider := func(env az.Environment) (string, error) {
		return endpointType.GetEndpointSuffix(env), nil
	}

	return ParseEnvironmentProperty(metadata, DefaultEndpointSuffixKey, envSuffixProvider)
}

// ParseAzureStorageQueueConnection parses queue connection string and returns credential and resource url
func ParseAzureStorageQueueConnection(ctx context.Context, httpClient util.HTTPDoer, podIdentity kedav1alpha1.AuthPodIdentity, connectionString, accountName, endpointSuffix string) (azqueue.Credential, *url.URL, error) {
	switch podIdentity.Provider {
	case kedav1alpha1.PodIdentityProviderAzure, kedav1alpha1.PodIdentityProviderAzureWorkload:
		token, endpoint, err := parseAccessTokenAndEndpoint(ctx, httpClient, accountName, endpointSuffix, podIdentity)
		if err != nil {
			return nil, nil, err
		}

		credential := azqueue.NewTokenCredential(token, nil)
		return credential, endpoint, nil
	case "", kedav1alpha1.PodIdentityProviderNone:
		endpoint, accountName, accountKey, err := parseAzureStorageConnectionString(connectionString, QueueEndpoint)
		if err != nil {
			return nil, nil, err
		}

		if accountName == "" && accountKey == "" {
			return azqueue.NewAnonymousCredential(), endpoint, nil
		}

		credential, err := azqueue.NewSharedKeyCredential(accountName, accountKey)
		if err != nil {
			return nil, nil, err
		}

		return credential, endpoint, nil
	default:
		return nil, nil, fmt.Errorf("azure queues doesn't support %s pod identity type", podIdentity.Provider)
	}
}

// ParseAzureStorageBlobConnection parses blob connection string and returns credential and resource url
func ParseAzureStorageBlobConnection(ctx context.Context, httpClient util.HTTPDoer, podIdentity kedav1alpha1.AuthPodIdentity, connectionString, accountName, endpointSuffix string) (azblob.Credential, *url.URL, error) {
	switch podIdentity.Provider {
	case kedav1alpha1.PodIdentityProviderAzure, kedav1alpha1.PodIdentityProviderAzureWorkload:
		token, endpoint, err := parseAccessTokenAndEndpoint(ctx, httpClient, accountName, endpointSuffix, podIdentity)
		if err != nil {
			return nil, nil, err
		}

		credential := azblob.NewTokenCredential(token, nil)
		return credential, endpoint, nil
	case "", kedav1alpha1.PodIdentityProviderNone:
		endpoint, accountName, accountKey, err := parseAzureStorageConnectionString(connectionString, BlobEndpoint)
		if err != nil {
			return nil, nil, err
		}

		if accountName == "" && accountKey == "" {
			return azblob.NewAnonymousCredential(), endpoint, nil
		}

		credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
		if err != nil {
			return nil, nil, err
		}

		return credential, endpoint, nil
	default:
		return nil, nil, fmt.Errorf("azure storage doesn't support %s pod identity type", podIdentity.Provider)
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

	var endpointProtocol, name, key, sas, endpointSuffix, endpoint string
	for _, v := range parts {
		switch {
		case strings.HasPrefix(v, "DefaultEndpointsProtocol"):
			endpointProtocol = getValue(v)
		case strings.HasPrefix(v, "AccountName"):
			name = getValue(v)
		case strings.HasPrefix(v, "AccountKey"):
			key = getValue(v)
		case strings.HasPrefix(v, "SharedAccessSignature"):
			sas = getValue(v)
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

	if sas != "" && endpoint != "" {
		u, err := url.Parse(fmt.Sprintf("%s?%s", endpoint, sas))
		if err != nil {
			return nil, "", "", err
		}
		return u, "", "", nil
	}

	if name == "" || key == "" {
		return nil, "", "", ErrAzureConnectionStringKeyName
	}

	if endpoint != "" {
		u, err := url.Parse(endpoint)
		if err != nil {
			return nil, "", "", err
		}
		return u, name, key, nil
	}

	if endpointProtocol == "" || endpointSuffix == "" {
		return nil, "", "", ErrAzureConnectionStringEndpoint
	}

	u, err := url.Parse(fmt.Sprintf("%s://%s.%s.%s", endpointProtocol, name, endpointType.Name(), endpointSuffix))
	if err != nil {
		return nil, "", "", err
	}

	return u, name, key, nil
}

func parseAccessTokenAndEndpoint(ctx context.Context, httpClient util.HTTPDoer, accountName string, endpointSuffix string,
	podIdentity kedav1alpha1.AuthPodIdentity) (string, *url.URL, error) {
	var token AADToken
	var err error

	switch podIdentity.Provider {
	case kedav1alpha1.PodIdentityProviderAzure:
		token, err = GetAzureADPodIdentityToken(ctx, httpClient, podIdentity.GetIdentityID(), storageResource)
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		token, err = GetAzureADWorkloadIdentityToken(ctx, podIdentity.GetIdentityID(), storageResource)
	}

	if err != nil {
		return "", nil, err
	}

	if accountName == "" {
		return "", nil, fmt.Errorf("accountName is required for podIdentity azure")
	}

	endpoint, _ := url.Parse(fmt.Sprintf("https://%s.%s", accountName, endpointSuffix))
	return token.AccessToken, endpoint, nil
}
