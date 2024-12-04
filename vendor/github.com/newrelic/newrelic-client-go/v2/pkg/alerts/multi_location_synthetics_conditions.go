package alerts

import (
	"context"
	"fmt"
	"strconv"

	"github.com/newrelic/newrelic-client-go/v2/pkg/errors"
)

// MultiLocationSyntheticsCondition represents a location-based failure condition.
//
// ViolationTimeLimitSeconds must be one of 3600, 7200, 14400, 28800, 43200, 86400.
type MultiLocationSyntheticsCondition struct {
	ID                        int                                    `json:"id,omitempty"`
	Name                      string                                 `json:"name,omitempty"`
	Enabled                   bool                                   `json:"enabled"`
	RunbookURL                string                                 `json:"runbook_url,omitempty"`
	Entities                  []string                               `json:"entities,omitempty"`
	Terms                     []MultiLocationSyntheticsConditionTerm `json:"terms,omitempty"`
	ViolationTimeLimitSeconds int                                    `json:"violation_time_limit_seconds,omitempty"`
}

// MultiLocationSyntheticsConditionTerm represents a single term for a location-based failure condition.
//
// Priority must be "warning" or "critical".
// Threshold must be greater than zero.
type MultiLocationSyntheticsConditionTerm struct {
	Priority  string `json:"priority,omitempty"`
	Threshold int    `json:"threshold,omitempty"`
}

// ListMultiLocationSyntheticsConditions returns alert conditions for a specified policy.
func (a *Alerts) ListMultiLocationSyntheticsConditions(policyID int) ([]*MultiLocationSyntheticsCondition, error) {
	return a.ListMultiLocationSyntheticsConditionsWithContext(context.Background(), policyID)
}

// ListMultiLocationSyntheticsConditionsWithContext returns alert conditions for a specified policy.
func (a *Alerts) ListMultiLocationSyntheticsConditionsWithContext(ctx context.Context, policyID int) ([]*MultiLocationSyntheticsCondition, error) {
	response := multiLocationSyntheticsConditionListResponse{}
	queryParams := listMultiLocationSyntheticsConditionsParams{
		PolicyID: policyID,
	}

	url := a.config.Region().RestURL("/alerts_location_failure_conditions/policies/", strconv.Itoa(policyID)+".json")
	_, err := a.client.GetWithContext(ctx, url, &queryParams, &response)

	if err != nil {
		return nil, err
	}

	return response.MultiLocationSyntheticsConditions, nil
}

// GetMultiLocationSyntheticsCondition retrieves a specific Synthetics alert condition.
func (a *Alerts) GetMultiLocationSyntheticsCondition(policyID int, conditionID int) (*MultiLocationSyntheticsCondition, error) {
	return a.GetMultiLocationSyntheticsConditionWithContext(context.Background(), policyID, conditionID)
}

// GetMultiLocationSyntheticsConditionWithContext retrieves a specific Synthetics alert condition.
func (a *Alerts) GetMultiLocationSyntheticsConditionWithContext(ctx context.Context, policyID int, conditionID int) (*MultiLocationSyntheticsCondition, error) {
	conditions, err := a.ListMultiLocationSyntheticsConditionsWithContext(ctx, policyID)

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

// CreateMultiLocationSyntheticsCondition creates an alert condition for a specified policy.
func (a *Alerts) CreateMultiLocationSyntheticsCondition(condition MultiLocationSyntheticsCondition, policyID int) (*MultiLocationSyntheticsCondition, error) {
	return a.CreateMultiLocationSyntheticsConditionWithContext(context.Background(), condition, policyID)

}

// CreateMultiLocationSyntheticsConditionWithContext creates an alert condition for a specified policy.
func (a *Alerts) CreateMultiLocationSyntheticsConditionWithContext(ctx context.Context, condition MultiLocationSyntheticsCondition, policyID int) (*MultiLocationSyntheticsCondition, error) {
	reqBody := multiLocationSyntheticsConditionRequestBody{
		MultiLocationSyntheticsCondition: condition,
	}
	resp := multiLocationSyntheticsConditionCreateResponse{}

	url := fmt.Sprintf("/alerts_location_failure_conditions/policies/%d.json", policyID)
	_, err := a.client.PostWithContext(ctx, a.config.Region().RestURL(url), nil, &reqBody, &resp)
	if err != nil {
		return nil, err
	}

	return &resp.MultiLocationSyntheticsCondition, nil
}

// UpdateMultiLocationSyntheticsCondition updates an alert condition.
func (a *Alerts) UpdateMultiLocationSyntheticsCondition(condition MultiLocationSyntheticsCondition) (*MultiLocationSyntheticsCondition, error) {
	return a.UpdateMultiLocationSyntheticsConditionWithContext(context.Background(), condition)
}

// UpdateMultiLocationSyntheticsConditionWithContext updates an alert condition.
func (a *Alerts) UpdateMultiLocationSyntheticsConditionWithContext(ctx context.Context, condition MultiLocationSyntheticsCondition) (*MultiLocationSyntheticsCondition, error) {
	reqBody := multiLocationSyntheticsConditionRequestBody{
		MultiLocationSyntheticsCondition: condition,
	}
	resp := multiLocationSyntheticsConditionCreateResponse{}

	url := fmt.Sprintf("/alerts_location_failure_conditions/%d.json", condition.ID)
	_, err := a.client.PutWithContext(ctx, a.config.Region().RestURL(url), nil, &reqBody, &resp)

	if err != nil {
		return nil, err
	}

	return &resp.MultiLocationSyntheticsCondition, nil
}

// DeleteMultiLocationSyntheticsCondition delete an alert condition.
func (a *Alerts) DeleteMultiLocationSyntheticsCondition(conditionID int) (*MultiLocationSyntheticsCondition, error) {
	return a.DeleteMultiLocationSyntheticsConditionWithContext(context.Background(), conditionID)
}

// DeleteMultiLocationSyntheticsConditionWithContext delete an alert condition.
func (a *Alerts) DeleteMultiLocationSyntheticsConditionWithContext(ctx context.Context, conditionID int) (*MultiLocationSyntheticsCondition, error) {
	resp := multiLocationSyntheticsConditionCreateResponse{}
	url := fmt.Sprintf("/alerts_conditions/%d.json", conditionID)

	_, err := a.client.DeleteWithContext(ctx, a.config.Region().RestURL(url), nil, &resp)

	if err != nil {
		return nil, err
	}

	return &resp.MultiLocationSyntheticsCondition, nil
}

type listMultiLocationSyntheticsConditionsParams struct {
	PolicyID int `url:"policy_id,omitempty"`
}

type multiLocationSyntheticsConditionListResponse struct {
	MultiLocationSyntheticsConditions []*MultiLocationSyntheticsCondition `json:"location_failure_conditions,omitempty"`
}

type multiLocationSyntheticsConditionCreateResponse struct {
	MultiLocationSyntheticsCondition MultiLocationSyntheticsCondition `json:"location_failure_condition,omitempty"`
}

type multiLocationSyntheticsConditionRequestBody struct {
	MultiLocationSyntheticsCondition MultiLocationSyntheticsCondition `json:"location_failure_condition,omitempty"`
}
