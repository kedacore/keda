// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SLOHistoryMetricsSeriesMetadataUnit An Object of metric units.
type SLOHistoryMetricsSeriesMetadataUnit struct {
	// The family of metric unit, for example `bytes` is the family for `kibibyte`, `byte`, and `bit` units.
	Family *string `json:"family,omitempty"`
	// The ID of the metric unit.
	Id *int64 `json:"id,omitempty"`
	// The unit of the metric, for instance `byte`.
	Name *string `json:"name,omitempty"`
	// The plural Unit of metric, for instance `bytes`.
	Plural NullableString `json:"plural,omitempty"`
	// The scale factor of metric unit, for instance `1.0`.
	ScaleFactor *float64 `json:"scale_factor,omitempty"`
	// A shorter and abbreviated version of the metric unit, for instance `B`.
	ShortName NullableString `json:"short_name,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSLOHistoryMetricsSeriesMetadataUnit instantiates a new SLOHistoryMetricsSeriesMetadataUnit object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSLOHistoryMetricsSeriesMetadataUnit() *SLOHistoryMetricsSeriesMetadataUnit {
	this := SLOHistoryMetricsSeriesMetadataUnit{}
	return &this
}

// NewSLOHistoryMetricsSeriesMetadataUnitWithDefaults instantiates a new SLOHistoryMetricsSeriesMetadataUnit object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSLOHistoryMetricsSeriesMetadataUnitWithDefaults() *SLOHistoryMetricsSeriesMetadataUnit {
	this := SLOHistoryMetricsSeriesMetadataUnit{}
	return &this
}

// GetFamily returns the Family field value if set, zero value otherwise.
func (o *SLOHistoryMetricsSeriesMetadataUnit) GetFamily() string {
	if o == nil || o.Family == nil {
		var ret string
		return ret
	}
	return *o.Family
}

// GetFamilyOk returns a tuple with the Family field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryMetricsSeriesMetadataUnit) GetFamilyOk() (*string, bool) {
	if o == nil || o.Family == nil {
		return nil, false
	}
	return o.Family, true
}

// HasFamily returns a boolean if a field has been set.
func (o *SLOHistoryMetricsSeriesMetadataUnit) HasFamily() bool {
	if o != nil && o.Family != nil {
		return true
	}

	return false
}

// SetFamily gets a reference to the given string and assigns it to the Family field.
func (o *SLOHistoryMetricsSeriesMetadataUnit) SetFamily(v string) {
	o.Family = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *SLOHistoryMetricsSeriesMetadataUnit) GetId() int64 {
	if o == nil || o.Id == nil {
		var ret int64
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryMetricsSeriesMetadataUnit) GetIdOk() (*int64, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *SLOHistoryMetricsSeriesMetadataUnit) HasId() bool {
	if o != nil && o.Id != nil {
		return true
	}

	return false
}

// SetId gets a reference to the given int64 and assigns it to the Id field.
func (o *SLOHistoryMetricsSeriesMetadataUnit) SetId(v int64) {
	o.Id = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *SLOHistoryMetricsSeriesMetadataUnit) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryMetricsSeriesMetadataUnit) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *SLOHistoryMetricsSeriesMetadataUnit) HasName() bool {
	if o != nil && o.Name != nil {
		return true
	}

	return false
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *SLOHistoryMetricsSeriesMetadataUnit) SetName(v string) {
	o.Name = &v
}

// GetPlural returns the Plural field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SLOHistoryMetricsSeriesMetadataUnit) GetPlural() string {
	if o == nil || o.Plural.Get() == nil {
		var ret string
		return ret
	}
	return *o.Plural.Get()
}

// GetPluralOk returns a tuple with the Plural field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SLOHistoryMetricsSeriesMetadataUnit) GetPluralOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.Plural.Get(), o.Plural.IsSet()
}

// HasPlural returns a boolean if a field has been set.
func (o *SLOHistoryMetricsSeriesMetadataUnit) HasPlural() bool {
	if o != nil && o.Plural.IsSet() {
		return true
	}

	return false
}

// SetPlural gets a reference to the given NullableString and assigns it to the Plural field.
func (o *SLOHistoryMetricsSeriesMetadataUnit) SetPlural(v string) {
	o.Plural.Set(&v)
}

// SetPluralNil sets the value for Plural to be an explicit nil.
func (o *SLOHistoryMetricsSeriesMetadataUnit) SetPluralNil() {
	o.Plural.Set(nil)
}

// UnsetPlural ensures that no value is present for Plural, not even an explicit nil.
func (o *SLOHistoryMetricsSeriesMetadataUnit) UnsetPlural() {
	o.Plural.Unset()
}

// GetScaleFactor returns the ScaleFactor field value if set, zero value otherwise.
func (o *SLOHistoryMetricsSeriesMetadataUnit) GetScaleFactor() float64 {
	if o == nil || o.ScaleFactor == nil {
		var ret float64
		return ret
	}
	return *o.ScaleFactor
}

// GetScaleFactorOk returns a tuple with the ScaleFactor field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOHistoryMetricsSeriesMetadataUnit) GetScaleFactorOk() (*float64, bool) {
	if o == nil || o.ScaleFactor == nil {
		return nil, false
	}
	return o.ScaleFactor, true
}

// HasScaleFactor returns a boolean if a field has been set.
func (o *SLOHistoryMetricsSeriesMetadataUnit) HasScaleFactor() bool {
	if o != nil && o.ScaleFactor != nil {
		return true
	}

	return false
}

// SetScaleFactor gets a reference to the given float64 and assigns it to the ScaleFactor field.
func (o *SLOHistoryMetricsSeriesMetadataUnit) SetScaleFactor(v float64) {
	o.ScaleFactor = &v
}

// GetShortName returns the ShortName field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SLOHistoryMetricsSeriesMetadataUnit) GetShortName() string {
	if o == nil || o.ShortName.Get() == nil {
		var ret string
		return ret
	}
	return *o.ShortName.Get()
}

// GetShortNameOk returns a tuple with the ShortName field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SLOHistoryMetricsSeriesMetadataUnit) GetShortNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.ShortName.Get(), o.ShortName.IsSet()
}

// HasShortName returns a boolean if a field has been set.
func (o *SLOHistoryMetricsSeriesMetadataUnit) HasShortName() bool {
	if o != nil && o.ShortName.IsSet() {
		return true
	}

	return false
}

// SetShortName gets a reference to the given NullableString and assigns it to the ShortName field.
func (o *SLOHistoryMetricsSeriesMetadataUnit) SetShortName(v string) {
	o.ShortName.Set(&v)
}

// SetShortNameNil sets the value for ShortName to be an explicit nil.
func (o *SLOHistoryMetricsSeriesMetadataUnit) SetShortNameNil() {
	o.ShortName.Set(nil)
}

// UnsetShortName ensures that no value is present for ShortName, not even an explicit nil.
func (o *SLOHistoryMetricsSeriesMetadataUnit) UnsetShortName() {
	o.ShortName.Unset()
}

// MarshalJSON serializes the struct using spec logic.
func (o SLOHistoryMetricsSeriesMetadataUnit) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Family != nil {
		toSerialize["family"] = o.Family
	}
	if o.Id != nil {
		toSerialize["id"] = o.Id
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.Plural.IsSet() {
		toSerialize["plural"] = o.Plural.Get()
	}
	if o.ScaleFactor != nil {
		toSerialize["scale_factor"] = o.ScaleFactor
	}
	if o.ShortName.IsSet() {
		toSerialize["short_name"] = o.ShortName.Get()
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SLOHistoryMetricsSeriesMetadataUnit) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Family      *string        `json:"family,omitempty"`
		Id          *int64         `json:"id,omitempty"`
		Name        *string        `json:"name,omitempty"`
		Plural      NullableString `json:"plural,omitempty"`
		ScaleFactor *float64       `json:"scale_factor,omitempty"`
		ShortName   NullableString `json:"short_name,omitempty"`
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
	o.Id = all.Id
	o.Name = all.Name
	o.Plural = all.Plural
	o.ScaleFactor = all.ScaleFactor
	o.ShortName = all.ShortName
	return nil
}

// NullableSLOHistoryMetricsSeriesMetadataUnit handles when a null is used for SLOHistoryMetricsSeriesMetadataUnit.
type NullableSLOHistoryMetricsSeriesMetadataUnit struct {
	value *SLOHistoryMetricsSeriesMetadataUnit
	isSet bool
}

// Get returns the associated value.
func (v NullableSLOHistoryMetricsSeriesMetadataUnit) Get() *SLOHistoryMetricsSeriesMetadataUnit {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSLOHistoryMetricsSeriesMetadataUnit) Set(val *SLOHistoryMetricsSeriesMetadataUnit) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSLOHistoryMetricsSeriesMetadataUnit) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableSLOHistoryMetricsSeriesMetadataUnit) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSLOHistoryMetricsSeriesMetadataUnit initializes the struct as if Set has been called.
func NewNullableSLOHistoryMetricsSeriesMetadataUnit(val *SLOHistoryMetricsSeriesMetadataUnit) *NullableSLOHistoryMetricsSeriesMetadataUnit {
	return &NullableSLOHistoryMetricsSeriesMetadataUnit{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSLOHistoryMetricsSeriesMetadataUnit) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSLOHistoryMetricsSeriesMetadataUnit) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
