// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SLOCorrectionResponseAttributesModifier Modifier of the object.
type SLOCorrectionResponseAttributesModifier struct {
	// Email of the Modifier.
	Email *string `json:"email,omitempty"`
	// Handle of the Modifier.
	Handle *string `json:"handle,omitempty"`
	// Name of the Modifier.
	Name *string `json:"name,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSLOCorrectionResponseAttributesModifier instantiates a new SLOCorrectionResponseAttributesModifier object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSLOCorrectionResponseAttributesModifier() *SLOCorrectionResponseAttributesModifier {
	this := SLOCorrectionResponseAttributesModifier{}
	return &this
}

// NewSLOCorrectionResponseAttributesModifierWithDefaults instantiates a new SLOCorrectionResponseAttributesModifier object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSLOCorrectionResponseAttributesModifierWithDefaults() *SLOCorrectionResponseAttributesModifier {
	this := SLOCorrectionResponseAttributesModifier{}
	return &this
}

// GetEmail returns the Email field value if set, zero value otherwise.
func (o *SLOCorrectionResponseAttributesModifier) GetEmail() string {
	if o == nil || o.Email == nil {
		var ret string
		return ret
	}
	return *o.Email
}

// GetEmailOk returns a tuple with the Email field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionResponseAttributesModifier) GetEmailOk() (*string, bool) {
	if o == nil || o.Email == nil {
		return nil, false
	}
	return o.Email, true
}

// HasEmail returns a boolean if a field has been set.
func (o *SLOCorrectionResponseAttributesModifier) HasEmail() bool {
	if o != nil && o.Email != nil {
		return true
	}

	return false
}

// SetEmail gets a reference to the given string and assigns it to the Email field.
func (o *SLOCorrectionResponseAttributesModifier) SetEmail(v string) {
	o.Email = &v
}

// GetHandle returns the Handle field value if set, zero value otherwise.
func (o *SLOCorrectionResponseAttributesModifier) GetHandle() string {
	if o == nil || o.Handle == nil {
		var ret string
		return ret
	}
	return *o.Handle
}

// GetHandleOk returns a tuple with the Handle field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionResponseAttributesModifier) GetHandleOk() (*string, bool) {
	if o == nil || o.Handle == nil {
		return nil, false
	}
	return o.Handle, true
}

// HasHandle returns a boolean if a field has been set.
func (o *SLOCorrectionResponseAttributesModifier) HasHandle() bool {
	if o != nil && o.Handle != nil {
		return true
	}

	return false
}

// SetHandle gets a reference to the given string and assigns it to the Handle field.
func (o *SLOCorrectionResponseAttributesModifier) SetHandle(v string) {
	o.Handle = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *SLOCorrectionResponseAttributesModifier) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionResponseAttributesModifier) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *SLOCorrectionResponseAttributesModifier) HasName() bool {
	if o != nil && o.Name != nil {
		return true
	}

	return false
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *SLOCorrectionResponseAttributesModifier) SetName(v string) {
	o.Name = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SLOCorrectionResponseAttributesModifier) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Email != nil {
		toSerialize["email"] = o.Email
	}
	if o.Handle != nil {
		toSerialize["handle"] = o.Handle
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SLOCorrectionResponseAttributesModifier) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Email  *string `json:"email,omitempty"`
		Handle *string `json:"handle,omitempty"`
		Name   *string `json:"name,omitempty"`
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
	o.Email = all.Email
	o.Handle = all.Handle
	o.Name = all.Name
	return nil
}

// NullableSLOCorrectionResponseAttributesModifier handles when a null is used for SLOCorrectionResponseAttributesModifier.
type NullableSLOCorrectionResponseAttributesModifier struct {
	value *SLOCorrectionResponseAttributesModifier
	isSet bool
}

// Get returns the associated value.
func (v NullableSLOCorrectionResponseAttributesModifier) Get() *SLOCorrectionResponseAttributesModifier {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableSLOCorrectionResponseAttributesModifier) Set(val *SLOCorrectionResponseAttributesModifier) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableSLOCorrectionResponseAttributesModifier) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableSLOCorrectionResponseAttributesModifier) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableSLOCorrectionResponseAttributesModifier initializes the struct as if Set has been called.
func NewNullableSLOCorrectionResponseAttributesModifier(val *SLOCorrectionResponseAttributesModifier) *NullableSLOCorrectionResponseAttributesModifier {
	return &NullableSLOCorrectionResponseAttributesModifier{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableSLOCorrectionResponseAttributesModifier) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableSLOCorrectionResponseAttributesModifier) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
