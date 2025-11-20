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
	"errors"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue"
	"github.com/go-logr/logr"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
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
func (e StorageEndpointType) GetEndpointSuffix(environment AzEnvironment) string {
	return fmt.Sprintf("%s.%s", e.Name(), environment.StorageEndpointSuffix)
}

// ParseAzureStorageEndpointSuffix parses cloud and endpointSuffix metadata and returns endpoint suffix
func ParseAzureStorageEndpointSuffix(metadata map[string]string, endpointType StorageEndpointType) (string, error) {
	envSuffixProvider := func(env AzEnvironment) (string, error) {
		return endpointType.GetEndpointSuffix(env), nil
	}

	return ParseEnvironmentProperty(metadata, DefaultEndpointSuffixKey, envSuffixProvider)
}

// GetStorageBlobClient returns storage blob client
func GetStorageBlobClient(logger logr.Logger, podIdentity kedav1alpha1.AuthPodIdentity, connectionString, accountName, endpointSuffix string, timeout time.Duration) (*azblob.Client, error) {
	opts := &azblob.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Transport: kedautil.CreateHTTPClient(timeout, false),
		},
	}

	switch podIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		blobClient, err := azblob.NewClientFromConnectionString(connectionString, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create hub client: %w", err)
		}
		return blobClient, nil
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		creds, chainedErr := NewChainedCredential(logger, podIdentity)
		if chainedErr != nil {
			return nil, chainedErr
		}
		srvURL := fmt.Sprintf("https://%s.%s", accountName, endpointSuffix)
		return azblob.NewClient(srvURL, creds, opts)
	}

	return nil, fmt.Errorf("event hub does not support pod identity %v", podIdentity.Provider)
}

// GetStorageQueueClient returns storage queue client
func GetStorageQueueClient(logger logr.Logger, podIdentity kedav1alpha1.AuthPodIdentity, connectionString, accountName, endpointSuffix, queueName string, timeout time.Duration) (*azqueue.QueueClient, error) {
	opts := &azqueue.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Transport: kedautil.CreateHTTPClient(timeout, false),
		},
	}

	switch podIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		queueClient, err := azqueue.NewQueueClientFromConnectionString(connectionString, queueName, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create hub client: %w", err)
		}
		return queueClient, nil
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		creds, chainedErr := NewChainedCredential(logger, podIdentity)
		if chainedErr != nil {
			return nil, chainedErr
		}
		srvURL := fmt.Sprintf("https://%s.%s/%s", accountName, endpointSuffix, queueName)
		return azqueue.NewQueueClient(srvURL, creds, opts)
	}

	return nil, fmt.Errorf("event hub does not support pod identity %v", podIdentity.Provider)
}
