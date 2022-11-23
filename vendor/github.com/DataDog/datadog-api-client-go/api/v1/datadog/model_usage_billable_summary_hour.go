// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"time"
)

// UsageBillableSummaryHour Response with monthly summary of data billed by Datadog.
type UsageBillableSummaryHour struct {
	// The billing plan.
	BillingPlan *string `json:"billing_plan,omitempty"`
	// Shows the last date of usage.
	EndDate *time.Time `json:"end_date,omitempty"`
	// The number of organizations.
	NumOrgs *int64 `json:"num_orgs,omitempty"`
	// The organization name.
	OrgName *string `json:"org_name,omitempty"`
	// The organization public ID.
	PublicId *string `json:"public_id,omitempty"`
	// Shows usage aggregation for a billing period.
	RatioInMonth *float64 `json:"ratio_in_month,omitempty"`
	// Shows the first date of usage.
	StartDate *time.Time `json:"start_date,omitempty"`
	// Response with aggregated usage types.
	Usage *UsageBillableSummaryKeys `json:"usage,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageBillableSummaryHour instantiates a new UsageBillableSummaryHour object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageBillableSummaryHour() *UsageBillableSummaryHour {
	this := UsageBillableSummaryHour{}
	return &this
}

// NewUsageBillableSummaryHourWithDefaults instantiates a new UsageBillableSummaryHour object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageBillableSummaryHourWithDefaults() *UsageBillableSummaryHour {
	this := UsageBillableSummaryHour{}
	return &this
}

// GetBillingPlan returns the BillingPlan field value if set, zero value otherwise.
func (o *UsageBillableSummaryHour) GetBillingPlan() string {
	if o == nil || o.BillingPlan == nil {
		var ret string
		return ret
	}
	return *o.BillingPlan
}

// GetBillingPlanOk returns a tuple with the BillingPlan field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryHour) GetBillingPlanOk() (*string, bool) {
	if o == nil || o.BillingPlan == nil {
		return nil, false
	}
	return o.BillingPlan, true
}

// HasBillingPlan returns a boolean if a field has been set.
func (o *UsageBillableSummaryHour) HasBillingPlan() bool {
	if o != nil && o.BillingPlan != nil {
		return true
	}

	return false
}

// SetBillingPlan gets a reference to the given string and assigns it to the BillingPlan field.
func (o *UsageBillableSummaryHour) SetBillingPlan(v string) {
	o.BillingPlan = &v
}

// GetEndDate returns the EndDate field value if set, zero value otherwise.
func (o *UsageBillableSummaryHour) GetEndDate() time.Time {
	if o == nil || o.EndDate == nil {
		var ret time.Time
		return ret
	}
	return *o.EndDate
}

// GetEndDateOk returns a tuple with the EndDate field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryHour) GetEndDateOk() (*time.Time, bool) {
	if o == nil || o.EndDate == nil {
		return nil, false
	}
	return o.EndDate, true
}

// HasEndDate returns a boolean if a field has been set.
func (o *UsageBillableSummaryHour) HasEndDate() bool {
	if o != nil && o.EndDate != nil {
		return true
	}

	return false
}

// SetEndDate gets a reference to the given time.Time and assigns it to the EndDate field.
func (o *UsageBillableSummaryHour) SetEndDate(v time.Time) {
	o.EndDate = &v
}

// GetNumOrgs returns the NumOrgs field value if set, zero value otherwise.
func (o *UsageBillableSummaryHour) GetNumOrgs() int64 {
	if o == nil || o.NumOrgs == nil {
		var ret int64
		return ret
	}
	return *o.NumOrgs
}

// GetNumOrgsOk returns a tuple with the NumOrgs field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryHour) GetNumOrgsOk() (*int64, bool) {
	if o == nil || o.NumOrgs == nil {
		return nil, false
	}
	return o.NumOrgs, true
}

// HasNumOrgs returns a boolean if a field has been set.
func (o *UsageBillableSummaryHour) HasNumOrgs() bool {
	if o != nil && o.NumOrgs != nil {
		return true
	}

	return false
}

// SetNumOrgs gets a reference to the given int64 and assigns it to the NumOrgs field.
func (o *UsageBillableSummaryHour) SetNumOrgs(v int64) {
	o.NumOrgs = &v
}

// GetOrgName returns the OrgName field value if set, zero value otherwise.
func (o *UsageBillableSummaryHour) GetOrgName() string {
	if o == nil || o.OrgName == nil {
		var ret string
		return ret
	}
	return *o.OrgName
}

// GetOrgNameOk returns a tuple with the OrgName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryHour) GetOrgNameOk() (*string, bool) {
	if o == nil || o.OrgName == nil {
		return nil, false
	}
	return o.OrgName, true
}

// HasOrgName returns a boolean if a field has been set.
func (o *UsageBillableSummaryHour) HasOrgName() bool {
	if o != nil && o.OrgName != nil {
		return true
	}

	return false
}

// SetOrgName gets a reference to the given string and assigns it to the OrgName field.
func (o *UsageBillableSummaryHour) SetOrgName(v string) {
	o.OrgName = &v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *UsageBillableSummaryHour) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryHour) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *UsageBillableSummaryHour) HasPublicId() bool {
	if o != nil && o.PublicId != nil {
		return true
	}

	return false
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *UsageBillableSummaryHour) SetPublicId(v string) {
	o.PublicId = &v
}

// GetRatioInMonth returns the RatioInMonth field value if set, zero value otherwise.
func (o *UsageBillableSummaryHour) GetRatioInMonth() float64 {
	if o == nil || o.RatioInMonth == nil {
		var ret float64
		return ret
	}
	return *o.RatioInMonth
}

// GetRatioInMonthOk returns a tuple with the RatioInMonth field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryHour) GetRatioInMonthOk() (*float64, bool) {
	if o == nil || o.RatioInMonth == nil {
		return nil, false
	}
	return o.RatioInMonth, true
}

// HasRatioInMonth returns a boolean if a field has been set.
func (o *UsageBillableSummaryHour) HasRatioInMonth() bool {
	if o != nil && o.RatioInMonth != nil {
		return true
	}

	return false
}

// SetRatioInMonth gets a reference to the given float64 and assigns it to the RatioInMonth field.
func (o *UsageBillableSummaryHour) SetRatioInMonth(v float64) {
	o.RatioInMonth = &v
}

// GetStartDate returns the StartDate field value if set, zero value otherwise.
func (o *UsageBillableSummaryHour) GetStartDate() time.Time {
	if o == nil || o.StartDate == nil {
		var ret time.Time
		return ret
	}
	return *o.StartDate
}

// GetStartDateOk returns a tuple with the StartDate field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryHour) GetStartDateOk() (*time.Time, bool) {
	if o == nil || o.StartDate == nil {
		return nil, false
	}
	return o.StartDate, true
}

// HasStartDate returns a boolean if a field has been set.
func (o *UsageBillableSummaryHour) HasStartDate() bool {
	if o != nil && o.StartDate != nil {
		return true
	}

	return false
}

// SetStartDate gets a reference to the given time.Time and assigns it to the StartDate field.
func (o *UsageBillableSummaryHour) SetStartDate(v time.Time) {
	o.StartDate = &v
}

// GetUsage returns the Usage field value if set, zero value otherwise.
func (o *UsageBillableSummaryHour) GetUsage() UsageBillableSummaryKeys {
	if o == nil || o.Usage == nil {
		var ret UsageBillableSummaryKeys
		return ret
	}
	return *o.Usage
}

// GetUsageOk returns a tuple with the Usage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryHour) GetUsageOk() (*UsageBillableSummaryKeys, bool) {
	if o == nil || o.Usage == nil {
		return nil, false
	}
	return o.Usage, true
}

// HasUsage returns a boolean if a field has been set.
func (o *UsageBillableSummaryHour) HasUsage() bool {
	if o != nil && o.Usage != nil {
		return true
	}

	return false
}

// SetUsage gets a reference to the given UsageBillableSummaryKeys and assigns it to the Usage field.
func (o *UsageBillableSummaryHour) SetUsage(v UsageBillableSummaryKeys) {
	o.Usage = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageBillableSummaryHour) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.BillingPlan != nil {
		toSerialize["billing_plan"] = o.BillingPlan
	}
	if o.EndDate != nil {
		if o.EndDate.Nanosecond() == 0 {
			toSerialize["end_date"] = o.EndDate.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["end_date"] = o.EndDate.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.NumOrgs != nil {
		toSerialize["num_orgs"] = o.NumOrgs
	}
	if o.OrgName != nil {
		toSerialize["org_name"] = o.OrgName
	}
	if o.PublicId != nil {
		toSerialize["public_id"] = o.PublicId
	}
	if o.RatioInMonth != nil {
		toSerialize["ratio_in_month"] = o.RatioInMonth
	}
	if o.StartDate != nil {
		if o.StartDate.Nanosecond() == 0 {
			toSerialize["start_date"] = o.StartDate.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["start_date"] = o.StartDate.Format("2006-01-02T15:04:05.000Z07:00")
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
func (o *UsageBillableSummaryHour) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		BillingPlan  *string                   `json:"billing_plan,omitempty"`
		EndDate      *time.Time                `json:"end_date,omitempty"`
		NumOrgs      *int64                    `json:"num_orgs,omitempty"`
		OrgName      *string                   `json:"org_name,omitempty"`
		PublicId     *string                   `json:"public_id,omitempty"`
		RatioInMonth *float64                  `json:"ratio_in_month,omitempty"`
		StartDate    *time.Time                `json:"start_date,omitempty"`
		Usage        *UsageBillableSummaryKeys `json:"usage,omitempty"`
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
	o.BillingPlan = all.BillingPlan
	o.EndDate = all.EndDate
	o.NumOrgs = all.NumOrgs
	o.OrgName = all.OrgName
	o.PublicId = all.PublicId
	o.RatioInMonth = all.RatioInMonth
	o.StartDate = all.StartDate
	if all.Usage != nil && all.Usage.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Usage = all.Usage
	return nil
}
