// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azeventhubs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/log"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/amqpwrap"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/eh"
	"github.com/Azure/go-amqp"
)

// EventHubProperties represents properties of an event hub, like the number of partitions.
type EventHubProperties struct {
	// CreatedOn is the time when the event hub was created.
	CreatedOn time.Time

	// Name of the event hub
	Name string

	// PartitionIDs for the event hub
	PartitionIDs []string

	// GeoReplicationEnabled is true if the event hub has geo-replication enabled.
	GeoReplicationEnabled bool
}

// GetEventHubPropertiesOptions contains optional parameters for the GetEventHubProperties function
type GetEventHubPropertiesOptions struct {
	// For future expansion
}

// getEventHubProperties gets event hub properties, like the available partition IDs and when the Event Hub was created.
func getEventHubProperties[LinkT internal.AMQPLink](ctx context.Context, eventName log.Event, ns internal.NamespaceForManagementOps, links *internal.Links[LinkT], eventHub string, retryOptions RetryOptions, options *GetEventHubPropertiesOptions) (EventHubProperties, error) {
	var props EventHubProperties

	err := links.RetryManagement(ctx, eventName, "getEventHubProperties", retryOptions, func(ctx context.Context, lwid internal.LinkWithID[amqpwrap.RPCLink]) error {
		tmpProps, err := getEventHubPropertiesInternal(ctx, ns, lwid.Link(), eventHub, options)

		if err != nil {
			return err
		}

		props = tmpProps
		return nil
	})

	return props, err

}

func getEventHubPropertiesInternal(ctx context.Context, ns internal.NamespaceForManagementOps, rpcLink amqpwrap.RPCLink, eventHub string, options *GetEventHubPropertiesOptions) (EventHubProperties, error) {
	token, err := ns.GetTokenForEntity(eventHub)

	if err != nil {
		return EventHubProperties{}, internal.TransformError(err)
	}

	amqpMsg := &amqp.Message{
		ApplicationProperties: map[string]any{
			"operation":      "READ",
			"name":           eventHub,
			"type":           "com.microsoft:eventhub",
			"security_token": token.Token,
		},
	}

	resp, err := rpcLink.RPC(ctx, amqpMsg)

	if err != nil {
		return EventHubProperties{}, err
	}

	if resp.Code >= 300 {
		return EventHubProperties{}, fmt.Errorf("failed getting partition properties: %v", resp.Description)
	}

	return newEventHubProperties(resp.Message.Value)
}

// PartitionProperties are the properties for a single partition.
type PartitionProperties struct {
	// BeginningSequenceNumber is the first sequence number for a partition.
	BeginningSequenceNumber int64
	// EventHubName is the name of the Event Hub for this partition.
	EventHubName string

	// IsEmpty is true if the partition is empty, false otherwise.
	IsEmpty bool

	// LastEnqueuedOffset is the offset of latest enqueued event.
	LastEnqueuedOffset string

	// LastEnqueuedOn is the date of latest enqueued event.
	LastEnqueuedOn time.Time

	// LastEnqueuedSequenceNumber is the sequence number of the latest enqueued event.
	LastEnqueuedSequenceNumber int64

	// PartitionID is the partition ID of this partition.
	PartitionID string
}

// GetPartitionPropertiesOptions are the options for the GetPartitionProperties function.
type GetPartitionPropertiesOptions struct {
	// For future expansion
}

// getPartitionProperties gets properties for a specific partition. This includes data like the last enqueued sequence number, the first sequence
// number and when an event was last enqueued to the partition.
func getPartitionProperties[LinkT internal.AMQPLink](ctx context.Context, eventName log.Event, ns internal.NamespaceForManagementOps, links *internal.Links[LinkT], eventHub string, partitionID string, retryOptions RetryOptions, options *GetPartitionPropertiesOptions) (PartitionProperties, error) {
	var props PartitionProperties

	err := links.RetryManagement(ctx, eventName, "getPartitionProperties", retryOptions, func(ctx context.Context, lwid internal.LinkWithID[amqpwrap.RPCLink]) error {
		tmpProps, err := getPartitionPropertiesInternal(ctx, ns, lwid.Link(), eventHub, partitionID, options)

		if err != nil {
			return err
		}

		props = tmpProps
		return nil
	})

	return props, err
}

func getPartitionPropertiesInternal(ctx context.Context, ns internal.NamespaceForManagementOps, rpcLink amqpwrap.RPCLink, eventHub string, partitionID string, options *GetPartitionPropertiesOptions) (PartitionProperties, error) {
	token, err := ns.GetTokenForEntity(eventHub)

	if err != nil {
		return PartitionProperties{}, err
	}

	amqpMsg := &amqp.Message{
		ApplicationProperties: map[string]any{
			"operation":      "READ",
			"name":           eventHub,
			"type":           "com.microsoft:partition",
			"partition":      partitionID,
			"security_token": token.Token,
		},
	}

	resp, err := rpcLink.RPC(context.Background(), amqpMsg)

	if err != nil {
		return PartitionProperties{}, internal.TransformError(err)
	}

	if resp.Code >= 300 {
		return PartitionProperties{}, fmt.Errorf("failed getting partition properties: %v", resp.Description)
	}

	return newPartitionProperties(resp.Message.Value)
}

func newEventHubProperties(amqpValue any) (EventHubProperties, error) {
	m, ok := amqpValue.(map[string]any)

	if !ok {
		return EventHubProperties{}, nil
	}

	partitionIDs, ok := m["partition_ids"].([]string)

	if !ok {
		return EventHubProperties{}, fmt.Errorf("invalid value for partition_ids")
	}

	name, ok := m["name"].(string)

	if !ok {
		return EventHubProperties{}, fmt.Errorf("invalid value for name")
	}

	createdOn, ok := m["created_at"].(time.Time)

	if !ok {
		return EventHubProperties{}, fmt.Errorf("invalid value for created_at")
	}

	geoFactor, ok := eh.ConvertToInt64(m["georeplication_factor"])

	if !ok {
		return EventHubProperties{}, fmt.Errorf("invalid value for georeplication_factor")
	}

	return EventHubProperties{
		Name:                  name,
		CreatedOn:             createdOn,
		PartitionIDs:          partitionIDs,
		GeoReplicationEnabled: geoFactor > 1,
	}, nil
}

func newPartitionProperties(amqpValue any) (PartitionProperties, error) {
	m, ok := amqpValue.(map[string]any)

	if !ok {
		return PartitionProperties{}, errors.New("invalid message format")
	}

	eventHubName, ok := m["name"].(string)

	if !ok {
		return PartitionProperties{}, errors.New("invalid message format")
	}

	partition, ok := m["partition"].(string)

	if !ok {
		return PartitionProperties{}, errors.New("invalid message format")
	}

	beginningSequenceNumber, ok := eh.ConvertToInt64(m["begin_sequence_number"])

	if !ok {
		return PartitionProperties{}, errors.New("invalid message format")
	}

	lastEnqueuedSequenceNumber, ok := eh.ConvertToInt64(m["last_enqueued_sequence_number"])

	if !ok {
		return PartitionProperties{}, errors.New("invalid message format")
	}

	lastEnqueuedOffsetStr, ok := m["last_enqueued_offset"].(string)

	if !ok {
		return PartitionProperties{}, errors.New("invalid message format")
	}

	lastEnqueuedTime, ok := m["last_enqueued_time_utc"].(time.Time)

	if !ok {
		return PartitionProperties{}, errors.New("invalid message format")
	}

	isEmpty, ok := m["is_partition_empty"].(bool)

	if !ok {
		return PartitionProperties{}, errors.New("invalid message format")
	}

	return PartitionProperties{
		BeginningSequenceNumber:    beginningSequenceNumber,
		LastEnqueuedSequenceNumber: lastEnqueuedSequenceNumber,
		LastEnqueuedOffset:         lastEnqueuedOffsetStr,
		LastEnqueuedOn:             lastEnqueuedTime,
		IsEmpty:                    isEmpty,
		PartitionID:                partition,
		EventHubName:               eventHubName,
	}, nil
}
