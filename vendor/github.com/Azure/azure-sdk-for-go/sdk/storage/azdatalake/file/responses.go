//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package file

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/generated"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/generated_blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/path"
	"io"
	"net/http"
	"time"
)

// SetExpiryResponse contains the response fields for the SetExpiry operation.
type SetExpiryResponse = generated_blob.BlobClientSetExpiryResponse

// AppendDataResponse contains the response from method Client.AppendData.
type AppendDataResponse = generated.PathClientAppendDataResponse

// FlushDataResponse contains the response from method Client.FlushData.
type FlushDataResponse = generated.PathClientFlushDataResponse

// RenameResponse contains the response fields for the Rename operation.
type RenameResponse = path.RenameResponse

// DownloadStreamResponse contains the response from the DownloadStream method.
// To read from the stream, read from the Body field, or call the NewRetryReader method.
type DownloadStreamResponse struct {
	// DownloadResponse contains response fields from DownloadStream.
	DownloadResponse
	client   *Client
	getInfo  httpGetterInfo
	cpkInfo  *CPKInfo
	cpkScope *CPKScopeInfo
}

// NewRetryReader constructs new RetryReader stream for reading data. If a connection fails while
// reading, it will make additional requests to reestablish a connection and continue reading.
// Pass nil for options to accept the default options.
// Callers of this method should not access the DownloadStreamResponse.Body field.
func (r *DownloadStreamResponse) NewRetryReader(ctx context.Context, options *RetryReaderOptions) *RetryReader {
	if options == nil {
		options = &RetryReaderOptions{}
	}

	return newRetryReader(ctx, r.Body, r.getInfo, func(ctx context.Context, getInfo httpGetterInfo) (io.ReadCloser, error) {
		accessConditions := &AccessConditions{
			ModifiedAccessConditions: &ModifiedAccessConditions{IfMatch: getInfo.ETag},
		}
		options := DownloadStreamOptions{
			Range:            getInfo.Range,
			AccessConditions: accessConditions,
			CPKInfo:          r.cpkInfo,
			CPKScopeInfo:     r.cpkScope,
		}
		resp, err := r.client.DownloadStream(ctx, &options)
		if err != nil {
			return nil, err
		}
		return resp.Body, err
	}, *options)
}

// DownloadResponse contains the response fields for the GetProperties operation.
type DownloadResponse struct {
	// AcceptRanges contains the information returned from the Accept-Ranges header response.
	AcceptRanges *string

	// AccessControlList contains the combined list of access that are set for user, group and other on the file
	AccessControlList *string

	// Body contains the streaming response.
	Body io.ReadCloser

	// CacheControl contains the information returned from the Cache-Control header response.
	CacheControl *string

	// ClientRequestID contains the information returned from the x-ms-client-request-id header response.
	ClientRequestID *string

	// ContentCRC64 contains the information returned from the x-ms-content-crc64 header response.
	ContentCRC64 []byte

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

	// ContentRange contains the information returned from the Content-Range header response.
	ContentRange *string

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

	// Date contains the information returned from the Date header response.
	Date *time.Time

	// ETag contains the information returned from the ETag header response.
	ETag *azcore.ETag

	// EncryptionKeySHA256 contains the information returned from the x-ms-encryption-key-sha256 header response.
	EncryptionKeySHA256 *string

	// EncryptionScope contains the information returned from the x-ms-encryption-scope header response.
	EncryptionScope *string

	// EncryptionContext contains the information returned from the x-ms-encryption-context header response.
	EncryptionContext *string

	// ErrorCode contains the information returned from the x-ms-error-code header response.
	ErrorCode *string

	// ImmutabilityPolicyExpiresOn contains the information returned from the x-ms-immutability-policy-until-date header response.
	ImmutabilityPolicyExpiresOn *time.Time

	// ImmutabilityPolicyMode contains the information returned from the x-ms-immutability-policy-mode header response.
	ImmutabilityPolicyMode *ImmutabilityPolicyMode

	// IsCurrentVersion contains the information returned from the x-ms-is-current-version header response.
	IsCurrentVersion *bool

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

	// RequestID contains the information returned from the x-ms-request-id header response.
	RequestID *string

	// TagCount contains the information returned from the x-ms-tag-count header response.
	TagCount *int64

	// Version contains the information returned from the x-ms-version header response.
	Version *string

	// VersionID contains the information returned from the x-ms-version-id header response.
	VersionID *string
}

func FormatDownloadStreamResponse(r *blob.DownloadStreamResponse, rawResponse *http.Response) DownloadResponse {
	newResp := DownloadResponse{}
	if r != nil {
		newResp.AcceptRanges = r.AcceptRanges
		newResp.Body = r.Body
		newResp.ContentCRC64 = r.ContentCRC64
		newResp.ContentRange = r.ContentRange
		newResp.CacheControl = r.CacheControl
		newResp.ErrorCode = r.ErrorCode
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
		newResp.Date = r.Date
		newResp.ETag = r.ETag
		newResp.EncryptionKeySHA256 = r.EncryptionKeySHA256
		newResp.EncryptionScope = r.EncryptionScope
		newResp.ImmutabilityPolicyExpiresOn = r.ImmutabilityPolicyExpiresOn
		newResp.ImmutabilityPolicyMode = r.ImmutabilityPolicyMode
		newResp.IsCurrentVersion = r.IsCurrentVersion
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
		newResp.ObjectReplicationRules = r.DownloadResponse.ObjectReplicationRules
		newResp.RequestID = r.RequestID
		newResp.TagCount = r.TagCount
		newResp.Version = r.Version
		newResp.VersionID = r.VersionID
	}
	if val := rawResponse.Header.Get("x-ms-encryption-context"); val != "" {
		newResp.EncryptionContext = &val
	}
	if val := rawResponse.Header.Get("x-ms-acl"); val != "" {
		newResp.AccessControlList = &val
	}
	return newResp
}

// ========================================== path imports ===========================================================

// SetAccessControlResponse contains the response fields for the SetAccessControl operation.
type SetAccessControlResponse = path.SetAccessControlResponse

// SetHTTPHeadersResponse contains the response from method Client.SetHTTPHeaders.
type SetHTTPHeadersResponse = path.SetHTTPHeadersResponse

// GetAccessControlResponse contains the response fields for the GetAccessControl operation.
type GetAccessControlResponse = path.GetAccessControlResponse

// GetPropertiesResponse contains the response fields for the GetProperties operation.
type GetPropertiesResponse = path.GetPropertiesResponse

// SetMetadataResponse contains the response fields for the SetMetadata operation.
type SetMetadataResponse = path.SetMetadataResponse

// CreateResponse contains the response fields for the Create operation.
type CreateResponse = path.CreateResponse

// DeleteResponse contains the response fields for the Delete operation.
type DeleteResponse = path.DeleteResponse

// UpdateAccessControlResponse contains the response fields for the UpdateAccessControlRecursive operation.
type UpdateAccessControlResponse = path.UpdateAccessControlResponse

// RemoveAccessControlResponse contains the response fields for the RemoveAccessControlRecursive operation.
type RemoveAccessControlResponse = path.RemoveAccessControlResponse
