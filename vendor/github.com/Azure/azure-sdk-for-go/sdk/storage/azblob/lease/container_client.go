//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package lease

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/internal/base"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/internal/generated"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/internal/shared"
)

// ContainerClient provides lease functionality for the underlying container client.
type ContainerClient struct {
	containerClient *container.Client
	leaseID         *string
}

// ContainerClientOptions contains the optional values when creating a ContainerClient.
type ContainerClientOptions struct {
	// LeaseID contains a caller-provided lease ID.
	LeaseID *string
}

// NewContainerClient creates a container lease client for the provided container client.
//   - client - an instance of a container client
//   - options - client options; pass nil to accept the default values
func NewContainerClient(client *container.Client, options *ContainerClientOptions) (*ContainerClient, error) {
	var leaseID *string
	if options != nil {
		leaseID = options.LeaseID
	}

	leaseID, err := shared.GenerateLeaseID(leaseID)
	if err != nil {
		return nil, err
	}

	return &ContainerClient{
		containerClient: client,
		leaseID:         leaseID,
	}, nil
}

func (c *ContainerClient) generated() *generated.ContainerClient {
	return base.InnerClient((*base.Client[generated.ContainerClient])(c.containerClient))
}

// LeaseID returns leaseID of the client.
func (c *ContainerClient) LeaseID() *string {
	return c.leaseID
}

// AcquireLease acquires a lease on the blob for write and delete operations.
// The lease Duration must be between 15 and 60 seconds, or infinite (-1).
// For more information, see https://docs.microsoft.com/rest/api/storageservices/lease-blob.
func (c *ContainerClient) AcquireLease(ctx context.Context, duration int32, o *ContainerAcquireOptions) (ContainerAcquireResponse, error) {
	blobAcquireLeaseOptions, modifiedAccessConditions := o.format()
	blobAcquireLeaseOptions.ProposedLeaseID = c.LeaseID()

	resp, err := c.generated().AcquireLease(ctx, duration, &blobAcquireLeaseOptions, modifiedAccessConditions)
	return resp, err
}

// BreakLease breaks the blob's previously-acquired lease (if it exists). Pass the LeaseBreakDefault (-1)
// constant to break a fixed-Duration lease when it expires or an infinite lease immediately.
// For more information, see https://docs.microsoft.com/rest/api/storageservices/lease-blob.
func (c *ContainerClient) BreakLease(ctx context.Context, o *ContainerBreakOptions) (ContainerBreakResponse, error) {
	blobBreakLeaseOptions, modifiedAccessConditions := o.format()
	resp, err := c.generated().BreakLease(ctx, blobBreakLeaseOptions, modifiedAccessConditions)
	return resp, err
}

// ChangeLease changes the blob's lease ID.
// For more information, see https://docs.microsoft.com/rest/api/storageservices/lease-blob.
func (c *ContainerClient) ChangeLease(ctx context.Context, proposedLeaseID string, o *ContainerChangeOptions) (ContainerChangeResponse, error) {
	if c.LeaseID() == nil {
		return ContainerChangeResponse{}, errors.New("leaseID cannot be nil")
	}
	changeLeaseOptions, modifiedAccessConditions, err := o.format()
	if err != nil {
		return ContainerChangeResponse{}, err
	}
	resp, err := c.generated().ChangeLease(ctx, *c.LeaseID(), proposedLeaseID, changeLeaseOptions, modifiedAccessConditions)

	// If lease has been changed successfully, set the leaseID in client
	if err == nil {
		c.leaseID = &proposedLeaseID
	}

	return resp, err
}

// RenewLease renews the blob's previously-acquired lease.
// For more information, see https://docs.microsoft.com/rest/api/storageservices/lease-blob.
func (c *ContainerClient) RenewLease(ctx context.Context, o *ContainerRenewOptions) (ContainerRenewResponse, error) {
	if c.LeaseID() == nil {
		return ContainerRenewResponse{}, errors.New("leaseID cannot be nil")
	}
	renewLeaseBlobOptions, modifiedAccessConditions := o.format()
	resp, err := c.generated().RenewLease(ctx, *c.LeaseID(), renewLeaseBlobOptions, modifiedAccessConditions)
	return resp, err
}

// ReleaseLease releases the blob's previously-acquired lease.
// For more information, see https://docs.microsoft.com/rest/api/storageservices/lease-blob.
func (c *ContainerClient) ReleaseLease(ctx context.Context, o *ContainerReleaseOptions) (ContainerReleaseResponse, error) {
	if c.LeaseID() == nil {
		return ContainerReleaseResponse{}, errors.New("leaseID cannot be nil")
	}
	renewLeaseBlobOptions, modifiedAccessConditions := o.format()
	resp, err := c.generated().ReleaseLease(ctx, *c.LeaseID(), renewLeaseBlobOptions, modifiedAccessConditions)
	return resp, err
}
