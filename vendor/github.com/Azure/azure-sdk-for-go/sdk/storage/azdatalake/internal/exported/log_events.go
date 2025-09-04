//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package exported

import (
	"github.com/Azure/azure-sdk-for-go/sdk/internal/log"
)

// NOTE: these are publicly exported via type-aliasing in azdatalake/log.go
const (
	// EventUpload is used when we compute number of chunks to upload and size of each chunk.
	EventUpload log.Event = "azdatalake.Upload"

	// EventError is used for logging errors.
	EventError log.Event = "azdatalake.Error"
)
