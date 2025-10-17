// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package exported

// NOTE: these are publicly exported via type-aliasing in azservicebus/receiver.go

// ReceiveMode represents the lock style to use for a reciever - either
// `PeekLock` or `ReceiveAndDelete`
type ReceiveMode int

const (
	// PeekLock will lock messages as they are received. These messages can then be settled using the
	// Receiver's (Complete|Abandon|DeadLetter|Defer)Message functions.
	PeekLock ReceiveMode = 0

	// ReceiveAndDelete will delete messages as they are received.
	//
	// NOTE: In ReceiveAndDelete mode you should continue to call ReceiveMessages(), to receive any cached messages, even after the Receiver
	// has been closed. See [receiver_and_delete_example] for an example of how to incorporate this into your code.
	//
	// This is not needed for Receivers in [PeekLock] mode, as cached messages are automatically released to the service.
	//
	// [receiver_and_delete_example]: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus#example-Receiver.ReceiveMessages-ReceiveAndDelete
	ReceiveAndDelete ReceiveMode = 1
)
