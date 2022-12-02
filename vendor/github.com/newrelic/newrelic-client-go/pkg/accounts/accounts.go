// Package accounts provides a programmatic API for interacting with New Relic accounts.
package accounts

import (
	"context"

	"github.com/newrelic/newrelic-client-go/internal/http"
	"github.com/newrelic/newrelic-client-go/pkg/config"
	"github.com/newrelic/newrelic-client-go/pkg/logging"
)

// Accounts is used to interact with New Relic accounts.
type Accounts struct {
	client http.Client
	logger logging.Logger
}

// New returns a new client for interacting with New Relic accounts.
func New(config config.Config) Accounts {
	return Accounts{
		client: http.NewClient(config),
		logger: config.GetLogger(),
	}
}

// ListAccountsParams represents the input parameters for the ListAcounts method.
type ListAccountsParams struct {
	Scope *RegionScope
}

// ListAccounts lists the accounts this user is authorized to view.
func (e *Accounts) ListAccounts(params ListAccountsParams) ([]AccountOutline, error) {
	return e.ListAccountsWithContext(context.Background(), params)
}

// ListAccountsWithContext lists the accounts this user is authorized to view.
func (e *Accounts) ListAccountsWithContext(ctx context.Context, params ListAccountsParams) ([]AccountOutline, error) {
	resp := accountsResponse{}
	vars := map[string]interface{}{
		"accountId": params.Scope,
	}

	if err := e.client.NerdGraphQueryWithContext(ctx, listAccountsQuery, vars, &resp); err != nil {
		return nil, err
	}

	return resp.Actor.Accounts, nil
}

type accountsResponse struct {
	Actor struct {
		Accounts []AccountOutline
	}
}

const (
	accountsSchemaFields = `
		name
		id
	`

	listAccountsQuery = `query($scope: RegionScope) { actor { accounts(scope: $scope) {
		` + accountsSchemaFields +
		` } } }`
)
