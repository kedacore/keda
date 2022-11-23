package alerts

import (
	"context"
	"fmt"

	"github.com/newrelic/newrelic-client-go/pkg/errors"

	"github.com/newrelic/newrelic-client-go/internal/http"
	"github.com/newrelic/newrelic-client-go/internal/serialization"
)

// IncidentPreferenceType specifies rollup settings for alert policies.
type IncidentPreferenceType string

var (
	// IncidentPreferenceTypes specifies the possible incident preferenece types for an alert policy.
	IncidentPreferenceTypes = struct {
		PerPolicy             IncidentPreferenceType
		PerCondition          IncidentPreferenceType
		PerConditionAndTarget IncidentPreferenceType
	}{
		PerPolicy:             "PER_POLICY",
		PerCondition:          "PER_CONDITION",
		PerConditionAndTarget: "PER_CONDITION_AND_TARGET",
	}
)

// Policy represents a New Relic alert policy.
type Policy struct {
	ID                 int                      `json:"id,omitempty"`
	IncidentPreference IncidentPreferenceType   `json:"incident_preference,omitempty"`
	Name               string                   `json:"name,omitempty"`
	CreatedAt          *serialization.EpochTime `json:"created_at,omitempty"`
	UpdatedAt          *serialization.EpochTime `json:"updated_at,omitempty"`
}

// ListPoliciesParams represents a set of filters to be used when querying New
// Relic alert policies.
type ListPoliciesParams struct {
	Name string `url:"filter[name],omitempty"`
}

// ListPolicies returns a list of Alert Policies for a given account.
func (a *Alerts) ListPolicies(params *ListPoliciesParams) ([]Policy, error) {
	return a.ListPoliciesWithContext(context.Background(), params)
}

// ListPoliciesWithContext returns a list of Alert Policies for a given account.
func (a *Alerts) ListPoliciesWithContext(ctx context.Context, params *ListPoliciesParams) ([]Policy, error) {
	alertPolicies := []Policy{}

	nextURL := a.config.Region().RestURL("/alerts_policies.json")

	for nextURL != "" {
		response := alertPoliciesResponse{}
		resp, err := a.client.GetWithContext(ctx, nextURL, &params, &response)

		if err != nil {
			return nil, err
		}

		alertPolicies = append(alertPolicies, response.Policies...)

		paging := a.pager.Parse(resp)
		nextURL = paging.Next
	}

	return alertPolicies, nil
}

// GetPolicy returns a specific alert policy by ID for a given account.
func (a *Alerts) GetPolicy(id int) (*Policy, error) {
	return a.GetPolicyWithContext(context.Background(), id)
}

// GetPolicyWithContext returns a specific alert policy by ID for a given account.
func (a *Alerts) GetPolicyWithContext(ctx context.Context, id int) (*Policy, error) {
	policies, err := a.ListPoliciesWithContext(ctx, nil)
	if err != nil {
		return nil, err
	}

	for _, policy := range policies {
		if policy.ID == id {
			return &policy, nil
		}
	}

	return nil, errors.NewNotFoundf("no alert policy found for id %d", id)
}

// CreatePolicy creates a new alert policy for a given account.
func (a *Alerts) CreatePolicy(policy Policy) (*Policy, error) {
	return a.CreatePolicyWithContext(context.Background(), policy)
}

// CreatePolicyWithContext creates a new alert policy for a given account.
func (a *Alerts) CreatePolicyWithContext(ctx context.Context, policy Policy) (*Policy, error) {
	reqBody := alertPolicyRequestBody{
		Policy: policy,
	}
	resp := alertPolicyResponse{}

	_, err := a.client.PostWithContext(ctx, a.config.Region().RestURL("/alerts_policies.json"), nil, &reqBody, &resp)

	if err != nil {
		return nil, err
	}

	return &resp.Policy, nil
}

// UpdatePolicy update an alert policy for a given account.
func (a *Alerts) UpdatePolicy(policy Policy) (*Policy, error) {
	return a.UpdatePolicyWithContext(context.Background(), policy)
}

// UpdatePolicyWithContext update an alert policy for a given account.
func (a *Alerts) UpdatePolicyWithContext(ctx context.Context, policy Policy) (*Policy, error) {
	reqBody := alertPolicyRequestBody{
		Policy: policy,
	}
	resp := alertPolicyResponse{}
	url := fmt.Sprintf("/alerts_policies/%d.json", policy.ID)

	_, err := a.client.PutWithContext(ctx, a.config.Region().RestURL(url), nil, &reqBody, &resp)
	if err != nil {
		return nil, err
	}

	return &resp.Policy, nil
}

// DeletePolicy deletes an existing alert policy for a given account.
func (a *Alerts) DeletePolicy(id int) (*Policy, error) {
	return a.DeletePolicyWithContext(context.Background(), id)
}

// DeletePolicyWithContext deletes an existing alert policy for a given account.
func (a *Alerts) DeletePolicyWithContext(ctx context.Context, id int) (*Policy, error) {
	resp := alertPolicyResponse{}
	url := fmt.Sprintf("/alerts_policies/%d.json", id)

	_, err := a.client.DeleteWithContext(ctx, a.config.Region().RestURL(url), nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp.Policy, nil
}

func (a *Alerts) CreatePolicyMutation(accountID int, policy AlertsPolicyInput) (*AlertsPolicy, error) {
	return a.CreatePolicyMutationWithContext(context.Background(), accountID, policy)
}

func (a *Alerts) CreatePolicyMutationWithContext(ctx context.Context, accountID int, policy AlertsPolicyInput) (*AlertsPolicy, error) {
	vars := map[string]interface{}{
		"accountID": accountID,
		"policy":    policy,
	}

	resp := alertQueryPolicyCreateResponse{}

	if err := a.client.NerdGraphQueryWithContext(ctx, alertsPolicyCreatePolicy, vars, &resp); err != nil {
		return nil, err
	}

	return &resp.AlertsPolicy, nil
}

func (a *Alerts) UpdatePolicyMutation(accountID int, policyID string, policy AlertsPolicyUpdateInput) (*AlertsPolicy, error) {
	return a.UpdatePolicyMutationWithContext(context.Background(), accountID, policyID, policy)
}

func (a *Alerts) UpdatePolicyMutationWithContext(ctx context.Context, accountID int, policyID string, policy AlertsPolicyUpdateInput) (*AlertsPolicy, error) {
	vars := map[string]interface{}{
		"accountID": accountID,
		"policyID":  policyID,
		"policy":    policy,
	}

	resp := alertQueryPolicyUpdateResponse{}

	if err := a.client.NerdGraphQueryWithContext(ctx, alertsPolicyUpdatePolicy, vars, &resp); err != nil {
		return nil, err
	}

	return &resp.AlertsPolicy, nil
}

// QueryPolicy queries NerdGraph for a policy matching the given account ID and
// policy ID.
func (a *Alerts) QueryPolicy(accountID int, id string) (*AlertsPolicy, error) {
	return a.QueryPolicyWithContext(context.Background(), accountID, id)
}

// QueryPolicyWithContext queries NerdGraph for a policy matching the given account ID and
// policy ID.
func (a *Alerts) QueryPolicyWithContext(ctx context.Context, accountID int, id string) (*AlertsPolicy, error) {
	resp := alertQueryPolicyResponse{}
	vars := map[string]interface{}{
		"accountID": accountID,
		"policyID":  id,
	}

	req, err := a.client.NewNerdGraphRequest(alertPolicyQueryPolicy, vars, &resp)
	if err != nil {
		return nil, err
	}

	req.WithContext(ctx)

	var errorResponse alertPoliciesErrorResponse
	req.SetErrorValue(&errorResponse)

	if _, err := a.client.Do(req); err != nil {
		return nil, err
	}

	return &resp.Actor.Account.Alerts.Policy, nil
}

// QueryPolicySearch searches NerdGraph for policies.
func (a *Alerts) QueryPolicySearch(accountID int, params AlertsPoliciesSearchCriteriaInput) ([]*AlertsPolicy, error) {
	return a.QueryPolicySearchWithContext(context.Background(), accountID, params)
}

// QueryPolicySearchWithContext searches NerdGraph for policies.
func (a *Alerts) QueryPolicySearchWithContext(ctx context.Context, accountID int, params AlertsPoliciesSearchCriteriaInput) ([]*AlertsPolicy, error) {
	policies := []*AlertsPolicy{}
	var nextCursor *string

	for ok := true; ok; ok = nextCursor != nil {
		resp := alertQueryPolicySearchResponse{}
		vars := map[string]interface{}{
			"accountID":      accountID,
			"cursor":         nextCursor,
			"searchCriteria": params,
		}

		if err := a.client.NerdGraphQueryWithContext(ctx, alertsPolicyQuerySearch, vars, &resp); err != nil {
			return nil, err
		}

		policies = append(policies, resp.Actor.Account.Alerts.PoliciesSearch.Policies...)

		nextCursor = resp.Actor.Account.Alerts.PoliciesSearch.NextCursor
	}

	return policies, nil
}

// DeletePolicyMutation is the NerdGraph mutation to delete a policy given the
// account ID and the policy ID.
func (a *Alerts) DeletePolicyMutation(accountID int, id string) (*AlertsPolicy, error) {
	return a.DeletePolicyMutationWithContext(context.Background(), accountID, id)
}

// DeletePolicyMutationWithContext is the NerdGraph mutation to delete a policy given the
// account ID and the policy ID.
func (a *Alerts) DeletePolicyMutationWithContext(ctx context.Context, accountID int, id string) (*AlertsPolicy, error) {
	policy := &AlertsPolicy{}

	resp := alertQueryPolicyDeleteRespose{}
	vars := map[string]interface{}{
		"accountID": accountID,
		"policyID":  id,
	}

	if err := a.client.NerdGraphQueryWithContext(ctx, alertPolicyDeletePolicy, vars, &resp); err != nil {
		return nil, err
	}

	return policy, nil
}

type alertPoliciesErrorResponse struct {
	http.GraphQLErrorResponse
}

func (r *alertPoliciesErrorResponse) IsNotFound() bool {
	if len(r.Errors) == 0 {
		return false
	}

	for _, err := range r.Errors {
		if err.Message == "Not Found" &&
			// TODO: When the alerts API begins using `errorClass`
			// instead of `code` to specify error type, the conditional
			// checking the `code` field can be removed.
			//
			// https://newrelic.atlassian.net/browse/AINTER-7746
			(err.Extensions.Code == "BAD_USER_INPUT" || err.Extensions.ErrorClass == "BAD_USER_INPUT") {
			return true
		}
	}

	return false
}

func (r *alertPoliciesErrorResponse) Error() string {
	return r.GraphQLErrorResponse.Error()
}

func (r *alertPoliciesErrorResponse) New() http.ErrorResponse {
	return &alertPoliciesErrorResponse{}
}

type alertPoliciesResponse struct {
	Policies []Policy `json:"policies,omitempty"`
}

type alertPolicyResponse struct {
	Policy Policy `json:"policy,omitempty"`
}

type alertPolicyRequestBody struct {
	Policy Policy `json:"policy"`
}

type alertQueryPolicySearchResponse struct {
	Actor struct {
		Account struct {
			Alerts struct {
				PoliciesSearch struct {
					NextCursor *string         `json:"nextCursor"`
					Policies   []*AlertsPolicy `json:"policies"`
					TotalCount int             `json:"totalCount"`
				} `json:"policiesSearch"`
			} `json:"alerts"`
		} `json:"account"`
	} `json:"actor"`
}

type alertQueryPolicyCreateResponse struct {
	AlertsPolicy AlertsPolicy `json:"alertsPolicyCreate"`
}

type alertQueryPolicyUpdateResponse struct {
	AlertsPolicy AlertsPolicy `json:"alertsPolicyUpdate"`
}

type alertQueryPolicyResponse struct {
	Actor struct {
		Account struct {
			Alerts struct {
				Policy AlertsPolicy `json:"policy"`
			} `json:"alerts"`
		} `json:"account"`
	} `json:"actor"`
}

type alertQueryPolicyDeleteRespose struct {
	AlertsPolicyDelete struct {
		ID int `json:"id,string"`
	} `json:"alertsPolicyDelete"`
}

const (
	graphqlAlertPolicyFields = `
						id
						name
						incidentPreference
						accountId
	`
	alertPolicyQueryPolicy = `query($accountID: Int!, $policyID: ID!) {
		actor {
			account(id: $accountID) {
				alerts {
					policy(id: $policyID) {` + graphqlAlertPolicyFields + `
					}
				}
			}
		}
	}`

	alertsPolicyQuerySearch = `query($accountID: Int!, $cursor: String, $criteria: AlertsPoliciesSearchCriteriaInput) {
		actor {
			account(id: $accountID) {
				alerts {
					policiesSearch(cursor: $cursor, searchCriteria: $criteria) {
						nextCursor
						totalCount
						policies {
							accountId
							id
							incidentPreference
							name
						}
					}
				}
			}
		}
	}`

	alertsPolicyCreatePolicy = `mutation CreatePolicy($accountID: Int!, $policy: AlertsPolicyInput!){
		alertsPolicyCreate(accountId: $accountID, policy: $policy) {` + graphqlAlertPolicyFields + `
		} }`

	alertsPolicyUpdatePolicy = `mutation UpdatePolicy($accountID: Int!, $policyID: ID!, $policy: AlertsPolicyUpdateInput!){
			alertsPolicyUpdate(accountId: $accountID, id: $policyID, policy: $policy) {` + graphqlAlertPolicyFields + `
			}
		}`

	alertPolicyDeletePolicy = `mutation DeletePolicy($accountID: Int!, $policyID: ID!){
		alertsPolicyDelete(accountId: $accountID, id: $policyID) {
			id
		} }`
)
