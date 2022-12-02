// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// UsageCustomReportsAttributes The response containing attributes for custom reports.
type UsageCustomReportsAttributes struct {
	// The date the specified custom report was computed.
	ComputedOn *string `json:"computed_on,omitempty"`
	// The ending date of custom report.
	EndDate *string `json:"end_date,omitempty"`
	// size
	Size *int64 `json:"size,omitempty"`
	// The starting date of custom report.
	StartDate *string `json:"start_date,omitempty"`
	// A list of tags to apply to custom reports.
	Tags []string `json:"tags,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageCustomReportsAttributes instantiates a new UsageCustomReportsAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageCustomReportsAttributes() *UsageCustomReportsAttributes {
	this := UsageCustomReportsAttributes{}
	return &this
}

// NewUsageCustomReportsAttributesWithDefaults instantiates a new UsageCustomReportsAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageCustomReportsAttributesWithDefaults() *UsageCustomReportsAttributes {
	this := UsageCustomReportsAttributes{}
	return &this
}

// GetComputedOn returns the ComputedOn field value if set, zero value otherwise.
func (o *UsageCustomReportsAttributes) GetComputedOn() string {
	if o == nil || o.ComputedOn == nil {
		var ret string
		return ret
	}
	return *o.ComputedOn
}

// GetComputedOnOk returns a tuple with the ComputedOn field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageCustomReportsAttributes) GetComputedOnOk() (*string, bool) {
	if o == nil || o.ComputedOn == nil {
		return nil, false
	}
	return o.ComputedOn, true
}

// HasComputedOn returns a boolean if a field has been set.
func (o *UsageCustomReportsAttributes) HasComputedOn() bool {
	if o != nil && o.ComputedOn != nil {
		return true
	}

	return false
}

// SetComputedOn gets a reference to the given string and assigns it to the ComputedOn field.
func (o *UsageCustomReportsAttributes) SetComputedOn(v string) {
	o.ComputedOn = &v
}

// GetEndDate returns the EndDate field value if set, zero value otherwise.
func (o *UsageCustomReportsAttributes) GetEndDate() string {
	if o == nil || o.EndDate == nil {
		var ret string
		return ret
	}
	return *o.EndDate
}

// GetEndDateOk returns a tuple with the EndDate field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageCustomReportsAttributes) GetEndDateOk() (*string, bool) {
	if o == nil || o.EndDate == nil {
		return nil, false
	}
	return o.EndDate, true
}

// HasEndDate returns a boolean if a field has been set.
func (o *UsageCustomReportsAttributes) HasEndDate() bool {
	if o != nil && o.EndDate != nil {
		return true
	}

	return false
}

// SetEndDate gets a reference to the given string and assigns it to the EndDate field.
func (o *UsageCustomReportsAttributes) SetEndDate(v string) {
	o.EndDate = &v
}

// GetSize returns the Size field value if set, zero value otherwise.
func (o *UsageCustomReportsAttributes) GetSize() int64 {
	if o == nil || o.Size == nil {
		var ret int64
		return ret
	}
	return *o.Size
}

// GetSizeOk returns a tuple with the Size field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageCustomReportsAttributes) GetSizeOk() (*int64, bool) {
	if o == nil || o.Size == nil {
		return nil, false
	}
	return o.Size, true
}

// HasSize returns a boolean if a field has been set.
func (o *UsageCustomReportsAttributes) HasSize() bool {
	if o != nil && o.Size != nil {
		return true
	}

	return false
}

// SetSize gets a reference to the given int64 and assigns it to the Size field.
func (o *UsageCustomReportsAttributes) SetSize(v int64) {
	o.Size = &v
}

// GetStartDate returns the StartDate field value if set, zero value otherwise.
func (o *UsageCustomReportsAttributes) GetStartDate() string {
	if o == nil || o.StartDate == nil {
		var ret string
		return ret
	}
	return *o.StartDate
}

// GetStartDateOk returns a tuple with the StartDate field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageCustomReportsAttributes) GetStartDateOk() (*string, bool) {
	if o == nil || o.StartDate == nil {
		return nil, false
	}
	return o.StartDate, true
}

// HasStartDate returns a boolean if a field has been set.
func (o *UsageCustomReportsAttributes) HasStartDate() bool {
	if o != nil && o.StartDate != nil {
		return true
	}

	return false
}

// SetStartDate gets a reference to the given string and assigns it to the StartDate field.
func (o *UsageCustomReportsAttributes) SetStartDate(v string) {
	o.StartDate = &v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *UsageCustomReportsAttributes) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageCustomReportsAttributes) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *UsageCustomReportsAttributes) HasTags() bool {
	if o != nil && o.Tags != nil {
		return true
	}

	return false
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *UsageCustomReportsAttributes) SetTags(v []string) {
	o.Tags = v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageCustomReportsAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.ComputedOn != nil {
		toSerialize["computed_on"] = o.ComputedOn
	}
	if o.EndDate != nil {
		toSerialize["end_date"] = o.EndDate
	}
	if o.Size != nil {
		toSerialize["size"] = o.Size
	}
	if o.StartDate != nil {
		toSerialize["start_date"] = o.StartDate
	}
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UsageCustomReportsAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		ComputedOn *string  `json:"computed_on,omitempty"`
		EndDate    *string  `json:"end_date,omitempty"`
		Size       *int64   `json:"size,omitempty"`
		StartDate  *string  `json:"start_date,omitempty"`
		Tags       []string `json:"tags,omitempty"`
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
	o.ComputedOn = all.ComputedOn
	o.EndDate = all.EndDate
	o.Size = all.Size
	o.StartDate = all.StartDate
	o.Tags = all.Tags
	return nil
}
