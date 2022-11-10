// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// UsageBillableSummaryResponse Response with monthly summary of data billed by Datadog.
type UsageBillableSummaryResponse struct {
	// An array of objects regarding usage of billable summary.
	Usage []UsageBillableSummaryHour `json:"usage,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageBillableSummaryResponse instantiates a new UsageBillableSummaryResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageBillableSummaryResponse() *UsageBillableSummaryResponse {
	this := UsageBillableSummaryResponse{}
	return &this
}

// NewUsageBillableSummaryResponseWithDefaults instantiates a new UsageBillableSummaryResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageBillableSummaryResponseWithDefaults() *UsageBillableSummaryResponse {
	this := UsageBillableSummaryResponse{}
	return &this
}

// GetUsage returns the Usage field value if set, zero value otherwise.
func (o *UsageBillableSummaryResponse) GetUsage() []UsageBillableSummaryHour {
	if o == nil || o.Usage == nil {
		var ret []UsageBillableSummaryHour
		return ret
	}
	return o.Usage
}

// GetUsageOk returns a tuple with the Usage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryResponse) GetUsageOk() (*[]UsageBillableSummaryHour, bool) {
	if o == nil || o.Usage == nil {
		return nil, false
	}
	return &o.Usage, true
}

// HasUsage returns a boolean if a field has been set.
func (o *UsageBillableSummaryResponse) HasUsage() bool {
	if o != nil && o.Usage != nil {
		return true
	}

	return false
}

// SetUsage gets a reference to the given []UsageBillableSummaryHour and assigns it to the Usage field.
func (o *UsageBillableSummaryResponse) SetUsage(v []UsageBillableSummaryHour) {
	o.Usage = v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageBillableSummaryResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Usage != nil {
		toSerialize["usage"] = o.Usage
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UsageBillableSummaryResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Usage []UsageBillableSummaryHour `json:"usage,omitempty"`
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
	o.Usage = all.Usage
	return nil
}
