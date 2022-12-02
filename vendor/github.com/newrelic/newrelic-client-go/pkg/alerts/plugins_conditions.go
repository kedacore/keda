package alerts

import (
	"context"
	"fmt"

	"github.com/newrelic/newrelic-client-go/pkg/errors"
)

// PluginsCondition represents an alert condition for New Relic Plugins.
type PluginsCondition struct {
	ID                int             `json:"id,omitempty"`
	Name              string          `json:"name,omitempty"`
	Enabled           bool            `json:"enabled"`
	Entities          []string        `json:"entities,omitempty"`
	Metric            string          `json:"metric,omitempty"`
	MetricDescription string          `json:"metric_description,omitempty"`
	RunbookURL        string          `json:"runbook_url,omitempty"`
	Terms             []ConditionTerm `json:"terms,omitempty"`
	ValueFunction     string          `json:"value_function,omitempty"`
	Plugin            AlertPlugin     `json:"plugin,omitempty"`
}

// AlertPlugin represents a plugin to use with a Plugin alert condition.
type AlertPlugin struct {
	ID   string `json:"id,omitempty"`
	GUID string `json:"guid,omitempty"`
}

// ListPluginsConditions returns alert conditions for New Relic plugins for a given alert policy.
func (a *Alerts) ListPluginsConditions(policyID int) ([]*PluginsCondition, error) {
	return a.ListPluginsConditionsWithContext(context.Background(), policyID)
}

// ListPluginsConditionsWithContext returns alert conditions for New Relic plugins for a given alert policy.
func (a *Alerts) ListPluginsConditionsWithContext(ctx context.Context, policyID int) ([]*PluginsCondition, error) {
	conditions := []*PluginsCondition{}
	queryParams := listPluginsConditionsParams{
		PolicyID: policyID,
	}

	nextURL := a.config.Region().RestURL("/alerts_plugins_conditions.json")

	for nextURL != "" {
		response := pluginsConditionsResponse{}
		resp, err := a.client.GetWithContext(ctx, nextURL, &queryParams, &response)

		if err != nil {
			return nil, err
		}

		conditions = append(conditions, response.PluginsConditions...)

		paging := a.pager.Parse(resp)
		nextURL = paging.Next
	}

	return conditions, nil
}

// GetPluginsCondition gets information about an alert condition for a plugin
// given a policy ID and plugin ID.
func (a *Alerts) GetPluginsCondition(policyID int, pluginID int) (*PluginsCondition, error) {
	return a.GetPluginsConditionWithContext(context.Background(), policyID, pluginID)
}

// GetPluginsConditionWithContext gets information about an alert condition for a plugin
// given a policy ID and plugin ID.
func (a *Alerts) GetPluginsConditionWithContext(ctx context.Context, policyID int, pluginID int) (*PluginsCondition, error) {
	conditions, err := a.ListPluginsConditionsWithContext(ctx, policyID)

	if err != nil {
		return nil, err
	}

	for _, condition := range conditions {
		if condition.ID == pluginID {
			return condition, nil
		}
	}

	return nil, errors.NewNotFoundf("no condition found for policy %d and condition ID %d", policyID, pluginID)
}

// CreatePluginsCondition creates an alert condition for a plugin.
func (a *Alerts) CreatePluginsCondition(policyID int, condition PluginsCondition) (*PluginsCondition, error) {
	return a.CreatePluginsConditionWithContext(context.Background(), policyID, condition)
}

// CreatePluginsConditionWithContext creates an alert condition for a plugin.
func (a *Alerts) CreatePluginsConditionWithContext(ctx context.Context, policyID int, condition PluginsCondition) (*PluginsCondition, error) {
	reqBody := pluginConditionRequestBody{
		PluginsCondition: condition,
	}
	resp := pluginConditionResponse{}

	url := fmt.Sprintf("/alerts_plugins_conditions/policies/%d.json", policyID)
	_, err := a.client.PostWithContext(ctx, a.config.Region().RestURL(url), nil, &reqBody, &resp)

	if err != nil {
		return nil, err
	}

	return &resp.PluginsCondition, nil
}

// UpdatePluginsCondition updates an alert condition for a plugin.
func (a *Alerts) UpdatePluginsCondition(condition PluginsCondition) (*PluginsCondition, error) {
	return a.UpdatePluginsConditionWithContext(context.Background(), condition)
}

// UpdatePluginsConditionWithContext updates an alert condition for a plugin.
func (a *Alerts) UpdatePluginsConditionWithContext(ctx context.Context, condition PluginsCondition) (*PluginsCondition, error) {
	reqBody := pluginConditionRequestBody{
		PluginsCondition: condition,
	}
	resp := pluginConditionResponse{}

	url := fmt.Sprintf("/alerts_plugins_conditions/%d.json", condition.ID)
	_, err := a.client.PutWithContext(ctx, a.config.Region().RestURL(url), nil, &reqBody, &resp)

	if err != nil {
		return nil, err
	}

	return &resp.PluginsCondition, nil
}

// DeletePluginsCondition deletes a plugin alert condition.
func (a *Alerts) DeletePluginsCondition(id int) (*PluginsCondition, error) {
	return a.DeletePluginsConditionWithContext(context.Background(), id)
}

// DeletePluginsConditionWithContext deletes a plugin alert condition.
func (a *Alerts) DeletePluginsConditionWithContext(ctx context.Context, id int) (*PluginsCondition, error) {
	resp := pluginConditionResponse{}
	url := fmt.Sprintf("/alerts_plugins_conditions/%d.json", id)

	_, err := a.client.DeleteWithContext(ctx, a.config.Region().RestURL(url), nil, &resp)

	if err != nil {
		return nil, err
	}

	return &resp.PluginsCondition, nil
}

type listPluginsConditionsParams struct {
	PolicyID int `url:"policy_id,omitempty"`
}

type pluginsConditionsResponse struct {
	PluginsConditions []*PluginsCondition `json:"plugins_conditions,omitempty"`
}

type pluginConditionResponse struct {
	PluginsCondition PluginsCondition `json:"plugins_condition,omitempty"`
}

type pluginConditionRequestBody struct {
	PluginsCondition PluginsCondition `json:"plugins_condition,omitempty"`
}
