// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SLOHistoryResponseData An array of service level objective objects.
type SLOHistoryResponseData struct {
	// The `from` timestamp in epoch seconds.
	FromTs *int64 `json:"from_ts,omitempty"`
	// For `metric` based SLOs where the query includes a group-by clause, this represents the list of grouping parameters.
	//
	// This is not included in responses for `monitor` based SLOs.
	GroupBy []string `json:"group_by,omitempty"`
	// For grouped SLOs, this represents SLI data for specific groups.
	//
	// This is not included in the responses for `metric` based SLOs.
	Groups []SLOHistoryMonitor `json:"groups,omitempty"`
	// For multi-monitor SLOs, this represents SLI data for specific monitors.
	//
	// This is not included in the responses for `metric` based SLOs.
	Monitors []SLOHistoryMonitor `json:"monitors,omitempty"`
	// An object that holds an SLI value and its associated data. It can represent an SLO's overall SLI value.
	// This can also represent the SLI value for a specific monitor in multi-monitor SLOs, or a group in grouped SLOs.
	Overall *SLOHistorySLIData `json:"overall,omitempty"`
	// A `metric` based SLO history response.
	//
	// This is not included in responses for `monitor` based SLOs.
	Series *SLOHistoryMetrics `json:"series,omitempty"`
	// mapping of string timeframe to the SLO threshold.
	Thresholds map[string]SLOThreshold `json:"thresholds,omitempty"`
	// The `to` timestamp in epoch seconds.
	ToTs *int64 `json:"to_ts,omitempty"`
	// The type of the service level objective.
	Type *SLOType `json:"type,omitempty"`
	// A numeric representation of the type of the service level objective (`0` for
	// monitor, `1` for metric). Always included in service level objective responses.
	// Ignored in create/update requests.
	TypeId *SLOTypeNumeric `json:"type_id,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSLOHistoryResponseData instantiates a new SLOHistoryResponseData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSLOHistoryResponseData() *SLOHistoryResponseData {
	this := SLOHistoryResponseData{}
	return &this
}

// NewSLOHistoryResponseDataWithDefaults instantiates a new SLOHistoryResponseData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSLOHistoryResponseDataWithDefaults() *SLOHistoryResponseData {
	this := SLOHistoryResponseData{}
	return &this
}

// GetFromTs returns the FromTs field value if set, zero value otherwise.
func (o *SLOHistoryResponseData) GetFromTs() int64 {
	if o == nil || o.FromTs == nil {
		var ret int64
		return ret
	}
	return *o.FromTs
}

// GetFromTsOk returns a tuple with the FromTs field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryResponseData) GetFromTsOk() (*int64, bool) {
	if o == nil || o.FromTs == nil {
		return nil, false
	}
	return o.FromTs, true
}

// HasFromTs returns a boolean if a field has been set.
func (o *SLOHistoryResponseData) HasFromTs() bool {
	if o != nil && o.FromTs != nil {
		return true
	}

	return false
}

// SetFromTs gets a reference to the given int64 and assigns it to the FromTs field.
func (o *SLOHistoryResponseData) SetFromTs(v int64) {
	o.FromTs = &v
}

// GetGroupBy returns the GroupBy field value if set, zero value otherwise.
func (o *SLOHistoryResponseData) GetGroupBy() []string {
	if o == nil || o.GroupBy == nil {
		var ret []string
		return ret
	}
	return o.GroupBy
}

// GetGroupByOk returns a tuple with the GroupBy field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryResponseData) GetGroupByOk() (*[]string, bool) {
	if o == nil || o.GroupBy == nil {
		return nil, false
	}
	return &o.GroupBy, true
}

// HasGroupBy returns a boolean if a field has been set.
func (o *SLOHistoryResponseData) HasGroupBy() bool {
	if o != nil && o.GroupBy != nil {
		return true
	}

	return false
}

// SetGroupBy gets a reference to the given []string and assigns it to the GroupBy field.
func (o *SLOHistoryResponseData) SetGroupBy(v []string) {
	o.GroupBy = v
}

// GetGroups returns the Groups field value if set, zero value otherwise.
func (o *SLOHistoryResponseData) GetGroups() []SLOHistoryMonitor {
	if o == nil || o.Groups == nil {
		var ret []SLOHistoryMonitor
		return ret
	}
	return o.Groups
}

// GetGroupsOk returns a tuple with the Groups field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryResponseData) GetGroupsOk() (*[]SLOHistoryMonitor, bool) {
	if o == nil || o.Groups == nil {
		return nil, false
	}
	return &o.Groups, true
}

// HasGroups returns a boolean if a field has been set.
func (o *SLOHistoryResponseData) HasGroups() bool {
	if o != nil && o.Groups != nil {
		return true
	}

	return false
}

// SetGroups gets a reference to the given []SLOHistoryMonitor and assigns it to the Groups field.
func (o *SLOHistoryResponseData) SetGroups(v []SLOHistoryMonitor) {
	o.Groups = v
}

// GetMonitors returns the Monitors field value if set, zero value otherwise.
func (o *SLOHistoryResponseData) GetMonitors() []SLOHistoryMonitor {
	if o == nil || o.Monitors == nil {
		var ret []SLOHistoryMonitor
		return ret
	}
	return o.Monitors
}

// GetMonitorsOk returns a tuple with the Monitors field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryResponseData) GetMonitorsOk() (*[]SLOHistoryMonitor, bool) {
	if o == nil || o.Monitors == nil {
		return nil, false
	}
	return &o.Monitors, true
}

// HasMonitors returns a boolean if a field has been set.
func (o *SLOHistoryResponseData) HasMonitors() bool {
	if o != nil && o.Monitors != nil {
		return true
	}

	return false
}

// SetMonitors gets a reference to the given []SLOHistoryMonitor and assigns it to the Monitors field.
func (o *SLOHistoryResponseData) SetMonitors(v []SLOHistoryMonitor) {
	o.Monitors = v
}

// GetOverall returns the Overall field value if set, zero value otherwise.
func (o *SLOHistoryResponseData) GetOverall() SLOHistorySLIData {
	if o == nil || o.Overall == nil {
		var ret SLOHistorySLIData
		return ret
	}
	return *o.Overall
}

// GetOverallOk returns a tuple with the Overall field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryResponseData) GetOverallOk() (*SLOHistorySLIData, bool) {
	if o == nil || o.Overall == nil {
		return nil, false
	}
	return o.Overall, true
}

// HasOverall returns a boolean if a field has been set.
func (o *SLOHistoryResponseData) HasOverall() bool {
	if o != nil && o.Overall != nil {
		return true
	}

	return false
}

// SetOverall gets a reference to the given SLOHistorySLIData and assigns it to the Overall field.
func (o *SLOHistoryResponseData) SetOverall(v SLOHistorySLIData) {
	o.Overall = &v
}

// GetSeries returns the Series field value if set, zero value otherwise.
func (o *SLOHistoryResponseData) GetSeries() SLOHistoryMetrics {
	if o == nil || o.Series == nil {
		var ret SLOHistoryMetrics
		return ret
	}
	return *o.Series
}

// GetSeriesOk returns a tuple with the Series field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryResponseData) GetSeriesOk() (*SLOHistoryMetrics, bool) {
	if o == nil || o.Series == nil {
		return nil, false
	}
	return o.Series, true
}

// HasSeries returns a boolean if a field has been set.
func (o *SLOHistoryResponseData) HasSeries() bool {
	if o != nil && o.Series != nil {
		return true
	}

	return false
}

// SetSeries gets a reference to the given SLOHistoryMetrics and assigns it to the Series field.
func (o *SLOHistoryResponseData) SetSeries(v SLOHistoryMetrics) {
	o.Series = &v
}

// GetThresholds returns the Thresholds field value if set, zero value otherwise.
func (o *SLOHistoryResponseData) GetThresholds() map[string]SLOThreshold {
	if o == nil || o.Thresholds == nil {
		var ret map[string]SLOThreshold
		return ret
	}
	return o.Thresholds
}

// GetThresholdsOk returns a tuple with the Thresholds field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryResponseData) GetThresholdsOk() (*map[string]SLOThreshold, bool) {
	if o == nil || o.Thresholds == nil {
		return nil, false
	}
	return &o.Thresholds, true
}

// HasThresholds returns a boolean if a field has been set.
func (o *SLOHistoryResponseData) HasThresholds() bool {
	if o != nil && o.Thresholds != nil {
		return true
	}

	return false
}

// SetThresholds gets a reference to the given map[string]SLOThreshold and assigns it to the Thresholds field.
func (o *SLOHistoryResponseData) SetThresholds(v map[string]SLOThreshold) {
	o.Thresholds = v
}

// GetToTs returns the ToTs field value if set, zero value otherwise.
func (o *SLOHistoryResponseData) GetToTs() int64 {
	if o == nil || o.ToTs == nil {
		var ret int64
		return ret
	}
	return *o.ToTs
}

// GetToTsOk returns a tuple with the ToTs field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryResponseData) GetToTsOk() (*int64, bool) {
	if o == nil || o.ToTs == nil {
		return nil, false
	}
	return o.ToTs, true
}

// HasToTs returns a boolean if a field has been set.
func (o *SLOHistoryResponseData) HasToTs() bool {
	if o != nil && o.ToTs != nil {
		return true
	}

	return false
}

// SetToTs gets a reference to the given int64 and assigns it to the ToTs field.
func (o *SLOHistoryResponseData) SetToTs(v int64) {
	o.ToTs = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *SLOHistoryResponseData) GetType() SLOType {
	if o == nil || o.Type == nil {
		var ret SLOType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryResponseData) GetTypeOk() (*SLOType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *SLOHistoryResponseData) HasType() bool {
	if o != nil && o.Type != nil {
		return true
	}

	return false
}

// SetType gets a reference to the given SLOType and assigns it to the Type field.
func (o *SLOHistoryResponseData) SetType(v SLOType) {
	o.Type = &v
}

// GetTypeId returns the TypeId field value if set, zero value otherwise.
func (o *SLOHistoryResponseData) GetTypeId() SLOTypeNumeric {
	if o == nil || o.TypeId == nil {
		var ret SLOTypeNumeric
		return ret
	}
	return *o.TypeId
}

// GetTypeIdOk returns a tuple with the TypeId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryResponseData) GetTypeIdOk() (*SLOTypeNumeric, bool) {
	if o == nil || o.TypeId == nil {
		return nil, false
	}
	return o.TypeId, true
}

// HasTypeId returns a boolean if a field has been set.
func (o *SLOHistoryResponseData) HasTypeId() bool {
	if o != nil && o.TypeId != nil {
		return true
	}

	return false
}

// SetTypeId gets a reference to the given SLOTypeNumeric and assigns it to the TypeId field.
func (o *SLOHistoryResponseData) SetTypeId(v SLOTypeNumeric) {
	o.TypeId = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SLOHistoryResponseData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.FromTs != nil {
		toSerialize["from_ts"] = o.FromTs
	}
	if o.GroupBy != nil {
		toSerialize["group_by"] = o.GroupBy
	}
	if o.Groups != nil {
		toSerialize["groups"] = o.Groups
	}
	if o.Monitors != nil {
		toSerialize["monitors"] = o.Monitors
	}
	if o.Overall != nil {
		toSerialize["overall"] = o.Overall
	}
	if o.Series != nil {
		toSerialize["series"] = o.Series
	}
	if o.Thresholds != nil {
		toSerialize["thresholds"] = o.Thresholds
	}
	if o.ToTs != nil {
		toSerialize["to_ts"] = o.ToTs
	}
	if o.Type != nil {
		toSerialize["type"] = o.Type
	}
	if o.TypeId != nil {
		toSerialize["type_id"] = o.TypeId
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SLOHistoryResponseData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		FromTs     *int64                  `json:"from_ts,omitempty"`
		GroupBy    []string                `json:"group_by,omitempty"`
		Groups     []SLOHistoryMonitor     `json:"groups,omitempty"`
		Monitors   []SLOHistoryMonitor     `json:"monitors,omitempty"`
		Overall    *SLOHistorySLIData      `json:"overall,omitempty"`
		Series     *SLOHistoryMetrics      `json:"series,omitempty"`
		Thresholds map[string]SLOThreshold `json:"thresholds,omitempty"`
		ToTs       *int64                  `json:"to_ts,omitempty"`
		Type       *SLOType                `json:"type,omitempty"`
		TypeId     *SLOTypeNumeric         `json:"type_id,omitempty"`
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
	if v := all.Type; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.TypeId; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.FromTs = all.FromTs
	o.GroupBy = all.GroupBy
	o.Groups = all.Groups
	o.Monitors = all.Monitors
	if all.Overall != nil && all.Overall.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Overall = all.Overall
	if all.Series != nil && all.Series.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Series = all.Series
	o.Thresholds = all.Thresholds
	o.ToTs = all.ToTs
	o.Type = all.Type
	o.TypeId = all.TypeId
	return nil
}
