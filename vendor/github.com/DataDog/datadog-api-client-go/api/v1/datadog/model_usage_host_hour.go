// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"time"
)

// UsageHostHour Number of hosts/containers recorded for each hour for a given organization.
type UsageHostHour struct {
	// Contains the total number of infrastructure hosts reporting
	// during a given hour that were running the Datadog Agent.
	AgentHostCount *int64 `json:"agent_host_count,omitempty"`
	// Contains the total number of hosts that reported through Alibaba integration
	// (and were NOT running the Datadog Agent).
	AlibabaHostCount *int64 `json:"alibaba_host_count,omitempty"`
	// Contains the total number of Azure App Services hosts using APM.
	ApmAzureAppServiceHostCount *int64 `json:"apm_azure_app_service_host_count,omitempty"`
	// Shows the total number of hosts using APM during the hour,
	// these are counted as billable (except during trial periods).
	ApmHostCount *int64 `json:"apm_host_count,omitempty"`
	// Contains the total number of hosts that reported through the AWS integration
	// (and were NOT running the Datadog Agent).
	AwsHostCount *int64 `json:"aws_host_count,omitempty"`
	// Contains the total number of hosts that reported through Azure integration
	// (and were NOT running the Datadog Agent).
	AzureHostCount *int64 `json:"azure_host_count,omitempty"`
	// Shows the total number of containers reported by the Docker integration during the hour.
	ContainerCount *int64 `json:"container_count,omitempty"`
	// Contains the total number of hosts that reported through the Google Cloud integration
	// (and were NOT running the Datadog Agent).
	GcpHostCount *int64 `json:"gcp_host_count,omitempty"`
	// Contains the total number of Heroku dynos reported by the Datadog Agent.
	HerokuHostCount *int64 `json:"heroku_host_count,omitempty"`
	// Contains the total number of billable infrastructure hosts reporting during a given hour.
	// This is the sum of `agent_host_count`, `aws_host_count`, and `gcp_host_count`.
	HostCount *int64 `json:"host_count,omitempty"`
	// The hour for the usage.
	Hour *time.Time `json:"hour,omitempty"`
	// Contains the total number of hosts that reported through the Azure App Services integration
	// (and were NOT running the Datadog Agent).
	InfraAzureAppService *int64 `json:"infra_azure_app_service,omitempty"`
	// Contains the total number of hosts reported by Datadog exporter for the OpenTelemetry Collector.
	OpentelemetryHostCount *int64 `json:"opentelemetry_host_count,omitempty"`
	// The organization name.
	OrgName *string `json:"org_name,omitempty"`
	// The organization public ID.
	PublicId *string `json:"public_id,omitempty"`
	// Contains the total number of hosts that reported through vSphere integration
	// (and were NOT running the Datadog Agent).
	VsphereHostCount *int64 `json:"vsphere_host_count,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageHostHour instantiates a new UsageHostHour object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageHostHour() *UsageHostHour {
	this := UsageHostHour{}
	return &this
}

// NewUsageHostHourWithDefaults instantiates a new UsageHostHour object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageHostHourWithDefaults() *UsageHostHour {
	this := UsageHostHour{}
	return &this
}

// GetAgentHostCount returns the AgentHostCount field value if set, zero value otherwise.
func (o *UsageHostHour) GetAgentHostCount() int64 {
	if o == nil || o.AgentHostCount == nil {
		var ret int64
		return ret
	}
	return *o.AgentHostCount
}

// GetAgentHostCountOk returns a tuple with the AgentHostCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageHostHour) GetAgentHostCountOk() (*int64, bool) {
	if o == nil || o.AgentHostCount == nil {
		return nil, false
	}
	return o.AgentHostCount, true
}

// HasAgentHostCount returns a boolean if a field has been set.
func (o *UsageHostHour) HasAgentHostCount() bool {
	if o != nil && o.AgentHostCount != nil {
		return true
	}

	return false
}

// SetAgentHostCount gets a reference to the given int64 and assigns it to the AgentHostCount field.
func (o *UsageHostHour) SetAgentHostCount(v int64) {
	o.AgentHostCount = &v
}

// GetAlibabaHostCount returns the AlibabaHostCount field value if set, zero value otherwise.
func (o *UsageHostHour) GetAlibabaHostCount() int64 {
	if o == nil || o.AlibabaHostCount == nil {
		var ret int64
		return ret
	}
	return *o.AlibabaHostCount
}

// GetAlibabaHostCountOk returns a tuple with the AlibabaHostCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageHostHour) GetAlibabaHostCountOk() (*int64, bool) {
	if o == nil || o.AlibabaHostCount == nil {
		return nil, false
	}
	return o.AlibabaHostCount, true
}

// HasAlibabaHostCount returns a boolean if a field has been set.
func (o *UsageHostHour) HasAlibabaHostCount() bool {
	if o != nil && o.AlibabaHostCount != nil {
		return true
	}

	return false
}

// SetAlibabaHostCount gets a reference to the given int64 and assigns it to the AlibabaHostCount field.
func (o *UsageHostHour) SetAlibabaHostCount(v int64) {
	o.AlibabaHostCount = &v
}

// GetApmAzureAppServiceHostCount returns the ApmAzureAppServiceHostCount field value if set, zero value otherwise.
func (o *UsageHostHour) GetApmAzureAppServiceHostCount() int64 {
	if o == nil || o.ApmAzureAppServiceHostCount == nil {
		var ret int64
		return ret
	}
	return *o.ApmAzureAppServiceHostCount
}

// GetApmAzureAppServiceHostCountOk returns a tuple with the ApmAzureAppServiceHostCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageHostHour) GetApmAzureAppServiceHostCountOk() (*int64, bool) {
	if o == nil || o.ApmAzureAppServiceHostCount == nil {
		return nil, false
	}
	return o.ApmAzureAppServiceHostCount, true
}

// HasApmAzureAppServiceHostCount returns a boolean if a field has been set.
func (o *UsageHostHour) HasApmAzureAppServiceHostCount() bool {
	if o != nil && o.ApmAzureAppServiceHostCount != nil {
		return true
	}

	return false
}

// SetApmAzureAppServiceHostCount gets a reference to the given int64 and assigns it to the ApmAzureAppServiceHostCount field.
func (o *UsageHostHour) SetApmAzureAppServiceHostCount(v int64) {
	o.ApmAzureAppServiceHostCount = &v
}

// GetApmHostCount returns the ApmHostCount field value if set, zero value otherwise.
func (o *UsageHostHour) GetApmHostCount() int64 {
	if o == nil || o.ApmHostCount == nil {
		var ret int64
		return ret
	}
	return *o.ApmHostCount
}

// GetApmHostCountOk returns a tuple with the ApmHostCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageHostHour) GetApmHostCountOk() (*int64, bool) {
	if o == nil || o.ApmHostCount == nil {
		return nil, false
	}
	return o.ApmHostCount, true
}

// HasApmHostCount returns a boolean if a field has been set.
func (o *UsageHostHour) HasApmHostCount() bool {
	if o != nil && o.ApmHostCount != nil {
		return true
	}

	return false
}

// SetApmHostCount gets a reference to the given int64 and assigns it to the ApmHostCount field.
func (o *UsageHostHour) SetApmHostCount(v int64) {
	o.ApmHostCount = &v
}

// GetAwsHostCount returns the AwsHostCount field value if set, zero value otherwise.
func (o *UsageHostHour) GetAwsHostCount() int64 {
	if o == nil || o.AwsHostCount == nil {
		var ret int64
		return ret
	}
	return *o.AwsHostCount
}

// GetAwsHostCountOk returns a tuple with the AwsHostCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageHostHour) GetAwsHostCountOk() (*int64, bool) {
	if o == nil || o.AwsHostCount == nil {
		return nil, false
	}
	return o.AwsHostCount, true
}

// HasAwsHostCount returns a boolean if a field has been set.
func (o *UsageHostHour) HasAwsHostCount() bool {
	if o != nil && o.AwsHostCount != nil {
		return true
	}

	return false
}

// SetAwsHostCount gets a reference to the given int64 and assigns it to the AwsHostCount field.
func (o *UsageHostHour) SetAwsHostCount(v int64) {
	o.AwsHostCount = &v
}

// GetAzureHostCount returns the AzureHostCount field value if set, zero value otherwise.
func (o *UsageHostHour) GetAzureHostCount() int64 {
	if o == nil || o.AzureHostCount == nil {
		var ret int64
		return ret
	}
	return *o.AzureHostCount
}

// GetAzureHostCountOk returns a tuple with the AzureHostCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageHostHour) GetAzureHostCountOk() (*int64, bool) {
	if o == nil || o.AzureHostCount == nil {
		return nil, false
	}
	return o.AzureHostCount, true
}

// HasAzureHostCount returns a boolean if a field has been set.
func (o *UsageHostHour) HasAzureHostCount() bool {
	if o != nil && o.AzureHostCount != nil {
		return true
	}

	return false
}

// SetAzureHostCount gets a reference to the given int64 and assigns it to the AzureHostCount field.
func (o *UsageHostHour) SetAzureHostCount(v int64) {
	o.AzureHostCount = &v
}

// GetContainerCount returns the ContainerCount field value if set, zero value otherwise.
func (o *UsageHostHour) GetContainerCount() int64 {
	if o == nil || o.ContainerCount == nil {
		var ret int64
		return ret
	}
	return *o.ContainerCount
}

// GetContainerCountOk returns a tuple with the ContainerCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageHostHour) GetContainerCountOk() (*int64, bool) {
	if o == nil || o.ContainerCount == nil {
		return nil, false
	}
	return o.ContainerCount, true
}

// HasContainerCount returns a boolean if a field has been set.
func (o *UsageHostHour) HasContainerCount() bool {
	if o != nil && o.ContainerCount != nil {
		return true
	}

	return false
}

// SetContainerCount gets a reference to the given int64 and assigns it to the ContainerCount field.
func (o *UsageHostHour) SetContainerCount(v int64) {
	o.ContainerCount = &v
}

// GetGcpHostCount returns the GcpHostCount field value if set, zero value otherwise.
func (o *UsageHostHour) GetGcpHostCount() int64 {
	if o == nil || o.GcpHostCount == nil {
		var ret int64
		return ret
	}
	return *o.GcpHostCount
}

// GetGcpHostCountOk returns a tuple with the GcpHostCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageHostHour) GetGcpHostCountOk() (*int64, bool) {
	if o == nil || o.GcpHostCount == nil {
		return nil, false
	}
	return o.GcpHostCount, true
}

// HasGcpHostCount returns a boolean if a field has been set.
func (o *UsageHostHour) HasGcpHostCount() bool {
	if o != nil && o.GcpHostCount != nil {
		return true
	}

	return false
}

// SetGcpHostCount gets a reference to the given int64 and assigns it to the GcpHostCount field.
func (o *UsageHostHour) SetGcpHostCount(v int64) {
	o.GcpHostCount = &v
}

// GetHerokuHostCount returns the HerokuHostCount field value if set, zero value otherwise.
func (o *UsageHostHour) GetHerokuHostCount() int64 {
	if o == nil || o.HerokuHostCount == nil {
		var ret int64
		return ret
	}
	return *o.HerokuHostCount
}

// GetHerokuHostCountOk returns a tuple with the HerokuHostCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageHostHour) GetHerokuHostCountOk() (*int64, bool) {
	if o == nil || o.HerokuHostCount == nil {
		return nil, false
	}
	return o.HerokuHostCount, true
}

// HasHerokuHostCount returns a boolean if a field has been set.
func (o *UsageHostHour) HasHerokuHostCount() bool {
	if o != nil && o.HerokuHostCount != nil {
		return true
	}

	return false
}

// SetHerokuHostCount gets a reference to the given int64 and assigns it to the HerokuHostCount field.
func (o *UsageHostHour) SetHerokuHostCount(v int64) {
	o.HerokuHostCount = &v
}

// GetHostCount returns the HostCount field value if set, zero value otherwise.
func (o *UsageHostHour) GetHostCount() int64 {
	if o == nil || o.HostCount == nil {
		var ret int64
		return ret
	}
	return *o.HostCount
}

// GetHostCountOk returns a tuple with the HostCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageHostHour) GetHostCountOk() (*int64, bool) {
	if o == nil || o.HostCount == nil {
		return nil, false
	}
	return o.HostCount, true
}

// HasHostCount returns a boolean if a field has been set.
func (o *UsageHostHour) HasHostCount() bool {
	if o != nil && o.HostCount != nil {
		return true
	}

	return false
}

// SetHostCount gets a reference to the given int64 and assigns it to the HostCount field.
func (o *UsageHostHour) SetHostCount(v int64) {
	o.HostCount = &v
}

// GetHour returns the Hour field value if set, zero value otherwise.
func (o *UsageHostHour) GetHour() time.Time {
	if o == nil || o.Hour == nil {
		var ret time.Time
		return ret
	}
	return *o.Hour
}

// GetHourOk returns a tuple with the Hour field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageHostHour) GetHourOk() (*time.Time, bool) {
	if o == nil || o.Hour == nil {
		return nil, false
	}
	return o.Hour, true
}

// HasHour returns a boolean if a field has been set.
func (o *UsageHostHour) HasHour() bool {
	if o != nil && o.Hour != nil {
		return true
	}

	return false
}

// SetHour gets a reference to the given time.Time and assigns it to the Hour field.
func (o *UsageHostHour) SetHour(v time.Time) {
	o.Hour = &v
}

// GetInfraAzureAppService returns the InfraAzureAppService field value if set, zero value otherwise.
func (o *UsageHostHour) GetInfraAzureAppService() int64 {
	if o == nil || o.InfraAzureAppService == nil {
		var ret int64
		return ret
	}
	return *o.InfraAzureAppService
}

// GetInfraAzureAppServiceOk returns a tuple with the InfraAzureAppService field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageHostHour) GetInfraAzureAppServiceOk() (*int64, bool) {
	if o == nil || o.InfraAzureAppService == nil {
		return nil, false
	}
	return o.InfraAzureAppService, true
}

// HasInfraAzureAppService returns a boolean if a field has been set.
func (o *UsageHostHour) HasInfraAzureAppService() bool {
	if o != nil && o.InfraAzureAppService != nil {
		return true
	}

	return false
}

// SetInfraAzureAppService gets a reference to the given int64 and assigns it to the InfraAzureAppService field.
func (o *UsageHostHour) SetInfraAzureAppService(v int64) {
	o.InfraAzureAppService = &v
}

// GetOpentelemetryHostCount returns the OpentelemetryHostCount field value if set, zero value otherwise.
func (o *UsageHostHour) GetOpentelemetryHostCount() int64 {
	if o == nil || o.OpentelemetryHostCount == nil {
		var ret int64
		return ret
	}
	return *o.OpentelemetryHostCount
}

// GetOpentelemetryHostCountOk returns a tuple with the OpentelemetryHostCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageHostHour) GetOpentelemetryHostCountOk() (*int64, bool) {
	if o == nil || o.OpentelemetryHostCount == nil {
		return nil, false
	}
	return o.OpentelemetryHostCount, true
}

// HasOpentelemetryHostCount returns a boolean if a field has been set.
func (o *UsageHostHour) HasOpentelemetryHostCount() bool {
	if o != nil && o.OpentelemetryHostCount != nil {
		return true
	}

	return false
}

// SetOpentelemetryHostCount gets a reference to the given int64 and assigns it to the OpentelemetryHostCount field.
func (o *UsageHostHour) SetOpentelemetryHostCount(v int64) {
	o.OpentelemetryHostCount = &v
}

// GetOrgName returns the OrgName field value if set, zero value otherwise.
func (o *UsageHostHour) GetOrgName() string {
	if o == nil || o.OrgName == nil {
		var ret string
		return ret
	}
	return *o.OrgName
}

// GetOrgNameOk returns a tuple with the OrgName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageHostHour) GetOrgNameOk() (*string, bool) {
	if o == nil || o.OrgName == nil {
		return nil, false
	}
	return o.OrgName, true
}

// HasOrgName returns a boolean if a field has been set.
func (o *UsageHostHour) HasOrgName() bool {
	if o != nil && o.OrgName != nil {
		return true
	}

	return false
}

// SetOrgName gets a reference to the given string and assigns it to the OrgName field.
func (o *UsageHostHour) SetOrgName(v string) {
	o.OrgName = &v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *UsageHostHour) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageHostHour) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *UsageHostHour) HasPublicId() bool {
	if o != nil && o.PublicId != nil {
		return true
	}

	return false
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *UsageHostHour) SetPublicId(v string) {
	o.PublicId = &v
}

// GetVsphereHostCount returns the VsphereHostCount field value if set, zero value otherwise.
func (o *UsageHostHour) GetVsphereHostCount() int64 {
	if o == nil || o.VsphereHostCount == nil {
		var ret int64
		return ret
	}
	return *o.VsphereHostCount
}

// GetVsphereHostCountOk returns a tuple with the VsphereHostCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageHostHour) GetVsphereHostCountOk() (*int64, bool) {
	if o == nil || o.VsphereHostCount == nil {
		return nil, false
	}
	return o.VsphereHostCount, true
}

// HasVsphereHostCount returns a boolean if a field has been set.
func (o *UsageHostHour) HasVsphereHostCount() bool {
	if o != nil && o.VsphereHostCount != nil {
		return true
	}

	return false
}

// SetVsphereHostCount gets a reference to the given int64 and assigns it to the VsphereHostCount field.
func (o *UsageHostHour) SetVsphereHostCount(v int64) {
	o.VsphereHostCount = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageHostHour) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AgentHostCount != nil {
		toSerialize["agent_host_count"] = o.AgentHostCount
	}
	if o.AlibabaHostCount != nil {
		toSerialize["alibaba_host_count"] = o.AlibabaHostCount
	}
	if o.ApmAzureAppServiceHostCount != nil {
		toSerialize["apm_azure_app_service_host_count"] = o.ApmAzureAppServiceHostCount
	}
	if o.ApmHostCount != nil {
		toSerialize["apm_host_count"] = o.ApmHostCount
	}
	if o.AwsHostCount != nil {
		toSerialize["aws_host_count"] = o.AwsHostCount
	}
	if o.AzureHostCount != nil {
		toSerialize["azure_host_count"] = o.AzureHostCount
	}
	if o.ContainerCount != nil {
		toSerialize["container_count"] = o.ContainerCount
	}
	if o.GcpHostCount != nil {
		toSerialize["gcp_host_count"] = o.GcpHostCount
	}
	if o.HerokuHostCount != nil {
		toSerialize["heroku_host_count"] = o.HerokuHostCount
	}
	if o.HostCount != nil {
		toSerialize["host_count"] = o.HostCount
	}
	if o.Hour != nil {
		if o.Hour.Nanosecond() == 0 {
			toSerialize["hour"] = o.Hour.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["hour"] = o.Hour.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.InfraAzureAppService != nil {
		toSerialize["infra_azure_app_service"] = o.InfraAzureAppService
	}
	if o.OpentelemetryHostCount != nil {
		toSerialize["opentelemetry_host_count"] = o.OpentelemetryHostCount
	}
	if o.OrgName != nil {
		toSerialize["org_name"] = o.OrgName
	}
	if o.PublicId != nil {
		toSerialize["public_id"] = o.PublicId
	}
	if o.VsphereHostCount != nil {
		toSerialize["vsphere_host_count"] = o.VsphereHostCount
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UsageHostHour) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		AgentHostCount              *int64     `json:"agent_host_count,omitempty"`
		AlibabaHostCount            *int64     `json:"alibaba_host_count,omitempty"`
		ApmAzureAppServiceHostCount *int64     `json:"apm_azure_app_service_host_count,omitempty"`
		ApmHostCount                *int64     `json:"apm_host_count,omitempty"`
		AwsHostCount                *int64     `json:"aws_host_count,omitempty"`
		AzureHostCount              *int64     `json:"azure_host_count,omitempty"`
		ContainerCount              *int64     `json:"container_count,omitempty"`
		GcpHostCount                *int64     `json:"gcp_host_count,omitempty"`
		HerokuHostCount             *int64     `json:"heroku_host_count,omitempty"`
		HostCount                   *int64     `json:"host_count,omitempty"`
		Hour                        *time.Time `json:"hour,omitempty"`
		InfraAzureAppService        *int64     `json:"infra_azure_app_service,omitempty"`
		OpentelemetryHostCount      *int64     `json:"opentelemetry_host_count,omitempty"`
		OrgName                     *string    `json:"org_name,omitempty"`
		PublicId                    *string    `json:"public_id,omitempty"`
		VsphereHostCount            *int64     `json:"vsphere_host_count,omitempty"`
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
	o.AgentHostCount = all.AgentHostCount
	o.AlibabaHostCount = all.AlibabaHostCount
	o.ApmAzureAppServiceHostCount = all.ApmAzureAppServiceHostCount
	o.ApmHostCount = all.ApmHostCount
	o.AwsHostCount = all.AwsHostCount
	o.AzureHostCount = all.AzureHostCount
	o.ContainerCount = all.ContainerCount
	o.GcpHostCount = all.GcpHostCount
	o.HerokuHostCount = all.HerokuHostCount
	o.HostCount = all.HostCount
	o.Hour = all.Hour
	o.InfraAzureAppService = all.InfraAzureAppService
	o.OpentelemetryHostCount = all.OpentelemetryHostCount
	o.OrgName = all.OrgName
	o.PublicId = all.PublicId
	o.VsphereHostCount = all.VsphereHostCount
	return nil
}
