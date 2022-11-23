// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"time"
)

// HourlyUsageAttributionBody The usage for one set of tags for one hour.
type HourlyUsageAttributionBody struct {
	// The hour for the usage.
	Hour *time.Time `json:"hour,omitempty"`
	// The name of the organization.
	OrgName *string `json:"org_name,omitempty"`
	// The organization public ID.
	PublicId *string `json:"public_id,omitempty"`
	// The source of the usage attribution tag configuration and the selected tags in the format of `<source_org_name>:::<selected tag 1>///<selected tag 2>///<selected tag 3>`.
	TagConfigSource *string `json:"tag_config_source,omitempty"`
	// Tag keys and values.
	//
	// A `null` value here means that the requested tag breakdown cannot be applied because it does not match the [tags
	// configured for usage attribution](https://docs.datadoghq.com/account_management/billing/usage_attribution/#getting-started).
	// In this scenario the API returns the total usage, not broken down by tags.
	Tags map[string][]string `json:"tags,omitempty"`
	// Total product usage for the given tags within the hour.
	TotalUsageSum *float64 `json:"total_usage_sum,omitempty"`
	// Shows the most recent hour in the current month for all organizations where usages are calculated.
	UpdatedAt *string `json:"updated_at,omitempty"`
	// Supported products for hourly usage attribution requests.
	UsageType *HourlyUsageAttributionUsageType `json:"usage_type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewHourlyUsageAttributionBody instantiates a new HourlyUsageAttributionBody object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewHourlyUsageAttributionBody() *HourlyUsageAttributionBody {
	this := HourlyUsageAttributionBody{}
	return &this
}

// NewHourlyUsageAttributionBodyWithDefaults instantiates a new HourlyUsageAttributionBody object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewHourlyUsageAttributionBodyWithDefaults() *HourlyUsageAttributionBody {
	this := HourlyUsageAttributionBody{}
	return &this
}

// GetHour returns the Hour field value if set, zero value otherwise.
func (o *HourlyUsageAttributionBody) GetHour() time.Time {
	if o == nil || o.Hour == nil {
		var ret time.Time
		return ret
	}
	return *o.Hour
}

// GetHourOk returns a tuple with the Hour field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HourlyUsageAttributionBody) GetHourOk() (*time.Time, bool) {
	if o == nil || o.Hour == nil {
		return nil, false
	}
	return o.Hour, true
}

// HasHour returns a boolean if a field has been set.
func (o *HourlyUsageAttributionBody) HasHour() bool {
	if o != nil && o.Hour != nil {
		return true
	}

	return false
}

// SetHour gets a reference to the given time.Time and assigns it to the Hour field.
func (o *HourlyUsageAttributionBody) SetHour(v time.Time) {
	o.Hour = &v
}

// GetOrgName returns the OrgName field value if set, zero value otherwise.
func (o *HourlyUsageAttributionBody) GetOrgName() string {
	if o == nil || o.OrgName == nil {
		var ret string
		return ret
	}
	return *o.OrgName
}

// GetOrgNameOk returns a tuple with the OrgName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HourlyUsageAttributionBody) GetOrgNameOk() (*string, bool) {
	if o == nil || o.OrgName == nil {
		return nil, false
	}
	return o.OrgName, true
}

// HasOrgName returns a boolean if a field has been set.
func (o *HourlyUsageAttributionBody) HasOrgName() bool {
	if o != nil && o.OrgName != nil {
		return true
	}

	return false
}

// SetOrgName gets a reference to the given string and assigns it to the OrgName field.
func (o *HourlyUsageAttributionBody) SetOrgName(v string) {
	o.OrgName = &v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *HourlyUsageAttributionBody) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HourlyUsageAttributionBody) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *HourlyUsageAttributionBody) HasPublicId() bool {
	if o != nil && o.PublicId != nil {
		return true
	}

	return false
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *HourlyUsageAttributionBody) SetPublicId(v string) {
	o.PublicId = &v
}

// GetTagConfigSource returns the TagConfigSource field value if set, zero value otherwise.
func (o *HourlyUsageAttributionBody) GetTagConfigSource() string {
	if o == nil || o.TagConfigSource == nil {
		var ret string
		return ret
	}
	return *o.TagConfigSource
}

// GetTagConfigSourceOk returns a tuple with the TagConfigSource field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HourlyUsageAttributionBody) GetTagConfigSourceOk() (*string, bool) {
	if o == nil || o.TagConfigSource == nil {
		return nil, false
	}
	return o.TagConfigSource, true
}

// HasTagConfigSource returns a boolean if a field has been set.
func (o *HourlyUsageAttributionBody) HasTagConfigSource() bool {
	if o != nil && o.TagConfigSource != nil {
		return true
	}

	return false
}

// SetTagConfigSource gets a reference to the given string and assigns it to the TagConfigSource field.
func (o *HourlyUsageAttributionBody) SetTagConfigSource(v string) {
	o.TagConfigSource = &v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *HourlyUsageAttributionBody) GetTags() map[string][]string {
	if o == nil || o.Tags == nil {
		var ret map[string][]string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HourlyUsageAttributionBody) GetTagsOk() (*map[string][]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *HourlyUsageAttributionBody) HasTags() bool {
	if o != nil && o.Tags != nil {
		return true
	}

	return false
}

// SetTags gets a reference to the given map[string][]string and assigns it to the Tags field.
func (o *HourlyUsageAttributionBody) SetTags(v map[string][]string) {
	o.Tags = v
}

// GetTotalUsageSum returns the TotalUsageSum field value if set, zero value otherwise.
func (o *HourlyUsageAttributionBody) GetTotalUsageSum() float64 {
	if o == nil || o.TotalUsageSum == nil {
		var ret float64
		return ret
	}
	return *o.TotalUsageSum
}

// GetTotalUsageSumOk returns a tuple with the TotalUsageSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HourlyUsageAttributionBody) GetTotalUsageSumOk() (*float64, bool) {
	if o == nil || o.TotalUsageSum == nil {
		return nil, false
	}
	return o.TotalUsageSum, true
}

// HasTotalUsageSum returns a boolean if a field has been set.
func (o *HourlyUsageAttributionBody) HasTotalUsageSum() bool {
	if o != nil && o.TotalUsageSum != nil {
		return true
	}

	return false
}

// SetTotalUsageSum gets a reference to the given float64 and assigns it to the TotalUsageSum field.
func (o *HourlyUsageAttributionBody) SetTotalUsageSum(v float64) {
	o.TotalUsageSum = &v
}

// GetUpdatedAt returns the UpdatedAt field value if set, zero value otherwise.
func (o *HourlyUsageAttributionBody) GetUpdatedAt() string {
	if o == nil || o.UpdatedAt == nil {
		var ret string
		return ret
	}
	return *o.UpdatedAt
}

// GetUpdatedAtOk returns a tuple with the UpdatedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HourlyUsageAttributionBody) GetUpdatedAtOk() (*string, bool) {
	if o == nil || o.UpdatedAt == nil {
		return nil, false
	}
	return o.UpdatedAt, true
}

// HasUpdatedAt returns a boolean if a field has been set.
func (o *HourlyUsageAttributionBody) HasUpdatedAt() bool {
	if o != nil && o.UpdatedAt != nil {
		return true
	}

	return false
}

// SetUpdatedAt gets a reference to the given string and assigns it to the UpdatedAt field.
func (o *HourlyUsageAttributionBody) SetUpdatedAt(v string) {
	o.UpdatedAt = &v
}

// GetUsageType returns the UsageType field value if set, zero value otherwise.
func (o *HourlyUsageAttributionBody) GetUsageType() HourlyUsageAttributionUsageType {
	if o == nil || o.UsageType == nil {
		var ret HourlyUsageAttributionUsageType
		return ret
	}
	return *o.UsageType
}

// GetUsageTypeOk returns a tuple with the UsageType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HourlyUsageAttributionBody) GetUsageTypeOk() (*HourlyUsageAttributionUsageType, bool) {
	if o == nil || o.UsageType == nil {
		return nil, false
	}
	return o.UsageType, true
}

// HasUsageType returns a boolean if a field has been set.
func (o *HourlyUsageAttributionBody) HasUsageType() bool {
	if o != nil && o.UsageType != nil {
		return true
	}

	return false
}

// SetUsageType gets a reference to the given HourlyUsageAttributionUsageType and assigns it to the UsageType field.
func (o *HourlyUsageAttributionBody) SetUsageType(v HourlyUsageAttributionUsageType) {
	o.UsageType = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o HourlyUsageAttributionBody) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Hour != nil {
		if o.Hour.Nanosecond() == 0 {
			toSerialize["hour"] = o.Hour.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["hour"] = o.Hour.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.OrgName != nil {
		toSerialize["org_name"] = o.OrgName
	}
	if o.PublicId != nil {
		toSerialize["public_id"] = o.PublicId
	}
	if o.TagConfigSource != nil {
		toSerialize["tag_config_source"] = o.TagConfigSource
	}
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
	}
	if o.TotalUsageSum != nil {
		toSerialize["total_usage_sum"] = o.TotalUsageSum
	}
	if o.UpdatedAt != nil {
		toSerialize["updated_at"] = o.UpdatedAt
	}
	if o.UsageType != nil {
		toSerialize["usage_type"] = o.UsageType
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *HourlyUsageAttributionBody) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Hour            *time.Time                       `json:"hour,omitempty"`
		OrgName         *string                          `json:"org_name,omitempty"`
		PublicId        *string                          `json:"public_id,omitempty"`
		TagConfigSource *string                          `json:"tag_config_source,omitempty"`
		Tags            map[string][]string              `json:"tags,omitempty"`
		TotalUsageSum   *float64                         `json:"total_usage_sum,omitempty"`
		UpdatedAt       *string                          `json:"updated_at,omitempty"`
		UsageType       *HourlyUsageAttributionUsageType `json:"usage_type,omitempty"`
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
	if v := all.UsageType; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Hour = all.Hour
	o.OrgName = all.OrgName
	o.PublicId = all.PublicId
	o.TagConfigSource = all.TagConfigSource
	o.Tags = all.Tags
	o.TotalUsageSum = all.TotalUsageSum
	o.UpdatedAt = all.UpdatedAt
	o.UsageType = all.UsageType
	return nil
}
