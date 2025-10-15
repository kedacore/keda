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

// AMQPReceiver is implemented by [*AMQPReceiverWrapper]
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
	Properties() map[string]any

	// wrapper only functions,

	// Credits returns the # of credits still active on this link.
	Credits() uint32
}

// AMQPReceiverCloser is implemented by [*AMQPReceiverWrapper]
type AMQPReceiverCloser interface {
	AMQPReceiver
	Close(ctx context.Context) error
}

// AMQPSender is implemented by [*AMQPSenderWrapper]
type AMQPSender interface {
	Send(ctx context.Context, msg *amqp.Message, o *amqp.SendOptions) error
	MaxMessageSize() uint64
	LinkName() string
}

// AMQPSenderCloser is implemented by [*AMQPSenderWrapper]
type AMQPSenderCloser interface {
	AMQPSender
	Close(ctx context.Context) error
}

// AMQPSession is a simple interface, implemented by [*AMQPSessionWrapper].
// It exists only so we can return AMQPReceiver/AMQPSender interfaces.
type AMQPSession interface {
	Close(ctx context.Context) error
	NewReceiver(ctx context.Context, source string, opts *amqp.ReceiverOptions) (AMQPReceiverCloser, error)
	NewSender(ctx context.Context, target string, opts *amqp.SenderOptions) (AMQPSenderCloser, error)
}

// AMQPClient is a simple interface, implemented by [*AMQPClientWrapper].
type AMQPClient interface {
	Close() error
	NewSession(ctx context.Context, opts *amqp.SessionOptions) (AMQPSession, error)
	Name() string
}

// goamqpConn is a simple interface, implemented by [*amqp.Conn]
type goamqpConn interface {
	NewSession(ctx context.Context, opts *amqp.SessionOptions) (*amqp.Session, error)
	Close() error
}

// goamqpSession is a simple interface, implemented by [*amqp.Session]
type goamqpSession interface {
	Close(ctx context.Context) error
	NewReceiver(ctx context.Context, source string, opts *amqp.ReceiverOptions) (*amqp.Receiver, error)
	NewSender(ctx context.Context, target string, opts *amqp.SenderOptions) (*amqp.Sender, error)
}

// goamqpReceiver is a simple interface, implemented by [*amqp.Receiver]
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
	Properties() map[string]any
	Close(ctx context.Context) error
}

// AMQPClientWrapper is a simple interface, implemented by *AMQPClientWrapper
// It exists only so we can return AMQPSession, which itself only exists so we can
// return interfaces for AMQPSender and AMQPReceiver from AMQPSession.
type AMQPClientWrapper struct {
	Inner goamqpConn
	ID    string
}

func (w *AMQPClientWrapper) Name() string {
	return w.ID
}

func (w *AMQPClientWrapper) Close() error {
	return w.Inner.Close()
}

func (w *AMQPClientWrapper) NewSession(ctx context.Context, opts *amqp.SessionOptions) (AMQPSession, error) {
	sess, err := w.Inner.NewSession(ctx, opts)

	if err != nil {
		return nil, err
	}

	return &AMQPSessionWrapper{
		Inner:                sess,
		ContextWithTimeoutFn: context.WithTimeout,
	}, nil
}

type AMQPSessionWrapper struct {
	Inner                goamqpSession
	ContextWithTimeoutFn ContextWithTimeoutFn
}

func (w *AMQPSessionWrapper) Close(ctx context.Context) error {
	ctx, cancel := w.ContextWithTimeoutFn(ctx, defaultCloseTimeout)
	defer cancel()
	return w.Inner.Close(ctx)
}

func (w *AMQPSessionWrapper) NewReceiver(ctx context.Context, source string, opts *amqp.ReceiverOptions) (AMQPReceiverCloser, error) {
	receiver, err := w.Inner.NewReceiver(ctx, source, opts)

	if err != nil {
		return nil, err
	}

	return &AMQPReceiverWrapper{Inner: receiver, ContextWithTimeoutFn: context.WithTimeout}, nil
}

func (w *AMQPSessionWrapper) NewSender(ctx context.Context, target string, opts *amqp.SenderOptions) (AMQPSenderCloser, error) {
	sender, err := w.Inner.NewSender(ctx, target, opts)

	if err != nil {
		return nil, err
	}

	return &AMQPSenderWrapper{Inner: sender, ContextWithTimeoutFn: context.WithTimeout}, nil
}

type AMQPReceiverWrapper struct {
	Inner                goamqpReceiver
	credits              uint32
	ContextWithTimeoutFn ContextWithTimeoutFn
}

func (rw *AMQPReceiverWrapper) Credits() uint32 {
	return rw.credits
}

func (rw *AMQPReceiverWrapper) IssueCredit(credit uint32) error {
	err := rw.Inner.IssueCredit(credit)

	if err == nil {
		rw.credits += credit
	}

	return err
}

func (rw *AMQPReceiverWrapper) Receive(ctx context.Context, o *amqp.ReceiveOptions) (*amqp.Message, error) {
	message, err := rw.Inner.Receive(ctx, o)

	if err != nil {
		return nil, err
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
	return rw.Inner.AcceptMessage(ctx, msg)
}

func (rw *AMQPReceiverWrapper) RejectMessage(ctx context.Context, msg *amqp.Message, e *amqp.Error) error {
	return rw.Inner.RejectMessage(ctx, msg, e)
}

func (rw *AMQPReceiverWrapper) ReleaseMessage(ctx context.Context, msg *amqp.Message) error {
	return rw.Inner.ReleaseMessage(ctx, msg)
}

func (rw *AMQPReceiverWrapper) ModifyMessage(ctx context.Context, msg *amqp.Message, options *amqp.ModifyMessageOptions) error {
	return rw.Inner.ModifyMessage(ctx, msg, options)
}

func (rw *AMQPReceiverWrapper) LinkName() string {
	return rw.Inner.LinkName()
}

func (rw *AMQPReceiverWrapper) LinkSourceFilterValue(name string) any {
	return rw.Inner.LinkSourceFilterValue(name)
}

func (rw *AMQPReceiverWrapper) Properties() map[string]any {
	return rw.Inner.Properties()
}

func (rw *AMQPReceiverWrapper) Close(ctx context.Context) error {
	ctx, cancel := rw.ContextWithTimeoutFn(ctx, defaultCloseTimeout)
	defer cancel()
	return rw.Inner.Close(ctx)
}

type AMQPSenderWrapper struct {
	Inner                AMQPSenderCloser
	ContextWithTimeoutFn ContextWithTimeoutFn
}

func (sw *AMQPSenderWrapper) Send(ctx context.Context, msg *amqp.Message, o *amqp.SendOptions) error {
	return sw.Inner.Send(ctx, msg, o)
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
	return sw.Inner.Close(ctx)
}

var ErrConnResetNeeded = errors.New("connection must be reset, link/connection state may be inconsistent")

const defaultCloseTimeout = time.Minute

// ContextWithTimeoutFn matches the signature for `context.WithTimeout` and is used when we want to
// stub things out for tests.
type ContextWithTimeoutFn func(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc)
