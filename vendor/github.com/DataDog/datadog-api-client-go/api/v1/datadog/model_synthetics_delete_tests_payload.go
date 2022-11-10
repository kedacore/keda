// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsDeleteTestsPayload A JSON list of the ID or IDs of the Synthetic tests that you want
// to delete.
type SyntheticsDeleteTestsPayload struct {
	// An array of Synthetic test IDs you want to delete.
	PublicIds []string `json:"public_ids,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsDeleteTestsPayload instantiates a new SyntheticsDeleteTestsPayload object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsDeleteTestsPayload() *SyntheticsDeleteTestsPayload {
	this := SyntheticsDeleteTestsPayload{}
	return &this
}

// NewSyntheticsDeleteTestsPayloadWithDefaults instantiates a new SyntheticsDeleteTestsPayload object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsDeleteTestsPayloadWithDefaults() *SyntheticsDeleteTestsPayload {
	this := SyntheticsDeleteTestsPayload{}
	return &this
}

// GetPublicIds returns the PublicIds field value if set, zero value otherwise.
func (o *SyntheticsDeleteTestsPayload) GetPublicIds() []string {
	if o == nil || o.PublicIds == nil {
		var ret []string
		return ret
	}
	return o.PublicIds
}

// GetPublicIdsOk returns a tuple with the PublicIds field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsDeleteTestsPayload) GetPublicIdsOk() (*[]string, bool) {
	if o == nil || o.PublicIds == nil {
		return nil, false
	}
	return &o.PublicIds, true
}

// HasPublicIds returns a boolean if a field has been set.
func (o *SyntheticsDeleteTestsPayload) HasPublicIds() bool {
	if o != nil && o.PublicIds != nil {
		return true
	}

	return false
}

// SetPublicIds gets a reference to the given []string and assigns it to the PublicIds field.
func (o *SyntheticsDeleteTestsPayload) SetPublicIds(v []string) {
	o.PublicIds = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsDeleteTestsPayload) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.PublicIds != nil {
		toSerialize["public_ids"] = o.PublicIds
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsDeleteTestsPayload) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		PublicIds []string `json:"public_ids,omitempty"`
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
	o.PublicIds = all.PublicIds
	return nil
}
