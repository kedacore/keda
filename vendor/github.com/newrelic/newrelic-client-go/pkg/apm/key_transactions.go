package apm

import (
	"context"
	"fmt"
)

// KeyTransaction represents information about a New Relic key transaction.
type KeyTransaction struct {
	ID              int                       `json:"id,omitempty"`
	Name            string                    `json:"name,omitempty"`
	TransactionName string                    `json:"transaction_name,omitempty"`
	HealthStatus    string                    `json:"health_status,omitempty"`
	LastReportedAt  string                    `json:"last_reported_at,omitempty"`
	Reporting       bool                      `json:"reporting"`
	Summary         ApplicationSummary        `json:"application_summary,omitempty"`
	EndUserSummary  ApplicationEndUserSummary `json:"end_user_summary,omitempty"`
	Links           KeyTransactionLinks       `json:"links,omitempty"`
}

// KeyTransactionLinks represents associations for a key transaction.
type KeyTransactionLinks struct {
	Application int `json:"application,omitempty"`
}

// ListKeyTransactionsParams represents a set of filters to be
// used when querying New Relic key transactions.
type ListKeyTransactionsParams struct {
	Name string `url:"filter[name],omitempty"`
	IDs  []int  `url:"filter[ids],omitempty,comma"`
}

// ListKeyTransactions returns all key transactions for an account.
func (a *APM) ListKeyTransactions(params *ListKeyTransactionsParams) ([]*KeyTransaction, error) {
	return a.ListKeyTransactionsWithContext(context.Background(), params)
}

// ListKeyTransactionsWithContext returns all key transactions for an account.
func (a *APM) ListKeyTransactionsWithContext(ctx context.Context, params *ListKeyTransactionsParams) ([]*KeyTransaction, error) {
	results := []*KeyTransaction{}
	nextURL := a.config.Region().RestURL("key_transactions.json")

	for nextURL != "" {
		response := keyTransactionsResponse{}
		resp, err := a.client.GetWithContext(ctx, nextURL, &params, &response)

		if err != nil {
			return nil, err
		}

		results = append(results, response.KeyTransactions...)

		paging := a.pager.Parse(resp)
		nextURL = paging.Next
	}

	return results, nil
}

// GetKeyTransaction returns a specific key transaction by ID.
func (a *APM) GetKeyTransaction(id int) (*KeyTransaction, error) {
	return a.GetKeyTransactionWithContext(context.Background(), id)
}

// GetKeyTransactionWithContext returns a specific key transaction by ID.
func (a *APM) GetKeyTransactionWithContext(ctx context.Context, id int) (*KeyTransaction, error) {
	response := keyTransactionResponse{}
	url := fmt.Sprintf("/key_transactions/%d.json", id)

	_, err := a.client.GetWithContext(ctx, a.config.Region().RestURL(url), nil, &response)

	if err != nil {
		return nil, err
	}

	return &response.KeyTransaction, nil
}

type keyTransactionsResponse struct {
	KeyTransactions []*KeyTransaction `json:"key_transactions,omitempty"`
}

type keyTransactionResponse struct {
	KeyTransaction KeyTransaction `json:"key_transaction,omitempty"`
}
