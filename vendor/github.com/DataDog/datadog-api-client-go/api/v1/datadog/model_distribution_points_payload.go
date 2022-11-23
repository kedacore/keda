// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// DistributionPointsPayload The distribution points payload.
type DistributionPointsPayload struct {
	// A list of distribution points series to submit to Datadog.
	Series []DistributionPointsSeries `json:"series"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewDistributionPointsPayload instantiates a new DistributionPointsPayload object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewDistributionPointsPayload(series []DistributionPointsSeries) *DistributionPointsPayload {
	this := DistributionPointsPayload{}
	this.Series = series
	return &this
}

// NewDistributionPointsPayloadWithDefaults instantiates a new DistributionPointsPayload object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewDistributionPointsPayloadWithDefaults() *DistributionPointsPayload {
	this := DistributionPointsPayload{}
	return &this
}

// GetSeries returns the Series field value.
func (o *DistributionPointsPayload) GetSeries() []DistributionPointsSeries {
	if o == nil {
		var ret []DistributionPointsSeries
		return ret
	}
	return o.Series
}

// GetSeriesOk returns a tuple with the Series field value
// and a boolean to check if the value has been set.
func (o *DistributionPointsPayload) GetSeriesOk() (*[]DistributionPointsSeries, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Series, true
}

// SetSeries sets field value.
func (o *DistributionPointsPayload) SetSeries(v []DistributionPointsSeries) {
	o.Series = v
}

// MarshalJSON serializes the struct using spec logic.
func (o DistributionPointsPayload) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["series"] = o.Series

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *DistributionPointsPayload) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Series *[]DistributionPointsSeries `json:"series"`
	}{}
	all := struct {
		Series []DistributionPointsSeries `json:"series"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Series == nil {
		return fmt.Errorf("Required field series missing")
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
	o.Series = all.Series
	return nil
}
