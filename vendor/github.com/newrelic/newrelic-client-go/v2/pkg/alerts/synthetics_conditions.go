package alerts

import (
	"context"
	"fmt"

	"github.com/newrelic/newrelic-client-go/v2/pkg/errors"
)

// SyntheticsCondition represents a New Relic Synthetics alert condition.
type SyntheticsCondition struct {
	ID         int    `json:"id,omitempty"`
	Name       string `json:"name,omitempty"`
	Enabled    bool   `json:"enabled"`
	RunbookURL string `json:"runbook_url,omitempty"`
	MonitorID  string `json:"monitor_id,omitempty"`
}

// ListSyntheticsConditions returns a list of Synthetics alert conditions for a given policy.
func (a *Alerts) ListSyntheticsConditions(policyID int) ([]*SyntheticsCondition, error) {
	return a.ListSyntheticsConditionsWithContext(context.Background(), policyID)
}

// ListSyntheticsConditionsWithContext returns a list of Synthetics alert conditions for a given policy.
func (a *Alerts) ListSyntheticsConditionsWithContext(ctx context.Context, policyID int) ([]*SyntheticsCondition, error) {
	conditions := []*SyntheticsCondition{}
	nextURL := a.config.Region().RestURL("/alerts_synthetics_conditions.json")
	queryParams := listSyntheticsConditionsParams{
		PolicyID: policyID,
	}

	for nextURL != "" {
		response := syntheticsConditionsResponse{}
		resp, err := a.client.GetWithContext(ctx, nextURL, &queryParams, &response)

		if err != nil {
			return nil, err
		}

		conditions = append(conditions, response.Conditions...)

		paging := a.pager.Parse(resp)
		nextURL = paging.Next
	}

	return conditions, nil
}

// GetSyntheticsCondition retrieves a specific Synthetics alert condition.
func (a *Alerts) GetSyntheticsCondition(policyID int, conditionID int) (*SyntheticsCondition, error) {
	return a.GetSyntheticsConditionWithContext(context.Background(), policyID, conditionID)
}

// GetSyntheticsConditionWithContext retrieves a specific Synthetics alert condition.
func (a *Alerts) GetSyntheticsConditionWithContext(ctx context.Context, policyID int, conditionID int) (*SyntheticsCondition, error) {
	conditions, err := a.ListSyntheticsConditionsWithContext(ctx, policyID)

	if err != nil {
		return nil, err
	}

	for _, c := range conditions {
		if c.ID == conditionID {
			return c, nil
		}
	}

	return nil, errors.NewNotFoundf("no condition found for policy %d and condition ID %d", policyID, conditionID)
}

// CreateSyntheticsCondition creates a new Synthetics alert condition.
func (a *Alerts) CreateSyntheticsCondition(policyID int, condition SyntheticsCondition) (*SyntheticsCondition, error) {
	return a.CreateSyntheticsConditionWithContext(context.Background(), policyID, condition)
}

// CreateSyntheticsConditionWithContext creates a new Synthetics alert condition.
func (a *Alerts) CreateSyntheticsConditionWithContext(ctx context.Context, policyID int, condition SyntheticsCondition) (*SyntheticsCondition, error) {
	resp := syntheticsConditionResponse{}
	reqBody := syntheticsConditionRequest{condition}
	url := fmt.Sprintf("/alerts_synthetics_conditions/policies/%d.json", policyID)
	_, err := a.client.PostWithContext(ctx, a.config.Region().RestURL(url), nil, &reqBody, &resp)

	if err != nil {
		return nil, err
	}

	return &resp.Condition, nil
}

// UpdateSyntheticsCondition updates an existing Synthetics alert condition.
func (a *Alerts) UpdateSyntheticsCondition(condition SyntheticsCondition) (*SyntheticsCondition, error) {
	return a.UpdateSyntheticsConditionWithContext(context.Background(), condition)
}

// UpdateSyntheticsConditionWithContext updates an existing Synthetics alert condition.
func (a *Alerts) UpdateSyntheticsConditionWithContext(ctx context.Context, condition SyntheticsCondition) (*SyntheticsCondition, error) {
	resp := syntheticsConditionResponse{}
	reqBody := syntheticsConditionRequest{condition}
	url := fmt.Sprintf("/alerts_synthetics_conditions/%d.json", condition.ID)
	_, err := a.client.PutWithContext(ctx, a.config.Region().RestURL(url), nil, &reqBody, &resp)

	if err != nil {
		return nil, err
	}

	return &resp.Condition, nil
}

// DeleteSyntheticsCondition deletes a Synthetics alert condition.
func (a *Alerts) DeleteSyntheticsCondition(conditionID int) (*SyntheticsCondition, error) {
	return a.DeleteSyntheticsConditionWithContext(context.Background(), conditionID)
}

// DeleteSyntheticsConditionWithContext deletes a Synthetics alert condition.
func (a *Alerts) DeleteSyntheticsConditionWithContext(ctx context.Context, conditionID int) (*SyntheticsCondition, error) {
	resp := syntheticsConditionResponse{}
	url := fmt.Sprintf("/alerts_synthetics_conditions/%d.json", conditionID)
	_, err := a.client.DeleteWithContext(ctx, a.config.Region().RestURL(url), nil, &resp)

	if err != nil {
		return nil, err
	}

	return &resp.Condition, nil
}

type listSyntheticsConditionsParams struct {
	PolicyID int `url:"policy_id,omitempty"`
}

type syntheticsConditionsResponse struct {
	Conditions []*SyntheticsCondition `json:"synthetics_conditions,omitempty"`
}

type syntheticsConditionResponse struct {
	Condition SyntheticsCondition `json:"synthetics_condition,omitempty"`
}

type syntheticsConditionRequest struct {
	Condition SyntheticsCondition `json:"synthetics_condition,omitempty"`
}
