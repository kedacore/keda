// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// MonitorFormulaAndFunctionEventQueryDefinitionSearch Search options.
type MonitorFormulaAndFunctionEventQueryDefinitionSearch struct {
	// Events search string.
	Query string `json:"query"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewMonitorFormulaAndFunctionEventQueryDefinitionSearch instantiates a new MonitorFormulaAndFunctionEventQueryDefinitionSearch object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMonitorFormulaAndFunctionEventQueryDefinitionSearch(query string) *MonitorFormulaAndFunctionEventQueryDefinitionSearch {
	this := MonitorFormulaAndFunctionEventQueryDefinitionSearch{}
	this.Query = query
	return &this
}

// NewMonitorFormulaAndFunctionEventQueryDefinitionSearchWithDefaults instantiates a new MonitorFormulaAndFunctionEventQueryDefinitionSearch object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMonitorFormulaAndFunctionEventQueryDefinitionSearchWithDefaults() *MonitorFormulaAndFunctionEventQueryDefinitionSearch {
	this := MonitorFormulaAndFunctionEventQueryDefinitionSearch{}
	return &this
}

// GetQuery returns the Query field value.
func (o *MonitorFormulaAndFunctionEventQueryDefinitionSearch) GetQuery() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Query
}

// GetQueryOk returns a tuple with the Query field value
// and a boolean to check if the value has been set.
func (o *MonitorFormulaAndFunctionEventQueryDefinitionSearch) GetQueryOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Query, true
}

// SetQuery sets field value.
func (o *MonitorFormulaAndFunctionEventQueryDefinitionSearch) SetQuery(v string) {
	o.Query = v
}

// MarshalJSON serializes the struct using spec logic.
func (o MonitorFormulaAndFunctionEventQueryDefinitionSearch) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["query"] = o.Query

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MonitorFormulaAndFunctionEventQueryDefinitionSearch) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Query *string `json:"query"`
	}{}
	all := struct {
		Query string `json:"query"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Query == nil {
		return fmt.Errorf("Required field query missing")
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
	o.Query = all.Query
	return nil
}
