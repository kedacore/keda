package deletegroups

import "github.com/segmentio/kafka-go/protocol"

func init() {
	protocol.Register(&Request{}, &Response{})
}

type Request struct {
	// We need at least one tagged field to indicate that this is a "flexible" message
	// type.
	_ struct{} `kafka:"min=v2,max=v2,tag"`

	GroupIDs []string `kafka:"min=v0,max=v2"`
}

func (r *Request) Group() string {
	// use first group to determine group coordinator
	if len(r.GroupIDs) > 0 {
		return r.GroupIDs[0]
	}
	return ""
}

func (r *Request) ApiKey() protocol.ApiKey { return protocol.DeleteGroups }

var (
	_ protocol.GroupMessage = (*Request)(nil)
)

type Response struct {
	// We need at least one tagged field to indicate that this is a "flexible" message
	// type.
	_ struct{} `kafka:"min=v2,max=v2,tag"`

	ThrottleTimeMs int32           `kafka:"min=v0,max=v2"`
	Responses      []ResponseGroup `kafka:"min=v0,max=v2"`
}

func (r *Response) ApiKey() protocol.ApiKey { return protocol.DeleteGroups }

type ResponseGroup struct {
	GroupID   string `kafka:"min=v0,max=v2"`
	ErrorCode int16  `kafka:"min=v0,max=v2"`
}
