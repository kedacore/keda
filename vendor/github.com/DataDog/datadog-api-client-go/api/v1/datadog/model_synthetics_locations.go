// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SyntheticsLocations List of Synthetics locations.
type SyntheticsLocations struct {
	// List of Synthetics locations.
	Locations []SyntheticsLocation `json:"locations,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsLocations instantiates a new SyntheticsLocations object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsLocations() *SyntheticsLocations {
	this := SyntheticsLocations{}
	return &this
}

// NewSyntheticsLocationsWithDefaults instantiates a new SyntheticsLocations object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsLocationsWithDefaults() *SyntheticsLocations {
	this := SyntheticsLocations{}
	return &this
}

// GetLocations returns the Locations field value if set, zero value otherwise.
func (o *SyntheticsLocations) GetLocations() []SyntheticsLocation {
	if o == nil || o.Locations == nil {
		var ret []SyntheticsLocation
		return ret
	}
	return o.Locations
}

// GetLocationsOk returns a tuple with the Locations field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsLocations) GetLocationsOk() (*[]SyntheticsLocation, bool) {
	if o == nil || o.Locations == nil {
		return nil, false
	}
	return &o.Locations, true
}

// HasLocations returns a boolean if a field has been set.
func (o *SyntheticsLocations) HasLocations() bool {
	if o != nil && o.Locations != nil {
		return true
	}

	return false
}

// SetLocations gets a reference to the given []SyntheticsLocation and assigns it to the Locations field.
func (o *SyntheticsLocations) SetLocations(v []SyntheticsLocation) {
	o.Locations = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsLocations) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Locations != nil {
		toSerialize["locations"] = o.Locations
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsLocations) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Locations []SyntheticsLocation `json:"locations,omitempty"`
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
	o.Locations = all.Locations
	return nil
}
