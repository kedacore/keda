package servicebus

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/Azure/azure-amqp-common-go/log"
	"github.com/Azure/azure-amqp-common-go/rpc"
	"pack.ag/amqp"
)

type (
	// MessageIterator offers a simple mechanism for iterating over a list of
	MessageIterator interface {
		Done() bool
		Next(context.Context) (*Message, error)
	}

	// MessageSliceIterator is a wrapper, which lets any slice of Message pointers be used as a MessageIterator.
	MessageSliceIterator struct {
		Target []*Message
		Cursor int
	}

	peekIterator struct {
		entity             entityConnector
		buffer             chan *Message
		lastSequenceNumber int64
	}

	// PeekOption allows customization of parameters when querying a Service Bus entity for messages without committing
	// to processing them.
	PeekOption func(*peekIterator) error
)

const (
	defaultPeekPageSize = 10
)

// AsMessageSliceIterator wraps a slice of Message pointers to allow it to be made into a MessageIterator.
func AsMessageSliceIterator(target []*Message) *MessageSliceIterator {
	return &MessageSliceIterator{
		Target: target,
	}
}

// Done communicates whether there are more messages remaining to be iterated over.
func (ms MessageSliceIterator) Done() bool {
	return ms.Cursor >= len(ms.Target)
}

// Next fetches the Message in the slice at a position one larger than the last one accessed.
func (ms *MessageSliceIterator) Next(_ context.Context) (*Message, error) {
	if ms.Done() {
		return nil, ErrNoMessages{}
	}

	retval := ms.Target[ms.Cursor]
	ms.Cursor++
	return retval, nil
}

func newPeekIterator(entityConnector entityConnector, options ...PeekOption) (*peekIterator, error) {
	retval := &peekIterator{
		entity: entityConnector,
	}

	foundPageSize := false
	for i := range options {
		if err := options[i](retval); err != nil {
			return nil, err
		}

		if retval.buffer != nil {
			foundPageSize = true
		}
	}

	if !foundPageSize {
		err := PeekWithPageSize(defaultPeekPageSize)(retval)
		if err != nil {
			return nil, err
		}
	}

	return retval, nil
}

// PeekWithPageSize adjusts how many messages are fetched at once while peeking from the server.
func PeekWithPageSize(pageSize int) PeekOption {
	return func(pi *peekIterator) error {
		if pageSize < 0 {
			return errors.New("page size must not be less than zero")
		}

		if pi.buffer != nil {
			return errors.New("cannot modify an existing peekIterator's buffer")
		}

		pi.buffer = make(chan *Message, pageSize)
		return nil
	}
}

// PeekFromSequenceNumber adds a filter to the Peek operation, so that no messages with a Sequence Number less than
// 'seq' are returned.
func PeekFromSequenceNumber(seq int64) PeekOption {
	return func(pi *peekIterator) error {
		pi.lastSequenceNumber = seq + 1
		return nil
	}
}

func (pi peekIterator) Done() bool {
	return false
}

func (pi *peekIterator) Next(ctx context.Context) (*Message, error) {
	span, ctx := startConsumerSpanFromContext(ctx, "sb.peekIterator.Next")
	defer span.Finish()

	if len(pi.buffer) == 0 {
		if err := pi.getNextPage(ctx); err != nil {
			return nil, err
		}
	}

	select {
	case next := <-pi.buffer:
		return next, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (pi *peekIterator) getNextPage(ctx context.Context) error {
	span, ctx := startConsumerSpanFromContext(ctx, "sb.peekIterator.getNextPage")
	defer span.Finish()

	const messagesField, messageField = "messages", "message"

	msg := &amqp.Message{
		ApplicationProperties: map[string]interface{}{
			operationFieldName: peekMessageOperationID,
		},
		Value: map[string]interface{}{
			"from-sequence-number": pi.lastSequenceNumber,
			"message-count":        int32(cap(pi.buffer)),
		},
	}

	if deadline, ok := ctx.Deadline(); ok {
		msg.ApplicationProperties["server-timeout"] = uint(time.Until(deadline) / time.Millisecond)
	}

	conn, err := pi.entity.connection(ctx)
	if err != nil {
		log.For(ctx).Error(err)
		return err
	}

	link, err := rpc.NewLink(conn, pi.entity.ManagementPath())
	if err != nil {
		log.For(ctx).Error(err)
		return err
	}

	rsp, err := link.RetryableRPC(ctx, 5, 5*time.Second, msg)
	if err != nil {
		log.For(ctx).Error(err)
		return err
	}

	if rsp.Code == 204 {
		return ErrNoMessages{}
	}

	// Peeked messages come back in a relatively convoluted manner:
	// a map (always with one key: "messages")
	// 	of arrays
	// 		of maps (always with one key: "message")
	// 			of an array with raw encoded Service Bus messages
	val, ok := rsp.Message.Value.(map[string]interface{})
	if !ok {
		err = newErrIncorrectType(messageField, map[string]interface{}{}, rsp.Message.Value)
		log.For(ctx).Error(err)
		return err
	}

	rawMessages, ok := val[messagesField]
	if !ok {
		err = ErrMissingField(messagesField)
		log.For(ctx).Error(err)
		return err
	}

	messages, ok := rawMessages.([]interface{})
	if !ok {
		err = newErrIncorrectType(messagesField, []interface{}{}, rawMessages)
		log.For(ctx).Error(err)
		return err
	}

	transformedMessages := make([]*Message, len(messages))
	for i := range messages {
		rawEntry, ok := messages[i].(map[string]interface{})
		if !ok {
			err = newErrIncorrectType(messageField, map[string]interface{}{}, messages[i])
			log.For(ctx).Error(err)
			return err
		}

		rawMessage, ok := rawEntry[messageField]
		if !ok {
			err = ErrMissingField(messageField)
			log.For(ctx).Error(err)
			return err
		}

		marshaled, ok := rawMessage.([]byte)
		if !ok {
			err = new(ErrMalformedMessage)
			log.For(ctx).Error(err)
			return err
		}

		var rehydrated amqp.Message
		err = rehydrated.UnmarshalBinary(marshaled)
		if err != nil {
			log.For(ctx).Error(err)
			return err
		}

		transformedMessages[i], err = messageFromAMQPMessage(&rehydrated)
		if err != nil {
			log.For(ctx).Error(err)
			return err
		}
	}

	// This sort is done to ensure that folks wanting to peek messages in sequence order may do so.
	sort.Slice(transformedMessages, func(i, j int) bool {
		iSeq := *transformedMessages[i].SystemProperties.SequenceNumber
		jSeq := *transformedMessages[j].SystemProperties.SequenceNumber
		return iSeq < jSeq
	})

	for i := range transformedMessages {
		select {
		case pi.buffer <- transformedMessages[i]:
			// Intentionally Left Blank
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Update last seen sequence number so that the next read starts from where this ended.
	pi.lastSequenceNumber = *transformedMessages[len(transformedMessages)-1].SystemProperties.SequenceNumber + 1
	return nil
}
