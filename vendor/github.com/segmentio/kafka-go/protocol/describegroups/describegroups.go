package describegroups

import (
	"github.com/segmentio/kafka-go/protocol"
)

func init() {
	protocol.Register(&Request{}, &Response{})
}

// Detailed API definition: https://kafka.apache.org/protocol#The_Messages_DescribeGroups
type Request struct {
	// We need at least one tagged field to indicate that this is a "flexible" message
	// type.
	_                           struct{} `kafka:"min=v5,max=v5,tag"`
	Groups                      []string `kafka:"min=v0,max=v4|min=v5,max=v5,compact"`
	IncludeAuthorizedOperations bool     `kafka:"min=v3,max=v5"`
}

func (r *Request) ApiKey() protocol.ApiKey { return protocol.DescribeGroups }

func (r *Request) Group() string {
	return r.Groups[0]
}

func (r *Request) Split(cluster protocol.Cluster) (
	[]protocol.Message,
	protocol.Merger,
	error,
) {
	messages := []protocol.Message{}

	// Split requests by group since they'll need to go to different coordinators.
	for _, group := range r.Groups {
		messages = append(
			messages,
			&Request{
				Groups:                      []string{group},
				IncludeAuthorizedOperations: r.IncludeAuthorizedOperations,
			},
		)
	}

	return messages, new(Response), nil
}

type Response struct {
	// We need at least one tagged field to indicate that this is a "flexible" message
	// type.
	_              struct{}        `kafka:"min=v5,max=v5,tag"`
	ThrottleTimeMs int32           `kafka:"min=v1,max=v5"`
	Groups         []ResponseGroup `kafka:"min=v0,max=v5"`
}

type ResponseGroup struct {
	// We need at least one tagged field to indicate that this is a "flexible" message
	// type.
	_                    struct{}              `kafka:"min=v5,max=v5,tag"`
	ErrorCode            int16                 `kafka:"min=v0,max=v5"`
	GroupID              string                `kafka:"min=v0,max=v4|min=v5,max=v5,compact"`
	GroupState           string                `kafka:"min=v0,max=v4|min=v5,max=v5,compact"`
	ProtocolType         string                `kafka:"min=v0,max=v4|min=v5,max=v5,compact"`
	ProtocolData         string                `kafka:"min=v0,max=v4|min=v5,max=v5,compact"`
	Members              []ResponseGroupMember `kafka:"min=v0,max=v4|min=v5,max=v5,compact"`
	AuthorizedOperations int32                 `kafka:"min=v3,max=v5"`
}

type ResponseGroupMember struct {
	// We need at least one tagged field to indicate that this is a "flexible" message
	// type.
	_                struct{} `kafka:"min=v5,max=v5,tag"`
	MemberID         string   `kafka:"min=v0,max=v4|min=v5,max=v5,compact"`
	GroupInstanceID  string   `kafka:"min=v4,max=v4,nullable|min=v5,max=v5,compact,nullable"`
	ClientID         string   `kafka:"min=v0,max=v4|min=v5,max=v5,compact"`
	ClientHost       string   `kafka:"min=v0,max=v4|min=v5,max=v5,compact"`
	MemberMetadata   []byte   `kafka:"min=v0,max=v4|min=v5,max=v5,compact"`
	MemberAssignment []byte   `kafka:"min=v0,max=v4|min=v5,max=v5,compact"`
}

func (r *Response) ApiKey() protocol.ApiKey { return protocol.DescribeGroups }

func (r *Response) Merge(requests []protocol.Message, results []interface{}) (
	protocol.Message,
	error,
) {
	response := &Response{}

	for _, result := range results {
		m, err := protocol.Result(result)
		if err != nil {
			return nil, err
		}
		response.Groups = append(response.Groups, m.(*Response).Groups...)
	}

	return response, nil
}
