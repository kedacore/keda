// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// TimeseriesBackground Set a timeseries on the widget background.
type TimeseriesBackground struct {
	// Timeseries is made using an area or bars.
	Type TimeseriesBackgroundType `json:"type"`
	// Axis controls for the widget.
	Yaxis *WidgetAxis `json:"yaxis,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewTimeseriesBackground instantiates a new TimeseriesBackground object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewTimeseriesBackground(typeVar TimeseriesBackgroundType) *TimeseriesBackground {
	this := TimeseriesBackground{}
	this.Type = typeVar
	return &this
}

// NewTimeseriesBackgroundWithDefaults instantiates a new TimeseriesBackground object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewTimeseriesBackgroundWithDefaults() *TimeseriesBackground {
	this := TimeseriesBackground{}
	var typeVar TimeseriesBackgroundType = TIMESERIESBACKGROUNDTYPE_AREA
	this.Type = typeVar
	return &this
}

// GetType returns the Type field value.
func (o *TimeseriesBackground) GetType() TimeseriesBackgroundType {
	if o == nil {
		var ret TimeseriesBackgroundType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *TimeseriesBackground) GetTypeOk() (*TimeseriesBackgroundType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *TimeseriesBackground) SetType(v TimeseriesBackgroundType) {
	o.Type = v
}

// GetYaxis returns the Yaxis field value if set, zero value otherwise.
func (o *TimeseriesBackground) GetYaxis() WidgetAxis {
	if o == nil || o.Yaxis == nil {
		var ret WidgetAxis
		return ret
	}
	return *o.Yaxis
}

// GetYaxisOk returns a tuple with the Yaxis field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TimeseriesBackground) GetYaxisOk() (*WidgetAxis, bool) {
	if o == nil || o.Yaxis == nil {
		return nil, false
	}
	return o.Yaxis, true
}

// HasYaxis returns a boolean if a field has been set.
func (o *TimeseriesBackground) HasYaxis() bool {
	if o != nil && o.Yaxis != nil {
		return true
	}

	return false
}

// SetYaxis gets a reference to the given WidgetAxis and assigns it to the Yaxis field.
func (o *TimeseriesBackground) SetYaxis(v WidgetAxis) {
	o.Yaxis = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o TimeseriesBackground) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["type"] = o.Type
	if o.Yaxis != nil {
		toSerialize["yaxis"] = o.Yaxis
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *TimeseriesBackground) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Type *TimeseriesBackgroundType `json:"type"`
	}{}
	all := struct {
		Type  TimeseriesBackgroundType `json:"type"`
		Yaxis *WidgetAxis              `json:"yaxis,omitempty"`
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
	o.Type = all.Type
	if all.Yaxis != nil && all.Yaxis.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Yaxis = all.Yaxis
	return nil
}
