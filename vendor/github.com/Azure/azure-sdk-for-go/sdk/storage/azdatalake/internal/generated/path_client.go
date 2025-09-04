//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package generated

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

func (client *PathClient) Endpoint() string {
	return client.endpoint
}

func (client *PathClient) InternalClient() *azcore.Client {
	return client.internal
}

// NewPathClient creates a new instance of ServiceClient with the specified values.
//   - endpoint - The URL of the service account, share, directory or file that is the target of the desired operation.
//   - azClient - azcore.Client is a basic HTTP client.  It consists of a pipeline and tracing provider.
func NewPathClient(endpoint string, azClient *azcore.Client) *PathClient {
	client := &PathClient{
		internal: azClient,
		endpoint: endpoint,
	}
	return client
}
