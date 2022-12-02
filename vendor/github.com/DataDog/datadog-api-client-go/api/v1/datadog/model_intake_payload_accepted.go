// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// IntakePayloadAccepted The payload accepted for intake.
type IntakePayloadAccepted struct {
	// The status of the intake payload.
	Status *string `json:"status,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewIntakePayloadAccepted instantiates a new IntakePayloadAccepted object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIntakePayloadAccepted() *IntakePayloadAccepted {
	this := IntakePayloadAccepted{}
	return &this
}

// NewIntakePayloadAcceptedWithDefaults instantiates a new IntakePayloadAccepted object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIntakePayloadAcceptedWithDefaults() *IntakePayloadAccepted {
	this := IntakePayloadAccepted{}
	return &this
}

// GetStatus returns the Status field value if set, zero value otherwise.
func (o *IntakePayloadAccepted) GetStatus() string {
	if o == nil || o.Status == nil {
		var ret string
		return ret
	}
	return *o.Status
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IntakePayloadAccepted) GetStatusOk() (*string, bool) {
	if o == nil || o.Status == nil {
		return nil, false
	}
	return o.Status, true
}

// HasStatus returns a boolean if a field has been set.
func (o *IntakePayloadAccepted) HasStatus() bool {
	if o != nil && o.Status != nil {
		return true
	}

	return false
}

// SetStatus gets a reference to the given string and assigns it to the Status field.
func (o *IntakePayloadAccepted) SetStatus(v string) {
	o.Status = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o IntakePayloadAccepted) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Status != nil {
		toSerialize["status"] = o.Status
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *IntakePayloadAccepted) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Status *string `json:"status,omitempty"`
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
	o.Status = all.Status
	return nil
}
