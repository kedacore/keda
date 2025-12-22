// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azeventhubs

import "context"

// ProcessorPartitionClient allows you to receive events, similar to a [PartitionClient], with a
// checkpoint store for tracking progress.
//
// This type is instantiated from [Processor.NextPartitionClient], which handles load balancing
// of partition ownership between multiple [Processor] instances.
//
// See [example_consuming_with_checkpoints_test.go] for an example.
//
// NOTE: If you do NOT want to use dynamic load balancing, and would prefer to track state and ownership
// manually, use the [ConsumerClient] instead.
//
// [example_consuming_with_checkpoints_test.go]: https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/messaging/azeventhubs/example_consuming_with_checkpoints_test.go
type ProcessorPartitionClient struct {
	partitionID           string
	innerClient           *PartitionClient
	checkpointStore       CheckpointStore
	cleanupFn             func()
	consumerClientDetails consumerClientDetails
}

// ReceiveEvents receives events until 'count' events have been received or the context
// has been cancelled.
//
// See [PartitionClient.ReceiveEvents] for more information, including troubleshooting.
func (c *ProcessorPartitionClient) ReceiveEvents(ctx context.Context, count int, options *ReceiveEventsOptions) ([]*ReceivedEventData, error) {
	return c.innerClient.ReceiveEvents(ctx, count, options)
}

// UpdateCheckpoint updates the checkpoint in the CheckpointStore. New Processors will resume after
// this checkpoint for this partition.
func (p *ProcessorPartitionClient) UpdateCheckpoint(ctx context.Context, latestEvent *ReceivedEventData, options *UpdateCheckpointOptions) error {
	seq := latestEvent.SequenceNumber
	offset := latestEvent.Offset

	return p.checkpointStore.SetCheckpoint(ctx, Checkpoint{
		ConsumerGroup:           p.consumerClientDetails.ConsumerGroup,
		EventHubName:            p.consumerClientDetails.EventHubName,
		FullyQualifiedNamespace: p.consumerClientDetails.FullyQualifiedNamespace,
		PartitionID:             p.partitionID,
		SequenceNumber:          &seq,
		Offset:                  &offset,
	}, nil)
}

// PartitionID is the partition ID of the partition we're receiving from.
// This will not change during the lifetime of this ProcessorPartitionClient.
func (p *ProcessorPartitionClient) PartitionID() string {
	return p.partitionID
}

// Close releases resources for the partition client.
// This does not close the ConsumerClient that the Processor was started with.
func (c *ProcessorPartitionClient) Close(ctx context.Context) error {
	c.cleanupFn()

	if c.innerClient != nil {
		return c.innerClient.Close(ctx)
	}

	return nil
}

// UpdateCheckpointOptions contains optional parameters for the [ProcessorPartitionClient.UpdateCheckpoint] function.
type UpdateCheckpointOptions struct {
	// For future expansion
}
