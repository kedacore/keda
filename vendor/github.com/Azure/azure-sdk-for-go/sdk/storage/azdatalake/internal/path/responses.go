//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package path

import (
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/generated"
)

// SetAccessControlResponse contains the response fields for the SetAccessControl operation.
type SetAccessControlResponse = generated.PathClientSetAccessControlResponse

// GetAccessControlResponse contains the response fields for the GetAccessControl operation.
type GetAccessControlResponse = generated.PathClientGetPropertiesResponse

// UpdateAccessControlResponse contains the response fields for the UpdateAccessControlRecursive operation.
type UpdateAccessControlResponse = generated.PathClientSetAccessControlRecursiveResponse

// RemoveAccessControlResponse contains the response fields for the RemoveAccessControlRecursive operation.
type RemoveAccessControlResponse = generated.PathClientSetAccessControlRecursiveResponse

// CreateResponse contains the response fields for the Create operation.
type CreateResponse = generated.PathClientCreateResponse

// DeleteResponse contains the response fields for the Delete operation.
type DeleteResponse = generated.PathClientDeleteResponse

type RenameResponse struct {
	// ContentLength contains the information returned from the Content-Length header response.
	ContentLength *int64

	// Continuation contains the information returned from the x-ms-continuation header response.
	Continuation *string

	// Date contains the information returned from the Date header response.
	Date *time.Time

	// ETag contains the information returned from the ETag header response.
	ETag *azcore.ETag

	// EncryptionKeySHA256 contains the information returned from the x-ms-encryption-key-sha256 header response.
	EncryptionKeySHA256 *string

	// IsServerEncrypted contains the information returned from the x-ms-request-server-encrypted header response.
	IsServerEncrypted *bool

	// LastModified contains the information returned from the Last-Modified header response.
	LastModified *time.Time

	// RequestID contains the information returned from the x-ms-request-id header response.
	RequestID *string

	// Version contains the information returned from the x-ms-version header response.
	Version *string
}

// We need to do this now in case we add the new client to renamed response - we don't want to break the cx

func FormatRenameResponse(createResp *CreateResponse) RenameResponse {
	newResp := RenameResponse{}
	newResp.ContentLength = createResp.ContentLength
	newResp.Continuation = createResp.Continuation
	newResp.Date = createResp.Date
	newResp.ETag = createResp.ETag
	newResp.EncryptionKeySHA256 = createResp.EncryptionKeySHA256
	newResp.IsServerEncrypted = createResp.IsServerEncrypted
	newResp.LastModified = createResp.LastModified
	newResp.RequestID = createResp.RequestID
	newResp.Version = createResp.Version
	return newResp
}

// removed BlobSequenceNumber, BlobCommittedBlockCount and BlobType headers from the original response:

// GetPropertiesResponse contains the response fields for the GetProperties operation.
type GetPropertiesResponse struct {
	// AcceptRanges contains the information returned from the Accept-Ranges header response.
	AcceptRanges *string

	// AccessControlList contains the combined list of access that are set for user, group and other on the file
	AccessControlList *string

	// AccessTier contains the information returned from the x-ms-access-tier header response.
	AccessTier *string

	// AccessTierChangeTime contains the information returned from the x-ms-access-tier-change-time header response.
	AccessTierChangeTime *time.Time

	// AccessTierInferred contains the information returned from the x-ms-access-tier-inferred header response.
	AccessTierInferred *bool

	// ArchiveStatus contains the information returned from the x-ms-archive-status header response.
	ArchiveStatus *string

	// CacheControl contains the information returned from the Cache-Control header response.
	CacheControl *string

	// ClientRequestID contains the information returned from the x-ms-client-request-id header response.
	ClientRequestID *string

	// ContentDisposition contains the information returned from the Content-Disposition header response.
	ContentDisposition *string

	// ContentEncoding contains the information returned from the Content-Encoding header response.
	ContentEncoding *string

	// ContentLanguage contains the information returned from the Content-Language header response.
	ContentLanguage *string

	// ContentLength contains the information returned from the Content-Length header response.
	ContentLength *int64

	// ContentMD5 contains the information returned from the Content-MD5 header response.
	ContentMD5 []byte

	// ContentType contains the information returned from the Content-Type header response.
	ContentType *string

	// CopyCompletionTime contains the information returned from the x-ms-copy-completion-time header response.
	CopyCompletionTime *time.Time

	// CopyID contains the information returned from the x-ms-copy-id header response.
	CopyID *string

	// CopyProgress contains the information returned from the x-ms-copy-progress header response.
	CopyProgress *string

	// CopySource contains the information returned from the x-ms-copy-source header response.
	CopySource *string

	// CopyStatus contains the information returned from the x-ms-copy-status header response.
	CopyStatus *CopyStatusType

	// CopyStatusDescription contains the information returned from the x-ms-copy-status-description header response.
	CopyStatusDescription *string

	// CreationTime contains the information returned from the x-ms-creation-time header response.
	CreationTime *time.Time

	// Date contains the information returned from the Date header response.
	Date *time.Time

	// DestinationSnapshot contains the information returned from the x-ms-copy-destination-snapshot header response.
	DestinationSnapshot *string

	// ETag contains the information returned from the ETag header response.
	ETag *azcore.ETag

	// EncryptionKeySHA256 contains the information returned from the x-ms-encryption-key-sha256 header response.
	EncryptionKeySHA256 *string

	// EncryptionScope contains the information returned from the x-ms-encryption-scope header response.
	EncryptionScope *string

	// EncryptionContext contains the information returned from the x-ms-encryption-context header response.
	EncryptionContext *string

	// ExpiresOn contains the information returned from the x-ms-expiry-time header response.
	ExpiresOn *time.Time

	// ImmutabilityPolicyExpiresOn contains the information returned from the x-ms-immutability-policy-until-date header response.
	ImmutabilityPolicyExpiresOn *time.Time

	// ImmutabilityPolicyMode contains the information returned from the x-ms-immutability-policy-mode header response.
	ImmutabilityPolicyMode *ImmutabilityPolicyMode

	// IsCurrentVersion contains the information returned from the x-ms-is-current-version header response.
	IsCurrentVersion *bool

	// IsIncrementalCopy contains the information returned from the x-ms-incremental-copy header response.
	IsIncrementalCopy *bool

	// IsSealed contains the information returned from the x-ms-blob-sealed header response.
	IsSealed *bool

	// IsServerEncrypted contains the information returned from the x-ms-server-encrypted header response.
	IsServerEncrypted *bool

	// LastAccessed contains the information returned from the x-ms-last-access-time header response.
	LastAccessed *time.Time

	// LastModified contains the information returned from the Last-Modified header response.
	LastModified *time.Time

	// LeaseDuration contains the information returned from the x-ms-lease-duration header response.
	LeaseDuration *DurationType

	// LeaseState contains the information returned from the x-ms-lease-state header response.
	LeaseState *StateType

	// LeaseStatus contains the information returned from the x-ms-lease-status header response.
	LeaseStatus *StatusType

	// LegalHold contains the information returned from the x-ms-legal-hold header response.
	LegalHold *bool

	// Metadata contains the information returned from the x-ms-meta header response.
	Metadata map[string]*string

	// ObjectReplicationPolicyID contains the information returned from the x-ms-or-policy-id header response.
	ObjectReplicationPolicyID *string

	// ObjectReplicationRules contains the information returned from the x-ms-or header response.
	ObjectReplicationRules map[string]*string

	// RehydratePriority contains the information returned from the x-ms-rehydrate-priority header response.
	RehydratePriority *string

	// RequestID contains the information returned from the x-ms-request-id header response.
	RequestID *string

	// TagCount contains the information returned from the x-ms-tag-count header response.
	TagCount *int64

	// Version contains the information returned from the x-ms-version header response.
	Version *string

	// VersionID contains the information returned from the x-ms-version-id header response.
	VersionID *string

	// Owner contains the information returned from the x-ms-owner header response.
	Owner *string

	// Group contains the information returned from the x-ms-group header response.
	Group *string

	// Permissions contains the information returned from the x-ms-permissions header response.
	Permissions *string

	// ResourceType contains the information returned from the x-ms-resource-type header response.
	ResourceType *string
}

func FormatGetPropertiesResponse(r *blob.GetPropertiesResponse, rawResponse *http.Response) GetPropertiesResponse {
	newResp := GetPropertiesResponse{}
	newResp.AcceptRanges = r.AcceptRanges
	newResp.AccessTier = r.AccessTier
	newResp.AccessTierChangeTime = r.AccessTierChangeTime
	newResp.AccessTierInferred = r.AccessTierInferred
	newResp.ArchiveStatus = r.ArchiveStatus
	newResp.CacheControl = r.CacheControl
	newResp.ClientRequestID = r.ClientRequestID
	newResp.ContentDisposition = r.ContentDisposition
	newResp.ContentEncoding = r.ContentEncoding
	newResp.ContentLanguage = r.ContentLanguage
	newResp.ContentLength = r.ContentLength
	newResp.ContentMD5 = r.ContentMD5
	newResp.ContentType = r.ContentType
	newResp.CopyCompletionTime = r.CopyCompletionTime
	newResp.CopyID = r.CopyID
	newResp.CopyProgress = r.CopyProgress
	newResp.CopySource = r.CopySource
	newResp.CopyStatus = r.CopyStatus
	newResp.CopyStatusDescription = r.CopyStatusDescription
	newResp.CreationTime = r.CreationTime
	newResp.Date = r.Date
	newResp.DestinationSnapshot = r.DestinationSnapshot
	newResp.ETag = r.ETag
	newResp.EncryptionKeySHA256 = r.EncryptionKeySHA256
	newResp.EncryptionScope = r.EncryptionScope
	newResp.ExpiresOn = r.ExpiresOn
	newResp.ImmutabilityPolicyExpiresOn = r.ImmutabilityPolicyExpiresOn
	newResp.ImmutabilityPolicyMode = r.ImmutabilityPolicyMode
	newResp.IsCurrentVersion = r.IsCurrentVersion
	newResp.IsIncrementalCopy = r.IsIncrementalCopy
	newResp.IsSealed = r.IsSealed
	newResp.IsServerEncrypted = r.IsServerEncrypted
	newResp.LastAccessed = r.LastAccessed
	newResp.LastModified = r.LastModified
	newResp.LeaseDuration = r.LeaseDuration
	newResp.LeaseState = r.LeaseState
	newResp.LeaseStatus = r.LeaseStatus
	newResp.LegalHold = r.LegalHold
	newResp.Metadata = r.Metadata
	newResp.ObjectReplicationPolicyID = r.ObjectReplicationPolicyID
	newResp.ObjectReplicationRules = r.ObjectReplicationRules
	newResp.RehydratePriority = r.RehydratePriority
	newResp.RequestID = r.RequestID
	newResp.TagCount = r.TagCount
	newResp.Version = r.Version
	newResp.VersionID = r.VersionID
	if val := rawResponse.Header.Get("x-ms-owner"); val != "" {
		newResp.Owner = &val
	}
	if val := rawResponse.Header.Get("x-ms-group"); val != "" {
		newResp.Group = &val
	}
	if val := rawResponse.Header.Get("x-ms-permissions"); val != "" {
		newResp.Permissions = &val
	}
	if val := rawResponse.Header.Get("x-ms-acl"); val != "" {
		newResp.AccessControlList = &val
	}
	if val := rawResponse.Header.Get("x-ms-resource-type"); val != "" {
		newResp.ResourceType = &val
	}
	if val := rawResponse.Header.Get("x-ms-encryption-context"); val != "" {
		newResp.EncryptionContext = &val
	}
	return newResp
}

// SetMetadataResponse contains the response fields for the SetMetadata operation.
type SetMetadataResponse = blob.SetMetadataResponse

// SetHTTPHeadersResponse contains the response from method Client.SetHTTPHeaders.
type SetHTTPHeadersResponse struct {
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

	// Version contains the information returned from the x-ms-version header response.
	Version *string
}

// removes blob sequence number from response

func FormatSetHTTPHeadersResponse(r *SetHTTPHeadersResponse, blobResp *blob.SetHTTPHeadersResponse) {
	r.ClientRequestID = blobResp.ClientRequestID
	r.Date = blobResp.Date
	r.ETag = blobResp.ETag
	r.LastModified = blobResp.LastModified
	r.RequestID = blobResp.RequestID
	r.Version = blobResp.Version
}
