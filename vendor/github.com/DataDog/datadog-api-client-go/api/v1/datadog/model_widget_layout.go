// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// WidgetLayout The layout for a widget on a `free` or **new dashboard layout** dashboard.
type WidgetLayout struct {
	// The height of the widget. Should be a non-negative integer.
	Height int64 `json:"height"`
	// Whether the widget should be the first one on the second column in high density or not.
	// **Note**: Only for the **new dashboard layout** and only one widget in the dashboard should have this property set to `true`.
	IsColumnBreak *bool `json:"is_column_break,omitempty"`
	// The width of the widget. Should be a non-negative integer.
	Width int64 `json:"width"`
	// The position of the widget on the x (horizontal) axis. Should be a non-negative integer.
	X int64 `json:"x"`
	// The position of the widget on the y (vertical) axis. Should be a non-negative integer.
	Y int64 `json:"y"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewWidgetLayout instantiates a new WidgetLayout object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewWidgetLayout(height int64, width int64, x int64, y int64) *WidgetLayout {
	this := WidgetLayout{}
	this.Height = height
	this.Width = width
	this.X = x
	this.Y = y
	return &this
}

// NewWidgetLayoutWithDefaults instantiates a new WidgetLayout object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewWidgetLayoutWithDefaults() *WidgetLayout {
	this := WidgetLayout{}
	return &this
}

// GetHeight returns the Height field value.
func (o *WidgetLayout) GetHeight() int64 {
	if o == nil {
		var ret int64
		return ret
	}
	return o.Height
}

// GetHeightOk returns a tuple with the Height field value
// and a boolean to check if the value has been set.
func (o *WidgetLayout) GetHeightOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Height, true
}

// SetHeight sets field value.
func (o *WidgetLayout) SetHeight(v int64) {
	o.Height = v
}

// GetIsColumnBreak returns the IsColumnBreak field value if set, zero value otherwise.
func (o *WidgetLayout) GetIsColumnBreak() bool {
	if o == nil || o.IsColumnBreak == nil {
		var ret bool
		return ret
	}
	return *o.IsColumnBreak
}

// GetIsColumnBreakOk returns a tuple with the IsColumnBreak field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WidgetLayout) GetIsColumnBreakOk() (*bool, bool) {
	if o == nil || o.IsColumnBreak == nil {
		return nil, false
	}
	return o.IsColumnBreak, true
}

// HasIsColumnBreak returns a boolean if a field has been set.
func (o *WidgetLayout) HasIsColumnBreak() bool {
	if o != nil && o.IsColumnBreak != nil {
		return true
	}

	return false
}

// SetIsColumnBreak gets a reference to the given bool and assigns it to the IsColumnBreak field.
func (o *WidgetLayout) SetIsColumnBreak(v bool) {
	o.IsColumnBreak = &v
}

// GetWidth returns the Width field value.
func (o *WidgetLayout) GetWidth() int64 {
	if o == nil {
		var ret int64
		return ret
	}
	return o.Width
}

// GetWidthOk returns a tuple with the Width field value
// and a boolean to check if the value has been set.
func (o *WidgetLayout) GetWidthOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Width, true
}

// SetWidth sets field value.
func (o *WidgetLayout) SetWidth(v int64) {
	o.Width = v
}

// GetX returns the X field value.
func (o *WidgetLayout) GetX() int64 {
	if o == nil {
		var ret int64
		return ret
	}
	return o.X
}

// GetXOk returns a tuple with the X field value
// and a boolean to check if the value has been set.
func (o *WidgetLayout) GetXOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.X, true
}

// SetX sets field value.
func (o *WidgetLayout) SetX(v int64) {
	o.X = v
}

// GetY returns the Y field value.
func (o *WidgetLayout) GetY() int64 {
	if o == nil {
		var ret int64
		return ret
	}
	return o.Y
}

// GetYOk returns a tuple with the Y field value
// and a boolean to check if the value has been set.
func (o *WidgetLayout) GetYOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Y, true
}

// SetY sets field value.
func (o *WidgetLayout) SetY(v int64) {
	o.Y = v
}

// MarshalJSON serializes the struct using spec logic.
func (o WidgetLayout) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["height"] = o.Height
	if o.IsColumnBreak != nil {
		toSerialize["is_column_break"] = o.IsColumnBreak
	}
	toSerialize["width"] = o.Width
	toSerialize["x"] = o.X
	toSerialize["y"] = o.Y

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *WidgetLayout) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Height *int64 `json:"height"`
		Width  *int64 `json:"width"`
		X      *int64 `json:"x"`
		Y      *int64 `json:"y"`
	}{}
	all := struct {
		Height        int64 `json:"height"`
		IsColumnBreak *bool `json:"is_column_break,omitempty"`
		Width         int64 `json:"width"`
		X             int64 `json:"x"`
		Y             int64 `json:"y"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Height == nil {
		return fmt.Errorf("Required field height missing")
	}
	if required.Width == nil {
		return fmt.Errorf("Required field width missing")
	}
	if required.X == nil {
		return fmt.Errorf("Required field x missing")
	}
	if required.Y == nil {
		return fmt.Errorf("Required field y missing")
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
	o.Height = all.Height
	o.IsColumnBreak = all.IsColumnBreak
	o.Width = all.Width
	o.X = all.X
	o.Y = all.Y
	return nil
}
