package azure

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
	"github.com/go-logr/logr"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

// EventHubInfo to keep event hub connection and resources
type EventHubInfo struct {
	EventHubConnection       string `keda:"name=connection,                order=authParams;resolvedEnv, optional"`
	EventHubConsumerGroup    string `keda:"name=consumerGroup,             order=triggerMetadata, default=$Default"`
	StorageConnection        string `keda:"name=storageConnection,         order=authParams;resolvedEnv, optional"`
	StorageAccountName       string `keda:"name=storageAccountName,        order=triggerMetadata, optional"`
	BlobStorageEndpoint      string
	BlobContainer            string `keda:"name=blobContainer,             order=triggerMetadata, optional"`
	Namespace                string `keda:"name=eventHubNamespace,         order=triggerMetadata;resolvedEnv, optional"`
	EventHubName             string `keda:"name=eventHubName,              order=triggerMetadata;resolvedEnv, optional"`
	CheckpointStrategy       string `keda:"name=checkpointStrategy,        order=triggerMetadata, optional"`
	ServiceBusEndpointSuffix string
	PodIdentity              kedav1alpha1.AuthPodIdentity
}

// GetEventHubClient returns eventhub client
func GetEventHubClient(info EventHubInfo, logger logr.Logger) (*azeventhubs.ProducerClient, error) {
	opts := &azeventhubs.ProducerClientOptions{TLSConfig: kedautil.CreateTLSClientConfig(false)}

	switch info.PodIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		hub, err := azeventhubs.NewProducerClientFromConnectionString(info.EventHubConnection, info.EventHubName, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create hub client: %w", err)
		}
		return hub, nil
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		creds, chainedErr := NewChainedCredential(logger, info.PodIdentity)
		if chainedErr != nil {
			return nil, chainedErr
		}

		return azeventhubs.NewProducerClient(fmt.Sprintf("%s.%s", info.Namespace, info.ServiceBusEndpointSuffix), info.EventHubName, creds, opts)
	}

	return nil, fmt.Errorf("event hub does not support pod identity %v", info.PodIdentity.Provider)
}

// parseAzureEventHubConnectionString parses Event Hub connection string into (namespace, name)
// Connection string should be in following format:
// Endpoint=sb://eventhub-namespace.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=secretKey123;EntityPath=eventhub-name
func parseAzureEventHubConnectionString(connectionString string) (string, string, error) {
	parts := strings.Split(connectionString, ";")

	var eventHubNamespace, eventHubName string
	for _, v := range parts {
		if strings.HasPrefix(v, "Endpoint") {
			endpointParts := strings.SplitN(v, "=", 2)
			if len(endpointParts) == 2 {
				endpointParts[1] = strings.TrimPrefix(endpointParts[1], "sb://")
				endpointParts[1] = strings.TrimSuffix(endpointParts[1], "/")
				eventHubNamespace = endpointParts[1]
			}
		} else if strings.HasPrefix(v, "EntityPath") {
			entityPathParts := strings.SplitN(v, "=", 2)
			if len(entityPathParts) == 2 {
				eventHubName = entityPathParts[1]
			}
		}
	}

	if eventHubNamespace == "" || eventHubName == "" {
		return "", "", errors.New("can't parse event hub connection string. Missing eventHubNamespace or eventHubName")
	}

	return eventHubNamespace, eventHubName, nil
}

func getHubAndNamespace(info EventHubInfo) (string, string, error) {
	var eventHubNamespace string
	var eventHubName string
	var err error
	if info.EventHubConnection != "" {
		eventHubNamespace, eventHubName, err = parseAzureEventHubConnectionString(info.EventHubConnection)
		if err != nil {
			return "", "", err
		}
	} else {
		eventHubNamespace = fmt.Sprintf("%s.%s", info.Namespace, info.ServiceBusEndpointSuffix)
		eventHubName = info.EventHubName
	}

	return eventHubNamespace, eventHubName, nil
}
