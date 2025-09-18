//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package base

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/exported"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/generated"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/generated_blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/shared"
	"strings"
)

// ClientOptions contains the optional parameters when creating a Client.
type ClientOptions struct {
	azcore.ClientOptions
	pipelineOptions *runtime.PipelineOptions
	// Audience to use when requesting tokens for Azure Active Directory authentication.
	// Only has an effect when credential is of type TokenCredential. The value could be
	// https://storage.azure.com/ (default) or https://<account>.blob.core.windows.net.
	Audience string
}

func GetPipelineOptions(clOpts *ClientOptions) *runtime.PipelineOptions {
	return clOpts.pipelineOptions
}

func SetPipelineOptions(clOpts *ClientOptions, plOpts *runtime.PipelineOptions) {
	clOpts.pipelineOptions = plOpts
}

type CompositeClient[T, K, U any] struct {
	// generated client with dfs
	innerT *T
	// generated client with blob
	innerK *K
	// blob client
	innerU       *U
	sharedKey    *exported.SharedKeyCredential
	identityCred *azcore.TokenCredential
	options      *ClientOptions
}

func InnerClients[T, K, U any](client *CompositeClient[T, K, U]) (*T, *K, *U) {
	return client.innerT, client.innerK, client.innerU
}

func SharedKeyComposite[T, K, U any](client *CompositeClient[T, K, U]) *exported.SharedKeyCredential {
	return client.sharedKey
}

func IdentityCredentialComposite[T, K, U any](client *CompositeClient[T, K, U]) *azcore.TokenCredential {
	return client.identityCred
}

func NewFileSystemClient(fsURL string, fsURLWithBlobEndpoint string, client *container.Client, azClient *azcore.Client, sharedKey *exported.SharedKeyCredential, identityCred *azcore.TokenCredential, options *ClientOptions) *CompositeClient[generated.FileSystemClient, generated.FileSystemClient, container.Client] {
	return &CompositeClient[generated.FileSystemClient, generated.FileSystemClient, container.Client]{
		innerT:       generated.NewFileSystemClient(fsURL, azClient),
		innerK:       generated.NewFileSystemClient(fsURLWithBlobEndpoint, azClient),
		sharedKey:    sharedKey,
		identityCred: identityCred,
		innerU:       client,
		options:      options,
	}
}

func NewServiceClient(serviceURL string, serviceURLWithBlobEndpoint string, client *service.Client, azClient *azcore.Client, sharedKey *exported.SharedKeyCredential, identityCred *azcore.TokenCredential, options *ClientOptions) *CompositeClient[generated.ServiceClient, generated_blob.ServiceClient, service.Client] {
	return &CompositeClient[generated.ServiceClient, generated_blob.ServiceClient, service.Client]{
		innerT:       generated.NewServiceClient(serviceURL, azClient),
		innerK:       generated_blob.NewServiceClient(serviceURLWithBlobEndpoint, azClient),
		sharedKey:    sharedKey,
		identityCred: identityCred,
		innerU:       client,
		options:      options,
	}
}

func NewPathClient(pathURL string, pathURLWithBlobEndpoint string, client *blockblob.Client, azClient *azcore.Client, sharedKey *exported.SharedKeyCredential, identityCred *azcore.TokenCredential, options *ClientOptions) *CompositeClient[generated.PathClient, generated_blob.BlobClient, blockblob.Client] {
	return &CompositeClient[generated.PathClient, generated_blob.BlobClient, blockblob.Client]{
		innerT:       generated.NewPathClient(pathURL, azClient),
		innerK:       generated_blob.NewBlobClient(pathURLWithBlobEndpoint, azClient),
		sharedKey:    sharedKey,
		identityCred: identityCred,
		innerU:       client,
		options:      options,
	}
}

func GetCompositeClientOptions[T, K, U any](client *CompositeClient[T, K, U]) *ClientOptions {
	return client.options
}

func GetAudience(clOpts *ClientOptions) string {
	if clOpts == nil || len(strings.TrimSpace(clOpts.Audience)) == 0 {
		return shared.TokenScope
	} else {
		return strings.TrimRight(clOpts.Audience, "/") + "/.default"
	}
}
