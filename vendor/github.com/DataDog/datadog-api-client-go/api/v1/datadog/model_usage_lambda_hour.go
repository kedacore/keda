// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"time"
)

// UsageLambdaHour Number of lambda functions and sum of the invocations of all lambda functions
// for each hour for a given organization.
type UsageLambdaHour struct {
	// Contains the number of different functions for each region and AWS account.
	FuncCount *int64 `json:"func_count,omitempty"`
	// The hour for the usage.
	Hour *time.Time `json:"hour,omitempty"`
	// Contains the sum of invocations of all functions.
	InvocationsSum *int64 `json:"invocations_sum,omitempty"`
	// The organization name.
	OrgName *string `json:"org_name,omitempty"`
	// The organization public ID.
	PublicId *string `json:"public_id,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageLambdaHour instantiates a new UsageLambdaHour object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageLambdaHour() *UsageLambdaHour {
	this := UsageLambdaHour{}
	return &this
}

// NewUsageLambdaHourWithDefaults instantiates a new UsageLambdaHour object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageLambdaHourWithDefaults() *UsageLambdaHour {
	this := UsageLambdaHour{}
	return &this
}

// GetFuncCount returns the FuncCount field value if set, zero value otherwise.
func (o *UsageLambdaHour) GetFuncCount() int64 {
	if o == nil || o.FuncCount == nil {
		var ret int64
		return ret
	}
	return *o.FuncCount
}

// GetFuncCountOk returns a tuple with the FuncCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLambdaHour) GetFuncCountOk() (*int64, bool) {
	if o == nil || o.FuncCount == nil {
		return nil, false
	}
	return o.FuncCount, true
}

// HasFuncCount returns a boolean if a field has been set.
func (o *UsageLambdaHour) HasFuncCount() bool {
	if o != nil && o.FuncCount != nil {
		return true
	}

	return false
}

// SetFuncCount gets a reference to the given int64 and assigns it to the FuncCount field.
func (o *UsageLambdaHour) SetFuncCount(v int64) {
	o.FuncCount = &v
}

// GetHour returns the Hour field value if set, zero value otherwise.
func (o *UsageLambdaHour) GetHour() time.Time {
	if o == nil || o.Hour == nil {
		var ret time.Time
		return ret
	}
	return *o.Hour
}

// GetHourOk returns a tuple with the Hour field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLambdaHour) GetHourOk() (*time.Time, bool) {
	if o == nil || o.Hour == nil {
		return nil, false
	}
	return o.Hour, true
}

// HasHour returns a boolean if a field has been set.
func (o *UsageLambdaHour) HasHour() bool {
	if o != nil && o.Hour != nil {
		return true
	}

	return false
}

// SetHour gets a reference to the given time.Time and assigns it to the Hour field.
func (o *UsageLambdaHour) SetHour(v time.Time) {
	o.Hour = &v
}

// GetInvocationsSum returns the InvocationsSum field value if set, zero value otherwise.
func (o *UsageLambdaHour) GetInvocationsSum() int64 {
	if o == nil || o.InvocationsSum == nil {
		var ret int64
		return ret
	}
	return *o.InvocationsSum
}

// GetInvocationsSumOk returns a tuple with the InvocationsSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLambdaHour) GetInvocationsSumOk() (*int64, bool) {
	if o == nil || o.InvocationsSum == nil {
		return nil, false
	}
	return o.InvocationsSum, true
}

// HasInvocationsSum returns a boolean if a field has been set.
func (o *UsageLambdaHour) HasInvocationsSum() bool {
	if o != nil && o.InvocationsSum != nil {
		return true
	}

	return false
}

// SetInvocationsSum gets a reference to the given int64 and assigns it to the InvocationsSum field.
func (o *UsageLambdaHour) SetInvocationsSum(v int64) {
	o.InvocationsSum = &v
}

// GetOrgName returns the OrgName field value if set, zero value otherwise.
func (o *UsageLambdaHour) GetOrgName() string {
	if o == nil || o.OrgName == nil {
		var ret string
		return ret
	}
	return *o.OrgName
}

// GetOrgNameOk returns a tuple with the OrgName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLambdaHour) GetOrgNameOk() (*string, bool) {
	if o == nil || o.OrgName == nil {
		return nil, false
	}
	return o.OrgName, true
}

// HasOrgName returns a boolean if a field has been set.
func (o *UsageLambdaHour) HasOrgName() bool {
	if o != nil && o.OrgName != nil {
		return true
	}

	return false
}

// SetOrgName gets a reference to the given string and assigns it to the OrgName field.
func (o *UsageLambdaHour) SetOrgName(v string) {
	o.OrgName = &v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *UsageLambdaHour) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageLambdaHour) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *UsageLambdaHour) HasPublicId() bool {
	if o != nil && o.PublicId != nil {
		return true
	}

	return false
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *UsageLambdaHour) SetPublicId(v string) {
	o.PublicId = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageLambdaHour) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.FuncCount != nil {
		toSerialize["func_count"] = o.FuncCount
	}
	if o.Hour != nil {
		if o.Hour.Nanosecond() == 0 {
			toSerialize["hour"] = o.Hour.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["hour"] = o.Hour.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.InvocationsSum != nil {
		toSerialize["invocations_sum"] = o.InvocationsSum
	}
	if o.OrgName != nil {
		toSerialize["org_name"] = o.OrgName
	}
	if o.PublicId != nil {
		toSerialize["public_id"] = o.PublicId
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UsageLambdaHour) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		FuncCount      *int64     `json:"func_count,omitempty"`
		Hour           *time.Time `json:"hour,omitempty"`
		InvocationsSum *int64     `json:"invocations_sum,omitempty"`
		OrgName        *string    `json:"org_name,omitempty"`
		PublicId       *string    `json:"public_id,omitempty"`
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
	o.FuncCount = all.FuncCount
	o.Hour = all.Hour
	o.InvocationsSum = all.InvocationsSum
	o.OrgName = all.OrgName
	o.PublicId = all.PublicId
	return nil
}
