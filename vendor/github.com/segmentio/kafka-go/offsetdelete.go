package kafka

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/segmentio/kafka-go/protocol/offsetdelete"
)

// OffsetDelete deletes the offset for a consumer group on a particular topic
// for a particular partition.
type OffsetDelete struct {
	Topic     string
	Partition int
}

// OffsetDeleteRequest represents a request sent to a kafka broker to delete
// the offsets for a partition on a given topic associated with a consumer group.
type OffsetDeleteRequest struct {
	// Address of the kafka broker to send the request to.
	Addr net.Addr

	// ID of the consumer group to delete the offsets for.
	GroupID string

	// Set of topic partitions to delete offsets for.
	Topics map[string][]int
}

// OffsetDeleteResponse represents a response from a kafka broker to a delete
// offset request.
type OffsetDeleteResponse struct {
	// An error that may have occurred while attempting to delete an offset
	Error error

	// The amount of time that the broker throttled the request.
	Throttle time.Duration

	// Set of topic partitions that the kafka broker has additional info (error?)
	// for.
	Topics map[string][]OffsetDeletePartition
}

// OffsetDeletePartition represents the state of a status of a partition in response
// to deleting offsets.
type OffsetDeletePartition struct {
	// ID of the partition.
	Partition int

	// An error that may have occurred while attempting to delete an offset for
	// this partition.
	Error error
}

// OffsetDelete sends a delete offset request to a kafka broker and returns the
// response.
func (c *Client) OffsetDelete(ctx context.Context, req *OffsetDeleteRequest) (*OffsetDeleteResponse, error) {
	topics := make([]offsetdelete.RequestTopic, 0, len(req.Topics))

	for topicName, partitionIndexes := range req.Topics {
		partitions := make([]offsetdelete.RequestPartition, len(partitionIndexes))

		for i, c := range partitionIndexes {
			partitions[i] = offsetdelete.RequestPartition{
				PartitionIndex: int32(c),
			}
		}

		topics = append(topics, offsetdelete.RequestTopic{
			Name:       topicName,
			Partitions: partitions,
		})
	}

	m, err := c.roundTrip(ctx, req.Addr, &offsetdelete.Request{
		GroupID: req.GroupID,
		Topics:  topics,
	})
	if err != nil {
		return nil, fmt.Errorf("kafka.(*Client).OffsetDelete: %w", err)
	}
	r := m.(*offsetdelete.Response)

	res := &OffsetDeleteResponse{
		Error:    makeError(r.ErrorCode, ""),
		Throttle: makeDuration(r.ThrottleTimeMs),
		Topics:   make(map[string][]OffsetDeletePartition, len(r.Topics)),
	}

	for _, topic := range r.Topics {
		partitions := make([]OffsetDeletePartition, len(topic.Partitions))

		for i, p := range topic.Partitions {
			partitions[i] = OffsetDeletePartition{
				Partition: int(p.PartitionIndex),
				Error:     makeError(p.ErrorCode, ""),
			}
		}

		res.Topics[topic.Name] = partitions
	}

	return res, nil
}
