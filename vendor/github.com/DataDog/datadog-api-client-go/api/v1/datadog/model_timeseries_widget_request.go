// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// TimeseriesWidgetRequest Updated timeseries widget.
type TimeseriesWidgetRequest struct {
	// The log query.
	ApmQuery *LogQueryDefinition `json:"apm_query,omitempty"`
	// The log query.
	AuditQuery *LogQueryDefinition `json:"audit_query,omitempty"`
	// Type of display to use for the request.
	DisplayType *WidgetDisplayType `json:"display_type,omitempty"`
	// The log query.
	EventQuery *LogQueryDefinition `json:"event_query,omitempty"`
	// List of formulas that operate on queries.
	Formulas []WidgetFormula `json:"formulas,omitempty"`
	// The log query.
	LogQuery *LogQueryDefinition `json:"log_query,omitempty"`
	// Used to define expression aliases.
	Metadata []TimeseriesWidgetExpressionAlias `json:"metadata,omitempty"`
	// The log query.
	NetworkQuery *LogQueryDefinition `json:"network_query,omitempty"`
	// Whether or not to display a second y-axis on the right.
	OnRightYaxis *bool `json:"on_right_yaxis,omitempty"`
	// The process query to use in the widget.
	ProcessQuery *ProcessQueryDefinition `json:"process_query,omitempty"`
	// The log query.
	ProfileMetricsQuery *LogQueryDefinition `json:"profile_metrics_query,omitempty"`
	// Widget query.
	Q *string `json:"q,omitempty"`
	// List of queries that can be returned directly or used in formulas.
	Queries []FormulaAndFunctionQueryDefinition `json:"queries,omitempty"`
	// Timeseries or Scalar response.
	ResponseFormat *FormulaAndFunctionResponseFormat `json:"response_format,omitempty"`
	// The log query.
	RumQuery *LogQueryDefinition `json:"rum_query,omitempty"`
	// The log query.
	SecurityQuery *LogQueryDefinition `json:"security_query,omitempty"`
	// Define request widget style.
	Style *WidgetRequestStyle `json:"style,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewTimeseriesWidgetRequest instantiates a new TimeseriesWidgetRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewTimeseriesWidgetRequest() *TimeseriesWidgetRequest {
	this := TimeseriesWidgetRequest{}
	return &this
}

// NewTimeseriesWidgetRequestWithDefaults instantiates a new TimeseriesWidgetRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewTimeseriesWidgetRequestWithDefaults() *TimeseriesWidgetRequest {
	this := TimeseriesWidgetRequest{}
	return &this
}

// GetApmQuery returns the ApmQuery field value if set, zero value otherwise.
func (o *TimeseriesWidgetRequest) GetApmQuery() LogQueryDefinition {
	if o == nil || o.ApmQuery == nil {
		var ret LogQueryDefinition
		return ret
	}
	return *o.ApmQuery
}

// GetApmQueryOk returns a tuple with the ApmQuery field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TimeseriesWidgetRequest) GetApmQueryOk() (*LogQueryDefinition, bool) {
	if o == nil || o.ApmQuery == nil {
		return nil, false
	}
	return o.ApmQuery, true
}

// HasApmQuery returns a boolean if a field has been set.
func (o *TimeseriesWidgetRequest) HasApmQuery() bool {
	if o != nil && o.ApmQuery != nil {
		return true
	}

	return false
}

// SetApmQuery gets a reference to the given LogQueryDefinition and assigns it to the ApmQuery field.
func (o *TimeseriesWidgetRequest) SetApmQuery(v LogQueryDefinition) {
	o.ApmQuery = &v
}

// GetAuditQuery returns the AuditQuery field value if set, zero value otherwise.
func (o *TimeseriesWidgetRequest) GetAuditQuery() LogQueryDefinition {
	if o == nil || o.AuditQuery == nil {
		var ret LogQueryDefinition
		return ret
	}
	return *o.AuditQuery
}

// GetAuditQueryOk returns a tuple with the AuditQuery field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TimeseriesWidgetRequest) GetAuditQueryOk() (*LogQueryDefinition, bool) {
	if o == nil || o.AuditQuery == nil {
		return nil, false
	}
	return o.AuditQuery, true
}

// HasAuditQuery returns a boolean if a field has been set.
func (o *TimeseriesWidgetRequest) HasAuditQuery() bool {
	if o != nil && o.AuditQuery != nil {
		return true
	}

	return false
}

// SetAuditQuery gets a reference to the given LogQueryDefinition and assigns it to the AuditQuery field.
func (o *TimeseriesWidgetRequest) SetAuditQuery(v LogQueryDefinition) {
	o.AuditQuery = &v
}

// GetDisplayType returns the DisplayType field value if set, zero value otherwise.
func (o *TimeseriesWidgetRequest) GetDisplayType() WidgetDisplayType {
	if o == nil || o.DisplayType == nil {
		var ret WidgetDisplayType
		return ret
	}
	return *o.DisplayType
}

// GetDisplayTypeOk returns a tuple with the DisplayType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TimeseriesWidgetRequest) GetDisplayTypeOk() (*WidgetDisplayType, bool) {
	if o == nil || o.DisplayType == nil {
		return nil, false
	}
	return o.DisplayType, true
}

// HasDisplayType returns a boolean if a field has been set.
func (o *TimeseriesWidgetRequest) HasDisplayType() bool {
	if o != nil && o.DisplayType != nil {
		return true
	}

	return false
}

// SetDisplayType gets a reference to the given WidgetDisplayType and assigns it to the DisplayType field.
func (o *TimeseriesWidgetRequest) SetDisplayType(v WidgetDisplayType) {
	o.DisplayType = &v
}

// GetEventQuery returns the EventQuery field value if set, zero value otherwise.
func (o *TimeseriesWidgetRequest) GetEventQuery() LogQueryDefinition {
	if o == nil || o.EventQuery == nil {
		var ret LogQueryDefinition
		return ret
	}
	return *o.EventQuery
}

// GetEventQueryOk returns a tuple with the EventQuery field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TimeseriesWidgetRequest) GetEventQueryOk() (*LogQueryDefinition, bool) {
	if o == nil || o.EventQuery == nil {
		return nil, false
	}
	return o.EventQuery, true
}

// HasEventQuery returns a boolean if a field has been set.
func (o *TimeseriesWidgetRequest) HasEventQuery() bool {
	if o != nil && o.EventQuery != nil {
		return true
	}

	return false
}

// SetEventQuery gets a reference to the given LogQueryDefinition and assigns it to the EventQuery field.
func (o *TimeseriesWidgetRequest) SetEventQuery(v LogQueryDefinition) {
	o.EventQuery = &v
}

// GetFormulas returns the Formulas field value if set, zero value otherwise.
func (o *TimeseriesWidgetRequest) GetFormulas() []WidgetFormula {
	if o == nil || o.Formulas == nil {
		var ret []WidgetFormula
		return ret
	}
	return o.Formulas
}

// GetFormulasOk returns a tuple with the Formulas field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TimeseriesWidgetRequest) GetFormulasOk() (*[]WidgetFormula, bool) {
	if o == nil || o.Formulas == nil {
		return nil, false
	}
	return &o.Formulas, true
}

// HasFormulas returns a boolean if a field has been set.
func (o *TimeseriesWidgetRequest) HasFormulas() bool {
	if o != nil && o.Formulas != nil {
		return true
	}

	return false
}

// SetFormulas gets a reference to the given []WidgetFormula and assigns it to the Formulas field.
func (o *TimeseriesWidgetRequest) SetFormulas(v []WidgetFormula) {
	o.Formulas = v
}

// GetLogQuery returns the LogQuery field value if set, zero value otherwise.
func (o *TimeseriesWidgetRequest) GetLogQuery() LogQueryDefinition {
	if o == nil || o.LogQuery == nil {
		var ret LogQueryDefinition
		return ret
	}
	return *o.LogQuery
}

// GetLogQueryOk returns a tuple with the LogQuery field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TimeseriesWidgetRequest) GetLogQueryOk() (*LogQueryDefinition, bool) {
	if o == nil || o.LogQuery == nil {
		return nil, false
	}
	return o.LogQuery, true
}

// HasLogQuery returns a boolean if a field has been set.
func (o *TimeseriesWidgetRequest) HasLogQuery() bool {
	if o != nil && o.LogQuery != nil {
		return true
	}

	return false
}

// SetLogQuery gets a reference to the given LogQueryDefinition and assigns it to the LogQuery field.
func (o *TimeseriesWidgetRequest) SetLogQuery(v LogQueryDefinition) {
	o.LogQuery = &v
}

// GetMetadata returns the Metadata field value if set, zero value otherwise.
func (o *TimeseriesWidgetRequest) GetMetadata() []TimeseriesWidgetExpressionAlias {
	if o == nil || o.Metadata == nil {
		var ret []TimeseriesWidgetExpressionAlias
		return ret
	}
	return o.Metadata
}

// GetMetadataOk returns a tuple with the Metadata field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TimeseriesWidgetRequest) GetMetadataOk() (*[]TimeseriesWidgetExpressionAlias, bool) {
	if o == nil || o.Metadata == nil {
		return nil, false
	}
	return &o.Metadata, true
}

// HasMetadata returns a boolean if a field has been set.
func (o *TimeseriesWidgetRequest) HasMetadata() bool {
	if o != nil && o.Metadata != nil {
		return true
	}

	return false
}

// SetMetadata gets a reference to the given []TimeseriesWidgetExpressionAlias and assigns it to the Metadata field.
func (o *TimeseriesWidgetRequest) SetMetadata(v []TimeseriesWidgetExpressionAlias) {
	o.Metadata = v
}

// GetNetworkQuery returns the NetworkQuery field value if set, zero value otherwise.
func (o *TimeseriesWidgetRequest) GetNetworkQuery() LogQueryDefinition {
	if o == nil || o.NetworkQuery == nil {
		var ret LogQueryDefinition
		return ret
	}
	return *o.NetworkQuery
}

// GetNetworkQueryOk returns a tuple with the NetworkQuery field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TimeseriesWidgetRequest) GetNetworkQueryOk() (*LogQueryDefinition, bool) {
	if o == nil || o.NetworkQuery == nil {
		return nil, false
	}
	return o.NetworkQuery, true
}

// HasNetworkQuery returns a boolean if a field has been set.
func (o *TimeseriesWidgetRequest) HasNetworkQuery() bool {
	if o != nil && o.NetworkQuery != nil {
		return true
	}

	return false
}

// SetNetworkQuery gets a reference to the given LogQueryDefinition and assigns it to the NetworkQuery field.
func (o *TimeseriesWidgetRequest) SetNetworkQuery(v LogQueryDefinition) {
	o.NetworkQuery = &v
}

// GetOnRightYaxis returns the OnRightYaxis field value if set, zero value otherwise.
func (o *TimeseriesWidgetRequest) GetOnRightYaxis() bool {
	if o == nil || o.OnRightYaxis == nil {
		var ret bool
		return ret
	}
	return *o.OnRightYaxis
}

// GetOnRightYaxisOk returns a tuple with the OnRightYaxis field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TimeseriesWidgetRequest) GetOnRightYaxisOk() (*bool, bool) {
	if o == nil || o.OnRightYaxis == nil {
		return nil, false
	}
	return o.OnRightYaxis, true
}

// HasOnRightYaxis returns a boolean if a field has been set.
func (o *TimeseriesWidgetRequest) HasOnRightYaxis() bool {
	if o != nil && o.OnRightYaxis != nil {
		return true
	}

	return false
}

// SetOnRightYaxis gets a reference to the given bool and assigns it to the OnRightYaxis field.
func (o *TimeseriesWidgetRequest) SetOnRightYaxis(v bool) {
	o.OnRightYaxis = &v
}

// GetProcessQuery returns the ProcessQuery field value if set, zero value otherwise.
func (o *TimeseriesWidgetRequest) GetProcessQuery() ProcessQueryDefinition {
	if o == nil || o.ProcessQuery == nil {
		var ret ProcessQueryDefinition
		return ret
	}
	return *o.ProcessQuery
}

// GetProcessQueryOk returns a tuple with the ProcessQuery field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TimeseriesWidgetRequest) GetProcessQueryOk() (*ProcessQueryDefinition, bool) {
	if o == nil || o.ProcessQuery == nil {
		return nil, false
	}
	return o.ProcessQuery, true
}

// HasProcessQuery returns a boolean if a field has been set.
func (o *TimeseriesWidgetRequest) HasProcessQuery() bool {
	if o != nil && o.ProcessQuery != nil {
		return true
	}

	return false
}

// SetProcessQuery gets a reference to the given ProcessQueryDefinition and assigns it to the ProcessQuery field.
func (o *TimeseriesWidgetRequest) SetProcessQuery(v ProcessQueryDefinition) {
	o.ProcessQuery = &v
}

// GetProfileMetricsQuery returns the ProfileMetricsQuery field value if set, zero value otherwise.
func (o *TimeseriesWidgetRequest) GetProfileMetricsQuery() LogQueryDefinition {
	if o == nil || o.ProfileMetricsQuery == nil {
		var ret LogQueryDefinition
		return ret
	}
	return *o.ProfileMetricsQuery
}

// GetProfileMetricsQueryOk returns a tuple with the ProfileMetricsQuery field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TimeseriesWidgetRequest) GetProfileMetricsQueryOk() (*LogQueryDefinition, bool) {
	if o == nil || o.ProfileMetricsQuery == nil {
		return nil, false
	}
	return o.ProfileMetricsQuery, true
}

// HasProfileMetricsQuery returns a boolean if a field has been set.
func (o *TimeseriesWidgetRequest) HasProfileMetricsQuery() bool {
	if o != nil && o.ProfileMetricsQuery != nil {
		return true
	}

	return false
}

// SetProfileMetricsQuery gets a reference to the given LogQueryDefinition and assigns it to the ProfileMetricsQuery field.
func (o *TimeseriesWidgetRequest) SetProfileMetricsQuery(v LogQueryDefinition) {
	o.ProfileMetricsQuery = &v
}

// GetQ returns the Q field value if set, zero value otherwise.
func (o *TimeseriesWidgetRequest) GetQ() string {
	if o == nil || o.Q == nil {
		var ret string
		return ret
	}
	return *o.Q
}

// GetQOk returns a tuple with the Q field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TimeseriesWidgetRequest) GetQOk() (*string, bool) {
	if o == nil || o.Q == nil {
		return nil, false
	}
	return o.Q, true
}

// HasQ returns a boolean if a field has been set.
func (o *TimeseriesWidgetRequest) HasQ() bool {
	if o != nil && o.Q != nil {
		return true
	}

	return false
}

// SetQ gets a reference to the given string and assigns it to the Q field.
func (o *TimeseriesWidgetRequest) SetQ(v string) {
	o.Q = &v
}

// GetQueries returns the Queries field value if set, zero value otherwise.
func (o *TimeseriesWidgetRequest) GetQueries() []FormulaAndFunctionQueryDefinition {
	if o == nil || o.Queries == nil {
		var ret []FormulaAndFunctionQueryDefinition
		return ret
	}
	return o.Queries
}

// GetQueriesOk returns a tuple with the Queries field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TimeseriesWidgetRequest) GetQueriesOk() (*[]FormulaAndFunctionQueryDefinition, bool) {
	if o == nil || o.Queries == nil {
		return nil, false
	}
	return &o.Queries, true
}

// HasQueries returns a boolean if a field has been set.
func (o *TimeseriesWidgetRequest) HasQueries() bool {
	if o != nil && o.Queries != nil {
		return true
	}

	return false
}

// SetQueries gets a reference to the given []FormulaAndFunctionQueryDefinition and assigns it to the Queries field.
func (o *TimeseriesWidgetRequest) SetQueries(v []FormulaAndFunctionQueryDefinition) {
	o.Queries = v
}

// GetResponseFormat returns the ResponseFormat field value if set, zero value otherwise.
func (o *TimeseriesWidgetRequest) GetResponseFormat() FormulaAndFunctionResponseFormat {
	if o == nil || o.ResponseFormat == nil {
		var ret FormulaAndFunctionResponseFormat
		return ret
	}
	return *o.ResponseFormat
}

// GetResponseFormatOk returns a tuple with the ResponseFormat field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TimeseriesWidgetRequest) GetResponseFormatOk() (*FormulaAndFunctionResponseFormat, bool) {
	if o == nil || o.ResponseFormat == nil {
		return nil, false
	}
	return o.ResponseFormat, true
}

// HasResponseFormat returns a boolean if a field has been set.
func (o *TimeseriesWidgetRequest) HasResponseFormat() bool {
	if o != nil && o.ResponseFormat != nil {
		return true
	}

	return false
}

// SetResponseFormat gets a reference to the given FormulaAndFunctionResponseFormat and assigns it to the ResponseFormat field.
func (o *TimeseriesWidgetRequest) SetResponseFormat(v FormulaAndFunctionResponseFormat) {
	o.ResponseFormat = &v
}

// GetRumQuery returns the RumQuery field value if set, zero value otherwise.
func (o *TimeseriesWidgetRequest) GetRumQuery() LogQueryDefinition {
	if o == nil || o.RumQuery == nil {
		var ret LogQueryDefinition
		return ret
	}
	return *o.RumQuery
}

// GetRumQueryOk returns a tuple with the RumQuery field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TimeseriesWidgetRequest) GetRumQueryOk() (*LogQueryDefinition, bool) {
	if o == nil || o.RumQuery == nil {
		return nil, false
	}
	return o.RumQuery, true
}

// HasRumQuery returns a boolean if a field has been set.
func (o *TimeseriesWidgetRequest) HasRumQuery() bool {
	if o != nil && o.RumQuery != nil {
		return true
	}

	return false
}

// SetRumQuery gets a reference to the given LogQueryDefinition and assigns it to the RumQuery field.
func (o *TimeseriesWidgetRequest) SetRumQuery(v LogQueryDefinition) {
	o.RumQuery = &v
}

// GetSecurityQuery returns the SecurityQuery field value if set, zero value otherwise.
func (o *TimeseriesWidgetRequest) GetSecurityQuery() LogQueryDefinition {
	if o == nil || o.SecurityQuery == nil {
		var ret LogQueryDefinition
		return ret
	}
	return *o.SecurityQuery
}

// GetSecurityQueryOk returns a tuple with the SecurityQuery field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TimeseriesWidgetRequest) GetSecurityQueryOk() (*LogQueryDefinition, bool) {
	if o == nil || o.SecurityQuery == nil {
		return nil, false
	}
	return o.SecurityQuery, true
}

// HasSecurityQuery returns a boolean if a field has been set.
func (o *TimeseriesWidgetRequest) HasSecurityQuery() bool {
	if o != nil && o.SecurityQuery != nil {
		return true
	}

	return false
}

// SetSecurityQuery gets a reference to the given LogQueryDefinition and assigns it to the SecurityQuery field.
func (o *TimeseriesWidgetRequest) SetSecurityQuery(v LogQueryDefinition) {
	o.SecurityQuery = &v
}

// GetStyle returns the Style field value if set, zero value otherwise.
func (o *TimeseriesWidgetRequest) GetStyle() WidgetRequestStyle {
	if o == nil || o.Style == nil {
		var ret WidgetRequestStyle
		return ret
	}
	return *o.Style
}

// GetStyleOk returns a tuple with the Style field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TimeseriesWidgetRequest) GetStyleOk() (*WidgetRequestStyle, bool) {
	if o == nil || o.Style == nil {
		return nil, false
	}
	return o.Style, true
}

// HasStyle returns a boolean if a field has been set.
func (o *TimeseriesWidgetRequest) HasStyle() bool {
	if o != nil && o.Style != nil {
		return true
	}

	return false
}

// SetStyle gets a reference to the given WidgetRequestStyle and assigns it to the Style field.
func (o *TimeseriesWidgetRequest) SetStyle(v WidgetRequestStyle) {
	o.Style = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o TimeseriesWidgetRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.ApmQuery != nil {
		toSerialize["apm_query"] = o.ApmQuery
	}
	if o.AuditQuery != nil {
		toSerialize["audit_query"] = o.AuditQuery
	}
	if o.DisplayType != nil {
		toSerialize["display_type"] = o.DisplayType
	}
	if o.EventQuery != nil {
		toSerialize["event_query"] = o.EventQuery
	}
	if o.Formulas != nil {
		toSerialize["formulas"] = o.Formulas
	}
	if o.LogQuery != nil {
		toSerialize["log_query"] = o.LogQuery
	}
	if o.Metadata != nil {
		toSerialize["metadata"] = o.Metadata
	}
	if o.NetworkQuery != nil {
		toSerialize["network_query"] = o.NetworkQuery
	}
	if o.OnRightYaxis != nil {
		toSerialize["on_right_yaxis"] = o.OnRightYaxis
	}
	if o.ProcessQuery != nil {
		toSerialize["process_query"] = o.ProcessQuery
	}
	if o.ProfileMetricsQuery != nil {
		toSerialize["profile_metrics_query"] = o.ProfileMetricsQuery
	}
	if o.Q != nil {
		toSerialize["q"] = o.Q
	}
	if o.Queries != nil {
		toSerialize["queries"] = o.Queries
	}
	if o.ResponseFormat != nil {
		toSerialize["response_format"] = o.ResponseFormat
	}
	if o.RumQuery != nil {
		toSerialize["rum_query"] = o.RumQuery
	}
	if o.SecurityQuery != nil {
		toSerialize["security_query"] = o.SecurityQuery
	}
	if o.Style != nil {
		toSerialize["style"] = o.Style
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *TimeseriesWidgetRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		ApmQuery            *LogQueryDefinition                 `json:"apm_query,omitempty"`
		AuditQuery          *LogQueryDefinition                 `json:"audit_query,omitempty"`
		DisplayType         *WidgetDisplayType                  `json:"display_type,omitempty"`
		EventQuery          *LogQueryDefinition                 `json:"event_query,omitempty"`
		Formulas            []WidgetFormula                     `json:"formulas,omitempty"`
		LogQuery            *LogQueryDefinition                 `json:"log_query,omitempty"`
		Metadata            []TimeseriesWidgetExpressionAlias   `json:"metadata,omitempty"`
		NetworkQuery        *LogQueryDefinition                 `json:"network_query,omitempty"`
		OnRightYaxis        *bool                               `json:"on_right_yaxis,omitempty"`
		ProcessQuery        *ProcessQueryDefinition             `json:"process_query,omitempty"`
		ProfileMetricsQuery *LogQueryDefinition                 `json:"profile_metrics_query,omitempty"`
		Q                   *string                             `json:"q,omitempty"`
		Queries             []FormulaAndFunctionQueryDefinition `json:"queries,omitempty"`
		ResponseFormat      *FormulaAndFunctionResponseFormat   `json:"response_format,omitempty"`
		RumQuery            *LogQueryDefinition                 `json:"rum_query,omitempty"`
		SecurityQuery       *LogQueryDefinition                 `json:"security_query,omitempty"`
		Style               *WidgetRequestStyle                 `json:"style,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &all)
	if err != nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.DisplayType; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.ResponseFormat; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if all.ApmQuery != nil && all.ApmQuery.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ApmQuery = all.ApmQuery
	if all.AuditQuery != nil && all.AuditQuery.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.AuditQuery = all.AuditQuery
	o.DisplayType = all.DisplayType
	if all.EventQuery != nil && all.EventQuery.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.EventQuery = all.EventQuery
	o.Formulas = all.Formulas
	if all.LogQuery != nil && all.LogQuery.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.LogQuery = all.LogQuery
	o.Metadata = all.Metadata
	if all.NetworkQuery != nil && all.NetworkQuery.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.NetworkQuery = all.NetworkQuery
	o.OnRightYaxis = all.OnRightYaxis
	if all.ProcessQuery != nil && all.ProcessQuery.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ProcessQuery = all.ProcessQuery
	if all.ProfileMetricsQuery != nil && all.ProfileMetricsQuery.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ProfileMetricsQuery = all.ProfileMetricsQuery
	o.Q = all.Q
	o.Queries = all.Queries
	o.ResponseFormat = all.ResponseFormat
	if all.RumQuery != nil && all.RumQuery.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.RumQuery = all.RumQuery
	if all.SecurityQuery != nil && all.SecurityQuery.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.SecurityQuery = all.SecurityQuery
	if all.Style != nil && all.Style.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Style = all.Style
	return nil
}
