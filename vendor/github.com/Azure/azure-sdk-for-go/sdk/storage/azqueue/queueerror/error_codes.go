//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package queueerror

import (
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/internal/generated"
)

// HasCode returns true if the provided error is an *azcore.ResponseError
// with its ErrorCode field equal to one of the specified Codes.
func HasCode(err error, codes ...Code) bool {
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

// Code - Error codes returned by the service
type Code = generated.StorageErrorCode

const (
	AccountAlreadyExists                 Code = "AccountAlreadyExists"
	AccountBeingCreated                  Code = "AccountBeingCreated"
	AccountIsDisabled                    Code = "AccountIsDisabled"
	AuthenticationFailed                 Code = "AuthenticationFailed"
	AuthorizationFailure                 Code = "AuthorizationFailure"
	AuthorizationPermissionMismatch      Code = "AuthorizationPermissionMismatch"
	AuthorizationProtocolMismatch        Code = "AuthorizationProtocolMismatch"
	AuthorizationResourceTypeMismatch    Code = "AuthorizationResourceTypeMismatch"
	AuthorizationServiceMismatch         Code = "AuthorizationServiceMismatch"
	AuthorizationSourceIPMismatch        Code = "AuthorizationSourceIPMismatch"
	ConditionHeadersNotSupported         Code = "ConditionHeadersNotSupported"
	ConditionNotMet                      Code = "ConditionNotMet"
	EmptyMetadataKey                     Code = "EmptyMetadataKey"
	FeatureVersionMismatch               Code = "FeatureVersionMismatch"
	InsufficientAccountPermissions       Code = "InsufficientAccountPermissions"
	InternalError                        Code = "InternalError"
	InvalidAuthenticationInfo            Code = "InvalidAuthenticationInfo"
	InvalidHTTPVerb                      Code = "InvalidHttpVerb"
	InvalidHeaderValue                   Code = "InvalidHeaderValue"
	InvalidInput                         Code = "InvalidInput"
	InvalidMD5                           Code = "InvalidMd5"
	InvalidMarker                        Code = "InvalidMarker"
	InvalidMetadata                      Code = "InvalidMetadata"
	InvalidQueryParameterValue           Code = "InvalidQueryParameterValue"
	InvalidRange                         Code = "InvalidRange"
	InvalidResourceName                  Code = "InvalidResourceName"
	InvalidURI                           Code = "InvalidUri"
	InvalidXMLDocument                   Code = "InvalidXmlDocument"
	InvalidXMLNodeValue                  Code = "InvalidXmlNodeValue"
	MD5Mismatch                          Code = "Md5Mismatch"
	MessageNotFound                      Code = "MessageNotFound"
	MessageTooLarge                      Code = "MessageTooLarge"
	MetadataTooLarge                     Code = "MetadataTooLarge"
	MissingContentLengthHeader           Code = "MissingContentLengthHeader"
	MissingRequiredHeader                Code = "MissingRequiredHeader"
	MissingRequiredQueryParameter        Code = "MissingRequiredQueryParameter"
	MissingRequiredXMLNode               Code = "MissingRequiredXmlNode"
	MultipleConditionHeadersNotSupported Code = "MultipleConditionHeadersNotSupported"
	OperationTimedOut                    Code = "OperationTimedOut"
	OutOfRangeInput                      Code = "OutOfRangeInput"
	OutOfRangeQueryParameterValue        Code = "OutOfRangeQueryParameterValue"
	PopReceiptMismatch                   Code = "PopReceiptMismatch"
	QueueAlreadyExists                   Code = "QueueAlreadyExists"
	QueueBeingDeleted                    Code = "QueueBeingDeleted"
	QueueDisabled                        Code = "QueueDisabled"
	QueueNotEmpty                        Code = "QueueNotEmpty"
	QueueNotFound                        Code = "QueueNotFound"
	RequestBodyTooLarge                  Code = "RequestBodyTooLarge"
	RequestURLFailedToParse              Code = "RequestUrlFailedToParse"
	ResourceAlreadyExists                Code = "ResourceAlreadyExists"
	ResourceNotFound                     Code = "ResourceNotFound"
	ResourceTypeMismatch                 Code = "ResourceTypeMismatch"
	ServerBusy                           Code = "ServerBusy"
	UnsupportedHTTPVerb                  Code = "UnsupportedHttpVerb"
	UnsupportedHeader                    Code = "UnsupportedHeader"
	UnsupportedQueryParameter            Code = "UnsupportedQueryParameter"
	UnsupportedXMLNode                   Code = "UnsupportedXmlNode"
)

var (
	// MissingSharedKeyCredential - Error is returned when SAS URL is being created without SharedKeyCredential.
	MissingSharedKeyCredential = errors.New("SAS can only be signed with a SharedKeyCredential")
)
