//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package lease

import "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/internal/generated"

// StatusType defines values for StatusType
type StatusType = generated.LeaseStatusType

const (
	StatusTypeLocked   StatusType = generated.LeaseStatusTypeLocked
	StatusTypeUnlocked StatusType = generated.LeaseStatusTypeUnlocked
)

// PossibleStatusTypeValues returns the possible values for the StatusType const type.
func PossibleStatusTypeValues() []StatusType {
	return generated.PossibleLeaseStatusTypeValues()
}

// DurationType defines values for DurationType
type DurationType = generated.LeaseDurationType

const (
	DurationTypeInfinite DurationType = generated.LeaseDurationTypeInfinite
	DurationTypeFixed    DurationType = generated.LeaseDurationTypeFixed
)

// PossibleDurationTypeValues returns the possible values for the DurationType const type.
func PossibleDurationTypeValues() []DurationType {
	return generated.PossibleLeaseDurationTypeValues()
}

// StateType defines values for StateType
type StateType = generated.LeaseStateType

const (
	StateTypeAvailable StateType = generated.LeaseStateTypeAvailable
	StateTypeLeased    StateType = generated.LeaseStateTypeLeased
	StateTypeExpired   StateType = generated.LeaseStateTypeExpired
	StateTypeBreaking  StateType = generated.LeaseStateTypeBreaking
	StateTypeBroken    StateType = generated.LeaseStateTypeBroken
)

// PossibleStateTypeValues returns the possible values for the StateType const type.
func PossibleStateTypeValues() []StateType {
	return generated.PossibleLeaseStateTypeValues()
}
