//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package lease

import "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/internal/generated"

// BlobAcquireResponse contains the response from method BlobClient.AcquireLease.
type BlobAcquireResponse = generated.BlobClientAcquireLeaseResponse

// BlobBreakResponse contains the response from method BlobClient.BreakLease.
type BlobBreakResponse = generated.BlobClientBreakLeaseResponse

// BlobChangeResponse contains the response from method BlobClient.ChangeLease.
type BlobChangeResponse = generated.BlobClientChangeLeaseResponse

// BlobReleaseResponse contains the response from method BlobClient.ReleaseLease.
type BlobReleaseResponse = generated.BlobClientReleaseLeaseResponse

// BlobRenewResponse contains the response from method BlobClient.RenewLease.
type BlobRenewResponse = generated.BlobClientRenewLeaseResponse

// ContainerAcquireResponse contains the response from method BlobClient.AcquireLease.
type ContainerAcquireResponse = generated.ContainerClientAcquireLeaseResponse

// ContainerBreakResponse contains the response from method BlobClient.BreakLease.
type ContainerBreakResponse = generated.ContainerClientBreakLeaseResponse

// ContainerChangeResponse contains the response from method BlobClient.ChangeLease.
type ContainerChangeResponse = generated.ContainerClientChangeLeaseResponse

// ContainerReleaseResponse contains the response from method BlobClient.ReleaseLease.
type ContainerReleaseResponse = generated.ContainerClientReleaseLeaseResponse

// ContainerRenewResponse contains the response from method BlobClient.RenewLease.
type ContainerRenewResponse = generated.ContainerClientRenewLeaseResponse
