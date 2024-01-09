package kafka

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/segmentio/kafka-go/protocol"
	produceAPI "github.com/segmentio/kafka-go/protocol/produce"
	"github.com/segmentio/kafka-go/protocol/rawproduce"
)

// RawProduceRequest represents a request sent to a kafka broker to produce records
// to a topic partition. The request contains a pre-encoded/raw record set.
type RawProduceRequest struct {
	// Address of the kafka broker to send the request to.
	Addr net.Addr

	// The topic to produce the records to.
	Topic string

	// The partition to produce the records to.
	Partition int

	// The level of required acknowledgements to ask the kafka broker for.
	RequiredAcks RequiredAcks

	// The message format version used when encoding the records.
	//
	// By default, the client automatically determine which version should be
	// used based on the version of the Produce API supported by the server.
	MessageVersion int

	// An optional transaction id when producing to the kafka broker is part of
	// a transaction.
	TransactionalID string

	// The sequence of records to produce to the topic partition.
	RawRecords protocol.RawRecordSet
}

// RawProduce sends a raw produce request to a kafka broker and returns the response.
//
// If the request contained no records, an error wrapping protocol.ErrNoRecord
// is returned.
//
// When the request is configured with RequiredAcks=none, both the response and
// the error will be nil on success.
func (c *Client) RawProduce(ctx context.Context, req *RawProduceRequest) (*ProduceResponse, error) {
	m, err := c.roundTrip(ctx, req.Addr, &rawproduce.Request{
		TransactionalID: req.TransactionalID,
		Acks:            int16(req.RequiredAcks),
		Timeout:         c.timeoutMs(ctx, defaultProduceTimeout),
		Topics: []rawproduce.RequestTopic{{
			Topic: req.Topic,
			Partitions: []rawproduce.RequestPartition{{
				Partition: int32(req.Partition),
				RecordSet: req.RawRecords,
			}},
		}},
	})

	switch {
	case err == nil:
	case errors.Is(err, protocol.ErrNoRecord):
		return new(ProduceResponse), nil
	default:
		return nil, fmt.Errorf("kafka.(*Client).RawProduce: %w", err)
	}

	if req.RequiredAcks == RequireNone {
		return nil, nil
	}

	res := m.(*produceAPI.Response)
	if len(res.Topics) == 0 {
		return nil, fmt.Errorf("kafka.(*Client).RawProduce: %w", protocol.ErrNoTopic)
	}
	topic := &res.Topics[0]
	if len(topic.Partitions) == 0 {
		return nil, fmt.Errorf("kafka.(*Client).RawProduce: %w", protocol.ErrNoPartition)
	}
	partition := &topic.Partitions[0]

	ret := &ProduceResponse{
		Throttle:       makeDuration(res.ThrottleTimeMs),
		Error:          makeError(partition.ErrorCode, partition.ErrorMessage),
		BaseOffset:     partition.BaseOffset,
		LogAppendTime:  makeTime(partition.LogAppendTime),
		LogStartOffset: partition.LogStartOffset,
	}

	if len(partition.RecordErrors) != 0 {
		ret.RecordErrors = make(map[int]error, len(partition.RecordErrors))

		for _, recErr := range partition.RecordErrors {
			ret.RecordErrors[int(recErr.BatchIndex)] = errors.New(recErr.BatchIndexErrorMessage)
		}
	}

	return ret, nil
}
