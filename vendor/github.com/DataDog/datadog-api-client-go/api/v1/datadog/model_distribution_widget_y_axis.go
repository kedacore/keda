// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// DistributionWidgetYAxis Y Axis controls for the distribution widget.
type DistributionWidgetYAxis struct {
	// True includes zero.
	IncludeZero *bool `json:"include_zero,omitempty"`
	// The label of the axis to display on the graph.
	Label *string `json:"label,omitempty"`
	// Specifies the maximum value to show on the y-axis. It takes a number, or auto for default behavior.
	Max *string `json:"max,omitempty"`
	// Specifies minimum value to show on the y-axis. It takes a number, or auto for default behavior.
	Min *string `json:"min,omitempty"`
	// Specifies the scale type. Possible values are `linear` or `log`.
	Scale *string `json:"scale,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewDistributionWidgetYAxis instantiates a new DistributionWidgetYAxis object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewDistributionWidgetYAxis() *DistributionWidgetYAxis {
	this := DistributionWidgetYAxis{}
	var max string = "auto"
	this.Max = &max
	var min string = "auto"
	this.Min = &min
	var scale string = "linear"
	this.Scale = &scale
	return &this
}

// NewDistributionWidgetYAxisWithDefaults instantiates a new DistributionWidgetYAxis object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewDistributionWidgetYAxisWithDefaults() *DistributionWidgetYAxis {
	this := DistributionWidgetYAxis{}
	var max string = "auto"
	this.Max = &max
	var min string = "auto"
	this.Min = &min
	var scale string = "linear"
	this.Scale = &scale
	return &this
}

// GetIncludeZero returns the IncludeZero field value if set, zero value otherwise.
func (o *DistributionWidgetYAxis) GetIncludeZero() bool {
	if o == nil || o.IncludeZero == nil {
		var ret bool
		return ret
	}
	return *o.IncludeZero
}

// GetIncludeZeroOk returns a tuple with the IncludeZero field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DistributionWidgetYAxis) GetIncludeZeroOk() (*bool, bool) {
	if o == nil || o.IncludeZero == nil {
		return nil, false
	}
	return o.IncludeZero, true
}

// HasIncludeZero returns a boolean if a field has been set.
func (o *DistributionWidgetYAxis) HasIncludeZero() bool {
	if o != nil && o.IncludeZero != nil {
		return true
	}

	return false
}

// SetIncludeZero gets a reference to the given bool and assigns it to the IncludeZero field.
func (o *DistributionWidgetYAxis) SetIncludeZero(v bool) {
	o.IncludeZero = &v
}

// GetLabel returns the Label field value if set, zero value otherwise.
func (o *DistributionWidgetYAxis) GetLabel() string {
	if o == nil || o.Label == nil {
		var ret string
		return ret
	}
	return *o.Label
}

// GetLabelOk returns a tuple with the Label field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DistributionWidgetYAxis) GetLabelOk() (*string, bool) {
	if o == nil || o.Label == nil {
		return nil, false
	}
	return o.Label, true
}

// HasLabel returns a boolean if a field has been set.
func (o *DistributionWidgetYAxis) HasLabel() bool {
	if o != nil && o.Label != nil {
		return true
	}

	return false
}

// SetLabel gets a reference to the given string and assigns it to the Label field.
func (o *DistributionWidgetYAxis) SetLabel(v string) {
	o.Label = &v
}

// GetMax returns the Max field value if set, zero value otherwise.
func (o *DistributionWidgetYAxis) GetMax() string {
	if o == nil || o.Max == nil {
		var ret string
		return ret
	}
	return *o.Max
}

// GetMaxOk returns a tuple with the Max field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DistributionWidgetYAxis) GetMaxOk() (*string, bool) {
	if o == nil || o.Max == nil {
		return nil, false
	}
	return o.Max, true
}

// HasMax returns a boolean if a field has been set.
func (o *DistributionWidgetYAxis) HasMax() bool {
	if o != nil && o.Max != nil {
		return true
	}

	return false
}

// SetMax gets a reference to the given string and assigns it to the Max field.
func (o *DistributionWidgetYAxis) SetMax(v string) {
	o.Max = &v
}

// GetMin returns the Min field value if set, zero value otherwise.
func (o *DistributionWidgetYAxis) GetMin() string {
	if o == nil || o.Min == nil {
		var ret string
		return ret
	}
	return *o.Min
}

// GetMinOk returns a tuple with the Min field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DistributionWidgetYAxis) GetMinOk() (*string, bool) {
	if o == nil || o.Min == nil {
		return nil, false
	}
	return o.Min, true
}

// HasMin returns a boolean if a field has been set.
func (o *DistributionWidgetYAxis) HasMin() bool {
	if o != nil && o.Min != nil {
		return true
	}

	return false
}

// SetMin gets a reference to the given string and assigns it to the Min field.
func (o *DistributionWidgetYAxis) SetMin(v string) {
	o.Min = &v
}

// GetScale returns the Scale field value if set, zero value otherwise.
func (o *DistributionWidgetYAxis) GetScale() string {
	if o == nil || o.Scale == nil {
		var ret string
		return ret
	}
	return *o.Scale
}

// GetScaleOk returns a tuple with the Scale field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DistributionWidgetYAxis) GetScaleOk() (*string, bool) {
	if o == nil || o.Scale == nil {
		return nil, false
	}
	return o.Scale, true
}

// HasScale returns a boolean if a field has been set.
func (o *DistributionWidgetYAxis) HasScale() bool {
	if o != nil && o.Scale != nil {
		return true
	}

	return false
}

// SetScale gets a reference to the given string and assigns it to the Scale field.
func (o *DistributionWidgetYAxis) SetScale(v string) {
	o.Scale = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o DistributionWidgetYAxis) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.IncludeZero != nil {
		toSerialize["include_zero"] = o.IncludeZero
	}
	if o.Label != nil {
		toSerialize["label"] = o.Label
	}
	if o.Max != nil {
		toSerialize["max"] = o.Max
	}
	if o.Min != nil {
		toSerialize["min"] = o.Min
	}
	if o.Scale != nil {
		toSerialize["scale"] = o.Scale
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *DistributionWidgetYAxis) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		IncludeZero *bool   `json:"include_zero,omitempty"`
		Label       *string `json:"label,omitempty"`
		Max         *string `json:"max,omitempty"`
		Min         *string `json:"min,omitempty"`
		Scale       *string `json:"scale,omitempty"`
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
	o.IncludeZero = all.IncludeZero
	o.Label = all.Label
	o.Max = all.Max
	o.Min = all.Min
	o.Scale = all.Scale
	return nil
}
