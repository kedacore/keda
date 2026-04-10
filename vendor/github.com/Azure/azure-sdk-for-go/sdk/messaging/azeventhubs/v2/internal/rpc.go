// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package internal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	azlog "github.com/Azure/azure-sdk-for-go/sdk/internal/log"
	"github.com/Azure/azure-sdk-for-go/sdk/internal/uuid"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/amqpwrap"
	"github.com/Azure/go-amqp"
)

const (
	replyPostfix           = "-reply-to-"
	statusCodeKey          = "status-code"
	descriptionKey         = "status-description"
	defaultReceiverCredits = 1000
)

var ErrRPCLinkClosed = errors.New("rpc link closed")

type (
	// rpcLink is the bidirectional communication structure used for CBS negotiation
	rpcLink struct {
		session  amqpwrap.AMQPSession
		receiver amqpwrap.AMQPReceiverCloser // *amqp.Receiver
		sender   amqpwrap.AMQPSenderCloser   // *amqp.Sender

		clientAddress string
		sessionID     *string
		id            string

		responseMu           sync.Mutex
		responseRouterClosed chan struct{}

		responseMap      map[string]chan rpcResponse
		rpcLinkCtx       context.Context
		rpcLinkCtxCancel context.CancelFunc
		broadcastErr     error // the error that caused the responseMap to be nil'd

		logEvent azlog.Event

		// for unit tests
		uuidNewV4 func() (uuid.UUID, error)
	}

	// RPCLinkOption provides a way to customize the construction of a Link
	RPCLinkOption func(link *rpcLink) error

	rpcResponse struct {
		message *amqp.Message
		err     error
	}
)

// RPCError is an error from an RPCLink.
// RPCLinks are used for communication with the $management and $cbs links.
type RPCError struct {
	Resp    *amqpwrap.RPCResponse
	Message string
}

// Error is a string representation of the error.
func (e RPCError) Error() string {
	return e.Message
}

// RPCCode is the code that comes back in the rpc response. This code is intended
// for programs toreact to programatically.
func (e RPCError) RPCCode() int {
	return e.Resp.Code
}

type RPCLinkArgs struct {
	Client              amqpwrap.AMQPClient
	Address             string
	LogEvent            azlog.Event
	DesiredCapabilities []string
}

// NewRPCLink will build a new request response link
func NewRPCLink(ctx context.Context, args RPCLinkArgs) (amqpwrap.RPCLink, error) {
	session, err := args.Client.NewSession(ctx, nil)

	if err != nil {
		return nil, err
	}

	linkID, err := uuid.New()
	if err != nil {
		_ = session.Close(ctx)
		return nil, err
	}

	id := linkID.String()

	link := &rpcLink{
		session:       session,
		clientAddress: strings.ReplaceAll("$", "", args.Address) + replyPostfix + id,
		id:            id,

		uuidNewV4:            uuid.New,
		responseMap:          map[string]chan rpcResponse{},
		responseRouterClosed: make(chan struct{}),
		logEvent:             args.LogEvent,
	}

	sender, err := session.NewSender(
		ctx,
		args.Address,
		"",
		&amqp.SenderOptions{
			DesiredCapabilities: args.DesiredCapabilities,
		},
	)
	if err != nil {
		_ = session.Close(ctx)
		return nil, err
	}

	receiverOpts := &amqp.ReceiverOptions{
		TargetAddress: link.clientAddress,
		Credit:        defaultReceiverCredits,

		// set our receiver link into the "receive and delete" mode - messages arrive pre-settled.
		SettlementMode:            amqp.ReceiverSettleModeFirst.Ptr(),
		RequestedSenderSettleMode: amqp.SenderSettleModeSettled.Ptr(),

		DesiredCapabilities: args.DesiredCapabilities,
	}

	if link.sessionID != nil {
		const name = "com.microsoft:session-filter"
		const code = uint64(0x00000137000000C)
		if link.sessionID == nil {
			receiverOpts.Filters = append(receiverOpts.Filters, amqp.NewLinkFilter(name, code, nil))
		} else {
			receiverOpts.Filters = append(receiverOpts.Filters, amqp.NewLinkFilter(name, code, link.sessionID))
		}
	}

	receiver, err := session.NewReceiver(ctx, args.Address, "", receiverOpts)
	if err != nil {
		_ = session.Close(ctx)
		return nil, err
	}

	link.sender = sender
	link.receiver = receiver
	link.rpcLinkCtx, link.rpcLinkCtxCancel = context.WithCancel(context.Background())

	go link.responseRouter()

	return link, nil
}

const responseRouterShutdownMessage = "Response router has shut down"

// responseRouter is responsible for taking any messages received on the 'response'
// link and forwarding it to the proper channel. The channel is being select'd by the
// original `RPC` call.
func (l *rpcLink) responseRouter() {
	defer azlog.Writef(l.logEvent, responseRouterShutdownMessage)
	defer close(l.responseRouterClosed)

	for {
		res, err := l.receiver.Receive(l.rpcLinkCtx, nil)

		if err != nil {
			// if the link or connection has a malfunction that would require it to restart then
			// we need to bail out, broadcasting to all affected callers/consumers.
			if GetRecoveryKind(err) != RecoveryKindNone {
				if IsCancelError(err) {
					err = ErrRPCLinkClosed
				} else {
					azlog.Writef(l.logEvent, "Error in RPCLink, stopping response router: %s", err.Error())
				}

				l.broadcastError(err)
				break
			}

			azlog.Writef(l.logEvent, "Non-fatal error in RPCLink, starting to receive again: %s", err.Error())
			continue
		}

		// I don't believe this should happen. The JS version of this same code
		// ignores errors as well since responses should always be correlated
		// to actual send requests. So this is just here for completeness.
		if res == nil {
			azlog.Writef(l.logEvent, "RPCLink received no error, but also got no response")
			continue
		}

		autogenMessageId, ok := res.Properties.CorrelationID.(string)

		if !ok {
			azlog.Writef(l.logEvent, "RPCLink message received without a CorrelationID %v", res)
			continue
		}

		ch := l.deleteChannelFromMap(autogenMessageId)

		if ch == nil {
			azlog.Writef(l.logEvent, "RPCLink had no response channel for correlation ID %v", autogenMessageId)
			continue
		}

		ch <- rpcResponse{message: res, err: err}
	}
}

func (l *rpcLink) RPC(ctx context.Context, msg *amqp.Message) (*amqpwrap.RPCResponse, error) {
	resp, err := l.internalRPC(ctx, msg)

	if err != nil {
		return nil, amqpwrap.WrapError(err, l.ConnID(), l.LinkName(), "")
	}

	return resp, nil
}

// RPC sends a request and waits on a response for that request
func (l *rpcLink) internalRPC(ctx context.Context, msg *amqp.Message) (*amqpwrap.RPCResponse, error) {
	copiedMessage, messageID, err := addMessageID(msg, l.uuidNewV4)

	if err != nil {
		return nil, err
	}

	// use the copiedMessage from this point
	msg = copiedMessage

	const altStatusCodeKey, altDescriptionKey = "statusCode", "statusDescription"

	msg.Properties.ReplyTo = &l.clientAddress

	if msg.ApplicationProperties == nil {
		msg.ApplicationProperties = make(map[string]any)
	}

	if _, ok := msg.ApplicationProperties["server-timeout"]; !ok {
		if deadline, ok := ctx.Deadline(); ok {
			msg.ApplicationProperties["server-timeout"] = uint(time.Until(deadline) / time.Millisecond)
		}
	}

	responseCh := l.addChannelToMap(messageID)

	if responseCh == nil {
		return nil, l.broadcastErr
	}

	err = l.sender.Send(ctx, msg, nil)

	if err != nil {
		l.deleteChannelFromMap(messageID)
		return nil, fmt.Errorf("failed to send message with ID %s: %w", messageID, err)
	}

	var res *amqp.Message

	select {
	case <-ctx.Done():
		l.deleteChannelFromMap(messageID)
		res, err = nil, ctx.Err()
	case resp := <-responseCh:
		// this will get triggered by the loop in 'startReceiverRouter' when it receives
		// a message with our autoGenMessageID set in the correlation_id property.
		res, err = resp.message, resp.err
	}

	if err != nil {
		return nil, err
	}

	var statusCode int
	statusCodeCandidates := []string{statusCodeKey, altStatusCodeKey}
	for i := range statusCodeCandidates {
		if rawStatusCode, ok := res.ApplicationProperties[statusCodeCandidates[i]]; ok {
			if cast, ok := rawStatusCode.(int32); ok {
				statusCode = int(cast)
				break
			}

			return nil, errors.New("status code was not of expected type int32")
		}
	}
	if statusCode == 0 {
		return nil, errors.New("status codes was not found on rpc message")
	}

	var description string
	descriptionCandidates := []string{descriptionKey, altDescriptionKey}
	for i := range descriptionCandidates {
		if rawDescription, ok := res.ApplicationProperties[descriptionCandidates[i]]; ok {
			if description, ok = rawDescription.(string); ok || rawDescription == nil {
				break
			} else {
				return nil, errors.New("status description was not of expected type string")
			}
		}
	}

	response := &amqpwrap.RPCResponse{
		Code:        int(statusCode),
		Description: description,
		Message:     res,
	}

	var rpcErr RPCError

	if asRPCError(response, &rpcErr) {
		return nil, rpcErr
	}

	return response, err
}

func (l *rpcLink) ConnID() uint64 {
	return l.session.ConnID()
}

// Close the link receiver, sender and session
func (l *rpcLink) Close(ctx context.Context) error {
	l.rpcLinkCtxCancel()

	select {
	case <-l.responseRouterClosed:
	case <-ctx.Done():
	}

	if l.session != nil {
		return l.session.Close(ctx)
	}

	return nil
}

// addChannelToMap adds a channel which will be used by the response router to
// notify when there is a response to the request.
// If l.responseMap is nil (for instance, via broadcastError) this function will
// return nil.
func (l *rpcLink) addChannelToMap(messageID string) chan rpcResponse {
	l.responseMu.Lock()
	defer l.responseMu.Unlock()

	if l.responseMap == nil {
		return nil
	}

	responseCh := make(chan rpcResponse, 1)
	l.responseMap[messageID] = responseCh

	return responseCh
}

// deleteChannelFromMap removes the message from our internal map and returns
// a channel that the corresponding RPC() call is waiting on.
// If l.responseMap is nil (for instance, via broadcastError) this function will
// return nil.
func (l *rpcLink) deleteChannelFromMap(messageID string) chan rpcResponse {
	l.responseMu.Lock()
	defer l.responseMu.Unlock()

	if l.responseMap == nil {
		return nil
	}

	ch := l.responseMap[messageID]
	delete(l.responseMap, messageID)

	return ch
}

// broadcastError notifies the anyone waiting for a response that the link/session/connection
// has closed.
func (l *rpcLink) broadcastError(err error) {
	l.responseMu.Lock()
	defer l.responseMu.Unlock()

	for _, ch := range l.responseMap {
		ch <- rpcResponse{err: err}
	}

	l.broadcastErr = err
	l.responseMap = nil
}

// addMessageID generates a unique UUID for the message. When the service
// responds it will fill out the correlation ID property of the response
// with this ID, allowing us to link the request and response together.
//
// NOTE: this function copies 'message', adding in a 'Properties' object
// if it does not already exist.
func addMessageID(message *amqp.Message, uuidNewV4 func() (uuid.UUID, error)) (*amqp.Message, string, error) {
	uuid, err := uuidNewV4()

	if err != nil {
		return nil, "", err
	}

	autoGenMessageID := uuid.String()

	// we need to modify the message so we'll make a copy
	copiedMessage := *message

	if message.Properties == nil {
		copiedMessage.Properties = &amqp.MessageProperties{
			MessageID: autoGenMessageID,
		}
	} else {
		// properties already exist, make a copy and then update
		// the message ID
		copiedProperties := *message.Properties
		copiedProperties.MessageID = autoGenMessageID

		copiedMessage.Properties = &copiedProperties
	}

	return &copiedMessage, autoGenMessageID, nil
}

// asRPCError checks to see if the res is actually a failed request
// (where failed means the status code was non-2xx). If so,
// it returns true and updates the struct pointed to by err.
func asRPCError(res *amqpwrap.RPCResponse, err *RPCError) bool {
	if res == nil {
		return false
	}

	if res.Code >= 200 && res.Code < 300 {
		return false
	}

	*err = RPCError{
		Message: fmt.Sprintf("rpc: failed, status code %d and description: %s", res.Code, res.Description),
		Resp:    res,
	}

	return true
}
