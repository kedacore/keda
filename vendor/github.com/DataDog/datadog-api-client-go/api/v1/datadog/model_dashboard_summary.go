// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// DashboardSummary Dashboard summary response.
type DashboardSummary struct {
	// List of dashboard definitions.
	Dashboards []DashboardSummaryDefinition `json:"dashboards,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewDashboardSummary instantiates a new DashboardSummary object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewDashboardSummary() *DashboardSummary {
	this := DashboardSummary{}
	return &this
}

// NewDashboardSummaryWithDefaults instantiates a new DashboardSummary object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewDashboardSummaryWithDefaults() *DashboardSummary {
	this := DashboardSummary{}
	return &this
}

// GetDashboards returns the Dashboards field value if set, zero value otherwise.
func (o *DashboardSummary) GetDashboards() []DashboardSummaryDefinition {
	if o == nil || o.Dashboards == nil {
		var ret []DashboardSummaryDefinition
		return ret
	}
	return o.Dashboards
}

// GetDashboardsOk returns a tuple with the Dashboards field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DashboardSummary) GetDashboardsOk() (*[]DashboardSummaryDefinition, bool) {
	if o == nil || o.Dashboards == nil {
		return nil, false
	}
	return &o.Dashboards, true
}

// HasDashboards returns a boolean if a field has been set.
func (o *DashboardSummary) HasDashboards() bool {
	if o != nil && o.Dashboards != nil {
		return true
	}

	return false
}

// SetDashboards gets a reference to the given []DashboardSummaryDefinition and assigns it to the Dashboards field.
func (o *DashboardSummary) SetDashboards(v []DashboardSummaryDefinition) {
	o.Dashboards = v
}

// MarshalJSON serializes the struct using spec logic.
func (o DashboardSummary) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Dashboards != nil {
		toSerialize["dashboards"] = o.Dashboards
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *DashboardSummary) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Dashboards []DashboardSummaryDefinition `json:"dashboards,omitempty"`
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
	o.Dashboards = all.Dashboards
	return nil
}
