package eventstometrics

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/newrelic/newrelic-client-go/v2/internal/http"
	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
	nrErrors "github.com/newrelic/newrelic-client-go/v2/pkg/errors"
	"github.com/newrelic/newrelic-client-go/v2/pkg/logging"
)

// EventsToMetrics is used to communicate with New Relic EventsToMetrics.
type EventsToMetrics struct {
	client http.Client
	config config.Config
	logger logging.Logger
}

// New is used to create a new EventsToMetrics client instance.
func New(config config.Config) EventsToMetrics {
	client := http.NewClient(config)
	client.SetAuthStrategy(&http.PersonalAPIKeyCapableV2Authorizer{})

	pkg := EventsToMetrics{
		client: client,
		config: config,
		logger: config.GetLogger(),
	}

	return pkg
}

// ListRules retrieves a set of New Relic events to metrics rules by their account ID.
func (e *EventsToMetrics) ListRules(accountID int) ([]EventsToMetricsRule, error) {
	return e.ListRulesWithContext(context.Background(), accountID)
}

// ListRulesWithContext retrieves a set of New Relic events to metrics rules by their account ID.
func (e *EventsToMetrics) ListRulesWithContext(ctx context.Context, accountID int) ([]EventsToMetricsRule, error) {
	resp := listRulesResponse{}
	vars := map[string]interface{}{
		"accountId": accountID,
	}

	if err := e.client.NerdGraphQueryWithContext(ctx, listEventsToMetricsRulesQuery, vars, &resp); err != nil {
		return nil, err
	}

	if len(resp.Actor.Account.EventsToMetrics.AllRules.Rules) == 0 {
		return nil, nrErrors.NewNotFound("")
	}

	return resp.Actor.Account.EventsToMetrics.AllRules.Rules, nil
}

// GetRule retrieves one or more New Relic events to metrics rules by their IDs.
func (e *EventsToMetrics) GetRule(accountID int, ruleID string) (*EventsToMetricsRule, error) {
	return e.GetRuleWithContext(context.Background(), accountID, ruleID)
}

// GetRuleWithContext retrieves one or more New Relic events to metrics rules by their IDs.
func (e *EventsToMetrics) GetRuleWithContext(ctx context.Context, accountID int, ruleID string) (*EventsToMetricsRule, error) {
	rules, err := e.GetRules(accountID, []string{ruleID})
	if err != nil {
		return nil, err
	}

	return &rules[0], nil
}

// GetRules retrieves one or more New Relic events to metrics rules by their IDs.
func (e *EventsToMetrics) GetRules(accountID int, ruleIDs []string) ([]EventsToMetricsRule, error) {
	return e.GetRulesWithContext(context.Background(), accountID, ruleIDs)
}

// GetRulesWithContext retrieves one or more New Relic events to metrics rules by their IDs.
func (e *EventsToMetrics) GetRulesWithContext(ctx context.Context, accountID int, ruleIDs []string) ([]EventsToMetricsRule, error) {
	resp := getRulesResponse{}
	vars := map[string]interface{}{
		"accountId": accountID,
		"ruleIds":   ruleIDs,
	}

	if err := e.client.NerdGraphQueryWithContext(ctx, getEventsToMetricsRulesQuery, vars, &resp); err != nil {
		return nil, err
	}

	if len(resp.Actor.Account.EventsToMetrics.RulesByID.Rules) == 0 {
		return nil, nrErrors.NewNotFound("")
	}

	return resp.Actor.Account.EventsToMetrics.RulesByID.Rules, nil
}

// CreateRules creates one or more New Relic events to metrics rules.
func (e *EventsToMetrics) CreateRules(createInput []EventsToMetricsCreateRuleInput) ([]EventsToMetricsRule, error) {
	return e.CreateRulesWithContext(context.Background(), createInput)
}

// CreateRulesWithContext creates one or more New Relic events to metrics rules.
func (e *EventsToMetrics) CreateRulesWithContext(ctx context.Context, createInput []EventsToMetricsCreateRuleInput) ([]EventsToMetricsRule, error) {
	resp := createRuleResponse{}
	vars := map[string]interface{}{
		"createInput": createInput,
	}

	if err := e.client.NerdGraphQueryWithContext(ctx, eventsToMetricsCreateRuleMutation, vars, &resp); err != nil {
		return nil, err
	}

	if len(resp.EventsToMetricsCreateRule.Failures) > 0 {
		return nil, errors.New(resp.EventsToMetricsCreateRule.Errors())
	}

	return resp.EventsToMetricsCreateRule.Successes, nil
}

// DeleteRules deletes one or more New Relic events to metrics rules.
func (e *EventsToMetrics) DeleteRules(deleteInput []EventsToMetricsDeleteRuleInput) ([]EventsToMetricsRule, error) {
	return e.DeleteRulesWithContext(context.Background(), deleteInput)
}

// DeleteRulesWithContext deletes one or more New Relic events to metrics rules.
func (e *EventsToMetrics) DeleteRulesWithContext(ctx context.Context, deleteInput []EventsToMetricsDeleteRuleInput) ([]EventsToMetricsRule, error) {
	resp := deleteRuleResponse{}
	vars := map[string]interface{}{
		"deleteInput": deleteInput,
	}

	if err := e.client.NerdGraphQueryWithContext(ctx, eventsToMetricsDeleteRuleMutation, vars, &resp); err != nil {
		return nil, err
	}

	if len(resp.EventsToMetricsDeleteRule.Failures) > 0 {
		return nil, errors.New(resp.EventsToMetricsDeleteRule.Errors())
	}

	return resp.EventsToMetricsDeleteRule.Successes, nil
}

// UpdateRules updates one or more New Relic events to metrics rules.
func (e *EventsToMetrics) UpdateRules(updateInput []EventsToMetricsUpdateRuleInput) ([]EventsToMetricsRule, error) {
	return e.UpdateRulesWithContext(context.Background(), updateInput)
}

// UpdateRulesWithContext updates one or more New Relic events to metrics rules.
func (e *EventsToMetrics) UpdateRulesWithContext(ctx context.Context, updateInput []EventsToMetricsUpdateRuleInput) ([]EventsToMetricsRule, error) {
	resp := updateRuleResponse{}
	vars := map[string]interface{}{
		"updateInput": updateInput,
	}

	if err := e.client.NerdGraphQueryWithContext(ctx, eventsToMetricsUpdateRuleMutation, vars, &resp); err != nil {
		return nil, err
	}

	if len(resp.EventsToMetricsUpdateRule.Failures) > 0 {
		return nil, errors.New(resp.EventsToMetricsUpdateRule.Errors())
	}

	return resp.EventsToMetricsUpdateRule.Successes, nil
}

const (
	// graphqlEventsToMetricsRuleStructFields is the set of fields that we want returned on queries,
	// and should map back directly to the EventsToMetricsRule struct
	graphqlEventsToMetricsRuleStructFields = `
		id
		name
		enabled
		createdAt
		accountId
		description
		nrql
		updatedAt
`

	graphqlEventsToMetricsMutationStructFields = `
		successes {
		` + graphqlEventsToMetricsRuleStructFields + `
		}
		failures {
			errors {
				description
				reason
			}
		}
`

	getEventsToMetricsRulesQuery = `query($ruleIds: [ID]!, $accountId: Int!) { actor { account(id: $accountId) { eventsToMetrics { rulesById(ruleIds: $ruleIds) { rules {` +
		graphqlEventsToMetricsRuleStructFields +
		` } } } } } }`

	listEventsToMetricsRulesQuery = `query($accountId: Int!) { actor { account(id: $accountId) { eventsToMetrics { allRules { rules {` +
		graphqlEventsToMetricsRuleStructFields +
		` } } } } } }`

	eventsToMetricsCreateRuleMutation = `
		mutation($createInput: [EventsToMetricsCreateRuleInput]!) {
			eventsToMetricsCreateRule(rules: $createInput) {` +
		graphqlEventsToMetricsMutationStructFields +
		` } }`

	eventsToMetricsDeleteRuleMutation = `
		mutation($deleteInput: [EventsToMetricsDeleteRuleInput]!) {
			eventsToMetricsDeleteRule(deletes: $deleteInput) {` +
		graphqlEventsToMetricsMutationStructFields +
		` } }`

	eventsToMetricsUpdateRuleMutation = `
		mutation($updateInput: [EventsToMetricsUpdateRuleInput]!) {
			eventsToMetricsUpdateRule(updates: $updateInput) {` +
		graphqlEventsToMetricsMutationStructFields +
		` } }`
)

type getRulesResponse struct {
	Actor struct {
		Account struct {
			EventsToMetrics struct {
				RulesByID struct {
					Rules []EventsToMetricsRule
				}
			}
		}
	}
}

type listRulesResponse struct {
	Actor struct {
		Account struct {
			EventsToMetrics struct {
				AllRules struct {
					Rules []EventsToMetricsRule
				}
			}
		}
	}
}

type createRuleResponse struct {
	EventsToMetricsCreateRule EventsToMetricsCreateRuleResult
}

type updateRuleResponse struct {
	EventsToMetricsUpdateRule EventsToMetricsUpdateRuleResult
}

type deleteRuleResponse struct {
	EventsToMetricsDeleteRule EventsToMetricsDeleteRuleResult
}

func (r EventsToMetricsCreateRuleResult) Errors() string {
	var errors []string
	for _, e := range r.Failures {
		errors = append(errors, e.String())
	}

	return strings.Join(errors, ", ")
}

func (f EventsToMetricsCreateRuleFailure) String() string {
	var errors []string
	for _, e := range f.Errors {
		errors = append(errors, e.String())
	}

	return strings.Join(errors, ", ")
}

func (r EventsToMetricsUpdateRuleResult) Errors() string {
	var errors []string
	for _, e := range r.Failures {
		errors = append(errors, e.String())
	}

	return strings.Join(errors, ", ")
}

func (f EventsToMetricsUpdateRuleFailure) String() string {
	var errors []string
	for _, e := range f.Errors {
		errors = append(errors, e.String())
	}

	return strings.Join(errors, ", ")
}

func (r EventsToMetricsDeleteRuleResult) Errors() string {
	var errors []string
	for _, e := range r.Failures {
		errors = append(errors, e.String())
	}

	return strings.Join(errors, ", ")
}

func (f EventsToMetricsDeleteRuleFailure) String() string {
	var errors []string
	for _, e := range f.Errors {
		errors = append(errors, e.String())
	}

	return strings.Join(errors, ", ")
}

func (e EventsToMetricsError) String() string {
	return fmt.Sprintf("Reason: %s, Description: %s", e.Reason, e.Description)
}
