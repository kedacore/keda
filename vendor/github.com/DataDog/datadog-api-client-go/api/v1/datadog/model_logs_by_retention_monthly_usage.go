// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"time"
)

// LogsByRetentionMonthlyUsage Object containing a summary of indexed logs usage by retention period for a single month.
type LogsByRetentionMonthlyUsage struct {
	// The month for the usage.
	Date *time.Time `json:"date,omitempty"`
	// Indexed logs usage for each active retention for the month.
	Usage []LogsRetentionSumUsage `json:"usage,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogsByRetentionMonthlyUsage instantiates a new LogsByRetentionMonthlyUsage object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsByRetentionMonthlyUsage() *LogsByRetentionMonthlyUsage {
	this := LogsByRetentionMonthlyUsage{}
	return &this
}

// NewLogsByRetentionMonthlyUsageWithDefaults instantiates a new LogsByRetentionMonthlyUsage object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsByRetentionMonthlyUsageWithDefaults() *LogsByRetentionMonthlyUsage {
	this := LogsByRetentionMonthlyUsage{}
	return &this
}

// GetDate returns the Date field value if set, zero value otherwise.
func (o *LogsByRetentionMonthlyUsage) GetDate() time.Time {
	if o == nil || o.Date == nil {
		var ret time.Time
		return ret
	}
	return *o.Date
}

// GetDateOk returns a tuple with the Date field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsByRetentionMonthlyUsage) GetDateOk() (*time.Time, bool) {
	if o == nil || o.Date == nil {
		return nil, false
	}
	return o.Date, true
}

// HasDate returns a boolean if a field has been set.
func (o *LogsByRetentionMonthlyUsage) HasDate() bool {
	if o != nil && o.Date != nil {
		return true
	}

	return false
}

// SetDate gets a reference to the given time.Time and assigns it to the Date field.
func (o *LogsByRetentionMonthlyUsage) SetDate(v time.Time) {
	o.Date = &v
}

// GetUsage returns the Usage field value if set, zero value otherwise.
func (o *LogsByRetentionMonthlyUsage) GetUsage() []LogsRetentionSumUsage {
	if o == nil || o.Usage == nil {
		var ret []LogsRetentionSumUsage
		return ret
	}
	return o.Usage
}

// GetUsageOk returns a tuple with the Usage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsByRetentionMonthlyUsage) GetUsageOk() (*[]LogsRetentionSumUsage, bool) {
	if o == nil || o.Usage == nil {
		return nil, false
	}
	return &o.Usage, true
}

// HasUsage returns a boolean if a field has been set.
func (o *LogsByRetentionMonthlyUsage) HasUsage() bool {
	if o != nil && o.Usage != nil {
		return true
	}

	return false
}

// SetUsage gets a reference to the given []LogsRetentionSumUsage and assigns it to the Usage field.
func (o *LogsByRetentionMonthlyUsage) SetUsage(v []LogsRetentionSumUsage) {
	o.Usage = v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsByRetentionMonthlyUsage) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Date != nil {
		if o.Date.Nanosecond() == 0 {
			toSerialize["date"] = o.Date.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["date"] = o.Date.Format("2006-01-02T15:04:05.000Z07:00")
		}
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
func (o *LogsByRetentionMonthlyUsage) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Date  *time.Time              `json:"date,omitempty"`
		Usage []LogsRetentionSumUsage `json:"usage,omitempty"`
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
	o.Date = all.Date
	o.Usage = all.Usage
	return nil
}
