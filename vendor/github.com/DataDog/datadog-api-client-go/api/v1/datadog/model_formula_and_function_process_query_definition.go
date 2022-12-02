// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// FormulaAndFunctionProcessQueryDefinition Process query using formulas and functions.
type FormulaAndFunctionProcessQueryDefinition struct {
	// The aggregation methods available for metrics queries.
	Aggregator *FormulaAndFunctionMetricAggregation `json:"aggregator,omitempty"`
	// Data sources that rely on the process backend.
	DataSource FormulaAndFunctionProcessQueryDataSource `json:"data_source"`
	// Whether to normalize the CPU percentages.
	IsNormalizedCpu *bool `json:"is_normalized_cpu,omitempty"`
	// Number of hits to return.
	Limit *int64 `json:"limit,omitempty"`
	// Process metric name.
	Metric string `json:"metric"`
	// Name of query for use in formulas.
	Name string `json:"name"`
	// Direction of sort.
	Sort *QuerySortOrder `json:"sort,omitempty"`
	// An array of tags to filter by.
	TagFilters []string `json:"tag_filters,omitempty"`
	// Text to use as filter.
	TextFilter *string `json:"text_filter,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewFormulaAndFunctionProcessQueryDefinition instantiates a new FormulaAndFunctionProcessQueryDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewFormulaAndFunctionProcessQueryDefinition(dataSource FormulaAndFunctionProcessQueryDataSource, metric string, name string) *FormulaAndFunctionProcessQueryDefinition {
	this := FormulaAndFunctionProcessQueryDefinition{}
	this.DataSource = dataSource
	this.Metric = metric
	this.Name = name
	var sort QuerySortOrder = QUERYSORTORDER_DESC
	this.Sort = &sort
	return &this
}

// NewFormulaAndFunctionProcessQueryDefinitionWithDefaults instantiates a new FormulaAndFunctionProcessQueryDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewFormulaAndFunctionProcessQueryDefinitionWithDefaults() *FormulaAndFunctionProcessQueryDefinition {
	this := FormulaAndFunctionProcessQueryDefinition{}
	var sort QuerySortOrder = QUERYSORTORDER_DESC
	this.Sort = &sort
	return &this
}

// GetAggregator returns the Aggregator field value if set, zero value otherwise.
func (o *FormulaAndFunctionProcessQueryDefinition) GetAggregator() FormulaAndFunctionMetricAggregation {
	if o == nil || o.Aggregator == nil {
		var ret FormulaAndFunctionMetricAggregation
		return ret
	}
	return *o.Aggregator
}

// GetAggregatorOk returns a tuple with the Aggregator field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionProcessQueryDefinition) GetAggregatorOk() (*FormulaAndFunctionMetricAggregation, bool) {
	if o == nil || o.Aggregator == nil {
		return nil, false
	}
	return o.Aggregator, true
}

// HasAggregator returns a boolean if a field has been set.
func (o *FormulaAndFunctionProcessQueryDefinition) HasAggregator() bool {
	if o != nil && o.Aggregator != nil {
		return true
	}

	return false
}

// SetAggregator gets a reference to the given FormulaAndFunctionMetricAggregation and assigns it to the Aggregator field.
func (o *FormulaAndFunctionProcessQueryDefinition) SetAggregator(v FormulaAndFunctionMetricAggregation) {
	o.Aggregator = &v
}

// GetDataSource returns the DataSource field value.
func (o *FormulaAndFunctionProcessQueryDefinition) GetDataSource() FormulaAndFunctionProcessQueryDataSource {
	if o == nil {
		var ret FormulaAndFunctionProcessQueryDataSource
		return ret
	}
	return o.DataSource
}

// GetDataSourceOk returns a tuple with the DataSource field value
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionProcessQueryDefinition) GetDataSourceOk() (*FormulaAndFunctionProcessQueryDataSource, bool) {
	if o == nil {
		return nil, false
	}
	return &o.DataSource, true
}

// SetDataSource sets field value.
func (o *FormulaAndFunctionProcessQueryDefinition) SetDataSource(v FormulaAndFunctionProcessQueryDataSource) {
	o.DataSource = v
}

// GetIsNormalizedCpu returns the IsNormalizedCpu field value if set, zero value otherwise.
func (o *FormulaAndFunctionProcessQueryDefinition) GetIsNormalizedCpu() bool {
	if o == nil || o.IsNormalizedCpu == nil {
		var ret bool
		return ret
	}
	return *o.IsNormalizedCpu
}

// GetIsNormalizedCpuOk returns a tuple with the IsNormalizedCpu field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionProcessQueryDefinition) GetIsNormalizedCpuOk() (*bool, bool) {
	if o == nil || o.IsNormalizedCpu == nil {
		return nil, false
	}
	return o.IsNormalizedCpu, true
}

// HasIsNormalizedCpu returns a boolean if a field has been set.
func (o *FormulaAndFunctionProcessQueryDefinition) HasIsNormalizedCpu() bool {
	if o != nil && o.IsNormalizedCpu != nil {
		return true
	}

	return false
}

// SetIsNormalizedCpu gets a reference to the given bool and assigns it to the IsNormalizedCpu field.
func (o *FormulaAndFunctionProcessQueryDefinition) SetIsNormalizedCpu(v bool) {
	o.IsNormalizedCpu = &v
}

// GetLimit returns the Limit field value if set, zero value otherwise.
func (o *FormulaAndFunctionProcessQueryDefinition) GetLimit() int64 {
	if o == nil || o.Limit == nil {
		var ret int64
		return ret
	}
	return *o.Limit
}

// GetLimitOk returns a tuple with the Limit field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionProcessQueryDefinition) GetLimitOk() (*int64, bool) {
	if o == nil || o.Limit == nil {
		return nil, false
	}
	return o.Limit, true
}

// HasLimit returns a boolean if a field has been set.
func (o *FormulaAndFunctionProcessQueryDefinition) HasLimit() bool {
	if o != nil && o.Limit != nil {
		return true
	}

	return false
}

// SetLimit gets a reference to the given int64 and assigns it to the Limit field.
func (o *FormulaAndFunctionProcessQueryDefinition) SetLimit(v int64) {
	o.Limit = &v
}

// GetMetric returns the Metric field value.
func (o *FormulaAndFunctionProcessQueryDefinition) GetMetric() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Metric
}

// GetMetricOk returns a tuple with the Metric field value
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionProcessQueryDefinition) GetMetricOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Metric, true
}

// SetMetric sets field value.
func (o *FormulaAndFunctionProcessQueryDefinition) SetMetric(v string) {
	o.Metric = v
}

// GetName returns the Name field value.
func (o *FormulaAndFunctionProcessQueryDefinition) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionProcessQueryDefinition) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *FormulaAndFunctionProcessQueryDefinition) SetName(v string) {
	o.Name = v
}

// GetSort returns the Sort field value if set, zero value otherwise.
func (o *FormulaAndFunctionProcessQueryDefinition) GetSort() QuerySortOrder {
	if o == nil || o.Sort == nil {
		var ret QuerySortOrder
		return ret
	}
	return *o.Sort
}

// GetSortOk returns a tuple with the Sort field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionProcessQueryDefinition) GetSortOk() (*QuerySortOrder, bool) {
	if o == nil || o.Sort == nil {
		return nil, false
	}
	return o.Sort, true
}

// HasSort returns a boolean if a field has been set.
func (o *FormulaAndFunctionProcessQueryDefinition) HasSort() bool {
	if o != nil && o.Sort != nil {
		return true
	}

	return false
}

// SetSort gets a reference to the given QuerySortOrder and assigns it to the Sort field.
func (o *FormulaAndFunctionProcessQueryDefinition) SetSort(v QuerySortOrder) {
	o.Sort = &v
}

// GetTagFilters returns the TagFilters field value if set, zero value otherwise.
func (o *FormulaAndFunctionProcessQueryDefinition) GetTagFilters() []string {
	if o == nil || o.TagFilters == nil {
		var ret []string
		return ret
	}
	return o.TagFilters
}

// GetTagFiltersOk returns a tuple with the TagFilters field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionProcessQueryDefinition) GetTagFiltersOk() (*[]string, bool) {
	if o == nil || o.TagFilters == nil {
		return nil, false
	}
	return &o.TagFilters, true
}

// HasTagFilters returns a boolean if a field has been set.
func (o *FormulaAndFunctionProcessQueryDefinition) HasTagFilters() bool {
	if o != nil && o.TagFilters != nil {
		return true
	}

	return false
}

// SetTagFilters gets a reference to the given []string and assigns it to the TagFilters field.
func (o *FormulaAndFunctionProcessQueryDefinition) SetTagFilters(v []string) {
	o.TagFilters = v
}

// GetTextFilter returns the TextFilter field value if set, zero value otherwise.
func (o *FormulaAndFunctionProcessQueryDefinition) GetTextFilter() string {
	if o == nil || o.TextFilter == nil {
		var ret string
		return ret
	}
	return *o.TextFilter
}

// GetTextFilterOk returns a tuple with the TextFilter field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionProcessQueryDefinition) GetTextFilterOk() (*string, bool) {
	if o == nil || o.TextFilter == nil {
		return nil, false
	}
	return o.TextFilter, true
}

// HasTextFilter returns a boolean if a field has been set.
func (o *FormulaAndFunctionProcessQueryDefinition) HasTextFilter() bool {
	if o != nil && o.TextFilter != nil {
		return true
	}

	return false
}

// SetTextFilter gets a reference to the given string and assigns it to the TextFilter field.
func (o *FormulaAndFunctionProcessQueryDefinition) SetTextFilter(v string) {
	o.TextFilter = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o FormulaAndFunctionProcessQueryDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Aggregator != nil {
		toSerialize["aggregator"] = o.Aggregator
	}
	toSerialize["data_source"] = o.DataSource
	if o.IsNormalizedCpu != nil {
		toSerialize["is_normalized_cpu"] = o.IsNormalizedCpu
	}
	if o.Limit != nil {
		toSerialize["limit"] = o.Limit
	}
	toSerialize["metric"] = o.Metric
	toSerialize["name"] = o.Name
	if o.Sort != nil {
		toSerialize["sort"] = o.Sort
	}
	if o.TagFilters != nil {
		toSerialize["tag_filters"] = o.TagFilters
	}
	if o.TextFilter != nil {
		toSerialize["text_filter"] = o.TextFilter
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *FormulaAndFunctionProcessQueryDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		DataSource *FormulaAndFunctionProcessQueryDataSource `json:"data_source"`
		Metric     *string                                   `json:"metric"`
		Name       *string                                   `json:"name"`
	}{}
	all := struct {
		Aggregator      *FormulaAndFunctionMetricAggregation     `json:"aggregator,omitempty"`
		DataSource      FormulaAndFunctionProcessQueryDataSource `json:"data_source"`
		IsNormalizedCpu *bool                                    `json:"is_normalized_cpu,omitempty"`
		Limit           *int64                                   `json:"limit,omitempty"`
		Metric          string                                   `json:"metric"`
		Name            string                                   `json:"name"`
		Sort            *QuerySortOrder                          `json:"sort,omitempty"`
		TagFilters      []string                                 `json:"tag_filters,omitempty"`
		TextFilter      *string                                  `json:"text_filter,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.DataSource == nil {
		return fmt.Errorf("Required field data_source missing")
	}
	if required.Metric == nil {
		return fmt.Errorf("Required field metric missing")
	}
	if required.Name == nil {
		return fmt.Errorf("Required field name missing")
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
	if v := all.Sort; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Aggregator = all.Aggregator
	o.DataSource = all.DataSource
	o.IsNormalizedCpu = all.IsNormalizedCpu
	o.Limit = all.Limit
	o.Metric = all.Metric
	o.Name = all.Name
	o.Sort = all.Sort
	o.TagFilters = all.TagFilters
	o.TextFilter = all.TextFilter
	return nil
}
