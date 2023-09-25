package offsetdelete

import "github.com/segmentio/kafka-go/protocol"

func init() {
	protocol.Register(&Request{}, &Response{})
}

type Request struct {
	GroupID string         `kafka:"min=v0,max=v0"`
	Topics  []RequestTopic `kafka:"min=v0,max=v0"`
}

func (r *Request) ApiKey() protocol.ApiKey { return protocol.OffsetDelete }

func (r *Request) Group() string { return r.GroupID }

type RequestTopic struct {
	Name       string             `kafka:"min=v0,max=v0"`
	Partitions []RequestPartition `kafka:"min=v0,max=v0"`
}

type RequestPartition struct {
	PartitionIndex int32 `kafka:"min=v0,max=v0"`
}

var (
	_ protocol.GroupMessage = (*Request)(nil)
)

type Response struct {
	ErrorCode      int16           `kafka:"min=v0,max=v0"`
	ThrottleTimeMs int32           `kafka:"min=v0,max=v0"`
	Topics         []ResponseTopic `kafka:"min=v0,max=v0"`
}

func (r *Response) ApiKey() protocol.ApiKey { return protocol.OffsetDelete }

type ResponseTopic struct {
	Name       string              `kafka:"min=v0,max=v0"`
	Partitions []ResponsePartition `kafka:"min=v0,max=v0"`
}

type ResponsePartition struct {
	PartitionIndex int32 `kafka:"min=v0,max=v0"`
	ErrorCode      int16 `kafka:"min=v0,max=v0"`
}
