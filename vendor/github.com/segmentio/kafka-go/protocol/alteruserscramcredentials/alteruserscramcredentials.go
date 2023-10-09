package alteruserscramcredentials

import "github.com/segmentio/kafka-go/protocol"

func init() {
	protocol.Register(&Request{}, &Response{})
}

type Request struct {
	// We need at least one tagged field to indicate that v2+ uses "flexible"
	// messages.
	_ struct{} `kafka:"min=v0,max=v0,tag"`

	Deletions  []RequestUserScramCredentialsDeletion  `kafka:"min=v0,max=v0"`
	Upsertions []RequestUserScramCredentialsUpsertion `kafka:"min=v0,max=v0"`
}

func (r *Request) ApiKey() protocol.ApiKey { return protocol.AlterUserScramCredentials }

func (r *Request) Broker(cluster protocol.Cluster) (protocol.Broker, error) {
	return cluster.Brokers[cluster.Controller], nil
}

type RequestUserScramCredentialsDeletion struct {
	// We need at least one tagged field to indicate that v2+ uses "flexible"
	// messages.
	_ struct{} `kafka:"min=v0,max=v0,tag"`

	Name      string `kafka:"min=v0,max=v0,compact"`
	Mechanism int8   `kafka:"min=v0,max=v0"`
}

type RequestUserScramCredentialsUpsertion struct {
	// We need at least one tagged field to indicate that v2+ uses "flexible"
	// messages.
	_ struct{} `kafka:"min=v0,max=v0,tag"`

	Name           string `kafka:"min=v0,max=v0,compact"`
	Mechanism      int8   `kafka:"min=v0,max=v0"`
	Iterations     int32  `kafka:"min=v0,max=v0"`
	Salt           []byte `kafka:"min=v0,max=v0,compact"`
	SaltedPassword []byte `kafka:"min=v0,max=v0,compact"`
}

type Response struct {
	// We need at least one tagged field to indicate that v2+ uses "flexible"
	// messages.
	_ struct{} `kafka:"min=v0,max=v0,tag"`

	ThrottleTimeMs int32                          `kafka:"min=v0,max=v0"`
	Results        []ResponseUserScramCredentials `kafka:"min=v0,max=v0"`
}

func (r *Response) ApiKey() protocol.ApiKey { return protocol.AlterUserScramCredentials }

type ResponseUserScramCredentials struct {
	// We need at least one tagged field to indicate that v2+ uses "flexible"
	// messages.
	_ struct{} `kafka:"min=v0,max=v0,tag"`

	User         string `kafka:"min=v0,max=v0,compact"`
	ErrorCode    int16  `kafka:"min=v0,max=v0"`
	ErrorMessage string `kafka:"min=v0,max=v0,nullable"`
}

var _ protocol.BrokerMessage = (*Request)(nil)
