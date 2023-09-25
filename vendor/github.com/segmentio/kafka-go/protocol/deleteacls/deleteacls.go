package deleteacls

import "github.com/segmentio/kafka-go/protocol"

func init() {
	protocol.Register(&Request{}, &Response{})
}

type Request struct {
	// We need at least one tagged field to indicate that v2+ uses "flexible"
	// messages.
	_ struct{} `kafka:"min=v2,max=v3,tag"`

	Filters []RequestFilter `kafka:"min=v0,max=v3"`
}

func (r *Request) ApiKey() protocol.ApiKey { return protocol.DeleteAcls }

func (r *Request) Broker(cluster protocol.Cluster) (protocol.Broker, error) {
	return cluster.Brokers[cluster.Controller], nil
}

type RequestFilter struct {
	// We need at least one tagged field to indicate that v2+ uses "flexible"
	// messages.
	_ struct{} `kafka:"min=v2,max=v3,tag"`

	ResourceTypeFilter        int8   `kafka:"min=v0,max=v3"`
	ResourceNameFilter        string `kafka:"min=v0,max=v1,nullable|min=v2,max=v3,nullable,compact"`
	ResourcePatternTypeFilter int8   `kafka:"min=v1,max=v3"`
	PrincipalFilter           string `kafka:"min=v0,max=v1,nullable|min=v2,max=v3,nullable,compact"`
	HostFilter                string `kafka:"min=v0,max=v1,nullable|min=v2,max=v3,nullable,compact"`
	Operation                 int8   `kafka:"min=v0,max=v3"`
	PermissionType            int8   `kafka:"min=v0,max=v3"`
}

type Response struct {
	// We need at least one tagged field to indicate that v2+ uses "flexible"
	// messages.
	_ struct{} `kafka:"min=v2,max=v3,tag"`

	ThrottleTimeMs int32          `kafka:"min=v0,max=v3"`
	FilterResults  []FilterResult `kafka:"min=v0,max=v3"`
}

func (r *Response) ApiKey() protocol.ApiKey { return protocol.DeleteAcls }

type FilterResult struct {
	// We need at least one tagged field to indicate that v2+ uses "flexible"
	// messages.
	_ struct{} `kafka:"min=v2,max=v3,tag"`

	ErrorCode    int16         `kafka:"min=v0,max=v3"`
	ErrorMessage string        `kafka:"min=v0,max=v1,nullable|min=v2,max=v3,nullable,compact"`
	MatchingACLs []MatchingACL `kafka:"min=v0,max=v3"`
}

type MatchingACL struct {
	// We need at least one tagged field to indicate that v2+ uses "flexible"
	// messages.
	_ struct{} `kafka:"min=v2,max=v3,tag"`

	ErrorCode           int16  `kafka:"min=v0,max=v3"`
	ErrorMessage        string `kafka:"min=v0,max=v1,nullable|min=v2,max=v3,nullable,compact"`
	ResourceType        int8   `kafka:"min=v0,max=v3"`
	ResourceName        string `kafka:"min=v0,max=v1|min=v2,max=v3,compact"`
	ResourcePatternType int8   `kafka:"min=v1,max=v3"`
	Principal           string `kafka:"min=v0,max=v1|min=v2,max=v3,compact"`
	Host                string `kafka:"min=v0,max=v1|min=v2,max=v3,compact"`
	Operation           int8   `kafka:"min=v0,max=v3"`
	PermissionType      int8   `kafka:"min=v0,max=v3"`
}

var _ protocol.BrokerMessage = (*Request)(nil)
