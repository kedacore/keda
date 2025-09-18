//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package path

import (
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/generated"
)

type EncryptionAlgorithmType = generated.EncryptionAlgorithmType

const (
	EncryptionAlgorithmTypeNone   EncryptionAlgorithmType = generated.EncryptionAlgorithmTypeNone
	EncryptionAlgorithmTypeAES256 EncryptionAlgorithmType = generated.EncryptionAlgorithmTypeAES256
)

type ImmutabilityPolicyMode = blob.ImmutabilityPolicyMode

const (
	ImmutabilityPolicyModeMutable  ImmutabilityPolicyMode = blob.ImmutabilityPolicyModeMutable
	ImmutabilityPolicyModeUnlocked ImmutabilityPolicyMode = blob.ImmutabilityPolicyModeUnlocked
	ImmutabilityPolicyModeLocked   ImmutabilityPolicyMode = blob.ImmutabilityPolicyModeLocked
)

// CopyStatusType defines values for CopyStatusType
type CopyStatusType = blob.CopyStatusType

const (
	CopyStatusTypePending CopyStatusType = blob.CopyStatusTypePending
	CopyStatusTypeSuccess CopyStatusType = blob.CopyStatusTypeSuccess
	CopyStatusTypeAborted CopyStatusType = blob.CopyStatusTypeAborted
	CopyStatusTypeFailed  CopyStatusType = blob.CopyStatusTypeFailed
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

type LeaseAction = generated.LeaseAction

const (
	LeaseActionAcquire        = generated.LeaseActionAcquire
	LeaseActionRelease        = generated.LeaseActionRelease
	LeaseActionAcquireRelease = generated.LeaseActionAcquireRelease
	LeaseActionRenew          = generated.LeaseActionAutoRenew
)
