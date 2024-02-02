//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package base

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/internal/exported"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/internal/generated"
)

type Client[T any] struct {
	inner     *T
	sharedKey *exported.SharedKeyCredential
}

func InnerClient[T any](client *Client[T]) *T {
	return client.inner
}

func SharedKey[T any](client *Client[T]) *exported.SharedKeyCredential {
	return client.sharedKey
}

func NewClient[T any](inner *T) *Client[T] {
	return &Client[T]{inner: inner}
}

func NewServiceClient(queueURL string, pipeline runtime.Pipeline, sharedKey *exported.SharedKeyCredential) *Client[generated.ServiceClient] {
	return &Client[generated.ServiceClient]{
		inner:     generated.NewServiceClient(queueURL, pipeline),
		sharedKey: sharedKey,
	}
}

func NewQueueClient(queueURL string, pipeline runtime.Pipeline, sharedKey *exported.SharedKeyCredential) *CompositeClient[generated.QueueClient, generated.MessagesClient] {
	return &CompositeClient[generated.QueueClient, generated.MessagesClient]{
		innerT:    generated.NewQueueClient(queueURL, pipeline),
		innerU:    generated.NewMessagesClient(runtime.JoinPaths(queueURL, "messages"), pipeline),
		sharedKey: sharedKey,
	}
}

type CompositeClient[T, U any] struct {
	innerT    *T
	innerU    *U
	sharedKey *exported.SharedKeyCredential
}

func InnerClients[T, U any](client *CompositeClient[T, U]) (*T, *U) {
	return client.innerT, client.innerU
}

func SharedKeyComposite[T, U any](client *CompositeClient[T, U]) *exported.SharedKeyCredential {
	return client.sharedKey
}
