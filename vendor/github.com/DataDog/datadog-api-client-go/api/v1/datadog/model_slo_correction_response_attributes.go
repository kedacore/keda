// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// SLOCorrectionResponseAttributes The attribute object associated with the SLO correction.
type SLOCorrectionResponseAttributes struct {
	// Category the SLO correction belongs to.
	Category *SLOCorrectionCategory `json:"category,omitempty"`
	// The epoch timestamp of when the correction was created at.
	CreatedAt *int64 `json:"created_at,omitempty"`
	// Object describing the creator of the shared element.
	Creator *Creator `json:"creator,omitempty"`
	// Description of the correction being made.
	Description *string `json:"description,omitempty"`
	// Length of time (in seconds) for a specified `rrule` recurring SLO correction.
	Duration NullableInt64 `json:"duration,omitempty"`
	// Ending time of the correction in epoch seconds.
	End *int64 `json:"end,omitempty"`
	// The epoch timestamp of when the correction was modified at.
	ModifiedAt *int64 `json:"modified_at,omitempty"`
	// Modifier of the object.
	Modifier NullableSLOCorrectionResponseAttributesModifier `json:"modifier,omitempty"`
	// The recurrence rules as defined in the iCalendar RFC 5545. The supported rules for SLO corrections
	// are `FREQ`, `INTERVAL`, `COUNT`, and `UNTIL`.
	Rrule NullableString `json:"rrule,omitempty"`
	// ID of the SLO that this correction applies to.
	SloId *string `json:"slo_id,omitempty"`
	// Starting time of the correction in epoch seconds.
	Start *int64 `json:"start,omitempty"`
	// The timezone to display in the UI for the correction times (defaults to "UTC").
	Timezone *string `json:"timezone,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewSLOCorrectionResponseAttributes instantiates a new SLOCorrectionResponseAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSLOCorrectionResponseAttributes() *SLOCorrectionResponseAttributes {
	this := SLOCorrectionResponseAttributes{}
	return &this
}

// NewSLOCorrectionResponseAttributesWithDefaults instantiates a new SLOCorrectionResponseAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSLOCorrectionResponseAttributesWithDefaults() *SLOCorrectionResponseAttributes {
	this := SLOCorrectionResponseAttributes{}
	return &this
}

// GetCategory returns the Category field value if set, zero value otherwise.
func (o *SLOCorrectionResponseAttributes) GetCategory() SLOCorrectionCategory {
	if o == nil || o.Category == nil {
		var ret SLOCorrectionCategory
		return ret
	}
	return *o.Category
}

// GetCategoryOk returns a tuple with the Category field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionResponseAttributes) GetCategoryOk() (*SLOCorrectionCategory, bool) {
	if o == nil || o.Category == nil {
		return nil, false
	}
	return o.Category, true
}

// HasCategory returns a boolean if a field has been set.
func (o *SLOCorrectionResponseAttributes) HasCategory() bool {
	if o != nil && o.Category != nil {
		return true
	}

	return false
}

// SetCategory gets a reference to the given SLOCorrectionCategory and assigns it to the Category field.
func (o *SLOCorrectionResponseAttributes) SetCategory(v SLOCorrectionCategory) {
	o.Category = &v
}

// GetCreatedAt returns the CreatedAt field value if set, zero value otherwise.
func (o *SLOCorrectionResponseAttributes) GetCreatedAt() int64 {
	if o == nil || o.CreatedAt == nil {
		var ret int64
		return ret
	}
	return *o.CreatedAt
}

// GetCreatedAtOk returns a tuple with the CreatedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionResponseAttributes) GetCreatedAtOk() (*int64, bool) {
	if o == nil || o.CreatedAt == nil {
		return nil, false
	}
	return o.CreatedAt, true
}

// HasCreatedAt returns a boolean if a field has been set.
func (o *SLOCorrectionResponseAttributes) HasCreatedAt() bool {
	if o != nil && o.CreatedAt != nil {
		return true
	}

	return false
}

// SetCreatedAt gets a reference to the given int64 and assigns it to the CreatedAt field.
func (o *SLOCorrectionResponseAttributes) SetCreatedAt(v int64) {
	o.CreatedAt = &v
}

// GetCreator returns the Creator field value if set, zero value otherwise.
func (o *SLOCorrectionResponseAttributes) GetCreator() Creator {
	if o == nil || o.Creator == nil {
		var ret Creator
		return ret
	}
	return *o.Creator
}

// GetCreatorOk returns a tuple with the Creator field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionResponseAttributes) GetCreatorOk() (*Creator, bool) {
	if o == nil || o.Creator == nil {
		return nil, false
	}
	return o.Creator, true
}

// HasCreator returns a boolean if a field has been set.
func (o *SLOCorrectionResponseAttributes) HasCreator() bool {
	if o != nil && o.Creator != nil {
		return true
	}

	return false
}

// SetCreator gets a reference to the given Creator and assigns it to the Creator field.
func (o *SLOCorrectionResponseAttributes) SetCreator(v Creator) {
	o.Creator = &v
}

// GetDescription returns the Description field value if set, zero value otherwise.
func (o *SLOCorrectionResponseAttributes) GetDescription() string {
	if o == nil || o.Description == nil {
		var ret string
		return ret
	}
	return *o.Description
}

// GetDescriptionOk returns a tuple with the Description field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionResponseAttributes) GetDescriptionOk() (*string, bool) {
	if o == nil || o.Description == nil {
		return nil, false
	}
	return o.Description, true
}

// HasDescription returns a boolean if a field has been set.
func (o *SLOCorrectionResponseAttributes) HasDescription() bool {
	if o != nil && o.Description != nil {
		return true
	}

	return false
}

// SetDescription gets a reference to the given string and assigns it to the Description field.
func (o *SLOCorrectionResponseAttributes) SetDescription(v string) {
	o.Description = &v
}

// GetDuration returns the Duration field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SLOCorrectionResponseAttributes) GetDuration() int64 {
	if o == nil || o.Duration.Get() == nil {
		var ret int64
		return ret
	}
	return *o.Duration.Get()
}

// GetDurationOk returns a tuple with the Duration field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SLOCorrectionResponseAttributes) GetDurationOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return o.Duration.Get(), o.Duration.IsSet()
}

// HasDuration returns a boolean if a field has been set.
func (o *SLOCorrectionResponseAttributes) HasDuration() bool {
	if o != nil && o.Duration.IsSet() {
		return true
	}

	return false
}

// SetDuration gets a reference to the given NullableInt64 and assigns it to the Duration field.
func (o *SLOCorrectionResponseAttributes) SetDuration(v int64) {
	o.Duration.Set(&v)
}

// SetDurationNil sets the value for Duration to be an explicit nil.
func (o *SLOCorrectionResponseAttributes) SetDurationNil() {
	o.Duration.Set(nil)
}

// UnsetDuration ensures that no value is present for Duration, not even an explicit nil.
func (o *SLOCorrectionResponseAttributes) UnsetDuration() {
	o.Duration.Unset()
}

// GetEnd returns the End field value if set, zero value otherwise.
func (o *SLOCorrectionResponseAttributes) GetEnd() int64 {
	if o == nil || o.End == nil {
		var ret int64
		return ret
	}
	return *o.End
}

// GetEndOk returns a tuple with the End field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionResponseAttributes) GetEndOk() (*int64, bool) {
	if o == nil || o.End == nil {
		return nil, false
	}
	return o.End, true
}

// HasEnd returns a boolean if a field has been set.
func (o *SLOCorrectionResponseAttributes) HasEnd() bool {
	if o != nil && o.End != nil {
		return true
	}

	return false
}

// SetEnd gets a reference to the given int64 and assigns it to the End field.
func (o *SLOCorrectionResponseAttributes) SetEnd(v int64) {
	o.End = &v
}

// GetModifiedAt returns the ModifiedAt field value if set, zero value otherwise.
func (o *SLOCorrectionResponseAttributes) GetModifiedAt() int64 {
	if o == nil || o.ModifiedAt == nil {
		var ret int64
		return ret
	}
	return *o.ModifiedAt
}

// GetModifiedAtOk returns a tuple with the ModifiedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionResponseAttributes) GetModifiedAtOk() (*int64, bool) {
	if o == nil || o.ModifiedAt == nil {
		return nil, false
	}
	return o.ModifiedAt, true
}

// HasModifiedAt returns a boolean if a field has been set.
func (o *SLOCorrectionResponseAttributes) HasModifiedAt() bool {
	if o != nil && o.ModifiedAt != nil {
		return true
	}

	return false
}

// SetModifiedAt gets a reference to the given int64 and assigns it to the ModifiedAt field.
func (o *SLOCorrectionResponseAttributes) SetModifiedAt(v int64) {
	o.ModifiedAt = &v
}

// GetModifier returns the Modifier field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SLOCorrectionResponseAttributes) GetModifier() SLOCorrectionResponseAttributesModifier {
	if o == nil || o.Modifier.Get() == nil {
		var ret SLOCorrectionResponseAttributesModifier
		return ret
	}
	return *o.Modifier.Get()
}

// GetModifierOk returns a tuple with the Modifier field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SLOCorrectionResponseAttributes) GetModifierOk() (*SLOCorrectionResponseAttributesModifier, bool) {
	if o == nil {
		return nil, false
	}
	return o.Modifier.Get(), o.Modifier.IsSet()
}

// HasModifier returns a boolean if a field has been set.
func (o *SLOCorrectionResponseAttributes) HasModifier() bool {
	if o != nil && o.Modifier.IsSet() {
		return true
	}

	return false
}

// SetModifier gets a reference to the given NullableSLOCorrectionResponseAttributesModifier and assigns it to the Modifier field.
func (o *SLOCorrectionResponseAttributes) SetModifier(v SLOCorrectionResponseAttributesModifier) {
	o.Modifier.Set(&v)
}

// SetModifierNil sets the value for Modifier to be an explicit nil.
func (o *SLOCorrectionResponseAttributes) SetModifierNil() {
	o.Modifier.Set(nil)
}

// UnsetModifier ensures that no value is present for Modifier, not even an explicit nil.
func (o *SLOCorrectionResponseAttributes) UnsetModifier() {
	o.Modifier.Unset()
}

// GetRrule returns the Rrule field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SLOCorrectionResponseAttributes) GetRrule() string {
	if o == nil || o.Rrule.Get() == nil {
		var ret string
		return ret
	}
	return *o.Rrule.Get()
}

// GetRruleOk returns a tuple with the Rrule field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SLOCorrectionResponseAttributes) GetRruleOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.Rrule.Get(), o.Rrule.IsSet()
}

// HasRrule returns a boolean if a field has been set.
func (o *SLOCorrectionResponseAttributes) HasRrule() bool {
	if o != nil && o.Rrule.IsSet() {
		return true
	}

	return false
}

// SetRrule gets a reference to the given NullableString and assigns it to the Rrule field.
func (o *SLOCorrectionResponseAttributes) SetRrule(v string) {
	o.Rrule.Set(&v)
}

// SetRruleNil sets the value for Rrule to be an explicit nil.
func (o *SLOCorrectionResponseAttributes) SetRruleNil() {
	o.Rrule.Set(nil)
}

// UnsetRrule ensures that no value is present for Rrule, not even an explicit nil.
func (o *SLOCorrectionResponseAttributes) UnsetRrule() {
	o.Rrule.Unset()
}

// GetSloId returns the SloId field value if set, zero value otherwise.
func (o *SLOCorrectionResponseAttributes) GetSloId() string {
	if o == nil || o.SloId == nil {
		var ret string
		return ret
	}
	return *o.SloId
}

// GetSloIdOk returns a tuple with the SloId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionResponseAttributes) GetSloIdOk() (*string, bool) {
	if o == nil || o.SloId == nil {
		return nil, false
	}
	return o.SloId, true
}

// HasSloId returns a boolean if a field has been set.
func (o *SLOCorrectionResponseAttributes) HasSloId() bool {
	if o != nil && o.SloId != nil {
		return true
	}

	return false
}

// SetSloId gets a reference to the given string and assigns it to the SloId field.
func (o *SLOCorrectionResponseAttributes) SetSloId(v string) {
	o.SloId = &v
}

// GetStart returns the Start field value if set, zero value otherwise.
func (o *SLOCorrectionResponseAttributes) GetStart() int64 {
	if o == nil || o.Start == nil {
		var ret int64
		return ret
	}
	return *o.Start
}

// GetStartOk returns a tuple with the Start field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionResponseAttributes) GetStartOk() (*int64, bool) {
	if o == nil || o.Start == nil {
		return nil, false
	}
	return o.Start, true
}

// HasStart returns a boolean if a field has been set.
func (o *SLOCorrectionResponseAttributes) HasStart() bool {
	if o != nil && o.Start != nil {
		return true
	}

	return false
}

// SetStart gets a reference to the given int64 and assigns it to the Start field.
func (o *SLOCorrectionResponseAttributes) SetStart(v int64) {
	o.Start = &v
}

// GetTimezone returns the Timezone field value if set, zero value otherwise.
func (o *SLOCorrectionResponseAttributes) GetTimezone() string {
	if o == nil || o.Timezone == nil {
		var ret string
		return ret
	}
	return *o.Timezone
}

// GetTimezoneOk returns a tuple with the Timezone field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOCorrectionResponseAttributes) GetTimezoneOk() (*string, bool) {
	if o == nil || o.Timezone == nil {
		return nil, false
	}
	return o.Timezone, true
}

// HasTimezone returns a boolean if a field has been set.
func (o *SLOCorrectionResponseAttributes) HasTimezone() bool {
	if o != nil && o.Timezone != nil {
		return true
	}

	return false
}

// SetTimezone gets a reference to the given string and assigns it to the Timezone field.
func (o *SLOCorrectionResponseAttributes) SetTimezone(v string) {
	o.Timezone = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SLOCorrectionResponseAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Category != nil {
		toSerialize["category"] = o.Category
	}
	if o.CreatedAt != nil {
		toSerialize["created_at"] = o.CreatedAt
	}
	if o.Creator != nil {
		toSerialize["creator"] = o.Creator
	}
	if o.Description != nil {
		toSerialize["description"] = o.Description
	}
	if o.Duration.IsSet() {
		toSerialize["duration"] = o.Duration.Get()
	}
	if o.End != nil {
		toSerialize["end"] = o.End
	}
	if o.ModifiedAt != nil {
		toSerialize["modified_at"] = o.ModifiedAt
	}
	if o.Modifier.IsSet() {
		toSerialize["modifier"] = o.Modifier.Get()
	}
	if o.Rrule.IsSet() {
		toSerialize["rrule"] = o.Rrule.Get()
	}
	if o.SloId != nil {
		toSerialize["slo_id"] = o.SloId
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
func (o *SLOCorrectionResponseAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Category    *SLOCorrectionCategory                          `json:"category,omitempty"`
		CreatedAt   *int64                                          `json:"created_at,omitempty"`
		Creator     *Creator                                        `json:"creator,omitempty"`
		Description *string                                         `json:"description,omitempty"`
		Duration    NullableInt64                                   `json:"duration,omitempty"`
		End         *int64                                          `json:"end,omitempty"`
		ModifiedAt  *int64                                          `json:"modified_at,omitempty"`
		Modifier    NullableSLOCorrectionResponseAttributesModifier `json:"modifier,omitempty"`
		Rrule       NullableString                                  `json:"rrule,omitempty"`
		SloId       *string                                         `json:"slo_id,omitempty"`
		Start       *int64                                          `json:"start,omitempty"`
		Timezone    *string                                         `json:"timezone,omitempty"`
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
	o.CreatedAt = all.CreatedAt
	if all.Creator != nil && all.Creator.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Creator = all.Creator
	o.Description = all.Description
	o.Duration = all.Duration
	o.End = all.End
	o.ModifiedAt = all.ModifiedAt
	o.Modifier = all.Modifier
	o.Rrule = all.Rrule
	o.SloId = all.SloId
	o.Start = all.Start
	o.Timezone = all.Timezone
	return nil
}
