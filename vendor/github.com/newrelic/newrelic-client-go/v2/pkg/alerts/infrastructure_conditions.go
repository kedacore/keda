package alerts

import (
	"context"
	"fmt"

	"github.com/newrelic/newrelic-client-go/v2/internal/serialization"
)

// InfrastructureCondition represents a New Relic Infrastructure alert condition.
type InfrastructureCondition struct {
	Comparison          string                            `json:"comparison,omitempty"`
	CreatedAt           *serialization.EpochTime          `json:"created_at_epoch_millis,omitempty"`
	Critical            *InfrastructureConditionThreshold `json:"critical_threshold,omitempty"`
	Enabled             bool                              `json:"enabled"`
	Event               string                            `json:"event_type,omitempty"`
	ID                  int                               `json:"id,omitempty"`
	IntegrationProvider string                            `json:"integration_provider,omitempty"`
	Name                string                            `json:"name,omitempty"`
	PolicyID            int                               `json:"policy_id,omitempty"`
	ProcessWhere        string                            `json:"process_where_clause,omitempty"`
	RunbookURL          string                            `json:"runbook_url,omitempty"`
	Select              string                            `json:"select_value,omitempty"`
	Type                string                            `json:"type,omitempty"`
	UpdatedAt           *serialization.EpochTime          `json:"updated_at_epoch_millis,omitempty"`
	ViolationCloseTimer *int                              `json:"violation_close_timer,omitempty"`
	Warning             *InfrastructureConditionThreshold `json:"warning_threshold,omitempty"`
	Where               string                            `json:"where_clause,omitempty"`
	Description         string                            `json:"description"`
}

// InfrastructureConditionThreshold represents an New Relic Infrastructure alert condition threshold.
type InfrastructureConditionThreshold struct {
	Duration int      `json:"duration_minutes,omitempty"`
	Function string   `json:"time_function,omitempty"`
	Value    *float64 `json:"value"`
}

// ListInfrastructureConditions is used to retrieve New Relic Infrastructure alert conditions.
func (a *Alerts) ListInfrastructureConditions(policyID int) ([]InfrastructureCondition, error) {
	return a.ListInfrastructureConditionsWithContext(context.Background(), policyID)
}

// ListInfrastructureConditionsWithContext is used to retrieve New Relic Infrastructure alert conditions.
func (a *Alerts) ListInfrastructureConditionsWithContext(ctx context.Context, policyID int) ([]InfrastructureCondition, error) {
	resp := infrastructureConditionsResponse{}
	queryParams := listInfrastructureConditionsParams{
		PolicyID: policyID,
	}
	_, err := a.infraClient.GetWithContext(ctx, a.config.Region().InfrastructureURL("/alerts/conditions"), &queryParams, &resp)

	if err != nil {
		return nil, err
	}

	return resp.Conditions, nil
}

// GetInfrastructureCondition is used to retrieve a specific New Relic Infrastructure alert condition.
func (a *Alerts) GetInfrastructureCondition(conditionID int) (*InfrastructureCondition, error) {
	return a.GetInfrastructureConditionWithContext(context.Background(), conditionID)
}

// GetInfrastructureConditionWithContext is used to retrieve a specific New Relic Infrastructure alert condition.
func (a *Alerts) GetInfrastructureConditionWithContext(ctx context.Context, conditionID int) (*InfrastructureCondition, error) {
	resp := infrastructureConditionResponse{}
	url := fmt.Sprintf("/alerts/conditions/%d", conditionID)
	_, err := a.infraClient.GetWithContext(ctx, a.config.Region().InfrastructureURL(url), nil, &resp)

	if err != nil {
		return nil, err
	}

	return &resp.Condition, nil
}

// CreateInfrastructureCondition is used to create a New Relic Infrastructure alert condition.
func (a *Alerts) CreateInfrastructureCondition(condition InfrastructureCondition) (*InfrastructureCondition, error) {
	return a.CreateInfrastructureConditionWithContext(context.Background(), condition)
}

// CreateInfrastructureConditionWithContext is used to create a New Relic Infrastructure alert condition.
func (a *Alerts) CreateInfrastructureConditionWithContext(ctx context.Context, condition InfrastructureCondition) (*InfrastructureCondition, error) {
	resp := infrastructureConditionResponse{}
	reqBody := infrastructureConditionRequest{condition}

	_, err := a.infraClient.PostWithContext(ctx, a.config.Region().InfrastructureURL("/alerts/conditions"), nil, &reqBody, &resp)

	if err != nil {
		return nil, err
	}

	return &resp.Condition, nil
}

// UpdateInfrastructureCondition is used to update a New Relic Infrastructure alert condition.
func (a *Alerts) UpdateInfrastructureCondition(condition InfrastructureCondition) (*InfrastructureCondition, error) {
	return a.UpdateInfrastructureConditionWithContext(context.Background(), condition)
}

// UpdateInfrastructureConditionWithContext is used to update a New Relic Infrastructure alert condition.
func (a *Alerts) UpdateInfrastructureConditionWithContext(ctx context.Context, condition InfrastructureCondition) (*InfrastructureCondition, error) {
	resp := infrastructureConditionResponse{}
	reqBody := infrastructureConditionRequest{condition}

	url := fmt.Sprintf("/alerts/conditions/%d", condition.ID)
	_, err := a.infraClient.PutWithContext(ctx, a.config.Region().InfrastructureURL(url), nil, &reqBody, &resp)

	if err != nil {
		return nil, err
	}

	return &resp.Condition, nil
}

// DeleteInfrastructureCondition is used to delete a New Relic Infrastructure alert condition.
func (a *Alerts) DeleteInfrastructureCondition(conditionID int) error {
	return a.DeleteInfrastructureConditionWithContext(context.Background(), conditionID)
}

// DeleteInfrastructureConditionWithContext is used to delete a New Relic Infrastructure alert condition.
func (a *Alerts) DeleteInfrastructureConditionWithContext(ctx context.Context, conditionID int) error {
	url := fmt.Sprintf("/alerts/conditions/%d", conditionID)
	_, err := a.infraClient.DeleteWithContext(ctx, a.config.Region().InfrastructureURL(url), nil, nil)

	if err != nil {
		return err
	}

	return nil
}

type listInfrastructureConditionsParams struct {
	PolicyID int `url:"policy_id,omitempty"`
}

type infrastructureConditionsResponse struct {
	Conditions []InfrastructureCondition `json:"data,omitempty"`
}

type infrastructureConditionResponse struct {
	Condition InfrastructureCondition `json:"data,omitempty"`
}

type infrastructureConditionRequest struct {
	Condition InfrastructureCondition `json:"data,omitempty"`
}
