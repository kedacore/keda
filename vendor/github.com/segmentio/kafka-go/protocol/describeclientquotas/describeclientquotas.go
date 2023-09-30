package describeclientquotas

import "github.com/segmentio/kafka-go/protocol"

func init() {
	protocol.Register(&Request{}, &Response{})
}

type Request struct {
	// We need at least one tagged field to indicate that this is a "flexible" message
	// type.
	_          struct{}    `kafka:"min=v1,max=v1,tag"`
	Components []Component `kafka:"min=v0,max=v1"`
	Strict     bool        `kafka:"min=v0,max=v1"`
}

func (r *Request) ApiKey() protocol.ApiKey { return protocol.DescribeClientQuotas }

func (r *Request) Broker(cluster protocol.Cluster) (protocol.Broker, error) {
	return cluster.Brokers[cluster.Controller], nil
}

type Component struct {
	// We need at least one tagged field to indicate that this is a "flexible" message
	// type.
	_          struct{} `kafka:"min=v1,max=v1,tag"`
	EntityType string   `kafka:"min=v0,max=v1"`
	MatchType  int8     `kafka:"min=v0,max=v1"`
	Match      string   `kafka:"min=v0,max=v1,nullable"`
}

type Response struct {
	// We need at least one tagged field to indicate that this is a "flexible" message
	// type.
	_              struct{}         `kafka:"min=v1,max=v1,tag"`
	ThrottleTimeMs int32            `kafka:"min=v0,max=v1"`
	ErrorCode      int16            `kafka:"min=v0,max=v1"`
	ErrorMessage   string           `kafka:"min=v0,max=v0,nullable|min=v1,max=v1,nullable,compact"`
	Entries        []ResponseQuotas `kafka:"min=v0,max=v1"`
}

func (r *Response) ApiKey() protocol.ApiKey { return protocol.DescribeClientQuotas }

type Entity struct {
	// We need at least one tagged field to indicate that this is a "flexible" message
	// type.
	_          struct{} `kafka:"min=v1,max=v1,tag"`
	EntityType string   `kafka:"min=v0,max=v0|min=v1,max=v1,compact"`
	EntityName string   `kafka:"min=v0,max=v0,nullable|min=v1,max=v1,nullable,compact"`
}

type Value struct {
	// We need at least one tagged field to indicate that this is a "flexible" message
	// type.
	_     struct{} `kafka:"min=v1,max=v1,tag"`
	Key   string   `kafka:"min=v0,max=v0|min=v1,max=v1,compact"`
	Value float64  `kafka:"min=v0,max=v1"`
}

type ResponseQuotas struct {
	// We need at least one tagged field to indicate that this is a "flexible" message
	// type.
	_        struct{} `kafka:"min=v1,max=v1,tag"`
	Entities []Entity `kafka:"min=v0,max=v1"`
	Values   []Value  `kafka:"min=v0,max=v1"`
}

var _ protocol.BrokerMessage = (*Request)(nil)
