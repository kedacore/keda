// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SyntheticsBrowserError Error response object for a browser test.
type SyntheticsBrowserError struct {
	// Description of the error.
	Description string `json:"description"`
	// Name of the error.
	Name string `json:"name"`
	// Status Code of the error.
	Status *int64 `json:"status,omitempty"`
	// Error type returned by a browser test.
	Type SyntheticsBrowserErrorType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsBrowserError instantiates a new SyntheticsBrowserError object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsBrowserError(description string, name string, typeVar SyntheticsBrowserErrorType) *SyntheticsBrowserError {
	this := SyntheticsBrowserError{}
	this.Description = description
	this.Name = name
	this.Type = typeVar
	return &this
}

// NewSyntheticsBrowserErrorWithDefaults instantiates a new SyntheticsBrowserError object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsBrowserErrorWithDefaults() *SyntheticsBrowserError {
	this := SyntheticsBrowserError{}
	return &this
}

// GetDescription returns the Description field value.
func (o *SyntheticsBrowserError) GetDescription() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Description
}

// GetDescriptionOk returns a tuple with the Description field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserError) GetDescriptionOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Description, true
}

// SetDescription sets field value.
func (o *SyntheticsBrowserError) SetDescription(v string) {
	o.Description = v
}

// GetName returns the Name field value.
func (o *SyntheticsBrowserError) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserError) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *SyntheticsBrowserError) SetName(v string) {
	o.Name = v
}

// GetStatus returns the Status field value if set, zero value otherwise.
func (o *SyntheticsBrowserError) GetStatus() int64 {
	if o == nil || o.Status == nil {
		var ret int64
		return ret
	}
	return *o.Status
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserError) GetStatusOk() (*int64, bool) {
	if o == nil || o.Status == nil {
		return nil, false
	}
	return o.Status, true
}

// HasStatus returns a boolean if a field has been set.
func (o *SyntheticsBrowserError) HasStatus() bool {
	if o != nil && o.Status != nil {
		return true
	}

	return false
}

// SetStatus gets a reference to the given int64 and assigns it to the Status field.
func (o *SyntheticsBrowserError) SetStatus(v int64) {
	o.Status = &v
}

// GetType returns the Type field value.
func (o *SyntheticsBrowserError) GetType() SyntheticsBrowserErrorType {
	if o == nil {
		var ret SyntheticsBrowserErrorType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBrowserError) GetTypeOk() (*SyntheticsBrowserErrorType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *SyntheticsBrowserError) SetType(v SyntheticsBrowserErrorType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsBrowserError) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["description"] = o.Description
	toSerialize["name"] = o.Name
	if o.Status != nil {
		toSerialize["status"] = o.Status
	}
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsBrowserError) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Description *string                     `json:"description"`
		Name        *string                     `json:"name"`
		Type        *SyntheticsBrowserErrorType `json:"type"`
	}{}
	all := struct {
		Description string                     `json:"description"`
		Name        string                     `json:"name"`
		Status      *int64                     `json:"status,omitempty"`
		Type        SyntheticsBrowserErrorType `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Description == nil {
		return fmt.Errorf("Required field description missing")
	}
	if required.Name == nil {
		return fmt.Errorf("Required field name missing")
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
	o.Description = all.Description
	o.Name = all.Name
	o.Status = all.Status
	o.Type = all.Type
	return nil
}
