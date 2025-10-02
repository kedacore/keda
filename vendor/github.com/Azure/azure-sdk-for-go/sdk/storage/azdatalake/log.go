//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azdatalake

import (
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/exported"
)

const (
	// EventUpload is used for logging events related to upload operation.
	EventUpload = exported.EventUpload

	// EventError is used for logging errors.
	EventError = exported.EventError
)
