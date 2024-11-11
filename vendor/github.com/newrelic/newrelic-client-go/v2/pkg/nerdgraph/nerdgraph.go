// Package nerdgraph provides a programmatic API for interacting with NerdGraph, New Relic One's GraphQL API.
package nerdgraph

import (
	"context"

	"github.com/newrelic/newrelic-client-go/v2/internal/http"
	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
	"github.com/newrelic/newrelic-client-go/v2/pkg/logging"
)

// NerdGraph is used to communicate with the New Relic's GraphQL API, NerdGraph.
type NerdGraph struct {
	client http.Client
	logger logging.Logger
}

// QueryResponse represents the top-level GraphQL response object returned
// from a NerdGraph query request.
type QueryResponse struct {
	Actor          interface{} `json:"actor,omitempty" yaml:"actor,omitempty"`
	Docs           interface{} `json:"docs,omitempty" yaml:"docs,omitempty"`
	RequestContext interface{} `json:"requestContext,omitempty" yaml:"requestContext,omitempty"`
}

// New returns a new GraphQL client for interacting with New Relic's GraphQL API, NerdGraph.
func New(config config.Config) NerdGraph {
	return NerdGraph{
		client: http.NewClient(config),
		logger: config.GetLogger(),
	}
}

// Query facilitates making a NerdGraph request with a raw GraphQL query. Variables may be provided
// in the form of a map. The response's data structure will vary based on the query provided.
func (n *NerdGraph) Query(query string, variables map[string]interface{}) (interface{}, error) {
	return n.QueryWithContext(context.Background(), query, variables)
}

// QueryWithContext facilitates making a NerdGraph request with a raw GraphQL query. Variables may be provided
// in the form of a map. The response's data structure will vary based on the query provided.
func (n *NerdGraph) QueryWithContext(ctx context.Context, query string, variables map[string]interface{}) (interface{}, error) {
	respBody := QueryResponse{}

	if err := n.QueryWithResponseAndContext(ctx, query, variables, &respBody); err != nil {
		return nil, err
	}

	return respBody, nil
}

// QueryWithResponse functions similarly to Query, but alows for full customization of the returned data payload.
// Query should be preferred most of the time.
func (n *NerdGraph) QueryWithResponse(query string, variables map[string]interface{}, respBody interface{}) error {
	return n.QueryWithResponseAndContext(context.Background(), query, variables, respBody)
}

// QueryWithResponseAndContext functions similarly to QueryWithContext, but alows for full customization of the returned data payload.
// QueryWithContext should be preferred most of the time.
func (n *NerdGraph) QueryWithResponseAndContext(ctx context.Context, query string, variables map[string]interface{}, respBody interface{}) error {
	return n.client.NerdGraphQueryWithContext(ctx, query, variables, respBody)
}

// AccountReference represents the NerdGraph schema for a New Relic account.
type AccountReference struct {
	ID   int    `json:"id,omitempty" yaml:"id,omitempty"`
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
}
