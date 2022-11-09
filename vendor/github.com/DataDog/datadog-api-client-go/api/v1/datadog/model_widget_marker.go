// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WidgetMarker Markers allow you to add visual conditional formatting for your graphs.
type WidgetMarker struct {
	// Combination of:
	//   - A severity error, warning, ok, or info
	//   - A line type: dashed, solid, or bold
	// In this case of a Distribution widget, this can be set to be `x_axis_percentile`.
	//
	DisplayType *string `json:"display_type,omitempty"`
	// Label to display over the marker.
	Label *string `json:"label,omitempty"`
	// Timestamp for the widget.
	Time *string `json:"time,omitempty"`
	// Value to apply. Can be a single value y = 15 or a range of values 0 < y < 10.
	Value string `json:"value"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewWidgetMarker instantiates a new WidgetMarker object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewWidgetMarker(value string) *WidgetMarker {
	this := WidgetMarker{}
	this.Value = value
	return &this
}

// NewWidgetMarkerWithDefaults instantiates a new WidgetMarker object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewWidgetMarkerWithDefaults() *WidgetMarker {
	this := WidgetMarker{}
	return &this
}

// GetDisplayType returns the DisplayType field value if set, zero value otherwise.
func (o *WidgetMarker) GetDisplayType() string {
	if o == nil || o.DisplayType == nil {
		var ret string
		return ret
	}
	return *o.DisplayType
}

// GetDisplayTypeOk returns a tuple with the DisplayType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WidgetMarker) GetDisplayTypeOk() (*string, bool) {
	if o == nil || o.DisplayType == nil {
		return nil, false
	}
	return o.DisplayType, true
}

// HasDisplayType returns a boolean if a field has been set.
func (o *WidgetMarker) HasDisplayType() bool {
	if o != nil && o.DisplayType != nil {
		return true
	}

	return false
}

// SetDisplayType gets a reference to the given string and assigns it to the DisplayType field.
func (o *WidgetMarker) SetDisplayType(v string) {
	o.DisplayType = &v
}

// GetLabel returns the Label field value if set, zero value otherwise.
func (o *WidgetMarker) GetLabel() string {
	if o == nil || o.Label == nil {
		var ret string
		return ret
	}
	return *o.Label
}

// GetLabelOk returns a tuple with the Label field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WidgetMarker) GetLabelOk() (*string, bool) {
	if o == nil || o.Label == nil {
		return nil, false
	}
	return o.Label, true
}

// HasLabel returns a boolean if a field has been set.
func (o *WidgetMarker) HasLabel() bool {
	if o != nil && o.Label != nil {
		return true
	}

	return false
}

// SetLabel gets a reference to the given string and assigns it to the Label field.
func (o *WidgetMarker) SetLabel(v string) {
	o.Label = &v
}

// GetTime returns the Time field value if set, zero value otherwise.
func (o *WidgetMarker) GetTime() string {
	if o == nil || o.Time == nil {
		var ret string
		return ret
	}
	return *o.Time
}

// GetTimeOk returns a tuple with the Time field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WidgetMarker) GetTimeOk() (*string, bool) {
	if o == nil || o.Time == nil {
		return nil, false
	}
	return o.Time, true
}

// HasTime returns a boolean if a field has been set.
func (o *WidgetMarker) HasTime() bool {
	if o != nil && o.Time != nil {
		return true
	}

	return false
}

// SetTime gets a reference to the given string and assigns it to the Time field.
func (o *WidgetMarker) SetTime(v string) {
	o.Time = &v
}

// GetValue returns the Value field value.
func (o *WidgetMarker) GetValue() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Value
}

// GetValueOk returns a tuple with the Value field value
// and a boolean to check if the value has been set.
func (o *WidgetMarker) GetValueOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Value, true
}

// SetValue sets field value.
func (o *WidgetMarker) SetValue(v string) {
	o.Value = v
}

// MarshalJSON serializes the struct using spec logic.
func (o WidgetMarker) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.DisplayType != nil {
		toSerialize["display_type"] = o.DisplayType
	}
	if o.Label != nil {
		toSerialize["label"] = o.Label
	}
	if o.Time != nil {
		toSerialize["time"] = o.Time
	}
	toSerialize["value"] = o.Value

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *WidgetMarker) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Value *string `json:"value"`
	}{}
	all := struct {
		DisplayType *string `json:"display_type,omitempty"`
		Label       *string `json:"label,omitempty"`
		Time        *string `json:"time,omitempty"`
		Value       string  `json:"value"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Value == nil {
		return fmt.Errorf("Required field value missing")
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
	o.DisplayType = all.DisplayType
	o.Label = all.Label
	o.Time = all.Time
	o.Value = all.Value
	return nil
}
