// Package nrdb provides a programmatic API for interacting with NRDB, New Relic's Datastore.
// This package is NOT covered by Tutone.

package nrdb

import "context"

// WARNING! The following function, 'Query' is used by newrelic-cli to run pre-install
// validation procedures before the actual installation begins; and is hence, extremely fragile.
// Please do not resort to changing this function unless necessary, such as in the case
// of a deprecation or an end-of-life. Kindly duplicate this function to allow more
// attributes, or carefully modify functions following this function, below.

// Query facilitates making an NRQL query using NerdGraph.
func (n *Nrdb) Query(accountID int, query NRQL) (*NRDBResultContainer, error) {
	return n.QueryWithContext(context.Background(), accountID, query)
}

// WARNING! This function is extremely fragile.
// Please read the note above the function 'Query': refrain from making changes unless extremely necessary.

// QueryWithContext facilitates making a NRQL query.
func (n *Nrdb) QueryWithContext(ctx context.Context, accountID int, query NRQL) (*NRDBResultContainer, error) {
	respBody := gqlNRQLQueryResponse{}

	vars := map[string]interface{}{
		"accountId": accountID,
		"query":     query,
	}

	if err := n.client.NerdGraphQueryWithContext(ctx, gqlNrqlQuery, vars, &respBody); err != nil {
		return nil, err
	}

	return &respBody.Actor.Account.NRQL, nil
}

// WARNING! This NerdGraph query is extremely fragile.
// Please read the note above the function 'Query': refrain from making changes unless extremely necessary.
const gqlNrqlQuery = `query (
	$query: Nrql!, 
	$accountId: Int!
) 
{
  actor {
    account(id: $accountId) {
      nrql(query: $query) {
        currentResults
        otherResult
        previousResults
        results
        totalResult
        metadata {
          eventTypes
          facets
          messages
          timeWindow {
            begin
            compareWith
            end
            since
            until
          }
        }
      }
    }
  }
}
`

func (n *Nrdb) QueryWithExtendedResponse(accountID int, query NRQL) (*NRDBResultContainer, error) {
	return n.QueryWithExtendedResponseWithContext(context.Background(), accountID, query)
}

// QueryWithExtendedResponseWithContext facilitates making a NRQL query with additional options.
func (n *Nrdb) QueryWithExtendedResponseWithContext(ctx context.Context, accountID int, query NRQL) (*NRDBResultContainer, error) {
	respBody := gqlNRQLQueryResponse{}

	vars := map[string]interface{}{
		"accountId": accountID,
		"query":     query,
	}

	if err := n.client.NerdGraphQueryWithContext(ctx, gqlNRQLQueryWithExtendedResponse, vars, &respBody); err != nil {
		return nil, err
	}

	return &respBody.Actor.Account.NRQL, nil
}

func (n *Nrdb) QueryWithAdditionalOptions(
	accountID int,
	query NRQL,
	timeout Seconds,
	async bool,
) (*NRDBResultContainer, error) {
	return n.QueryWithAdditionalOptionsWithContext(
		context.Background(),
		accountID,
		query,
		timeout,
		async,
	)
}

// QueryWithAdditionalOptionsWithContext facilitates making a NRQL query with the specification of a timeout between 5 and 120 seconds.
func (n *Nrdb) QueryWithAdditionalOptionsWithContext(
	ctx context.Context,
	accountID int,
	query NRQL,
	timeout Seconds,
	async bool,
) (*NRDBResultContainer, error) {
	respBody := gqlNRQLQueryResponse{}

	vars := map[string]interface{}{
		"accountId": accountID,
		"query":     query,
		"timeout":   timeout,
		"async":     async,
	}

	if err := n.client.NerdGraphQueryWithContext(ctx, gqlNRQLQueryWithAdditionalOptions, vars, &respBody); err != nil {
		return nil, err
	}

	return &respBody.Actor.Account.NRQL, nil
}

func (n *Nrdb) QueryHistory() (*[]NRQLHistoricalQuery, error) {
	return n.QueryHistoryWithContext(context.Background())
}

func (n *Nrdb) QueryHistoryWithContext(ctx context.Context) (*[]NRQLHistoricalQuery, error) {
	respBody := gqlNRQLQueryHistoryResponse{}
	vars := map[string]interface{}{}

	if err := n.client.NerdGraphQueryWithContext(ctx, gqlNRQLQueryHistoryQuery, vars, &respBody); err != nil {
		return nil, err
	}

	return &respBody.Actor.QueryHistory.Nrql, nil
}

const gqlNRQLQueryHistoryQuery = `
	{
	  actor {
		queryHistory {
		  nrql {
			accountIds
			query
			createdAt
		  }
		}
	  }
}`

const gqlNRQLQueryWithExtendedResponse = `query(
	$query: Nrql!, 
	$accountId: Int!
) 
{
  actor {
    account(id: $accountId) {
      nrql(query: $query) {
        currentResults
        nrql
        otherResult
        previousResults
        queryProgress {
          completed
          queryId
          retryAfter
          resultExpiration
          retryDeadline
        }
        rawResponse
        results
        totalResult
        metadata {
          eventTypes
          facets
          messages
          timeWindow {
            begin
            compareWith
            end
            since
            until
          }
        }
        suggestedFacets {
          attributes
          nrql
        }
        eventDefinitions {
          definition
          label
          name
          attributes {
            definition
            documentationUrl
            label
            name
          }
        }
      }
    }
  }
}
`

const gqlNRQLQueryWithAdditionalOptions = `query (
	$query: Nrql!, 
	$accountId: Int!, 
	$timeout: Seconds, 
	$async: Boolean
) 
{
  actor {
    account(id: $accountId) {
      nrql(query: $query, timeout: $timeout, async: $async) {
        currentResults
        nrql
        otherResult
        previousResults
        queryProgress {
          completed
          queryId
          retryAfter
          resultExpiration
          retryDeadline
        }
        rawResponse
        results
        totalResult
        metadata {
          eventTypes
          facets
          messages
          timeWindow {
            begin
            compareWith
            end
            since
            until
          }
        }
        suggestedFacets {
          attributes
          nrql
        }
        eventDefinitions {
          definition
          label
          name
          attributes {
            definition
            documentationUrl
            label
            name
          }
        }
      }
    }
  }
}
`

type gqlNRQLQueryResponse struct {
	Actor struct {
		Account struct {
			NRQL NRDBResultContainer
		}
	}
}

type gqlNRQLQueryHistoryResponse struct {
	Actor struct {
		QueryHistory struct {
			Nrql []NRQLHistoricalQuery
		}
	}
}
