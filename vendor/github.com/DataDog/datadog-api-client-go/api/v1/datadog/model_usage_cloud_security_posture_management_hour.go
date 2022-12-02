// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"time"
)

// UsageCloudSecurityPostureManagementHour Cloud Security Posture Management usage for a given organization for a given hour.
type UsageCloudSecurityPostureManagementHour struct {
	// The number of Cloud Security Posture Management Azure app services hosts during a given hour.
	AasHostCount NullableFloat64 `json:"aas_host_count,omitempty"`
	// The number of Cloud Security Posture Management Azure hosts during a given hour.
	AzureHostCount NullableFloat64 `json:"azure_host_count,omitempty"`
	// The number of Cloud Security Posture Management hosts during a given hour.
	ComplianceHostCount NullableFloat64 `json:"compliance_host_count,omitempty"`
	// The total number of Cloud Security Posture Management containers during a given hour.
	ContainerCount NullableFloat64 `json:"container_count,omitempty"`
	// The total number of Cloud Security Posture Management hosts during a given hour.
	HostCount NullableFloat64 `json:"host_count,omitempty"`
	// The hour for the usage.
	Hour *time.Time `json:"hour,omitempty"`
	// The organization name.
	OrgName *string `json:"org_name,omitempty"`
	// The organization public ID.
	PublicId *string `json:"public_id,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageCloudSecurityPostureManagementHour instantiates a new UsageCloudSecurityPostureManagementHour object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageCloudSecurityPostureManagementHour() *UsageCloudSecurityPostureManagementHour {
	this := UsageCloudSecurityPostureManagementHour{}
	return &this
}

// NewUsageCloudSecurityPostureManagementHourWithDefaults instantiates a new UsageCloudSecurityPostureManagementHour object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageCloudSecurityPostureManagementHourWithDefaults() *UsageCloudSecurityPostureManagementHour {
	this := UsageCloudSecurityPostureManagementHour{}
	return &this
}

// GetAasHostCount returns the AasHostCount field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *UsageCloudSecurityPostureManagementHour) GetAasHostCount() float64 {
	if o == nil || o.AasHostCount.Get() == nil {
		var ret float64
		return ret
	}
	return *o.AasHostCount.Get()
}

// GetAasHostCountOk returns a tuple with the AasHostCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *UsageCloudSecurityPostureManagementHour) GetAasHostCountOk() (*float64, bool) {
	if o == nil {
		return nil, false
	}
	return o.AasHostCount.Get(), o.AasHostCount.IsSet()
}

// HasAasHostCount returns a boolean if a field has been set.
func (o *UsageCloudSecurityPostureManagementHour) HasAasHostCount() bool {
	if o != nil && o.AasHostCount.IsSet() {
		return true
	}

	return false
}

// SetAasHostCount gets a reference to the given NullableFloat64 and assigns it to the AasHostCount field.
func (o *UsageCloudSecurityPostureManagementHour) SetAasHostCount(v float64) {
	o.AasHostCount.Set(&v)
}

// SetAasHostCountNil sets the value for AasHostCount to be an explicit nil.
func (o *UsageCloudSecurityPostureManagementHour) SetAasHostCountNil() {
	o.AasHostCount.Set(nil)
}

// UnsetAasHostCount ensures that no value is present for AasHostCount, not even an explicit nil.
func (o *UsageCloudSecurityPostureManagementHour) UnsetAasHostCount() {
	o.AasHostCount.Unset()
}

// GetAzureHostCount returns the AzureHostCount field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *UsageCloudSecurityPostureManagementHour) GetAzureHostCount() float64 {
	if o == nil || o.AzureHostCount.Get() == nil {
		var ret float64
		return ret
	}
	return *o.AzureHostCount.Get()
}

// GetAzureHostCountOk returns a tuple with the AzureHostCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *UsageCloudSecurityPostureManagementHour) GetAzureHostCountOk() (*float64, bool) {
	if o == nil {
		return nil, false
	}
	return o.AzureHostCount.Get(), o.AzureHostCount.IsSet()
}

// HasAzureHostCount returns a boolean if a field has been set.
func (o *UsageCloudSecurityPostureManagementHour) HasAzureHostCount() bool {
	if o != nil && o.AzureHostCount.IsSet() {
		return true
	}

	return false
}

// SetAzureHostCount gets a reference to the given NullableFloat64 and assigns it to the AzureHostCount field.
func (o *UsageCloudSecurityPostureManagementHour) SetAzureHostCount(v float64) {
	o.AzureHostCount.Set(&v)
}

// SetAzureHostCountNil sets the value for AzureHostCount to be an explicit nil.
func (o *UsageCloudSecurityPostureManagementHour) SetAzureHostCountNil() {
	o.AzureHostCount.Set(nil)
}

// UnsetAzureHostCount ensures that no value is present for AzureHostCount, not even an explicit nil.
func (o *UsageCloudSecurityPostureManagementHour) UnsetAzureHostCount() {
	o.AzureHostCount.Unset()
}

// GetComplianceHostCount returns the ComplianceHostCount field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *UsageCloudSecurityPostureManagementHour) GetComplianceHostCount() float64 {
	if o == nil || o.ComplianceHostCount.Get() == nil {
		var ret float64
		return ret
	}
	return *o.ComplianceHostCount.Get()
}

// GetComplianceHostCountOk returns a tuple with the ComplianceHostCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *UsageCloudSecurityPostureManagementHour) GetComplianceHostCountOk() (*float64, bool) {
	if o == nil {
		return nil, false
	}
	return o.ComplianceHostCount.Get(), o.ComplianceHostCount.IsSet()
}

// HasComplianceHostCount returns a boolean if a field has been set.
func (o *UsageCloudSecurityPostureManagementHour) HasComplianceHostCount() bool {
	if o != nil && o.ComplianceHostCount.IsSet() {
		return true
	}

	return false
}

// SetComplianceHostCount gets a reference to the given NullableFloat64 and assigns it to the ComplianceHostCount field.
func (o *UsageCloudSecurityPostureManagementHour) SetComplianceHostCount(v float64) {
	o.ComplianceHostCount.Set(&v)
}

// SetComplianceHostCountNil sets the value for ComplianceHostCount to be an explicit nil.
func (o *UsageCloudSecurityPostureManagementHour) SetComplianceHostCountNil() {
	o.ComplianceHostCount.Set(nil)
}

// UnsetComplianceHostCount ensures that no value is present for ComplianceHostCount, not even an explicit nil.
func (o *UsageCloudSecurityPostureManagementHour) UnsetComplianceHostCount() {
	o.ComplianceHostCount.Unset()
}

// GetContainerCount returns the ContainerCount field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *UsageCloudSecurityPostureManagementHour) GetContainerCount() float64 {
	if o == nil || o.ContainerCount.Get() == nil {
		var ret float64
		return ret
	}
	return *o.ContainerCount.Get()
}

// GetContainerCountOk returns a tuple with the ContainerCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *UsageCloudSecurityPostureManagementHour) GetContainerCountOk() (*float64, bool) {
	if o == nil {
		return nil, false
	}
	return o.ContainerCount.Get(), o.ContainerCount.IsSet()
}

// HasContainerCount returns a boolean if a field has been set.
func (o *UsageCloudSecurityPostureManagementHour) HasContainerCount() bool {
	if o != nil && o.ContainerCount.IsSet() {
		return true
	}

	return false
}

// SetContainerCount gets a reference to the given NullableFloat64 and assigns it to the ContainerCount field.
func (o *UsageCloudSecurityPostureManagementHour) SetContainerCount(v float64) {
	o.ContainerCount.Set(&v)
}

// SetContainerCountNil sets the value for ContainerCount to be an explicit nil.
func (o *UsageCloudSecurityPostureManagementHour) SetContainerCountNil() {
	o.ContainerCount.Set(nil)
}

// UnsetContainerCount ensures that no value is present for ContainerCount, not even an explicit nil.
func (o *UsageCloudSecurityPostureManagementHour) UnsetContainerCount() {
	o.ContainerCount.Unset()
}

// GetHostCount returns the HostCount field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *UsageCloudSecurityPostureManagementHour) GetHostCount() float64 {
	if o == nil || o.HostCount.Get() == nil {
		var ret float64
		return ret
	}
	return *o.HostCount.Get()
}

// GetHostCountOk returns a tuple with the HostCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *UsageCloudSecurityPostureManagementHour) GetHostCountOk() (*float64, bool) {
	if o == nil {
		return nil, false
	}
	return o.HostCount.Get(), o.HostCount.IsSet()
}

// HasHostCount returns a boolean if a field has been set.
func (o *UsageCloudSecurityPostureManagementHour) HasHostCount() bool {
	if o != nil && o.HostCount.IsSet() {
		return true
	}

	return false
}

// SetHostCount gets a reference to the given NullableFloat64 and assigns it to the HostCount field.
func (o *UsageCloudSecurityPostureManagementHour) SetHostCount(v float64) {
	o.HostCount.Set(&v)
}

// SetHostCountNil sets the value for HostCount to be an explicit nil.
func (o *UsageCloudSecurityPostureManagementHour) SetHostCountNil() {
	o.HostCount.Set(nil)
}

// UnsetHostCount ensures that no value is present for HostCount, not even an explicit nil.
func (o *UsageCloudSecurityPostureManagementHour) UnsetHostCount() {
	o.HostCount.Unset()
}

// GetHour returns the Hour field value if set, zero value otherwise.
func (o *UsageCloudSecurityPostureManagementHour) GetHour() time.Time {
	if o == nil || o.Hour == nil {
		var ret time.Time
		return ret
	}
	return *o.Hour
}

// GetHourOk returns a tuple with the Hour field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageCloudSecurityPostureManagementHour) GetHourOk() (*time.Time, bool) {
	if o == nil || o.Hour == nil {
		return nil, false
	}
	return o.Hour, true
}

// HasHour returns a boolean if a field has been set.
func (o *UsageCloudSecurityPostureManagementHour) HasHour() bool {
	if o != nil && o.Hour != nil {
		return true
	}

	return false
}

// SetHour gets a reference to the given time.Time and assigns it to the Hour field.
func (o *UsageCloudSecurityPostureManagementHour) SetHour(v time.Time) {
	o.Hour = &v
}

// GetOrgName returns the OrgName field value if set, zero value otherwise.
func (o *UsageCloudSecurityPostureManagementHour) GetOrgName() string {
	if o == nil || o.OrgName == nil {
		var ret string
		return ret
	}
	return *o.OrgName
}

// GetOrgNameOk returns a tuple with the OrgName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageCloudSecurityPostureManagementHour) GetOrgNameOk() (*string, bool) {
	if o == nil || o.OrgName == nil {
		return nil, false
	}
	return o.OrgName, true
}

// HasOrgName returns a boolean if a field has been set.
func (o *UsageCloudSecurityPostureManagementHour) HasOrgName() bool {
	if o != nil && o.OrgName != nil {
		return true
	}

	return false
}

// SetOrgName gets a reference to the given string and assigns it to the OrgName field.
func (o *UsageCloudSecurityPostureManagementHour) SetOrgName(v string) {
	o.OrgName = &v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *UsageCloudSecurityPostureManagementHour) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageCloudSecurityPostureManagementHour) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *UsageCloudSecurityPostureManagementHour) HasPublicId() bool {
	if o != nil && o.PublicId != nil {
		return true
	}

	return false
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *UsageCloudSecurityPostureManagementHour) SetPublicId(v string) {
	o.PublicId = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageCloudSecurityPostureManagementHour) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AasHostCount.IsSet() {
		toSerialize["aas_host_count"] = o.AasHostCount.Get()
	}
	if o.AzureHostCount.IsSet() {
		toSerialize["azure_host_count"] = o.AzureHostCount.Get()
	}
	if o.ComplianceHostCount.IsSet() {
		toSerialize["compliance_host_count"] = o.ComplianceHostCount.Get()
	}
	if o.ContainerCount.IsSet() {
		toSerialize["container_count"] = o.ContainerCount.Get()
	}
	if o.HostCount.IsSet() {
		toSerialize["host_count"] = o.HostCount.Get()
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

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UsageCloudSecurityPostureManagementHour) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		AasHostCount        NullableFloat64 `json:"aas_host_count,omitempty"`
		AzureHostCount      NullableFloat64 `json:"azure_host_count,omitempty"`
		ComplianceHostCount NullableFloat64 `json:"compliance_host_count,omitempty"`
		ContainerCount      NullableFloat64 `json:"container_count,omitempty"`
		HostCount           NullableFloat64 `json:"host_count,omitempty"`
		Hour                *time.Time      `json:"hour,omitempty"`
		OrgName             *string         `json:"org_name,omitempty"`
		PublicId            *string         `json:"public_id,omitempty"`
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
	o.AasHostCount = all.AasHostCount
	o.AzureHostCount = all.AzureHostCount
	o.ComplianceHostCount = all.ComplianceHostCount
	o.ContainerCount = all.ContainerCount
	o.HostCount = all.HostCount
	o.Hour = all.Hour
	o.OrgName = all.OrgName
	o.PublicId = all.PublicId
	return nil
}
