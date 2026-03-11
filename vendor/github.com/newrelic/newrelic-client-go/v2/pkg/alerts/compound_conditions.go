package alerts

import (
	"context"
)

// CompoundCondition represents a New Relic compound alert condition.
// Compound conditions allow you to create alert conditions based on multiple conditions
// combined with logical expressions.
type CompoundCondition struct {
	ID                    string               `json:"id,omitempty"`
	Name                  string               `json:"name,omitempty"`
	Enabled               bool                 `json:"enabled,omitempty"`
	PolicyID              string               `json:"policyId,omitempty"`
	ComponentConditions   []ComponentCondition `json:"componentConditions,omitempty"`
	FacetMatchingBehavior string               `json:"facetMatchingBehavior,omitempty"`
	RunbookURL            string               `json:"runbookUrl,omitempty"`
	ThresholdDuration     int                  `json:"thresholdDuration,omitempty"`
	TriggerExpression     string               `json:"triggerExpression,omitempty"`
}

// ComponentCondition represents a component condition within a compound condition.
type ComponentCondition struct {
	ID    string `json:"id,omitempty"`
	Alias string `json:"alias,omitempty"`
}

// CompoundConditionCreateInput represents the input for creating a compound condition.
type CompoundConditionCreateInput struct {
	Name                  string                    `json:"name"`
	Enabled               bool                      `json:"enabled"`
	ComponentConditions   []ComponentConditionInput `json:"componentConditions,omitempty"`
	FacetMatchingBehavior *string                   `json:"facetMatchingBehavior"`
	RunbookURL            *string                   `json:"runbookUrl"`
	ThresholdDuration     *int                      `json:"thresholdDuration"`
	TriggerExpression     string                    `json:"triggerExpression"`
}

// CompoundConditionUpdateInput represents the input for updating a compound condition.
type CompoundConditionUpdateInput struct {
	Name                  *string                   `json:"name"`
	Enabled               *bool                     `json:"enabled"`
	PolicyID              *string                   `json:"policyId"`
	ComponentConditions   []ComponentConditionInput `json:"componentConditions"`
	FacetMatchingBehavior *string                   `json:"facetMatchingBehavior"`
	RunbookURL            *string                   `json:"runbookUrl"`
	ThresholdDuration     *int                      `json:"thresholdDuration"`
	TriggerExpression     *string                   `json:"triggerExpression"`
}

// ComponentConditionInput represents the input for a component condition within a compound condition.
type ComponentConditionInput struct {
	ID    string `json:"id"`
	Alias string `json:"alias"`
}

// SearchCompoundConditions searches for compound alert conditions via New Relic's NerdGraph API.
func (a *Alerts) SearchCompoundConditions(
	accountID int,
	filter *AlertsCompoundConditionFilterInput,
	sort []AlertsCompoundConditionSortInput,
	cursor *string,
) ([]*CompoundCondition, error) {
	return a.SearchCompoundConditionsWithContext(context.Background(), accountID, filter, sort, cursor)
}

// SearchCompoundConditionsWithContext searches for compound alert conditions via New Relic's NerdGraph API.
func (a *Alerts) SearchCompoundConditionsWithContext(
	ctx context.Context,
	accountID int,
	filter *AlertsCompoundConditionFilterInput,
	sort []AlertsCompoundConditionSortInput,
	cursor *string,
) ([]*CompoundCondition, error) {
	conditions := []*CompoundCondition{}
	nextCursor := cursor

	for ok := true; ok; ok = nextCursor != nil {
		resp := searchCompoundConditionsResponse{}
		vars := map[string]interface{}{
			"accountId": accountID,
			"filter":    filter,
			"sort":      sort,
			"cursor":    nextCursor,
		}

		if err := a.NerdGraphQueryWithContext(ctx, searchCompoundConditionsQuery, vars, &resp); err != nil {
			return nil, err
		}

		conditions = append(conditions, resp.Actor.Account.Alerts.CompoundConditions.Items...)
		nextCursor = resp.Actor.Account.Alerts.CompoundConditions.NextCursor
	}

	return conditions, nil
}

// AlertsCompoundConditionFilterInput represents the filter criteria for compound conditions.
type AlertsCompoundConditionFilterInput struct {
	Id *AlertsCompoundConditionIDFilter `json:"id,omitempty"`
}

// AlertsCompoundConditionIDFilter represents the ID filter operators for compound conditions.
type AlertsCompoundConditionIDFilter struct {
	Eq *string  `json:"eq,omitempty"`
	In []string `json:"in,omitempty"`
}

// AlertsCompoundConditionSortInput represents the sort criteria for compound conditions.
type AlertsCompoundConditionSortInput struct {
	Key       string `json:"key"`
	Direction string `json:"direction"`
}

// AlertsCompoundConditionSortDirection - Sort direction for compound conditions
type AlertsCompoundConditionSortDirection string

var AlertsCompoundConditionSortDirectionTypes = struct {
	ASCENDING  AlertsCompoundConditionSortDirection
	DESCENDING AlertsCompoundConditionSortDirection
}{
	ASCENDING:  "ASCENDING",
	DESCENDING: "DESCENDING",
}

// AlertsCompoundConditionSortKey - Sort key for compound conditions
type AlertsCompoundConditionSortKey string

var AlertsCompoundConditionSortKeyTypes = struct {
	ENABLED AlertsCompoundConditionSortKey
	ID      AlertsCompoundConditionSortKey
	NAME    AlertsCompoundConditionSortKey
}{
	ENABLED: "ENABLED",
	ID:      "ID",
	NAME:    "NAME",
}

// AlertsFacetMatchingBehavior - Facet matching behavior for compound conditions
type AlertsFacetMatchingBehavior string

var AlertsFacetMatchingBehaviorTypes = struct {
	FACETS_IGNORED AlertsFacetMatchingBehavior
	FACETS_MATCH   AlertsFacetMatchingBehavior
}{
	FACETS_IGNORED: "FACETS_IGNORED",
	FACETS_MATCH:   "FACETS_MATCH",
}

// CreateCompoundCondition creates a compound alert condition via New Relic's NerdGraph API.
func (a *Alerts) CreateCompoundCondition(
	accountID int,
	policyID string,
	condition CompoundConditionCreateInput,
) (*CompoundCondition, error) {
	return a.CreateCompoundConditionWithContext(context.Background(), accountID, policyID, condition)
}

// CreateCompoundConditionWithContext creates a compound alert condition via New Relic's NerdGraph API.
func (a *Alerts) CreateCompoundConditionWithContext(
	ctx context.Context,
	accountID int,
	policyID string,
	condition CompoundConditionCreateInput,
) (*CompoundCondition, error) {
	resp := compoundConditionCreateResponse{}
	vars := map[string]interface{}{
		"accountId":              accountID,
		"policyId":               policyID,
		"compoundAlertCondition": condition,
	}

	if err := a.NerdGraphQueryWithContext(ctx, createCompoundConditionMutation, vars, &resp); err != nil {
		return nil, err
	}

	return &resp.AlertsCompoundConditionCreate, nil
}

// UpdateCompoundCondition updates a compound alert condition via New Relic's NerdGraph API.
func (a *Alerts) UpdateCompoundCondition(
	accountID int,
	conditionID string,
	condition CompoundConditionUpdateInput,
) (*CompoundCondition, error) {
	return a.UpdateCompoundConditionWithContext(context.Background(), accountID, conditionID, condition)
}

// UpdateCompoundConditionWithContext updates a compound alert condition via New Relic's NerdGraph API.
func (a *Alerts) UpdateCompoundConditionWithContext(
	ctx context.Context,
	accountID int,
	conditionID string,
	condition CompoundConditionUpdateInput,
) (*CompoundCondition, error) {
	resp := compoundConditionUpdateResponse{}
	vars := map[string]interface{}{
		"accountId":              accountID,
		"id":                     conditionID,
		"compoundAlertCondition": condition,
	}

	if err := a.NerdGraphQueryWithContext(ctx, updateCompoundConditionMutation, vars, &resp); err != nil {
		return nil, err
	}

	return &resp.AlertsCompoundConditionUpdate, nil
}

// DeleteCompoundCondition deletes a compound alert condition via New Relic's NerdGraph API.
func (a *Alerts) DeleteCompoundCondition(
	accountID int,
	conditionID string,
) (string, error) {
	return a.DeleteCompoundConditionWithContext(context.Background(), accountID, conditionID)
}

// DeleteCompoundConditionWithContext deletes a compound alert condition via New Relic's NerdGraph API.
func (a *Alerts) DeleteCompoundConditionWithContext(
	ctx context.Context,
	accountID int,
	conditionID string,
) (string, error) {
	resp := compoundConditionDeleteResponse{}
	vars := map[string]interface{}{
		"accountId": accountID,
		"id":        conditionID,
	}

	if err := a.NerdGraphQueryWithContext(ctx, deleteCompoundConditionMutation, vars, &resp); err != nil {
		return "", err
	}

	return resp.AlertsCompoundConditionDelete.ID, nil
}

// Response types for GraphQL queries/mutations
type searchCompoundConditionsResponse struct {
	Actor struct {
		Account struct {
			Alerts struct {
				CompoundConditions struct {
					NextCursor *string              `json:"nextCursor"`
					TotalCount int                  `json:"totalCount"`
					Items      []*CompoundCondition `json:"items"`
				} `json:"compoundConditions"`
			} `json:"alerts"`
		} `json:"account"`
	} `json:"actor"`
}

type compoundConditionCreateResponse struct {
	AlertsCompoundConditionCreate CompoundCondition `json:"alertsCompoundConditionCreate"`
}

type compoundConditionUpdateResponse struct {
	AlertsCompoundConditionUpdate CompoundCondition `json:"alertsCompoundConditionUpdate"`
}

type compoundConditionDeleteResponse struct {
	AlertsCompoundConditionDelete struct {
		ID string `json:"id"`
	} `json:"alertsCompoundConditionDelete"`
}

// GraphQL query and mutation definitions
const (
	graphqlCompoundConditionStructFields = `
		id
		componentConditions {
			id
			alias
		}
		enabled
		facetMatchingBehavior
		name
		policyId
		runbookUrl
		thresholdDuration
		triggerExpression
	`

	searchCompoundConditionsQuery = `
		query($accountId: Int!, $filter: AlertsCompoundConditionFilterInput, $sort: [AlertsCompoundConditionSortInput!], $cursor: String) {
			actor {
				account(id: $accountId) {
					alerts {
						compoundConditions(
							filter: $filter
							sort: $sort
							cursor: $cursor
						) {
							totalCount
							nextCursor
							items {` +
		graphqlCompoundConditionStructFields +
		`}
						}
					}
				}
			}
		}`

	createCompoundConditionMutation = `
		mutation($accountId: Int!, $policyId: ID!, $compoundAlertCondition: AlertsCompoundConditionInput!) {
			alertsCompoundConditionCreate(
				accountId: $accountId,
				policyId: $policyId,
				compoundAlertCondition: $compoundAlertCondition
			) {` +
		graphqlCompoundConditionStructFields +
		`}
		}
	`

	updateCompoundConditionMutation = `
		mutation($id: ID!, $accountId: Int!, $compoundAlertCondition: AlertsCompoundConditionUpdateInput!) {
			alertsCompoundConditionUpdate(
				id: $id,
				accountId: $accountId,
				compoundAlertCondition: $compoundAlertCondition
			) {` +
		graphqlCompoundConditionStructFields +
		`}
		}
	`

	deleteCompoundConditionMutation = `
		mutation($accountId: Int!, $id: ID!) {
			alertsCompoundConditionDelete(
				id: $id,
				accountId: $accountId
			) {
				id
			}
		}
	`
)
