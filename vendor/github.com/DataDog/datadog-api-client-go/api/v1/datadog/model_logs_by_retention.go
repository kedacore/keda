// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// LogsByRetention Object containing logs usage data broken down by retention period.
type LogsByRetention struct {
	// Indexed logs usage summary for each organization for each retention period with usage.
	Orgs *LogsByRetentionOrgs `json:"orgs,omitempty"`
	// Aggregated index logs usage for each retention period with usage.
	Usage []LogsRetentionAggSumUsage `json:"usage,omitempty"`
	// Object containing a summary of indexed logs usage by retention period for a single month.
	UsageByMonth *LogsByRetentionMonthlyUsage `json:"usage_by_month,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewLogsByRetention instantiates a new LogsByRetention object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsByRetention() *LogsByRetention {
	this := LogsByRetention{}
	return &this
}

// NewLogsByRetentionWithDefaults instantiates a new LogsByRetention object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsByRetentionWithDefaults() *LogsByRetention {
	this := LogsByRetention{}
	return &this
}

// GetOrgs returns the Orgs field value if set, zero value otherwise.
func (o *LogsByRetention) GetOrgs() LogsByRetentionOrgs {
	if o == nil || o.Orgs == nil {
		var ret LogsByRetentionOrgs
		return ret
	}
	return *o.Orgs
}

// GetOrgsOk returns a tuple with the Orgs field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsByRetention) GetOrgsOk() (*LogsByRetentionOrgs, bool) {
	if o == nil || o.Orgs == nil {
		return nil, false
	}
	return o.Orgs, true
}

// HasOrgs returns a boolean if a field has been set.
func (o *LogsByRetention) HasOrgs() bool {
	if o != nil && o.Orgs != nil {
		return true
	}

	return false
}

// SetOrgs gets a reference to the given LogsByRetentionOrgs and assigns it to the Orgs field.
func (o *LogsByRetention) SetOrgs(v LogsByRetentionOrgs) {
	o.Orgs = &v
}

// GetUsage returns the Usage field value if set, zero value otherwise.
func (o *LogsByRetention) GetUsage() []LogsRetentionAggSumUsage {
	if o == nil || o.Usage == nil {
		var ret []LogsRetentionAggSumUsage
		return ret
	}
	return o.Usage
}

// GetUsageOk returns a tuple with the Usage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsByRetention) GetUsageOk() (*[]LogsRetentionAggSumUsage, bool) {
	if o == nil || o.Usage == nil {
		return nil, false
	}
	return &o.Usage, true
}

// HasUsage returns a boolean if a field has been set.
func (o *LogsByRetention) HasUsage() bool {
	if o != nil && o.Usage != nil {
		return true
	}

	return false
}

// SetUsage gets a reference to the given []LogsRetentionAggSumUsage and assigns it to the Usage field.
func (o *LogsByRetention) SetUsage(v []LogsRetentionAggSumUsage) {
	o.Usage = v
}

// GetUsageByMonth returns the UsageByMonth field value if set, zero value otherwise.
func (o *LogsByRetention) GetUsageByMonth() LogsByRetentionMonthlyUsage {
	if o == nil || o.UsageByMonth == nil {
		var ret LogsByRetentionMonthlyUsage
		return ret
	}
	return *o.UsageByMonth
}

// GetUsageByMonthOk returns a tuple with the UsageByMonth field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsByRetention) GetUsageByMonthOk() (*LogsByRetentionMonthlyUsage, bool) {
	if o == nil || o.UsageByMonth == nil {
		return nil, false
	}
	return o.UsageByMonth, true
}

// HasUsageByMonth returns a boolean if a field has been set.
func (o *LogsByRetention) HasUsageByMonth() bool {
	if o != nil && o.UsageByMonth != nil {
		return true
	}

	return false
}

// SetUsageByMonth gets a reference to the given LogsByRetentionMonthlyUsage and assigns it to the UsageByMonth field.
func (o *LogsByRetention) SetUsageByMonth(v LogsByRetentionMonthlyUsage) {
	o.UsageByMonth = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsByRetention) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Orgs != nil {
		toSerialize["orgs"] = o.Orgs
	}
	if o.Usage != nil {
		toSerialize["usage"] = o.Usage
	}
	if o.UsageByMonth != nil {
		toSerialize["usage_by_month"] = o.UsageByMonth
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsByRetention) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Orgs         *LogsByRetentionOrgs         `json:"orgs,omitempty"`
		Usage        []LogsRetentionAggSumUsage   `json:"usage,omitempty"`
		UsageByMonth *LogsByRetentionMonthlyUsage `json:"usage_by_month,omitempty"`
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
	if all.Orgs != nil && all.Orgs.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Orgs = all.Orgs
	o.Usage = all.Usage
	if all.UsageByMonth != nil && all.UsageByMonth.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.UsageByMonth = all.UsageByMonth
	return nil
}
