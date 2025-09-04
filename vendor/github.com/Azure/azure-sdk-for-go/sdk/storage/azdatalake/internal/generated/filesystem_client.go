//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package generated

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"time"
)

// used to convert times from UTC to GMT before sending across the wire
var gmt = time.FixedZone("GMT", 0)

func (client *FileSystemClient) Endpoint() string {
	return client.endpoint
}

func (client *FileSystemClient) InternalClient() *azcore.Client {
	return client.internal
}

// NewFileSystemClient creates a new instance of ServiceClient with the specified values.
//   - endpoint - The URL of the service account, share, directory or file that is the target of the desired operation.
//   - azClient - azcore.Client is a basic HTTP client.  It consists of a pipeline and tracing provider.
func NewFileSystemClient(endpoint string, azClient *azcore.Client) *FileSystemClient {
	client := &FileSystemClient{
		internal: azClient,
		endpoint: endpoint,
	}
	return client
}
