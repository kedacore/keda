//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package azqueue

import "github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/internal/generated"

// GeoReplicationStatus - The status of the secondary location
type GeoReplicationStatus = generated.GeoReplicationStatus

const (
	GeoReplicationStatusLive        GeoReplicationStatus = generated.GeoReplicationStatusLive
	GeoReplicationStatusBootstrap   GeoReplicationStatus = generated.GeoReplicationStatusBootstrap
	GeoReplicationStatusUnavailable GeoReplicationStatus = generated.GeoReplicationStatusUnavailable
)
