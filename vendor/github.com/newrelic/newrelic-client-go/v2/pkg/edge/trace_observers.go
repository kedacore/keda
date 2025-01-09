package edge

import (
	"context"
	"fmt"
)

// ListTraceObservers lists the trace observers for an account.
func (e *Edge) ListTraceObservers(accountID int) ([]EdgeTraceObserver, error) {
	return e.ListTraceObserversWithContext(context.Background(), accountID)
}

// ListTraceObserversWithContext lists the trace observers for an account.
func (e *Edge) ListTraceObserversWithContext(ctx context.Context, accountID int) ([]EdgeTraceObserver, error) {
	resp := traceObserverResponse{}
	vars := map[string]interface{}{
		"accountId": accountID,
	}

	if err := e.client.NerdGraphQueryWithContext(ctx, listTraceObserversQuery, vars, &resp); err != nil {
		return nil, err
	}

	return resp.Actor.Account.Edge.Tracing.TraceObservers, nil
}

// CreateTraceObserver creates a trace observer for an account.
func (e *Edge) CreateTraceObserver(accountID int, name string, providerRegion EdgeProviderRegion) (*EdgeTraceObserver, error) {
	return e.CreateTraceObserverWithContext(context.Background(), accountID, name, providerRegion)
}

// CreateTraceObserverWithContext creates a trace observer for an account.
func (e *Edge) CreateTraceObserverWithContext(ctx context.Context, accountID int, name string, providerRegion EdgeProviderRegion) (*EdgeTraceObserver, error) {
	resp := createTraceObserverResponse{}
	vars := map[string]interface{}{
		"accountId":            accountID,
		"traceObserverConfigs": []EdgeCreateTraceObserverInput{{true, name, providerRegion}},
	}

	if err := e.client.NerdGraphQueryWithContext(ctx, createTraceObserverMutation, vars, &resp); err != nil {
		return nil, err
	}

	errors := resp.EdgeCreateTraceObserver.Responses[0].Errors
	if len(errors) > 0 {
		return nil, fmt.Errorf("error creating trace observer: %s", errors[0].Message)
	}

	return &resp.EdgeCreateTraceObserver.Responses[0].TraceObserver, nil
}

// DeleteTraceObserver deletes a trace observer for an account.
func (e *Edge) DeleteTraceObserver(accountID int, id int) (*EdgeTraceObserver, error) {
	return e.DeleteTraceObserverWithContext(context.Background(), accountID, id)
}

// DeleteTraceObserverWithContext deletes a trace observer for an account.
func (e *Edge) DeleteTraceObserverWithContext(ctx context.Context, accountID int, id int) (*EdgeTraceObserver, error) {
	resp := deleteTraceObserversResponse{}

	vars := map[string]interface{}{
		"accountId":            accountID,
		"traceObserverConfigs": []EdgeDeleteTraceObserverInput{{id}},
	}

	if err := e.client.NerdGraphQueryWithContext(ctx, deleteTraceObserverMutation, vars, &resp); err != nil {
		return nil, err
	}

	errors := resp.EdgeDeleteTraceObservers.Responses[0].Errors
	if len(errors) > 0 {
		return nil, fmt.Errorf("error deleting trace observer: %s", errors[0].Message)
	}

	return &resp.EdgeDeleteTraceObservers.Responses[0].TraceObserver, nil
}

type traceObserverResponse struct {
	Actor struct {
		Account struct {
			Edge struct {
				Tracing EdgeTracing
			}
		}
	}
}

const (
	traceObserverSchemaFields = `
		status
		providerRegion
		name
		id
		endpoints {
			https {
				url
				port
				host
			}
			endpointType
			agent {
				port
				host
			}
			status
		}`

	traceObserverErrorSchema = `
		errors {
			type
			message
		}`

	listTraceObserversQuery = `query($accountId: Int!) { actor { account(id: $accountId) { edge { tracing {
			traceObservers { ` +
		traceObserverSchemaFields + `
			} } } } } }`

	createTraceObserverMutation = `
	mutation($traceObserverConfigs: [EdgeCreateTraceObserverInput!]!, $accountId: Int!) {
		edgeCreateTraceObserver(traceObserverConfigs: $traceObserverConfigs, accountId: $accountId) {
			responses {
				traceObserver { ` +
		traceObserverSchemaFields + `
				} ` +
		traceObserverErrorSchema + `
		} } }`

	deleteTraceObserverMutation = `
	mutation($traceObserverConfigs: [EdgeDeleteTraceObserverInput!]!, $accountId: Int!) {
		edgeDeleteTraceObservers(traceObserverConfigs: $traceObserverConfigs, accountId: $accountId) {
			responses {
				traceObserver { ` +
		traceObserverSchemaFields + `
				} ` +
		traceObserverErrorSchema + `
		} } }`
)

type createTraceObserverResponse struct {
	EdgeCreateTraceObserver struct {
		Responses []EdgeCreateTraceObserverResponse
	}
}

type deleteTraceObserversResponse struct {
	EdgeDeleteTraceObservers struct {
		Responses []EdgeDeleteTraceObserverResponse
	}
}
