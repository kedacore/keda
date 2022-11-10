// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// FormulaAndFunctionMetricQueryDefinition A formula and functions metrics query.
type FormulaAndFunctionMetricQueryDefinition struct {
	// The aggregation methods available for metrics queries.
	Aggregator *FormulaAndFunctionMetricAggregation `json:"aggregator,omitempty"`
	// Data source for metrics queries.
	DataSource FormulaAndFunctionMetricDataSource `json:"data_source"`
	// Name of the query for use in formulas.
	Name string `json:"name"`
	// Metrics query definition.
	Query string `json:"query"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewFormulaAndFunctionMetricQueryDefinition instantiates a new FormulaAndFunctionMetricQueryDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewFormulaAndFunctionMetricQueryDefinition(dataSource FormulaAndFunctionMetricDataSource, name string, query string) *FormulaAndFunctionMetricQueryDefinition {
	this := FormulaAndFunctionMetricQueryDefinition{}
	this.DataSource = dataSource
	this.Name = name
	this.Query = query
	return &this
}

// NewFormulaAndFunctionMetricQueryDefinitionWithDefaults instantiates a new FormulaAndFunctionMetricQueryDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewFormulaAndFunctionMetricQueryDefinitionWithDefaults() *FormulaAndFunctionMetricQueryDefinition {
	this := FormulaAndFunctionMetricQueryDefinition{}
	return &this
}

// GetAggregator returns the Aggregator field value if set, zero value otherwise.
func (o *FormulaAndFunctionMetricQueryDefinition) GetAggregator() FormulaAndFunctionMetricAggregation {
	if o == nil || o.Aggregator == nil {
		var ret FormulaAndFunctionMetricAggregation
		return ret
	}
	return *o.Aggregator
}

// GetAggregatorOk returns a tuple with the Aggregator field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionMetricQueryDefinition) GetAggregatorOk() (*FormulaAndFunctionMetricAggregation, bool) {
	if o == nil || o.Aggregator == nil {
		return nil, false
	}
	return o.Aggregator, true
}

// HasAggregator returns a boolean if a field has been set.
func (o *FormulaAndFunctionMetricQueryDefinition) HasAggregator() bool {
	if o != nil && o.Aggregator != nil {
		return true
	}

	return false
}

// SetAggregator gets a reference to the given FormulaAndFunctionMetricAggregation and assigns it to the Aggregator field.
func (o *FormulaAndFunctionMetricQueryDefinition) SetAggregator(v FormulaAndFunctionMetricAggregation) {
	o.Aggregator = &v
}

// GetDataSource returns the DataSource field value.
func (o *FormulaAndFunctionMetricQueryDefinition) GetDataSource() FormulaAndFunctionMetricDataSource {
	if o == nil {
		var ret FormulaAndFunctionMetricDataSource
		return ret
	}
	return o.DataSource
}

// GetDataSourceOk returns a tuple with the DataSource field value
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionMetricQueryDefinition) GetDataSourceOk() (*FormulaAndFunctionMetricDataSource, bool) {
	if o == nil {
		return nil, false
	}
	return &o.DataSource, true
}

// SetDataSource sets field value.
func (o *FormulaAndFunctionMetricQueryDefinition) SetDataSource(v FormulaAndFunctionMetricDataSource) {
	o.DataSource = v
}

// GetName returns the Name field value.
func (o *FormulaAndFunctionMetricQueryDefinition) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionMetricQueryDefinition) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *FormulaAndFunctionMetricQueryDefinition) SetName(v string) {
	o.Name = v
}

// GetQuery returns the Query field value.
func (o *FormulaAndFunctionMetricQueryDefinition) GetQuery() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Query
}

// GetQueryOk returns a tuple with the Query field value
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionMetricQueryDefinition) GetQueryOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Query, true
}

// SetQuery sets field value.
func (o *FormulaAndFunctionMetricQueryDefinition) SetQuery(v string) {
	o.Query = v
}

// MarshalJSON serializes the struct using spec logic.
func (o FormulaAndFunctionMetricQueryDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Aggregator != nil {
		toSerialize["aggregator"] = o.Aggregator
	}
	toSerialize["data_source"] = o.DataSource
	toSerialize["name"] = o.Name
	toSerialize["query"] = o.Query

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *FormulaAndFunctionMetricQueryDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		DataSource *FormulaAndFunctionMetricDataSource `json:"data_source"`
		Name       *string                             `json:"name"`
		Query      *string                             `json:"query"`
	}{}
	all := struct {
		Aggregator *FormulaAndFunctionMetricAggregation `json:"aggregator,omitempty"`
		DataSource FormulaAndFunctionMetricDataSource   `json:"data_source"`
		Name       string                               `json:"name"`
		Query      string                               `json:"query"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.DataSource == nil {
		return fmt.Errorf("Required field data_source missing")
	}
	if required.Name == nil {
		return fmt.Errorf("Required field name missing")
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
	if v := all.Aggregator; v != nil && !v.IsValid() {
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
	o.Aggregator = all.Aggregator
	o.DataSource = all.DataSource
	o.Name = all.Name
	o.Query = all.Query
	return nil
}
