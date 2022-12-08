// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// DashboardTemplateVariable Template variable.
type DashboardTemplateVariable struct {
	// The list of values that the template variable drop-down is limited to.
	AvailableValues []string `json:"available_values,omitempty"`
	// The default value for the template variable on dashboard load.
	Default NullableString `json:"default,omitempty"`
	// The name of the variable.
	Name string `json:"name"`
	// The tag prefix associated with the variable. Only tags with this prefix appear in the variable drop-down.
	Prefix NullableString `json:"prefix,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewDashboardTemplateVariable instantiates a new DashboardTemplateVariable object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewDashboardTemplateVariable(name string) *DashboardTemplateVariable {
	this := DashboardTemplateVariable{}
	this.Name = name
	return &this
}

// NewDashboardTemplateVariableWithDefaults instantiates a new DashboardTemplateVariable object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewDashboardTemplateVariableWithDefaults() *DashboardTemplateVariable {
	this := DashboardTemplateVariable{}
	return &this
}

// GetAvailableValues returns the AvailableValues field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *DashboardTemplateVariable) GetAvailableValues() []string {
	if o == nil {
		var ret []string
		return ret
	}
	return o.AvailableValues
}

// GetAvailableValuesOk returns a tuple with the AvailableValues field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *DashboardTemplateVariable) GetAvailableValuesOk() (*[]string, bool) {
	if o == nil || o.AvailableValues == nil {
		return nil, false
	}
	return &o.AvailableValues, true
}

// HasAvailableValues returns a boolean if a field has been set.
func (o *DashboardTemplateVariable) HasAvailableValues() bool {
	if o != nil && o.AvailableValues != nil {
		return true
	}

	return false
}

// SetAvailableValues gets a reference to the given []string and assigns it to the AvailableValues field.
func (o *DashboardTemplateVariable) SetAvailableValues(v []string) {
	o.AvailableValues = v
}

// GetDefault returns the Default field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *DashboardTemplateVariable) GetDefault() string {
	if o == nil || o.Default.Get() == nil {
		var ret string
		return ret
	}
	return *o.Default.Get()
}

// GetDefaultOk returns a tuple with the Default field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *DashboardTemplateVariable) GetDefaultOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.Default.Get(), o.Default.IsSet()
}

// HasDefault returns a boolean if a field has been set.
func (o *DashboardTemplateVariable) HasDefault() bool {
	if o != nil && o.Default.IsSet() {
		return true
	}

	return false
}

// SetDefault gets a reference to the given NullableString and assigns it to the Default field.
func (o *DashboardTemplateVariable) SetDefault(v string) {
	o.Default.Set(&v)
}

// SetDefaultNil sets the value for Default to be an explicit nil.
func (o *DashboardTemplateVariable) SetDefaultNil() {
	o.Default.Set(nil)
}

// UnsetDefault ensures that no value is present for Default, not even an explicit nil.
func (o *DashboardTemplateVariable) UnsetDefault() {
	o.Default.Unset()
}

// GetName returns the Name field value.
func (o *DashboardTemplateVariable) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *DashboardTemplateVariable) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *DashboardTemplateVariable) SetName(v string) {
	o.Name = v
}

// GetPrefix returns the Prefix field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *DashboardTemplateVariable) GetPrefix() string {
	if o == nil || o.Prefix.Get() == nil {
		var ret string
		return ret
	}
	return *o.Prefix.Get()
}

// GetPrefixOk returns a tuple with the Prefix field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *DashboardTemplateVariable) GetPrefixOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.Prefix.Get(), o.Prefix.IsSet()
}

// HasPrefix returns a boolean if a field has been set.
func (o *DashboardTemplateVariable) HasPrefix() bool {
	if o != nil && o.Prefix.IsSet() {
		return true
	}

	return false
}

// SetPrefix gets a reference to the given NullableString and assigns it to the Prefix field.
func (o *DashboardTemplateVariable) SetPrefix(v string) {
	o.Prefix.Set(&v)
}

// SetPrefixNil sets the value for Prefix to be an explicit nil.
func (o *DashboardTemplateVariable) SetPrefixNil() {
	o.Prefix.Set(nil)
}

// UnsetPrefix ensures that no value is present for Prefix, not even an explicit nil.
func (o *DashboardTemplateVariable) UnsetPrefix() {
	o.Prefix.Unset()
}

// MarshalJSON serializes the struct using spec logic.
func (o DashboardTemplateVariable) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AvailableValues != nil {
		toSerialize["available_values"] = o.AvailableValues
	}
	if o.Default.IsSet() {
		toSerialize["default"] = o.Default.Get()
	}
	toSerialize["name"] = o.Name
	if o.Prefix.IsSet() {
		toSerialize["prefix"] = o.Prefix.Get()
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *DashboardTemplateVariable) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Name *string `json:"name"`
	}{}
	all := struct {
		AvailableValues []string       `json:"available_values,omitempty"`
		Default         NullableString `json:"default,omitempty"`
		Name            string         `json:"name"`
		Prefix          NullableString `json:"prefix,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
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
	o.AvailableValues = all.AvailableValues
	o.Default = all.Default
	o.Name = all.Name
	o.Prefix = all.Prefix
	return nil
}
