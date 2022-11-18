// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
	"time"
)

// LogsListRequestTime Timeframe to retrieve the log from.
type LogsListRequestTime struct {
	// Minimum timestamp for requested logs.
	From time.Time `json:"from"`
	// Timezone can be specified both as an offset (for example "UTC+03:00")
	// or a regional zone (for example "Europe/Paris").
	Timezone *string `json:"timezone,omitempty"`
	// Maximum timestamp for requested logs.
	To time.Time `json:"to"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogsListRequestTime instantiates a new LogsListRequestTime object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsListRequestTime(from time.Time, to time.Time) *LogsListRequestTime {
	this := LogsListRequestTime{}
	this.From = from
	this.To = to
	return &this
}

// NewLogsListRequestTimeWithDefaults instantiates a new LogsListRequestTime object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsListRequestTimeWithDefaults() *LogsListRequestTime {
	this := LogsListRequestTime{}
	return &this
}

// GetFrom returns the From field value.
func (o *LogsListRequestTime) GetFrom() time.Time {
	if o == nil {
		var ret time.Time
		return ret
	}
	return o.From
}

// GetFromOk returns a tuple with the From field value
// and a boolean to check if the value has been set.
func (o *LogsListRequestTime) GetFromOk() (*time.Time, bool) {
	if o == nil {
		return nil, false
	}
	return &o.From, true
}

// SetFrom sets field value.
func (o *LogsListRequestTime) SetFrom(v time.Time) {
	o.From = v
}

// GetTimezone returns the Timezone field value if set, zero value otherwise.
func (o *LogsListRequestTime) GetTimezone() string {
	if o == nil || o.Timezone == nil {
		var ret string
		return ret
	}
	return *o.Timezone
}

// GetTimezoneOk returns a tuple with the Timezone field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsListRequestTime) GetTimezoneOk() (*string, bool) {
	if o == nil || o.Timezone == nil {
		return nil, false
	}
	return o.Timezone, true
}

// HasTimezone returns a boolean if a field has been set.
func (o *LogsListRequestTime) HasTimezone() bool {
	if o != nil && o.Timezone != nil {
		return true
	}

	return false
}

// SetTimezone gets a reference to the given string and assigns it to the Timezone field.
func (o *LogsListRequestTime) SetTimezone(v string) {
	o.Timezone = &v
}

// GetTo returns the To field value.
func (o *LogsListRequestTime) GetTo() time.Time {
	if o == nil {
		var ret time.Time
		return ret
	}
	return o.To
}

// GetToOk returns a tuple with the To field value
// and a boolean to check if the value has been set.
func (o *LogsListRequestTime) GetToOk() (*time.Time, bool) {
	if o == nil {
		return nil, false
	}
	return &o.To, true
}

// SetTo sets field value.
func (o *LogsListRequestTime) SetTo(v time.Time) {
	o.To = v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsListRequestTime) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.From.Nanosecond() == 0 {
		toSerialize["from"] = o.From.Format("2006-01-02T15:04:05Z07:00")
	} else {
		toSerialize["from"] = o.From.Format("2006-01-02T15:04:05.000Z07:00")
	}
	if o.Timezone != nil {
		toSerialize["timezone"] = o.Timezone
	}
	if o.To.Nanosecond() == 0 {
		toSerialize["to"] = o.To.Format("2006-01-02T15:04:05Z07:00")
	} else {
		toSerialize["to"] = o.To.Format("2006-01-02T15:04:05.000Z07:00")
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsListRequestTime) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		From *time.Time `json:"from"`
		To   *time.Time `json:"to"`
	}{}
	all := struct {
		From     time.Time `json:"from"`
		Timezone *string   `json:"timezone,omitempty"`
		To       time.Time `json:"to"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.From == nil {
		return fmt.Errorf("Required field from missing")
	}
	if required.To == nil {
		return fmt.Errorf("Required field to missing")
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
	o.From = all.From
	o.Timezone = all.Timezone
	o.To = all.To
	return nil
}
