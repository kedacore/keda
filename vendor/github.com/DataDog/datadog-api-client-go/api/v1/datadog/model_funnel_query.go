// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// FunnelQuery Updated funnel widget.
type FunnelQuery struct {
	// Source from which to query items to display in the funnel.
	DataSource FunnelSource `json:"data_source"`
	// The widget query.
	QueryString string `json:"query_string"`
	// List of funnel steps.
	Steps []FunnelStep `json:"steps"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewFunnelQuery instantiates a new FunnelQuery object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewFunnelQuery(dataSource FunnelSource, queryString string, steps []FunnelStep) *FunnelQuery {
	this := FunnelQuery{}
	this.DataSource = dataSource
	this.QueryString = queryString
	this.Steps = steps
	return &this
}

// NewFunnelQueryWithDefaults instantiates a new FunnelQuery object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewFunnelQueryWithDefaults() *FunnelQuery {
	this := FunnelQuery{}
	var dataSource FunnelSource = FUNNELSOURCE_RUM
	this.DataSource = dataSource
	return &this
}

// GetDataSource returns the DataSource field value.
func (o *FunnelQuery) GetDataSource() FunnelSource {
	if o == nil {
		var ret FunnelSource
		return ret
	}
	return o.DataSource
}

// GetDataSourceOk returns a tuple with the DataSource field value
// and a boolean to check if the value has been set.
func (o *FunnelQuery) GetDataSourceOk() (*FunnelSource, bool) {
	if o == nil {
		return nil, false
	}
	return &o.DataSource, true
}

// SetDataSource sets field value.
func (o *FunnelQuery) SetDataSource(v FunnelSource) {
	o.DataSource = v
}

// GetQueryString returns the QueryString field value.
func (o *FunnelQuery) GetQueryString() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.QueryString
}

// GetQueryStringOk returns a tuple with the QueryString field value
// and a boolean to check if the value has been set.
func (o *FunnelQuery) GetQueryStringOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.QueryString, true
}

// SetQueryString sets field value.
func (o *FunnelQuery) SetQueryString(v string) {
	o.QueryString = v
}

// GetSteps returns the Steps field value.
func (o *FunnelQuery) GetSteps() []FunnelStep {
	if o == nil {
		var ret []FunnelStep
		return ret
	}
	return o.Steps
}

// GetStepsOk returns a tuple with the Steps field value
// and a boolean to check if the value has been set.
func (o *FunnelQuery) GetStepsOk() (*[]FunnelStep, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Steps, true
}

// SetSteps sets field value.
func (o *FunnelQuery) SetSteps(v []FunnelStep) {
	o.Steps = v
}

// MarshalJSON serializes the struct using spec logic.
func (o FunnelQuery) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["data_source"] = o.DataSource
	toSerialize["query_string"] = o.QueryString
	toSerialize["steps"] = o.Steps

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *FunnelQuery) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		DataSource  *FunnelSource `json:"data_source"`
		QueryString *string       `json:"query_string"`
		Steps       *[]FunnelStep `json:"steps"`
	}{}
	all := struct {
		DataSource  FunnelSource `json:"data_source"`
		QueryString string       `json:"query_string"`
		Steps       []FunnelStep `json:"steps"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.DataSource == nil {
		return fmt.Errorf("Required field data_source missing")
	}
	if required.QueryString == nil {
		return fmt.Errorf("Required field query_string missing")
	}
	if required.Steps == nil {
		return fmt.Errorf("Required field steps missing")
	}
	err = json.Unmarshal(bytes, &all)
	if err != nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.DataSource; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.DataSource = all.DataSource
	o.QueryString = all.QueryString
	o.Steps = all.Steps
	return nil
}
