// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsDevice Object describing the device used to perform the Synthetic test.
type SyntheticsDevice struct {
	// Screen height of the device.
	Height int64 `json:"height"`
	// The device ID.
	Id SyntheticsDeviceID `json:"id"`
	// Whether or not the device is a mobile.
	IsMobile *bool `json:"isMobile,omitempty"`
	// The device name.
	Name string `json:"name"`
	// Screen width of the device.
	Width int64 `json:"width"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsDevice instantiates a new SyntheticsDevice object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsDevice(height int64, id SyntheticsDeviceID, name string, width int64) *SyntheticsDevice {
	this := SyntheticsDevice{}
	this.Height = height
	this.Id = id
	this.Name = name
	this.Width = width
	return &this
}

// NewSyntheticsDeviceWithDefaults instantiates a new SyntheticsDevice object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsDeviceWithDefaults() *SyntheticsDevice {
	this := SyntheticsDevice{}
	return &this
}

// GetHeight returns the Height field value.
func (o *SyntheticsDevice) GetHeight() int64 {
	if o == nil {
		var ret int64
		return ret
	}
	return o.Height
}

// GetHeightOk returns a tuple with the Height field value
// and a boolean to check if the value has been set.
func (o *SyntheticsDevice) GetHeightOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Height, true
}

// SetHeight sets field value.
func (o *SyntheticsDevice) SetHeight(v int64) {
	o.Height = v
}

// GetId returns the Id field value.
func (o *SyntheticsDevice) GetId() SyntheticsDeviceID {
	if o == nil {
		var ret SyntheticsDeviceID
		return ret
	}
	return o.Id
}

// GetIdOk returns a tuple with the Id field value
// and a boolean to check if the value has been set.
func (o *SyntheticsDevice) GetIdOk() (*SyntheticsDeviceID, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Id, true
}

// SetId sets field value.
func (o *SyntheticsDevice) SetId(v SyntheticsDeviceID) {
	o.Id = v
}

// GetIsMobile returns the IsMobile field value if set, zero value otherwise.
func (o *SyntheticsDevice) GetIsMobile() bool {
	if o == nil || o.IsMobile == nil {
		var ret bool
		return ret
	}
	return *o.IsMobile
}

// GetIsMobileOk returns a tuple with the IsMobile field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsDevice) GetIsMobileOk() (*bool, bool) {
	if o == nil || o.IsMobile == nil {
		return nil, false
	}
	return o.IsMobile, true
}

// HasIsMobile returns a boolean if a field has been set.
func (o *SyntheticsDevice) HasIsMobile() bool {
	if o != nil && o.IsMobile != nil {
		return true
	}

	return false
}

// SetIsMobile gets a reference to the given bool and assigns it to the IsMobile field.
func (o *SyntheticsDevice) SetIsMobile(v bool) {
	o.IsMobile = &v
}

// GetName returns the Name field value.
func (o *SyntheticsDevice) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *SyntheticsDevice) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *SyntheticsDevice) SetName(v string) {
	o.Name = v
}

// GetWidth returns the Width field value.
func (o *SyntheticsDevice) GetWidth() int64 {
	if o == nil {
		var ret int64
		return ret
	}
	return o.Width
}

// GetWidthOk returns a tuple with the Width field value
// and a boolean to check if the value has been set.
func (o *SyntheticsDevice) GetWidthOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Width, true
}

// SetWidth sets field value.
func (o *SyntheticsDevice) SetWidth(v int64) {
	o.Width = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsDevice) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["height"] = o.Height
	toSerialize["id"] = o.Id
	if o.IsMobile != nil {
		toSerialize["isMobile"] = o.IsMobile
	}
	toSerialize["name"] = o.Name
	toSerialize["width"] = o.Width

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsDevice) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Height *int64              `json:"height"`
		Id     *SyntheticsDeviceID `json:"id"`
		Name   *string             `json:"name"`
		Width  *int64              `json:"width"`
	}{}
	all := struct {
		Height   int64              `json:"height"`
		Id       SyntheticsDeviceID `json:"id"`
		IsMobile *bool              `json:"isMobile,omitempty"`
		Name     string             `json:"name"`
		Width    int64              `json:"width"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Height == nil {
		return fmt.Errorf("Required field height missing")
	}
	if required.Id == nil {
		return fmt.Errorf("Required field id missing")
	}
	if required.Name == nil {
		return fmt.Errorf("Required field name missing")
	}
	if required.Width == nil {
		return fmt.Errorf("Required field width missing")
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
	if v := all.Id; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Height = all.Height
	o.Id = all.Id
	o.IsMobile = all.IsMobile
	o.Name = all.Name
	o.Width = all.Width
	return nil
}
