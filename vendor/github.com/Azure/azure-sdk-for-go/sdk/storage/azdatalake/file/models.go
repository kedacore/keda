//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package file

import (
	"errors"
	"io"
	"reflect"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/exported"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/generated"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/path"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/shared"
)

const (
	_1MiB      = 1024 * 1024
	CountToEnd = 0

	// MaxAppendBytes indicates the maximum number of bytes that can be updated in a call to Client.AppendData.
	MaxAppendBytes = 100 * 1024 * 1024 // 100iB

	// MaxFileSize indicates the maximum size of the file allowed.
	MaxFileSize = 4 * 1024 * 1024 * 1024 * 1024 // 4 TiB
)

// CreateOptions contains the optional parameters when calling the Create operation.
type CreateOptions struct {
	// AccessConditions contains parameters for accessing the file.
	AccessConditions *AccessConditions
	// CPKInfo contains a group of parameters for client provided encryption key.
	CPKInfo *CPKInfo
	// HTTPHeaders contains the HTTP headers for path operations.
	HTTPHeaders *HTTPHeaders
	// Expiry specifies the type and time of expiry for the file.
	Expiry CreateExpiryValues
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
	// EncryptionContext stores non-encrypted data that can be used to derive the customer-provided key for a file.
	EncryptionContext *string
}

// CreateExpiryValues describes when a newly created file should expire.
// A zero-value indicates the file has no expiration date.
type CreateExpiryValues struct {
	// ExpiryType indicates how the value of ExpiresOn should be interpreted (absolute, relative to now, etc).
	ExpiryType CreateExpiryType

	// ExpiresOn contains the time the file should expire.
	// The value will either be an absolute UTC time in RFC1123 format or an integer expressing a number of milliseconds.
	// NOTE: when ExpiryType is CreateExpiryTypeNeverExpire, this value is ignored.
	ExpiresOn string
}

func (o *CreateOptions) format() (*generated.LeaseAccessConditions, *generated.ModifiedAccessConditions, *generated.PathHTTPHeaders, *generated.PathClientCreateOptions, *generated.CPKInfo) {
	resource := generated.PathResourceTypeFile
	createOpts := &generated.PathClientCreateOptions{
		Resource: &resource,
	}
	if o == nil {
		return nil, nil, nil, createOpts, nil
	}
	leaseAccessConditions, modifiedAccessConditions := exported.FormatPathAccessConditions(o.AccessConditions)
	if !reflect.ValueOf(o.Expiry).IsZero() {
		createOpts.ExpiryOptions = &o.Expiry.ExpiryType
		if o.Expiry.ExpiryType != CreateExpiryTypeNeverExpire {
			createOpts.ExpiresOn = &o.Expiry.ExpiresOn
		}
	}
	createOpts.ACL = o.ACL
	createOpts.Group = o.Group
	createOpts.Owner = o.Owner
	createOpts.Umask = o.Umask
	createOpts.Permissions = o.Permissions
	createOpts.ProposedLeaseID = o.ProposedLeaseID
	createOpts.LeaseDuration = o.LeaseDuration
	createOpts.EncryptionContext = o.EncryptionContext

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

// UpdateAccessControlOptions contains the optional parameters when calling the UpdateAccessControlRecursive operation.
type UpdateAccessControlOptions struct {
	// placeholder
}

func (o *UpdateAccessControlOptions) format(acl string) (*generated.PathClientSetAccessControlRecursiveOptions, generated.PathSetAccessControlRecursiveMode) {
	mode := generated.PathSetAccessControlRecursiveModeModify
	return &generated.PathClientSetAccessControlRecursiveOptions{
		ACL: &acl,
	}, mode
}

// RemoveAccessControlOptions contains the optional parameters when calling the RemoveAccessControlRecursive operation.
type RemoveAccessControlOptions struct {
	// placeholder
}

func (o *RemoveAccessControlOptions) format(acl string) (*generated.PathClientSetAccessControlRecursiveOptions, generated.PathSetAccessControlRecursiveMode) {
	mode := generated.PathSetAccessControlRecursiveModeRemove
	return &generated.PathClientSetAccessControlRecursiveOptions{
		ACL: &acl,
	}, mode
}

// ===================================== PATH IMPORTS ===========================================

// uploadFromReaderOptions identifies options used by the UploadBuffer and UploadFile functions.
type uploadFromReaderOptions struct {
	// ChunkSize specifies the chunk size to use in bytes; the default (and maximum size) is MaxAppendBytes.
	ChunkSize int64
	// Progress is a function that is invoked periodically as bytes are sent to the FileClient.
	// Note that the progress reporting is not always increasing; it can go down when retrying a request.
	Progress func(bytesTransferred int64)
	// Concurrency indicates the maximum number of chunks to upload in parallel (default is 5)
	Concurrency uint16
	// AccessConditions contains optional parameters to access leased entity.
	AccessConditions *AccessConditions
	// HTTPHeaders contains the optional path HTTP headers to set when the file is created.
	HTTPHeaders *HTTPHeaders
	// CPKInfo contains optional parameters to perform encryption using customer-provided key.
	CPKInfo *CPKInfo
	// EncryptionContext contains the information returned from the x-ms-encryption-context header response.
	EncryptionContext *string
}

// UploadStreamOptions provides set of configurations for Client.UploadStream operation.
type UploadStreamOptions struct {
	// ChunkSize specifies the chunk size to use in bytes; the default (and maximum size) is MaxAppendBytes.
	ChunkSize int64
	// Concurrency indicates the maximum number of chunks to upload in parallel (default is 5)
	Concurrency uint16
	// AccessConditions contains optional parameters to access leased entity.
	AccessConditions *AccessConditions
	// HTTPHeaders contains the optional path HTTP headers to set when the file is created.
	HTTPHeaders *HTTPHeaders
	// CPKInfo contains optional parameters to perform encryption using customer-provided key.
	CPKInfo *CPKInfo
	// EncryptionContext contains the information returned from the x-ms-encryption-context header response.
	EncryptionContext *string
}

// UploadBufferOptions provides set of configurations for Client.UploadBuffer operation.
type UploadBufferOptions = uploadFromReaderOptions

// UploadFileOptions provides set of configurations for Client.UploadFile operation.
type UploadFileOptions = uploadFromReaderOptions

// FlushDataOptions contains the optional parameters for the Client.FlushData method.
type FlushDataOptions struct {
	// AccessConditions contains parameters for accessing the file.
	AccessConditions *AccessConditions
	// CPKInfo contains optional parameters to perform encryption using customer-provided key.
	CPKInfo *CPKInfo
	// HTTPHeaders contains the HTTP headers for path operations.
	HTTPHeaders *HTTPHeaders
	// Close This event has a property indicating whether this is the final change to distinguish the
	// difference between an intermediate flush to a file stream and the final close of a file stream.
	Close *bool
	// RetainUncommittedData if "true", uncommitted data is retained after the flush operation
	// completes, otherwise, the uncommitted data is deleted after the flush operation.
	RetainUncommittedData *bool
	// LeaseAction Describes actions that can be performed on a lease.
	LeaseAction *LeaseAction
	// LeaseDuration specifies the duration of the lease, in seconds, or negative one
	// (-1) for a lease that never expires. A non-infinite lease can be between 15 and 60 seconds.
	LeaseDuration *int64
	// ProposedLeaseID specifies the proposed lease ID for the file.
	ProposedLeaseID *string
}

func (o *FlushDataOptions) format(offset int64) (*generated.PathClientFlushDataOptions, *generated.ModifiedAccessConditions, *generated.LeaseAccessConditions, *generated.PathHTTPHeaders, *generated.CPKInfo, error) {
	defaultRetainUncommitted := false
	defaultClose := false
	contentLength := int64(0)

	var httpHeaderOpts *generated.PathHTTPHeaders
	var leaseAccessConditions *generated.LeaseAccessConditions
	var modifiedAccessConditions *generated.ModifiedAccessConditions
	var cpkInfoOpts *generated.CPKInfo
	flushDataOpts := &generated.PathClientFlushDataOptions{ContentLength: &contentLength, Position: &offset}

	if o == nil {
		flushDataOpts.RetainUncommittedData = &defaultRetainUncommitted
		flushDataOpts.Close = &defaultClose
		return flushDataOpts, nil, nil, nil, nil, nil
	}

	if o != nil {
		if o.RetainUncommittedData == nil {
			flushDataOpts.RetainUncommittedData = &defaultRetainUncommitted
		} else {
			flushDataOpts.RetainUncommittedData = o.RetainUncommittedData
		}
		if o.Close == nil {
			flushDataOpts.Close = &defaultClose
		} else {
			flushDataOpts.Close = o.Close
		}
		leaseAccessConditions, modifiedAccessConditions = exported.FormatPathAccessConditions(o.AccessConditions)
		if o.HTTPHeaders != nil {
			httpHeaderOpts = &generated.PathHTTPHeaders{}
			httpHeaderOpts.ContentMD5 = o.HTTPHeaders.ContentMD5
			httpHeaderOpts.ContentType = o.HTTPHeaders.ContentType
			httpHeaderOpts.CacheControl = o.HTTPHeaders.CacheControl
			httpHeaderOpts.ContentDisposition = o.HTTPHeaders.ContentDisposition
			httpHeaderOpts.ContentEncoding = o.HTTPHeaders.ContentEncoding
		}
		if o.CPKInfo != nil {
			cpkInfoOpts = &generated.CPKInfo{}
			cpkInfoOpts.EncryptionKey = o.CPKInfo.EncryptionKey
			cpkInfoOpts.EncryptionKeySHA256 = o.CPKInfo.EncryptionKeySHA256
			cpkInfoOpts.EncryptionAlgorithm = o.CPKInfo.EncryptionAlgorithm
		}
		flushDataOpts.LeaseAction = o.LeaseAction
		flushDataOpts.LeaseDuration = o.LeaseDuration
		flushDataOpts.ProposedLeaseID = o.ProposedLeaseID
	}
	return flushDataOpts, modifiedAccessConditions, leaseAccessConditions, httpHeaderOpts, cpkInfoOpts, nil
}

// AppendDataOptions contains the optional parameters for the Client.AppendData method.
type AppendDataOptions struct {
	// TransactionalValidation specifies the transfer validation type to use.
	// The default is nil (no transfer validation).
	TransactionalValidation TransferValidationType
	// LeaseAccessConditions contains optional parameters to access leased entity.
	LeaseAccessConditions *LeaseAccessConditions
	// LeaseAction describes actions that can be performed on a lease.
	LeaseAction *LeaseAction
	// LeaseDuration specifies the duration of the lease, in seconds, or negative one
	// (-1) for a lease that never expires. A non-infinite lease can be between 15 and 60 seconds.
	LeaseDuration *int64
	// ProposedLeaseID specifies the proposed lease ID for the file.
	ProposedLeaseID *string
	// CPKInfo contains optional parameters to perform encryption using customer-provided key.
	CPKInfo *CPKInfo
	// Flush Optional. If true, the file will be flushed after append.
	Flush *bool
}

func (o *AppendDataOptions) format(offset int64, body io.ReadSeekCloser) (*generated.PathClientAppendDataOptions,
	*generated.LeaseAccessConditions, *generated.CPKInfo, error) {

	if offset < 0 || body == nil {
		return nil, nil, nil, errors.New("invalid argument: offset must be >= 0 and body must not be nil")
	}

	count, err := shared.ValidateSeekableStreamAt0AndGetCount(body)
	if err != nil {
		return nil, nil, nil, err
	}

	if count == 0 {
		return nil, nil, nil, errors.New("invalid argument: body must contain readable data whose size is > 0")
	}

	appendDataOptions := &generated.PathClientAppendDataOptions{}
	httpRange := exported.FormatHTTPRange(HTTPRange{
		Offset: offset,
		Count:  count,
	})
	if httpRange != nil {
		appendDataOptions.Position = &offset
		appendDataOptions.ContentLength = &count
	}

	var leaseAccessConditions *LeaseAccessConditions
	var cpkInfoOpts *generated.CPKInfo

	if o != nil {
		leaseAccessConditions = o.LeaseAccessConditions
		if o.CPKInfo != nil {
			cpkInfoOpts = &generated.CPKInfo{}
			cpkInfoOpts.EncryptionKey = o.CPKInfo.EncryptionKey
			cpkInfoOpts.EncryptionKeySHA256 = o.CPKInfo.EncryptionKeySHA256
			cpkInfoOpts.EncryptionAlgorithm = o.CPKInfo.EncryptionAlgorithm
		}

		appendDataOptions.LeaseAction = o.LeaseAction
		appendDataOptions.LeaseDuration = o.LeaseDuration
		appendDataOptions.ProposedLeaseID = o.ProposedLeaseID
		appendDataOptions.Flush = o.Flush

		if o.TransactionalValidation != nil {
			_, err = o.TransactionalValidation.Apply(body, appendDataOptions)
			if err != nil {
				return nil, nil, nil, err
			}
		}
	}

	return appendDataOptions, leaseAccessConditions, cpkInfoOpts, nil
}

func (u *UploadStreamOptions) setDefaults() {
	if u.Concurrency == 0 {
		u.Concurrency = 1
	}

	if u.ChunkSize < _1MiB {
		u.ChunkSize = _1MiB
	}
}

func (u *uploadFromReaderOptions) getAppendDataOptions() *AppendDataOptions {
	if u == nil {
		return nil
	}
	leaseAccessConditions, _ := exported.FormatPathAccessConditions(u.AccessConditions)
	return &AppendDataOptions{
		LeaseAccessConditions: leaseAccessConditions,
		CPKInfo:               u.CPKInfo,
	}
}

func (u *uploadFromReaderOptions) getFlushDataOptions() *FlushDataOptions {
	if u == nil {
		return nil
	}
	return &FlushDataOptions{
		AccessConditions: u.AccessConditions,
		HTTPHeaders:      u.HTTPHeaders,
		CPKInfo:          u.CPKInfo,
	}
}

func (u *UploadStreamOptions) getAppendDataOptions() *AppendDataOptions {
	if u == nil {
		return nil
	}
	leaseAccessConditions, _ := exported.FormatPathAccessConditions(u.AccessConditions)
	return &AppendDataOptions{
		LeaseAccessConditions: leaseAccessConditions,
		CPKInfo:               u.CPKInfo,
	}
}

func (u *UploadStreamOptions) getFlushDataOptions() *FlushDataOptions {
	if u == nil {
		return nil
	}
	return &FlushDataOptions{
		AccessConditions: u.AccessConditions,
		HTTPHeaders:      u.HTTPHeaders,
		CPKInfo:          u.CPKInfo,
	}
}

// DownloadStreamOptions contains the optional parameters for the Client.DownloadStream method.
type DownloadStreamOptions struct {
	// When set to true and specified together with the Range, the service returns the MD5 hash for the range, as long as the
	// range is less than or equal to 4 MB in size.
	RangeGetContentMD5 *bool
	// Range specifies a range of bytes.  The default value is all bytes.
	Range *HTTPRange
	// AccessConditions contains parameters for accessing the file.
	AccessConditions *AccessConditions
	// CPKInfo contains optional parameters to perform encryption using customer-provided key.
	CPKInfo *CPKInfo
	// CPKScopeInfo contains a group of parameters for client provided encryption scope.
	CPKScopeInfo *CPKScopeInfo
}

func (o *DownloadStreamOptions) format() *blob.DownloadStreamOptions {
	if o == nil {
		return nil
	}

	downloadStreamOptions := &blob.DownloadStreamOptions{}
	if o.Range != nil {
		downloadStreamOptions.Range = *o.Range
	}
	if o.CPKInfo != nil {
		downloadStreamOptions.CPKInfo = &blob.CPKInfo{
			EncryptionKey:       o.CPKInfo.EncryptionKey,
			EncryptionKeySHA256: o.CPKInfo.EncryptionKeySHA256,
			EncryptionAlgorithm: (*blob.EncryptionAlgorithmType)(o.CPKInfo.EncryptionAlgorithm),
		}
	}

	downloadStreamOptions.RangeGetContentMD5 = o.RangeGetContentMD5
	downloadStreamOptions.AccessConditions = exported.FormatBlobAccessConditions(o.AccessConditions)
	downloadStreamOptions.CPKScopeInfo = o.CPKScopeInfo
	return downloadStreamOptions
}

// DownloadBufferOptions contains the optional parameters for the DownloadBuffer method.
type DownloadBufferOptions struct {
	// Range specifies a range of bytes.  The default value is all bytes.
	Range *HTTPRange
	// ChunkSize specifies the chunk size to use for each parallel download; the default size is 4MB.
	ChunkSize int64
	// Progress is a function that is invoked periodically as bytes are received.
	Progress func(bytesTransferred int64)
	// AccessConditions indicates the access conditions used when making HTTP GET requests against the file.
	AccessConditions *AccessConditions
	// CPKInfo contains a group of parameters for client provided encryption key.
	CPKInfo *CPKInfo
	// CPKScopeInfo contains a group of parameters for client provided encryption scope.
	CPKScopeInfo *CPKScopeInfo
	// Concurrency indicates the maximum number of chunks to download in parallel (0=default).
	Concurrency uint16
	// RetryReaderOptionsPerChunk is used when downloading each chunk.
	RetryReaderOptionsPerChunk *RetryReaderOptions
}

func (o *DownloadBufferOptions) format() *blob.DownloadBufferOptions {
	if o == nil {
		return nil
	}

	downloadBufferOptions := &blob.DownloadBufferOptions{}
	if o.Range != nil {
		downloadBufferOptions.Range = blob.HTTPRange{
			Offset: o.Range.Offset,
			Count:  o.Range.Count,
		}
	}
	if o.CPKInfo != nil {
		downloadBufferOptions.CPKInfo = &blob.CPKInfo{
			EncryptionKey:       o.CPKInfo.EncryptionKey,
			EncryptionKeySHA256: o.CPKInfo.EncryptionKeySHA256,
			EncryptionAlgorithm: (*blob.EncryptionAlgorithmType)(o.CPKInfo.EncryptionAlgorithm),
		}
	}

	downloadBufferOptions.AccessConditions = exported.FormatBlobAccessConditions(o.AccessConditions)
	downloadBufferOptions.CPKScopeInfo = o.CPKScopeInfo
	downloadBufferOptions.BlockSize = o.ChunkSize
	downloadBufferOptions.Progress = o.Progress
	downloadBufferOptions.Concurrency = o.Concurrency
	if o.RetryReaderOptionsPerChunk != nil {
		downloadBufferOptions.RetryReaderOptionsPerBlock.OnFailedRead = o.RetryReaderOptionsPerChunk.OnFailedRead
		downloadBufferOptions.RetryReaderOptionsPerBlock.EarlyCloseAsError = o.RetryReaderOptionsPerChunk.EarlyCloseAsError
		downloadBufferOptions.RetryReaderOptionsPerBlock.MaxRetries = o.RetryReaderOptionsPerChunk.MaxRetries
	}

	return downloadBufferOptions
}

// DownloadFileOptions contains the optional parameters for the Client.DownloadFile method.
type DownloadFileOptions struct {
	// Range specifies a range of bytes.  The default value is all bytes.
	Range *HTTPRange
	// ChunkSize specifies the chunk size to use for each parallel download; the default size is 4MB.
	ChunkSize int64
	// Progress is a function that is invoked periodically as bytes are received.
	Progress func(bytesTransferred int64)
	// AccessConditions indicates the access conditions used when making HTTP GET requests against the file.
	AccessConditions *AccessConditions
	// CPKInfo contains a group of parameters for client provided encryption key.
	CPKInfo *CPKInfo
	// CPKScopeInfo contains a group of parameters for client provided encryption scope.
	CPKScopeInfo *CPKScopeInfo
	// Concurrency indicates the maximum number of chunks to download in parallel. The default value is 5.
	Concurrency uint16
	// RetryReaderOptionsPerChunk is used when downloading each chunk.
	RetryReaderOptionsPerChunk *RetryReaderOptions
}

func (o *DownloadFileOptions) format() *blob.DownloadFileOptions {
	if o == nil {
		return nil
	}

	downloadFileOptions := &blob.DownloadFileOptions{}
	if o.Range != nil {
		downloadFileOptions.Range = blob.HTTPRange{
			Offset: o.Range.Offset,
			Count:  o.Range.Count,
		}
	}
	if o.CPKInfo != nil {
		downloadFileOptions.CPKInfo = &blob.CPKInfo{
			EncryptionKey:       o.CPKInfo.EncryptionKey,
			EncryptionKeySHA256: o.CPKInfo.EncryptionKeySHA256,
			EncryptionAlgorithm: (*blob.EncryptionAlgorithmType)(o.CPKInfo.EncryptionAlgorithm),
		}
	}

	downloadFileOptions.AccessConditions = exported.FormatBlobAccessConditions(o.AccessConditions)
	downloadFileOptions.CPKScopeInfo = o.CPKScopeInfo
	downloadFileOptions.BlockSize = o.ChunkSize
	downloadFileOptions.Progress = o.Progress
	downloadFileOptions.Concurrency = o.Concurrency
	if o.RetryReaderOptionsPerChunk != nil {
		downloadFileOptions.RetryReaderOptionsPerBlock.OnFailedRead = o.RetryReaderOptionsPerChunk.OnFailedRead
		downloadFileOptions.RetryReaderOptionsPerBlock.EarlyCloseAsError = o.RetryReaderOptionsPerChunk.EarlyCloseAsError
		downloadFileOptions.RetryReaderOptionsPerBlock.MaxRetries = o.RetryReaderOptionsPerChunk.MaxRetries
	}

	return downloadFileOptions
}

// SetExpiryValues describes when a file should expire.
// A zero-value indicates the file has no expiration date.
type SetExpiryValues struct {
	// ExpiryType indicates how the value of ExpiresOn should be interpreted (absolute, relative to now, etc).
	ExpiryType SetExpiryType

	// ExpiresOn contains the time the file should expire.
	// The value will either be an absolute UTC time in RFC1123 format or an integer expressing a number of milliseconds.
	// NOTE: when ExpiryType is SetExpiryTypeNeverExpire, this value is ignored.
	ExpiresOn string
}

// ACLFailedEntry contains the failed ACL entry (response model).
type ACLFailedEntry = path.ACLFailedEntry

// SetAccessControlRecursiveResponse contains part of the response data returned by the []OP_AccessControl operations.
type SetAccessControlRecursiveResponse = generated.SetAccessControlRecursiveResponse

// SetExpiryOptions contains the optional parameters for the Client.SetExpiry method.
type SetExpiryOptions struct {
	// placeholder for future options
}

// HTTPRange defines a range of bytes within an HTTP resource, starting at offset and
// ending at offset+count. A zero-value HTTPRange indicates the entire resource. An HTTPRange
// which has an offset and zero value count indicates from the offset to the resource's end.
type HTTPRange = exported.HTTPRange

// ================================= path imports ==================================

// DeleteOptions contains the optional parameters when calling the Delete operation.
type DeleteOptions = path.DeleteOptions

// RenameOptions contains the optional parameters when calling the Rename operation.
type RenameOptions = path.RenameOptions

// GetPropertiesOptions contains the optional parameters for the Client.GetProperties method
type GetPropertiesOptions = path.GetPropertiesOptions

// SetAccessControlOptions contains the optional parameters when calling the SetAccessControl operation.
type SetAccessControlOptions = path.SetAccessControlOptions

// GetAccessControlOptions contains the optional parameters when calling the GetAccessControl operation.
type GetAccessControlOptions = path.GetAccessControlOptions

// CPKInfo contains CPK related information.
type CPKInfo = path.CPKInfo

// GetSASURLOptions contains the optional parameters for the Client.GetSASURL method.
type GetSASURLOptions = path.GetSASURLOptions

// SetHTTPHeadersOptions contains the optional parameters for the Client.SetHTTPHeaders method.
type SetHTTPHeadersOptions = path.SetHTTPHeadersOptions

// HTTPHeaders contains the HTTP headers for path operations.
type HTTPHeaders = path.HTTPHeaders

// SetMetadataOptions provides set of configurations for Set Metadata on path operation
type SetMetadataOptions = path.SetMetadataOptions

// SharedKeyCredential contains an account's name and its primary or secondary key.
type SharedKeyCredential = path.SharedKeyCredential

// AccessConditions identifies file-specific access conditions which you optionally set.
type AccessConditions = path.AccessConditions

// SourceAccessConditions identifies file-specific source access conditions which you optionally set.
type SourceAccessConditions = path.SourceAccessConditions

// LeaseAccessConditions contains optional parameters to access leased entity.
type LeaseAccessConditions = path.LeaseAccessConditions

// ModifiedAccessConditions contains a group of parameters for specifying access conditions.
type ModifiedAccessConditions = path.ModifiedAccessConditions

// SourceModifiedAccessConditions contains a group of parameters for specifying access conditions.
type SourceModifiedAccessConditions = path.SourceModifiedAccessConditions

// CPKScopeInfo contains a group of parameters for the PathClient.SetMetadata method.
type CPKScopeInfo = path.CPKScopeInfo
