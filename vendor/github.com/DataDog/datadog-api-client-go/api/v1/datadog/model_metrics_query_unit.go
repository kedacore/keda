// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// MetricsQueryUnit Object containing the metric unit family, scale factor, name, and short name.
type MetricsQueryUnit struct {
	// Unit family, allows for conversion between units of the same family, for scaling.
	Family *string `json:"family,omitempty"`
	// Unit name
	Name *string `json:"name,omitempty"`
	// Plural form of the unit name.
	Plural *string `json:"plural,omitempty"`
	// Factor for scaling between units of the same family.
	ScaleFactor *float64 `json:"scale_factor,omitempty"`
	// Abbreviation of the unit.
	ShortName *string `json:"short_name,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewMetricsQueryUnit instantiates a new MetricsQueryUnit object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMetricsQueryUnit() *MetricsQueryUnit {
	this := MetricsQueryUnit{}
	return &this
}

// NewMetricsQueryUnitWithDefaults instantiates a new MetricsQueryUnit object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMetricsQueryUnitWithDefaults() *MetricsQueryUnit {
	this := MetricsQueryUnit{}
	return &this
}

// GetFamily returns the Family field value if set, zero value otherwise.
func (o *MetricsQueryUnit) GetFamily() string {
	if o == nil || o.Family == nil {
		var ret string
		return ret
	}
	return *o.Family
}

// GetFamilyOk returns a tuple with the Family field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryUnit) GetFamilyOk() (*string, bool) {
	if o == nil || o.Family == nil {
		return nil, false
	}
	return o.Family, true
}

// HasFamily returns a boolean if a field has been set.
func (o *MetricsQueryUnit) HasFamily() bool {
	if o != nil && o.Family != nil {
		return true
	}

	return false
}

// SetFamily gets a reference to the given string and assigns it to the Family field.
func (o *MetricsQueryUnit) SetFamily(v string) {
	o.Family = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *MetricsQueryUnit) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryUnit) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *MetricsQueryUnit) HasName() bool {
	if o != nil && o.Name != nil {
		return true
	}

	return false
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *MetricsQueryUnit) SetName(v string) {
	o.Name = &v
}

// GetPlural returns the Plural field value if set, zero value otherwise.
func (o *MetricsQueryUnit) GetPlural() string {
	if o == nil || o.Plural == nil {
		var ret string
		return ret
	}
	return *o.Plural
}

// GetPluralOk returns a tuple with the Plural field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryUnit) GetPluralOk() (*string, bool) {
	if o == nil || o.Plural == nil {
		return nil, false
	}
	return o.Plural, true
}

// HasPlural returns a boolean if a field has been set.
func (o *MetricsQueryUnit) HasPlural() bool {
	if o != nil && o.Plural != nil {
		return true
	}

	return false
}

// SetPlural gets a reference to the given string and assigns it to the Plural field.
func (o *MetricsQueryUnit) SetPlural(v string) {
	o.Plural = &v
}

// GetScaleFactor returns the ScaleFactor field value if set, zero value otherwise.
func (o *MetricsQueryUnit) GetScaleFactor() float64 {
	if o == nil || o.ScaleFactor == nil {
		var ret float64
		return ret
	}
	return *o.ScaleFactor
}

// GetScaleFactorOk returns a tuple with the ScaleFactor field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryUnit) GetScaleFactorOk() (*float64, bool) {
	if o == nil || o.ScaleFactor == nil {
		return nil, false
	}
	return o.ScaleFactor, true
}

// HasScaleFactor returns a boolean if a field has been set.
func (o *MetricsQueryUnit) HasScaleFactor() bool {
	if o != nil && o.ScaleFactor != nil {
		return true
	}

	return false
}

// SetScaleFactor gets a reference to the given float64 and assigns it to the ScaleFactor field.
func (o *MetricsQueryUnit) SetScaleFactor(v float64) {
	o.ScaleFactor = &v
}

// GetShortName returns the ShortName field value if set, zero value otherwise.
func (o *MetricsQueryUnit) GetShortName() string {
	if o == nil || o.ShortName == nil {
		var ret string
		return ret
	}
	return *o.ShortName
}

// GetShortNameOk returns a tuple with the ShortName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricsQueryUnit) GetShortNameOk() (*string, bool) {
	if o == nil || o.ShortName == nil {
		return nil, false
	}
	return o.ShortName, true
}

// HasShortName returns a boolean if a field has been set.
func (o *MetricsQueryUnit) HasShortName() bool {
	if o != nil && o.ShortName != nil {
		return true
	}

	return false
}

// SetShortName gets a reference to the given string and assigns it to the ShortName field.
func (o *MetricsQueryUnit) SetShortName(v string) {
	o.ShortName = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o MetricsQueryUnit) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Family != nil {
		toSerialize["family"] = o.Family
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.Plural != nil {
		toSerialize["plural"] = o.Plural
	}
	if o.ScaleFactor != nil {
		toSerialize["scale_factor"] = o.ScaleFactor
	}
	if o.ShortName != nil {
		toSerialize["short_name"] = o.ShortName
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MetricsQueryUnit) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Family      *string  `json:"family,omitempty"`
		Name        *string  `json:"name,omitempty"`
		Plural      *string  `json:"plural,omitempty"`
		ScaleFactor *float64 `json:"scale_factor,omitempty"`
		ShortName   *string  `json:"short_name,omitempty"`
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
	o.Family = all.Family
	o.Name = all.Name
	o.Plural = all.Plural
	o.ScaleFactor = all.ScaleFactor
	o.ShortName = all.ShortName
	return nil
}

// NullableMetricsQueryUnit handles when a null is used for MetricsQueryUnit.
type NullableMetricsQueryUnit struct {
	value *MetricsQueryUnit
	isSet bool
}

// Get returns the associated value.
func (v NullableMetricsQueryUnit) Get() *MetricsQueryUnit {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableMetricsQueryUnit) Set(val *MetricsQueryUnit) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableMetricsQueryUnit) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableMetricsQueryUnit) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableMetricsQueryUnit initializes the struct as if Set has been called.
func NewNullableMetricsQueryUnit(val *MetricsQueryUnit) *NullableMetricsQueryUnit {
	return &NullableMetricsQueryUnit{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableMetricsQueryUnit) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableMetricsQueryUnit) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
