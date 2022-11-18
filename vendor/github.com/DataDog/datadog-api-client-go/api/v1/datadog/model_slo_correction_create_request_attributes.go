// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// SLOCorrectionCreateRequestAttributes The attribute object associated with the SLO correction to be created.
type SLOCorrectionCreateRequestAttributes struct {
	// Category the SLO correction belongs to.
	Category SLOCorrectionCategory `json:"category"`
	// Description of the correction being made.
	Description *string `json:"description,omitempty"`
	// Length of time (in seconds) for a specified `rrule` recurring SLO correction.
	Duration *int64 `json:"duration,omitempty"`
	// Ending time of the correction in epoch seconds.
	End *int64 `json:"end,omitempty"`
	// The recurrence rules as defined in the iCalendar RFC 5545. The supported rules for SLO corrections
	// are `FREQ`, `INTERVAL`, `COUNT` and `UNTIL`.
	Rrule *string `json:"rrule,omitempty"`
	// ID of the SLO that this correction applies to.
	SloId string `json:"slo_id"`
	// Starting time of the correction in epoch seconds.
	Start int64 `json:"start"`
	// The timezone to display in the UI for the correction times (defaults to "UTC").
	Timezone *string `json:"timezone,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSLOCorrectionCreateRequestAttributes instantiates a new SLOCorrectionCreateRequestAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSLOCorrectionCreateRequestAttributes(category SLOCorrectionCategory, sloId string, start int64) *SLOCorrectionCreateRequestAttributes {
	this := SLOCorrectionCreateRequestAttributes{}
	this.Category = category
	this.SloId = sloId
	this.Start = start
	return &this
}

// NewSLOCorrectionCreateRequestAttributesWithDefaults instantiates a new SLOCorrectionCreateRequestAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSLOCorrectionCreateRequestAttributesWithDefaults() *SLOCorrectionCreateRequestAttributes {
	this := SLOCorrectionCreateRequestAttributes{}
	return &this
}

// GetCategory returns the Category field value.
func (o *SLOCorrectionCreateRequestAttributes) GetCategory() SLOCorrectionCategory {
	if o == nil {
		var ret SLOCorrectionCategory
		return ret
	}
	return o.Category
}

// GetCategoryOk returns a tuple with the Category field value
// and a boolean to check if the value has been set.
func (o *SLOCorrectionCreateRequestAttributes) GetCategoryOk() (*SLOCorrectionCategory, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Category, true
}

// SetCategory sets field value.
func (o *SLOCorrectionCreateRequestAttributes) SetCategory(v SLOCorrectionCategory) {
	o.Category = v
}

// GetDescription returns the Description field value if set, zero value otherwise.
func (o *SLOCorrectionCreateRequestAttributes) GetDescription() string {
	if o == nil || o.Description == nil {
		var ret string
		return ret
	}
	return *o.Description
}

// GetDescriptionOk returns a tuple with the Description field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionCreateRequestAttributes) GetDescriptionOk() (*string, bool) {
	if o == nil || o.Description == nil {
		return nil, false
	}
	return o.Description, true
}

// HasDescription returns a boolean if a field has been set.
func (o *SLOCorrectionCreateRequestAttributes) HasDescription() bool {
	if o != nil && o.Description != nil {
		return true
	}

	return false
}

// SetDescription gets a reference to the given string and assigns it to the Description field.
func (o *SLOCorrectionCreateRequestAttributes) SetDescription(v string) {
	o.Description = &v
}

// GetDuration returns the Duration field value if set, zero value otherwise.
func (o *SLOCorrectionCreateRequestAttributes) GetDuration() int64 {
	if o == nil || o.Duration == nil {
		var ret int64
		return ret
	}
	return *o.Duration
}

// GetDurationOk returns a tuple with the Duration field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionCreateRequestAttributes) GetDurationOk() (*int64, bool) {
	if o == nil || o.Duration == nil {
		return nil, false
	}
	return o.Duration, true
}

// HasDuration returns a boolean if a field has been set.
func (o *SLOCorrectionCreateRequestAttributes) HasDuration() bool {
	if o != nil && o.Duration != nil {
		return true
	}

	return false
}

// SetDuration gets a reference to the given int64 and assigns it to the Duration field.
func (o *SLOCorrectionCreateRequestAttributes) SetDuration(v int64) {
	o.Duration = &v
}

// GetEnd returns the End field value if set, zero value otherwise.
func (o *SLOCorrectionCreateRequestAttributes) GetEnd() int64 {
	if o == nil || o.End == nil {
		var ret int64
		return ret
	}
	return *o.End
}

// GetEndOk returns a tuple with the End field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionCreateRequestAttributes) GetEndOk() (*int64, bool) {
	if o == nil || o.End == nil {
		return nil, false
	}
	return o.End, true
}

// HasEnd returns a boolean if a field has been set.
func (o *SLOCorrectionCreateRequestAttributes) HasEnd() bool {
	if o != nil && o.End != nil {
		return true
	}

	return false
}

// SetEnd gets a reference to the given int64 and assigns it to the End field.
func (o *SLOCorrectionCreateRequestAttributes) SetEnd(v int64) {
	o.End = &v
}

// GetRrule returns the Rrule field value if set, zero value otherwise.
func (o *SLOCorrectionCreateRequestAttributes) GetRrule() string {
	if o == nil || o.Rrule == nil {
		var ret string
		return ret
	}
	return *o.Rrule
}

// GetRruleOk returns a tuple with the Rrule field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionCreateRequestAttributes) GetRruleOk() (*string, bool) {
	if o == nil || o.Rrule == nil {
		return nil, false
	}
	return o.Rrule, true
}

// HasRrule returns a boolean if a field has been set.
func (o *SLOCorrectionCreateRequestAttributes) HasRrule() bool {
	if o != nil && o.Rrule != nil {
		return true
	}

	return false
}

// SetRrule gets a reference to the given string and assigns it to the Rrule field.
func (o *SLOCorrectionCreateRequestAttributes) SetRrule(v string) {
	o.Rrule = &v
}

// GetSloId returns the SloId field value.
func (o *SLOCorrectionCreateRequestAttributes) GetSloId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.SloId
}

// GetSloIdOk returns a tuple with the SloId field value
// and a boolean to check if the value has been set.
func (o *SLOCorrectionCreateRequestAttributes) GetSloIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.SloId, true
}

// SetSloId sets field value.
func (o *SLOCorrectionCreateRequestAttributes) SetSloId(v string) {
	o.SloId = v
}

// GetStart returns the Start field value.
func (o *SLOCorrectionCreateRequestAttributes) GetStart() int64 {
	if o == nil {
		var ret int64
		return ret
	}
	return o.Start
}

// GetStartOk returns a tuple with the Start field value
// and a boolean to check if the value has been set.
func (o *SLOCorrectionCreateRequestAttributes) GetStartOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Start, true
}

// SetStart sets field value.
func (o *SLOCorrectionCreateRequestAttributes) SetStart(v int64) {
	o.Start = v
}

// GetTimezone returns the Timezone field value if set, zero value otherwise.
func (o *SLOCorrectionCreateRequestAttributes) GetTimezone() string {
	if o == nil || o.Timezone == nil {
		var ret string
		return ret
	}
	return *o.Timezone
}

// GetTimezoneOk returns a tuple with the Timezone field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionCreateRequestAttributes) GetTimezoneOk() (*string, bool) {
	if o == nil || o.Timezone == nil {
		return nil, false
	}
	return o.Timezone, true
}

// HasTimezone returns a boolean if a field has been set.
func (o *SLOCorrectionCreateRequestAttributes) HasTimezone() bool {
	if o != nil && o.Timezone != nil {
		return true
	}

	return false
}

// SetTimezone gets a reference to the given string and assigns it to the Timezone field.
func (o *SLOCorrectionCreateRequestAttributes) SetTimezone(v string) {
	o.Timezone = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SLOCorrectionCreateRequestAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["category"] = o.Category
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
	toSerialize["slo_id"] = o.SloId
	toSerialize["start"] = o.Start
	if o.Timezone != nil {
		toSerialize["timezone"] = o.Timezone
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SLOCorrectionCreateRequestAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Category *SLOCorrectionCategory `json:"category"`
		SloId    *string                `json:"slo_id"`
		Start    *int64                 `json:"start"`
	}{}
	all := struct {
		Category    SLOCorrectionCategory `json:"category"`
		Description *string               `json:"description,omitempty"`
		Duration    *int64                `json:"duration,omitempty"`
		End         *int64                `json:"end,omitempty"`
		Rrule       *string               `json:"rrule,omitempty"`
		SloId       string                `json:"slo_id"`
		Start       int64                 `json:"start"`
		Timezone    *string               `json:"timezone,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Category == nil {
		return fmt.Errorf("Required field category missing")
	}
	if required.SloId == nil {
		return fmt.Errorf("Required field slo_id missing")
	}
	if required.Start == nil {
		return fmt.Errorf("Required field start missing")
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
	if v := all.Category; !v.IsValid() {
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
	o.SloId = all.SloId
	o.Start = all.Start
	o.Timezone = all.Timezone
	return nil
}
