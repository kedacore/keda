package alerts

import (
	"context"
	"fmt"

	"github.com/newrelic/newrelic-client-go/v2/pkg/common"

	"github.com/newrelic/newrelic-client-go/v2/pkg/errors"
)

// AlertsNrqlConditionExpiration
// Settings for how violations are opened or closed when a signal expires.
// nolint:revive
type AlertsNrqlConditionExpiration struct {
	ExpirationDuration          *int `json:"expirationDuration"`
	CloseViolationsOnExpiration bool `json:"closeViolationsOnExpiration"`
	OpenViolationOnExpiration   bool `json:"openViolationOnExpiration"`
	IgnoreOnExpectedTermination bool `json:"ignoreOnExpectedTermination"`
}

// AlertsNrqlConditionSignal - Configuration that defines the signal that the NRQL condition will use to evaluate.
// nolint:revive
type AlertsNrqlConditionSignal struct {
	AggregationWindow *int                            `json:"aggregationWindow,omitempty"`
	EvaluationOffset  *int                            `json:"evaluationOffset,omitempty"`
	EvaluationDelay   *int                            `json:"evaluationDelay,omitempty"`
	FillOption        *AlertsFillOption               `json:"fillOption"`
	FillValue         *float64                        `json:"fillValue"`
	AggregationMethod *NrqlConditionAggregationMethod `json:"aggregationMethod,omitempty"`
	AggregationDelay  *int                            `json:"aggregationDelay,omitempty"`
	AggregationTimer  *int                            `json:"aggregationTimer,omitempty"`
	SlideBy           *int                            `json:"slideBy,omitempty"`
}

// AlertsNrqlConditionCreateSignal - Configuration that defines the signal that the NRQL condition will use to evaluate for Create.
// nolint:revive
type AlertsNrqlConditionCreateSignal struct {
	AggregationWindow *int                            `json:"aggregationWindow,omitempty"`
	EvaluationOffset  *int                            `json:"evaluationOffset,omitempty"`
	EvaluationDelay   *int                            `json:"evaluationDelay,omitempty"`
	FillOption        *AlertsFillOption               `json:"fillOption"`
	FillValue         *float64                        `json:"fillValue"`
	AggregationMethod *NrqlConditionAggregationMethod `json:"aggregationMethod,omitempty"`
	AggregationDelay  *int                            `json:"aggregationDelay,omitempty"`
	AggregationTimer  *int                            `json:"aggregationTimer,omitempty"`
	SlideBy           *int                            `json:"slideBy,omitempty"`
}

// AlertsNrqlConditionUpdateSignal - Configuration that defines the signal that the NRQL condition will use to evaluate for Update.
// nolint:revive
type AlertsNrqlConditionUpdateSignal struct {
	AggregationWindow *int                            `json:"aggregationWindow,omitempty"`
	EvaluationOffset  *int                            `json:"evaluationOffset,omitempty"`
	EvaluationDelay   *int                            `json:"evaluationDelay,omitempty"`
	FillOption        *AlertsFillOption               `json:"fillOption"`
	FillValue         *float64                        `json:"fillValue"`
	AggregationMethod *NrqlConditionAggregationMethod `json:"aggregationMethod"`
	AggregationDelay  *int                            `json:"aggregationDelay"`
	AggregationTimer  *int                            `json:"aggregationTimer"`
	SlideBy           *int                            `json:"slideBy"`
}

// NrqlConditionAggregationMethod - The available aggregation methods.
type NrqlConditionAggregationMethod string

var NrqlConditionAggregationMethodTypes = struct {
	// Streams data points as the clocks at New Relic advance past the end of their window. This ensures a rigorous evaluation cadence,
	// but does not take into account extraneous data latency.
	Cadence NrqlConditionAggregationMethod
	// Streams data points for evaluation as data for newer time windows arrive. Whenever data is received,
	// any data points older than the specified delay will be evaluated.
	EventFlow NrqlConditionAggregationMethod
	// Streams data points after the specified timer elapses since data last arrived for that window. Special measures are
	// taken to make sure data points flow in order.
	EventTimer NrqlConditionAggregationMethod
}{
	// Streams data points as the clocks at New Relic advance past the end of their window. This ensures a rigorous evaluation cadence,
	// but does not take into account extraneous data latency.
	Cadence: "CADENCE",
	// Streams data points for evaluation as data for newer time windows arrive. Whenever data is received,
	// any data points older than the specified delay will be evaluated.
	EventFlow: "EVENT_FLOW",
	// Streams data points after the specified timer elapses since data last arrived for that window. Special measures are
	// taken to make sure data points flow in order.
	EventTimer: "EVENT_TIMER",
}

// AlertsFillOption - The available fill options.
// nolint:revive
type AlertsFillOption string // nolint:golint

// nolint:revive
var AlertsFillOptionTypes = struct {
	// Fill using the last known value.
	LAST_VALUE AlertsFillOption // nolint:golint
	// Do not fill data.
	NONE AlertsFillOption
	// Fill using a static value.
	STATIC AlertsFillOption
}{
	// Fill using the last known value.
	LAST_VALUE: "LAST_VALUE",
	// Do not fill data.
	NONE: "NONE",
	// Fill using a static value.
	STATIC: "STATIC",
}

// ThresholdOccurrence specifies the threshold occurrence for NRQL alert condition terms.
type ThresholdOccurrence string

var (
	// ThresholdOccurrences enumerates the possible threshold occurrence values for NRQL alert condition terms.
	ThresholdOccurrences = struct {
		All         ThresholdOccurrence
		AtLeastOnce ThresholdOccurrence
	}{
		All:         "ALL",
		AtLeastOnce: "AT_LEAST_ONCE",
	}
)

// NrqlConditionType specifies the type of NRQL alert condition.
type NrqlConditionType string

var (
	// NrqlConditionTypes enumerates the possible NRQL condition type values for NRQL alert conditions.
	NrqlConditionTypes = struct {
		Baseline NrqlConditionType
		Static   NrqlConditionType
	}{
		Baseline: "BASELINE",
		Static:   "STATIC",
	}
)

// NrqlConditionViolationTimeLimit specifies the value function of NRQL alert condition.
type NrqlConditionViolationTimeLimit string

var (
	// NrqlConditionViolationTimeLimits enumerates the possible NRQL condition violation time limit values for NRQL alert conditions.
	NrqlConditionViolationTimeLimits = struct {
		OneHour         NrqlConditionViolationTimeLimit
		TwoHours        NrqlConditionViolationTimeLimit
		FourHours       NrqlConditionViolationTimeLimit
		EightHours      NrqlConditionViolationTimeLimit
		TwelveHours     NrqlConditionViolationTimeLimit
		TwentyFourHours NrqlConditionViolationTimeLimit
	}{
		OneHour:         "ONE_HOUR",
		TwoHours:        "TWO_HOURS",
		FourHours:       "FOUR_HOURS",
		EightHours:      "EIGHT_HOURS",
		TwelveHours:     "TWELVE_HOURS",
		TwentyFourHours: "TWENTY_FOUR_HOURS",
	}
)

// NrqlConditionOperator specifies the operator for alert condition terms.
type NrqlConditionOperator string

var (
	// NrqlConditionOperators enumerates the possible operator values for alert condition terms.
	NrqlConditionOperators = struct {
		Above NrqlConditionOperator
		Below NrqlConditionOperator
		Equal NrqlConditionOperator
	}{
		Above: "ABOVE",
		Below: "BELOW",
		Equal: "EQUAL",
	}
)

// NrqlConditionPriority specifies the priority for alert condition terms.
type NrqlConditionPriority string

var (
	// NrqlConditionPriorities enumerates the possible priority values for alert condition terms.
	NrqlConditionPriorities = struct {
		Critical NrqlConditionPriority
		Warning  NrqlConditionPriority
	}{
		Critical: "CRITICAL",
		Warning:  "WARNING",
	}
)

// NrqlBaselineDirection
type NrqlBaselineDirection string

var (
	// NrqlBaselineDirections enumerates the possible baseline direction values for a baseline NRQL alert condition.
	NrqlBaselineDirections = struct {
		LowerOnly     NrqlBaselineDirection
		UpperAndLower NrqlBaselineDirection
		UpperOnly     NrqlBaselineDirection
	}{
		LowerOnly:     "LOWER_ONLY",
		UpperAndLower: "UPPER_AND_LOWER",
		UpperOnly:     "UPPER_ONLY",
	}
)

type NrqlSignalSeasonality string

var (
	// NrqlSignalSeasonalities enumerates the possible signal seasonality values for a baseline NRQL alert condition.
	NrqlSignalSeasonalities = struct {
		NewRelicCalculation NrqlSignalSeasonality
		Hourly              NrqlSignalSeasonality
		Daily               NrqlSignalSeasonality
		Weekly              NrqlSignalSeasonality
		None                NrqlSignalSeasonality
	}{
		NewRelicCalculation: "NEW_RELIC_CALCULATION",
		Hourly:              "HOURLY",
		Daily:               "DAILY",
		Weekly:              "WEEKLY",
		None:                "NONE",
	}
)

type NrqlConditionThresholdPrediction struct {
	PredictBy                 int  `json:"predictBy,omitempty"`
	PreferPredictionViolation bool `json:"preferPredictionViolation"`
}

// NrqlConditionTerm represents the a single term of a New Relic alert condition.
type NrqlConditionTerm struct {
	Operator             AlertsNRQLConditionTermsOperator  `json:"operator,omitempty"`
	Priority             NrqlConditionPriority             `json:"priority,omitempty"`
	Threshold            *float64                          `json:"threshold"`
	ThresholdDuration    int                               `json:"thresholdDuration,omitempty"`
	ThresholdOccurrences ThresholdOccurrence               `json:"thresholdOccurrences,omitempty"`
	Prediction           *NrqlConditionThresholdPrediction `json:"prediction,omitempty"`
}

// NrqlConditionQuery represents the NRQL query object returned in a NerdGraph response object.
type NrqlConditionQuery struct {
	Query            string `json:"query,omitempty"`
	DataAccountId    *int   `json:"dataAccountId,omitempty"`
	EvaluationOffset *int   `json:"evaluationOffset,omitempty"`
}

// NrqlConditionCreateQuery represents the NRQL query object for create.
type NrqlConditionCreateQuery struct {
	Query            string `json:"query,omitempty"`
	DataAccountId    *int   `json:"dataAccountId,omitempty"`
	EvaluationOffset *int   `json:"evaluationOffset,omitempty"`
}

// NrqlConditionUpdateQuery represents the NRQL query object for update.
type NrqlConditionUpdateQuery struct {
	Query            string `json:"query"`
	DataAccountId    *int   `json:"dataAccountId,omitempty"`
	EvaluationOffset *int   `json:"evaluationOffset"`
}

// NrqlConditionBase represents the base fields for a New Relic NRQL Alert condition.
type NrqlConditionBase struct {
	Description               string                          `json:"description,omitempty"`
	Enabled                   bool                            `json:"enabled"`
	Name                      string                          `json:"name,omitempty"`
	Nrql                      NrqlConditionQuery              `json:"nrql,omitempty"`
	RunbookURL                string                          `json:"runbookUrl,omitempty"`
	Terms                     []NrqlConditionTerm             `json:"terms,omitempty"`
	Type                      NrqlConditionType               `json:"type,omitempty"`
	ViolationTimeLimit        NrqlConditionViolationTimeLimit `json:"violationTimeLimit,omitempty"`
	ViolationTimeLimitSeconds int                             `json:"violationTimeLimitSeconds,omitempty"`
	Expiration                *AlertsNrqlConditionExpiration  `json:"expiration,omitempty"`
	Signal                    *AlertsNrqlConditionSignal      `json:"signal,omitempty"`
	EntityGUID                common.EntityGUID               `json:"entityGuid,omitempty"`
	TitleTemplate             *string                         `json:"titleTemplate,omitempty"`
}

// NrqlConditionCreateBase represents the base fields for creating a New Relic NRQL Alert condition.
type NrqlConditionCreateBase struct {
	Description               string                           `json:"description,omitempty"`
	Enabled                   bool                             `json:"enabled"`
	Name                      string                           `json:"name,omitempty"`
	Nrql                      NrqlConditionCreateQuery         `json:"nrql,omitempty"`
	RunbookURL                string                           `json:"runbookUrl,omitempty"`
	Terms                     []NrqlConditionTerm              `json:"terms,omitempty"`
	Type                      NrqlConditionType                `json:"type,omitempty"`
	ViolationTimeLimit        NrqlConditionViolationTimeLimit  `json:"violationTimeLimit,omitempty"`
	ViolationTimeLimitSeconds int                              `json:"violationTimeLimitSeconds,omitempty"`
	Expiration                *AlertsNrqlConditionExpiration   `json:"expiration,omitempty"`
	Signal                    *AlertsNrqlConditionCreateSignal `json:"signal,omitempty"`
	TitleTemplate             *string                          `json:"titleTemplate,omitempty"`
}

// NrqlConditionUpdateBase represents the base fields for updating a New Relic NRQL Alert condition.
type NrqlConditionUpdateBase struct {
	Description               string                           `json:"description,omitempty"`
	Enabled                   bool                             `json:"enabled"`
	Name                      string                           `json:"name,omitempty"`
	Nrql                      NrqlConditionUpdateQuery         `json:"nrql"`
	RunbookURL                string                           `json:"runbookUrl"`
	Terms                     []NrqlConditionTerm              `json:"terms,omitempty"`
	Type                      NrqlConditionType                `json:"type,omitempty"`
	ViolationTimeLimit        NrqlConditionViolationTimeLimit  `json:"violationTimeLimit,omitempty"`
	ViolationTimeLimitSeconds int                              `json:"violationTimeLimitSeconds,omitempty"`
	Expiration                *AlertsNrqlConditionExpiration   `json:"expiration,omitempty"`
	Signal                    *AlertsNrqlConditionUpdateSignal `json:"signal"`
	TitleTemplate             *string                          `json:"titleTemplate"`
}

// NrqlConditionCreateInput represents the input options for creating a Nrql Condition.
type NrqlConditionCreateInput struct {
	NrqlConditionCreateBase

	// BaselineDirection ONLY applies to NRQL conditions of type BASELINE.
	BaselineDirection *NrqlBaselineDirection `json:"baselineDirection,omitempty"`
	// SignalSeasonality ONLY applies to NRQL conditions of type BASELINE.
	SignalSeasonality *NrqlSignalSeasonality `json:"signalSeasonality,omitempty"`
}

// NrqlConditionUpdateInput represents the input options for updating a Nrql Condition.
type NrqlConditionUpdateInput struct {
	NrqlConditionUpdateBase

	// BaselineDirection ONLY applies to NRQL conditions of type BASELINE.
	BaselineDirection *NrqlBaselineDirection `json:"baselineDirection,omitempty"`
	// SignalSeasonality ONLY applies to NRQL conditions of type BASELINE.
	SignalSeasonality *NrqlSignalSeasonality `json:"signalSeasonality,omitempty"`
}

type NrqlConditionsSearchCriteria struct {
	Name      string `json:"name,omitempty"`
	NameLike  string `json:"nameLike,omitempty"`
	PolicyID  string `json:"policyId,omitempty"`
	Query     string `json:"query,omitempty"`
	QueryLike string `json:"queryLike,omitempty"`
}

// NrqlAlertCondition represents a NerdGraph NRQL alert condition, which is type AlertsNrqlCondition in NerdGraph.
// NrqlAlertCondition could be a baseline condition or static condition.
type NrqlAlertCondition struct {
	NrqlConditionBase
	ID       string `json:"id,omitempty"`
	PolicyID string `json:"policyId,omitempty"`

	// BaselineDirection exists ONLY for NRQL conditions of type BASELINE.
	BaselineDirection *NrqlBaselineDirection `json:"baselineDirection,omitempty"`
	// SignalSeasonality exists ONLY for NRQL conditions of type BASELINE.
	SignalSeasonality *NrqlSignalSeasonality `json:"signalSeasonality,omitempty"`
}

// NrqlCondition represents a New Relic NRQL Alert condition.
type NrqlCondition struct {
	Enabled             bool               `json:"enabled"`
	ID                  int                `json:"id,omitempty"`
	ViolationCloseTimer int                `json:"violation_time_limit_seconds,omitempty"`
	Name                string             `json:"name,omitempty"`
	Nrql                NrqlQuery          `json:"nrql,omitempty"`
	RunbookURL          string             `json:"runbook_url,omitempty"`
	Terms               []ConditionTerm    `json:"terms,omitempty"`
	Type                string             `json:"type,omitempty"`
	EntityGUID          *common.EntityGUID `json:"entity_guid,omitempty"`
	TitleTemplate       *string            `json:"titleTemplate,omitempty"`
}

// NrqlQuery represents a NRQL query to use with a NRQL alert condition
type NrqlQuery struct {
	Query      string `json:"query,omitempty"`
	SinceValue string `json:"since_value,omitempty"`
}

// ListNrqlConditions returns NRQL alert conditions for a specified policy.
func (a *Alerts) ListNrqlConditions(policyID int) ([]*NrqlCondition, error) {
	return a.ListNrqlConditionsWithContext(context.Background(), policyID)
}

// ListNrqlConditionsWithContext returns NRQL alert conditions for a specified policy.
func (a *Alerts) ListNrqlConditionsWithContext(ctx context.Context, policyID int) ([]*NrqlCondition, error) {
	conditions := []*NrqlCondition{}
	queryParams := listNrqlConditionsParams{
		PolicyID: policyID,
	}

	nextURL := a.config.Region().RestURL("/alerts_nrql_conditions.json")

	for nextURL != "" {
		response := nrqlConditionsResponse{}
		resp, err := a.client.GetWithContext(ctx, nextURL, &queryParams, &response)

		if err != nil {
			return nil, err
		}

		conditions = append(conditions, response.NrqlConditions...)

		paging := a.pager.Parse(resp)
		nextURL = paging.Next
	}

	return conditions, nil
}

// GetNrqlCondition gets information about a NRQL alert condition
// for a specified policy ID and condition ID.
func (a *Alerts) GetNrqlCondition(policyID int, id int) (*NrqlCondition, error) {
	return a.GetNrqlConditionWithContext(context.Background(), policyID, id)
}

// GetNrqlConditionWithContext gets information about a NRQL alert condition
// for a specified policy ID and condition ID.
func (a *Alerts) GetNrqlConditionWithContext(ctx context.Context, policyID int, id int) (*NrqlCondition, error) {
	conditions, err := a.ListNrqlConditionsWithContext(ctx, policyID)
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

// CreateNrqlCondition creates a NRQL alert condition.
func (a *Alerts) CreateNrqlCondition(policyID int, condition NrqlCondition) (*NrqlCondition, error) {
	return a.CreateNrqlConditionWithContext(context.Background(), policyID, condition)
}

// CreateNrqlConditionWithContext creates a NRQL alert condition.
func (a *Alerts) CreateNrqlConditionWithContext(ctx context.Context, policyID int, condition NrqlCondition) (*NrqlCondition, error) {
	reqBody := nrqlConditionRequestBody{
		NrqlCondition: condition,
	}
	resp := nrqlConditionResponse{}

	url := fmt.Sprintf("/alerts_nrql_conditions/policies/%d.json", policyID)
	_, err := a.client.PostWithContext(ctx, a.config.Region().RestURL(url), nil, &reqBody, &resp)

	if err != nil {
		return nil, err
	}

	return &resp.NrqlCondition, nil
}

// UpdateNrqlCondition updates a NRQL alert condition.
func (a *Alerts) UpdateNrqlCondition(condition NrqlCondition) (*NrqlCondition, error) {
	return a.UpdateNrqlConditionWithContext(context.Background(), condition)
}

// UpdateNrqlConditionWithContext updates a NRQL alert condition.
func (a *Alerts) UpdateNrqlConditionWithContext(ctx context.Context, condition NrqlCondition) (*NrqlCondition, error) {
	reqBody := nrqlConditionRequestBody{
		NrqlCondition: condition,
	}
	resp := nrqlConditionResponse{}

	url := fmt.Sprintf("/alerts_nrql_conditions/%d.json", condition.ID)
	_, err := a.client.PutWithContext(ctx, a.config.Region().RestURL(url), nil, &reqBody, &resp)

	if err != nil {
		return nil, err
	}

	return &resp.NrqlCondition, nil
}

// DeleteNrqlCondition deletes a NRQL alert condition.
func (a *Alerts) DeleteNrqlCondition(id int) (*NrqlCondition, error) {
	return a.DeleteNrqlConditionWithContext(context.Background(), id)
}

// DeleteNrqlConditionWithContext deletes a NRQL alert condition.
func (a *Alerts) DeleteNrqlConditionWithContext(ctx context.Context, id int) (*NrqlCondition, error) {
	resp := nrqlConditionResponse{}
	url := fmt.Sprintf("/alerts_nrql_conditions/%d.json", id)

	_, err := a.client.DeleteWithContext(ctx, a.config.Region().RestURL(url), nil, &resp)

	if err != nil {
		return nil, err
	}

	return &resp.NrqlCondition, nil
}

// GetNrqlConditionQuery fetches a NRQL alert condition via New Relic's NerdGraph API.
func (a *Alerts) GetNrqlConditionQuery(
	accountID int,
	conditionID string,
) (*NrqlAlertCondition, error) {
	return a.GetNrqlConditionQueryWithContext(context.Background(), accountID, conditionID)
}

// GetNrqlConditionQueryWithContext fetches a NRQL alert condition via New Relic's NerdGraph API.
func (a *Alerts) GetNrqlConditionQueryWithContext(
	ctx context.Context,
	accountID int,
	conditionID string,
) (*NrqlAlertCondition, error) {
	resp := getNrqlConditionQueryResponse{}
	vars := map[string]interface{}{
		"accountId": accountID,
		"id":        conditionID,
	}

	if err := a.NerdGraphQueryWithContext(ctx, getNrqlConditionQuery, vars, &resp); err != nil {
		return nil, err
	}

	return &resp.Actor.Account.Alerts.NrqlCondition, nil
}

// SearchNrqlConditionsQuery fetches multiple NRQL alert conditions based on the provided search criteria via New Relic's NerdGraph API.
func (a *Alerts) SearchNrqlConditionsQuery(
	accountID int,
	searchCriteria NrqlConditionsSearchCriteria,
) ([]*NrqlAlertCondition, error) {
	return a.SearchNrqlConditionsQueryWithContext(context.Background(), accountID, searchCriteria)
}

// SearchNrqlConditionsQueryWithContext fetches multiple NRQL alert conditions based on the provided search criteria via New Relic's NerdGraph API.
func (a *Alerts) SearchNrqlConditionsQueryWithContext(
	ctx context.Context,
	accountID int,
	searchCriteria NrqlConditionsSearchCriteria,
) ([]*NrqlAlertCondition, error) {
	conditions := []*NrqlAlertCondition{}
	var nextCursor *string

	for ok := true; ok; ok = nextCursor != nil {
		resp := searchNrqlConditionsResponse{}
		vars := map[string]interface{}{
			"accountId":      accountID,
			"searchCriteria": searchCriteria,
			"cursor":         nextCursor,
		}

		if err := a.NerdGraphQueryWithContext(ctx, searchNrqlConditionsQuery, vars, &resp); err != nil {
			return nil, err
		}

		conditions = append(conditions, resp.Actor.Account.Alerts.NrqlConditionsSearch.NrqlConditions...)
		nextCursor = resp.Actor.Account.Alerts.NrqlConditionsSearch.NextCursor
	}

	return conditions, nil
}

// CreateNrqlConditionBaselineMutation creates a baseline NRQL alert condition via New Relic's NerdGraph API.
func (a *Alerts) CreateNrqlConditionBaselineMutation(
	accountID int,
	policyID string,
	nrqlCondition NrqlConditionCreateInput,
) (*NrqlAlertCondition, error) {
	return a.CreateNrqlConditionBaselineMutationWithContext(context.Background(), accountID, policyID, nrqlCondition)
}

// CreateNrqlConditionBaselineMutationWithContext creates a baseline NRQL alert condition via New Relic's NerdGraph API.
func (a *Alerts) CreateNrqlConditionBaselineMutationWithContext(
	ctx context.Context,
	accountID int,
	policyID string,
	nrqlCondition NrqlConditionCreateInput,
) (*NrqlAlertCondition, error) {
	resp := nrqlConditionBaselineCreateResponse{}
	vars := map[string]interface{}{
		"accountId": accountID,
		"policyId":  policyID,
		"condition": nrqlCondition,
	}

	if err := a.NerdGraphQueryWithContext(ctx, createNrqlConditionBaselineMutation, vars, &resp); err != nil {
		return nil, err
	}

	return &resp.AlertsNrqlConditionBaselineCreate, nil
}

// UpdateNrqlConditionBaselineMutation updates a baseline NRQL alert condition via New Relic's NerdGraph API.
func (a *Alerts) UpdateNrqlConditionBaselineMutation(
	accountID int,
	conditionID string,
	nrqlCondition NrqlConditionUpdateInput,
) (*NrqlAlertCondition, error) {
	return a.UpdateNrqlConditionBaselineMutationWithContext(context.Background(), accountID, conditionID, nrqlCondition)
}

// UpdateNrqlConditionBaselineMutationWithContext updates a baseline NRQL alert condition via New Relic's NerdGraph API.
func (a *Alerts) UpdateNrqlConditionBaselineMutationWithContext(
	ctx context.Context,
	accountID int,
	conditionID string,
	nrqlCondition NrqlConditionUpdateInput,
) (*NrqlAlertCondition, error) {
	resp := nrqlConditionBaselineUpdateResponse{}
	vars := map[string]interface{}{
		"accountId": accountID,
		"id":        conditionID,
		"condition": nrqlCondition,
	}

	if err := a.NerdGraphQueryWithContext(ctx, updateNrqlConditionBaselineMutation, vars, &resp); err != nil {
		return nil, err
	}

	return &resp.AlertsNrqlConditionBaselineUpdate, nil
}

// CreateNrqlConditionStaticMutation creates a static NRQL alert condition via New Relic's NerdGraph API.
func (a *Alerts) CreateNrqlConditionStaticMutation(
	accountID int,
	policyID string,
	nrqlCondition NrqlConditionCreateInput,
) (*NrqlAlertCondition, error) {
	return a.CreateNrqlConditionStaticMutationWithContext(context.Background(), accountID, policyID, nrqlCondition)
}

// CreateNrqlConditionStaticMutationWithContext creates a static NRQL alert condition via New Relic's NerdGraph API.
func (a *Alerts) CreateNrqlConditionStaticMutationWithContext(
	ctx context.Context,
	accountID int,
	policyID string,
	nrqlCondition NrqlConditionCreateInput,
) (*NrqlAlertCondition, error) {
	resp := nrqlConditionStaticCreateResponse{}
	vars := map[string]interface{}{
		"accountId": accountID,
		"policyId":  policyID,
		"condition": nrqlCondition,
	}

	if err := a.NerdGraphQueryWithContext(ctx, createNrqlConditionStaticMutation, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.AlertsNrqlConditionStaticCreate, nil
}

// UpdateNrqlConditionStaticMutation updates a static NRQL alert condition via New Relic's NerdGraph API.
func (a *Alerts) UpdateNrqlConditionStaticMutation(
	accountID int,
	conditionID string,
	nrqlCondition NrqlConditionUpdateInput,
) (*NrqlAlertCondition, error) {
	return a.UpdateNrqlConditionStaticMutationWithContext(context.Background(), accountID, conditionID, nrqlCondition)
}

// UpdateNrqlConditionStaticMutationWithContext updates a static NRQL alert condition via New Relic's NerdGraph API.
func (a *Alerts) UpdateNrqlConditionStaticMutationWithContext(
	ctx context.Context,
	accountID int,
	conditionID string,
	nrqlCondition NrqlConditionUpdateInput,
) (*NrqlAlertCondition, error) {
	resp := nrqlConditionStaticUpdateResponse{}
	vars := map[string]interface{}{
		"accountId": accountID,
		"id":        conditionID,
		"condition": nrqlCondition,
	}

	if err := a.NerdGraphQueryWithContext(ctx, updateNrqlConditionStaticMutation, vars, &resp); err != nil {
		return nil, err
	}

	return &resp.AlertsNrqlConditionStaticUpdate, nil
}

func (a *Alerts) DeleteNrqlConditionMutation(
	accountID int,
	conditionID string,
) (string, error) {
	return a.DeleteNrqlConditionMutationWithContext(context.Background(), accountID, conditionID)
}

func (a *Alerts) DeleteNrqlConditionMutationWithContext(
	ctx context.Context,
	accountID int,
	conditionID string,
) (string, error) {
	result, err := a.DeleteConditionMutationWithContext(ctx, accountID, conditionID)
	if err != nil {
		return "", err
	}

	return result, nil
}

type listNrqlConditionsParams struct {
	PolicyID int `url:"policy_id,omitempty"`
}

type nrqlConditionsResponse struct {
	NrqlConditions []*NrqlCondition `json:"nrql_conditions,omitempty"`
}

type nrqlConditionResponse struct {
	NrqlCondition NrqlCondition `json:"nrql_condition,omitempty"`
}

type nrqlConditionRequestBody struct {
	NrqlCondition NrqlCondition `json:"nrql_condition,omitempty"`
}

type nrqlConditionBaselineCreateResponse struct {
	AlertsNrqlConditionBaselineCreate NrqlAlertCondition `json:"alertsNrqlConditionBaselineCreate"`
}

type nrqlConditionBaselineUpdateResponse struct {
	AlertsNrqlConditionBaselineUpdate NrqlAlertCondition `json:"alertsNrqlConditionBaselineUpdate"`
}

type nrqlConditionStaticCreateResponse struct {
	AlertsNrqlConditionStaticCreate NrqlAlertCondition `json:"alertsNrqlConditionStaticCreate"`
}

type nrqlConditionStaticUpdateResponse struct {
	AlertsNrqlConditionStaticUpdate NrqlAlertCondition `json:"alertsNrqlConditionStaticUpdate"`
}

type searchNrqlConditionsResponse struct {
	Actor struct {
		Account struct {
			Alerts struct {
				NrqlConditionsSearch struct {
					NextCursor     *string
					NrqlConditions []*NrqlAlertCondition `json:"nrqlConditions"`
				} `json:"nrqlConditionsSearch"`
			} `json:"alerts"`
		} `json:"account"`
	} `json:"actor"`
}

type getNrqlConditionQueryResponse struct {
	Actor struct {
		Account struct {
			Alerts struct {
				NrqlCondition NrqlAlertCondition `json:"nrqlCondition"`
			} `json:"alerts"`
		} `json:"account"`
	} `json:"actor"`
}

const (
	graphqlNrqlConditionStructFields = `
    id
    name
    nrql {
      evaluationOffset
      query
      dataAccountId
    }
    enabled
    entityGuid
    description
    titleTemplate
    policyId
    runbookUrl
    terms {
      operator
      priority
      threshold
      thresholdDuration
      thresholdOccurrences
    }
    type
    violationTimeLimit
    violationTimeLimitSeconds
    expiration {
      closeViolationsOnExpiration
      expirationDuration
      openViolationOnExpiration
      ignoreOnExpectedTermination
    }
    signal {
      aggregationWindow
      evaluationOffset
      evaluationDelay
      fillOption
      fillValue
      aggregationMethod
      aggregationDelay
      aggregationTimer
      slideBy
    }
  `

	graphqlFragmentNrqlBaselineConditionFields = `
		... on AlertsNrqlBaselineCondition {
			baselineDirection
			signalSeasonality
		}
	`

	graphqlFragmentNrqlStaticConditionFields = `
		... on AlertsNrqlStaticCondition {
			terms {
				prediction {
					predictBy
					preferPredictionViolation
				}
			}
		}
	`

	searchNrqlConditionsQuery = `
		query($accountId: Int!, $searchCriteria: AlertsNrqlConditionsSearchCriteriaInput, $cursor: String) {
			actor {
				account(id: $accountId) {
					alerts {
						nrqlConditionsSearch(searchCriteria: $searchCriteria, cursor: $cursor) {
							nextCursor
							totalCount
							nrqlConditions {` +
		graphqlNrqlConditionStructFields +
		graphqlFragmentNrqlBaselineConditionFields +
		graphqlFragmentNrqlStaticConditionFields +
		`} } } } } }`

	getNrqlConditionQuery = `
		query ($accountId: Int!, $id: ID!) {
			actor {
				account(id: $accountId) {
					alerts {
						nrqlCondition(id: $id) {` +
		graphqlNrqlConditionStructFields +
		graphqlFragmentNrqlBaselineConditionFields +
		graphqlFragmentNrqlStaticConditionFields +
		`} } } } }`

	// Baseline
	createNrqlConditionBaselineMutation = `
		mutation($accountId: Int!, $policyId: ID!, $condition: AlertsNrqlConditionBaselineInput!) {
			alertsNrqlConditionBaselineCreate(accountId: $accountId, policyId: $policyId, condition: $condition) {` +
		graphqlNrqlConditionStructFields +
		graphqlFragmentNrqlBaselineConditionFields +
		` } }`

	// Baseline
	updateNrqlConditionBaselineMutation = `
		mutation($accountId: Int!, $id: ID!, $condition: AlertsNrqlConditionUpdateBaselineInput!) {
			alertsNrqlConditionBaselineUpdate(accountId: $accountId, id: $id, condition: $condition) { ` +
		graphqlNrqlConditionStructFields +
		graphqlFragmentNrqlBaselineConditionFields +
		` } }`

	// Static
	createNrqlConditionStaticMutation = `
		mutation($accountId: Int!, $policyId: ID!, $condition: AlertsNrqlConditionStaticInput!) {
			alertsNrqlConditionStaticCreate(accountId: $accountId, policyId: $policyId, condition: $condition) {` +
		graphqlNrqlConditionStructFields +
		graphqlFragmentNrqlStaticConditionFields +
		` } }`

	// Static
	updateNrqlConditionStaticMutation = `
		mutation($accountId: Int!, $id: ID!, $condition: AlertsNrqlConditionUpdateStaticInput!) {
			alertsNrqlConditionStaticUpdate(accountId: $accountId, id: $id, condition: $condition) {` +
		graphqlNrqlConditionStructFields +
		graphqlFragmentNrqlStaticConditionFields +
		` } }`
)
