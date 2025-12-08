// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package internal

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/amqpwrap"
	"github.com/Azure/go-amqp"
)

type FakeNSForPartClient struct {
	NamespaceForAMQPLinks

	Receiver          *FakeAMQPReceiver
	NewReceiverErr    error
	NewReceiverCalled int

	Sender          *FakeAMQPSender
	NewSenderErr    error
	NewSenderCalled int

	RecoverFn func(ctx context.Context, clientRevision uint64) error
}

type FakeAMQPSession struct {
	amqpwrap.AMQPSession
	NS          *FakeNSForPartClient
	CloseCalled int
}

type FakeAMQPReceiver struct {
	amqpwrap.AMQPReceiverCloser

	// ActiveCredits are incremented and decremented by IssueCredit and Receive.
	ActiveCredits int32

	// IssuedCredit just accumulates, so we can get an idea of how many credits we issued overall.
	IssuedCredit []uint32

	// CreditsSetFromOptions is similar to issuedCredit, but only tracks credits added in via the LinkOptions.Credit
	// field (ie, enabling prefetch).
	CreditsSetFromOptions int32

	// ManualCreditsSetFromOptions is the value of the LinkOptions.ManualCredits value.
	ManualCreditsSetFromOptions bool

	Messages []*amqp.Message

	NameForLink string

	CloseCalled int
	CloseError  error
}

func (ns *FakeNSForPartClient) Recover(ctx context.Context, clientRevision uint64) error {
	return ns.RecoverFn(ctx, clientRevision)
}

func (ns *FakeNSForPartClient) NegotiateClaim(ctx context.Context, entityPath string) (context.CancelFunc, <-chan struct{}, error) {
	ctx, cancel := context.WithCancel(ctx)
	return cancel, ctx.Done(), nil
}

func (ns *FakeNSForPartClient) NewAMQPSession(ctx context.Context) (amqpwrap.AMQPSession, uint64, error) {
	return &FakeAMQPSession{
		NS: ns,
	}, 1, nil
}

func (sess *FakeAMQPSession) NewReceiver(ctx context.Context, source string, partitionID string, opts *amqp.ReceiverOptions) (amqpwrap.AMQPReceiverCloser, error) {
	sess.NS.NewReceiverCalled++
	sess.NS.Receiver.ManualCreditsSetFromOptions = opts.Credit == -1
	sess.NS.Receiver.CreditsSetFromOptions = opts.Credit

	if opts.Credit > 0 {
		sess.NS.Receiver.ActiveCredits = opts.Credit
	}

	return sess.NS.Receiver, sess.NS.NewReceiverErr
}

func (sess *FakeAMQPSession) NewSender(ctx context.Context, target string, partitionID string, opts *amqp.SenderOptions) (AMQPSenderCloser, error) {
	sess.NS.NewSenderCalled++
	return sess.NS.Sender, sess.NS.NewSenderErr
}

func (sess *FakeAMQPSession) Close(ctx context.Context) error {
	sess.CloseCalled++
	return nil
}

func (r *FakeAMQPReceiver) Credits() uint32 {
	return uint32(r.ActiveCredits)
}

func (r *FakeAMQPReceiver) IssueCredit(credit uint32) error {
	r.ActiveCredits += int32(credit)
	r.IssuedCredit = append(r.IssuedCredit, credit)
	return nil
}

func (r *FakeAMQPReceiver) LinkName() string {
	return r.NameForLink
}

func (r *FakeAMQPReceiver) Receive(ctx context.Context, o *amqp.ReceiveOptions) (*amqp.Message, error) {
	if len(r.Messages) > 0 {
		r.ActiveCredits--
		m := r.Messages[0]
		r.Messages = r.Messages[1:]
		return m, nil
	} else {
		<-ctx.Done()
		return nil, ctx.Err()
	}
}

func (r *FakeAMQPReceiver) Close(ctx context.Context) error {
	r.CloseCalled++
	return r.CloseError
}

type FakeAMQPSender struct {
	amqpwrap.AMQPSenderCloser
	CloseCalled int
	CloseError  error
}

func (s *FakeAMQPSender) Close(ctx context.Context) error {
	s.CloseCalled++
	return s.CloseError
}

type fakeAMQPClient struct {
	amqpwrap.AMQPClient
	closeCalled int
	session     *FakeAMQPSession
}

func (f *fakeAMQPClient) NewSession(ctx context.Context, opts *amqp.SessionOptions) (amqpwrap.AMQPSession, error) {
	return f.session, nil
}

func (f *fakeAMQPClient) Close() error {
	f.closeCalled++
	return nil
}
