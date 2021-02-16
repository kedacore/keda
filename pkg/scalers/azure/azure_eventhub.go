package azure

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-amqp-common-go/v3/aad"
	eventhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/Azure/go-autorest/autorest/azure"
)

// EventHubInfo to keep event hub connection and resources
type EventHubInfo struct {
	EventHubConnection    string
	EventHubConsumerGroup string
	StorageConnection     string
	BlobContainer         string
	Namespace             string
	EventHubName          string
	CheckpointType        string
}

// GetEventHubClient returns eventhub client
func GetEventHubClient(info EventHubInfo) (*eventhub.Hub, error) {
	// The user wants to use a connectionstring, not a pod identity
	if info.EventHubConnection != "" {
		hub, err := eventhub.NewHubFromConnectionString(info.EventHubConnection)
		if err != nil {
			return nil, fmt.Errorf("failed to create hub client: %s", err)
		}
		return hub, nil
	}

	// Since there is no connectionstring, then user wants to use pod identity
	// Internally, the JWTProvider will use Managed Service Identity to authenticate if no Service Principal info supplied
	provider, aadErr := aad.NewJWTProvider(func(config *aad.TokenProviderConfiguration) error {
		if config.Env == nil {
			config.Env = &azure.PublicCloud
		}
		return nil
	})

	if aadErr == nil {
		return eventhub.NewHub(info.Namespace, info.EventHubName, provider)
	}

	return nil, aadErr
}

// ParseAzureEventHubConnectionString parses Event Hub connection string into (namespace, name)
// Connection string should be in following format:
// Endpoint=sb://eventhub-namespace.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=secretKey123;EntityPath=eventhub-name
func ParseAzureEventHubConnectionString(connectionString string) (string, string, error) {
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
		eventHubNamespace, eventHubName, err = ParseAzureEventHubConnectionString(info.EventHubConnection)
		if err != nil {
			return "", "", err
		}
	} else {
		eventHubNamespace = info.Namespace
		eventHubName = info.EventHubName
	}

	return eventHubNamespace, eventHubName, nil
}
