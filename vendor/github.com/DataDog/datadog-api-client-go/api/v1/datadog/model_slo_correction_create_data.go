// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SLOCorrectionCreateData The data object associated with the SLO correction to be created.
type SLOCorrectionCreateData struct {
	// The attribute object associated with the SLO correction to be created.
	Attributes *SLOCorrectionCreateRequestAttributes `json:"attributes,omitempty"`
	// SLO correction resource type.
	Type SLOCorrectionType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSLOCorrectionCreateData instantiates a new SLOCorrectionCreateData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSLOCorrectionCreateData(typeVar SLOCorrectionType) *SLOCorrectionCreateData {
	this := SLOCorrectionCreateData{}
	this.Type = typeVar
	return &this
}

// NewSLOCorrectionCreateDataWithDefaults instantiates a new SLOCorrectionCreateData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSLOCorrectionCreateDataWithDefaults() *SLOCorrectionCreateData {
	this := SLOCorrectionCreateData{}
	var typeVar SLOCorrectionType = SLOCORRECTIONTYPE_CORRECTION
	this.Type = typeVar
	return &this
}

// GetAttributes returns the Attributes field value if set, zero value otherwise.
func (o *SLOCorrectionCreateData) GetAttributes() SLOCorrectionCreateRequestAttributes {
	if o == nil || o.Attributes == nil {
		var ret SLOCorrectionCreateRequestAttributes
		return ret
	}
	return *o.Attributes
}

// GetAttributesOk returns a tuple with the Attributes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionCreateData) GetAttributesOk() (*SLOCorrectionCreateRequestAttributes, bool) {
	if o == nil || o.Attributes == nil {
		return nil, false
	}
	return o.Attributes, true
}

// HasAttributes returns a boolean if a field has been set.
func (o *SLOCorrectionCreateData) HasAttributes() bool {
	if o != nil && o.Attributes != nil {
		return true
	}

	return false
}

// SetAttributes gets a reference to the given SLOCorrectionCreateRequestAttributes and assigns it to the Attributes field.
func (o *SLOCorrectionCreateData) SetAttributes(v SLOCorrectionCreateRequestAttributes) {
	o.Attributes = &v
}

// GetType returns the Type field value.
func (o *SLOCorrectionCreateData) GetType() SLOCorrectionType {
	if o == nil {
		var ret SLOCorrectionType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *SLOCorrectionCreateData) GetTypeOk() (*SLOCorrectionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *SLOCorrectionCreateData) SetType(v SLOCorrectionType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SLOCorrectionCreateData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Attributes != nil {
		toSerialize["attributes"] = o.Attributes
	}
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SLOCorrectionCreateData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Type *SLOCorrectionType `json:"type"`
	}{}
	all := struct {
		Attributes *SLOCorrectionCreateRequestAttributes `json:"attributes,omitempty"`
		Type       SLOCorrectionType                     `json:"type"`
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
	if all.Attributes != nil && all.Attributes.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Attributes = all.Attributes
	o.Type = all.Type
	return nil
}
