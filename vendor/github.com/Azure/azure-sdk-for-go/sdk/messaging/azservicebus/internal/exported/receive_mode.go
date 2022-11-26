// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package exported

// NOTE: these are publicly exported via type-aliasing in azservicebus/receiver.go

// ReceiveMode represents the lock style to use for a reciever - either
// `PeekLock` or `ReceiveAndDelete`
type ReceiveMode int

const (
	// PeekLock will lock messages as they are received and can be settled
	// using the Receiver's (Complete|Abandon|DeadLetter|Defer)Message
	// functions.
	PeekLock ReceiveMode = 0
	// ReceiveAndDelete will delete messages as they are received.
	ReceiveAndDelete ReceiveMode = 1
)
