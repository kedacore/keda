// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// ApmStatsQueryDefinition The APM stats query for table and distributions widgets.
type ApmStatsQueryDefinition struct {
	// Column properties used by the front end for display.
	Columns []ApmStatsQueryColumnType `json:"columns,omitempty"`
	// Environment name.
	Env string `json:"env"`
	// Operation name associated with service.
	Name string `json:"name"`
	// The organization's host group name and value.
	PrimaryTag string `json:"primary_tag"`
	// Resource name.
	Resource *string `json:"resource,omitempty"`
	// The level of detail for the request.
	RowType ApmStatsQueryRowType `json:"row_type"`
	// Service name.
	Service string `json:"service"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewApmStatsQueryDefinition instantiates a new ApmStatsQueryDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewApmStatsQueryDefinition(env string, name string, primaryTag string, rowType ApmStatsQueryRowType, service string) *ApmStatsQueryDefinition {
	this := ApmStatsQueryDefinition{}
	this.Env = env
	this.Name = name
	this.PrimaryTag = primaryTag
	this.RowType = rowType
	this.Service = service
	return &this
}

// NewApmStatsQueryDefinitionWithDefaults instantiates a new ApmStatsQueryDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewApmStatsQueryDefinitionWithDefaults() *ApmStatsQueryDefinition {
	this := ApmStatsQueryDefinition{}
	return &this
}

// GetColumns returns the Columns field value if set, zero value otherwise.
func (o *ApmStatsQueryDefinition) GetColumns() []ApmStatsQueryColumnType {
	if o == nil || o.Columns == nil {
		var ret []ApmStatsQueryColumnType
		return ret
	}
	return o.Columns
}

// GetColumnsOk returns a tuple with the Columns field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ApmStatsQueryDefinition) GetColumnsOk() (*[]ApmStatsQueryColumnType, bool) {
	if o == nil || o.Columns == nil {
		return nil, false
	}
	return &o.Columns, true
}

// HasColumns returns a boolean if a field has been set.
func (o *ApmStatsQueryDefinition) HasColumns() bool {
	if o != nil && o.Columns != nil {
		return true
	}

	return false
}

// SetColumns gets a reference to the given []ApmStatsQueryColumnType and assigns it to the Columns field.
func (o *ApmStatsQueryDefinition) SetColumns(v []ApmStatsQueryColumnType) {
	o.Columns = v
}

// GetEnv returns the Env field value.
func (o *ApmStatsQueryDefinition) GetEnv() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Env
}

// GetEnvOk returns a tuple with the Env field value
// and a boolean to check if the value has been set.
func (o *ApmStatsQueryDefinition) GetEnvOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Env, true
}

// SetEnv sets field value.
func (o *ApmStatsQueryDefinition) SetEnv(v string) {
	o.Env = v
}

// GetName returns the Name field value.
func (o *ApmStatsQueryDefinition) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *ApmStatsQueryDefinition) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *ApmStatsQueryDefinition) SetName(v string) {
	o.Name = v
}

// GetPrimaryTag returns the PrimaryTag field value.
func (o *ApmStatsQueryDefinition) GetPrimaryTag() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.PrimaryTag
}

// GetPrimaryTagOk returns a tuple with the PrimaryTag field value
// and a boolean to check if the value has been set.
func (o *ApmStatsQueryDefinition) GetPrimaryTagOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.PrimaryTag, true
}

// SetPrimaryTag sets field value.
func (o *ApmStatsQueryDefinition) SetPrimaryTag(v string) {
	o.PrimaryTag = v
}

// GetResource returns the Resource field value if set, zero value otherwise.
func (o *ApmStatsQueryDefinition) GetResource() string {
	if o == nil || o.Resource == nil {
		var ret string
		return ret
	}
	return *o.Resource
}

// GetResourceOk returns a tuple with the Resource field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ApmStatsQueryDefinition) GetResourceOk() (*string, bool) {
	if o == nil || o.Resource == nil {
		return nil, false
	}
	return o.Resource, true
}

// HasResource returns a boolean if a field has been set.
func (o *ApmStatsQueryDefinition) HasResource() bool {
	if o != nil && o.Resource != nil {
		return true
	}

	return false
}

// SetResource gets a reference to the given string and assigns it to the Resource field.
func (o *ApmStatsQueryDefinition) SetResource(v string) {
	o.Resource = &v
}

// GetRowType returns the RowType field value.
func (o *ApmStatsQueryDefinition) GetRowType() ApmStatsQueryRowType {
	if o == nil {
		var ret ApmStatsQueryRowType
		return ret
	}
	return o.RowType
}

// GetRowTypeOk returns a tuple with the RowType field value
// and a boolean to check if the value has been set.
func (o *ApmStatsQueryDefinition) GetRowTypeOk() (*ApmStatsQueryRowType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.RowType, true
}

// SetRowType sets field value.
func (o *ApmStatsQueryDefinition) SetRowType(v ApmStatsQueryRowType) {
	o.RowType = v
}

// GetService returns the Service field value.
func (o *ApmStatsQueryDefinition) GetService() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Service
}

// GetServiceOk returns a tuple with the Service field value
// and a boolean to check if the value has been set.
func (o *ApmStatsQueryDefinition) GetServiceOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Service, true
}

// SetService sets field value.
func (o *ApmStatsQueryDefinition) SetService(v string) {
	o.Service = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ApmStatsQueryDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Columns != nil {
		toSerialize["columns"] = o.Columns
	}
	toSerialize["env"] = o.Env
	toSerialize["name"] = o.Name
	toSerialize["primary_tag"] = o.PrimaryTag
	if o.Resource != nil {
		toSerialize["resource"] = o.Resource
	}
	toSerialize["row_type"] = o.RowType
	toSerialize["service"] = o.Service

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ApmStatsQueryDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Env        *string               `json:"env"`
		Name       *string               `json:"name"`
		PrimaryTag *string               `json:"primary_tag"`
		RowType    *ApmStatsQueryRowType `json:"row_type"`
		Service    *string               `json:"service"`
	}{}
	all := struct {
		Columns    []ApmStatsQueryColumnType `json:"columns,omitempty"`
		Env        string                    `json:"env"`
		Name       string                    `json:"name"`
		PrimaryTag string                    `json:"primary_tag"`
		Resource   *string                   `json:"resource,omitempty"`
		RowType    ApmStatsQueryRowType      `json:"row_type"`
		Service    string                    `json:"service"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Env == nil {
		return fmt.Errorf("Required field env missing")
	}
	if required.Name == nil {
		return fmt.Errorf("Required field name missing")
	}
	if required.PrimaryTag == nil {
		return fmt.Errorf("Required field primary_tag missing")
	}
	if required.RowType == nil {
		return fmt.Errorf("Required field row_type missing")
	}
	if required.Service == nil {
		return fmt.Errorf("Required field service missing")
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
	if v := all.RowType; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Columns = all.Columns
	o.Env = all.Env
	o.Name = all.Name
	o.PrimaryTag = all.PrimaryTag
	o.Resource = all.Resource
	o.RowType = all.RowType
	o.Service = all.Service
	return nil
}
