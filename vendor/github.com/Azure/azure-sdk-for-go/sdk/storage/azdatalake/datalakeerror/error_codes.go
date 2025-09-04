//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package datalakeerror

import (
	"errors"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

// HasCode returns true if the provided error is an *azcore.ResponseError
// with its ErrorCode field equal to one of the specified Codes.
func HasCode(err error, codes ...StorageErrorCode) bool {
	var respErr *azcore.ResponseError
	if !errors.As(err, &respErr) {
		return false
	}

	for _, code := range codes {
		if respErr.ErrorCode == string(code) {
			return true
		}
	}

	return false
}

// StorageErrorCode - Error codes returned by the service
type StorageErrorCode string

// dfs errors
const (
	ContentLengthMustBeZero                StorageErrorCode = "ContentLengthMustBeZero"
	InvalidFlushPosition                   StorageErrorCode = "InvalidFlushPosition"
	InvalidPropertyName                    StorageErrorCode = "InvalidPropertyName"
	InvalidSourceURI                       StorageErrorCode = "InvalidSourceUri"
	UnsupportedRestVersion                 StorageErrorCode = "UnsupportedRestVersion"
	RenameDestinationParentPathNotFound    StorageErrorCode = "RenameDestinationParentPathNotFound"
	SourcePathNotFound                     StorageErrorCode = "SourcePathNotFound"
	DestinationPathIsBeingDeleted          StorageErrorCode = "DestinationPathIsBeingDeleted"
	InvalidDestinationPath                 StorageErrorCode = "InvalidDestinationPath"
	InvalidRenameSourcePath                StorageErrorCode = "InvalidRenameSourcePath"
	InvalidSourceOrDestinationResourceType StorageErrorCode = "InvalidSourceOrDestinationResourceType"
	LeaseIsAlreadyBroken                   StorageErrorCode = "LeaseIsAlreadyBroken"
	LeaseNameMismatch                      StorageErrorCode = "LeaseNameMismatch"
	PathConflict                           StorageErrorCode = "PathConflict"
	SourcePathIsBeingDeleted               StorageErrorCode = "SourcePathIsBeingDeleted"
)

// (converted) blob errors - these errors are what we expect after we do a replace on the error string using the ConvertBlobError function
const (
	AccountAlreadyExists                              StorageErrorCode = "AccountAlreadyExists"
	AccountBeingCreated                               StorageErrorCode = "AccountBeingCreated"
	AccountIsDisabled                                 StorageErrorCode = "AccountIsDisabled"
	AppendPositionConditionNotMet                     StorageErrorCode = "AppendPositionConditionNotMet"
	AuthenticationFailed                              StorageErrorCode = "AuthenticationFailed"
	AuthorizationFailure                              StorageErrorCode = "AuthorizationFailure"
	AuthorizationPermissionMismatch                   StorageErrorCode = "AuthorizationPermissionMismatch"
	AuthorizationProtocolMismatch                     StorageErrorCode = "AuthorizationProtocolMismatch"
	AuthorizationResourceTypeMismatch                 StorageErrorCode = "AuthorizationResourceTypeMismatch"
	AuthorizationServiceMismatch                      StorageErrorCode = "AuthorizationServiceMismatch"
	AuthorizationSourceIPMismatch                     StorageErrorCode = "AuthorizationSourceIPMismatch"
	BlobNotFound                                      StorageErrorCode = "BlobNotFound"
	PathAlreadyExists                                 StorageErrorCode = "PathAlreadyExists"
	PathArchived                                      StorageErrorCode = "PathArchived"
	PathBeingRehydrated                               StorageErrorCode = "PathBeingRehydrated"
	PathImmutableDueToPolicy                          StorageErrorCode = "PathImmutableDueToPolicy"
	PathNotArchived                                   StorageErrorCode = "PathNotArchived"
	PathNotFound                                      StorageErrorCode = "PathNotFound"
	PathOverwritten                                   StorageErrorCode = "PathOverwritten"
	PathTierInadequateForContentLength                StorageErrorCode = "PathTierInadequateForContentLength"
	PathUsesCustomerSpecifiedEncryption               StorageErrorCode = "PathUsesCustomerSpecifiedEncryption"
	BlockCountExceedsLimit                            StorageErrorCode = "BlockCountExceedsLimit"
	BlockListTooLong                                  StorageErrorCode = "BlockListTooLong"
	CannotChangeToLowerTier                           StorageErrorCode = "CannotChangeToLowerTier"
	CannotVerifyCopySource                            StorageErrorCode = "CannotVerifyCopySource"
	ConditionHeadersNotSupported                      StorageErrorCode = "ConditionHeadersNotSupported"
	ConditionNotMet                                   StorageErrorCode = "ConditionNotMet"
	FileSystemAlreadyExists                           StorageErrorCode = "FileSystemAlreadyExists"
	FileSystemBeingDeleted                            StorageErrorCode = "FileSystemBeingDeleted"
	FileSystemDisabled                                StorageErrorCode = "FileSystemDisabled"
	FileSystemNotFound                                StorageErrorCode = "FileSystemNotFound"
	ContentLengthLargerThanTierLimit                  StorageErrorCode = "ContentLengthLargerThanTierLimit"
	CopyAcrossAccountsNotSupported                    StorageErrorCode = "CopyAcrossAccountsNotSupported"
	CopyIDMismatch                                    StorageErrorCode = "CopyIdMismatch"
	EmptyMetadataKey                                  StorageErrorCode = "EmptyMetadataKey"
	FeatureVersionMismatch                            StorageErrorCode = "FeatureVersionMismatch"
	IncrementalCopyPathMismatch                       StorageErrorCode = "IncrementalCopyPathMismatch"
	IncrementalCopyOfEarlierVersionSnapshotNotAllowed StorageErrorCode = "IncrementalCopyOfEarlierVersionSnapshotNotAllowed"
	IncrementalCopySourceMustBeSnapshot               StorageErrorCode = "IncrementalCopySourceMustBeSnapshot"
	InfiniteLeaseDurationRequired                     StorageErrorCode = "InfiniteLeaseDurationRequired"
	InsufficientAccountPermissions                    StorageErrorCode = "InsufficientAccountPermissions"
	InternalError                                     StorageErrorCode = "InternalError"
	InvalidAuthenticationInfo                         StorageErrorCode = "InvalidAuthenticationInfo"
	InvalidPathOrBlock                                StorageErrorCode = "InvalidPathOrBlock"
	InvalidPathTier                                   StorageErrorCode = "InvalidPathTier"
	InvalidPathType                                   StorageErrorCode = "InvalidPathType"
	InvalidBlockID                                    StorageErrorCode = "InvalidBlockId"
	InvalidBlockList                                  StorageErrorCode = "InvalidBlockList"
	InvalidHTTPVerb                                   StorageErrorCode = "InvalidHttpVerb"
	InvalidHeaderValue                                StorageErrorCode = "InvalidHeaderValue"
	InvalidInput                                      StorageErrorCode = "InvalidInput"
	InvalidMD5                                        StorageErrorCode = "InvalidMd5"
	InvalidMetadata                                   StorageErrorCode = "InvalidMetadata"
	InvalidOperation                                  StorageErrorCode = "InvalidOperation"
	InvalidPageRange                                  StorageErrorCode = "InvalidPageRange"
	InvalidQueryParameterValue                        StorageErrorCode = "InvalidQueryParameterValue"
	InvalidRange                                      StorageErrorCode = "InvalidRange"
	InvalidResourceName                               StorageErrorCode = "InvalidResourceName"
	InvalidSourcePathType                             StorageErrorCode = "InvalidSourcePathType"
	InvalidSourcePathURL                              StorageErrorCode = "InvalidSourcePathUrl"
	InvalidURI                                        StorageErrorCode = "InvalidUri"
	InvalidVersionForPagePathOperation                StorageErrorCode = "InvalidVersionForPagePathOperation"
	InvalidXMLDocument                                StorageErrorCode = "InvalidXmlDocument"
	InvalidXMLNodeValue                               StorageErrorCode = "InvalidXmlNodeValue"
	LeaseAlreadyBroken                                StorageErrorCode = "LeaseAlreadyBroken"
	LeaseAlreadyPresent                               StorageErrorCode = "LeaseAlreadyPresent"
	LeaseIDMismatchWithPathOperation                  StorageErrorCode = "LeaseIdMismatchWithPathOperation"
	LeaseIDMismatchWithFileSystemOperation            StorageErrorCode = "LeaseIdMismatchWithFileSystemOperation"
	LeaseIDMismatchWithLeaseOperation                 StorageErrorCode = "LeaseIdMismatchWithLeaseOperation"
	LeaseIDMissing                                    StorageErrorCode = "LeaseIdMissing"
	LeaseIsBreakingAndCannotBeAcquired                StorageErrorCode = "LeaseIsBreakingAndCannotBeAcquired"
	LeaseIsBreakingAndCannotBeChanged                 StorageErrorCode = "LeaseIsBreakingAndCannotBeChanged"
	LeaseIsBrokenAndCannotBeRenewed                   StorageErrorCode = "LeaseIsBrokenAndCannotBeRenewed"
	LeaseLost                                         StorageErrorCode = "LeaseLost"
	LeaseNotPresentWithPathOperation                  StorageErrorCode = "LeaseNotPresentWithPathOperation"
	LeaseNotPresentWithFileSystemOperation            StorageErrorCode = "LeaseNotPresentWithFileSystemOperation"
	LeaseNotPresentWithLeaseOperation                 StorageErrorCode = "LeaseNotPresentWithLeaseOperation"
	MD5Mismatch                                       StorageErrorCode = "Md5Mismatch"
	CRC64Mismatch                                     StorageErrorCode = "Crc64Mismatch"
	MaxPathSizeConditionNotMet                        StorageErrorCode = "MaxPathSizeConditionNotMet"
	MetadataTooLarge                                  StorageErrorCode = "MetadataTooLarge"
	MissingContentLengthHeader                        StorageErrorCode = "MissingContentLengthHeader"
	MissingRequiredHeader                             StorageErrorCode = "MissingRequiredHeader"
	MissingRequiredQueryParameter                     StorageErrorCode = "MissingRequiredQueryParameter"
	MissingRequiredXMLNode                            StorageErrorCode = "MissingRequiredXmlNode"
	MultipleConditionHeadersNotSupported              StorageErrorCode = "MultipleConditionHeadersNotSupported"
	NoAuthenticationInformation                       StorageErrorCode = "NoAuthenticationInformation"
	NoPendingCopyOperation                            StorageErrorCode = "NoPendingCopyOperation"
	OperationNotAllowedOnIncrementalCopyPath          StorageErrorCode = "OperationNotAllowedOnIncrementalCopyPath"
	OperationTimedOut                                 StorageErrorCode = "OperationTimedOut"
	OutOfRangeInput                                   StorageErrorCode = "OutOfRangeInput"
	OutOfRangeQueryParameterValue                     StorageErrorCode = "OutOfRangeQueryParameterValue"
	PendingCopyOperation                              StorageErrorCode = "PendingCopyOperation"
	PreviousSnapshotCannotBeNewer                     StorageErrorCode = "PreviousSnapshotCannotBeNewer"
	PreviousSnapshotNotFound                          StorageErrorCode = "PreviousSnapshotNotFound"
	PreviousSnapshotOperationNotSupported             StorageErrorCode = "PreviousSnapshotOperationNotSupported"
	RequestBodyTooLarge                               StorageErrorCode = "RequestBodyTooLarge"
	RequestURLFailedToParse                           StorageErrorCode = "RequestUrlFailedToParse"
	ResourceAlreadyExists                             StorageErrorCode = "ResourceAlreadyExists"
	ResourceNotFound                                  StorageErrorCode = "ResourceNotFound"
	ResourceTypeMismatch                              StorageErrorCode = "ResourceTypeMismatch"
	SequenceNumberConditionNotMet                     StorageErrorCode = "SequenceNumberConditionNotMet"
	SequenceNumberIncrementTooLarge                   StorageErrorCode = "SequenceNumberIncrementTooLarge"
	ServerBusy                                        StorageErrorCode = "ServerBusy"
	SnapshotCountExceeded                             StorageErrorCode = "SnapshotCountExceeded"
	SnapshotOperationRateExceeded                     StorageErrorCode = "SnapshotOperationRateExceeded"
	SnapshotsPresent                                  StorageErrorCode = "SnapshotsPresent"
	SourceConditionNotMet                             StorageErrorCode = "SourceConditionNotMet"
	SystemInUse                                       StorageErrorCode = "SystemInUse"
	TargetConditionNotMet                             StorageErrorCode = "TargetConditionNotMet"
	UnauthorizedPathOverwrite                         StorageErrorCode = "UnauthorizedPathOverwrite"
	UnsupportedHTTPVerb                               StorageErrorCode = "UnsupportedHttpVerb"
	UnsupportedHeader                                 StorageErrorCode = "UnsupportedHeader"
	UnsupportedQueryParameter                         StorageErrorCode = "UnsupportedQueryParameter"
	UnsupportedXMLNode                                StorageErrorCode = "UnsupportedXmlNode"
)

var (
	// BlobNotFound - Error is returned when resource is not found.

	// MissingSharedKeyCredential - Error is returned when SAS URL is being created without SharedKeyCredential.
	MissingSharedKeyCredential = bloberror.MissingSharedKeyCredential

	// MissingParameters - Error is returned when at least one parameter should be set for any API.
	MissingParameters = errors.New("at least one parameter should be set for SetAccessControl API")
)
