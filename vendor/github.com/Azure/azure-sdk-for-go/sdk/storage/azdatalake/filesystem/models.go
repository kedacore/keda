//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package filesystem

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/directory"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/file"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/exported"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/generated"
)

// SetAccessPolicyOptions provides set of configurations for FileSystem.SetAccessPolicy operation.
type SetAccessPolicyOptions struct {
	// Access Specifies whether data in the filesystem may be accessed publicly and the level of access.
	// If this header is not included in the request, filesystem data is private to the account owner.
	Access *PublicAccessType
	// AccessConditions identifies filesystem-specific access conditions which you optionally set.
	AccessConditions *AccessConditions
	// FileSystemACL sets the access policy for the filesystem.
	FileSystemACL []*SignedIdentifier
}

func (o *SetAccessPolicyOptions) format() *container.SetAccessPolicyOptions {
	if o == nil {
		return nil
	}
	return &container.SetAccessPolicyOptions{
		Access:           o.Access,
		AccessConditions: exported.FormatContainerAccessConditions(o.AccessConditions),
		ContainerACL:     o.FileSystemACL,
	}
}

// CreateOptions contains the optional parameters for the Client.Create method.
type CreateOptions struct {
	// Access specifies whether data in the filesystem may be accessed publicly and the level of access.
	Access *PublicAccessType
	// Metadata specifies a user-defined name-value pair associated with the filesystem.
	Metadata map[string]*string
	// CPKScopeInfo specifies the encryption scope settings to set on the filesystem.
	CPKScopeInfo *CPKScopeInfo
}

func (o *CreateOptions) format() *container.CreateOptions {
	if o == nil {
		return nil
	}
	return &container.CreateOptions{
		Access:       o.Access,
		Metadata:     o.Metadata,
		CPKScopeInfo: o.CPKScopeInfo,
	}
}

// DeleteOptions contains the optional parameters for the Client.Delete method.
type DeleteOptions struct {
	// AccessConditions identifies filesystem-specific access conditions which you optionally set.
	AccessConditions *AccessConditions
}

func (o *DeleteOptions) format() *container.DeleteOptions {
	if o == nil {
		return nil
	}
	return &container.DeleteOptions{
		AccessConditions: exported.FormatContainerAccessConditions(o.AccessConditions),
	}
}

// GetPropertiesOptions contains the optional parameters for the FileSystemClient.GetProperties method.
type GetPropertiesOptions struct {
	// LeaseAccessConditions contains parameters to access leased filesystem.
	LeaseAccessConditions *LeaseAccessConditions
}

func (o *GetPropertiesOptions) format() *container.GetPropertiesOptions {
	if o == nil {
		return nil
	}
	if o.LeaseAccessConditions == nil {
		o.LeaseAccessConditions = &LeaseAccessConditions{}
	}
	return &container.GetPropertiesOptions{
		LeaseAccessConditions: &container.LeaseAccessConditions{
			LeaseID: o.LeaseAccessConditions.LeaseID,
		},
	}
}

// SetMetadataOptions contains the optional parameters for the Client.SetMetadata method.
type SetMetadataOptions struct {
	// Metadata sets the metadata key-value pairs to set on the filesystem.
	Metadata map[string]*string
	// AccessConditions identifies filesystem-specific access conditions which you optionally set.
	AccessConditions *AccessConditions
}

func (o *SetMetadataOptions) format() *container.SetMetadataOptions {
	if o == nil {
		return nil
	}
	accConditions := exported.FormatContainerAccessConditions(o.AccessConditions)
	return &container.SetMetadataOptions{
		Metadata:                 o.Metadata,
		LeaseAccessConditions:    accConditions.LeaseAccessConditions,
		ModifiedAccessConditions: accConditions.ModifiedAccessConditions,
	}
}

// GetAccessPolicyOptions contains the optional parameters for the Client.GetAccessPolicy method.
type GetAccessPolicyOptions struct {
	// LeaseAccessConditions contains parameters to access leased filesystem.
	LeaseAccessConditions *LeaseAccessConditions
}

func (o *GetAccessPolicyOptions) format() *container.GetAccessPolicyOptions {
	if o == nil {
		return nil
	}
	if o.LeaseAccessConditions == nil {
		o.LeaseAccessConditions = &LeaseAccessConditions{}
	}
	return &container.GetAccessPolicyOptions{
		LeaseAccessConditions: &container.LeaseAccessConditions{
			LeaseID: o.LeaseAccessConditions.LeaseID,
		},
	}
}

// ListPathsOptions contains the optional parameters for the FileSystem.ListPaths operation.
type ListPathsOptions struct {
	// Marker contains last continuation token returned from the service for listing.
	Marker *string
	// MaxResults sets the maximum number of paths that will be returned per page.
	MaxResults *int32
	// Prefix filters the results to return only paths whose names begin with the specified prefix path.
	Prefix *string
	// UPN is the user principal name.
	UPN *bool
}

func (o *ListPathsOptions) format() generated.FileSystemClientListPathsOptions {
	if o == nil {
		return generated.FileSystemClientListPathsOptions{}
	}

	return generated.FileSystemClientListPathsOptions{
		Continuation: o.Marker,
		MaxResults:   o.MaxResults,
		Path:         o.Prefix,
		Upn:          o.UPN,
	}
}

// ListDirectoryPathsOptions contains the optional parameters from the FileSystem.ListDirectoryPathsOptions.
type ListDirectoryPathsOptions struct {
	// Marker contains last continuation token returned from the service for listing.
	Marker *string
	// MaxResults sets the maximum number of paths that will be returned per page.
	MaxResults *int32
	// Prefix filters the results to return only paths whose names begin with the specified prefix path.
	Prefix *string
}

func (o *ListDirectoryPathsOptions) format() generated.FileSystemClientListBlobHierarchySegmentOptions {
	showOnly := generated.ListBlobsShowOnlyDirectories
	if o == nil {
		return generated.FileSystemClientListBlobHierarchySegmentOptions{
			Showonly: &showOnly,
		}
	}
	return generated.FileSystemClientListBlobHierarchySegmentOptions{
		Marker:     o.Marker,
		MaxResults: o.MaxResults,
		Prefix:     o.Prefix,
		Showonly:   &showOnly,
	}
}

// ListDeletedPathsOptions contains the optional parameters for the FileSystem.ListDeletedPaths operation.
type ListDeletedPathsOptions struct {
	// Marker contains last continuation token returned from the service for listing.
	Marker *string
	// MaxResults sets the maximum number of paths that will be returned per page.
	MaxResults *int32
	// Prefix filters the results to return only paths whose names begin with the specified prefix path.
	Prefix *string
}

func (o *ListDeletedPathsOptions) format() generated.FileSystemClientListBlobHierarchySegmentOptions {
	showOnly := generated.ListBlobsShowOnlyDeleted
	if o == nil {
		return generated.FileSystemClientListBlobHierarchySegmentOptions{Showonly: &showOnly}
	}
	return generated.FileSystemClientListBlobHierarchySegmentOptions{
		Marker:     o.Marker,
		MaxResults: o.MaxResults,
		Prefix:     o.Prefix,
		Showonly:   &showOnly,
	}
}

// GetSASURLOptions contains the optional parameters for the Client.GetSASURL method.
type GetSASURLOptions struct {
	// StartTime is the time after which the SAS will become valid.
	StartTime *time.Time
}

func (o *GetSASURLOptions) format() time.Time {
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

// UndeletePathOptions contains the optional parameters for the FileSystem.UndeletePath operation.
// type UndeletePathOptions struct {
//	// placeholder
// }
// func (o *UndeletePathOptions) format() *UndeletePathOptions {
//	if o == nil {
//		return nil
//	}
//	return &UndeletePathOptions{}
// }

// CPKScopeInfo contains a group of parameters for the FileSystemClient.Create method.
type CPKScopeInfo = container.CPKScopeInfo

// AccessPolicy - An Access policy.
type AccessPolicy = container.AccessPolicy

// AccessPolicyPermission type simplifies creating the permissions string for a container's access policy.
// Initialize an instance of this type and then call its String method to set AccessPolicy's Permission field.
type AccessPolicyPermission = exported.AccessPolicyPermission

// SignedIdentifier - signed identifier.
type SignedIdentifier = container.SignedIdentifier

// SharedKeyCredential contains an account's name and its primary or secondary key.
type SharedKeyCredential = exported.SharedKeyCredential

// AccessConditions identifies filesystem-specific access conditions which you optionally set.
type AccessConditions = exported.AccessConditions

// LeaseAccessConditions contains optional parameters to access leased entity.
type LeaseAccessConditions = exported.LeaseAccessConditions

// ModifiedAccessConditions contains a group of parameters for specifying access conditions.
type ModifiedAccessConditions = exported.ModifiedAccessConditions

// PathList contains the path list from the ListPaths operation
type PathList = generated.PathList

// Path contains the path properties from the ListPaths operation
type Path = generated.Path

// PathItem contains the response from method FileSystemClient.ListPathsHierarchySegment.
type PathItem = generated.PathItemInternal

// PathProperties contains the response from method FileSystemClient.ListPathsHierarchySegment.
type PathProperties = generated.PathPropertiesInternal

// PathPrefix contains the response from method FileSystemClient.ListPathsHierarchySegment.
type PathPrefix = generated.PathPrefix

// CreateFileOptions contains the optional parameters when calling the CreateFile operation.
type CreateFileOptions = file.CreateOptions

// CreateDirectoryOptions contains the optional parameters when calling the CreateDirectory operation.
type CreateDirectoryOptions = directory.CreateOptions
