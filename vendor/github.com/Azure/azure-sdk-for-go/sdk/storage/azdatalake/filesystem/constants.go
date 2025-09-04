//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package filesystem

import (
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake"
)

// PublicAccessType defines values for AccessType - private (default) or file or filesystem.
type PublicAccessType = azblob.PublicAccessType

const (
	File       PublicAccessType = azblob.PublicAccessTypeBlob
	FileSystem PublicAccessType = azblob.PublicAccessTypeContainer
)

// StatusType defines values for StatusType
type StatusType = azdatalake.StatusType

const (
	StatusTypeLocked   StatusType = azdatalake.StatusTypeLocked
	StatusTypeUnlocked StatusType = azdatalake.StatusTypeUnlocked
)

// PossibleStatusTypeValues returns the possible values for the StatusType const type.
func PossibleStatusTypeValues() []StatusType {
	return azdatalake.PossibleStatusTypeValues()
}

// DurationType defines values for DurationType
type DurationType = azdatalake.DurationType

const (
	DurationTypeInfinite DurationType = azdatalake.DurationTypeInfinite
	DurationTypeFixed    DurationType = azdatalake.DurationTypeFixed
)

// PossibleDurationTypeValues returns the possible values for the DurationType const type.
func PossibleDurationTypeValues() []DurationType {
	return azdatalake.PossibleDurationTypeValues()
}

// StateType defines values for StateType
type StateType = azdatalake.StateType

const (
	StateTypeAvailable StateType = azdatalake.StateTypeAvailable
	StateTypeLeased    StateType = azdatalake.StateTypeLeased
	StateTypeExpired   StateType = azdatalake.StateTypeExpired
	StateTypeBreaking  StateType = azdatalake.StateTypeBreaking
	StateTypeBroken    StateType = azdatalake.StateTypeBroken
)
