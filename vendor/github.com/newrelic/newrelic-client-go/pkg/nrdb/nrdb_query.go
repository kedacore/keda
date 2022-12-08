// Package nrdb provides a programmatic API for interacting with NRDB, New Relic's Datastore
package nrdb

import "context"

func (n *Nrdb) Query(accountID int, query NRQL) (*NRDBResultContainer, error) {
	return n.QueryWithContext(context.Background(), accountID, query)
}

// QueryWithContext facilitates making a NRQL query.
func (n *Nrdb) QueryWithContext(ctx context.Context, accountID int, query NRQL) (*NRDBResultContainer, error) {
	respBody := gqlNrglQueryResponse{}

	vars := map[string]interface{}{
		"accountId": accountID,
		"query":     query,
	}

	if err := n.client.NerdGraphQueryWithContext(ctx, gqlNrqlQuery, vars, &respBody); err != nil {
		return nil, err
	}

	return &respBody.Actor.Account.NRQL, nil
}

func (n *Nrdb) QueryHistory() (*[]NRQLHistoricalQuery, error) {
	return n.QueryHistoryWithContext(context.Background())
}

func (n *Nrdb) QueryHistoryWithContext(ctx context.Context) (*[]NRQLHistoricalQuery, error) {
	respBody := gqlNrglQueryHistoryResponse{}
	vars := map[string]interface{}{}

	if err := n.client.NerdGraphQueryWithContext(ctx, gqlNrqlQueryHistoryQuery, vars, &respBody); err != nil {
		return nil, err
	}

	return &respBody.Actor.NRQLQueryHistory, nil
}

const (
	gqlNrqlQueryHistoryQuery = `{ actor { nrqlQueryHistory { accountId nrql timestamp } } }`

	gqlNrqlQuery = `query($query: Nrql!, $accountId: Int!) { actor { account(id: $accountId) { nrql(query: $query) {
    currentResults otherResult previousResults results totalResult
    metadata { eventTypes facets messages timeWindow { begin compareWith end since until } }
  } } } }`
)

type gqlNrglQueryResponse struct {
	Actor struct {
		Account struct {
			NRQL NRDBResultContainer
		}
	}
}

type gqlNrglQueryHistoryResponse struct {
	Actor struct {
		NRQLQueryHistory []NRQLHistoricalQuery
	}
}
