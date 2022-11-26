// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SLOCorrectionUpdateRequestAttributes The attribute object associated with the SLO correction to be updated.
type SLOCorrectionUpdateRequestAttributes struct {
	// Category the SLO correction belongs to.
	Category *SLOCorrectionCategory `json:"category,omitempty"`
	// Description of the correction being made.
	Description *string `json:"description,omitempty"`
	// Length of time (in seconds) for a specified `rrule` recurring SLO correction.
	Duration *int64 `json:"duration,omitempty"`
	// Ending time of the correction in epoch seconds.
	End *int64 `json:"end,omitempty"`
	// The recurrence rules as defined in the iCalendar RFC 5545. The supported rules for SLO corrections
	// are `FREQ`, `INTERVAL`, `COUNT`, and `UNTIL`.
	Rrule *string `json:"rrule,omitempty"`
	// Starting time of the correction in epoch seconds.
	Start *int64 `json:"start,omitempty"`
	// The timezone to display in the UI for the correction times (defaults to "UTC").
	Timezone *string `json:"timezone,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSLOCorrectionUpdateRequestAttributes instantiates a new SLOCorrectionUpdateRequestAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSLOCorrectionUpdateRequestAttributes() *SLOCorrectionUpdateRequestAttributes {
	this := SLOCorrectionUpdateRequestAttributes{}
	return &this
}

// NewSLOCorrectionUpdateRequestAttributesWithDefaults instantiates a new SLOCorrectionUpdateRequestAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSLOCorrectionUpdateRequestAttributesWithDefaults() *SLOCorrectionUpdateRequestAttributes {
	this := SLOCorrectionUpdateRequestAttributes{}
	return &this
}

// GetCategory returns the Category field value if set, zero value otherwise.
func (o *SLOCorrectionUpdateRequestAttributes) GetCategory() SLOCorrectionCategory {
	if o == nil || o.Category == nil {
		var ret SLOCorrectionCategory
		return ret
	}
	return *o.Category
}

// GetCategoryOk returns a tuple with the Category field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionUpdateRequestAttributes) GetCategoryOk() (*SLOCorrectionCategory, bool) {
	if o == nil || o.Category == nil {
		return nil, false
	}
	return o.Category, true
}

// HasCategory returns a boolean if a field has been set.
func (o *SLOCorrectionUpdateRequestAttributes) HasCategory() bool {
	if o != nil && o.Category != nil {
		return true
	}

	return false
}

// SetCategory gets a reference to the given SLOCorrectionCategory and assigns it to the Category field.
func (o *SLOCorrectionUpdateRequestAttributes) SetCategory(v SLOCorrectionCategory) {
	o.Category = &v
}

// GetDescription returns the Description field value if set, zero value otherwise.
func (o *SLOCorrectionUpdateRequestAttributes) GetDescription() string {
	if o == nil || o.Description == nil {
		var ret string
		return ret
	}
	return *o.Description
}

// GetDescriptionOk returns a tuple with the Description field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionUpdateRequestAttributes) GetDescriptionOk() (*string, bool) {
	if o == nil || o.Description == nil {
		return nil, false
	}
	return o.Description, true
}

// HasDescription returns a boolean if a field has been set.
func (o *SLOCorrectionUpdateRequestAttributes) HasDescription() bool {
	if o != nil && o.Description != nil {
		return true
	}

	return false
}

// SetDescription gets a reference to the given string and assigns it to the Description field.
func (o *SLOCorrectionUpdateRequestAttributes) SetDescription(v string) {
	o.Description = &v
}

// GetDuration returns the Duration field value if set, zero value otherwise.
func (o *SLOCorrectionUpdateRequestAttributes) GetDuration() int64 {
	if o == nil || o.Duration == nil {
		var ret int64
		return ret
	}
	return *o.Duration
}

// GetDurationOk returns a tuple with the Duration field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionUpdateRequestAttributes) GetDurationOk() (*int64, bool) {
	if o == nil || o.Duration == nil {
		return nil, false
	}
	return o.Duration, true
}

// HasDuration returns a boolean if a field has been set.
func (o *SLOCorrectionUpdateRequestAttributes) HasDuration() bool {
	if o != nil && o.Duration != nil {
		return true
	}

	return false
}

// SetDuration gets a reference to the given int64 and assigns it to the Duration field.
func (o *SLOCorrectionUpdateRequestAttributes) SetDuration(v int64) {
	o.Duration = &v
}

// GetEnd returns the End field value if set, zero value otherwise.
func (o *SLOCorrectionUpdateRequestAttributes) GetEnd() int64 {
	if o == nil || o.End == nil {
		var ret int64
		return ret
	}
	return *o.End
}

// GetEndOk returns a tuple with the End field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionUpdateRequestAttributes) GetEndOk() (*int64, bool) {
	if o == nil || o.End == nil {
		return nil, false
	}
	return o.End, true
}

// HasEnd returns a boolean if a field has been set.
func (o *SLOCorrectionUpdateRequestAttributes) HasEnd() bool {
	if o != nil && o.End != nil {
		return true
	}

	return false
}

// SetEnd gets a reference to the given int64 and assigns it to the End field.
func (o *SLOCorrectionUpdateRequestAttributes) SetEnd(v int64) {
	o.End = &v
}

// GetRrule returns the Rrule field value if set, zero value otherwise.
func (o *SLOCorrectionUpdateRequestAttributes) GetRrule() string {
	if o == nil || o.Rrule == nil {
		var ret string
		return ret
	}
	return *o.Rrule
}

// GetRruleOk returns a tuple with the Rrule field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionUpdateRequestAttributes) GetRruleOk() (*string, bool) {
	if o == nil || o.Rrule == nil {
		return nil, false
	}
	return o.Rrule, true
}

// HasRrule returns a boolean if a field has been set.
func (o *SLOCorrectionUpdateRequestAttributes) HasRrule() bool {
	if o != nil && o.Rrule != nil {
		return true
	}

	return false
}

// SetRrule gets a reference to the given string and assigns it to the Rrule field.
func (o *SLOCorrectionUpdateRequestAttributes) SetRrule(v string) {
	o.Rrule = &v
}

// GetStart returns the Start field value if set, zero value otherwise.
func (o *SLOCorrectionUpdateRequestAttributes) GetStart() int64 {
	if o == nil || o.Start == nil {
		var ret int64
		return ret
	}
	return *o.Start
}

// GetStartOk returns a tuple with the Start field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionUpdateRequestAttributes) GetStartOk() (*int64, bool) {
	if o == nil || o.Start == nil {
		return nil, false
	}
	return o.Start, true
}

// HasStart returns a boolean if a field has been set.
func (o *SLOCorrectionUpdateRequestAttributes) HasStart() bool {
	if o != nil && o.Start != nil {
		return true
	}

	return false
}

// SetStart gets a reference to the given int64 and assigns it to the Start field.
func (o *SLOCorrectionUpdateRequestAttributes) SetStart(v int64) {
	o.Start = &v
}

// GetTimezone returns the Timezone field value if set, zero value otherwise.
func (o *SLOCorrectionUpdateRequestAttributes) GetTimezone() string {
	if o == nil || o.Timezone == nil {
		var ret string
		return ret
	}
	return *o.Timezone
}

// GetTimezoneOk returns a tuple with the Timezone field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionUpdateRequestAttributes) GetTimezoneOk() (*string, bool) {
	if o == nil || o.Timezone == nil {
		return nil, false
	}
	return o.Timezone, true
}

// HasTimezone returns a boolean if a field has been set.
func (o *SLOCorrectionUpdateRequestAttributes) HasTimezone() bool {
	if o != nil && o.Timezone != nil {
		return true
	}

	return false
}

// SetTimezone gets a reference to the given string and assigns it to the Timezone field.
func (o *SLOCorrectionUpdateRequestAttributes) SetTimezone(v string) {
	o.Timezone = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SLOCorrectionUpdateRequestAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Category != nil {
		toSerialize["category"] = o.Category
	}
	if o.Description != nil {
		toSerialize["description"] = o.Description
	}
	if o.Duration != nil {
		toSerialize["duration"] = o.Duration
	}
	if o.End != nil {
		toSerialize["end"] = o.End
	}
	if o.Rrule != nil {
		toSerialize["rrule"] = o.Rrule
	}
	if o.Start != nil {
		toSerialize["start"] = o.Start
	}
	if o.Timezone != nil {
		toSerialize["timezone"] = o.Timezone
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SLOCorrectionUpdateRequestAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Category    *SLOCorrectionCategory `json:"category,omitempty"`
		Description *string                `json:"description,omitempty"`
		Duration    *int64                 `json:"duration,omitempty"`
		End         *int64                 `json:"end,omitempty"`
		Rrule       *string                `json:"rrule,omitempty"`
		Start       *int64                 `json:"start,omitempty"`
		Timezone    *string                `json:"timezone,omitempty"`
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
	if v := all.Category; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Category = all.Category
	o.Description = all.Description
	o.Duration = all.Duration
	o.End = all.End
	o.Rrule = all.Rrule
	o.Start = all.Start
	o.Timezone = all.Timezone
	return nil
}
