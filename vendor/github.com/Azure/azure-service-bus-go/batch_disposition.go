package servicebus

import (
	"context"
	"fmt"

	"github.com/Azure/azure-amqp-common-go/uuid"
)

type (
	// MessageStatus defines an acceptable Message disposition status.
	MessageStatus dispositionStatus
	// BatchDispositionIterator provides an iterator over LockTokenIDs
	BatchDispositionIterator struct {
		LockTokenIDs []*uuid.UUID
		Status       MessageStatus
		cursor       int
	}
)

const (
	// Complete exposes completedDisposition
	Complete MessageStatus = MessageStatus(completedDisposition)
	// Abort exposes abandonedDisposition
	Abort MessageStatus = MessageStatus(abandonedDisposition)
)

// Done communicates whether there are more messages remaining to be iterated over.
func (bdi *BatchDispositionIterator) Done() bool {
	return len(bdi.LockTokenIDs) == bdi.cursor
}

// Next iterates to the next LockToken
func (bdi *BatchDispositionIterator) Next() (uuid *uuid.UUID) {
	if done := bdi.Done(); done == false {
		uuid = bdi.LockTokenIDs[bdi.cursor]
		bdi.cursor++
	}
	return uuid
}

func (bdi *BatchDispositionIterator) doUpdate(ctx context.Context, ec entityConnector) error {
	for !bdi.Done() {
		if uuid := bdi.Next(); uuid != nil {
			m := &Message{
				LockToken: uuid,
			}
			m.ec = ec
			err := m.sendDisposition(ctx, bdi.Status)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// SendBatchDisposition updates the LockToken id to the desired status.
func (q *Queue) SendBatchDisposition(ctx context.Context, iterator BatchDispositionIterator) error {
	span, ctx := q.startSpanFromContext(ctx, "sb.Queue.SendBatchDisposition")
	defer span.Finish()
	return iterator.doUpdate(ctx, q)
}

// SendBatchDisposition updates the LockToken id to the desired status.
func (s *Subscription) SendBatchDisposition(ctx context.Context, iterator BatchDispositionIterator) error {
	span, ctx := s.startSpanFromContext(ctx, "sb.Subscription.SendBatchDisposition")
	defer span.Finish()
	return iterator.doUpdate(ctx, s)
}

func (m *Message) sendDisposition(ctx context.Context, dispositionStatus MessageStatus) (err error) {
	switch dispositionStatus {
	case Complete:
		err = m.Complete(ctx)
	case Abort:
		err = m.Abandon(ctx)
	default:
		err = fmt.Errorf("unsupported bulk disposition status %q", dispositionStatus)
	}
	return err
}
