package kafka

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/segmentio/kafka-go/protocol/describeclientquotas"
)

// DescribeClientQuotasRequest represents a request sent to a kafka broker to
// describe client quotas.
type DescribeClientQuotasRequest struct {
	// Address of the kafka broker to send the request to
	Addr net.Addr

	// List of quota components to describe.
	Components []DescribeClientQuotasRequestComponent

	// Whether the match is strict, i.e. should exclude entities with
	// unspecified entity types.
	Strict bool
}

type DescribeClientQuotasRequestComponent struct {
	// The entity type that the filter component applies to.
	EntityType string

	// How to match the entity (0 = exact name, 1 = default name,
	// 2 = any specified name).
	MatchType int8

	// The string to match against, or null if unused for the match type.
	Match string
}

// DescribeClientQuotasReesponse represents a response from a kafka broker to a describe client quota request.
type DescribeClientQuotasResponse struct {
	// The amount of time that the broker throttled the request.
	Throttle time.Duration

	// Error is set to a non-nil value including the code and message if a top-level
	// error was encountered when doing the update.
	Error error

	// List of describe client quota responses.
	Entries []DescribeClientQuotasResponseQuotas
}

type DescribeClientQuotasEntity struct {
	// The quota entity type.
	EntityType string

	// The name of the quota entity, or null if the default.
	EntityName string
}

type DescribeClientQuotasValue struct {
	// The quota configuration key.
	Key string

	// The quota configuration value.
	Value float64
}

type DescribeClientQuotasResponseQuotas struct {
	// List of client quota entities and their descriptions.
	Entities []DescribeClientQuotasEntity

	// The client quota configuration values.
	Values []DescribeClientQuotasValue
}

// DescribeClientQuotas sends a describe client quotas request to a kafka broker and returns
// the response.
func (c *Client) DescribeClientQuotas(ctx context.Context, req *DescribeClientQuotasRequest) (*DescribeClientQuotasResponse, error) {
	components := make([]describeclientquotas.Component, len(req.Components))

	for componentIdx, component := range req.Components {
		components[componentIdx] = describeclientquotas.Component{
			EntityType: component.EntityType,
			MatchType:  component.MatchType,
			Match:      component.Match,
		}
	}

	m, err := c.roundTrip(ctx, req.Addr, &describeclientquotas.Request{
		Components: components,
		Strict:     req.Strict,
	})
	if err != nil {
		return nil, fmt.Errorf("kafka.(*Client).DescribeClientQuotas: %w", err)
	}

	res := m.(*describeclientquotas.Response)
	responseEntries := make([]DescribeClientQuotasResponseQuotas, len(res.Entries))

	for responseEntryIdx, responseEntry := range res.Entries {
		responseEntities := make([]DescribeClientQuotasEntity, len(responseEntry.Entities))
		for responseEntityIdx, responseEntity := range responseEntry.Entities {
			responseEntities[responseEntityIdx] = DescribeClientQuotasEntity{
				EntityType: responseEntity.EntityType,
				EntityName: responseEntity.EntityName,
			}
		}

		responseValues := make([]DescribeClientQuotasValue, len(responseEntry.Values))
		for responseValueIdx, responseValue := range responseEntry.Values {
			responseValues[responseValueIdx] = DescribeClientQuotasValue{
				Key:   responseValue.Key,
				Value: responseValue.Value,
			}
		}
		responseEntries[responseEntryIdx] = DescribeClientQuotasResponseQuotas{
			Entities: responseEntities,
			Values:   responseValues,
		}
	}
	ret := &DescribeClientQuotasResponse{
		Throttle: time.Duration(res.ThrottleTimeMs),
		Entries:  responseEntries,
	}

	return ret, nil
}
