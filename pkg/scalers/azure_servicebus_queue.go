package scalers

import (
	"context"

	servicebus "github.com/Azure/azure-service-bus-go"
)

// GetAzureServiceBusQueueLength returns the length of a service bus queue, or -1 on error
func GetAzureServiceBusQueueLength(ctx context.Context, connectionString, queueName string) (int32, error) {
	// get namespace
	namespace, err := servicebus.NewNamespace(servicebus.NamespaceWithConnectionString(connectionString))
	if err != nil {
		return -1, err
	}

	// get queue manager from namespace
	queueManager := namespace.NewQueueManager()

	// queue manager.get(ctx, queueName) -> QueueEntitity
	queueEntity, err := queueManager.Get(ctx, queueName)
	if err != nil {
		return -1, err
	}

	// return QueueEntitity.CountDetails.ActiveMessageCount
	return *queueEntity.CountDetails.ActiveMessageCount, nil
}
