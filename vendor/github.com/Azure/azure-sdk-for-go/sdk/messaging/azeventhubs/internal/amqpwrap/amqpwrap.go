// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

// Package amqpwrap has some simple wrappers to make it easier to
// abstract the go-amqp types.
package amqpwrap

import (
	"context"
	"errors"
	"time"

	"github.com/Azure/go-amqp"
)

// AMQPReceiver is implemented by *amqp.Receiver
type AMQPReceiver interface {
	IssueCredit(credit uint32) error
	Receive(ctx context.Context, o *amqp.ReceiveOptions) (*amqp.Message, error)
	Prefetched() *amqp.Message

	// settlement functions
	AcceptMessage(ctx context.Context, msg *amqp.Message) error
	RejectMessage(ctx context.Context, msg *amqp.Message, e *amqp.Error) error
	ReleaseMessage(ctx context.Context, msg *amqp.Message) error
	ModifyMessage(ctx context.Context, msg *amqp.Message, options *amqp.ModifyMessageOptions) error

	LinkName() string
	LinkSourceFilterValue(name string) any

	// wrapper only functions

	// Credits returns the # of credits still active on this link.
	Credits() uint32

	ConnID() uint64
}

// AMQPReceiverCloser is implemented by *amqp.Receiver
type AMQPReceiverCloser interface {
	AMQPReceiver
	Close(ctx context.Context) error
}

// AMQPSender is implemented by *amqp.Sender
type AMQPSender interface {
	Send(ctx context.Context, msg *amqp.Message, o *amqp.SendOptions) error
	MaxMessageSize() uint64
	LinkName() string
	ConnID() uint64
}

// AMQPSenderCloser is implemented by *amqp.Sender
type AMQPSenderCloser interface {
	AMQPSender
	Close(ctx context.Context) error
}

// AMQPSession is a simple interface, implemented by *AMQPSessionWrapper.
// It exists only so we can return AMQPReceiver/AMQPSender interfaces.
type AMQPSession interface {
	Close(ctx context.Context) error
	ConnID() uint64
	NewReceiver(ctx context.Context, source string, partitionID string, opts *amqp.ReceiverOptions) (AMQPReceiverCloser, error)
	NewSender(ctx context.Context, target string, partitionID string, opts *amqp.SenderOptions) (AMQPSenderCloser, error)
}

type AMQPClient interface {
	Close() error
	NewSession(ctx context.Context, opts *amqp.SessionOptions) (AMQPSession, error)
	ID() uint64
}

type goamqpConn interface {
	NewSession(ctx context.Context, opts *amqp.SessionOptions) (*amqp.Session, error)
	Close() error
}

type goamqpSession interface {
	Close(ctx context.Context) error
	NewReceiver(ctx context.Context, source string, opts *amqp.ReceiverOptions) (*amqp.Receiver, error)
	NewSender(ctx context.Context, target string, opts *amqp.SenderOptions) (*amqp.Sender, error)
}

type goamqpReceiver interface {
	IssueCredit(credit uint32) error
	Receive(ctx context.Context, o *amqp.ReceiveOptions) (*amqp.Message, error)
	Prefetched() *amqp.Message

	// settlement functions
	AcceptMessage(ctx context.Context, msg *amqp.Message) error
	RejectMessage(ctx context.Context, msg *amqp.Message, e *amqp.Error) error
	ReleaseMessage(ctx context.Context, msg *amqp.Message) error
	ModifyMessage(ctx context.Context, msg *amqp.Message, options *amqp.ModifyMessageOptions) error

	LinkName() string
	LinkSourceFilterValue(name string) any
	Close(ctx context.Context) error
}

type goamqpSender interface {
	Send(ctx context.Context, msg *amqp.Message, o *amqp.SendOptions) error
	MaxMessageSize() uint64
	LinkName() string
	Close(ctx context.Context) error
}

// AMQPClientWrapper is a simple interface, implemented by *AMQPClientWrapper
// It exists only so we can return AMQPSession, which itself only exists so we can
// return interfaces for AMQPSender and AMQPReceiver from AMQPSession.
type AMQPClientWrapper struct {
	ConnID uint64
	Inner  goamqpConn
}

func (w *AMQPClientWrapper) ID() uint64 {
	return w.ConnID
}

func (w *AMQPClientWrapper) Close() error {
	err := w.Inner.Close()
	return WrapError(err, w.ConnID, "", "")
}

func (w *AMQPClientWrapper) NewSession(ctx context.Context, opts *amqp.SessionOptions) (AMQPSession, error) {
	sess, err := w.Inner.NewSession(ctx, opts)

	if err != nil {
		return nil, WrapError(err, w.ConnID, "", "")
	}

	return &AMQPSessionWrapper{
		connID:               w.ConnID,
		Inner:                sess,
		ContextWithTimeoutFn: context.WithTimeout,
	}, nil
}

type AMQPSessionWrapper struct {
	connID               uint64
	Inner                goamqpSession
	ContextWithTimeoutFn ContextWithTimeoutFn
}

func (w *AMQPSessionWrapper) ConnID() uint64 {
	return w.connID
}

func (w *AMQPSessionWrapper) Close(ctx context.Context) error {
	ctx, cancel := w.ContextWithTimeoutFn(ctx, defaultCloseTimeout)
	defer cancel()
	err := w.Inner.Close(ctx)
	return WrapError(err, w.connID, "", "")
}

func (w *AMQPSessionWrapper) NewReceiver(ctx context.Context, source string, partitionID string, opts *amqp.ReceiverOptions) (AMQPReceiverCloser, error) {
	receiver, err := w.Inner.NewReceiver(ctx, source, opts)

	if err != nil {
		return nil, WrapError(err, w.connID, "", partitionID)
	}

	return &AMQPReceiverWrapper{
		connID:               w.connID,
		partitionID:          partitionID,
		Inner:                receiver,
		ContextWithTimeoutFn: context.WithTimeout}, nil
}

func (w *AMQPSessionWrapper) NewSender(ctx context.Context, target string, partitionID string, opts *amqp.SenderOptions) (AMQPSenderCloser, error) {
	sender, err := w.Inner.NewSender(ctx, target, opts)

	if err != nil {
		return nil, WrapError(err, w.connID, "", partitionID)
	}

	return &AMQPSenderWrapper{
		connID:               w.connID,
		partitionID:          partitionID,
		Inner:                sender,
		ContextWithTimeoutFn: context.WithTimeout}, nil
}

type AMQPReceiverWrapper struct {
	connID               uint64
	partitionID          string
	Inner                goamqpReceiver
	credits              uint32
	ContextWithTimeoutFn ContextWithTimeoutFn
}

func (rw *AMQPReceiverWrapper) ConnID() uint64 {
	return rw.connID
}

func (rw *AMQPReceiverWrapper) Credits() uint32 {
	return rw.credits
}

func (rw *AMQPReceiverWrapper) IssueCredit(credit uint32) error {
	err := rw.Inner.IssueCredit(credit)

	if err == nil {
		rw.credits += credit
	}

	return WrapError(err, rw.connID, rw.LinkName(), rw.partitionID)
}

func (rw *AMQPReceiverWrapper) Receive(ctx context.Context, o *amqp.ReceiveOptions) (*amqp.Message, error) {
	message, err := rw.Inner.Receive(ctx, o)

	if err != nil {
		return nil, WrapError(err, rw.connID, rw.LinkName(), rw.partitionID)
	}

	rw.credits--
	return message, nil
}

func (rw *AMQPReceiverWrapper) Prefetched() *amqp.Message {
	msg := rw.Inner.Prefetched()

	if msg == nil {
		return nil
	}

	rw.credits--
	return msg
}

// settlement functions
func (rw *AMQPReceiverWrapper) AcceptMessage(ctx context.Context, msg *amqp.Message) error {
	err := rw.Inner.AcceptMessage(ctx, msg)
	return WrapError(err, rw.connID, rw.LinkName(), rw.partitionID)
}

func (rw *AMQPReceiverWrapper) RejectMessage(ctx context.Context, msg *amqp.Message, e *amqp.Error) error {
	err := rw.Inner.RejectMessage(ctx, msg, e)
	return WrapError(err, rw.connID, rw.LinkName(), rw.partitionID)
}

func (rw *AMQPReceiverWrapper) ReleaseMessage(ctx context.Context, msg *amqp.Message) error {
	err := rw.Inner.ReleaseMessage(ctx, msg)
	return WrapError(err, rw.connID, rw.LinkName(), rw.partitionID)
}

func (rw *AMQPReceiverWrapper) ModifyMessage(ctx context.Context, msg *amqp.Message, options *amqp.ModifyMessageOptions) error {
	err := rw.Inner.ModifyMessage(ctx, msg, options)
	return WrapError(err, rw.connID, rw.LinkName(), rw.partitionID)
}

func (rw *AMQPReceiverWrapper) LinkName() string {
	return rw.Inner.LinkName()
}

func (rw *AMQPReceiverWrapper) LinkSourceFilterValue(name string) any {
	return rw.Inner.LinkSourceFilterValue(name)
}

func (rw *AMQPReceiverWrapper) Close(ctx context.Context) error {
	ctx, cancel := rw.ContextWithTimeoutFn(ctx, defaultCloseTimeout)
	defer cancel()
	err := rw.Inner.Close(ctx)

	return WrapError(err, rw.connID, rw.LinkName(), rw.partitionID)
}

type AMQPSenderWrapper struct {
	connID               uint64
	partitionID          string
	Inner                goamqpSender
	ContextWithTimeoutFn ContextWithTimeoutFn
}

func (sw *AMQPSenderWrapper) ConnID() uint64 {
	return sw.connID
}

func (sw *AMQPSenderWrapper) Send(ctx context.Context, msg *amqp.Message, o *amqp.SendOptions) error {
	err := sw.Inner.Send(ctx, msg, o)
	return WrapError(err, sw.connID, sw.LinkName(), sw.partitionID)
}

func (sw *AMQPSenderWrapper) MaxMessageSize() uint64 {
	return sw.Inner.MaxMessageSize()
}

func (sw *AMQPSenderWrapper) LinkName() string {
	return sw.Inner.LinkName()
}

func (sw *AMQPSenderWrapper) Close(ctx context.Context) error {
	ctx, cancel := sw.ContextWithTimeoutFn(ctx, defaultCloseTimeout)
	defer cancel()
	err := sw.Inner.Close(ctx)

	return WrapError(err, sw.connID, sw.LinkName(), sw.partitionID)
}

var ErrConnResetNeeded = errors.New("connection must be reset, link/connection state may be inconsistent")

const defaultCloseTimeout = time.Minute

// ContextWithTimeoutFn matches the signature for `context.WithTimeout` and is used when we want to
// stub things out for tests.
type ContextWithTimeoutFn func(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc)
