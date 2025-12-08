// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azeventhubs

import "github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/exported"

// ConnectionStringProperties are the properties of a connection string
// as returned by [ParseConnectionString].
type ConnectionStringProperties = exported.ConnectionStringProperties

// ParseConnectionString takes a connection string from the Azure portal and returns the
// parsed representation.
//
// There are two supported formats:
//  1. Connection strings generated from the portal (or elsewhere) that contain an embedded key and keyname.
//  2. A connection string with an embedded SharedAccessSignature:
//     Endpoint=sb://<sb>.servicebus.windows.net;SharedAccessSignature=SharedAccessSignature sr=<sb>.servicebus.windows.net&sig=<base64-sig>&se=<expiry>&skn=<keyname>"
func ParseConnectionString(connStr string) (ConnectionStringProperties, error) {
	return exported.ParseConnectionString(connStr)
}
