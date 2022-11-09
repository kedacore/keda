// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// NotebookRelativeTime Relative timeframe.
type NotebookRelativeTime struct {
	// The available timeframes depend on the widget you are using.
	LiveSpan WidgetLiveSpan `json:"live_span"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewNotebookRelativeTime instantiates a new NotebookRelativeTime object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewNotebookRelativeTime(liveSpan WidgetLiveSpan) *NotebookRelativeTime {
	this := NotebookRelativeTime{}
	this.LiveSpan = liveSpan
	return &this
}

// NewNotebookRelativeTimeWithDefaults instantiates a new NotebookRelativeTime object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewNotebookRelativeTimeWithDefaults() *NotebookRelativeTime {
	this := NotebookRelativeTime{}
	return &this
}

// GetLiveSpan returns the LiveSpan field value.
func (o *NotebookRelativeTime) GetLiveSpan() WidgetLiveSpan {
	if o == nil {
		var ret WidgetLiveSpan
		return ret
	}
	return o.LiveSpan
}

// GetLiveSpanOk returns a tuple with the LiveSpan field value
// and a boolean to check if the value has been set.
func (o *NotebookRelativeTime) GetLiveSpanOk() (*WidgetLiveSpan, bool) {
	if o == nil {
		return nil, false
	}
	return &o.LiveSpan, true
}

// SetLiveSpan sets field value.
func (o *NotebookRelativeTime) SetLiveSpan(v WidgetLiveSpan) {
	o.LiveSpan = v
}

// MarshalJSON serializes the struct using spec logic.
func (o NotebookRelativeTime) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["live_span"] = o.LiveSpan

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *NotebookRelativeTime) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		LiveSpan *WidgetLiveSpan `json:"live_span"`
	}{}
	all := struct {
		LiveSpan WidgetLiveSpan `json:"live_span"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.LiveSpan == nil {
		return fmt.Errorf("Required field live_span missing")
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
	if v := all.LiveSpan; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.LiveSpan = all.LiveSpan
	return nil
}

// NullableNotebookRelativeTime handles when a null is used for NotebookRelativeTime.
type NullableNotebookRelativeTime struct {
	value *NotebookRelativeTime
	isSet bool
}

// Get returns the associated value.
func (v NullableNotebookRelativeTime) Get() *NotebookRelativeTime {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableNotebookRelativeTime) Set(val *NotebookRelativeTime) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableNotebookRelativeTime) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableNotebookRelativeTime) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableNotebookRelativeTime initializes the struct as if Set has been called.
func NewNullableNotebookRelativeTime(val *NotebookRelativeTime) *NullableNotebookRelativeTime {
	return &NullableNotebookRelativeTime{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableNotebookRelativeTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableNotebookRelativeTime) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
