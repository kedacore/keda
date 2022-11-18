// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// UsageAttributionValues Fields in Usage Summary by tag(s).
type UsageAttributionValues struct {
	// The percentage of synthetic API test usage by tag(s).
	ApiPercentage *float64 `json:"api_percentage,omitempty"`
	// The synthetic API test usage by tag(s).
	ApiUsage *float64 `json:"api_usage,omitempty"`
	// The percentage of APM host usage by tag(s).
	ApmHostPercentage *float64 `json:"apm_host_percentage,omitempty"`
	// The APM host usage by tag(s).
	ApmHostUsage *float64 `json:"apm_host_usage,omitempty"`
	// The percentage of Application Security Monitoring host usage by tag(s).
	AppsecPercentage *float64 `json:"appsec_percentage,omitempty"`
	// The Application Security Monitoring host usage by tag(s).
	AppsecUsage *float64 `json:"appsec_usage,omitempty"`
	// The percentage of synthetic browser test usage by tag(s).
	BrowserPercentage *float64 `json:"browser_percentage,omitempty"`
	// The synthetic browser test usage by tag(s).
	BrowserUsage *float64 `json:"browser_usage,omitempty"`
	// The percentage of container usage by tag(s).
	ContainerPercentage *float64 `json:"container_percentage,omitempty"`
	// The container usage by tag(s).
	ContainerUsage *float64 `json:"container_usage,omitempty"`
	// The percentage of Cloud Security Posture Management container usage by tag(s)
	CspmContainerPercentage *float64 `json:"cspm_container_percentage,omitempty"`
	// The Cloud Security Posture Management container usage by tag(s)
	CspmContainerUsage *float64 `json:"cspm_container_usage,omitempty"`
	// The percentage of Cloud Security Posture Management host usage by tag(s)
	CspmHostPercentage *float64 `json:"cspm_host_percentage,omitempty"`
	// The Cloud Security Posture Management host usage by tag(s)
	CspmHostUsage *float64 `json:"cspm_host_usage,omitempty"`
	// The percentage of custom metrics usage by tag(s).
	CustomTimeseriesPercentage *float64 `json:"custom_timeseries_percentage,omitempty"`
	// The custom metrics usage by tag(s).
	CustomTimeseriesUsage *float64 `json:"custom_timeseries_usage,omitempty"`
	// The percentage of Cloud Workload Security container usage by tag(s)
	CwsContainerPercentage *float64 `json:"cws_container_percentage,omitempty"`
	// The Cloud Workload Security container usage by tag(s)
	CwsContainerUsage *float64 `json:"cws_container_usage,omitempty"`
	// The percentage of Cloud Workload Security host usage by tag(s)
	CwsHostPercentage *float64 `json:"cws_host_percentage,omitempty"`
	// The Cloud Workload Security host usage by tag(s)
	CwsHostUsage *float64 `json:"cws_host_usage,omitempty"`
	// The percentage of Database Monitoring host usage by tag(s).
	DbmHostsPercentage *float64 `json:"dbm_hosts_percentage,omitempty"`
	// The Database Monitoring host usage by tag(s).
	DbmHostsUsage *float64 `json:"dbm_hosts_usage,omitempty"`
	// The percentage of Database Monitoring normalized queries usage by tag(s).
	DbmQueriesPercentage *float64 `json:"dbm_queries_percentage,omitempty"`
	// The Database Monitoring normalized queries usage by tag(s).
	DbmQueriesUsage *float64 `json:"dbm_queries_usage,omitempty"`
	// The percentage of estimated live indexed logs usage by tag(s). Note this field is in private beta.
	EstimatedIndexedLogsPercentage *float64 `json:"estimated_indexed_logs_percentage,omitempty"`
	// The estimated live indexed logs usage by tag(s). Note this field is in private beta.
	EstimatedIndexedLogsUsage *float64 `json:"estimated_indexed_logs_usage,omitempty"`
	// The percentage of estimated indexed spans usage by tag(s). Note this field is in private beta.
	EstimatedIndexedSpansPercentage *float64 `json:"estimated_indexed_spans_percentage,omitempty"`
	// The estimated indexed spans usage by tag(s). Note this field is in private beta.
	EstimatedIndexedSpansUsage *float64 `json:"estimated_indexed_spans_usage,omitempty"`
	// The percentage of estimated ingested spans usage by tag(s). Note this field is in private beta.
	EstimatedIngestedSpansPercentage *float64 `json:"estimated_ingested_spans_percentage,omitempty"`
	// The estimated ingested spans usage by tag(s). Note this field is in private beta.
	EstimatedIngestedSpansUsage *float64 `json:"estimated_ingested_spans_usage,omitempty"`
	// The percentage of infrastructure host usage by tag(s).
	InfraHostPercentage *float64 `json:"infra_host_percentage,omitempty"`
	// The infrastructure host usage by tag(s).
	InfraHostUsage *float64 `json:"infra_host_usage,omitempty"`
	// The percentage of Lambda function usage by tag(s).
	LambdaFunctionsPercentage *float64 `json:"lambda_functions_percentage,omitempty"`
	// The Lambda function usage by tag(s).
	LambdaFunctionsUsage *float64 `json:"lambda_functions_usage,omitempty"`
	// The percentage of Lambda invocation usage by tag(s).
	LambdaInvocationsPercentage *float64 `json:"lambda_invocations_percentage,omitempty"`
	// The Lambda invocation usage by tag(s).
	LambdaInvocationsUsage *float64 `json:"lambda_invocations_usage,omitempty"`
	// The percentage of network host usage by tag(s).
	NpmHostPercentage *float64 `json:"npm_host_percentage,omitempty"`
	// The network host usage by tag(s).
	NpmHostUsage *float64 `json:"npm_host_usage,omitempty"`
	// The percentage of profiled containers usage by tag(s).
	ProfiledContainerPercentage *float64 `json:"profiled_container_percentage,omitempty"`
	// The profiled container usage by tag(s).
	ProfiledContainerUsage *float64 `json:"profiled_container_usage,omitempty"`
	// The percentage of profiled hosts usage by tag(s).
	ProfiledHostsPercentage *float64 `json:"profiled_hosts_percentage,omitempty"`
	// The profiled host usage by tag(s).
	ProfiledHostsUsage *float64 `json:"profiled_hosts_usage,omitempty"`
	// The percentage of network device usage by tag(s).
	SnmpPercentage *float64 `json:"snmp_percentage,omitempty"`
	// The network device usage by tag(s).
	SnmpUsage *float64 `json:"snmp_usage,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageAttributionValues instantiates a new UsageAttributionValues object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageAttributionValues() *UsageAttributionValues {
	this := UsageAttributionValues{}
	return &this
}

// NewUsageAttributionValuesWithDefaults instantiates a new UsageAttributionValues object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageAttributionValuesWithDefaults() *UsageAttributionValues {
	this := UsageAttributionValues{}
	return &this
}

// GetApiPercentage returns the ApiPercentage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetApiPercentage() float64 {
	if o == nil || o.ApiPercentage == nil {
		var ret float64
		return ret
	}
	return *o.ApiPercentage
}

// GetApiPercentageOk returns a tuple with the ApiPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetApiPercentageOk() (*float64, bool) {
	if o == nil || o.ApiPercentage == nil {
		return nil, false
	}
	return o.ApiPercentage, true
}

// HasApiPercentage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasApiPercentage() bool {
	if o != nil && o.ApiPercentage != nil {
		return true
	}

	return false
}

// SetApiPercentage gets a reference to the given float64 and assigns it to the ApiPercentage field.
func (o *UsageAttributionValues) SetApiPercentage(v float64) {
	o.ApiPercentage = &v
}

// GetApiUsage returns the ApiUsage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetApiUsage() float64 {
	if o == nil || o.ApiUsage == nil {
		var ret float64
		return ret
	}
	return *o.ApiUsage
}

// GetApiUsageOk returns a tuple with the ApiUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetApiUsageOk() (*float64, bool) {
	if o == nil || o.ApiUsage == nil {
		return nil, false
	}
	return o.ApiUsage, true
}

// HasApiUsage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasApiUsage() bool {
	if o != nil && o.ApiUsage != nil {
		return true
	}

	return false
}

// SetApiUsage gets a reference to the given float64 and assigns it to the ApiUsage field.
func (o *UsageAttributionValues) SetApiUsage(v float64) {
	o.ApiUsage = &v
}

// GetApmHostPercentage returns the ApmHostPercentage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetApmHostPercentage() float64 {
	if o == nil || o.ApmHostPercentage == nil {
		var ret float64
		return ret
	}
	return *o.ApmHostPercentage
}

// GetApmHostPercentageOk returns a tuple with the ApmHostPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetApmHostPercentageOk() (*float64, bool) {
	if o == nil || o.ApmHostPercentage == nil {
		return nil, false
	}
	return o.ApmHostPercentage, true
}

// HasApmHostPercentage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasApmHostPercentage() bool {
	if o != nil && o.ApmHostPercentage != nil {
		return true
	}

	return false
}

// SetApmHostPercentage gets a reference to the given float64 and assigns it to the ApmHostPercentage field.
func (o *UsageAttributionValues) SetApmHostPercentage(v float64) {
	o.ApmHostPercentage = &v
}

// GetApmHostUsage returns the ApmHostUsage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetApmHostUsage() float64 {
	if o == nil || o.ApmHostUsage == nil {
		var ret float64
		return ret
	}
	return *o.ApmHostUsage
}

// GetApmHostUsageOk returns a tuple with the ApmHostUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetApmHostUsageOk() (*float64, bool) {
	if o == nil || o.ApmHostUsage == nil {
		return nil, false
	}
	return o.ApmHostUsage, true
}

// HasApmHostUsage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasApmHostUsage() bool {
	if o != nil && o.ApmHostUsage != nil {
		return true
	}

	return false
}

// SetApmHostUsage gets a reference to the given float64 and assigns it to the ApmHostUsage field.
func (o *UsageAttributionValues) SetApmHostUsage(v float64) {
	o.ApmHostUsage = &v
}

// GetAppsecPercentage returns the AppsecPercentage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetAppsecPercentage() float64 {
	if o == nil || o.AppsecPercentage == nil {
		var ret float64
		return ret
	}
	return *o.AppsecPercentage
}

// GetAppsecPercentageOk returns a tuple with the AppsecPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetAppsecPercentageOk() (*float64, bool) {
	if o == nil || o.AppsecPercentage == nil {
		return nil, false
	}
	return o.AppsecPercentage, true
}

// HasAppsecPercentage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasAppsecPercentage() bool {
	if o != nil && o.AppsecPercentage != nil {
		return true
	}

	return false
}

// SetAppsecPercentage gets a reference to the given float64 and assigns it to the AppsecPercentage field.
func (o *UsageAttributionValues) SetAppsecPercentage(v float64) {
	o.AppsecPercentage = &v
}

// GetAppsecUsage returns the AppsecUsage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetAppsecUsage() float64 {
	if o == nil || o.AppsecUsage == nil {
		var ret float64
		return ret
	}
	return *o.AppsecUsage
}

// GetAppsecUsageOk returns a tuple with the AppsecUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetAppsecUsageOk() (*float64, bool) {
	if o == nil || o.AppsecUsage == nil {
		return nil, false
	}
	return o.AppsecUsage, true
}

// HasAppsecUsage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasAppsecUsage() bool {
	if o != nil && o.AppsecUsage != nil {
		return true
	}

	return false
}

// SetAppsecUsage gets a reference to the given float64 and assigns it to the AppsecUsage field.
func (o *UsageAttributionValues) SetAppsecUsage(v float64) {
	o.AppsecUsage = &v
}

// GetBrowserPercentage returns the BrowserPercentage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetBrowserPercentage() float64 {
	if o == nil || o.BrowserPercentage == nil {
		var ret float64
		return ret
	}
	return *o.BrowserPercentage
}

// GetBrowserPercentageOk returns a tuple with the BrowserPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetBrowserPercentageOk() (*float64, bool) {
	if o == nil || o.BrowserPercentage == nil {
		return nil, false
	}
	return o.BrowserPercentage, true
}

// HasBrowserPercentage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasBrowserPercentage() bool {
	if o != nil && o.BrowserPercentage != nil {
		return true
	}

	return false
}

// SetBrowserPercentage gets a reference to the given float64 and assigns it to the BrowserPercentage field.
func (o *UsageAttributionValues) SetBrowserPercentage(v float64) {
	o.BrowserPercentage = &v
}

// GetBrowserUsage returns the BrowserUsage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetBrowserUsage() float64 {
	if o == nil || o.BrowserUsage == nil {
		var ret float64
		return ret
	}
	return *o.BrowserUsage
}

// GetBrowserUsageOk returns a tuple with the BrowserUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetBrowserUsageOk() (*float64, bool) {
	if o == nil || o.BrowserUsage == nil {
		return nil, false
	}
	return o.BrowserUsage, true
}

// HasBrowserUsage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasBrowserUsage() bool {
	if o != nil && o.BrowserUsage != nil {
		return true
	}

	return false
}

// SetBrowserUsage gets a reference to the given float64 and assigns it to the BrowserUsage field.
func (o *UsageAttributionValues) SetBrowserUsage(v float64) {
	o.BrowserUsage = &v
}

// GetContainerPercentage returns the ContainerPercentage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetContainerPercentage() float64 {
	if o == nil || o.ContainerPercentage == nil {
		var ret float64
		return ret
	}
	return *o.ContainerPercentage
}

// GetContainerPercentageOk returns a tuple with the ContainerPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetContainerPercentageOk() (*float64, bool) {
	if o == nil || o.ContainerPercentage == nil {
		return nil, false
	}
	return o.ContainerPercentage, true
}

// HasContainerPercentage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasContainerPercentage() bool {
	if o != nil && o.ContainerPercentage != nil {
		return true
	}

	return false
}

// SetContainerPercentage gets a reference to the given float64 and assigns it to the ContainerPercentage field.
func (o *UsageAttributionValues) SetContainerPercentage(v float64) {
	o.ContainerPercentage = &v
}

// GetContainerUsage returns the ContainerUsage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetContainerUsage() float64 {
	if o == nil || o.ContainerUsage == nil {
		var ret float64
		return ret
	}
	return *o.ContainerUsage
}

// GetContainerUsageOk returns a tuple with the ContainerUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetContainerUsageOk() (*float64, bool) {
	if o == nil || o.ContainerUsage == nil {
		return nil, false
	}
	return o.ContainerUsage, true
}

// HasContainerUsage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasContainerUsage() bool {
	if o != nil && o.ContainerUsage != nil {
		return true
	}

	return false
}

// SetContainerUsage gets a reference to the given float64 and assigns it to the ContainerUsage field.
func (o *UsageAttributionValues) SetContainerUsage(v float64) {
	o.ContainerUsage = &v
}

// GetCspmContainerPercentage returns the CspmContainerPercentage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetCspmContainerPercentage() float64 {
	if o == nil || o.CspmContainerPercentage == nil {
		var ret float64
		return ret
	}
	return *o.CspmContainerPercentage
}

// GetCspmContainerPercentageOk returns a tuple with the CspmContainerPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetCspmContainerPercentageOk() (*float64, bool) {
	if o == nil || o.CspmContainerPercentage == nil {
		return nil, false
	}
	return o.CspmContainerPercentage, true
}

// HasCspmContainerPercentage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasCspmContainerPercentage() bool {
	if o != nil && o.CspmContainerPercentage != nil {
		return true
	}

	return false
}

// SetCspmContainerPercentage gets a reference to the given float64 and assigns it to the CspmContainerPercentage field.
func (o *UsageAttributionValues) SetCspmContainerPercentage(v float64) {
	o.CspmContainerPercentage = &v
}

// GetCspmContainerUsage returns the CspmContainerUsage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetCspmContainerUsage() float64 {
	if o == nil || o.CspmContainerUsage == nil {
		var ret float64
		return ret
	}
	return *o.CspmContainerUsage
}

// GetCspmContainerUsageOk returns a tuple with the CspmContainerUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetCspmContainerUsageOk() (*float64, bool) {
	if o == nil || o.CspmContainerUsage == nil {
		return nil, false
	}
	return o.CspmContainerUsage, true
}

// HasCspmContainerUsage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasCspmContainerUsage() bool {
	if o != nil && o.CspmContainerUsage != nil {
		return true
	}

	return false
}

// SetCspmContainerUsage gets a reference to the given float64 and assigns it to the CspmContainerUsage field.
func (o *UsageAttributionValues) SetCspmContainerUsage(v float64) {
	o.CspmContainerUsage = &v
}

// GetCspmHostPercentage returns the CspmHostPercentage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetCspmHostPercentage() float64 {
	if o == nil || o.CspmHostPercentage == nil {
		var ret float64
		return ret
	}
	return *o.CspmHostPercentage
}

// GetCspmHostPercentageOk returns a tuple with the CspmHostPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetCspmHostPercentageOk() (*float64, bool) {
	if o == nil || o.CspmHostPercentage == nil {
		return nil, false
	}
	return o.CspmHostPercentage, true
}

// HasCspmHostPercentage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasCspmHostPercentage() bool {
	if o != nil && o.CspmHostPercentage != nil {
		return true
	}

	return false
}

// SetCspmHostPercentage gets a reference to the given float64 and assigns it to the CspmHostPercentage field.
func (o *UsageAttributionValues) SetCspmHostPercentage(v float64) {
	o.CspmHostPercentage = &v
}

// GetCspmHostUsage returns the CspmHostUsage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetCspmHostUsage() float64 {
	if o == nil || o.CspmHostUsage == nil {
		var ret float64
		return ret
	}
	return *o.CspmHostUsage
}

// GetCspmHostUsageOk returns a tuple with the CspmHostUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetCspmHostUsageOk() (*float64, bool) {
	if o == nil || o.CspmHostUsage == nil {
		return nil, false
	}
	return o.CspmHostUsage, true
}

// HasCspmHostUsage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasCspmHostUsage() bool {
	if o != nil && o.CspmHostUsage != nil {
		return true
	}

	return false
}

// SetCspmHostUsage gets a reference to the given float64 and assigns it to the CspmHostUsage field.
func (o *UsageAttributionValues) SetCspmHostUsage(v float64) {
	o.CspmHostUsage = &v
}

// GetCustomTimeseriesPercentage returns the CustomTimeseriesPercentage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetCustomTimeseriesPercentage() float64 {
	if o == nil || o.CustomTimeseriesPercentage == nil {
		var ret float64
		return ret
	}
	return *o.CustomTimeseriesPercentage
}

// GetCustomTimeseriesPercentageOk returns a tuple with the CustomTimeseriesPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetCustomTimeseriesPercentageOk() (*float64, bool) {
	if o == nil || o.CustomTimeseriesPercentage == nil {
		return nil, false
	}
	return o.CustomTimeseriesPercentage, true
}

// HasCustomTimeseriesPercentage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasCustomTimeseriesPercentage() bool {
	if o != nil && o.CustomTimeseriesPercentage != nil {
		return true
	}

	return false
}

// SetCustomTimeseriesPercentage gets a reference to the given float64 and assigns it to the CustomTimeseriesPercentage field.
func (o *UsageAttributionValues) SetCustomTimeseriesPercentage(v float64) {
	o.CustomTimeseriesPercentage = &v
}

// GetCustomTimeseriesUsage returns the CustomTimeseriesUsage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetCustomTimeseriesUsage() float64 {
	if o == nil || o.CustomTimeseriesUsage == nil {
		var ret float64
		return ret
	}
	return *o.CustomTimeseriesUsage
}

// GetCustomTimeseriesUsageOk returns a tuple with the CustomTimeseriesUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetCustomTimeseriesUsageOk() (*float64, bool) {
	if o == nil || o.CustomTimeseriesUsage == nil {
		return nil, false
	}
	return o.CustomTimeseriesUsage, true
}

// HasCustomTimeseriesUsage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasCustomTimeseriesUsage() bool {
	if o != nil && o.CustomTimeseriesUsage != nil {
		return true
	}

	return false
}

// SetCustomTimeseriesUsage gets a reference to the given float64 and assigns it to the CustomTimeseriesUsage field.
func (o *UsageAttributionValues) SetCustomTimeseriesUsage(v float64) {
	o.CustomTimeseriesUsage = &v
}

// GetCwsContainerPercentage returns the CwsContainerPercentage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetCwsContainerPercentage() float64 {
	if o == nil || o.CwsContainerPercentage == nil {
		var ret float64
		return ret
	}
	return *o.CwsContainerPercentage
}

// GetCwsContainerPercentageOk returns a tuple with the CwsContainerPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetCwsContainerPercentageOk() (*float64, bool) {
	if o == nil || o.CwsContainerPercentage == nil {
		return nil, false
	}
	return o.CwsContainerPercentage, true
}

// HasCwsContainerPercentage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasCwsContainerPercentage() bool {
	if o != nil && o.CwsContainerPercentage != nil {
		return true
	}

	return false
}

// SetCwsContainerPercentage gets a reference to the given float64 and assigns it to the CwsContainerPercentage field.
func (o *UsageAttributionValues) SetCwsContainerPercentage(v float64) {
	o.CwsContainerPercentage = &v
}

// GetCwsContainerUsage returns the CwsContainerUsage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetCwsContainerUsage() float64 {
	if o == nil || o.CwsContainerUsage == nil {
		var ret float64
		return ret
	}
	return *o.CwsContainerUsage
}

// GetCwsContainerUsageOk returns a tuple with the CwsContainerUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetCwsContainerUsageOk() (*float64, bool) {
	if o == nil || o.CwsContainerUsage == nil {
		return nil, false
	}
	return o.CwsContainerUsage, true
}

// HasCwsContainerUsage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasCwsContainerUsage() bool {
	if o != nil && o.CwsContainerUsage != nil {
		return true
	}

	return false
}

// SetCwsContainerUsage gets a reference to the given float64 and assigns it to the CwsContainerUsage field.
func (o *UsageAttributionValues) SetCwsContainerUsage(v float64) {
	o.CwsContainerUsage = &v
}

// GetCwsHostPercentage returns the CwsHostPercentage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetCwsHostPercentage() float64 {
	if o == nil || o.CwsHostPercentage == nil {
		var ret float64
		return ret
	}
	return *o.CwsHostPercentage
}

// GetCwsHostPercentageOk returns a tuple with the CwsHostPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetCwsHostPercentageOk() (*float64, bool) {
	if o == nil || o.CwsHostPercentage == nil {
		return nil, false
	}
	return o.CwsHostPercentage, true
}

// HasCwsHostPercentage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasCwsHostPercentage() bool {
	if o != nil && o.CwsHostPercentage != nil {
		return true
	}

	return false
}

// SetCwsHostPercentage gets a reference to the given float64 and assigns it to the CwsHostPercentage field.
func (o *UsageAttributionValues) SetCwsHostPercentage(v float64) {
	o.CwsHostPercentage = &v
}

// GetCwsHostUsage returns the CwsHostUsage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetCwsHostUsage() float64 {
	if o == nil || o.CwsHostUsage == nil {
		var ret float64
		return ret
	}
	return *o.CwsHostUsage
}

// GetCwsHostUsageOk returns a tuple with the CwsHostUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetCwsHostUsageOk() (*float64, bool) {
	if o == nil || o.CwsHostUsage == nil {
		return nil, false
	}
	return o.CwsHostUsage, true
}

// HasCwsHostUsage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasCwsHostUsage() bool {
	if o != nil && o.CwsHostUsage != nil {
		return true
	}

	return false
}

// SetCwsHostUsage gets a reference to the given float64 and assigns it to the CwsHostUsage field.
func (o *UsageAttributionValues) SetCwsHostUsage(v float64) {
	o.CwsHostUsage = &v
}

// GetDbmHostsPercentage returns the DbmHostsPercentage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetDbmHostsPercentage() float64 {
	if o == nil || o.DbmHostsPercentage == nil {
		var ret float64
		return ret
	}
	return *o.DbmHostsPercentage
}

// GetDbmHostsPercentageOk returns a tuple with the DbmHostsPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetDbmHostsPercentageOk() (*float64, bool) {
	if o == nil || o.DbmHostsPercentage == nil {
		return nil, false
	}
	return o.DbmHostsPercentage, true
}

// HasDbmHostsPercentage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasDbmHostsPercentage() bool {
	if o != nil && o.DbmHostsPercentage != nil {
		return true
	}

	return false
}

// SetDbmHostsPercentage gets a reference to the given float64 and assigns it to the DbmHostsPercentage field.
func (o *UsageAttributionValues) SetDbmHostsPercentage(v float64) {
	o.DbmHostsPercentage = &v
}

// GetDbmHostsUsage returns the DbmHostsUsage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetDbmHostsUsage() float64 {
	if o == nil || o.DbmHostsUsage == nil {
		var ret float64
		return ret
	}
	return *o.DbmHostsUsage
}

// GetDbmHostsUsageOk returns a tuple with the DbmHostsUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetDbmHostsUsageOk() (*float64, bool) {
	if o == nil || o.DbmHostsUsage == nil {
		return nil, false
	}
	return o.DbmHostsUsage, true
}

// HasDbmHostsUsage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasDbmHostsUsage() bool {
	if o != nil && o.DbmHostsUsage != nil {
		return true
	}

	return false
}

// SetDbmHostsUsage gets a reference to the given float64 and assigns it to the DbmHostsUsage field.
func (o *UsageAttributionValues) SetDbmHostsUsage(v float64) {
	o.DbmHostsUsage = &v
}

// GetDbmQueriesPercentage returns the DbmQueriesPercentage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetDbmQueriesPercentage() float64 {
	if o == nil || o.DbmQueriesPercentage == nil {
		var ret float64
		return ret
	}
	return *o.DbmQueriesPercentage
}

// GetDbmQueriesPercentageOk returns a tuple with the DbmQueriesPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetDbmQueriesPercentageOk() (*float64, bool) {
	if o == nil || o.DbmQueriesPercentage == nil {
		return nil, false
	}
	return o.DbmQueriesPercentage, true
}

// HasDbmQueriesPercentage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasDbmQueriesPercentage() bool {
	if o != nil && o.DbmQueriesPercentage != nil {
		return true
	}

	return false
}

// SetDbmQueriesPercentage gets a reference to the given float64 and assigns it to the DbmQueriesPercentage field.
func (o *UsageAttributionValues) SetDbmQueriesPercentage(v float64) {
	o.DbmQueriesPercentage = &v
}

// GetDbmQueriesUsage returns the DbmQueriesUsage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetDbmQueriesUsage() float64 {
	if o == nil || o.DbmQueriesUsage == nil {
		var ret float64
		return ret
	}
	return *o.DbmQueriesUsage
}

// GetDbmQueriesUsageOk returns a tuple with the DbmQueriesUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetDbmQueriesUsageOk() (*float64, bool) {
	if o == nil || o.DbmQueriesUsage == nil {
		return nil, false
	}
	return o.DbmQueriesUsage, true
}

// HasDbmQueriesUsage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasDbmQueriesUsage() bool {
	if o != nil && o.DbmQueriesUsage != nil {
		return true
	}

	return false
}

// SetDbmQueriesUsage gets a reference to the given float64 and assigns it to the DbmQueriesUsage field.
func (o *UsageAttributionValues) SetDbmQueriesUsage(v float64) {
	o.DbmQueriesUsage = &v
}

// GetEstimatedIndexedLogsPercentage returns the EstimatedIndexedLogsPercentage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetEstimatedIndexedLogsPercentage() float64 {
	if o == nil || o.EstimatedIndexedLogsPercentage == nil {
		var ret float64
		return ret
	}
	return *o.EstimatedIndexedLogsPercentage
}

// GetEstimatedIndexedLogsPercentageOk returns a tuple with the EstimatedIndexedLogsPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetEstimatedIndexedLogsPercentageOk() (*float64, bool) {
	if o == nil || o.EstimatedIndexedLogsPercentage == nil {
		return nil, false
	}
	return o.EstimatedIndexedLogsPercentage, true
}

// HasEstimatedIndexedLogsPercentage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasEstimatedIndexedLogsPercentage() bool {
	if o != nil && o.EstimatedIndexedLogsPercentage != nil {
		return true
	}

	return false
}

// SetEstimatedIndexedLogsPercentage gets a reference to the given float64 and assigns it to the EstimatedIndexedLogsPercentage field.
func (o *UsageAttributionValues) SetEstimatedIndexedLogsPercentage(v float64) {
	o.EstimatedIndexedLogsPercentage = &v
}

// GetEstimatedIndexedLogsUsage returns the EstimatedIndexedLogsUsage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetEstimatedIndexedLogsUsage() float64 {
	if o == nil || o.EstimatedIndexedLogsUsage == nil {
		var ret float64
		return ret
	}
	return *o.EstimatedIndexedLogsUsage
}

// GetEstimatedIndexedLogsUsageOk returns a tuple with the EstimatedIndexedLogsUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetEstimatedIndexedLogsUsageOk() (*float64, bool) {
	if o == nil || o.EstimatedIndexedLogsUsage == nil {
		return nil, false
	}
	return o.EstimatedIndexedLogsUsage, true
}

// HasEstimatedIndexedLogsUsage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasEstimatedIndexedLogsUsage() bool {
	if o != nil && o.EstimatedIndexedLogsUsage != nil {
		return true
	}

	return false
}

// SetEstimatedIndexedLogsUsage gets a reference to the given float64 and assigns it to the EstimatedIndexedLogsUsage field.
func (o *UsageAttributionValues) SetEstimatedIndexedLogsUsage(v float64) {
	o.EstimatedIndexedLogsUsage = &v
}

// GetEstimatedIndexedSpansPercentage returns the EstimatedIndexedSpansPercentage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetEstimatedIndexedSpansPercentage() float64 {
	if o == nil || o.EstimatedIndexedSpansPercentage == nil {
		var ret float64
		return ret
	}
	return *o.EstimatedIndexedSpansPercentage
}

// GetEstimatedIndexedSpansPercentageOk returns a tuple with the EstimatedIndexedSpansPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetEstimatedIndexedSpansPercentageOk() (*float64, bool) {
	if o == nil || o.EstimatedIndexedSpansPercentage == nil {
		return nil, false
	}
	return o.EstimatedIndexedSpansPercentage, true
}

// HasEstimatedIndexedSpansPercentage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasEstimatedIndexedSpansPercentage() bool {
	if o != nil && o.EstimatedIndexedSpansPercentage != nil {
		return true
	}

	return false
}

// SetEstimatedIndexedSpansPercentage gets a reference to the given float64 and assigns it to the EstimatedIndexedSpansPercentage field.
func (o *UsageAttributionValues) SetEstimatedIndexedSpansPercentage(v float64) {
	o.EstimatedIndexedSpansPercentage = &v
}

// GetEstimatedIndexedSpansUsage returns the EstimatedIndexedSpansUsage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetEstimatedIndexedSpansUsage() float64 {
	if o == nil || o.EstimatedIndexedSpansUsage == nil {
		var ret float64
		return ret
	}
	return *o.EstimatedIndexedSpansUsage
}

// GetEstimatedIndexedSpansUsageOk returns a tuple with the EstimatedIndexedSpansUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetEstimatedIndexedSpansUsageOk() (*float64, bool) {
	if o == nil || o.EstimatedIndexedSpansUsage == nil {
		return nil, false
	}
	return o.EstimatedIndexedSpansUsage, true
}

// HasEstimatedIndexedSpansUsage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasEstimatedIndexedSpansUsage() bool {
	if o != nil && o.EstimatedIndexedSpansUsage != nil {
		return true
	}

	return false
}

// SetEstimatedIndexedSpansUsage gets a reference to the given float64 and assigns it to the EstimatedIndexedSpansUsage field.
func (o *UsageAttributionValues) SetEstimatedIndexedSpansUsage(v float64) {
	o.EstimatedIndexedSpansUsage = &v
}

// GetEstimatedIngestedSpansPercentage returns the EstimatedIngestedSpansPercentage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetEstimatedIngestedSpansPercentage() float64 {
	if o == nil || o.EstimatedIngestedSpansPercentage == nil {
		var ret float64
		return ret
	}
	return *o.EstimatedIngestedSpansPercentage
}

// GetEstimatedIngestedSpansPercentageOk returns a tuple with the EstimatedIngestedSpansPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetEstimatedIngestedSpansPercentageOk() (*float64, bool) {
	if o == nil || o.EstimatedIngestedSpansPercentage == nil {
		return nil, false
	}
	return o.EstimatedIngestedSpansPercentage, true
}

// HasEstimatedIngestedSpansPercentage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasEstimatedIngestedSpansPercentage() bool {
	if o != nil && o.EstimatedIngestedSpansPercentage != nil {
		return true
	}

	return false
}

// SetEstimatedIngestedSpansPercentage gets a reference to the given float64 and assigns it to the EstimatedIngestedSpansPercentage field.
func (o *UsageAttributionValues) SetEstimatedIngestedSpansPercentage(v float64) {
	o.EstimatedIngestedSpansPercentage = &v
}

// GetEstimatedIngestedSpansUsage returns the EstimatedIngestedSpansUsage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetEstimatedIngestedSpansUsage() float64 {
	if o == nil || o.EstimatedIngestedSpansUsage == nil {
		var ret float64
		return ret
	}
	return *o.EstimatedIngestedSpansUsage
}

// GetEstimatedIngestedSpansUsageOk returns a tuple with the EstimatedIngestedSpansUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetEstimatedIngestedSpansUsageOk() (*float64, bool) {
	if o == nil || o.EstimatedIngestedSpansUsage == nil {
		return nil, false
	}
	return o.EstimatedIngestedSpansUsage, true
}

// HasEstimatedIngestedSpansUsage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasEstimatedIngestedSpansUsage() bool {
	if o != nil && o.EstimatedIngestedSpansUsage != nil {
		return true
	}

	return false
}

// SetEstimatedIngestedSpansUsage gets a reference to the given float64 and assigns it to the EstimatedIngestedSpansUsage field.
func (o *UsageAttributionValues) SetEstimatedIngestedSpansUsage(v float64) {
	o.EstimatedIngestedSpansUsage = &v
}

// GetInfraHostPercentage returns the InfraHostPercentage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetInfraHostPercentage() float64 {
	if o == nil || o.InfraHostPercentage == nil {
		var ret float64
		return ret
	}
	return *o.InfraHostPercentage
}

// GetInfraHostPercentageOk returns a tuple with the InfraHostPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetInfraHostPercentageOk() (*float64, bool) {
	if o == nil || o.InfraHostPercentage == nil {
		return nil, false
	}
	return o.InfraHostPercentage, true
}

// HasInfraHostPercentage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasInfraHostPercentage() bool {
	if o != nil && o.InfraHostPercentage != nil {
		return true
	}

	return false
}

// SetInfraHostPercentage gets a reference to the given float64 and assigns it to the InfraHostPercentage field.
func (o *UsageAttributionValues) SetInfraHostPercentage(v float64) {
	o.InfraHostPercentage = &v
}

// GetInfraHostUsage returns the InfraHostUsage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetInfraHostUsage() float64 {
	if o == nil || o.InfraHostUsage == nil {
		var ret float64
		return ret
	}
	return *o.InfraHostUsage
}

// GetInfraHostUsageOk returns a tuple with the InfraHostUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetInfraHostUsageOk() (*float64, bool) {
	if o == nil || o.InfraHostUsage == nil {
		return nil, false
	}
	return o.InfraHostUsage, true
}

// HasInfraHostUsage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasInfraHostUsage() bool {
	if o != nil && o.InfraHostUsage != nil {
		return true
	}

	return false
}

// SetInfraHostUsage gets a reference to the given float64 and assigns it to the InfraHostUsage field.
func (o *UsageAttributionValues) SetInfraHostUsage(v float64) {
	o.InfraHostUsage = &v
}

// GetLambdaFunctionsPercentage returns the LambdaFunctionsPercentage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetLambdaFunctionsPercentage() float64 {
	if o == nil || o.LambdaFunctionsPercentage == nil {
		var ret float64
		return ret
	}
	return *o.LambdaFunctionsPercentage
}

// GetLambdaFunctionsPercentageOk returns a tuple with the LambdaFunctionsPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetLambdaFunctionsPercentageOk() (*float64, bool) {
	if o == nil || o.LambdaFunctionsPercentage == nil {
		return nil, false
	}
	return o.LambdaFunctionsPercentage, true
}

// HasLambdaFunctionsPercentage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasLambdaFunctionsPercentage() bool {
	if o != nil && o.LambdaFunctionsPercentage != nil {
		return true
	}

	return false
}

// SetLambdaFunctionsPercentage gets a reference to the given float64 and assigns it to the LambdaFunctionsPercentage field.
func (o *UsageAttributionValues) SetLambdaFunctionsPercentage(v float64) {
	o.LambdaFunctionsPercentage = &v
}

// GetLambdaFunctionsUsage returns the LambdaFunctionsUsage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetLambdaFunctionsUsage() float64 {
	if o == nil || o.LambdaFunctionsUsage == nil {
		var ret float64
		return ret
	}
	return *o.LambdaFunctionsUsage
}

// GetLambdaFunctionsUsageOk returns a tuple with the LambdaFunctionsUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetLambdaFunctionsUsageOk() (*float64, bool) {
	if o == nil || o.LambdaFunctionsUsage == nil {
		return nil, false
	}
	return o.LambdaFunctionsUsage, true
}

// HasLambdaFunctionsUsage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasLambdaFunctionsUsage() bool {
	if o != nil && o.LambdaFunctionsUsage != nil {
		return true
	}

	return false
}

// SetLambdaFunctionsUsage gets a reference to the given float64 and assigns it to the LambdaFunctionsUsage field.
func (o *UsageAttributionValues) SetLambdaFunctionsUsage(v float64) {
	o.LambdaFunctionsUsage = &v
}

// GetLambdaInvocationsPercentage returns the LambdaInvocationsPercentage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetLambdaInvocationsPercentage() float64 {
	if o == nil || o.LambdaInvocationsPercentage == nil {
		var ret float64
		return ret
	}
	return *o.LambdaInvocationsPercentage
}

// GetLambdaInvocationsPercentageOk returns a tuple with the LambdaInvocationsPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetLambdaInvocationsPercentageOk() (*float64, bool) {
	if o == nil || o.LambdaInvocationsPercentage == nil {
		return nil, false
	}
	return o.LambdaInvocationsPercentage, true
}

// HasLambdaInvocationsPercentage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasLambdaInvocationsPercentage() bool {
	if o != nil && o.LambdaInvocationsPercentage != nil {
		return true
	}

	return false
}

// SetLambdaInvocationsPercentage gets a reference to the given float64 and assigns it to the LambdaInvocationsPercentage field.
func (o *UsageAttributionValues) SetLambdaInvocationsPercentage(v float64) {
	o.LambdaInvocationsPercentage = &v
}

// GetLambdaInvocationsUsage returns the LambdaInvocationsUsage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetLambdaInvocationsUsage() float64 {
	if o == nil || o.LambdaInvocationsUsage == nil {
		var ret float64
		return ret
	}
	return *o.LambdaInvocationsUsage
}

// GetLambdaInvocationsUsageOk returns a tuple with the LambdaInvocationsUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetLambdaInvocationsUsageOk() (*float64, bool) {
	if o == nil || o.LambdaInvocationsUsage == nil {
		return nil, false
	}
	return o.LambdaInvocationsUsage, true
}

// HasLambdaInvocationsUsage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasLambdaInvocationsUsage() bool {
	if o != nil && o.LambdaInvocationsUsage != nil {
		return true
	}

	return false
}

// SetLambdaInvocationsUsage gets a reference to the given float64 and assigns it to the LambdaInvocationsUsage field.
func (o *UsageAttributionValues) SetLambdaInvocationsUsage(v float64) {
	o.LambdaInvocationsUsage = &v
}

// GetNpmHostPercentage returns the NpmHostPercentage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetNpmHostPercentage() float64 {
	if o == nil || o.NpmHostPercentage == nil {
		var ret float64
		return ret
	}
	return *o.NpmHostPercentage
}

// GetNpmHostPercentageOk returns a tuple with the NpmHostPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetNpmHostPercentageOk() (*float64, bool) {
	if o == nil || o.NpmHostPercentage == nil {
		return nil, false
	}
	return o.NpmHostPercentage, true
}

// HasNpmHostPercentage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasNpmHostPercentage() bool {
	if o != nil && o.NpmHostPercentage != nil {
		return true
	}

	return false
}

// SetNpmHostPercentage gets a reference to the given float64 and assigns it to the NpmHostPercentage field.
func (o *UsageAttributionValues) SetNpmHostPercentage(v float64) {
	o.NpmHostPercentage = &v
}

// GetNpmHostUsage returns the NpmHostUsage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetNpmHostUsage() float64 {
	if o == nil || o.NpmHostUsage == nil {
		var ret float64
		return ret
	}
	return *o.NpmHostUsage
}

// GetNpmHostUsageOk returns a tuple with the NpmHostUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetNpmHostUsageOk() (*float64, bool) {
	if o == nil || o.NpmHostUsage == nil {
		return nil, false
	}
	return o.NpmHostUsage, true
}

// HasNpmHostUsage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasNpmHostUsage() bool {
	if o != nil && o.NpmHostUsage != nil {
		return true
	}

	return false
}

// SetNpmHostUsage gets a reference to the given float64 and assigns it to the NpmHostUsage field.
func (o *UsageAttributionValues) SetNpmHostUsage(v float64) {
	o.NpmHostUsage = &v
}

// GetProfiledContainerPercentage returns the ProfiledContainerPercentage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetProfiledContainerPercentage() float64 {
	if o == nil || o.ProfiledContainerPercentage == nil {
		var ret float64
		return ret
	}
	return *o.ProfiledContainerPercentage
}

// GetProfiledContainerPercentageOk returns a tuple with the ProfiledContainerPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetProfiledContainerPercentageOk() (*float64, bool) {
	if o == nil || o.ProfiledContainerPercentage == nil {
		return nil, false
	}
	return o.ProfiledContainerPercentage, true
}

// HasProfiledContainerPercentage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasProfiledContainerPercentage() bool {
	if o != nil && o.ProfiledContainerPercentage != nil {
		return true
	}

	return false
}

// SetProfiledContainerPercentage gets a reference to the given float64 and assigns it to the ProfiledContainerPercentage field.
func (o *UsageAttributionValues) SetProfiledContainerPercentage(v float64) {
	o.ProfiledContainerPercentage = &v
}

// GetProfiledContainerUsage returns the ProfiledContainerUsage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetProfiledContainerUsage() float64 {
	if o == nil || o.ProfiledContainerUsage == nil {
		var ret float64
		return ret
	}
	return *o.ProfiledContainerUsage
}

// GetProfiledContainerUsageOk returns a tuple with the ProfiledContainerUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetProfiledContainerUsageOk() (*float64, bool) {
	if o == nil || o.ProfiledContainerUsage == nil {
		return nil, false
	}
	return o.ProfiledContainerUsage, true
}

// HasProfiledContainerUsage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasProfiledContainerUsage() bool {
	if o != nil && o.ProfiledContainerUsage != nil {
		return true
	}

	return false
}

// SetProfiledContainerUsage gets a reference to the given float64 and assigns it to the ProfiledContainerUsage field.
func (o *UsageAttributionValues) SetProfiledContainerUsage(v float64) {
	o.ProfiledContainerUsage = &v
}

// GetProfiledHostsPercentage returns the ProfiledHostsPercentage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetProfiledHostsPercentage() float64 {
	if o == nil || o.ProfiledHostsPercentage == nil {
		var ret float64
		return ret
	}
	return *o.ProfiledHostsPercentage
}

// GetProfiledHostsPercentageOk returns a tuple with the ProfiledHostsPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetProfiledHostsPercentageOk() (*float64, bool) {
	if o == nil || o.ProfiledHostsPercentage == nil {
		return nil, false
	}
	return o.ProfiledHostsPercentage, true
}

// HasProfiledHostsPercentage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasProfiledHostsPercentage() bool {
	if o != nil && o.ProfiledHostsPercentage != nil {
		return true
	}

	return false
}

// SetProfiledHostsPercentage gets a reference to the given float64 and assigns it to the ProfiledHostsPercentage field.
func (o *UsageAttributionValues) SetProfiledHostsPercentage(v float64) {
	o.ProfiledHostsPercentage = &v
}

// GetProfiledHostsUsage returns the ProfiledHostsUsage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetProfiledHostsUsage() float64 {
	if o == nil || o.ProfiledHostsUsage == nil {
		var ret float64
		return ret
	}
	return *o.ProfiledHostsUsage
}

// GetProfiledHostsUsageOk returns a tuple with the ProfiledHostsUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetProfiledHostsUsageOk() (*float64, bool) {
	if o == nil || o.ProfiledHostsUsage == nil {
		return nil, false
	}
	return o.ProfiledHostsUsage, true
}

// HasProfiledHostsUsage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasProfiledHostsUsage() bool {
	if o != nil && o.ProfiledHostsUsage != nil {
		return true
	}

	return false
}

// SetProfiledHostsUsage gets a reference to the given float64 and assigns it to the ProfiledHostsUsage field.
func (o *UsageAttributionValues) SetProfiledHostsUsage(v float64) {
	o.ProfiledHostsUsage = &v
}

// GetSnmpPercentage returns the SnmpPercentage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetSnmpPercentage() float64 {
	if o == nil || o.SnmpPercentage == nil {
		var ret float64
		return ret
	}
	return *o.SnmpPercentage
}

// GetSnmpPercentageOk returns a tuple with the SnmpPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetSnmpPercentageOk() (*float64, bool) {
	if o == nil || o.SnmpPercentage == nil {
		return nil, false
	}
	return o.SnmpPercentage, true
}

// HasSnmpPercentage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasSnmpPercentage() bool {
	if o != nil && o.SnmpPercentage != nil {
		return true
	}

	return false
}

// SetSnmpPercentage gets a reference to the given float64 and assigns it to the SnmpPercentage field.
func (o *UsageAttributionValues) SetSnmpPercentage(v float64) {
	o.SnmpPercentage = &v
}

// GetSnmpUsage returns the SnmpUsage field value if set, zero value otherwise.
func (o *UsageAttributionValues) GetSnmpUsage() float64 {
	if o == nil || o.SnmpUsage == nil {
		var ret float64
		return ret
	}
	return *o.SnmpUsage
}

// GetSnmpUsageOk returns a tuple with the SnmpUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageAttributionValues) GetSnmpUsageOk() (*float64, bool) {
	if o == nil || o.SnmpUsage == nil {
		return nil, false
	}
	return o.SnmpUsage, true
}

// HasSnmpUsage returns a boolean if a field has been set.
func (o *UsageAttributionValues) HasSnmpUsage() bool {
	if o != nil && o.SnmpUsage != nil {
		return true
	}

	return false
}

// SetSnmpUsage gets a reference to the given float64 and assigns it to the SnmpUsage field.
func (o *UsageAttributionValues) SetSnmpUsage(v float64) {
	o.SnmpUsage = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageAttributionValues) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.ApiPercentage != nil {
		toSerialize["api_percentage"] = o.ApiPercentage
	}
	if o.ApiUsage != nil {
		toSerialize["api_usage"] = o.ApiUsage
	}
	if o.ApmHostPercentage != nil {
		toSerialize["apm_host_percentage"] = o.ApmHostPercentage
	}
	if o.ApmHostUsage != nil {
		toSerialize["apm_host_usage"] = o.ApmHostUsage
	}
	if o.AppsecPercentage != nil {
		toSerialize["appsec_percentage"] = o.AppsecPercentage
	}
	if o.AppsecUsage != nil {
		toSerialize["appsec_usage"] = o.AppsecUsage
	}
	if o.BrowserPercentage != nil {
		toSerialize["browser_percentage"] = o.BrowserPercentage
	}
	if o.BrowserUsage != nil {
		toSerialize["browser_usage"] = o.BrowserUsage
	}
	if o.ContainerPercentage != nil {
		toSerialize["container_percentage"] = o.ContainerPercentage
	}
	if o.ContainerUsage != nil {
		toSerialize["container_usage"] = o.ContainerUsage
	}
	if o.CspmContainerPercentage != nil {
		toSerialize["cspm_container_percentage"] = o.CspmContainerPercentage
	}
	if o.CspmContainerUsage != nil {
		toSerialize["cspm_container_usage"] = o.CspmContainerUsage
	}
	if o.CspmHostPercentage != nil {
		toSerialize["cspm_host_percentage"] = o.CspmHostPercentage
	}
	if o.CspmHostUsage != nil {
		toSerialize["cspm_host_usage"] = o.CspmHostUsage
	}
	if o.CustomTimeseriesPercentage != nil {
		toSerialize["custom_timeseries_percentage"] = o.CustomTimeseriesPercentage
	}
	if o.CustomTimeseriesUsage != nil {
		toSerialize["custom_timeseries_usage"] = o.CustomTimeseriesUsage
	}
	if o.CwsContainerPercentage != nil {
		toSerialize["cws_container_percentage"] = o.CwsContainerPercentage
	}
	if o.CwsContainerUsage != nil {
		toSerialize["cws_container_usage"] = o.CwsContainerUsage
	}
	if o.CwsHostPercentage != nil {
		toSerialize["cws_host_percentage"] = o.CwsHostPercentage
	}
	if o.CwsHostUsage != nil {
		toSerialize["cws_host_usage"] = o.CwsHostUsage
	}
	if o.DbmHostsPercentage != nil {
		toSerialize["dbm_hosts_percentage"] = o.DbmHostsPercentage
	}
	if o.DbmHostsUsage != nil {
		toSerialize["dbm_hosts_usage"] = o.DbmHostsUsage
	}
	if o.DbmQueriesPercentage != nil {
		toSerialize["dbm_queries_percentage"] = o.DbmQueriesPercentage
	}
	if o.DbmQueriesUsage != nil {
		toSerialize["dbm_queries_usage"] = o.DbmQueriesUsage
	}
	if o.EstimatedIndexedLogsPercentage != nil {
		toSerialize["estimated_indexed_logs_percentage"] = o.EstimatedIndexedLogsPercentage
	}
	if o.EstimatedIndexedLogsUsage != nil {
		toSerialize["estimated_indexed_logs_usage"] = o.EstimatedIndexedLogsUsage
	}
	if o.EstimatedIndexedSpansPercentage != nil {
		toSerialize["estimated_indexed_spans_percentage"] = o.EstimatedIndexedSpansPercentage
	}
	if o.EstimatedIndexedSpansUsage != nil {
		toSerialize["estimated_indexed_spans_usage"] = o.EstimatedIndexedSpansUsage
	}
	if o.EstimatedIngestedSpansPercentage != nil {
		toSerialize["estimated_ingested_spans_percentage"] = o.EstimatedIngestedSpansPercentage
	}
	if o.EstimatedIngestedSpansUsage != nil {
		toSerialize["estimated_ingested_spans_usage"] = o.EstimatedIngestedSpansUsage
	}
	if o.InfraHostPercentage != nil {
		toSerialize["infra_host_percentage"] = o.InfraHostPercentage
	}
	if o.InfraHostUsage != nil {
		toSerialize["infra_host_usage"] = o.InfraHostUsage
	}
	if o.LambdaFunctionsPercentage != nil {
		toSerialize["lambda_functions_percentage"] = o.LambdaFunctionsPercentage
	}
	if o.LambdaFunctionsUsage != nil {
		toSerialize["lambda_functions_usage"] = o.LambdaFunctionsUsage
	}
	if o.LambdaInvocationsPercentage != nil {
		toSerialize["lambda_invocations_percentage"] = o.LambdaInvocationsPercentage
	}
	if o.LambdaInvocationsUsage != nil {
		toSerialize["lambda_invocations_usage"] = o.LambdaInvocationsUsage
	}
	if o.NpmHostPercentage != nil {
		toSerialize["npm_host_percentage"] = o.NpmHostPercentage
	}
	if o.NpmHostUsage != nil {
		toSerialize["npm_host_usage"] = o.NpmHostUsage
	}
	if o.ProfiledContainerPercentage != nil {
		toSerialize["profiled_container_percentage"] = o.ProfiledContainerPercentage
	}
	if o.ProfiledContainerUsage != nil {
		toSerialize["profiled_container_usage"] = o.ProfiledContainerUsage
	}
	if o.ProfiledHostsPercentage != nil {
		toSerialize["profiled_hosts_percentage"] = o.ProfiledHostsPercentage
	}
	if o.ProfiledHostsUsage != nil {
		toSerialize["profiled_hosts_usage"] = o.ProfiledHostsUsage
	}
	if o.SnmpPercentage != nil {
		toSerialize["snmp_percentage"] = o.SnmpPercentage
	}
	if o.SnmpUsage != nil {
		toSerialize["snmp_usage"] = o.SnmpUsage
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UsageAttributionValues) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		ApiPercentage                    *float64 `json:"api_percentage,omitempty"`
		ApiUsage                         *float64 `json:"api_usage,omitempty"`
		ApmHostPercentage                *float64 `json:"apm_host_percentage,omitempty"`
		ApmHostUsage                     *float64 `json:"apm_host_usage,omitempty"`
		AppsecPercentage                 *float64 `json:"appsec_percentage,omitempty"`
		AppsecUsage                      *float64 `json:"appsec_usage,omitempty"`
		BrowserPercentage                *float64 `json:"browser_percentage,omitempty"`
		BrowserUsage                     *float64 `json:"browser_usage,omitempty"`
		ContainerPercentage              *float64 `json:"container_percentage,omitempty"`
		ContainerUsage                   *float64 `json:"container_usage,omitempty"`
		CspmContainerPercentage          *float64 `json:"cspm_container_percentage,omitempty"`
		CspmContainerUsage               *float64 `json:"cspm_container_usage,omitempty"`
		CspmHostPercentage               *float64 `json:"cspm_host_percentage,omitempty"`
		CspmHostUsage                    *float64 `json:"cspm_host_usage,omitempty"`
		CustomTimeseriesPercentage       *float64 `json:"custom_timeseries_percentage,omitempty"`
		CustomTimeseriesUsage            *float64 `json:"custom_timeseries_usage,omitempty"`
		CwsContainerPercentage           *float64 `json:"cws_container_percentage,omitempty"`
		CwsContainerUsage                *float64 `json:"cws_container_usage,omitempty"`
		CwsHostPercentage                *float64 `json:"cws_host_percentage,omitempty"`
		CwsHostUsage                     *float64 `json:"cws_host_usage,omitempty"`
		DbmHostsPercentage               *float64 `json:"dbm_hosts_percentage,omitempty"`
		DbmHostsUsage                    *float64 `json:"dbm_hosts_usage,omitempty"`
		DbmQueriesPercentage             *float64 `json:"dbm_queries_percentage,omitempty"`
		DbmQueriesUsage                  *float64 `json:"dbm_queries_usage,omitempty"`
		EstimatedIndexedLogsPercentage   *float64 `json:"estimated_indexed_logs_percentage,omitempty"`
		EstimatedIndexedLogsUsage        *float64 `json:"estimated_indexed_logs_usage,omitempty"`
		EstimatedIndexedSpansPercentage  *float64 `json:"estimated_indexed_spans_percentage,omitempty"`
		EstimatedIndexedSpansUsage       *float64 `json:"estimated_indexed_spans_usage,omitempty"`
		EstimatedIngestedSpansPercentage *float64 `json:"estimated_ingested_spans_percentage,omitempty"`
		EstimatedIngestedSpansUsage      *float64 `json:"estimated_ingested_spans_usage,omitempty"`
		InfraHostPercentage              *float64 `json:"infra_host_percentage,omitempty"`
		InfraHostUsage                   *float64 `json:"infra_host_usage,omitempty"`
		LambdaFunctionsPercentage        *float64 `json:"lambda_functions_percentage,omitempty"`
		LambdaFunctionsUsage             *float64 `json:"lambda_functions_usage,omitempty"`
		LambdaInvocationsPercentage      *float64 `json:"lambda_invocations_percentage,omitempty"`
		LambdaInvocationsUsage           *float64 `json:"lambda_invocations_usage,omitempty"`
		NpmHostPercentage                *float64 `json:"npm_host_percentage,omitempty"`
		NpmHostUsage                     *float64 `json:"npm_host_usage,omitempty"`
		ProfiledContainerPercentage      *float64 `json:"profiled_container_percentage,omitempty"`
		ProfiledContainerUsage           *float64 `json:"profiled_container_usage,omitempty"`
		ProfiledHostsPercentage          *float64 `json:"profiled_hosts_percentage,omitempty"`
		ProfiledHostsUsage               *float64 `json:"profiled_hosts_usage,omitempty"`
		SnmpPercentage                   *float64 `json:"snmp_percentage,omitempty"`
		SnmpUsage                        *float64 `json:"snmp_usage,omitempty"`
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
	o.ApiPercentage = all.ApiPercentage
	o.ApiUsage = all.ApiUsage
	o.ApmHostPercentage = all.ApmHostPercentage
	o.ApmHostUsage = all.ApmHostUsage
	o.AppsecPercentage = all.AppsecPercentage
	o.AppsecUsage = all.AppsecUsage
	o.BrowserPercentage = all.BrowserPercentage
	o.BrowserUsage = all.BrowserUsage
	o.ContainerPercentage = all.ContainerPercentage
	o.ContainerUsage = all.ContainerUsage
	o.CspmContainerPercentage = all.CspmContainerPercentage
	o.CspmContainerUsage = all.CspmContainerUsage
	o.CspmHostPercentage = all.CspmHostPercentage
	o.CspmHostUsage = all.CspmHostUsage
	o.CustomTimeseriesPercentage = all.CustomTimeseriesPercentage
	o.CustomTimeseriesUsage = all.CustomTimeseriesUsage
	o.CwsContainerPercentage = all.CwsContainerPercentage
	o.CwsContainerUsage = all.CwsContainerUsage
	o.CwsHostPercentage = all.CwsHostPercentage
	o.CwsHostUsage = all.CwsHostUsage
	o.DbmHostsPercentage = all.DbmHostsPercentage
	o.DbmHostsUsage = all.DbmHostsUsage
	o.DbmQueriesPercentage = all.DbmQueriesPercentage
	o.DbmQueriesUsage = all.DbmQueriesUsage
	o.EstimatedIndexedLogsPercentage = all.EstimatedIndexedLogsPercentage
	o.EstimatedIndexedLogsUsage = all.EstimatedIndexedLogsUsage
	o.EstimatedIndexedSpansPercentage = all.EstimatedIndexedSpansPercentage
	o.EstimatedIndexedSpansUsage = all.EstimatedIndexedSpansUsage
	o.EstimatedIngestedSpansPercentage = all.EstimatedIngestedSpansPercentage
	o.EstimatedIngestedSpansUsage = all.EstimatedIngestedSpansUsage
	o.InfraHostPercentage = all.InfraHostPercentage
	o.InfraHostUsage = all.InfraHostUsage
	o.LambdaFunctionsPercentage = all.LambdaFunctionsPercentage
	o.LambdaFunctionsUsage = all.LambdaFunctionsUsage
	o.LambdaInvocationsPercentage = all.LambdaInvocationsPercentage
	o.LambdaInvocationsUsage = all.LambdaInvocationsUsage
	o.NpmHostPercentage = all.NpmHostPercentage
	o.NpmHostUsage = all.NpmHostUsage
	o.ProfiledContainerPercentage = all.ProfiledContainerPercentage
	o.ProfiledContainerUsage = all.ProfiledContainerUsage
	o.ProfiledHostsPercentage = all.ProfiledHostsPercentage
	o.ProfiledHostsUsage = all.ProfiledHostsUsage
	o.SnmpPercentage = all.SnmpPercentage
	o.SnmpUsage = all.SnmpUsage
	return nil
}
