// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// UsageSpecifiedCustomReportsAttributes The response containing attributes for specified custom reports.
type UsageSpecifiedCustomReportsAttributes struct {
	// The date the specified custom report was computed.
	ComputedOn *string `json:"computed_on,omitempty"`
	// The ending date of specified custom report.
	EndDate *string `json:"end_date,omitempty"`
	// A downloadable file for the specified custom reporting file.
	Location *string `json:"location,omitempty"`
	// size
	Size *int64 `json:"size,omitempty"`
	// The starting date of specified custom report.
	StartDate *string `json:"start_date,omitempty"`
	// A list of tags to apply to specified custom reports.
	Tags []string `json:"tags,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageSpecifiedCustomReportsAttributes instantiates a new UsageSpecifiedCustomReportsAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageSpecifiedCustomReportsAttributes() *UsageSpecifiedCustomReportsAttributes {
	this := UsageSpecifiedCustomReportsAttributes{}
	return &this
}

// NewUsageSpecifiedCustomReportsAttributesWithDefaults instantiates a new UsageSpecifiedCustomReportsAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageSpecifiedCustomReportsAttributesWithDefaults() *UsageSpecifiedCustomReportsAttributes {
	this := UsageSpecifiedCustomReportsAttributes{}
	return &this
}

// GetComputedOn returns the ComputedOn field value if set, zero value otherwise.
func (o *UsageSpecifiedCustomReportsAttributes) GetComputedOn() string {
	if o == nil || o.ComputedOn == nil {
		var ret string
		return ret
	}
	return *o.ComputedOn
}

// GetComputedOnOk returns a tuple with the ComputedOn field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSpecifiedCustomReportsAttributes) GetComputedOnOk() (*string, bool) {
	if o == nil || o.ComputedOn == nil {
		return nil, false
	}
	return o.ComputedOn, true
}

// HasComputedOn returns a boolean if a field has been set.
func (o *UsageSpecifiedCustomReportsAttributes) HasComputedOn() bool {
	if o != nil && o.ComputedOn != nil {
		return true
	}

	return false
}

// SetComputedOn gets a reference to the given string and assigns it to the ComputedOn field.
func (o *UsageSpecifiedCustomReportsAttributes) SetComputedOn(v string) {
	o.ComputedOn = &v
}

// GetEndDate returns the EndDate field value if set, zero value otherwise.
func (o *UsageSpecifiedCustomReportsAttributes) GetEndDate() string {
	if o == nil || o.EndDate == nil {
		var ret string
		return ret
	}
	return *o.EndDate
}

// GetEndDateOk returns a tuple with the EndDate field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSpecifiedCustomReportsAttributes) GetEndDateOk() (*string, bool) {
	if o == nil || o.EndDate == nil {
		return nil, false
	}
	return o.EndDate, true
}

// HasEndDate returns a boolean if a field has been set.
func (o *UsageSpecifiedCustomReportsAttributes) HasEndDate() bool {
	if o != nil && o.EndDate != nil {
		return true
	}

	return false
}

// SetEndDate gets a reference to the given string and assigns it to the EndDate field.
func (o *UsageSpecifiedCustomReportsAttributes) SetEndDate(v string) {
	o.EndDate = &v
}

// GetLocation returns the Location field value if set, zero value otherwise.
func (o *UsageSpecifiedCustomReportsAttributes) GetLocation() string {
	if o == nil || o.Location == nil {
		var ret string
		return ret
	}
	return *o.Location
}

// GetLocationOk returns a tuple with the Location field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSpecifiedCustomReportsAttributes) GetLocationOk() (*string, bool) {
	if o == nil || o.Location == nil {
		return nil, false
	}
	return o.Location, true
}

// HasLocation returns a boolean if a field has been set.
func (o *UsageSpecifiedCustomReportsAttributes) HasLocation() bool {
	if o != nil && o.Location != nil {
		return true
	}

	return false
}

// SetLocation gets a reference to the given string and assigns it to the Location field.
func (o *UsageSpecifiedCustomReportsAttributes) SetLocation(v string) {
	o.Location = &v
}

// GetSize returns the Size field value if set, zero value otherwise.
func (o *UsageSpecifiedCustomReportsAttributes) GetSize() int64 {
	if o == nil || o.Size == nil {
		var ret int64
		return ret
	}
	return *o.Size
}

// GetSizeOk returns a tuple with the Size field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSpecifiedCustomReportsAttributes) GetSizeOk() (*int64, bool) {
	if o == nil || o.Size == nil {
		return nil, false
	}
	return o.Size, true
}

// HasSize returns a boolean if a field has been set.
func (o *UsageSpecifiedCustomReportsAttributes) HasSize() bool {
	if o != nil && o.Size != nil {
		return true
	}

	return false
}

// SetSize gets a reference to the given int64 and assigns it to the Size field.
func (o *UsageSpecifiedCustomReportsAttributes) SetSize(v int64) {
	o.Size = &v
}

// GetStartDate returns the StartDate field value if set, zero value otherwise.
func (o *UsageSpecifiedCustomReportsAttributes) GetStartDate() string {
	if o == nil || o.StartDate == nil {
		var ret string
		return ret
	}
	return *o.StartDate
}

// GetStartDateOk returns a tuple with the StartDate field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSpecifiedCustomReportsAttributes) GetStartDateOk() (*string, bool) {
	if o == nil || o.StartDate == nil {
		return nil, false
	}
	return o.StartDate, true
}

// HasStartDate returns a boolean if a field has been set.
func (o *UsageSpecifiedCustomReportsAttributes) HasStartDate() bool {
	if o != nil && o.StartDate != nil {
		return true
	}

	return false
}

// SetStartDate gets a reference to the given string and assigns it to the StartDate field.
func (o *UsageSpecifiedCustomReportsAttributes) SetStartDate(v string) {
	o.StartDate = &v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *UsageSpecifiedCustomReportsAttributes) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSpecifiedCustomReportsAttributes) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *UsageSpecifiedCustomReportsAttributes) HasTags() bool {
	if o != nil && o.Tags != nil {
		return true
	}

	return false
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *UsageSpecifiedCustomReportsAttributes) SetTags(v []string) {
	o.Tags = v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageSpecifiedCustomReportsAttributes) MarshalJSON() ([]byte, error) {
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
	if o.Location != nil {
		toSerialize["location"] = o.Location
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
func (o *UsageSpecifiedCustomReportsAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		ComputedOn *string  `json:"computed_on,omitempty"`
		EndDate    *string  `json:"end_date,omitempty"`
		Location   *string  `json:"location,omitempty"`
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
	o.Location = all.Location
	o.Size = all.Size
	o.StartDate = all.StartDate
	o.Tags = all.Tags
	return nil
}
