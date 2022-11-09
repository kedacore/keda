// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// WidgetTime Time setting for the widget.
type WidgetTime struct {
	// The available timeframes depend on the widget you are using.
	LiveSpan *WidgetLiveSpan `json:"live_span,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewWidgetTime instantiates a new WidgetTime object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewWidgetTime() *WidgetTime {
	this := WidgetTime{}
	return &this
}

// NewWidgetTimeWithDefaults instantiates a new WidgetTime object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewWidgetTimeWithDefaults() *WidgetTime {
	this := WidgetTime{}
	return &this
}

// GetLiveSpan returns the LiveSpan field value if set, zero value otherwise.
func (o *WidgetTime) GetLiveSpan() WidgetLiveSpan {
	if o == nil || o.LiveSpan == nil {
		var ret WidgetLiveSpan
		return ret
	}
	return *o.LiveSpan
}

// GetLiveSpanOk returns a tuple with the LiveSpan field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WidgetTime) GetLiveSpanOk() (*WidgetLiveSpan, bool) {
	if o == nil || o.LiveSpan == nil {
		return nil, false
	}
	return o.LiveSpan, true
}

// HasLiveSpan returns a boolean if a field has been set.
func (o *WidgetTime) HasLiveSpan() bool {
	if o != nil && o.LiveSpan != nil {
		return true
	}

	return false
}

// SetLiveSpan gets a reference to the given WidgetLiveSpan and assigns it to the LiveSpan field.
func (o *WidgetTime) SetLiveSpan(v WidgetLiveSpan) {
	o.LiveSpan = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o WidgetTime) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.LiveSpan != nil {
		toSerialize["live_span"] = o.LiveSpan
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *WidgetTime) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		LiveSpan *WidgetLiveSpan `json:"live_span,omitempty"`
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
	if v := all.LiveSpan; v != nil && !v.IsValid() {
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
