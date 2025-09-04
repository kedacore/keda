//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package filesystem

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/generated"
)

// GetAccessPolicyResponse contains the response from method FileSystemClient.GetAccessPolicy.
type GetAccessPolicyResponse struct {
	// PublicAccess contains the information returned from the x-ms-blob-public-access header response.
	PublicAccess *PublicAccessType

	// ClientRequestID contains the information returned from the x-ms-client-request-id header response.
	ClientRequestID *string

	// Date contains the information returned from the Date header response.
	Date *time.Time

	// ETag contains the information returned from the ETag header response.
	ETag *azcore.ETag

	// LastModified contains the information returned from the Last-Modified header response.
	LastModified *time.Time

	// RequestID contains the information returned from the x-ms-request-id header response.
	RequestID *string

	// a collection of signed identifiers
	SignedIdentifiers []*SignedIdentifier

	// Version contains the information returned from the x-ms-version header response.
	Version *string
}

// since we want to remove the blob prefix in access type
func formatGetAccessPolicyResponse(r *GetAccessPolicyResponse, contResp *container.GetAccessPolicyResponse) {
	r.PublicAccess = contResp.BlobPublicAccess
	r.ClientRequestID = contResp.ClientRequestID
	r.Date = contResp.Date
	r.ETag = contResp.ETag
	r.LastModified = contResp.LastModified
	r.RequestID = contResp.RequestID
	r.SignedIdentifiers = contResp.SignedIdentifiers
	r.Version = contResp.Version
}

// GetPropertiesResponse contains the response from method FileSystemClient.GetProperties.
type GetPropertiesResponse struct {
	// BlobPublicAccess contains the information returned from the x-ms-blob-public-access header response.
	PublicAccess *PublicAccessType

	// ClientRequestID contains the information returned from the x-ms-client-request-id header response.
	ClientRequestID *string

	// Date contains the information returned from the Date header response.
	Date *time.Time

	// DefaultEncryptionScope contains the information returned from the x-ms-default-encryption-scope header response.
	DefaultEncryptionScope *string

	// DenyEncryptionScopeOverride contains the information returned from the x-ms-deny-encryption-scope-override header response.
	DenyEncryptionScopeOverride *bool

	// ETag contains the information returned from the ETag header response.
	ETag *azcore.ETag

	// HasImmutabilityPolicy contains the information returned from the x-ms-has-immutability-policy header response.
	HasImmutabilityPolicy *bool

	// HasLegalHold contains the information returned from the x-ms-has-legal-hold header response.
	HasLegalHold *bool

	// IsImmutableStorageWithVersioningEnabled contains the information returned from the x-ms-immutable-storage-with-versioning-enabled
	// header response.
	IsImmutableStorageWithVersioningEnabled *bool

	// LastModified contains the information returned from the Last-Modified header response.
	LastModified *time.Time

	// LeaseDuration contains the information returned from the x-ms-lease-duration header response.
	LeaseDuration *DurationType

	// LeaseState contains the information returned from the x-ms-lease-state header response.
	LeaseState *StateType

	// LeaseStatus contains the information returned from the x-ms-lease-status header response.
	LeaseStatus *StatusType

	// Metadata contains the information returned from the x-ms-meta header response.
	Metadata map[string]*string

	// RequestID contains the information returned from the x-ms-request-id header response.
	RequestID *string

	// Version contains the information returned from the x-ms-version header response.
	Version *string
}

// removes the blob prefix in access type
func formatFileSystemProperties(r *GetPropertiesResponse, contResp *container.GetPropertiesResponse) {
	r.PublicAccess = contResp.BlobPublicAccess
	r.ClientRequestID = contResp.ClientRequestID
	r.Date = contResp.Date
	r.DefaultEncryptionScope = contResp.DefaultEncryptionScope
	r.DenyEncryptionScopeOverride = contResp.DenyEncryptionScopeOverride
	r.ETag = contResp.ETag
	r.HasImmutabilityPolicy = contResp.HasImmutabilityPolicy
	r.HasLegalHold = contResp.HasLegalHold
	r.IsImmutableStorageWithVersioningEnabled = contResp.IsImmutableStorageWithVersioningEnabled
	r.LastModified = contResp.LastModified
	r.LeaseDuration = contResp.LeaseDuration
	r.LeaseState = contResp.LeaseState
	r.LeaseStatus = contResp.LeaseStatus
	r.Metadata = contResp.Metadata
	r.RequestID = contResp.RequestID
	r.Version = contResp.Version
}

// CreateResponse contains the response from method FileSystemClient.Create.
type CreateResponse = container.CreateResponse

// DeleteResponse contains the response from method FileSystemClient.Delete.
type DeleteResponse = container.DeleteResponse

// SetMetadataResponse contains the response from method FileSystemClient.SetMetadata.
type SetMetadataResponse = container.SetMetadataResponse

// SetAccessPolicyResponse contains the response from method FileSystemClient.SetAccessPolicy.
type SetAccessPolicyResponse = container.SetAccessPolicyResponse

// ListPathsSegmentResponse contains the response from method FileSystemClient.ListPathsSegment.
type ListPathsSegmentResponse = generated.FileSystemClientListPathsResponse

// UndeletePathResponse contains the response from method FileSystemClient.UndeletePath.
type UndeletePathResponse = generated.PathClientUndeleteResponse

// ListDeletedPathsSegmentResponse contains the response from method FileSystemClient.ListPathsSegment.
type ListDeletedPathsSegmentResponse = generated.FileSystemClientListPathHierarchySegmentResponse

// ListDirectoryPathsSegmentResponse contains the response from method FileSystemClient.ListDirectoryPathsSegmentResponse.
type ListDirectoryPathsSegmentResponse = generated.FileSystemClientListPathHierarchySegmentResponse

// ListPathsHierarchySegmentResponse contains the response from method FileSystemClient.ListPathsHierarchySegment.
type ListPathsHierarchySegmentResponse = generated.ListPathsHierarchySegmentResponse

// PathHierarchyListSegment contains the response from method FileSystemClient.ListPathsHierarchySegment.
type PathHierarchyListSegment = generated.PathHierarchyListSegment

// CreateFileResponse contains the response from method FileSystemClient.CreateFile.
type CreateFileResponse = generated.PathClientCreateResponse

// CreateDirectoryResponse contains the response from method FileSystemClient.CreateDirectory.
type CreateDirectoryResponse = generated.PathClientCreateResponse
