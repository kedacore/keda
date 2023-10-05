package describeuserscramcredentials

import "github.com/segmentio/kafka-go/protocol"

func init() {
	protocol.Register(&Request{}, &Response{})
}

type Request struct {
	// We need at least one tagged field to indicate that v2+ uses "flexible"
	// messages.
	_ struct{} `kafka:"min=v0,max=v0,tag"`

	Users []RequestUser `kafka:"min=v0,max=v0"`
}

func (r *Request) ApiKey() protocol.ApiKey { return protocol.DescribeUserScramCredentials }

func (r *Request) Broker(cluster protocol.Cluster) (protocol.Broker, error) {
	return cluster.Brokers[cluster.Controller], nil
}

type RequestUser struct {
	// We need at least one tagged field to indicate that v2+ uses "flexible"
	// messages.
	_ struct{} `kafka:"min=v0,max=v0,tag"`

	Name string `kafka:"min=v0,max=v0,compact"`
}

type Response struct {
	// We need at least one tagged field to indicate that v2+ uses "flexible"
	// messages.
	_ struct{} `kafka:"min=v0,max=v0,tag"`

	ThrottleTimeMs int32            `kafka:"min=v0,max=v0"`
	ErrorCode      int16            `kafka:"min=v0,max=v0"`
	ErrorMessage   string           `kafka:"min=v0,max=v0,nullable"`
	Results        []ResponseResult `kafka:"min=v0,max=v0"`
}

func (r *Response) ApiKey() protocol.ApiKey { return protocol.DescribeUserScramCredentials }

type ResponseResult struct {
	// We need at least one tagged field to indicate that v2+ uses "flexible"
	// messages.
	_ struct{} `kafka:"min=v0,max=v0,tag"`

	User            string           `kafka:"min=v0,max=v0,compact"`
	ErrorCode       int16            `kafka:"min=v0,max=v0"`
	ErrorMessage    string           `kafka:"min=v0,max=v0,nullable"`
	CredentialInfos []CredentialInfo `kafka:"min=v0,max=v0"`
}

type CredentialInfo struct {
	// We need at least one tagged field to indicate that v2+ uses "flexible"
	// messages.
	_ struct{} `kafka:"min=v0,max=v0,tag"`

	Mechanism  int8  `kafka:"min=v0,max=v0"`
	Iterations int32 `kafka:"min=v0,max=v0"`
}

var _ protocol.BrokerMessage = (*Request)(nil)
