//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package path

import (
	"errors"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/datalakeerror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/exported"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/generated"
	"time"
)

// DeleteOptions contains the optional parameters when calling the Delete operation.
type DeleteOptions struct {
	// AccessConditions contains parameters for accessing the path.
	AccessConditions *AccessConditions
	Paginated        *bool
}

func FormatDeleteOptions(o *DeleteOptions, recursive bool) (*generated.LeaseAccessConditions, *generated.ModifiedAccessConditions, *generated.PathClientDeleteOptions) {
	deleteOpts := &generated.PathClientDeleteOptions{
		Recursive: &recursive,
	}
	if o == nil {
		return nil, nil, deleteOpts
	}
	deleteOpts.Paginated = o.Paginated
	leaseAccessConditions, modifiedAccessConditions := exported.FormatPathAccessConditions(o.AccessConditions)
	return leaseAccessConditions, modifiedAccessConditions, deleteOpts
}

// RenameOptions contains the optional parameters when calling the Rename operation.
type RenameOptions struct {
	// SourceAccessConditions identifies the source path access conditions.
	SourceAccessConditions *SourceAccessConditions
	// AccessConditions contains parameters for accessing the path.
	AccessConditions *AccessConditions
	// CPKInfo contains CPK related information.
	CPKInfo *CPKInfo
}

func FormatRenameOptions(o *RenameOptions, path string) (*generated.LeaseAccessConditions, *generated.ModifiedAccessConditions, *generated.SourceModifiedAccessConditions, *generated.PathClientCreateOptions, *generated.CPKInfo) {
	// we don't need sourceModAccCond since this is not rename
	mode := generated.PathRenameModeLegacy
	createOpts := &generated.PathClientCreateOptions{
		Mode:         &mode,
		RenameSource: &path,
	}
	if o == nil {
		return nil, nil, nil, createOpts, nil
	}
	var cpkOpts *generated.CPKInfo
	if o.CPKInfo != nil {
		cpkOpts = &generated.CPKInfo{
			EncryptionAlgorithm: o.CPKInfo.EncryptionAlgorithm,
			EncryptionKey:       o.CPKInfo.EncryptionKey,
			EncryptionKeySHA256: o.CPKInfo.EncryptionKeySHA256,
		}
	}
	leaseAccessConditions, modifiedAccessConditions := exported.FormatPathAccessConditions(o.AccessConditions)
	if o.SourceAccessConditions != nil {
		if o.SourceAccessConditions.SourceLeaseAccessConditions != nil {
			createOpts.SourceLeaseID = o.SourceAccessConditions.SourceLeaseAccessConditions.LeaseID
		}
		if o.SourceAccessConditions.SourceModifiedAccessConditions != nil {
			sourceModifiedAccessConditions := &generated.SourceModifiedAccessConditions{
				SourceIfMatch:           o.SourceAccessConditions.SourceModifiedAccessConditions.SourceIfMatch,
				SourceIfModifiedSince:   o.SourceAccessConditions.SourceModifiedAccessConditions.SourceIfModifiedSince,
				SourceIfNoneMatch:       o.SourceAccessConditions.SourceModifiedAccessConditions.SourceIfNoneMatch,
				SourceIfUnmodifiedSince: o.SourceAccessConditions.SourceModifiedAccessConditions.SourceIfUnmodifiedSince,
			}
			return leaseAccessConditions, modifiedAccessConditions, sourceModifiedAccessConditions, createOpts, cpkOpts
		}
	}
	return leaseAccessConditions, modifiedAccessConditions, nil, createOpts, cpkOpts
}

// GetPropertiesOptions contains the optional parameters for the Client.GetProperties method.
type GetPropertiesOptions struct {
	AccessConditions *AccessConditions
	CPKInfo          *CPKInfo
}

func FormatGetPropertiesOptions(o *GetPropertiesOptions) *blob.GetPropertiesOptions {
	if o == nil {
		return nil
	}
	accessConditions := exported.FormatBlobAccessConditions(o.AccessConditions)
	if o.CPKInfo == nil {
		o.CPKInfo = &CPKInfo{}
	}
	return &blob.GetPropertiesOptions{
		AccessConditions: accessConditions,
		CPKInfo: &blob.CPKInfo{
			EncryptionKey:       o.CPKInfo.EncryptionKey,
			EncryptionAlgorithm: (*blob.EncryptionAlgorithmType)(o.CPKInfo.EncryptionAlgorithm),
			EncryptionKeySHA256: o.CPKInfo.EncryptionKeySHA256,
		},
	}
}

// ===================================== PATH IMPORTS ===========================================

// SetAccessControlOptions contains the optional parameters when calling the SetAccessControl operation.
type SetAccessControlOptions struct {
	// Owner is the owner of the path.
	Owner *string
	// Group is the owning group of the path.
	Group *string
	// ACL is the access control list for the path.
	ACL *string
	// Permissions is the octal representation of the permissions for user, group and mask.
	Permissions *string
	// AccessConditions contains parameters for accessing the path.
	AccessConditions *AccessConditions
}

func FormatSetAccessControlOptions(o *SetAccessControlOptions) (*generated.PathClientSetAccessControlOptions, *generated.LeaseAccessConditions, *generated.ModifiedAccessConditions, error) {
	if o == nil {
		return nil, nil, nil, datalakeerror.MissingParameters
	}
	// call path formatter since we're hitting dfs in this operation
	leaseAccessConditions, modifiedAccessConditions := exported.FormatPathAccessConditions(o.AccessConditions)
	if o.Owner == nil && o.Group == nil && o.ACL == nil && o.Permissions == nil {
		return nil, nil, nil, errors.New("at least one parameter should be set for SetAccessControl API")
	}
	return &generated.PathClientSetAccessControlOptions{
		Owner:       o.Owner,
		Group:       o.Group,
		ACL:         o.ACL,
		Permissions: o.Permissions,
	}, leaseAccessConditions, modifiedAccessConditions, nil
}

// GetAccessControlOptions contains the optional parameters when calling the GetAccessControl operation.
type GetAccessControlOptions struct {
	// UPN is the user principal name.
	UPN *bool
	// AccessConditions contains parameters for accessing the path.
	AccessConditions *AccessConditions
}

func FormatGetAccessControlOptions(o *GetAccessControlOptions) (*generated.PathClientGetPropertiesOptions, *generated.LeaseAccessConditions, *generated.ModifiedAccessConditions) {
	action := generated.PathGetPropertiesActionGetAccessControl
	if o == nil {
		return &generated.PathClientGetPropertiesOptions{
			Action: &action,
		}, nil, nil
	}
	// call path formatter since we're hitting dfs in this operation
	leaseAccessConditions, modifiedAccessConditions := exported.FormatPathAccessConditions(o.AccessConditions)
	return &generated.PathClientGetPropertiesOptions{
		Upn:    o.UPN,
		Action: &action,
	}, leaseAccessConditions, modifiedAccessConditions
}

// CPKInfo contains CPK related information.
type CPKInfo struct {
	// EncryptionAlgorithm is the algorithm used to encrypt the data.
	EncryptionAlgorithm *EncryptionAlgorithmType
	// EncryptionKey is the base64 encoded encryption key.
	EncryptionKey *string
	// EncryptionKeySHA256 is the base64 encoded SHA256 of the encryption key.
	EncryptionKeySHA256 *string
}

// GetSASURLOptions contains the optional parameters for the Client.GetSASURL method.
type GetSASURLOptions struct {
	// StartTime is the start time for this SAS token.
	StartTime *time.Time
}

func FormatGetSASURLOptions(o *GetSASURLOptions) time.Time {
	if o == nil {
		return time.Time{}
	}

	var st time.Time
	if o.StartTime != nil {
		st = o.StartTime.UTC()
	} else {
		st = time.Time{}
	}
	return st
}

// SetHTTPHeadersOptions contains the optional parameters for the Client.SetHTTPHeaders method.
type SetHTTPHeadersOptions struct {
	// AccessConditions contains parameters for accessing the path.
	AccessConditions *AccessConditions
}

func FormatSetHTTPHeadersOptions(o *SetHTTPHeadersOptions, httpHeaders HTTPHeaders) (*blob.SetHTTPHeadersOptions, blob.HTTPHeaders) {
	httpHeaderOpts := blob.HTTPHeaders{
		BlobCacheControl:       httpHeaders.CacheControl,
		BlobContentDisposition: httpHeaders.ContentDisposition,
		BlobContentEncoding:    httpHeaders.ContentEncoding,
		BlobContentLanguage:    httpHeaders.ContentLanguage,
		BlobContentMD5:         httpHeaders.ContentMD5,
		BlobContentType:        httpHeaders.ContentType,
	}
	if o == nil {
		return nil, httpHeaderOpts
	}
	accessConditions := exported.FormatBlobAccessConditions(o.AccessConditions)
	return &blob.SetHTTPHeadersOptions{
		AccessConditions: accessConditions,
	}, httpHeaderOpts
}

// HTTPHeaders contains the HTTP headers for path operations.
type HTTPHeaders struct {
	// CacheControl Sets the path's cache control. If specified, this property is stored with the path and returned with a read request.
	CacheControl *string
	// ContentDisposition Sets the path's Content-Disposition header.
	ContentDisposition *string
	// ContentEncoding Sets the path's content encoding. If specified, this property is stored with the path and returned with a read
	// request.
	ContentEncoding *string
	// ContentLanguage Set the path's content language. If specified, this property is stored with the path and returned with a read
	// request.
	ContentLanguage *string
	// ContentMD5 Specify the transactional md5 for the body, to be validated by the service.
	ContentMD5 []byte
	// ContentType Sets the path's content type. If specified, this property is stored with the path and returned with a read request.
	ContentType *string
}

func FormatBlobHTTPHeaders(o *HTTPHeaders) *blob.HTTPHeaders {

	opts := &blob.HTTPHeaders{
		BlobCacheControl:       o.CacheControl,
		BlobContentDisposition: o.ContentDisposition,
		BlobContentEncoding:    o.ContentEncoding,
		BlobContentLanguage:    o.ContentLanguage,
		BlobContentMD5:         o.ContentMD5,
		BlobContentType:        o.ContentType,
	}
	return opts
}

func FormatPathHTTPHeaders(o *HTTPHeaders) *generated.PathHTTPHeaders {
	if o == nil {
		return nil
	}
	opts := generated.PathHTTPHeaders{
		CacheControl:             o.CacheControl,
		ContentDisposition:       o.ContentDisposition,
		ContentEncoding:          o.ContentEncoding,
		ContentLanguage:          o.ContentLanguage,
		ContentMD5:               o.ContentMD5,
		ContentType:              o.ContentType,
		TransactionalContentHash: o.ContentMD5,
	}
	return &opts
}

// SetMetadataOptions provides set of configurations for Set Metadata on path operation
type SetMetadataOptions struct {
	// AccessConditions contains parameters for accessing the path.
	AccessConditions *AccessConditions
	// CPKInfo contains CPK related information.
	CPKInfo *CPKInfo
	// CPKScopeInfo specifies the encryption scope settings.
	CPKScopeInfo *CPKScopeInfo
}

func FormatSetMetadataOptions(o *SetMetadataOptions) *blob.SetMetadataOptions {
	if o == nil {
		return nil
	}
	accessConditions := exported.FormatBlobAccessConditions(o.AccessConditions)
	opts := &blob.SetMetadataOptions{
		AccessConditions: accessConditions,
	}
	if o.CPKInfo != nil {
		opts.CPKInfo = &blob.CPKInfo{
			EncryptionKey:       o.CPKInfo.EncryptionKey,
			EncryptionAlgorithm: (*blob.EncryptionAlgorithmType)(o.CPKInfo.EncryptionAlgorithm),
			EncryptionKeySHA256: o.CPKInfo.EncryptionKeySHA256,
		}
	}
	if o.CPKScopeInfo != nil {
		opts.CPKScopeInfo = o.CPKScopeInfo
	}
	return opts
}

// ========================================= constants =========================================

// SharedKeyCredential contains an account's name and its primary or secondary key.
type SharedKeyCredential = exported.SharedKeyCredential

// AccessConditions identifies access conditions which you optionally set.
type AccessConditions = exported.AccessConditions

// SourceAccessConditions identifies source access conditions which you optionally set.
type SourceAccessConditions = exported.SourceAccessConditions

// LeaseAccessConditions contains optional parameters to access leased entity.
type LeaseAccessConditions = exported.LeaseAccessConditions

// ModifiedAccessConditions contains a group of parameters for specifying access conditions.
type ModifiedAccessConditions = exported.ModifiedAccessConditions

// SourceModifiedAccessConditions contains a group of parameters for specifying access conditions.
type SourceModifiedAccessConditions = exported.SourceModifiedAccessConditions

// CPKScopeInfo contains a group of parameters for the Client.SetMetadata() method.
type CPKScopeInfo = blob.CPKScopeInfo

// ACLFailedEntry contains the failed ACL entry (response model).
type ACLFailedEntry = generated.ACLFailedEntry
