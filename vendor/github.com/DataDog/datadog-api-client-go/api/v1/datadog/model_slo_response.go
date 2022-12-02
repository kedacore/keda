// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SLOResponse A service level objective response containing a single service level objective.
type SLOResponse struct {
	// A service level objective object includes a service level indicator, thresholds
	// for one or more timeframes, and metadata (`name`, `description`, `tags`, etc.).
	Data *SLOResponseData `json:"data,omitempty"`
	// An array of error messages. Each endpoint documents how/whether this field is
	// used.
	Errors []string `json:"errors,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSLOResponse instantiates a new SLOResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSLOResponse() *SLOResponse {
	this := SLOResponse{}
	return &this
}

// NewSLOResponseWithDefaults instantiates a new SLOResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSLOResponseWithDefaults() *SLOResponse {
	this := SLOResponse{}
	return &this
}

// GetData returns the Data field value if set, zero value otherwise.
func (o *SLOResponse) GetData() SLOResponseData {
	if o == nil || o.Data == nil {
		var ret SLOResponseData
		return ret
	}
	return *o.Data
}

// GetDataOk returns a tuple with the Data field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOResponse) GetDataOk() (*SLOResponseData, bool) {
	if o == nil || o.Data == nil {
		return nil, false
	}
	return o.Data, true
}

// HasData returns a boolean if a field has been set.
func (o *SLOResponse) HasData() bool {
	if o != nil && o.Data != nil {
		return true
	}

	return false
}

// SetData gets a reference to the given SLOResponseData and assigns it to the Data field.
func (o *SLOResponse) SetData(v SLOResponseData) {
	o.Data = &v
}

// GetErrors returns the Errors field value if set, zero value otherwise.
func (o *SLOResponse) GetErrors() []string {
	if o == nil || o.Errors == nil {
		var ret []string
		return ret
	}
	return o.Errors
}

// GetErrorsOk returns a tuple with the Errors field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOResponse) GetErrorsOk() (*[]string, bool) {
	if o == nil || o.Errors == nil {
		return nil, false
	}
	return &o.Errors, true
}

// HasErrors returns a boolean if a field has been set.
func (o *SLOResponse) HasErrors() bool {
	if o != nil && o.Errors != nil {
		return true
	}

	return false
}

// SetErrors gets a reference to the given []string and assigns it to the Errors field.
func (o *SLOResponse) SetErrors(v []string) {
	o.Errors = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SLOResponse) MarshalJSON() ([]byte, error) {
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
func (o *SLOResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Data   *SLOResponseData `json:"data,omitempty"`
		Errors []string         `json:"errors,omitempty"`
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
