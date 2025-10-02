//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package directory

import (
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/path"
)

// RenameResponse contains the response fields for the Rename operation.
type RenameResponse = path.RenameResponse

type setAccessControlRecursiveResponse struct {
	DirectoriesSuccessful *int32
	FailureCount          *int32
	FilesSuccessful       *int32
	FailedEntries         []*ACLFailedEntry
}

// SetAccessControlRecursiveResponse contains the response fields for the SetAccessControlRecursive operation.
type SetAccessControlRecursiveResponse = setAccessControlRecursiveResponse

// UpdateAccessControlRecursiveResponse contains the response fields for the UpdateAccessControlRecursive operation.
type UpdateAccessControlRecursiveResponse = setAccessControlRecursiveResponse

// RemoveAccessControlRecursiveResponse contains the response fields for the RemoveAccessControlRecursive operation.
type RemoveAccessControlRecursiveResponse = setAccessControlRecursiveResponse

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
