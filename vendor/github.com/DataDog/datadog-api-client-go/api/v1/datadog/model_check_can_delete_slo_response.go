// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// CheckCanDeleteSLOResponse A service level objective response containing the requested object.
type CheckCanDeleteSLOResponse struct {
	// An array of service level objective objects.
	Data *CheckCanDeleteSLOResponseData `json:"data,omitempty"`
	// A mapping of SLO id to it's current usages.
	Errors map[string]string `json:"errors,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewCheckCanDeleteSLOResponse instantiates a new CheckCanDeleteSLOResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCheckCanDeleteSLOResponse() *CheckCanDeleteSLOResponse {
	this := CheckCanDeleteSLOResponse{}
	return &this
}

// NewCheckCanDeleteSLOResponseWithDefaults instantiates a new CheckCanDeleteSLOResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCheckCanDeleteSLOResponseWithDefaults() *CheckCanDeleteSLOResponse {
	this := CheckCanDeleteSLOResponse{}
	return &this
}

// GetData returns the Data field value if set, zero value otherwise.
func (o *CheckCanDeleteSLOResponse) GetData() CheckCanDeleteSLOResponseData {
	if o == nil || o.Data == nil {
		var ret CheckCanDeleteSLOResponseData
		return ret
	}
	return *o.Data
}

// GetDataOk returns a tuple with the Data field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CheckCanDeleteSLOResponse) GetDataOk() (*CheckCanDeleteSLOResponseData, bool) {
	if o == nil || o.Data == nil {
		return nil, false
	}
	return o.Data, true
}

// HasData returns a boolean if a field has been set.
func (o *CheckCanDeleteSLOResponse) HasData() bool {
	if o != nil && o.Data != nil {
		return true
	}

	return false
}

// SetData gets a reference to the given CheckCanDeleteSLOResponseData and assigns it to the Data field.
func (o *CheckCanDeleteSLOResponse) SetData(v CheckCanDeleteSLOResponseData) {
	o.Data = &v
}

// GetErrors returns the Errors field value if set, zero value otherwise.
func (o *CheckCanDeleteSLOResponse) GetErrors() map[string]string {
	if o == nil || o.Errors == nil {
		var ret map[string]string
		return ret
	}
	return o.Errors
}

// GetErrorsOk returns a tuple with the Errors field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CheckCanDeleteSLOResponse) GetErrorsOk() (*map[string]string, bool) {
	if o == nil || o.Errors == nil {
		return nil, false
	}
	return &o.Errors, true
}

// HasErrors returns a boolean if a field has been set.
func (o *CheckCanDeleteSLOResponse) HasErrors() bool {
	if o != nil && o.Errors != nil {
		return true
	}

	return false
}

// SetErrors gets a reference to the given map[string]string and assigns it to the Errors field.
func (o *CheckCanDeleteSLOResponse) SetErrors(v map[string]string) {
	o.Errors = v
}

// MarshalJSON serializes the struct using spec logic.
func (o CheckCanDeleteSLOResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Data != nil {
		toSerialize["data"] = o.Data
	}
	if o.Errors != nil {
		toSerialize["errors"] = o.Errors
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *CheckCanDeleteSLOResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Data   *CheckCanDeleteSLOResponseData `json:"data,omitempty"`
		Errors map[string]string              `json:"errors,omitempty"`
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
	if all.Data != nil && all.Data.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Data = all.Data
	o.Errors = all.Errors
	return nil
}
