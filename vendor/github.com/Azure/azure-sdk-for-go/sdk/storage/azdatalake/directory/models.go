//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package directory

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/exported"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/generated"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/path"
)

// CreateOptions contains the optional parameters when calling the Create operation.
type CreateOptions struct {
	// AccessConditions contains parameters for accessing the file.
	AccessConditions *AccessConditions
	// CPKInfo contains a group of parameters for client provided encryption key.
	CPKInfo *CPKInfo
	// HTTPHeaders contains the HTTP headers for path operations.
	HTTPHeaders *HTTPHeaders
	// LeaseDuration specifies the duration of the lease, in seconds, or negative one
	// (-1) for a lease that never expires. A non-infinite lease can be
	// between 15 and 60 seconds.
	LeaseDuration *int64
	// ProposedLeaseID specifies the proposed lease ID for the file.
	ProposedLeaseID *string
	// Permissions is the octal representation of the permissions for user, group and mask.
	Permissions *string
	// Umask is the umask for the file.
	Umask *string
	// Owner is the owner of the file.
	Owner *string
	// Group is the owning group of the file.
	Group *string
	// ACL is the access control list for the file.
	ACL *string
}

func (o *CreateOptions) format() (*generated.LeaseAccessConditions, *generated.ModifiedAccessConditions, *generated.PathHTTPHeaders, *generated.PathClientCreateOptions, *generated.CPKInfo) {
	resource := generated.PathResourceTypeDirectory
	createOpts := &generated.PathClientCreateOptions{
		Resource: &resource,
	}
	if o == nil {
		return nil, nil, nil, createOpts, nil
	}
	leaseAccessConditions, modifiedAccessConditions := exported.FormatPathAccessConditions(o.AccessConditions)
	createOpts.ACL = o.ACL
	createOpts.Group = o.Group
	createOpts.Owner = o.Owner
	createOpts.Umask = o.Umask
	createOpts.Permissions = o.Permissions
	createOpts.ProposedLeaseID = o.ProposedLeaseID
	createOpts.LeaseDuration = o.LeaseDuration

	var httpHeaders *generated.PathHTTPHeaders
	var cpkOpts *generated.CPKInfo

	if o.HTTPHeaders != nil {
		httpHeaders = path.FormatPathHTTPHeaders(o.HTTPHeaders)
	}
	if o.CPKInfo != nil {
		cpkOpts = &generated.CPKInfo{
			EncryptionAlgorithm: o.CPKInfo.EncryptionAlgorithm,
			EncryptionKey:       o.CPKInfo.EncryptionKey,
			EncryptionKeySHA256: o.CPKInfo.EncryptionKeySHA256,
		}
	}
	return leaseAccessConditions, modifiedAccessConditions, httpHeaders, createOpts, cpkOpts
}

// ===================================== PATH IMPORTS ===========================================

// SetAccessControlRecursiveOptions contains the optional parameters when calling the SetAccessControlRecursive operation.
type accessControlRecursiveOptions struct {
	// Max batch size is 2000 paths
	BatchSize *int32
	// Number of recursive calls to be made to set access control.
	MaxBatches *int32
	// ContinueOnFailure indicates whether to continue on failure when the operation encounters an error.
	ContinueOnFailure *bool
	// Marker is the continuation token to use when continuing the operation.
	Marker *string
}

func (o *accessControlRecursiveOptions) format(acl, mode string) (generated.PathSetAccessControlRecursiveMode, *generated.PathClientSetAccessControlRecursiveOptions) {
	defaultMaxBatches := to.Ptr(int32(-1))
	defaultForceFlag := to.Ptr(false)
	opts := &generated.PathClientSetAccessControlRecursiveOptions{
		ACL: &acl,
	}
	newMode := generated.PathSetAccessControlRecursiveMode(mode)
	if o.ContinueOnFailure == nil {
		opts.ForceFlag = defaultForceFlag
	} else {
		opts.ForceFlag = o.ContinueOnFailure
	}
	opts.Continuation = o.Marker
	opts.MaxRecords = o.BatchSize
	if o.MaxBatches == nil {
		o.MaxBatches = defaultMaxBatches
	}
	return newMode, opts
}

// SetAccessControlRecursiveOptions contains the optional parameters when calling the UpdateAccessControlRecursive operation.
type SetAccessControlRecursiveOptions = accessControlRecursiveOptions

// UpdateAccessControlRecursiveOptions contains the optional parameters when calling the UpdateAccessControlRecursive operation.
type UpdateAccessControlRecursiveOptions = accessControlRecursiveOptions

// RemoveAccessControlRecursiveOptions contains the optional parameters when calling the RemoveAccessControlRecursive operation.
type RemoveAccessControlRecursiveOptions = accessControlRecursiveOptions

// ================================= path imports ==================================

// DeleteOptions contains the optional parameters when calling the Delete operation.
type DeleteOptions = path.DeleteOptions

// RenameOptions contains the optional parameters when calling the Rename operation.
type RenameOptions = path.RenameOptions

// GetPropertiesOptions contains the optional parameters for the GetProperties method.
type GetPropertiesOptions = path.GetPropertiesOptions

// SetAccessControlOptions contains the optional parameters when calling the SetAccessControl operation.
type SetAccessControlOptions = path.SetAccessControlOptions

// GetAccessControlOptions contains the optional parameters when calling the GetAccessControl operation.
type GetAccessControlOptions = path.GetAccessControlOptions

// CPKInfo contains a group of parameters for client provided encryption key.
type CPKInfo = path.CPKInfo

// GetSASURLOptions contains the optional parameters for the GetSASURL method.
type GetSASURLOptions = path.GetSASURLOptions

// SetHTTPHeadersOptions contains the optional parameters for the SetHTTPHeaders method.
type SetHTTPHeadersOptions = path.SetHTTPHeadersOptions

// HTTPHeaders contains the HTTP headers for path operations.
type HTTPHeaders = path.HTTPHeaders

// SetMetadataOptions provides set of configurations for SetMetadata.
type SetMetadataOptions = path.SetMetadataOptions

// SharedKeyCredential contains an account's name and its primary or secondary key.
type SharedKeyCredential = path.SharedKeyCredential

// AccessConditions identifies path-specific access conditions which you optionally set.
type AccessConditions = path.AccessConditions

// SourceAccessConditions identifies source path-specific access conditions which you optionally set.
type SourceAccessConditions = path.SourceAccessConditions

// LeaseAccessConditions contains optional parameters to access leased entity.
type LeaseAccessConditions = path.LeaseAccessConditions

// ModifiedAccessConditions contains a group of parameters for specifying access conditions.
type ModifiedAccessConditions = path.ModifiedAccessConditions

// SourceModifiedAccessConditions contains a group of parameters for specifying source access conditions.
type SourceModifiedAccessConditions = path.SourceModifiedAccessConditions

// CPKScopeInfo contains a group of parameters for client provided encryption scope.
type CPKScopeInfo path.CPKScopeInfo

// ACLFailedEntry contains the failed ACL entry (response model).
type ACLFailedEntry = path.ACLFailedEntry
