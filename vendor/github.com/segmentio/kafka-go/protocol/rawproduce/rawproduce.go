package rawproduce

import (
	"fmt"

	"github.com/segmentio/kafka-go/protocol"
	"github.com/segmentio/kafka-go/protocol/produce"
)

func init() {
	// Register a type override so that raw produce requests will be encoded with the correct type.
	req := &Request{}
	protocol.RegisterOverride(req, &produce.Response{}, req.TypeKey())
}

type Request struct {
	TransactionalID string         `kafka:"min=v3,max=v8,nullable"`
	Acks            int16          `kafka:"min=v0,max=v8"`
	Timeout         int32          `kafka:"min=v0,max=v8"`
	Topics          []RequestTopic `kafka:"min=v0,max=v8"`
}

func (r *Request) ApiKey() protocol.ApiKey { return protocol.Produce }

func (r *Request) TypeKey() protocol.OverrideTypeKey { return protocol.RawProduceOverride }

func (r *Request) Broker(cluster protocol.Cluster) (protocol.Broker, error) {
	broker := protocol.Broker{ID: -1}

	for i := range r.Topics {
		t := &r.Topics[i]

		topic, ok := cluster.Topics[t.Topic]
		if !ok {
			return broker, NewError(protocol.NewErrNoTopic(t.Topic))
		}

		for j := range t.Partitions {
			p := &t.Partitions[j]

			partition, ok := topic.Partitions[p.Partition]
			if !ok {
				return broker, NewError(protocol.NewErrNoPartition(t.Topic, p.Partition))
			}

			if b, ok := cluster.Brokers[partition.Leader]; !ok {
				return broker, NewError(protocol.NewErrNoLeader(t.Topic, p.Partition))
			} else if broker.ID < 0 {
				broker = b
			} else if b.ID != broker.ID {
				return broker, NewError(fmt.Errorf("mismatching leaders (%d!=%d)", b.ID, broker.ID))
			}
		}
	}

	return broker, nil
}

func (r *Request) HasResponse() bool {
	return r.Acks != 0
}

type RequestTopic struct {
	Topic      string             `kafka:"min=v0,max=v8"`
	Partitions []RequestPartition `kafka:"min=v0,max=v8"`
}

type RequestPartition struct {
	Partition int32                 `kafka:"min=v0,max=v8"`
	RecordSet protocol.RawRecordSet `kafka:"min=v0,max=v8"`
}

var (
	_ protocol.BrokerMessage = (*Request)(nil)
)

type Error struct {
	Err error
}

func NewError(err error) *Error {
	return &Error{Err: err}
}

func (e *Error) Error() string {
	return fmt.Sprintf("fetch request error: %v", e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}
