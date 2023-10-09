package kafka

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/segmentio/kafka-go/protocol/deletegroups"
)

// DeleteGroupsRequest represents a request sent to a kafka broker to delete
// consumer groups.
type DeleteGroupsRequest struct {
	// Address of the kafka broker to send the request to.
	Addr net.Addr

	// Identifiers of groups to delete.
	GroupIDs []string
}

// DeleteGroupsResponse represents a response from a kafka broker to a consumer group
// deletion request.
type DeleteGroupsResponse struct {
	// The amount of time that the broker throttled the request.
	Throttle time.Duration

	// Mapping of group ids to errors that occurred while attempting to delete those groups.
	//
	// The errors contain the kafka error code. Programs may use the standard
	// errors.Is function to test the error against kafka error codes.
	Errors map[string]error
}

// DeleteGroups sends a delete groups request and returns the response. The request is sent to the group coordinator of the first group
// of the request. All deleted groups must be managed by the same group coordinator.
func (c *Client) DeleteGroups(
	ctx context.Context,
	req *DeleteGroupsRequest,
) (*DeleteGroupsResponse, error) {
	m, err := c.roundTrip(ctx, req.Addr, &deletegroups.Request{
		GroupIDs: req.GroupIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("kafka.(*Client).DeleteGroups: %w", err)
	}

	r := m.(*deletegroups.Response)

	ret := &DeleteGroupsResponse{
		Throttle: makeDuration(r.ThrottleTimeMs),
		Errors:   make(map[string]error, len(r.Responses)),
	}

	for _, t := range r.Responses {
		ret.Errors[t.GroupID] = makeError(t.ErrorCode, "")
	}

	return ret, nil
}
