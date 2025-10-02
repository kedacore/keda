//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package file

import (
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/exported"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/generated"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/generated_blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azdatalake/internal/path"
)

// EncryptionAlgorithmType defines values for EncryptionAlgorithmType.
type EncryptionAlgorithmType = path.EncryptionAlgorithmType

const (
	EncryptionAlgorithmTypeNone   EncryptionAlgorithmType = path.EncryptionAlgorithmTypeNone
	EncryptionAlgorithmTypeAES256 EncryptionAlgorithmType = path.EncryptionAlgorithmTypeAES256
)

// response models:

// ImmutabilityPolicyMode Specifies the immutability policy mode to set on the file.
type ImmutabilityPolicyMode = path.ImmutabilityPolicyMode

const (
	ImmutabilityPolicyModeMutable  ImmutabilityPolicyMode = path.ImmutabilityPolicyModeMutable
	ImmutabilityPolicyModeUnlocked ImmutabilityPolicyMode = path.ImmutabilityPolicyModeUnlocked
	ImmutabilityPolicyModeLocked   ImmutabilityPolicyMode = path.ImmutabilityPolicyModeLocked
)

// CopyStatusType defines values for CopyStatusType
type CopyStatusType = path.CopyStatusType

const (
	CopyStatusTypePending CopyStatusType = path.CopyStatusTypePending
	CopyStatusTypeSuccess CopyStatusType = path.CopyStatusTypeSuccess
	CopyStatusTypeAborted CopyStatusType = path.CopyStatusTypeAborted
	CopyStatusTypeFailed  CopyStatusType = path.CopyStatusTypeFailed
)

// TransferValidationType abstracts the various mechanisms used to verify a transfer.
type TransferValidationType = exported.TransferValidationType

// TransferValidationTypeCRC64 is a TransferValidationType used to provide a precomputed crc64.
type TransferValidationTypeCRC64 = exported.TransferValidationTypeCRC64

// TransferValidationTypeComputeCRC64 is a TransferValidationType that indicates a CRC64 should be computed during transfer.
func TransferValidationTypeComputeCRC64() TransferValidationType {
	return exported.TransferValidationTypeComputeCRC64()
}

// SetExpiryType defines the values for modes of file expiration.
type SetExpiryType = generated_blob.ExpiryOptions

const (
	// SetExpiryTypeAbsolute sets the expiration date as an absolute value expressed in RFC1123 format.
	SetExpiryTypeAbsolute SetExpiryType = generated_blob.ExpiryOptionsAbsolute

	// SetExpiryTypeNeverExpire sets the file to never expire or removes the current expiration date.
	SetExpiryTypeNeverExpire SetExpiryType = generated_blob.ExpiryOptionsNeverExpire

	// SetExpiryTypeRelativeToCreation sets the expiration date relative to the time of file creation.
	// The value is expressed as the number of miliseconds to elapse from the time of creation.
	SetExpiryTypeRelativeToCreation SetExpiryType = generated_blob.ExpiryOptionsRelativeToCreation

	// SetExpiryTypeRelativeToNow sets the expiration date relative to the current time.
	// The value is expressed as the number of milliseconds to elapse from the present time.
	SetExpiryTypeRelativeToNow SetExpiryType = generated_blob.ExpiryOptionsRelativeToNow
)

// CreateExpiryType defines the values for modes of file expiration specified during creation.
type CreateExpiryType = generated.PathExpiryOptions

const (
	// CreateExpiryTypeAbsolute sets the expiration date as an absolute value expressed in RFC1123 format.
	CreateExpiryTypeAbsolute CreateExpiryType = generated.PathExpiryOptionsAbsolute

	// CreateExpiryTypeNeverExpire sets the file to never expire or removes the current expiration date.
	CreateExpiryTypeNeverExpire CreateExpiryType = generated.PathExpiryOptionsNeverExpire

	// CreateExpiryTypeRelativeToNow sets the expiration date relative to the current time.
	// The value is expressed as the number of milliseconds to elapse from the present time.
	CreateExpiryTypeRelativeToNow CreateExpiryType = generated.PathExpiryOptionsRelativeToNow
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

// LeaseAction Describes actions that can be performed on a lease.
type LeaseAction = path.LeaseAction

var (
	LeaseActionAcquire        LeaseAction = path.LeaseActionAcquire
	LeaseActionRelease        LeaseAction = path.LeaseActionRelease
	LeaseActionAcquireRelease LeaseAction = path.LeaseActionAcquireRelease
	LeaseActionRenew          LeaseAction = path.LeaseActionRenew
)
