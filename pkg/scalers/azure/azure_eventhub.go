package azure

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
	"github.com/go-logr/logr"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

// EventHubInfo to keep event hub connection and resources
type EventHubInfo struct {
	EventHubConnection       string
	EventHubConsumerGroup    string
	StorageConnection        string
	StorageAccountName       string
	BlobStorageEndpoint      string
	BlobContainer            string
	Namespace                string
	EventHubName             string
	CheckpointStrategy       string
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
		creds, chainedErr := NewChainedCredential(logger, info.PodIdentity.GetIdentityID(), info.PodIdentity.Provider)
		if chainedErr != nil {
			return nil, chainedErr
		}

		return azeventhubs.NewProducerClient(fmt.Sprintf("%s.%s", info.Namespace, info.ServiceBusEndpointSuffix), info.EventHubName, creds, opts)
	}

	return nil, fmt.Errorf("event hub does not support pod identity %v", info.PodIdentity.Provider)
}

func getHubAndNamespace(info EventHubInfo) (string, string, error) {
	var eventHubNamespace string
	var eventHubName string
	if info.EventHubConnection != "" {
		fields, err := azeventhubs.ParseConnectionString(info.EventHubConnection)
		if err != nil {
			return "", "", err
		}
		eventHubNamespace = fields.Endpoint
		if fields.EntityPath != nil {
			eventHubName = *fields.EntityPath
		}
	} else {
		eventHubNamespace = fmt.Sprintf("%s.%s", info.Namespace, info.ServiceBusEndpointSuffix)
		eventHubName = info.EventHubName
	}

	return eventHubNamespace, eventHubName, nil
}
