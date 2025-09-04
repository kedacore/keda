//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package file

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/streaming"
	"github.com/Azure/azure-sdk-for-go/sdk/internal/log"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/datalakeerror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/base"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/exported"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/generated"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/generated_blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/path"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/shared"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/sas"
)

// ClientOptions contains the optional parameters when creating a Client.
type ClientOptions base.ClientOptions

// Client represents a URL to the Azure Datalake Storage service.
type Client base.CompositeClient[generated.PathClient, generated_blob.BlobClient, blockblob.Client]

// NewClient creates an instance of Client with the specified values.
//   - fileURL - the URL of the blob e.g. https://<account>.dfs.core.windows.net/fs/file.txt
//   - cred - an Azure AD credential, typically obtained via the azidentity module
//   - options - client options; pass nil to accept the default values
func NewClient(fileURL string, cred azcore.TokenCredential, options *ClientOptions) (*Client, error) {
	blobURL, fileURL := shared.GetURLs(fileURL)
	audience := base.GetAudience((*base.ClientOptions)(options))
	conOptions := shared.GetClientOptions(options)
	authPolicy := shared.NewStorageChallengePolicy(cred, audience, conOptions.InsecureAllowCredentialWithHTTP)
	plOpts := runtime.PipelineOptions{
		PerRetry: []policy.Policy{authPolicy},
	}
	base.SetPipelineOptions((*base.ClientOptions)(conOptions), &plOpts)

	azClient, err := azcore.NewClient(exported.ModuleName, exported.ModuleVersion, plOpts, &conOptions.ClientOptions)
	if err != nil {
		return nil, err
	}

	if options == nil {
		options = &ClientOptions{}
	}
	perCallPolicies := []policy.Policy{shared.NewIncludeBlobResponsePolicy()}
	if options.ClientOptions.PerCallPolicies != nil {
		perCallPolicies = append(perCallPolicies, options.ClientOptions.PerCallPolicies...)
	}
	options.ClientOptions.PerCallPolicies = perCallPolicies
	blobClientOpts := blockblob.ClientOptions{
		ClientOptions: options.ClientOptions,
	}
	blobClient, _ := blockblob.NewClient(blobURL, cred, &blobClientOpts)
	fileClient := base.NewPathClient(fileURL, blobURL, blobClient, azClient, nil, &cred, (*base.ClientOptions)(conOptions))

	return (*Client)(fileClient), nil
}

// NewClientWithNoCredential creates an instance of Client with the specified values.
// This is used to anonymously access a storage account or with a shared access signature (SAS) token.
//   - fileURL - the URL of the storage account e.g. https://<account>.dfs.core.windows.net/fs/file.txt?<sas token>
//   - options - client options; pass nil to accept the default values
func NewClientWithNoCredential(fileURL string, options *ClientOptions) (*Client, error) {
	blobURL, fileURL := shared.GetURLs(fileURL)

	conOptions := shared.GetClientOptions(options)
	plOpts := runtime.PipelineOptions{}
	base.SetPipelineOptions((*base.ClientOptions)(conOptions), &plOpts)

	azClient, err := azcore.NewClient(exported.ModuleName, exported.ModuleVersion, plOpts, &conOptions.ClientOptions)
	if err != nil {
		return nil, err
	}

	if options == nil {
		options = &ClientOptions{}
	}
	perCallPolicies := []policy.Policy{shared.NewIncludeBlobResponsePolicy()}
	if options.ClientOptions.PerCallPolicies != nil {
		perCallPolicies = append(perCallPolicies, options.ClientOptions.PerCallPolicies...)
	}
	options.ClientOptions.PerCallPolicies = perCallPolicies
	blobClientOpts := blockblob.ClientOptions{
		ClientOptions: options.ClientOptions,
	}
	blobClient, _ := blockblob.NewClientWithNoCredential(blobURL, &blobClientOpts)
	fileClient := base.NewPathClient(fileURL, blobURL, blobClient, azClient, nil, nil, (*base.ClientOptions)(conOptions))

	return (*Client)(fileClient), nil
}

// NewClientWithSharedKeyCredential creates an instance of Client with the specified values.
//   - fileURL - the URL of the storage account e.g. https://<account>.dfs.core.windows.net/fs/file.txt
//   - cred - a SharedKeyCredential created with the matching storage account and access key
//   - options - client options; pass nil to accept the default values
func NewClientWithSharedKeyCredential(fileURL string, cred *SharedKeyCredential, options *ClientOptions) (*Client, error) {
	blobURL, fileURL := shared.GetURLs(fileURL)

	authPolicy := exported.NewSharedKeyCredPolicy(cred)
	conOptions := shared.GetClientOptions(options)
	plOpts := runtime.PipelineOptions{
		PerRetry: []policy.Policy{authPolicy},
	}
	base.SetPipelineOptions((*base.ClientOptions)(conOptions), &plOpts)

	azClient, err := azcore.NewClient(exported.ModuleName, exported.ModuleVersion, plOpts, &conOptions.ClientOptions)
	if err != nil {
		return nil, err
	}

	if options == nil {
		options = &ClientOptions{}
	}
	perCallPolicies := []policy.Policy{shared.NewIncludeBlobResponsePolicy()}
	if options.ClientOptions.PerCallPolicies != nil {
		perCallPolicies = append(perCallPolicies, options.ClientOptions.PerCallPolicies...)
	}
	options.ClientOptions.PerCallPolicies = perCallPolicies
	blobClientOpts := blockblob.ClientOptions{
		ClientOptions: options.ClientOptions,
	}
	blobSharedKey, err := exported.ConvertToBlobSharedKey(cred)
	if err != nil {
		return nil, err
	}
	blobClient, _ := blockblob.NewClientWithSharedKeyCredential(blobURL, blobSharedKey, &blobClientOpts)
	fileClient := base.NewPathClient(fileURL, blobURL, blobClient, azClient, cred, nil, (*base.ClientOptions)(conOptions))

	return (*Client)(fileClient), nil
}

// NewClientFromConnectionString creates an instance of Client with the specified values.
//   - connectionString - a connection string for the desired storage account
//   - options - client options; pass nil to accept the default values
func NewClientFromConnectionString(connectionString string, filePath, fsName string, options *ClientOptions) (*Client, error) {
	parsed, err := shared.ParseConnectionString(connectionString)
	if err != nil {
		return nil, err
	}

	filePath = strings.ReplaceAll(filePath, "\\", "/")
	parsed.ServiceURL = runtime.JoinPaths(parsed.ServiceURL, fsName, filePath)

	if parsed.AccountKey != "" && parsed.AccountName != "" {
		credential, err := exported.NewSharedKeyCredential(parsed.AccountName, parsed.AccountKey)
		if err != nil {
			return nil, err
		}
		return NewClientWithSharedKeyCredential(parsed.ServiceURL, credential, options)
	}

	return NewClientWithNoCredential(parsed.ServiceURL, options)
}

func (f *Client) generatedFileClientWithDFS() *generated.PathClient {
	// base.SharedKeyComposite((*base.CompositeClient[generated.BlobClient, generated.BlockBlobClient])(bb))
	dirClientWithDFS, _, _ := base.InnerClients((*base.CompositeClient[generated.PathClient, generated_blob.BlobClient, blockblob.Client])(f))
	return dirClientWithDFS
}

func (f *Client) generatedFileClientWithBlob() *generated_blob.BlobClient {
	_, dirClientWithBlob, _ := base.InnerClients((*base.CompositeClient[generated.PathClient, generated_blob.BlobClient, blockblob.Client])(f))
	return dirClientWithBlob
}

func (f *Client) blobClient() *blockblob.Client {
	_, _, blobClient := base.InnerClients((*base.CompositeClient[generated.PathClient, generated_blob.BlobClient, blockblob.Client])(f))
	return blobClient
}

func (f *Client) sharedKey() *exported.SharedKeyCredential {
	return base.SharedKeyComposite((*base.CompositeClient[generated.PathClient, generated_blob.BlobClient, blockblob.Client])(f))
}

func (f *Client) identityCredential() *azcore.TokenCredential {
	return base.IdentityCredentialComposite((*base.CompositeClient[generated.PathClient, generated_blob.BlobClient, blockblob.Client])(f))
}

func (f *Client) getClientOptions() *base.ClientOptions {
	return base.GetCompositeClientOptions((*base.CompositeClient[generated.PathClient, generated_blob.BlobClient, blockblob.Client])(f))
}

// DFSURL returns the URL endpoint used by the Client object.
func (f *Client) DFSURL() string {
	return f.generatedFileClientWithDFS().Endpoint()
}

// BlobURL returns the URL endpoint used by the Client object.
func (f *Client) BlobURL() string {
	return f.generatedFileClientWithBlob().Endpoint()
}

// Create creates a new file.
func (f *Client) Create(ctx context.Context, options *CreateOptions) (CreateResponse, error) {
	lac, mac, httpHeaders, createOpts, cpkOpts := options.format()
	resp, err := f.generatedFileClientWithDFS().Create(ctx, createOpts, httpHeaders, lac, mac, nil, cpkOpts)
	err = exported.ConvertToDFSError(err)
	return resp, err
}

// Delete deletes a file.
func (f *Client) Delete(ctx context.Context, options *DeleteOptions) (DeleteResponse, error) {
	lac, mac, deleteOpts := path.FormatDeleteOptions(options, false)
	resp, err := f.generatedFileClientWithDFS().Delete(ctx, deleteOpts, lac, mac)
	err = exported.ConvertToDFSError(err)
	return resp, err
}

// GetProperties gets the properties of a file.
func (f *Client) GetProperties(ctx context.Context, options *GetPropertiesOptions) (GetPropertiesResponse, error) {
	opts := path.FormatGetPropertiesOptions(options)
	var respFromCtx *http.Response
	ctxWithResp := shared.WithCaptureBlobResponse(ctx, &respFromCtx)
	resp, err := f.blobClient().GetProperties(ctxWithResp, opts)
	if err != nil {
		err = exported.ConvertToDFSError(err)
		return GetPropertiesResponse{}, err
	}
	newResp := path.FormatGetPropertiesResponse(&resp, respFromCtx)
	return newResp, err
}

// Rename renames a file. The original file will no longer exist and the client will be stale.
func (f *Client) Rename(ctx context.Context, destinationPath string, options *RenameOptions) (RenameResponse, error) {
	var newBlobClient *blockblob.Client
	destinationPath = strings.Trim(strings.TrimSpace(destinationPath), "/")
	if len(destinationPath) == 0 {
		return RenameResponse{}, errors.New("destination path must not be empty")
	}
	urlParts, err := sas.ParseURL(f.DFSURL())
	if err != nil {
		return RenameResponse{}, err
	}

	oldPath, err := url.Parse(f.DFSURL())
	if err != nil {
		return RenameResponse{}, err
	}
	srcParts := strings.Split(f.DFSURL(), "?")
	newSrcPath := oldPath.Path
	newSrcQuery := ""
	if len(srcParts) == 2 {
		newSrcQuery = srcParts[1]
	}
	if len(newSrcQuery) > 0 {
		newSrcPath = newSrcPath + "?" + newSrcQuery
	}

	destParts := strings.Split(destinationPath, "?")
	newDestPath := destParts[0]
	newDestQuery := ""
	if len(destParts) == 2 {
		newDestQuery = destParts[1]
	}

	urlParts.PathName = newDestPath
	newPathURL := urlParts.String()
	// replace the query part if it is present in destination path
	if len(newDestQuery) > 0 {
		newPathURL = strings.Split(newPathURL, "?")[0] + "?" + newDestQuery
	}
	newBlobURL, _ := shared.GetURLs(newPathURL)
	lac, mac, smac, createOpts, cpkOpts := path.FormatRenameOptions(options, newSrcPath)

	if f.identityCredential() != nil {
		newBlobClient, err = blockblob.NewClient(newBlobURL, *f.identityCredential(), nil)
	} else if f.sharedKey() != nil {
		blobSharedKey, _ := exported.ConvertToBlobSharedKey(f.sharedKey())
		newBlobClient, err = blockblob.NewClientWithSharedKeyCredential(newBlobURL, blobSharedKey, nil)
	} else {
		newBlobClient, err = blockblob.NewClientWithNoCredential(newBlobURL, nil)
	}

	if err != nil {
		return RenameResponse{}, exported.ConvertToDFSError(err)
	}
	newFileClient := (*Client)(base.NewPathClient(newPathURL, newBlobURL, newBlobClient, f.generatedFileClientWithDFS().InternalClient().WithClientName(exported.ModuleName), f.sharedKey(), f.identityCredential(), f.getClientOptions()))
	resp, err := newFileClient.generatedFileClientWithDFS().Create(ctx, createOpts, nil, lac, mac, smac, cpkOpts)
	return path.FormatRenameResponse(&resp), exported.ConvertToDFSError(err)
}

// SetExpiry operation sets an expiry time on an existing file (blob2).
func (f *Client) SetExpiry(ctx context.Context, expiryValues SetExpiryValues, o *SetExpiryOptions) (SetExpiryResponse, error) {
	if reflect.ValueOf(expiryValues).IsZero() {
		expiryValues.ExpiryType = SetExpiryTypeNeverExpire
	}
	opts := &generated_blob.BlobClientSetExpiryOptions{}
	if expiryValues.ExpiryType != SetExpiryTypeNeverExpire {
		opts.ExpiresOn = &expiryValues.ExpiresOn
	}
	resp, err := f.generatedFileClientWithBlob().SetExpiry(ctx, expiryValues.ExpiryType, opts)
	err = exported.ConvertToDFSError(err)
	return resp, err
}

// SetAccessControl sets the owner, owning group, and permissions for a file.
func (f *Client) SetAccessControl(ctx context.Context, options *SetAccessControlOptions) (SetAccessControlResponse, error) {
	opts, lac, mac, err := path.FormatSetAccessControlOptions(options)
	if err != nil {
		return SetAccessControlResponse{}, err
	}
	resp, err := f.generatedFileClientWithDFS().SetAccessControl(ctx, opts, lac, mac)
	err = exported.ConvertToDFSError(err)
	return resp, err
}

// UpdateAccessControl updates the owner, owning group, and permissions for a file.
func (f *Client) UpdateAccessControl(ctx context.Context, acl string, options *UpdateAccessControlOptions) (UpdateAccessControlResponse, error) {
	opts, mode := options.format(acl)
	resp, err := f.generatedFileClientWithDFS().SetAccessControlRecursive(ctx, mode, opts)
	err = exported.ConvertToDFSError(err)
	return resp, err
}

// GetAccessControl gets the owner, owning group, and permissions for a file.
func (f *Client) GetAccessControl(ctx context.Context, options *GetAccessControlOptions) (GetAccessControlResponse, error) {
	opts, lac, mac := path.FormatGetAccessControlOptions(options)
	resp, err := f.generatedFileClientWithDFS().GetProperties(ctx, opts, lac, mac)
	err = exported.ConvertToDFSError(err)
	return resp, err
}

// RemoveAccessControl removes the owner, owning group, and permissions for a file.
func (f *Client) RemoveAccessControl(ctx context.Context, acl string, options *RemoveAccessControlOptions) (RemoveAccessControlResponse, error) {
	opts, mode := options.format(acl)
	resp, err := f.generatedFileClientWithDFS().SetAccessControlRecursive(ctx, mode, opts)
	err = exported.ConvertToDFSError(err)
	return resp, err
}

// SetMetadata sets the metadata for a file.
func (f *Client) SetMetadata(ctx context.Context, metadata map[string]*string, options *SetMetadataOptions) (SetMetadataResponse, error) {
	opts := path.FormatSetMetadataOptions(options)
	resp, err := f.blobClient().SetMetadata(ctx, metadata, opts)
	err = exported.ConvertToDFSError(err)
	return resp, err
}

// SetHTTPHeaders sets the HTTP headers for a file.
func (f *Client) SetHTTPHeaders(ctx context.Context, httpHeaders HTTPHeaders, options *SetHTTPHeadersOptions) (SetHTTPHeadersResponse, error) {
	opts, blobHTTPHeaders := path.FormatSetHTTPHeadersOptions(options, httpHeaders)
	resp, err := f.blobClient().SetHTTPHeaders(ctx, blobHTTPHeaders, opts)
	newResp := SetHTTPHeadersResponse{}
	path.FormatSetHTTPHeadersResponse(&newResp, &resp)
	err = exported.ConvertToDFSError(err)
	return newResp, err
}

// GetSASURL is a convenience method for generating a SAS token for the currently pointed at file.
// It can only be used if the credential supplied during creation was a SharedKeyCredential.
func (f *Client) GetSASURL(permissions sas.FilePermissions, expiry time.Time, o *GetSASURLOptions) (string, error) {
	if f.sharedKey() == nil {
		return "", datalakeerror.MissingSharedKeyCredential
	}

	urlParts, err := sas.ParseURL(f.BlobURL())
	err = exported.ConvertToDFSError(err)
	if err != nil {
		return "", err
	}

	st := path.FormatGetSASURLOptions(o)

	qps, err := sas.DatalakeSignatureValues{
		FilePath:       urlParts.PathName,
		FileSystemName: urlParts.FileSystemName,
		Version:        sas.Version,
		Permissions:    permissions.String(),
		StartTime:      st,
		ExpiryTime:     expiry.UTC(),
	}.SignWithSharedKey(f.sharedKey())

	err = exported.ConvertToDFSError(err)
	if err != nil {
		return "", err
	}

	endpoint := f.BlobURL() + "?" + qps.Encode()

	return endpoint, nil
}

// AppendData appends data to existing file with a given offset.
func (f *Client) AppendData(ctx context.Context, offset int64, body io.ReadSeekCloser, options *AppendDataOptions) (AppendDataResponse, error) {
	appendDataOptions, leaseAccessConditions, cpkInfo, err := options.format(offset, body)
	if err != nil {
		return AppendDataResponse{}, err
	}
	resp, err := f.generatedFileClientWithDFS().AppendData(ctx, body, appendDataOptions, nil, leaseAccessConditions, cpkInfo)
	return resp, exported.ConvertToDFSError(err)
}

// FlushData commits appended data to file
func (f *Client) FlushData(ctx context.Context, offset int64, options *FlushDataOptions) (FlushDataResponse, error) {
	flushDataOpts, modifiedAccessConditions, leaseAccessConditions, httpHeaderOpts, cpkInfoOpts, err := options.format(offset)
	if err != nil {
		return FlushDataResponse{}, err
	}

	resp, err := f.generatedFileClientWithDFS().FlushData(ctx, flushDataOpts, httpHeaderOpts, leaseAccessConditions, modifiedAccessConditions, cpkInfoOpts)
	return resp, exported.ConvertToDFSError(err)
}

// Concurrent Upload Functions -----------------------------------------------------------------------------------------

// uploadFromReader uploads a buffer in chunks to an Azure file.
func (f *Client) uploadFromReader(ctx context.Context, reader io.ReaderAt, actualSize int64, o *uploadFromReaderOptions) error {
	if actualSize > MaxFileSize {
		return errors.New("buffer is too large to upload to a file")
	}
	if o.ChunkSize == 0 {
		o.ChunkSize = MaxAppendBytes
	}

	if log.Should(exported.EventUpload) {
		urlParts, err := azdatalake.ParseURL(f.DFSURL())
		if err == nil {
			log.Writef(exported.EventUpload, "file name %s actual size %v chunk-size %v chunk-count %v",
				urlParts.PathName, actualSize, o.ChunkSize, ((actualSize-1)/o.ChunkSize)+1)
		}
	}

	if o.EncryptionContext != nil {
		_, err := f.Create(ctx, &CreateOptions{EncryptionContext: o.EncryptionContext})
		if err != nil {
			return err
		}
	}

	progress := int64(0)
	progressLock := &sync.Mutex{}

	err := shared.DoBatchTransfer(ctx, &shared.BatchTransferOptions{
		OperationName: "uploadFromReader",
		TransferSize:  actualSize,
		ChunkSize:     o.ChunkSize,
		Concurrency:   o.Concurrency,
		Operation: func(ctx context.Context, offset int64, chunkSize int64) error {
			// This function is called once per file range.
			// It is passed this file's offset within the buffer and its count of bytes
			// Prepare to read the proper range/section of the buffer
			if chunkSize < o.ChunkSize {
				// this is the last file range.  Its actual size might be less
				// than the calculated size due to rounding up of the payload
				// size to fit in a whole number of chunks.
				chunkSize = actualSize - offset
			}
			var body io.ReadSeeker = io.NewSectionReader(reader, offset, chunkSize)
			if o.Progress != nil {
				chunkProgress := int64(0)
				body = streaming.NewRequestProgress(streaming.NopCloser(body),
					func(bytesTransferred int64) {
						diff := bytesTransferred - chunkProgress
						chunkProgress = bytesTransferred
						progressLock.Lock() // 1 goroutine at a time gets progress report
						progress += diff
						o.Progress(progress)
						progressLock.Unlock()
					})
			}

			uploadRangeOptions := o.getAppendDataOptions()
			_, err := f.AppendData(ctx, offset, streaming.NopCloser(body), uploadRangeOptions)
			return exported.ConvertToDFSError(err)
		},
	})

	if err != nil {
		if o.EncryptionContext != nil {
			_, err2 := f.Delete(ctx, nil)
			if err2 != nil {
				return exported.ConvertToDFSError(err2)
			}
		}
		return exported.ConvertToDFSError(err)
	}
	// All appends were successful, call to flush
	flushOpts := o.getFlushDataOptions()
	_, err = f.FlushData(ctx, actualSize, flushOpts)
	return exported.ConvertToDFSError(err)
}

// UploadBuffer uploads a buffer in chunks to a file.
func (f *Client) UploadBuffer(ctx context.Context, buffer []byte, options *UploadBufferOptions) error {
	uploadOptions := uploadFromReaderOptions{}
	if options != nil {
		uploadOptions = *options
	}
	return exported.ConvertToDFSError(f.uploadFromReader(ctx, bytes.NewReader(buffer), int64(len(buffer)), &uploadOptions))
}

// UploadFile uploads a file in chunks to a file.
func (f *Client) UploadFile(ctx context.Context, file *os.File, options *UploadFileOptions) error {
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	uploadOptions := uploadFromReaderOptions{}
	if options != nil {
		uploadOptions = *options
	}
	return exported.ConvertToDFSError(f.uploadFromReader(ctx, file, stat.Size(), &uploadOptions))
}

// UploadStream copies the file held in io.Reader to the file at fileClient.
// A Context deadline or cancellation will cause this to error.
func (f *Client) UploadStream(ctx context.Context, body io.Reader, options *UploadStreamOptions) error {
	if options == nil {
		options = &UploadStreamOptions{}
	}

	if options.EncryptionContext != nil {
		_, err := f.Create(ctx, &CreateOptions{EncryptionContext: options.EncryptionContext})
		if err != nil {
			return err
		}
	}
	err := copyFromReader(ctx, body, f, *options, newMMBPool)

	if err != nil && options.EncryptionContext != nil {
		_, err2 := f.Delete(ctx, nil)
		if err2 != nil {
			return exported.ConvertToDFSError(err2)
		}
	}
	return exported.ConvertToDFSError(err)
}

// DownloadStream reads a range of bytes from a file. The response also includes the file's properties and metadata.
// For more information, see https://docs.microsoft.com/rest/api/storageservices/get-blob.
func (f *Client) DownloadStream(ctx context.Context, o *DownloadStreamOptions) (DownloadStreamResponse, error) {
	if o == nil {
		o = &DownloadStreamOptions{}
	}
	opts := o.format()
	var respFromCtx *http.Response
	ctxWithResp := shared.WithCaptureBlobResponse(ctx, &respFromCtx)
	resp, err := f.blobClient().DownloadStream(ctxWithResp, opts)
	if err != nil {
		return DownloadStreamResponse{}, exported.ConvertToDFSError(err)
	}
	newResp := FormatDownloadStreamResponse(&resp, respFromCtx)
	fullResp := DownloadStreamResponse{
		client:           f,
		DownloadResponse: newResp,
		getInfo:          httpGetterInfo{Range: o.Range, ETag: newResp.ETag},
		cpkInfo:          o.CPKInfo,
		cpkScope:         o.CPKScopeInfo,
	}

	return fullResp, nil
}

// DownloadBuffer downloads an Azure file to a buffer with parallel.
func (f *Client) DownloadBuffer(ctx context.Context, buffer []byte, o *DownloadBufferOptions) (int64, error) {
	opts := o.format()
	val, err := f.blobClient().DownloadBuffer(ctx, shared.NewBytesWriter(buffer), opts)
	return val, exported.ConvertToDFSError(err)
}

// DownloadFile downloads a datalake file to a local file.
// The file would be truncated if the size doesn't match.
func (f *Client) DownloadFile(ctx context.Context, file *os.File, o *DownloadFileOptions) (int64, error) {
	opts := o.format()
	val, err := f.blobClient().DownloadFile(ctx, file, opts)
	return val, exported.ConvertToDFSError(err)
}

// TODO: Undelete()
