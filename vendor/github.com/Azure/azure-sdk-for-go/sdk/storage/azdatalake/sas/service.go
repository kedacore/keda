//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package sas

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/exported"
)

// DatalakeSignatureValues is used to generate a Shared Access Signature (SAS) for an Azure Storage filesystem or path.
// For more information on creating service sas, see https://docs.microsoft.com/rest/api/storageservices/constructing-a-service-sas
// For more information on creating user delegation sas, see https://docs.microsoft.com/rest/api/storageservices/create-user-delegation-sas
type DatalakeSignatureValues struct {
	Version        string    `param:"sv"`  // If not specified, this defaults to Version
	Protocol       Protocol  `param:"spr"` // See the Protocol* constants
	StartTime      time.Time `param:"st"`  // Not specified if IsZero
	ExpiryTime     time.Time `param:"se"`  // Not specified if IsZero
	Permissions    string    `param:"sp"`  // Create by initializing FileSystemPermissions, FilePermissions or DirectoryPermissions and then call String()
	IPRange        IPRange   `param:"sip"`
	Identifier     string    `param:"si"`
	FileSystemName string
	// Use "" to create a FileSystem SAS
	// DirectoryPath will set this to "" if it is passed
	FilePath string
	// Not nil for a directory SAS (ie sr=d)
	// Use "" to create a FileSystem SAS
	DirectoryPath        string
	CacheControl         string // rscc
	ContentDisposition   string // rscd
	ContentEncoding      string // rsce
	ContentLanguage      string // rscl
	ContentType          string // rsct
	AuthorizedObjectID   string // saoid
	UnauthorizedObjectID string // suoid
	CorrelationID        string // scid
	EncryptionScope      string `param:"ses"`
}

// TODO: add snapshot and versioning support in the future

func getDirectoryDepth(path string) string {
	if path == "" {
		return ""
	}
	return fmt.Sprint(strings.Count(path, "/") + 1)
}

// SignWithSharedKey uses an account's SharedKeyCredential to sign this signature values to produce the proper SAS query parameters.
func (v DatalakeSignatureValues) SignWithSharedKey(sharedKeyCredential *SharedKeyCredential) (QueryParameters, error) {
	if v.Identifier == "" && v.ExpiryTime.IsZero() || v.Permissions == "" {
		return QueryParameters{}, errors.New("service SAS is missing at least one of these: ExpiryTime or Permissions")
	}

	// Make sure the permission characters are in the correct order
	perms, err := parsePathPermissions(v.Permissions)
	if err != nil {
		return QueryParameters{}, err
	}
	v.Permissions = perms.String()

	resource := "c"
	if v.DirectoryPath != "" {
		resource = "d"
		v.FilePath = ""
	} else if v.FilePath == "" {
		// do nothing
	} else {
		resource = "b"
	}

	if v.Version == "" {
		v.Version = Version
	}
	startTime, expiryTime := formatTimesForSigning(v.StartTime, v.ExpiryTime)

	signedIdentifier := v.Identifier

	// String to sign: http://msdn.microsoft.com/en-us/library/azure/dn140255.aspx
	stringToSign := strings.Join([]string{
		v.Permissions,
		startTime,
		expiryTime,
		getCanonicalName(sharedKeyCredential.AccountName(), v.FileSystemName, v.FilePath, v.DirectoryPath),
		signedIdentifier,
		v.IPRange.String(),
		string(v.Protocol),
		v.Version,
		resource,
		"", // snapshot not supported
		v.EncryptionScope,
		v.CacheControl,       // rscc
		v.ContentDisposition, // rscd
		v.ContentEncoding,    // rsce
		v.ContentLanguage,    // rscl
		v.ContentType},       // rsct
		"\n")

	signature, err := exported.ComputeHMACSHA256(sharedKeyCredential, stringToSign)
	if err != nil {
		return QueryParameters{}, err
	}

	p := QueryParameters{
		// Common SAS parameters
		version:         v.Version,
		protocol:        v.Protocol,
		startTime:       v.StartTime,
		expiryTime:      v.ExpiryTime,
		permissions:     v.Permissions,
		ipRange:         v.IPRange,
		encryptionScope: v.EncryptionScope,
		// Container/Blob-specific SAS parameters
		resource:             resource,
		cacheControl:         v.CacheControl,
		contentDisposition:   v.ContentDisposition,
		contentEncoding:      v.ContentEncoding,
		contentLanguage:      v.ContentLanguage,
		contentType:          v.ContentType,
		signedDirectoryDepth: getDirectoryDepth(v.DirectoryPath),
		authorizedObjectID:   v.AuthorizedObjectID,
		unauthorizedObjectID: v.UnauthorizedObjectID,
		correlationID:        v.CorrelationID,
		// Calculated SAS signature
		signature:  signature,
		identifier: signedIdentifier,
	}

	return p, nil
}

// SignWithUserDelegation uses an account's UserDelegationCredential to sign this signature values to produce the proper SAS query parameters.
func (v DatalakeSignatureValues) SignWithUserDelegation(userDelegationCredential *UserDelegationCredential) (QueryParameters, error) {
	if userDelegationCredential == nil {
		return QueryParameters{}, fmt.Errorf("cannot sign SAS query without User Delegation Key")
	}

	if v.ExpiryTime.IsZero() || v.Permissions == "" {
		return QueryParameters{}, errors.New("user delegation SAS is missing at least one of these: ExpiryTime or Permissions")
	}

	// Parse the resource
	resource := "c"
	if v.DirectoryPath != "" {
		resource = "d"
		v.FilePath = ""
	} else if v.FilePath == "" {
		// do nothing
	} else {
		resource = "b"
	}
	// make sure the permission characters are in the correct order
	if resource == "c" {
		perms, err := parseFileSystemPermissions(v.Permissions)
		if err != nil {
			return QueryParameters{}, err
		}
		v.Permissions = perms.String()
	} else {
		perms, err := parsePathPermissions(v.Permissions)
		if err != nil {
			return QueryParameters{}, err
		}
		v.Permissions = perms.String()
	}

	if v.Version == "" {
		v.Version = Version
	}
	startTime, expiryTime := formatTimesForSigning(v.StartTime, v.ExpiryTime)

	udk := exported.GetUDKParams(userDelegationCredential)

	udkStart, udkExpiry := formatTimesForSigning(*udk.SignedStart, *udk.SignedExpiry)

	stringToSign := strings.Join([]string{
		v.Permissions,
		startTime,
		expiryTime,
		getCanonicalName(exported.GetAccountName(userDelegationCredential), v.FileSystemName, v.FilePath, v.DirectoryPath),
		*udk.SignedOID,
		*udk.SignedTID,
		udkStart,
		udkExpiry,
		*udk.SignedService,
		*udk.SignedVersion,
		v.AuthorizedObjectID,
		v.UnauthorizedObjectID,
		v.CorrelationID,
		"", // Placeholder for SignedKeyDelegatedUserTenantId (future field)
		"", // Placeholder for SignedDelegatedUserObjectId (future field)
		v.IPRange.String(),
		string(v.Protocol),
		v.Version,
		resource,
		"", // snapshot not supported
		v.EncryptionScope,
		v.CacheControl,       // rscc
		v.ContentDisposition, // rscd
		v.ContentEncoding,    // rsce
		v.ContentLanguage,    // rscl
		v.ContentType},       // rsct
		"\n")

	signature, err := exported.ComputeUDCHMACSHA256(userDelegationCredential, stringToSign)
	if err != nil {
		return QueryParameters{}, err
	}

	p := QueryParameters{
		// Common SAS parameters
		version:         v.Version,
		protocol:        v.Protocol,
		startTime:       v.StartTime,
		expiryTime:      v.ExpiryTime,
		permissions:     v.Permissions,
		ipRange:         v.IPRange,
		encryptionScope: v.EncryptionScope,

		// Container/Blob-specific SAS parameters
		resource:             resource,
		identifier:           v.Identifier,
		cacheControl:         v.CacheControl,
		contentDisposition:   v.ContentDisposition,
		contentEncoding:      v.ContentEncoding,
		contentLanguage:      v.ContentLanguage,
		contentType:          v.ContentType,
		signedDirectoryDepth: getDirectoryDepth(v.DirectoryPath),
		authorizedObjectID:   v.AuthorizedObjectID,
		unauthorizedObjectID: v.UnauthorizedObjectID,
		correlationID:        v.CorrelationID,
		// Calculated SAS signature
		signature: signature,
	}

	// User delegation SAS specific parameters
	p.signedOID = *udk.SignedOID
	p.signedTID = *udk.SignedTID
	p.signedStart = *udk.SignedStart
	p.signedExpiry = *udk.SignedExpiry
	p.signedService = *udk.SignedService
	p.signedVersion = *udk.SignedVersion

	return p, nil
}

// getCanonicalName computes the canonical name for a container or blob resource for SAS signing.
func getCanonicalName(account string, filesystemName string, fileName string, directoryName string) string {
	// Container: "/blob/account/containername"
	// Blob:      "/blob/account/containername/blobname"
	elements := []string{"/blob/", account, "/", filesystemName}
	if fileName != "" {
		elements = append(elements, "/", strings.ReplaceAll(fileName, "\\", "/"))
	} else if directoryName != "" {
		elements = append(elements, "/", directoryName)
	}
	return strings.Join(elements, "")
}

// FileSystemPermissions type simplifies creating the permissions string for an Azure Storage container SAS.
// Initialize an instance of this type and then call its String method to set BlobSignatureValues' Permissions field.
// All permissions descriptions can be found here: https://docs.microsoft.com/en-us/rest/api/storageservices/create-service-sas#permissions-for-a-directory-container-or-blob
type FileSystemPermissions struct {
	Read, Add, Create, Write, Delete, List, Move bool
	Execute, ModifyOwnership, ModifyPermissions  bool // Meant for hierarchical namespace accounts
}

// String produces the SAS permissions string for an Azure Storage container.
// Call this method to set BlobSignatureValues' Permissions field.
func (p *FileSystemPermissions) String() string {
	var b bytes.Buffer
	if p.Read {
		b.WriteRune('r')
	}
	if p.Add {
		b.WriteRune('a')
	}
	if p.Create {
		b.WriteRune('c')
	}
	if p.Write {
		b.WriteRune('w')
	}
	if p.Delete {
		b.WriteRune('d')
	}
	if p.List {
		b.WriteRune('l')
	}
	if p.Move {
		b.WriteRune('m')
	}
	if p.Execute {
		b.WriteRune('e')
	}
	if p.ModifyOwnership {
		b.WriteRune('o')
	}
	if p.ModifyPermissions {
		b.WriteRune('p')
	}
	return b.String()
}

// Parse initializes ContainerPermissions' fields from a string.
func parseFileSystemPermissions(s string) (FileSystemPermissions, error) {
	p := FileSystemPermissions{} // Clear the flags
	for _, r := range s {
		switch r {
		case 'r':
			p.Read = true
		case 'a':
			p.Add = true
		case 'c':
			p.Create = true
		case 'w':
			p.Write = true
		case 'd':
			p.Delete = true
		case 'l':
			p.List = true
		case 'm':
			p.Move = true
		case 'e':
			p.Execute = true
		case 'o':
			p.ModifyOwnership = true
		case 'p':
			p.ModifyPermissions = true
		default:
			return FileSystemPermissions{}, fmt.Errorf("invalid permission: '%v'", r)
		}
	}
	return p, nil
}

// FilePermissions type simplifies creating the permissions string for an Azure Storage blob SAS.
// Initialize an instance of this type and then call its String method to set BlobSignatureValues' Permissions field.
type FilePermissions struct {
	Read, Add, Create, Write, Delete, List, Move bool
	Execute, Ownership, Permissions              bool
}

// String produces the SAS permissions string for an Azure Storage blob.
// Call this method to set BlobSignatureValues' Permissions field.
func (p *FilePermissions) String() string {
	var b bytes.Buffer
	if p.Read {
		b.WriteRune('r')
	}
	if p.Add {
		b.WriteRune('a')
	}
	if p.Create {
		b.WriteRune('c')
	}
	if p.Write {
		b.WriteRune('w')
	}
	if p.Delete {
		b.WriteRune('d')
	}
	if p.List {
		b.WriteRune('l')
	}
	if p.Move {
		b.WriteRune('m')
	}
	if p.Execute {
		b.WriteRune('e')
	}
	if p.Ownership {
		b.WriteRune('o')
	}
	if p.Permissions {
		b.WriteRune('p')
	}
	return b.String()
}

// DirectoryPermissions type simplifies creating the permissions string for an Azure Storage blob SAS.
// Initialize an instance of this type and then call its String method to set BlobSignatureValues' Permissions field.
type DirectoryPermissions struct {
	Read, Add, Create, Write, Delete, List, Move bool
	Execute, Ownership, Permissions              bool
}

// String produces the SAS permissions string for an Azure Storage blob.
// Call this method to set BlobSignatureValues' Permissions field.
func (p *DirectoryPermissions) String() string {
	var b bytes.Buffer
	if p.Read {
		b.WriteRune('r')
	}
	if p.Add {
		b.WriteRune('a')
	}
	if p.Create {
		b.WriteRune('c')
	}
	if p.Write {
		b.WriteRune('w')
	}
	if p.Delete {
		b.WriteRune('d')
	}
	if p.List {
		b.WriteRune('l')
	}
	if p.Move {
		b.WriteRune('m')
	}
	if p.Execute {
		b.WriteRune('e')
	}
	if p.Ownership {
		b.WriteRune('o')
	}
	if p.Permissions {
		b.WriteRune('p')
	}
	return b.String()
}

// Since this is internal we can just always convert to FilePermissions to avoid some duplication here
func parsePathPermissions(s string) (FilePermissions, error) {
	p := FilePermissions{} // Clear the flags
	for _, r := range s {
		switch r {
		case 'r':
			p.Read = true
		case 'a':
			p.Add = true
		case 'c':
			p.Create = true
		case 'w':
			p.Write = true
		case 'd':
			p.Delete = true
		case 'l':
			p.List = true
		case 'm':
			p.Move = true
		case 'e':
			p.Execute = true
		case 'o':
			p.Ownership = true
		case 'p':
			p.Permissions = true
		default:
			return FilePermissions{}, fmt.Errorf("invalid permission: '%v'", r)
		}
	}
	return p, nil
}
