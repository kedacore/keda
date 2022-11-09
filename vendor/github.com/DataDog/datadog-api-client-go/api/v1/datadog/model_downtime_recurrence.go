// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// DowntimeRecurrence An object defining the recurrence of the downtime.
type DowntimeRecurrence struct {
	// How often to repeat as an integer.
	// For example, to repeat every 3 days, select a type of `days` and a period of `3`.
	Period *int32 `json:"period,omitempty"`
	// The `RRULE` standard for defining recurring events (**requires to set "type" to rrule**)
	// For example, to have a recurring event on the first day of each month, set the type to `rrule` and set the `FREQ` to `MONTHLY` and `BYMONTHDAY` to `1`.
	// Most common `rrule` options from the [iCalendar Spec](https://tools.ietf.org/html/rfc5545) are supported.
	//
	// **Note**: Attributes specifying the duration in `RRULE` are not supported (for example, `DTSTART`, `DTEND`, `DURATION`).
	// More examples available in this [downtime guide](https://docs.datadoghq.com/monitors/guide/suppress-alert-with-downtimes/?tab=api)
	Rrule *string `json:"rrule,omitempty"`
	// The type of recurrence. Choose from `days`, `weeks`, `months`, `years`, `rrule`.
	Type *string `json:"type,omitempty"`
	// The date at which the recurrence should end as a POSIX timestamp.
	// `until_occurences` and `until_date` are mutually exclusive.
	UntilDate NullableInt64 `json:"until_date,omitempty"`
	// How many times the downtime is rescheduled.
	// `until_occurences` and `until_date` are mutually exclusive.
	UntilOccurrences NullableInt32 `json:"until_occurrences,omitempty"`
	// A list of week days to repeat on. Choose from `Mon`, `Tue`, `Wed`, `Thu`, `Fri`, `Sat` or `Sun`.
	// Only applicable when type is weeks. First letter must be capitalized.
	WeekDays []string `json:"week_days,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewDowntimeRecurrence instantiates a new DowntimeRecurrence object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewDowntimeRecurrence() *DowntimeRecurrence {
	this := DowntimeRecurrence{}
	return &this
}

// NewDowntimeRecurrenceWithDefaults instantiates a new DowntimeRecurrence object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewDowntimeRecurrenceWithDefaults() *DowntimeRecurrence {
	this := DowntimeRecurrence{}
	return &this
}

// GetPeriod returns the Period field value if set, zero value otherwise.
func (o *DowntimeRecurrence) GetPeriod() int32 {
	if o == nil || o.Period == nil {
		var ret int32
		return ret
	}
	return *o.Period
}

// GetPeriodOk returns a tuple with the Period field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DowntimeRecurrence) GetPeriodOk() (*int32, bool) {
	if o == nil || o.Period == nil {
		return nil, false
	}
	return o.Period, true
}

// HasPeriod returns a boolean if a field has been set.
func (o *DowntimeRecurrence) HasPeriod() bool {
	if o != nil && o.Period != nil {
		return true
	}

	return false
}

// SetPeriod gets a reference to the given int32 and assigns it to the Period field.
func (o *DowntimeRecurrence) SetPeriod(v int32) {
	o.Period = &v
}

// GetRrule returns the Rrule field value if set, zero value otherwise.
func (o *DowntimeRecurrence) GetRrule() string {
	if o == nil || o.Rrule == nil {
		var ret string
		return ret
	}
	return *o.Rrule
}

// GetRruleOk returns a tuple with the Rrule field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DowntimeRecurrence) GetRruleOk() (*string, bool) {
	if o == nil || o.Rrule == nil {
		return nil, false
	}
	return o.Rrule, true
}

// HasRrule returns a boolean if a field has been set.
func (o *DowntimeRecurrence) HasRrule() bool {
	if o != nil && o.Rrule != nil {
		return true
	}

	return false
}

// SetRrule gets a reference to the given string and assigns it to the Rrule field.
func (o *DowntimeRecurrence) SetRrule(v string) {
	o.Rrule = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *DowntimeRecurrence) GetType() string {
	if o == nil || o.Type == nil {
		var ret string
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DowntimeRecurrence) GetTypeOk() (*string, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *DowntimeRecurrence) HasType() bool {
	if o != nil && o.Type != nil {
		return true
	}

	return false
}

// SetType gets a reference to the given string and assigns it to the Type field.
func (o *DowntimeRecurrence) SetType(v string) {
	o.Type = &v
}

// GetUntilDate returns the UntilDate field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *DowntimeRecurrence) GetUntilDate() int64 {
	if o == nil || o.UntilDate.Get() == nil {
		var ret int64
		return ret
	}
	return *o.UntilDate.Get()
}

// GetUntilDateOk returns a tuple with the UntilDate field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *DowntimeRecurrence) GetUntilDateOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return o.UntilDate.Get(), o.UntilDate.IsSet()
}

// HasUntilDate returns a boolean if a field has been set.
func (o *DowntimeRecurrence) HasUntilDate() bool {
	if o != nil && o.UntilDate.IsSet() {
		return true
	}

	return false
}

// SetUntilDate gets a reference to the given NullableInt64 and assigns it to the UntilDate field.
func (o *DowntimeRecurrence) SetUntilDate(v int64) {
	o.UntilDate.Set(&v)
}

// SetUntilDateNil sets the value for UntilDate to be an explicit nil.
func (o *DowntimeRecurrence) SetUntilDateNil() {
	o.UntilDate.Set(nil)
}

// UnsetUntilDate ensures that no value is present for UntilDate, not even an explicit nil.
func (o *DowntimeRecurrence) UnsetUntilDate() {
	o.UntilDate.Unset()
}

// GetUntilOccurrences returns the UntilOccurrences field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *DowntimeRecurrence) GetUntilOccurrences() int32 {
	if o == nil || o.UntilOccurrences.Get() == nil {
		var ret int32
		return ret
	}
	return *o.UntilOccurrences.Get()
}

// GetUntilOccurrencesOk returns a tuple with the UntilOccurrences field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *DowntimeRecurrence) GetUntilOccurrencesOk() (*int32, bool) {
	if o == nil {
		return nil, false
	}
	return o.UntilOccurrences.Get(), o.UntilOccurrences.IsSet()
}

// HasUntilOccurrences returns a boolean if a field has been set.
func (o *DowntimeRecurrence) HasUntilOccurrences() bool {
	if o != nil && o.UntilOccurrences.IsSet() {
		return true
	}

	return false
}

// SetUntilOccurrences gets a reference to the given NullableInt32 and assigns it to the UntilOccurrences field.
func (o *DowntimeRecurrence) SetUntilOccurrences(v int32) {
	o.UntilOccurrences.Set(&v)
}

// SetUntilOccurrencesNil sets the value for UntilOccurrences to be an explicit nil.
func (o *DowntimeRecurrence) SetUntilOccurrencesNil() {
	o.UntilOccurrences.Set(nil)
}

// UnsetUntilOccurrences ensures that no value is present for UntilOccurrences, not even an explicit nil.
func (o *DowntimeRecurrence) UnsetUntilOccurrences() {
	o.UntilOccurrences.Unset()
}

// GetWeekDays returns the WeekDays field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *DowntimeRecurrence) GetWeekDays() []string {
	if o == nil {
		var ret []string
		return ret
	}
	return o.WeekDays
}

// GetWeekDaysOk returns a tuple with the WeekDays field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *DowntimeRecurrence) GetWeekDaysOk() (*[]string, bool) {
	if o == nil || o.WeekDays == nil {
		return nil, false
	}
	return &o.WeekDays, true
}

// HasWeekDays returns a boolean if a field has been set.
func (o *DowntimeRecurrence) HasWeekDays() bool {
	if o != nil && o.WeekDays != nil {
		return true
	}

	return false
}

// SetWeekDays gets a reference to the given []string and assigns it to the WeekDays field.
func (o *DowntimeRecurrence) SetWeekDays(v []string) {
	o.WeekDays = v
}

// MarshalJSON serializes the struct using spec logic.
func (o DowntimeRecurrence) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Period != nil {
		toSerialize["period"] = o.Period
	}
	if o.Rrule != nil {
		toSerialize["rrule"] = o.Rrule
	}
	if o.Type != nil {
		toSerialize["type"] = o.Type
	}
	if o.UntilDate.IsSet() {
		toSerialize["until_date"] = o.UntilDate.Get()
	}
	if o.UntilOccurrences.IsSet() {
		toSerialize["until_occurrences"] = o.UntilOccurrences.Get()
	}
	if o.WeekDays != nil {
		toSerialize["week_days"] = o.WeekDays
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *DowntimeRecurrence) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Period           *int32        `json:"period,omitempty"`
		Rrule            *string       `json:"rrule,omitempty"`
		Type             *string       `json:"type,omitempty"`
		UntilDate        NullableInt64 `json:"until_date,omitempty"`
		UntilOccurrences NullableInt32 `json:"until_occurrences,omitempty"`
		WeekDays         []string      `json:"week_days,omitempty"`
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
	o.Period = all.Period
	o.Rrule = all.Rrule
	o.Type = all.Type
	o.UntilDate = all.UntilDate
	o.UntilOccurrences = all.UntilOccurrences
	o.WeekDays = all.WeekDays
	return nil
}

// NullableDowntimeRecurrence handles when a null is used for DowntimeRecurrence.
type NullableDowntimeRecurrence struct {
	value *DowntimeRecurrence
	isSet bool
}

// Get returns the associated value.
func (v NullableDowntimeRecurrence) Get() *DowntimeRecurrence {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableDowntimeRecurrence) Set(val *DowntimeRecurrence) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableDowntimeRecurrence) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableDowntimeRecurrence) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableDowntimeRecurrence initializes the struct as if Set has been called.
func NewNullableDowntimeRecurrence(val *DowntimeRecurrence) *NullableDowntimeRecurrence {
	return &NullableDowntimeRecurrence{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableDowntimeRecurrence) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableDowntimeRecurrence) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
