// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// FormulaAndFunctionApmDependencyStatsQueryDefinition A formula and functions APM dependency stats query.
type FormulaAndFunctionApmDependencyStatsQueryDefinition struct {
	// Data source for APM dependency stats queries.
	DataSource FormulaAndFunctionApmDependencyStatsDataSource `json:"data_source"`
	// APM environment.
	Env string `json:"env"`
	// Determines whether stats for upstream or downstream dependencies should be queried.
	IsUpstream *bool `json:"is_upstream,omitempty"`
	// Name of query to use in formulas.
	Name string `json:"name"`
	// Name of operation on service.
	OperationName string `json:"operation_name"`
	// The name of the second primary tag used within APM; required when `primary_tag_value` is specified. See https://docs.datadoghq.com/tracing/guide/setting_primary_tags_to_scope/#add-a-second-primary-tag-in-datadog.
	PrimaryTagName *string `json:"primary_tag_name,omitempty"`
	// Filter APM data by the second primary tag. `primary_tag_name` must also be specified.
	PrimaryTagValue *string `json:"primary_tag_value,omitempty"`
	// APM resource.
	ResourceName string `json:"resource_name"`
	// APM service.
	Service string `json:"service"`
	// APM statistic.
	Stat FormulaAndFunctionApmDependencyStatName `json:"stat"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewFormulaAndFunctionApmDependencyStatsQueryDefinition instantiates a new FormulaAndFunctionApmDependencyStatsQueryDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewFormulaAndFunctionApmDependencyStatsQueryDefinition(dataSource FormulaAndFunctionApmDependencyStatsDataSource, env string, name string, operationName string, resourceName string, service string, stat FormulaAndFunctionApmDependencyStatName) *FormulaAndFunctionApmDependencyStatsQueryDefinition {
	this := FormulaAndFunctionApmDependencyStatsQueryDefinition{}
	this.DataSource = dataSource
	this.Env = env
	this.Name = name
	this.OperationName = operationName
	this.ResourceName = resourceName
	this.Service = service
	this.Stat = stat
	return &this
}

// NewFormulaAndFunctionApmDependencyStatsQueryDefinitionWithDefaults instantiates a new FormulaAndFunctionApmDependencyStatsQueryDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewFormulaAndFunctionApmDependencyStatsQueryDefinitionWithDefaults() *FormulaAndFunctionApmDependencyStatsQueryDefinition {
	this := FormulaAndFunctionApmDependencyStatsQueryDefinition{}
	return &this
}

// GetDataSource returns the DataSource field value.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) GetDataSource() FormulaAndFunctionApmDependencyStatsDataSource {
	if o == nil {
		var ret FormulaAndFunctionApmDependencyStatsDataSource
		return ret
	}
	return o.DataSource
}

// GetDataSourceOk returns a tuple with the DataSource field value
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) GetDataSourceOk() (*FormulaAndFunctionApmDependencyStatsDataSource, bool) {
	if o == nil {
		return nil, false
	}
	return &o.DataSource, true
}

// SetDataSource sets field value.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) SetDataSource(v FormulaAndFunctionApmDependencyStatsDataSource) {
	o.DataSource = v
}

// GetEnv returns the Env field value.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) GetEnv() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Env
}

// GetEnvOk returns a tuple with the Env field value
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) GetEnvOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Env, true
}

// SetEnv sets field value.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) SetEnv(v string) {
	o.Env = v
}

// GetIsUpstream returns the IsUpstream field value if set, zero value otherwise.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) GetIsUpstream() bool {
	if o == nil || o.IsUpstream == nil {
		var ret bool
		return ret
	}
	return *o.IsUpstream
}

// GetIsUpstreamOk returns a tuple with the IsUpstream field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) GetIsUpstreamOk() (*bool, bool) {
	if o == nil || o.IsUpstream == nil {
		return nil, false
	}
	return o.IsUpstream, true
}

// HasIsUpstream returns a boolean if a field has been set.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) HasIsUpstream() bool {
	if o != nil && o.IsUpstream != nil {
		return true
	}

	return false
}

// SetIsUpstream gets a reference to the given bool and assigns it to the IsUpstream field.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) SetIsUpstream(v bool) {
	o.IsUpstream = &v
}

// GetName returns the Name field value.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) SetName(v string) {
	o.Name = v
}

// GetOperationName returns the OperationName field value.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) GetOperationName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.OperationName
}

// GetOperationNameOk returns a tuple with the OperationName field value
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) GetOperationNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.OperationName, true
}

// SetOperationName sets field value.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) SetOperationName(v string) {
	o.OperationName = v
}

// GetPrimaryTagName returns the PrimaryTagName field value if set, zero value otherwise.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) GetPrimaryTagName() string {
	if o == nil || o.PrimaryTagName == nil {
		var ret string
		return ret
	}
	return *o.PrimaryTagName
}

// GetPrimaryTagNameOk returns a tuple with the PrimaryTagName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) GetPrimaryTagNameOk() (*string, bool) {
	if o == nil || o.PrimaryTagName == nil {
		return nil, false
	}
	return o.PrimaryTagName, true
}

// HasPrimaryTagName returns a boolean if a field has been set.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) HasPrimaryTagName() bool {
	if o != nil && o.PrimaryTagName != nil {
		return true
	}

	return false
}

// SetPrimaryTagName gets a reference to the given string and assigns it to the PrimaryTagName field.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) SetPrimaryTagName(v string) {
	o.PrimaryTagName = &v
}

// GetPrimaryTagValue returns the PrimaryTagValue field value if set, zero value otherwise.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) GetPrimaryTagValue() string {
	if o == nil || o.PrimaryTagValue == nil {
		var ret string
		return ret
	}
	return *o.PrimaryTagValue
}

// GetPrimaryTagValueOk returns a tuple with the PrimaryTagValue field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) GetPrimaryTagValueOk() (*string, bool) {
	if o == nil || o.PrimaryTagValue == nil {
		return nil, false
	}
	return o.PrimaryTagValue, true
}

// HasPrimaryTagValue returns a boolean if a field has been set.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) HasPrimaryTagValue() bool {
	if o != nil && o.PrimaryTagValue != nil {
		return true
	}

	return false
}

// SetPrimaryTagValue gets a reference to the given string and assigns it to the PrimaryTagValue field.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) SetPrimaryTagValue(v string) {
	o.PrimaryTagValue = &v
}

// GetResourceName returns the ResourceName field value.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) GetResourceName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.ResourceName
}

// GetResourceNameOk returns a tuple with the ResourceName field value
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) GetResourceNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ResourceName, true
}

// SetResourceName sets field value.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) SetResourceName(v string) {
	o.ResourceName = v
}

// GetService returns the Service field value.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) GetService() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Service
}

// GetServiceOk returns a tuple with the Service field value
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) GetServiceOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Service, true
}

// SetService sets field value.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) SetService(v string) {
	o.Service = v
}

// GetStat returns the Stat field value.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) GetStat() FormulaAndFunctionApmDependencyStatName {
	if o == nil {
		var ret FormulaAndFunctionApmDependencyStatName
		return ret
	}
	return o.Stat
}

// GetStatOk returns a tuple with the Stat field value
// and a boolean to check if the value has been set.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) GetStatOk() (*FormulaAndFunctionApmDependencyStatName, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Stat, true
}

// SetStat sets field value.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) SetStat(v FormulaAndFunctionApmDependencyStatName) {
	o.Stat = v
}

// MarshalJSON serializes the struct using spec logic.
func (o FormulaAndFunctionApmDependencyStatsQueryDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["data_source"] = o.DataSource
	toSerialize["env"] = o.Env
	if o.IsUpstream != nil {
		toSerialize["is_upstream"] = o.IsUpstream
	}
	toSerialize["name"] = o.Name
	toSerialize["operation_name"] = o.OperationName
	if o.PrimaryTagName != nil {
		toSerialize["primary_tag_name"] = o.PrimaryTagName
	}
	if o.PrimaryTagValue != nil {
		toSerialize["primary_tag_value"] = o.PrimaryTagValue
	}
	toSerialize["resource_name"] = o.ResourceName
	toSerialize["service"] = o.Service
	toSerialize["stat"] = o.Stat

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *FormulaAndFunctionApmDependencyStatsQueryDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		DataSource    *FormulaAndFunctionApmDependencyStatsDataSource `json:"data_source"`
		Env           *string                                         `json:"env"`
		Name          *string                                         `json:"name"`
		OperationName *string                                         `json:"operation_name"`
		ResourceName  *string                                         `json:"resource_name"`
		Service       *string                                         `json:"service"`
		Stat          *FormulaAndFunctionApmDependencyStatName        `json:"stat"`
	}{}
	all := struct {
		DataSource      FormulaAndFunctionApmDependencyStatsDataSource `json:"data_source"`
		Env             string                                         `json:"env"`
		IsUpstream      *bool                                          `json:"is_upstream,omitempty"`
		Name            string                                         `json:"name"`
		OperationName   string                                         `json:"operation_name"`
		PrimaryTagName  *string                                        `json:"primary_tag_name,omitempty"`
		PrimaryTagValue *string                                        `json:"primary_tag_value,omitempty"`
		ResourceName    string                                         `json:"resource_name"`
		Service         string                                         `json:"service"`
		Stat            FormulaAndFunctionApmDependencyStatName        `json:"stat"`
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
	if required.OperationName == nil {
		return fmt.Errorf("Required field operation_name missing")
	}
	if required.ResourceName == nil {
		return fmt.Errorf("Required field resource_name missing")
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
	o.IsUpstream = all.IsUpstream
	o.Name = all.Name
	o.OperationName = all.OperationName
	o.PrimaryTagName = all.PrimaryTagName
	o.PrimaryTagValue = all.PrimaryTagValue
	o.ResourceName = all.ResourceName
	o.Service = all.Service
	o.Stat = all.Stat
	return nil
}
