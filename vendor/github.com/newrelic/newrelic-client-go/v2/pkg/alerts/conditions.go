package alerts

import (
	"context"
	"fmt"

	"github.com/newrelic/newrelic-client-go/v2/pkg/errors"
)

// ConditionType specifies the condition type used when creating the alert condition.
type ConditionType string

var (
	// ConditionTypes enumerates the possible condition types for an alert condition.
	ConditionTypes = struct {
		APMApplicationMetric    ConditionType
		APMKeyTransactionMetric ConditionType
		ServersMetric           ConditionType
		BrowserMetric           ConditionType
		MobileMetric            ConditionType
	}{
		APMApplicationMetric:    "apm_app_metric",
		APMKeyTransactionMetric: "apm_kt_metric",
		ServersMetric:           "servers_metric",
		BrowserMetric:           "browser_metric",
		MobileMetric:            "mobile_metric",
	}
)

// MetricType specifies the metric type used when creating the alert condition.
type MetricType string

var (
	// MetricTypes enumerates the possible metric types for an alert condition.
	// Not all metric types are valid for all condition types.  See the docuentation for more details.
	MetricTypes = struct {
		AjaxResponseTime       MetricType
		AjaxThroughput         MetricType
		Apdex                  MetricType
		CPUPercentage          MetricType
		Database               MetricType
		DiskIOPercentage       MetricType
		DomProcessing          MetricType
		EndUserApdex           MetricType
		ErrorCount             MetricType
		ErrorPercentage        MetricType
		FullestDiskPercentage  MetricType
		Images                 MetricType
		JSON                   MetricType
		LoadAverageOneMinute   MetricType
		MemoryPercentage       MetricType
		MobileCrashRate        MetricType
		Network                MetricType
		NetworkErrorPercentage MetricType
		PageRendering          MetricType
		PageViewThroughput     MetricType
		PageViewsWithJsErrors  MetricType
		RequestQueuing         MetricType
		ResponseTime           MetricType
		ResponseTimeBackground MetricType
		ResponseTimeWeb        MetricType
		StatusErrorPercentage  MetricType
		Throughput             MetricType
		ThroughputBackground   MetricType
		ThroughputWeb          MetricType
		TotalPageLoad          MetricType
		UserDefined            MetricType
		ViewLoading            MetricType
		WebApplication         MetricType
	}{
		AjaxResponseTime:       "ajax_response_time",
		AjaxThroughput:         "ajax_throughput",
		Apdex:                  "apdex",
		CPUPercentage:          "cpu_percentage",
		Database:               "database",
		DiskIOPercentage:       "disk_io_percentage",
		DomProcessing:          "dom_processing",
		EndUserApdex:           "end_user_apdex",
		ErrorCount:             "error_count",
		ErrorPercentage:        "error_percentage",
		FullestDiskPercentage:  "fullest_disk_percentage",
		Images:                 "images",
		JSON:                   "json",
		LoadAverageOneMinute:   "load_average_one_minute",
		MemoryPercentage:       "memory_percentage",
		MobileCrashRate:        "mobile_crash_rate",
		Network:                "network",
		NetworkErrorPercentage: "network_error_percentage",
		PageRendering:          "page_rendering",
		PageViewThroughput:     "page_view_throughput",
		PageViewsWithJsErrors:  "page_views_with_js_errors",
		RequestQueuing:         "request_queuing",
		ResponseTime:           "response_time",
		ResponseTimeBackground: "response_time_background",
		ResponseTimeWeb:        "response_time_web",
		StatusErrorPercentage:  "status_error_percentage",
		Throughput:             "throughput",
		ThroughputBackground:   "throughput_background",
		ThroughputWeb:          "throughput_web",
		TotalPageLoad:          "total_page_load",
		UserDefined:            "user_defined",
		ViewLoading:            "view_loading",
		WebApplication:         "web_application",
	}
)

// OperatorType specifies the operator for alert condition terms.
type OperatorType string

var (
	// OperatorTypes enumerates the possible operator values for alert condition terms.
	OperatorTypes = struct {
		Above OperatorType
		Below OperatorType
		Equal OperatorType
	}{
		Above: "above",
		Below: "below",
		Equal: "equal",
	}
)

// PriorityType specifies the priority for alert condition terms.
type PriorityType string

var (
	// PriorityTypes enumerates the possible priority values for alert condition terms.
	PriorityTypes = struct {
		Critical PriorityType
		Warning  PriorityType
	}{
		Critical: "critical",
		Warning:  "warning",
	}
)

// TimeFunctionType specifies the time function to be used for alert condition terms.
type TimeFunctionType string

var (
	// TimeFunctionTypes enumerates the possible time function types for alert condition terms.
	TimeFunctionTypes = struct {
		All TimeFunctionType
		Any TimeFunctionType
	}{
		All: "all",
		Any: "any",
	}
)

// ValueFunctionType specifies the value function to be used for returning custom metric data.
type ValueFunctionType string

var (
	// ValueFunctionTypes enumerates the possible value function types for custom metrics.
	ValueFunctionTypes = struct {
		Average     ValueFunctionType
		Min         ValueFunctionType
		Max         ValueFunctionType
		Total       ValueFunctionType
		SampleSize  ValueFunctionType
		SingleValue ValueFunctionType
		Rate        ValueFunctionType
		Percent     ValueFunctionType
	}{
		Average:     "average",
		Min:         "min",
		Max:         "max",
		Total:       "total",
		SampleSize:  "sample_size",
		SingleValue: "single_value",
		Rate:        "rate",
		Percent:     "percent",
	}
)

// Condition represents a New Relic alert condition.
// TODO: custom unmarshal entities to ints?
type Condition struct {
	ID                  int                  `json:"id,omitempty"`
	Type                ConditionType        `json:"type,omitempty"`
	Name                string               `json:"name,omitempty"`
	Enabled             bool                 `json:"enabled"`
	Entities            []string             `json:"entities,omitempty"`
	Metric              MetricType           `json:"metric,omitempty"`
	RunbookURL          string               `json:"runbook_url"`
	Terms               []ConditionTerm      `json:"terms,omitempty"`
	UserDefined         ConditionUserDefined `json:"user_defined,omitempty"`
	Scope               string               `json:"condition_scope,omitempty"`
	GCMetric            string               `json:"gc_metric,omitempty"`
	ViolationCloseTimer int                  `json:"violation_close_timer,omitempty"`
}

// ConditionUserDefined represents user defined metrics for the New Relic alert condition.
type ConditionUserDefined struct {
	Metric        string            `json:"metric,omitempty"`
	ValueFunction ValueFunctionType `json:"value_function,omitempty"`
}

// ConditionTerm represents the terms of a New Relic alert condition.
type ConditionTerm struct {
	Duration     int              `json:"duration,string,omitempty"`
	Operator     OperatorType     `json:"operator,omitempty"`
	Priority     PriorityType     `json:"priority,omitempty"`
	Threshold    float64          `json:"threshold,string"`
	TimeFunction TimeFunctionType `json:"time_function,omitempty"`
}

// ListConditions returns alert conditions for a specified policy.
func (a *Alerts) ListConditions(policyID int) ([]*Condition, error) {
	return a.ListConditionsWithContext(context.Background(), policyID)
}

// ListConditionsWithContext returns alert conditions for a specified policy.
func (a *Alerts) ListConditionsWithContext(ctx context.Context, policyID int) ([]*Condition, error) {
	alertConditions := []*Condition{}
	queryParams := listConditionsParams{
		PolicyID: policyID,
	}

	nextURL := a.config.Region().RestURL("/alerts_conditions.json")

	for nextURL != "" {
		response := alertConditionsResponse{}
		resp, err := a.client.GetWithContext(ctx, nextURL, &queryParams, &response)

		if err != nil {
			return nil, err
		}

		alertConditions = append(alertConditions, response.Conditions...)

		paging := a.pager.Parse(resp)
		nextURL = paging.Next
	}

	return alertConditions, nil
}

// GetCondition gets an alert condition for a specified policy ID and condition ID.
func (a *Alerts) GetCondition(policyID int, id int) (*Condition, error) {
	return a.GetConditionWithContext(context.Background(), policyID, id)
}

// GetConditionWithContext gets an alert condition for a specified policy ID and condition ID.
func (a *Alerts) GetConditionWithContext(ctx context.Context, policyID int, id int) (*Condition, error) {
	conditions, err := a.ListConditionsWithContext(ctx, policyID)
	if err != nil {
		return nil, err
	}

	for _, condition := range conditions {
		if condition.ID == id {
			return condition, nil
		}
	}

	return nil, errors.NewNotFoundf("no condition found for policy %d and condition ID %d", policyID, id)
}

// CreateCondition creates an alert condition for a specified policy.
func (a *Alerts) CreateCondition(policyID int, condition Condition) (*Condition, error) {
	return a.CreateConditionWithContext(context.Background(), policyID, condition)
}

// CreateConditionWithContext creates an alert condition for a specified policy.
func (a *Alerts) CreateConditionWithContext(ctx context.Context, policyID int, condition Condition) (*Condition, error) {
	reqBody := alertConditionRequestBody{
		Condition: condition,
	}
	resp := alertConditionResponse{}

	url := fmt.Sprintf("/alerts_conditions/policies/%d.json", policyID)
	_, err := a.client.PostWithContext(ctx, a.config.Region().RestURL(url), nil, &reqBody, &resp)

	if err != nil {
		return nil, err
	}

	return &resp.Condition, nil
}

// UpdateCondition updates an alert condition.
func (a *Alerts) UpdateCondition(condition Condition) (*Condition, error) {
	return a.UpdateConditionWithContext(context.Background(), condition)
}

// UpdateConditionWithContext updates an alert condition.
func (a *Alerts) UpdateConditionWithContext(ctx context.Context, condition Condition) (*Condition, error) {
	reqBody := alertConditionRequestBody{
		Condition: condition,
	}
	resp := alertConditionResponse{}

	url := fmt.Sprintf("/alerts_conditions/%d.json", condition.ID)
	_, err := a.client.PutWithContext(ctx, a.config.Region().RestURL(url), nil, &reqBody, &resp)

	if err != nil {
		return nil, err
	}

	return &resp.Condition, nil
}

// DeleteCondition deletes an alert condition.
func (a *Alerts) DeleteCondition(id int) (*Condition, error) {
	return a.DeleteConditionWithContext(context.Background(), id)
}

// DeleteConditionWithContext deletes an alert condition.
func (a *Alerts) DeleteConditionWithContext(ctx context.Context, id int) (*Condition, error) {
	resp := alertConditionResponse{}
	url := fmt.Sprintf("/alerts_conditions/%d.json", id)

	_, err := a.client.DeleteWithContext(ctx, a.config.Region().RestURL(url), nil, &resp)

	if err != nil {
		return nil, err
	}

	return &resp.Condition, nil
}

// DeleteConditionMutation deletes any type of alert condition via New Relic's NerdGraph API.
func (a *Alerts) DeleteConditionMutation(
	accountID int,
	conditionID string,
) (string, error) {
	return a.DeleteConditionMutationWithContext(context.Background(), accountID, conditionID)
}

// DeleteConditionMutationWithContext deletes any type of alert condition via New Relic's NerdGraph API.
func (a *Alerts) DeleteConditionMutationWithContext(
	ctx context.Context,
	accountID int,
	conditionID string,
) (string, error) {
	resp := conditionDeleteResponse{}
	vars := map[string]interface{}{
		"accountId": accountID,
		"id":        conditionID,
	}

	if err := a.client.NerdGraphQueryWithContext(ctx, deleteConditionMutation, vars, &resp); err != nil {
		return "", err
	}

	return resp.AlertsConditionDelete.ID, nil
}

type listConditionsParams struct {
	PolicyID int `url:"policy_id,omitempty"`
}

type alertConditionsResponse struct {
	Conditions []*Condition `json:"conditions,omitempty"`
}

type alertConditionResponse struct {
	Condition Condition `json:"condition,omitempty"`
}

type alertConditionRequestBody struct {
	Condition Condition `json:"condition,omitempty"`
}

type conditionDeleteResponse struct {
	AlertsConditionDelete struct {
		ID string `json:"id,omitempty"`
	} `json:"alertsConditionDelete"`
}

const (
	deleteConditionMutation = `
		mutation($accountId: Int!, $id: ID!) {
			alertsConditionDelete(accountId: $accountId, id: $id) {
				id
			}
		}`
)
