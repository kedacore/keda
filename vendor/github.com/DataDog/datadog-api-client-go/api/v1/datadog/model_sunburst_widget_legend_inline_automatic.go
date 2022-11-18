// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SunburstWidgetLegendInlineAutomatic Configuration of inline or automatic legends.
type SunburstWidgetLegendInlineAutomatic struct {
	// Whether to hide the percentages of the groups.
	HidePercent *bool `json:"hide_percent,omitempty"`
	// Whether to hide the values of the groups.
	HideValue *bool `json:"hide_value,omitempty"`
	// Whether to show the legend inline or let it be automatically generated.
	Type SunburstWidgetLegendInlineAutomaticType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSunburstWidgetLegendInlineAutomatic instantiates a new SunburstWidgetLegendInlineAutomatic object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSunburstWidgetLegendInlineAutomatic(typeVar SunburstWidgetLegendInlineAutomaticType) *SunburstWidgetLegendInlineAutomatic {
	this := SunburstWidgetLegendInlineAutomatic{}
	this.Type = typeVar
	return &this
}

// NewSunburstWidgetLegendInlineAutomaticWithDefaults instantiates a new SunburstWidgetLegendInlineAutomatic object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSunburstWidgetLegendInlineAutomaticWithDefaults() *SunburstWidgetLegendInlineAutomatic {
	this := SunburstWidgetLegendInlineAutomatic{}
	return &this
}

// GetHidePercent returns the HidePercent field value if set, zero value otherwise.
func (o *SunburstWidgetLegendInlineAutomatic) GetHidePercent() bool {
	if o == nil || o.HidePercent == nil {
		var ret bool
		return ret
	}
	return *o.HidePercent
}

// GetHidePercentOk returns a tuple with the HidePercent field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SunburstWidgetLegendInlineAutomatic) GetHidePercentOk() (*bool, bool) {
	if o == nil || o.HidePercent == nil {
		return nil, false
	}
	return o.HidePercent, true
}

// HasHidePercent returns a boolean if a field has been set.
func (o *SunburstWidgetLegendInlineAutomatic) HasHidePercent() bool {
	if o != nil && o.HidePercent != nil {
		return true
	}

	return false
}

// SetHidePercent gets a reference to the given bool and assigns it to the HidePercent field.
func (o *SunburstWidgetLegendInlineAutomatic) SetHidePercent(v bool) {
	o.HidePercent = &v
}

// GetHideValue returns the HideValue field value if set, zero value otherwise.
func (o *SunburstWidgetLegendInlineAutomatic) GetHideValue() bool {
	if o == nil || o.HideValue == nil {
		var ret bool
		return ret
	}
	return *o.HideValue
}

// GetHideValueOk returns a tuple with the HideValue field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SunburstWidgetLegendInlineAutomatic) GetHideValueOk() (*bool, bool) {
	if o == nil || o.HideValue == nil {
		return nil, false
	}
	return o.HideValue, true
}

// HasHideValue returns a boolean if a field has been set.
func (o *SunburstWidgetLegendInlineAutomatic) HasHideValue() bool {
	if o != nil && o.HideValue != nil {
		return true
	}

	return false
}

// SetHideValue gets a reference to the given bool and assigns it to the HideValue field.
func (o *SunburstWidgetLegendInlineAutomatic) SetHideValue(v bool) {
	o.HideValue = &v
}

// GetType returns the Type field value.
func (o *SunburstWidgetLegendInlineAutomatic) GetType() SunburstWidgetLegendInlineAutomaticType {
	if o == nil {
		var ret SunburstWidgetLegendInlineAutomaticType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *SunburstWidgetLegendInlineAutomatic) GetTypeOk() (*SunburstWidgetLegendInlineAutomaticType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *SunburstWidgetLegendInlineAutomatic) SetType(v SunburstWidgetLegendInlineAutomaticType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SunburstWidgetLegendInlineAutomatic) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.HidePercent != nil {
		toSerialize["hide_percent"] = o.HidePercent
	}
	if o.HideValue != nil {
		toSerialize["hide_value"] = o.HideValue
	}
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SunburstWidgetLegendInlineAutomatic) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Type *SunburstWidgetLegendInlineAutomaticType `json:"type"`
	}{}
	all := struct {
		HidePercent *bool                                   `json:"hide_percent,omitempty"`
		HideValue   *bool                                   `json:"hide_value,omitempty"`
		Type        SunburstWidgetLegendInlineAutomaticType `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Type == nil {
		return fmt.Errorf("Required field type missing")
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
	if v := all.Type; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.HidePercent = all.HidePercent
	o.HideValue = all.HideValue
	o.Type = all.Type
	return nil
}
