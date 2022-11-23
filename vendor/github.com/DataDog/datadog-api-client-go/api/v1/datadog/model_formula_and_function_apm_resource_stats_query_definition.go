// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// FormulaAndFunctionApmResourceStatsQueryDefinition APM resource stats query using formulas and functions.
type FormulaAndFunctionApmResourceStatsQueryDefinition struct {
	// Data source for APM resource stats queries.
	DataSource FormulaAndFunctionApmResourceStatsDataSource `json:"data_source"`
	// APM environment.
	Env string `json:"env"`
	// Array of fields to group results by.
	GroupBy []string `json:"group_by,omitempty"`
	// Name of this query to use in formulas.
	Name string `json:"name"`
	// Name of operation on service.
	OperationName *string `json:"operation_name,omitempty"`
	// Name of the second primary tag used within APM. Required when `primary_tag_value` is specified. See https://docs.datadoghq.com/tracing/guide/setting_primary_tags_to_scope/#add-a-second-primary-tag-in-datadog
	PrimaryTagName *string `json:"primary_tag_name,omitempty"`
	// Value of the second primary tag by which to filter APM data. `primary_tag_name` must also be specified.
	PrimaryTagValue *string `json:"primary_tag_value,omitempty"`
	// APM resource name.
	ResourceName *string `json:"resource_name,omitempty"`
	// APM service name.
	Service string `json:"service"`
	// APM resource stat name.
	Stat FormulaAndFunctionApmResourceStatName `json:"stat"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewFormulaAndFunctionApmResourceStatsQueryDefinition instantiates a new FormulaAndFunctionApmResourceStatsQueryDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewFormulaAndFunctionApmResourceStatsQueryDefinition(dataSource FormulaAndFunctionApmResourceStatsDataSource, env string, name string, service string, stat FormulaAndFunctionApmResourceStatName) *FormulaAndFunctionApmResourceStatsQueryDefinition {
	this := FormulaAndFunctionApmResourceStatsQueryDefinition{}
	this.DataSource = dataSource
	this.Env = env
	this.Name = name
	this.Service = service
	this.Stat = stat
	return &this
}

// NewFormulaAndFunctionApmResourceStatsQueryDefinitionWithDefaults instantiates a new FormulaAndFunctionApmResourceStatsQueryDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewFormulaAndFunctionApmResourceStatsQueryDefinitionWithDefaults() *FormulaAndFunctionApmResourceStatsQueryDefinition {
	this := FormulaAndFunctionApmResourceStatsQueryDefinition{}
	return &this
}

// GetDataSource returns the DataSource field value.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) GetDataSource() FormulaAndFunctionApmResourceStatsDataSource {
	if o == nil {
		var ret FormulaAndFunctionApmResourceStatsDataSource
		return ret
	}
	return o.DataSource
}

// GetDataSourceOk returns a tuple with the DataSource field value
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) GetDataSourceOk() (*FormulaAndFunctionApmResourceStatsDataSource, bool) {
	if o == nil {
		return nil, false
	}
	return &o.DataSource, true
}

// SetDataSource sets field value.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) SetDataSource(v FormulaAndFunctionApmResourceStatsDataSource) {
	o.DataSource = v
}

// GetEnv returns the Env field value.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) GetEnv() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Env
}

// GetEnvOk returns a tuple with the Env field value
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) GetEnvOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Env, true
}

// SetEnv sets field value.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) SetEnv(v string) {
	o.Env = v
}

// GetGroupBy returns the GroupBy field value if set, zero value otherwise.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) GetGroupBy() []string {
	if o == nil || o.GroupBy == nil {
		var ret []string
		return ret
	}
	return o.GroupBy
}

// GetGroupByOk returns a tuple with the GroupBy field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) GetGroupByOk() (*[]string, bool) {
	if o == nil || o.GroupBy == nil {
		return nil, false
	}
	return &o.GroupBy, true
}

// HasGroupBy returns a boolean if a field has been set.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) HasGroupBy() bool {
	if o != nil && o.GroupBy != nil {
		return true
	}

	return false
}

// SetGroupBy gets a reference to the given []string and assigns it to the GroupBy field.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) SetGroupBy(v []string) {
	o.GroupBy = v
}

// GetName returns the Name field value.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) SetName(v string) {
	o.Name = v
}

// GetOperationName returns the OperationName field value if set, zero value otherwise.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) GetOperationName() string {
	if o == nil || o.OperationName == nil {
		var ret string
		return ret
	}
	return *o.OperationName
}

// GetOperationNameOk returns a tuple with the OperationName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) GetOperationNameOk() (*string, bool) {
	if o == nil || o.OperationName == nil {
		return nil, false
	}
	return o.OperationName, true
}

// HasOperationName returns a boolean if a field has been set.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) HasOperationName() bool {
	if o != nil && o.OperationName != nil {
		return true
	}

	return false
}

// SetOperationName gets a reference to the given string and assigns it to the OperationName field.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) SetOperationName(v string) {
	o.OperationName = &v
}

// GetPrimaryTagName returns the PrimaryTagName field value if set, zero value otherwise.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) GetPrimaryTagName() string {
	if o == nil || o.PrimaryTagName == nil {
		var ret string
		return ret
	}
	return *o.PrimaryTagName
}

// GetPrimaryTagNameOk returns a tuple with the PrimaryTagName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) GetPrimaryTagNameOk() (*string, bool) {
	if o == nil || o.PrimaryTagName == nil {
		return nil, false
	}
	return o.PrimaryTagName, true
}

// HasPrimaryTagName returns a boolean if a field has been set.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) HasPrimaryTagName() bool {
	if o != nil && o.PrimaryTagName != nil {
		return true
	}

	return false
}

// SetPrimaryTagName gets a reference to the given string and assigns it to the PrimaryTagName field.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) SetPrimaryTagName(v string) {
	o.PrimaryTagName = &v
}

// GetPrimaryTagValue returns the PrimaryTagValue field value if set, zero value otherwise.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) GetPrimaryTagValue() string {
	if o == nil || o.PrimaryTagValue == nil {
		var ret string
		return ret
	}
	return *o.PrimaryTagValue
}

// GetPrimaryTagValueOk returns a tuple with the PrimaryTagValue field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) GetPrimaryTagValueOk() (*string, bool) {
	if o == nil || o.PrimaryTagValue == nil {
		return nil, false
	}
	return o.PrimaryTagValue, true
}

// HasPrimaryTagValue returns a boolean if a field has been set.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) HasPrimaryTagValue() bool {
	if o != nil && o.PrimaryTagValue != nil {
		return true
	}

	return false
}

// SetPrimaryTagValue gets a reference to the given string and assigns it to the PrimaryTagValue field.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) SetPrimaryTagValue(v string) {
	o.PrimaryTagValue = &v
}

// GetResourceName returns the ResourceName field value if set, zero value otherwise.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) GetResourceName() string {
	if o == nil || o.ResourceName == nil {
		var ret string
		return ret
	}
	return *o.ResourceName
}

// GetResourceNameOk returns a tuple with the ResourceName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) GetResourceNameOk() (*string, bool) {
	if o == nil || o.ResourceName == nil {
		return nil, false
	}
	return o.ResourceName, true
}

// HasResourceName returns a boolean if a field has been set.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) HasResourceName() bool {
	if o != nil && o.ResourceName != nil {
		return true
	}

	return false
}

// SetResourceName gets a reference to the given string and assigns it to the ResourceName field.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) SetResourceName(v string) {
	o.ResourceName = &v
}

// GetService returns the Service field value.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) GetService() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Service
}

// GetServiceOk returns a tuple with the Service field value
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) GetServiceOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Service, true
}

// SetService sets field value.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) SetService(v string) {
	o.Service = v
}

// GetStat returns the Stat field value.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) GetStat() FormulaAndFunctionApmResourceStatName {
	if o == nil {
		var ret FormulaAndFunctionApmResourceStatName
		return ret
	}
	return o.Stat
}

// GetStatOk returns a tuple with the Stat field value
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) GetStatOk() (*FormulaAndFunctionApmResourceStatName, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Stat, true
}

// SetStat sets field value.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) SetStat(v FormulaAndFunctionApmResourceStatName) {
	o.Stat = v
}

// MarshalJSON serializes the struct using spec logic.
func (o FormulaAndFunctionApmResourceStatsQueryDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["data_source"] = o.DataSource
	toSerialize["env"] = o.Env
	if o.GroupBy != nil {
		toSerialize["group_by"] = o.GroupBy
	}
	toSerialize["name"] = o.Name
	if o.OperationName != nil {
		toSerialize["operation_name"] = o.OperationName
	}
	if o.PrimaryTagName != nil {
		toSerialize["primary_tag_name"] = o.PrimaryTagName
	}
	if o.PrimaryTagValue != nil {
		toSerialize["primary_tag_value"] = o.PrimaryTagValue
	}
	if o.ResourceName != nil {
		toSerialize["resource_name"] = o.ResourceName
	}
	toSerialize["service"] = o.Service
	toSerialize["stat"] = o.Stat

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *FormulaAndFunctionApmResourceStatsQueryDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		DataSource *FormulaAndFunctionApmResourceStatsDataSource `json:"data_source"`
		Env        *string                                       `json:"env"`
		Name       *string                                       `json:"name"`
		Service    *string                                       `json:"service"`
		Stat       *FormulaAndFunctionApmResourceStatName        `json:"stat"`
	}{}
	all := struct {
		DataSource      FormulaAndFunctionApmResourceStatsDataSource `json:"data_source"`
		Env             string                                       `json:"env"`
		GroupBy         []string                                     `json:"group_by,omitempty"`
		Name            string                                       `json:"name"`
		OperationName   *string                                      `json:"operation_name,omitempty"`
		PrimaryTagName  *string                                      `json:"primary_tag_name,omitempty"`
		PrimaryTagValue *string                                      `json:"primary_tag_value,omitempty"`
		ResourceName    *string                                      `json:"resource_name,omitempty"`
		Service         string                                       `json:"service"`
		Stat            FormulaAndFunctionApmResourceStatName        `json:"stat"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.DataSource == nil {
		return fmt.Errorf("Required field data_source missing")
	}
	if required.Env == nil {
		return fmt.Errorf("Required field env missing")
	}
	if required.Name == nil {
		return fmt.Errorf("Required field name missing")
	}
	if required.Service == nil {
		return fmt.Errorf("Required field service missing")
	}
	if required.Stat == nil {
		return fmt.Errorf("Required field stat missing")
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
	if v := all.Stat; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.DataSource = all.DataSource
	o.Env = all.Env
	o.GroupBy = all.GroupBy
	o.Name = all.Name
	o.OperationName = all.OperationName
	o.PrimaryTagName = all.PrimaryTagName
	o.PrimaryTagValue = all.PrimaryTagValue
	o.ResourceName = all.ResourceName
	o.Service = all.Service
	o.Stat = all.Stat
	return nil
}
