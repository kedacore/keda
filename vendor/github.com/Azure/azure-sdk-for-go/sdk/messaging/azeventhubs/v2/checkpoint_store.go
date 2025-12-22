// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azeventhubs

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

// CheckpointStore is used by multiple consumers to coordinate progress and ownership for partitions.
type CheckpointStore interface {
	// ClaimOwnership attempts to claim ownership of the partitions in partitionOwnership and returns
	// the actual partitions that were claimed.
	ClaimOwnership(ctx context.Context, partitionOwnership []Ownership, options *ClaimOwnershipOptions) ([]Ownership, error)

	// ListCheckpoints lists all the available checkpoints.
	ListCheckpoints(ctx context.Context, fullyQualifiedNamespace string, eventHubName string, consumerGroup string, options *ListCheckpointsOptions) ([]Checkpoint, error)

	// ListOwnership lists all ownerships.
	ListOwnership(ctx context.Context, fullyQualifiedNamespace string, eventHubName string, consumerGroup string, options *ListOwnershipOptions) ([]Ownership, error)

	// SetCheckpoint updates a specific checkpoint with a sequence and offset.
	SetCheckpoint(ctx context.Context, checkpoint Checkpoint, options *SetCheckpointOptions) error
}

// Ownership tracks which consumer owns a particular partition.
type Ownership struct {
	ConsumerGroup           string
	EventHubName            string
	FullyQualifiedNamespace string
	PartitionID             string

	OwnerID          string       // the owner ID of the Processor
	LastModifiedTime time.Time    // used when calculating if ownership has expired
	ETag             *azcore.ETag // the ETag, used when attempting to claim or update ownership of a partition.
}

// Checkpoint tracks the last succesfully processed event in a partition.
type Checkpoint struct {
	ConsumerGroup           string
	EventHubName            string
	FullyQualifiedNamespace string
	PartitionID             string

	Offset         *string // the last succesfully processed Offset.
	SequenceNumber *int64  // the last succesfully processed SequenceNumber.
}

// ListCheckpointsOptions contains optional parameters for the ListCheckpoints function
type ListCheckpointsOptions struct {
	// For future expansion
}

// ListOwnershipOptions contains optional parameters for the ListOwnership function
type ListOwnershipOptions struct {
	// For future expansion
}

// SetCheckpointOptions contains optional parameters for the UpdateCheckpoint function
type SetCheckpointOptions struct {
	// For future expansion
}

// ClaimOwnershipOptions contains optional parameters for the ClaimOwnership function
type ClaimOwnershipOptions struct {
	// For future expansion
}
