// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SLOHistoryMonitor An object that holds an SLI value and its associated data. It can represent an SLO's overall SLI value.
// This can also represent the SLI value for a specific monitor in multi-monitor SLOs, or a group in grouped SLOs.
type SLOHistoryMonitor struct {
	// A mapping of threshold `timeframe` to the remaining error budget.
	ErrorBudgetRemaining map[string]float64 `json:"error_budget_remaining,omitempty"`
	// An array of error objects returned while querying the history data for the service level objective.
	Errors []SLOHistoryResponseErrorWithType `json:"errors,omitempty"`
	// For groups in a grouped SLO, this is the group name.
	Group *string `json:"group,omitempty"`
	// For `monitor` based SLOs, this includes the aggregated history as arrays that include time series and uptime data where `0=monitor` is in `OK` state and `1=monitor` is in `alert` state.
	History [][]float64 `json:"history,omitempty"`
	// For `monitor` based SLOs, this is the last modified timestamp in epoch seconds of the monitor.
	MonitorModified *int64 `json:"monitor_modified,omitempty"`
	// For `monitor` based SLOs, this describes the type of monitor.
	MonitorType *string `json:"monitor_type,omitempty"`
	// For groups in a grouped SLO, this is the group name. For monitors in a multi-monitor SLO, this is the monitor name.
	Name *string `json:"name,omitempty"`
	// The amount of decimal places the SLI value is accurate to for the given from `&&` to timestamp. Use `span_precision` instead.
	// Deprecated
	Precision *float64 `json:"precision,omitempty"`
	// For `monitor` based SLOs, when `true` this indicates that a replay is in progress to give an accurate uptime
	// calculation.
	Preview *bool `json:"preview,omitempty"`
	// The current SLI value of the SLO over the history window.
	SliValue *float64 `json:"sli_value,omitempty"`
	// The amount of decimal places the SLI value is accurate to for the given from `&&` to timestamp.
	SpanPrecision *float64 `json:"span_precision,omitempty"`
	// Use `sli_value` instead.
	// Deprecated
	Uptime *float64 `json:"uptime,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSLOHistoryMonitor instantiates a new SLOHistoryMonitor object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSLOHistoryMonitor() *SLOHistoryMonitor {
	this := SLOHistoryMonitor{}
	return &this
}

// NewSLOHistoryMonitorWithDefaults instantiates a new SLOHistoryMonitor object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSLOHistoryMonitorWithDefaults() *SLOHistoryMonitor {
	this := SLOHistoryMonitor{}
	return &this
}

// GetErrorBudgetRemaining returns the ErrorBudgetRemaining field value if set, zero value otherwise.
func (o *SLOHistoryMonitor) GetErrorBudgetRemaining() map[string]float64 {
	if o == nil || o.ErrorBudgetRemaining == nil {
		var ret map[string]float64
		return ret
	}
	return o.ErrorBudgetRemaining
}

// GetErrorBudgetRemainingOk returns a tuple with the ErrorBudgetRemaining field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryMonitor) GetErrorBudgetRemainingOk() (*map[string]float64, bool) {
	if o == nil || o.ErrorBudgetRemaining == nil {
		return nil, false
	}
	return &o.ErrorBudgetRemaining, true
}

// HasErrorBudgetRemaining returns a boolean if a field has been set.
func (o *SLOHistoryMonitor) HasErrorBudgetRemaining() bool {
	if o != nil && o.ErrorBudgetRemaining != nil {
		return true
	}

	return false
}

// SetErrorBudgetRemaining gets a reference to the given map[string]float64 and assigns it to the ErrorBudgetRemaining field.
func (o *SLOHistoryMonitor) SetErrorBudgetRemaining(v map[string]float64) {
	o.ErrorBudgetRemaining = v
}

// GetErrors returns the Errors field value if set, zero value otherwise.
func (o *SLOHistoryMonitor) GetErrors() []SLOHistoryResponseErrorWithType {
	if o == nil || o.Errors == nil {
		var ret []SLOHistoryResponseErrorWithType
		return ret
	}
	return o.Errors
}

// GetErrorsOk returns a tuple with the Errors field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryMonitor) GetErrorsOk() (*[]SLOHistoryResponseErrorWithType, bool) {
	if o == nil || o.Errors == nil {
		return nil, false
	}
	return &o.Errors, true
}

// HasErrors returns a boolean if a field has been set.
func (o *SLOHistoryMonitor) HasErrors() bool {
	if o != nil && o.Errors != nil {
		return true
	}

	return false
}

// SetErrors gets a reference to the given []SLOHistoryResponseErrorWithType and assigns it to the Errors field.
func (o *SLOHistoryMonitor) SetErrors(v []SLOHistoryResponseErrorWithType) {
	o.Errors = v
}

// GetGroup returns the Group field value if set, zero value otherwise.
func (o *SLOHistoryMonitor) GetGroup() string {
	if o == nil || o.Group == nil {
		var ret string
		return ret
	}
	return *o.Group
}

// GetGroupOk returns a tuple with the Group field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryMonitor) GetGroupOk() (*string, bool) {
	if o == nil || o.Group == nil {
		return nil, false
	}
	return o.Group, true
}

// HasGroup returns a boolean if a field has been set.
func (o *SLOHistoryMonitor) HasGroup() bool {
	if o != nil && o.Group != nil {
		return true
	}

	return false
}

// SetGroup gets a reference to the given string and assigns it to the Group field.
func (o *SLOHistoryMonitor) SetGroup(v string) {
	o.Group = &v
}

// GetHistory returns the History field value if set, zero value otherwise.
func (o *SLOHistoryMonitor) GetHistory() [][]float64 {
	if o == nil || o.History == nil {
		var ret [][]float64
		return ret
	}
	return o.History
}

// GetHistoryOk returns a tuple with the History field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryMonitor) GetHistoryOk() (*[][]float64, bool) {
	if o == nil || o.History == nil {
		return nil, false
	}
	return &o.History, true
}

// HasHistory returns a boolean if a field has been set.
func (o *SLOHistoryMonitor) HasHistory() bool {
	if o != nil && o.History != nil {
		return true
	}

	return false
}

// SetHistory gets a reference to the given [][]float64 and assigns it to the History field.
func (o *SLOHistoryMonitor) SetHistory(v [][]float64) {
	o.History = v
}

// GetMonitorModified returns the MonitorModified field value if set, zero value otherwise.
func (o *SLOHistoryMonitor) GetMonitorModified() int64 {
	if o == nil || o.MonitorModified == nil {
		var ret int64
		return ret
	}
	return *o.MonitorModified
}

// GetMonitorModifiedOk returns a tuple with the MonitorModified field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryMonitor) GetMonitorModifiedOk() (*int64, bool) {
	if o == nil || o.MonitorModified == nil {
		return nil, false
	}
	return o.MonitorModified, true
}

// HasMonitorModified returns a boolean if a field has been set.
func (o *SLOHistoryMonitor) HasMonitorModified() bool {
	if o != nil && o.MonitorModified != nil {
		return true
	}

	return false
}

// SetMonitorModified gets a reference to the given int64 and assigns it to the MonitorModified field.
func (o *SLOHistoryMonitor) SetMonitorModified(v int64) {
	o.MonitorModified = &v
}

// GetMonitorType returns the MonitorType field value if set, zero value otherwise.
func (o *SLOHistoryMonitor) GetMonitorType() string {
	if o == nil || o.MonitorType == nil {
		var ret string
		return ret
	}
	return *o.MonitorType
}

// GetMonitorTypeOk returns a tuple with the MonitorType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryMonitor) GetMonitorTypeOk() (*string, bool) {
	if o == nil || o.MonitorType == nil {
		return nil, false
	}
	return o.MonitorType, true
}

// HasMonitorType returns a boolean if a field has been set.
func (o *SLOHistoryMonitor) HasMonitorType() bool {
	if o != nil && o.MonitorType != nil {
		return true
	}

	return false
}

// SetMonitorType gets a reference to the given string and assigns it to the MonitorType field.
func (o *SLOHistoryMonitor) SetMonitorType(v string) {
	o.MonitorType = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *SLOHistoryMonitor) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryMonitor) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *SLOHistoryMonitor) HasName() bool {
	if o != nil && o.Name != nil {
		return true
	}

	return false
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *SLOHistoryMonitor) SetName(v string) {
	o.Name = &v
}

// GetPrecision returns the Precision field value if set, zero value otherwise.
// Deprecated
func (o *SLOHistoryMonitor) GetPrecision() float64 {
	if o == nil || o.Precision == nil {
		var ret float64
		return ret
	}
	return *o.Precision
}

// GetPrecisionOk returns a tuple with the Precision field value if set, nil otherwise
// and a boolean to check if the value has been set.
// Deprecated
func (o *SLOHistoryMonitor) GetPrecisionOk() (*float64, bool) {
	if o == nil || o.Precision == nil {
		return nil, false
	}
	return o.Precision, true
}

// HasPrecision returns a boolean if a field has been set.
func (o *SLOHistoryMonitor) HasPrecision() bool {
	if o != nil && o.Precision != nil {
		return true
	}

	return false
}

// SetPrecision gets a reference to the given float64 and assigns it to the Precision field.
// Deprecated
func (o *SLOHistoryMonitor) SetPrecision(v float64) {
	o.Precision = &v
}

// GetPreview returns the Preview field value if set, zero value otherwise.
func (o *SLOHistoryMonitor) GetPreview() bool {
	if o == nil || o.Preview == nil {
		var ret bool
		return ret
	}
	return *o.Preview
}

// GetPreviewOk returns a tuple with the Preview field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryMonitor) GetPreviewOk() (*bool, bool) {
	if o == nil || o.Preview == nil {
		return nil, false
	}
	return o.Preview, true
}

// HasPreview returns a boolean if a field has been set.
func (o *SLOHistoryMonitor) HasPreview() bool {
	if o != nil && o.Preview != nil {
		return true
	}

	return false
}

// SetPreview gets a reference to the given bool and assigns it to the Preview field.
func (o *SLOHistoryMonitor) SetPreview(v bool) {
	o.Preview = &v
}

// GetSliValue returns the SliValue field value if set, zero value otherwise.
func (o *SLOHistoryMonitor) GetSliValue() float64 {
	if o == nil || o.SliValue == nil {
		var ret float64
		return ret
	}
	return *o.SliValue
}

// GetSliValueOk returns a tuple with the SliValue field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryMonitor) GetSliValueOk() (*float64, bool) {
	if o == nil || o.SliValue == nil {
		return nil, false
	}
	return o.SliValue, true
}

// HasSliValue returns a boolean if a field has been set.
func (o *SLOHistoryMonitor) HasSliValue() bool {
	if o != nil && o.SliValue != nil {
		return true
	}

	return false
}

// SetSliValue gets a reference to the given float64 and assigns it to the SliValue field.
func (o *SLOHistoryMonitor) SetSliValue(v float64) {
	o.SliValue = &v
}

// GetSpanPrecision returns the SpanPrecision field value if set, zero value otherwise.
func (o *SLOHistoryMonitor) GetSpanPrecision() float64 {
	if o == nil || o.SpanPrecision == nil {
		var ret float64
		return ret
	}
	return *o.SpanPrecision
}

// GetSpanPrecisionOk returns a tuple with the SpanPrecision field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryMonitor) GetSpanPrecisionOk() (*float64, bool) {
	if o == nil || o.SpanPrecision == nil {
		return nil, false
	}
	return o.SpanPrecision, true
}

// HasSpanPrecision returns a boolean if a field has been set.
func (o *SLOHistoryMonitor) HasSpanPrecision() bool {
	if o != nil && o.SpanPrecision != nil {
		return true
	}

	return false
}

// SetSpanPrecision gets a reference to the given float64 and assigns it to the SpanPrecision field.
func (o *SLOHistoryMonitor) SetSpanPrecision(v float64) {
	o.SpanPrecision = &v
}

// GetUptime returns the Uptime field value if set, zero value otherwise.
// Deprecated
func (o *SLOHistoryMonitor) GetUptime() float64 {
	if o == nil || o.Uptime == nil {
		var ret float64
		return ret
	}
	return *o.Uptime
}

// GetUptimeOk returns a tuple with the Uptime field value if set, nil otherwise
// and a boolean to check if the value has been set.
// Deprecated
func (o *SLOHistoryMonitor) GetUptimeOk() (*float64, bool) {
	if o == nil || o.Uptime == nil {
		return nil, false
	}
	return o.Uptime, true
}

// HasUptime returns a boolean if a field has been set.
func (o *SLOHistoryMonitor) HasUptime() bool {
	if o != nil && o.Uptime != nil {
		return true
	}

	return false
}

// SetUptime gets a reference to the given float64 and assigns it to the Uptime field.
// Deprecated
func (o *SLOHistoryMonitor) SetUptime(v float64) {
	o.Uptime = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SLOHistoryMonitor) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.ErrorBudgetRemaining != nil {
		toSerialize["error_budget_remaining"] = o.ErrorBudgetRemaining
	}
	if o.Errors != nil {
		toSerialize["errors"] = o.Errors
	}
	if o.Group != nil {
		toSerialize["group"] = o.Group
	}
	if o.History != nil {
		toSerialize["history"] = o.History
	}
	if o.MonitorModified != nil {
		toSerialize["monitor_modified"] = o.MonitorModified
	}
	if o.MonitorType != nil {
		toSerialize["monitor_type"] = o.MonitorType
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.Precision != nil {
		toSerialize["precision"] = o.Precision
	}
	if o.Preview != nil {
		toSerialize["preview"] = o.Preview
	}
	if o.SliValue != nil {
		toSerialize["sli_value"] = o.SliValue
	}
	if o.SpanPrecision != nil {
		toSerialize["span_precision"] = o.SpanPrecision
	}
	if o.Uptime != nil {
		toSerialize["uptime"] = o.Uptime
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SLOHistoryMonitor) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		ErrorBudgetRemaining map[string]float64                `json:"error_budget_remaining,omitempty"`
		Errors               []SLOHistoryResponseErrorWithType `json:"errors,omitempty"`
		Group                *string                           `json:"group,omitempty"`
		History              [][]float64                       `json:"history,omitempty"`
		MonitorModified      *int64                            `json:"monitor_modified,omitempty"`
		MonitorType          *string                           `json:"monitor_type,omitempty"`
		Name                 *string                           `json:"name,omitempty"`
		Precision            *float64                          `json:"precision,omitempty"`
		Preview              *bool                             `json:"preview,omitempty"`
		SliValue             *float64                          `json:"sli_value,omitempty"`
		SpanPrecision        *float64                          `json:"span_precision,omitempty"`
		Uptime               *float64                          `json:"uptime,omitempty"`
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
	o.ErrorBudgetRemaining = all.ErrorBudgetRemaining
	o.Errors = all.Errors
	o.Group = all.Group
	o.History = all.History
	o.MonitorModified = all.MonitorModified
	o.MonitorType = all.MonitorType
	o.Name = all.Name
	o.Precision = all.Precision
	o.Preview = all.Preview
	o.SliValue = all.SliValue
	o.SpanPrecision = all.SpanPrecision
	o.Uptime = all.Uptime
	return nil
}
