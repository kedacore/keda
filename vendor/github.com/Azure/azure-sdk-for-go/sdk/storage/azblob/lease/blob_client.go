//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package lease

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/appendblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/internal/base"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/internal/generated"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/internal/shared"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/pageblob"
)

// BlobClient provides lease functionality for the underlying blob client.
type BlobClient struct {
	blobClient *blob.Client
	leaseID    *string
}

// BlobClientOptions contains the optional values when creating a BlobClient.
type BlobClientOptions struct {
	// LeaseID contains a caller-provided lease ID.
	LeaseID *string
}

// NewBlobClient creates a blob lease client for the provided blob client.
//   - client - an instance of a blob client
//   - options - client options; pass nil to accept the default values
func NewBlobClient[T appendblob.Client | blob.Client | blockblob.Client | pageblob.Client](client *T, options *BlobClientOptions) (*BlobClient, error) {
	var leaseID *string
	if options != nil {
		leaseID = options.LeaseID
	}

	leaseID, err := shared.GenerateLeaseID(leaseID)
	if err != nil {
		return nil, err
	}

	// TODO: improve once generics supports this scenario
	var blobClient *blob.Client
	switch t := any(client).(type) {
	case *appendblob.Client:
		rawClient, _ := base.InnerClients((*base.CompositeClient[generated.BlobClient, generated.AppendBlobClient])(t))
		blobClient = (*blob.Client)(rawClient)
	case *blockblob.Client:
		rawClient, _ := base.InnerClients((*base.CompositeClient[generated.BlobClient, generated.BlockBlobClient])(t))
		blobClient = (*blob.Client)(rawClient)
	case *pageblob.Client:
		rawClient, _ := base.InnerClients((*base.CompositeClient[generated.BlobClient, generated.PageBlobClient])(t))
		blobClient = (*blob.Client)(rawClient)
	case *blob.Client:
		blobClient = t
	default:
		// this shouldn't happen due to the generic type constraint
		return nil, fmt.Errorf("unhandled client type %T", client)
	}

	return &BlobClient{
		blobClient: blobClient,
		leaseID:    leaseID,
	}, nil
}

func (c *BlobClient) generated() *generated.BlobClient {
	return base.InnerClient((*base.Client[generated.BlobClient])(c.blobClient))
}

// LeaseID returns leaseID of the client.
func (c *BlobClient) LeaseID() *string {
	return c.leaseID
}

// AcquireLease acquires a lease on the blob for write and delete operations.
// The lease Duration must be between 15 and 60 seconds, or infinite (-1).
// For more information, see https://docs.microsoft.com/rest/api/storageservices/lease-blob.
func (c *BlobClient) AcquireLease(ctx context.Context, duration int32, o *BlobAcquireOptions) (BlobAcquireResponse, error) {
	blobAcquireLeaseOptions, modifiedAccessConditions := o.format()
	blobAcquireLeaseOptions.ProposedLeaseID = c.LeaseID()

	resp, err := c.generated().AcquireLease(ctx, duration, &blobAcquireLeaseOptions, modifiedAccessConditions)
	return resp, err
}

// BreakLease breaks the blob's previously-acquired lease (if it exists). Pass the LeaseBreakDefault (-1)
// constant to break a fixed-Duration lease when it expires or an infinite lease immediately.
// For more information, see https://docs.microsoft.com/rest/api/storageservices/lease-blob.
func (c *BlobClient) BreakLease(ctx context.Context, o *BlobBreakOptions) (BlobBreakResponse, error) {
	blobBreakLeaseOptions, modifiedAccessConditions := o.format()
	resp, err := c.generated().BreakLease(ctx, blobBreakLeaseOptions, modifiedAccessConditions)
	return resp, err
}

// ChangeLease changes the blob's lease ID.
// For more information, see https://docs.microsoft.com/rest/api/storageservices/lease-blob.
func (c *BlobClient) ChangeLease(ctx context.Context, proposedLeaseID string, o *BlobChangeOptions) (BlobChangeResponse, error) {
	if c.LeaseID() == nil {
		return BlobChangeResponse{}, errors.New("leaseID cannot be nil")
	}
	changeLeaseOptions, modifiedAccessConditions, err := o.format()
	if err != nil {
		return BlobChangeResponse{}, err
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
func (c *BlobClient) RenewLease(ctx context.Context, o *BlobRenewOptions) (BlobRenewResponse, error) {
	if c.LeaseID() == nil {
		return BlobRenewResponse{}, errors.New("leaseID cannot be nil")
	}
	renewLeaseBlobOptions, modifiedAccessConditions := o.format()
	resp, err := c.generated().RenewLease(ctx, *c.LeaseID(), renewLeaseBlobOptions, modifiedAccessConditions)
	return resp, err
}

// ReleaseLease releases the blob's previously-acquired lease.
// For more information, see https://docs.microsoft.com/rest/api/storageservices/lease-blob.
func (c *BlobClient) ReleaseLease(ctx context.Context, o *BlobReleaseOptions) (BlobReleaseResponse, error) {
	if c.LeaseID() == nil {
		return BlobReleaseResponse{}, errors.New("leaseID cannot be nil")
	}
	renewLeaseBlobOptions, modifiedAccessConditions := o.format()
	resp, err := c.generated().ReleaseLease(ctx, *c.LeaseID(), renewLeaseBlobOptions, modifiedAccessConditions)
	return resp, err
}
