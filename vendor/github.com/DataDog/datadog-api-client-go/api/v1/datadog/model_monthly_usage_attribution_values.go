// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// MonthlyUsageAttributionValues Fields in Usage Summary by tag(s).
type MonthlyUsageAttributionValues struct {
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
	// The percentage of custom metrics usage by tag(s).
	CustomTimeseriesPercentage *float64 `json:"custom_timeseries_percentage,omitempty"`
	// The custom metrics usage by tag(s).
	CustomTimeseriesUsage *float64 `json:"custom_timeseries_usage,omitempty"`
	// The percentage of estimated live indexed logs usage by tag(s). This field is in private beta.
	EstimatedIndexedLogsPercentage *float64 `json:"estimated_indexed_logs_percentage,omitempty"`
	// The estimated live indexed logs usage by tag(s). This field is in private beta.
	EstimatedIndexedLogsUsage *float64 `json:"estimated_indexed_logs_usage,omitempty"`
	// The percentage of estimated indexed spans usage by tag(s). This field is in private beta.
	EstimatedIndexedSpansPercentage *float64 `json:"estimated_indexed_spans_percentage,omitempty"`
	// The estimated indexed spans usage by tag(s). This field is in private beta.
	EstimatedIndexedSpansUsage *float64 `json:"estimated_indexed_spans_usage,omitempty"`
	// The percentage of estimated ingested spans usage by tag(s). This field is in private beta.
	EstimatedIngestedSpansPercentage *float64 `json:"estimated_ingested_spans_percentage,omitempty"`
	// The estimated ingested spans usage by tag(s). This field is in private beta.
	EstimatedIngestedSpansUsage *float64 `json:"estimated_ingested_spans_usage,omitempty"`
	// The percentage of Fargate usage by tags.
	FargatePercentage *float64 `json:"fargate_percentage,omitempty"`
	// The Fargate usage by tags.
	FargateUsage *float64 `json:"fargate_usage,omitempty"`
	// The percentage of Lambda function usage by tag(s).
	FunctionsPercentage *float64 `json:"functions_percentage,omitempty"`
	// The Lambda function usage by tag(s).
	FunctionsUsage *float64 `json:"functions_usage,omitempty"`
	// The percentage of indexed logs usage by tags.
	IndexedLogsPercentage *float64 `json:"indexed_logs_percentage,omitempty"`
	// The indexed logs usage by tags.
	IndexedLogsUsage *float64 `json:"indexed_logs_usage,omitempty"`
	// The percentage of infrastructure host usage by tag(s).
	InfraHostPercentage *float64 `json:"infra_host_percentage,omitempty"`
	// The infrastructure host usage by tag(s).
	InfraHostUsage *float64 `json:"infra_host_usage,omitempty"`
	// The percentage of Lambda invocation usage by tag(s).
	InvocationsPercentage *float64 `json:"invocations_percentage,omitempty"`
	// The Lambda invocation usage by tag(s).
	InvocationsUsage *float64 `json:"invocations_usage,omitempty"`
	// The percentage of network host usage by tag(s).
	NpmHostPercentage *float64 `json:"npm_host_percentage,omitempty"`
	// The network host usage by tag(s).
	NpmHostUsage *float64 `json:"npm_host_usage,omitempty"`
	// The percentage of profiled container usage by tag(s).
	ProfiledContainerPercentage *float64 `json:"profiled_container_percentage,omitempty"`
	// The profiled container usage by tag(s).
	ProfiledContainerUsage *float64 `json:"profiled_container_usage,omitempty"`
	// The percentage of profiled hosts usage by tag(s).
	ProfiledHostPercentage *float64 `json:"profiled_host_percentage,omitempty"`
	// The profiled hosts usage by tag(s).
	ProfiledHostUsage *float64 `json:"profiled_host_usage,omitempty"`
	// The percentage of network device usage by tag(s).
	SnmpPercentage *float64 `json:"snmp_percentage,omitempty"`
	// The network device usage by tag(s).
	SnmpUsage *float64 `json:"snmp_usage,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewMonthlyUsageAttributionValues instantiates a new MonthlyUsageAttributionValues object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMonthlyUsageAttributionValues() *MonthlyUsageAttributionValues {
	this := MonthlyUsageAttributionValues{}
	return &this
}

// NewMonthlyUsageAttributionValuesWithDefaults instantiates a new MonthlyUsageAttributionValues object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMonthlyUsageAttributionValuesWithDefaults() *MonthlyUsageAttributionValues {
	this := MonthlyUsageAttributionValues{}
	return &this
}

// GetApiPercentage returns the ApiPercentage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetApiPercentage() float64 {
	if o == nil || o.ApiPercentage == nil {
		var ret float64
		return ret
	}
	return *o.ApiPercentage
}

// GetApiPercentageOk returns a tuple with the ApiPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetApiPercentageOk() (*float64, bool) {
	if o == nil || o.ApiPercentage == nil {
		return nil, false
	}
	return o.ApiPercentage, true
}

// HasApiPercentage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasApiPercentage() bool {
	if o != nil && o.ApiPercentage != nil {
		return true
	}

	return false
}

// SetApiPercentage gets a reference to the given float64 and assigns it to the ApiPercentage field.
func (o *MonthlyUsageAttributionValues) SetApiPercentage(v float64) {
	o.ApiPercentage = &v
}

// GetApiUsage returns the ApiUsage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetApiUsage() float64 {
	if o == nil || o.ApiUsage == nil {
		var ret float64
		return ret
	}
	return *o.ApiUsage
}

// GetApiUsageOk returns a tuple with the ApiUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetApiUsageOk() (*float64, bool) {
	if o == nil || o.ApiUsage == nil {
		return nil, false
	}
	return o.ApiUsage, true
}

// HasApiUsage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasApiUsage() bool {
	if o != nil && o.ApiUsage != nil {
		return true
	}

	return false
}

// SetApiUsage gets a reference to the given float64 and assigns it to the ApiUsage field.
func (o *MonthlyUsageAttributionValues) SetApiUsage(v float64) {
	o.ApiUsage = &v
}

// GetApmHostPercentage returns the ApmHostPercentage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetApmHostPercentage() float64 {
	if o == nil || o.ApmHostPercentage == nil {
		var ret float64
		return ret
	}
	return *o.ApmHostPercentage
}

// GetApmHostPercentageOk returns a tuple with the ApmHostPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetApmHostPercentageOk() (*float64, bool) {
	if o == nil || o.ApmHostPercentage == nil {
		return nil, false
	}
	return o.ApmHostPercentage, true
}

// HasApmHostPercentage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasApmHostPercentage() bool {
	if o != nil && o.ApmHostPercentage != nil {
		return true
	}

	return false
}

// SetApmHostPercentage gets a reference to the given float64 and assigns it to the ApmHostPercentage field.
func (o *MonthlyUsageAttributionValues) SetApmHostPercentage(v float64) {
	o.ApmHostPercentage = &v
}

// GetApmHostUsage returns the ApmHostUsage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetApmHostUsage() float64 {
	if o == nil || o.ApmHostUsage == nil {
		var ret float64
		return ret
	}
	return *o.ApmHostUsage
}

// GetApmHostUsageOk returns a tuple with the ApmHostUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetApmHostUsageOk() (*float64, bool) {
	if o == nil || o.ApmHostUsage == nil {
		return nil, false
	}
	return o.ApmHostUsage, true
}

// HasApmHostUsage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasApmHostUsage() bool {
	if o != nil && o.ApmHostUsage != nil {
		return true
	}

	return false
}

// SetApmHostUsage gets a reference to the given float64 and assigns it to the ApmHostUsage field.
func (o *MonthlyUsageAttributionValues) SetApmHostUsage(v float64) {
	o.ApmHostUsage = &v
}

// GetAppsecPercentage returns the AppsecPercentage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetAppsecPercentage() float64 {
	if o == nil || o.AppsecPercentage == nil {
		var ret float64
		return ret
	}
	return *o.AppsecPercentage
}

// GetAppsecPercentageOk returns a tuple with the AppsecPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetAppsecPercentageOk() (*float64, bool) {
	if o == nil || o.AppsecPercentage == nil {
		return nil, false
	}
	return o.AppsecPercentage, true
}

// HasAppsecPercentage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasAppsecPercentage() bool {
	if o != nil && o.AppsecPercentage != nil {
		return true
	}

	return false
}

// SetAppsecPercentage gets a reference to the given float64 and assigns it to the AppsecPercentage field.
func (o *MonthlyUsageAttributionValues) SetAppsecPercentage(v float64) {
	o.AppsecPercentage = &v
}

// GetAppsecUsage returns the AppsecUsage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetAppsecUsage() float64 {
	if o == nil || o.AppsecUsage == nil {
		var ret float64
		return ret
	}
	return *o.AppsecUsage
}

// GetAppsecUsageOk returns a tuple with the AppsecUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetAppsecUsageOk() (*float64, bool) {
	if o == nil || o.AppsecUsage == nil {
		return nil, false
	}
	return o.AppsecUsage, true
}

// HasAppsecUsage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasAppsecUsage() bool {
	if o != nil && o.AppsecUsage != nil {
		return true
	}

	return false
}

// SetAppsecUsage gets a reference to the given float64 and assigns it to the AppsecUsage field.
func (o *MonthlyUsageAttributionValues) SetAppsecUsage(v float64) {
	o.AppsecUsage = &v
}

// GetBrowserPercentage returns the BrowserPercentage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetBrowserPercentage() float64 {
	if o == nil || o.BrowserPercentage == nil {
		var ret float64
		return ret
	}
	return *o.BrowserPercentage
}

// GetBrowserPercentageOk returns a tuple with the BrowserPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetBrowserPercentageOk() (*float64, bool) {
	if o == nil || o.BrowserPercentage == nil {
		return nil, false
	}
	return o.BrowserPercentage, true
}

// HasBrowserPercentage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasBrowserPercentage() bool {
	if o != nil && o.BrowserPercentage != nil {
		return true
	}

	return false
}

// SetBrowserPercentage gets a reference to the given float64 and assigns it to the BrowserPercentage field.
func (o *MonthlyUsageAttributionValues) SetBrowserPercentage(v float64) {
	o.BrowserPercentage = &v
}

// GetBrowserUsage returns the BrowserUsage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetBrowserUsage() float64 {
	if o == nil || o.BrowserUsage == nil {
		var ret float64
		return ret
	}
	return *o.BrowserUsage
}

// GetBrowserUsageOk returns a tuple with the BrowserUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetBrowserUsageOk() (*float64, bool) {
	if o == nil || o.BrowserUsage == nil {
		return nil, false
	}
	return o.BrowserUsage, true
}

// HasBrowserUsage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasBrowserUsage() bool {
	if o != nil && o.BrowserUsage != nil {
		return true
	}

	return false
}

// SetBrowserUsage gets a reference to the given float64 and assigns it to the BrowserUsage field.
func (o *MonthlyUsageAttributionValues) SetBrowserUsage(v float64) {
	o.BrowserUsage = &v
}

// GetContainerPercentage returns the ContainerPercentage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetContainerPercentage() float64 {
	if o == nil || o.ContainerPercentage == nil {
		var ret float64
		return ret
	}
	return *o.ContainerPercentage
}

// GetContainerPercentageOk returns a tuple with the ContainerPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetContainerPercentageOk() (*float64, bool) {
	if o == nil || o.ContainerPercentage == nil {
		return nil, false
	}
	return o.ContainerPercentage, true
}

// HasContainerPercentage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasContainerPercentage() bool {
	if o != nil && o.ContainerPercentage != nil {
		return true
	}

	return false
}

// SetContainerPercentage gets a reference to the given float64 and assigns it to the ContainerPercentage field.
func (o *MonthlyUsageAttributionValues) SetContainerPercentage(v float64) {
	o.ContainerPercentage = &v
}

// GetContainerUsage returns the ContainerUsage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetContainerUsage() float64 {
	if o == nil || o.ContainerUsage == nil {
		var ret float64
		return ret
	}
	return *o.ContainerUsage
}

// GetContainerUsageOk returns a tuple with the ContainerUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetContainerUsageOk() (*float64, bool) {
	if o == nil || o.ContainerUsage == nil {
		return nil, false
	}
	return o.ContainerUsage, true
}

// HasContainerUsage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasContainerUsage() bool {
	if o != nil && o.ContainerUsage != nil {
		return true
	}

	return false
}

// SetContainerUsage gets a reference to the given float64 and assigns it to the ContainerUsage field.
func (o *MonthlyUsageAttributionValues) SetContainerUsage(v float64) {
	o.ContainerUsage = &v
}

// GetCustomTimeseriesPercentage returns the CustomTimeseriesPercentage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetCustomTimeseriesPercentage() float64 {
	if o == nil || o.CustomTimeseriesPercentage == nil {
		var ret float64
		return ret
	}
	return *o.CustomTimeseriesPercentage
}

// GetCustomTimeseriesPercentageOk returns a tuple with the CustomTimeseriesPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetCustomTimeseriesPercentageOk() (*float64, bool) {
	if o == nil || o.CustomTimeseriesPercentage == nil {
		return nil, false
	}
	return o.CustomTimeseriesPercentage, true
}

// HasCustomTimeseriesPercentage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasCustomTimeseriesPercentage() bool {
	if o != nil && o.CustomTimeseriesPercentage != nil {
		return true
	}

	return false
}

// SetCustomTimeseriesPercentage gets a reference to the given float64 and assigns it to the CustomTimeseriesPercentage field.
func (o *MonthlyUsageAttributionValues) SetCustomTimeseriesPercentage(v float64) {
	o.CustomTimeseriesPercentage = &v
}

// GetCustomTimeseriesUsage returns the CustomTimeseriesUsage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetCustomTimeseriesUsage() float64 {
	if o == nil || o.CustomTimeseriesUsage == nil {
		var ret float64
		return ret
	}
	return *o.CustomTimeseriesUsage
}

// GetCustomTimeseriesUsageOk returns a tuple with the CustomTimeseriesUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetCustomTimeseriesUsageOk() (*float64, bool) {
	if o == nil || o.CustomTimeseriesUsage == nil {
		return nil, false
	}
	return o.CustomTimeseriesUsage, true
}

// HasCustomTimeseriesUsage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasCustomTimeseriesUsage() bool {
	if o != nil && o.CustomTimeseriesUsage != nil {
		return true
	}

	return false
}

// SetCustomTimeseriesUsage gets a reference to the given float64 and assigns it to the CustomTimeseriesUsage field.
func (o *MonthlyUsageAttributionValues) SetCustomTimeseriesUsage(v float64) {
	o.CustomTimeseriesUsage = &v
}

// GetEstimatedIndexedLogsPercentage returns the EstimatedIndexedLogsPercentage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetEstimatedIndexedLogsPercentage() float64 {
	if o == nil || o.EstimatedIndexedLogsPercentage == nil {
		var ret float64
		return ret
	}
	return *o.EstimatedIndexedLogsPercentage
}

// GetEstimatedIndexedLogsPercentageOk returns a tuple with the EstimatedIndexedLogsPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetEstimatedIndexedLogsPercentageOk() (*float64, bool) {
	if o == nil || o.EstimatedIndexedLogsPercentage == nil {
		return nil, false
	}
	return o.EstimatedIndexedLogsPercentage, true
}

// HasEstimatedIndexedLogsPercentage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasEstimatedIndexedLogsPercentage() bool {
	if o != nil && o.EstimatedIndexedLogsPercentage != nil {
		return true
	}

	return false
}

// SetEstimatedIndexedLogsPercentage gets a reference to the given float64 and assigns it to the EstimatedIndexedLogsPercentage field.
func (o *MonthlyUsageAttributionValues) SetEstimatedIndexedLogsPercentage(v float64) {
	o.EstimatedIndexedLogsPercentage = &v
}

// GetEstimatedIndexedLogsUsage returns the EstimatedIndexedLogsUsage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetEstimatedIndexedLogsUsage() float64 {
	if o == nil || o.EstimatedIndexedLogsUsage == nil {
		var ret float64
		return ret
	}
	return *o.EstimatedIndexedLogsUsage
}

// GetEstimatedIndexedLogsUsageOk returns a tuple with the EstimatedIndexedLogsUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetEstimatedIndexedLogsUsageOk() (*float64, bool) {
	if o == nil || o.EstimatedIndexedLogsUsage == nil {
		return nil, false
	}
	return o.EstimatedIndexedLogsUsage, true
}

// HasEstimatedIndexedLogsUsage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasEstimatedIndexedLogsUsage() bool {
	if o != nil && o.EstimatedIndexedLogsUsage != nil {
		return true
	}

	return false
}

// SetEstimatedIndexedLogsUsage gets a reference to the given float64 and assigns it to the EstimatedIndexedLogsUsage field.
func (o *MonthlyUsageAttributionValues) SetEstimatedIndexedLogsUsage(v float64) {
	o.EstimatedIndexedLogsUsage = &v
}

// GetEstimatedIndexedSpansPercentage returns the EstimatedIndexedSpansPercentage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetEstimatedIndexedSpansPercentage() float64 {
	if o == nil || o.EstimatedIndexedSpansPercentage == nil {
		var ret float64
		return ret
	}
	return *o.EstimatedIndexedSpansPercentage
}

// GetEstimatedIndexedSpansPercentageOk returns a tuple with the EstimatedIndexedSpansPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetEstimatedIndexedSpansPercentageOk() (*float64, bool) {
	if o == nil || o.EstimatedIndexedSpansPercentage == nil {
		return nil, false
	}
	return o.EstimatedIndexedSpansPercentage, true
}

// HasEstimatedIndexedSpansPercentage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasEstimatedIndexedSpansPercentage() bool {
	if o != nil && o.EstimatedIndexedSpansPercentage != nil {
		return true
	}

	return false
}

// SetEstimatedIndexedSpansPercentage gets a reference to the given float64 and assigns it to the EstimatedIndexedSpansPercentage field.
func (o *MonthlyUsageAttributionValues) SetEstimatedIndexedSpansPercentage(v float64) {
	o.EstimatedIndexedSpansPercentage = &v
}

// GetEstimatedIndexedSpansUsage returns the EstimatedIndexedSpansUsage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetEstimatedIndexedSpansUsage() float64 {
	if o == nil || o.EstimatedIndexedSpansUsage == nil {
		var ret float64
		return ret
	}
	return *o.EstimatedIndexedSpansUsage
}

// GetEstimatedIndexedSpansUsageOk returns a tuple with the EstimatedIndexedSpansUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetEstimatedIndexedSpansUsageOk() (*float64, bool) {
	if o == nil || o.EstimatedIndexedSpansUsage == nil {
		return nil, false
	}
	return o.EstimatedIndexedSpansUsage, true
}

// HasEstimatedIndexedSpansUsage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasEstimatedIndexedSpansUsage() bool {
	if o != nil && o.EstimatedIndexedSpansUsage != nil {
		return true
	}

	return false
}

// SetEstimatedIndexedSpansUsage gets a reference to the given float64 and assigns it to the EstimatedIndexedSpansUsage field.
func (o *MonthlyUsageAttributionValues) SetEstimatedIndexedSpansUsage(v float64) {
	o.EstimatedIndexedSpansUsage = &v
}

// GetEstimatedIngestedSpansPercentage returns the EstimatedIngestedSpansPercentage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetEstimatedIngestedSpansPercentage() float64 {
	if o == nil || o.EstimatedIngestedSpansPercentage == nil {
		var ret float64
		return ret
	}
	return *o.EstimatedIngestedSpansPercentage
}

// GetEstimatedIngestedSpansPercentageOk returns a tuple with the EstimatedIngestedSpansPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetEstimatedIngestedSpansPercentageOk() (*float64, bool) {
	if o == nil || o.EstimatedIngestedSpansPercentage == nil {
		return nil, false
	}
	return o.EstimatedIngestedSpansPercentage, true
}

// HasEstimatedIngestedSpansPercentage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasEstimatedIngestedSpansPercentage() bool {
	if o != nil && o.EstimatedIngestedSpansPercentage != nil {
		return true
	}

	return false
}

// SetEstimatedIngestedSpansPercentage gets a reference to the given float64 and assigns it to the EstimatedIngestedSpansPercentage field.
func (o *MonthlyUsageAttributionValues) SetEstimatedIngestedSpansPercentage(v float64) {
	o.EstimatedIngestedSpansPercentage = &v
}

// GetEstimatedIngestedSpansUsage returns the EstimatedIngestedSpansUsage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetEstimatedIngestedSpansUsage() float64 {
	if o == nil || o.EstimatedIngestedSpansUsage == nil {
		var ret float64
		return ret
	}
	return *o.EstimatedIngestedSpansUsage
}

// GetEstimatedIngestedSpansUsageOk returns a tuple with the EstimatedIngestedSpansUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetEstimatedIngestedSpansUsageOk() (*float64, bool) {
	if o == nil || o.EstimatedIngestedSpansUsage == nil {
		return nil, false
	}
	return o.EstimatedIngestedSpansUsage, true
}

// HasEstimatedIngestedSpansUsage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasEstimatedIngestedSpansUsage() bool {
	if o != nil && o.EstimatedIngestedSpansUsage != nil {
		return true
	}

	return false
}

// SetEstimatedIngestedSpansUsage gets a reference to the given float64 and assigns it to the EstimatedIngestedSpansUsage field.
func (o *MonthlyUsageAttributionValues) SetEstimatedIngestedSpansUsage(v float64) {
	o.EstimatedIngestedSpansUsage = &v
}

// GetFargatePercentage returns the FargatePercentage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetFargatePercentage() float64 {
	if o == nil || o.FargatePercentage == nil {
		var ret float64
		return ret
	}
	return *o.FargatePercentage
}

// GetFargatePercentageOk returns a tuple with the FargatePercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetFargatePercentageOk() (*float64, bool) {
	if o == nil || o.FargatePercentage == nil {
		return nil, false
	}
	return o.FargatePercentage, true
}

// HasFargatePercentage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasFargatePercentage() bool {
	if o != nil && o.FargatePercentage != nil {
		return true
	}

	return false
}

// SetFargatePercentage gets a reference to the given float64 and assigns it to the FargatePercentage field.
func (o *MonthlyUsageAttributionValues) SetFargatePercentage(v float64) {
	o.FargatePercentage = &v
}

// GetFargateUsage returns the FargateUsage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetFargateUsage() float64 {
	if o == nil || o.FargateUsage == nil {
		var ret float64
		return ret
	}
	return *o.FargateUsage
}

// GetFargateUsageOk returns a tuple with the FargateUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetFargateUsageOk() (*float64, bool) {
	if o == nil || o.FargateUsage == nil {
		return nil, false
	}
	return o.FargateUsage, true
}

// HasFargateUsage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasFargateUsage() bool {
	if o != nil && o.FargateUsage != nil {
		return true
	}

	return false
}

// SetFargateUsage gets a reference to the given float64 and assigns it to the FargateUsage field.
func (o *MonthlyUsageAttributionValues) SetFargateUsage(v float64) {
	o.FargateUsage = &v
}

// GetFunctionsPercentage returns the FunctionsPercentage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetFunctionsPercentage() float64 {
	if o == nil || o.FunctionsPercentage == nil {
		var ret float64
		return ret
	}
	return *o.FunctionsPercentage
}

// GetFunctionsPercentageOk returns a tuple with the FunctionsPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetFunctionsPercentageOk() (*float64, bool) {
	if o == nil || o.FunctionsPercentage == nil {
		return nil, false
	}
	return o.FunctionsPercentage, true
}

// HasFunctionsPercentage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasFunctionsPercentage() bool {
	if o != nil && o.FunctionsPercentage != nil {
		return true
	}

	return false
}

// SetFunctionsPercentage gets a reference to the given float64 and assigns it to the FunctionsPercentage field.
func (o *MonthlyUsageAttributionValues) SetFunctionsPercentage(v float64) {
	o.FunctionsPercentage = &v
}

// GetFunctionsUsage returns the FunctionsUsage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetFunctionsUsage() float64 {
	if o == nil || o.FunctionsUsage == nil {
		var ret float64
		return ret
	}
	return *o.FunctionsUsage
}

// GetFunctionsUsageOk returns a tuple with the FunctionsUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetFunctionsUsageOk() (*float64, bool) {
	if o == nil || o.FunctionsUsage == nil {
		return nil, false
	}
	return o.FunctionsUsage, true
}

// HasFunctionsUsage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasFunctionsUsage() bool {
	if o != nil && o.FunctionsUsage != nil {
		return true
	}

	return false
}

// SetFunctionsUsage gets a reference to the given float64 and assigns it to the FunctionsUsage field.
func (o *MonthlyUsageAttributionValues) SetFunctionsUsage(v float64) {
	o.FunctionsUsage = &v
}

// GetIndexedLogsPercentage returns the IndexedLogsPercentage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetIndexedLogsPercentage() float64 {
	if o == nil || o.IndexedLogsPercentage == nil {
		var ret float64
		return ret
	}
	return *o.IndexedLogsPercentage
}

// GetIndexedLogsPercentageOk returns a tuple with the IndexedLogsPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetIndexedLogsPercentageOk() (*float64, bool) {
	if o == nil || o.IndexedLogsPercentage == nil {
		return nil, false
	}
	return o.IndexedLogsPercentage, true
}

// HasIndexedLogsPercentage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasIndexedLogsPercentage() bool {
	if o != nil && o.IndexedLogsPercentage != nil {
		return true
	}

	return false
}

// SetIndexedLogsPercentage gets a reference to the given float64 and assigns it to the IndexedLogsPercentage field.
func (o *MonthlyUsageAttributionValues) SetIndexedLogsPercentage(v float64) {
	o.IndexedLogsPercentage = &v
}

// GetIndexedLogsUsage returns the IndexedLogsUsage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetIndexedLogsUsage() float64 {
	if o == nil || o.IndexedLogsUsage == nil {
		var ret float64
		return ret
	}
	return *o.IndexedLogsUsage
}

// GetIndexedLogsUsageOk returns a tuple with the IndexedLogsUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetIndexedLogsUsageOk() (*float64, bool) {
	if o == nil || o.IndexedLogsUsage == nil {
		return nil, false
	}
	return o.IndexedLogsUsage, true
}

// HasIndexedLogsUsage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasIndexedLogsUsage() bool {
	if o != nil && o.IndexedLogsUsage != nil {
		return true
	}

	return false
}

// SetIndexedLogsUsage gets a reference to the given float64 and assigns it to the IndexedLogsUsage field.
func (o *MonthlyUsageAttributionValues) SetIndexedLogsUsage(v float64) {
	o.IndexedLogsUsage = &v
}

// GetInfraHostPercentage returns the InfraHostPercentage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetInfraHostPercentage() float64 {
	if o == nil || o.InfraHostPercentage == nil {
		var ret float64
		return ret
	}
	return *o.InfraHostPercentage
}

// GetInfraHostPercentageOk returns a tuple with the InfraHostPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetInfraHostPercentageOk() (*float64, bool) {
	if o == nil || o.InfraHostPercentage == nil {
		return nil, false
	}
	return o.InfraHostPercentage, true
}

// HasInfraHostPercentage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasInfraHostPercentage() bool {
	if o != nil && o.InfraHostPercentage != nil {
		return true
	}

	return false
}

// SetInfraHostPercentage gets a reference to the given float64 and assigns it to the InfraHostPercentage field.
func (o *MonthlyUsageAttributionValues) SetInfraHostPercentage(v float64) {
	o.InfraHostPercentage = &v
}

// GetInfraHostUsage returns the InfraHostUsage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetInfraHostUsage() float64 {
	if o == nil || o.InfraHostUsage == nil {
		var ret float64
		return ret
	}
	return *o.InfraHostUsage
}

// GetInfraHostUsageOk returns a tuple with the InfraHostUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetInfraHostUsageOk() (*float64, bool) {
	if o == nil || o.InfraHostUsage == nil {
		return nil, false
	}
	return o.InfraHostUsage, true
}

// HasInfraHostUsage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasInfraHostUsage() bool {
	if o != nil && o.InfraHostUsage != nil {
		return true
	}

	return false
}

// SetInfraHostUsage gets a reference to the given float64 and assigns it to the InfraHostUsage field.
func (o *MonthlyUsageAttributionValues) SetInfraHostUsage(v float64) {
	o.InfraHostUsage = &v
}

// GetInvocationsPercentage returns the InvocationsPercentage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetInvocationsPercentage() float64 {
	if o == nil || o.InvocationsPercentage == nil {
		var ret float64
		return ret
	}
	return *o.InvocationsPercentage
}

// GetInvocationsPercentageOk returns a tuple with the InvocationsPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetInvocationsPercentageOk() (*float64, bool) {
	if o == nil || o.InvocationsPercentage == nil {
		return nil, false
	}
	return o.InvocationsPercentage, true
}

// HasInvocationsPercentage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasInvocationsPercentage() bool {
	if o != nil && o.InvocationsPercentage != nil {
		return true
	}

	return false
}

// SetInvocationsPercentage gets a reference to the given float64 and assigns it to the InvocationsPercentage field.
func (o *MonthlyUsageAttributionValues) SetInvocationsPercentage(v float64) {
	o.InvocationsPercentage = &v
}

// GetInvocationsUsage returns the InvocationsUsage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetInvocationsUsage() float64 {
	if o == nil || o.InvocationsUsage == nil {
		var ret float64
		return ret
	}
	return *o.InvocationsUsage
}

// GetInvocationsUsageOk returns a tuple with the InvocationsUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetInvocationsUsageOk() (*float64, bool) {
	if o == nil || o.InvocationsUsage == nil {
		return nil, false
	}
	return o.InvocationsUsage, true
}

// HasInvocationsUsage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasInvocationsUsage() bool {
	if o != nil && o.InvocationsUsage != nil {
		return true
	}

	return false
}

// SetInvocationsUsage gets a reference to the given float64 and assigns it to the InvocationsUsage field.
func (o *MonthlyUsageAttributionValues) SetInvocationsUsage(v float64) {
	o.InvocationsUsage = &v
}

// GetNpmHostPercentage returns the NpmHostPercentage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetNpmHostPercentage() float64 {
	if o == nil || o.NpmHostPercentage == nil {
		var ret float64
		return ret
	}
	return *o.NpmHostPercentage
}

// GetNpmHostPercentageOk returns a tuple with the NpmHostPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetNpmHostPercentageOk() (*float64, bool) {
	if o == nil || o.NpmHostPercentage == nil {
		return nil, false
	}
	return o.NpmHostPercentage, true
}

// HasNpmHostPercentage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasNpmHostPercentage() bool {
	if o != nil && o.NpmHostPercentage != nil {
		return true
	}

	return false
}

// SetNpmHostPercentage gets a reference to the given float64 and assigns it to the NpmHostPercentage field.
func (o *MonthlyUsageAttributionValues) SetNpmHostPercentage(v float64) {
	o.NpmHostPercentage = &v
}

// GetNpmHostUsage returns the NpmHostUsage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetNpmHostUsage() float64 {
	if o == nil || o.NpmHostUsage == nil {
		var ret float64
		return ret
	}
	return *o.NpmHostUsage
}

// GetNpmHostUsageOk returns a tuple with the NpmHostUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetNpmHostUsageOk() (*float64, bool) {
	if o == nil || o.NpmHostUsage == nil {
		return nil, false
	}
	return o.NpmHostUsage, true
}

// HasNpmHostUsage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasNpmHostUsage() bool {
	if o != nil && o.NpmHostUsage != nil {
		return true
	}

	return false
}

// SetNpmHostUsage gets a reference to the given float64 and assigns it to the NpmHostUsage field.
func (o *MonthlyUsageAttributionValues) SetNpmHostUsage(v float64) {
	o.NpmHostUsage = &v
}

// GetProfiledContainerPercentage returns the ProfiledContainerPercentage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetProfiledContainerPercentage() float64 {
	if o == nil || o.ProfiledContainerPercentage == nil {
		var ret float64
		return ret
	}
	return *o.ProfiledContainerPercentage
}

// GetProfiledContainerPercentageOk returns a tuple with the ProfiledContainerPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetProfiledContainerPercentageOk() (*float64, bool) {
	if o == nil || o.ProfiledContainerPercentage == nil {
		return nil, false
	}
	return o.ProfiledContainerPercentage, true
}

// HasProfiledContainerPercentage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasProfiledContainerPercentage() bool {
	if o != nil && o.ProfiledContainerPercentage != nil {
		return true
	}

	return false
}

// SetProfiledContainerPercentage gets a reference to the given float64 and assigns it to the ProfiledContainerPercentage field.
func (o *MonthlyUsageAttributionValues) SetProfiledContainerPercentage(v float64) {
	o.ProfiledContainerPercentage = &v
}

// GetProfiledContainerUsage returns the ProfiledContainerUsage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetProfiledContainerUsage() float64 {
	if o == nil || o.ProfiledContainerUsage == nil {
		var ret float64
		return ret
	}
	return *o.ProfiledContainerUsage
}

// GetProfiledContainerUsageOk returns a tuple with the ProfiledContainerUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetProfiledContainerUsageOk() (*float64, bool) {
	if o == nil || o.ProfiledContainerUsage == nil {
		return nil, false
	}
	return o.ProfiledContainerUsage, true
}

// HasProfiledContainerUsage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasProfiledContainerUsage() bool {
	if o != nil && o.ProfiledContainerUsage != nil {
		return true
	}

	return false
}

// SetProfiledContainerUsage gets a reference to the given float64 and assigns it to the ProfiledContainerUsage field.
func (o *MonthlyUsageAttributionValues) SetProfiledContainerUsage(v float64) {
	o.ProfiledContainerUsage = &v
}

// GetProfiledHostPercentage returns the ProfiledHostPercentage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetProfiledHostPercentage() float64 {
	if o == nil || o.ProfiledHostPercentage == nil {
		var ret float64
		return ret
	}
	return *o.ProfiledHostPercentage
}

// GetProfiledHostPercentageOk returns a tuple with the ProfiledHostPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetProfiledHostPercentageOk() (*float64, bool) {
	if o == nil || o.ProfiledHostPercentage == nil {
		return nil, false
	}
	return o.ProfiledHostPercentage, true
}

// HasProfiledHostPercentage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasProfiledHostPercentage() bool {
	if o != nil && o.ProfiledHostPercentage != nil {
		return true
	}

	return false
}

// SetProfiledHostPercentage gets a reference to the given float64 and assigns it to the ProfiledHostPercentage field.
func (o *MonthlyUsageAttributionValues) SetProfiledHostPercentage(v float64) {
	o.ProfiledHostPercentage = &v
}

// GetProfiledHostUsage returns the ProfiledHostUsage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetProfiledHostUsage() float64 {
	if o == nil || o.ProfiledHostUsage == nil {
		var ret float64
		return ret
	}
	return *o.ProfiledHostUsage
}

// GetProfiledHostUsageOk returns a tuple with the ProfiledHostUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetProfiledHostUsageOk() (*float64, bool) {
	if o == nil || o.ProfiledHostUsage == nil {
		return nil, false
	}
	return o.ProfiledHostUsage, true
}

// HasProfiledHostUsage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasProfiledHostUsage() bool {
	if o != nil && o.ProfiledHostUsage != nil {
		return true
	}

	return false
}

// SetProfiledHostUsage gets a reference to the given float64 and assigns it to the ProfiledHostUsage field.
func (o *MonthlyUsageAttributionValues) SetProfiledHostUsage(v float64) {
	o.ProfiledHostUsage = &v
}

// GetSnmpPercentage returns the SnmpPercentage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetSnmpPercentage() float64 {
	if o == nil || o.SnmpPercentage == nil {
		var ret float64
		return ret
	}
	return *o.SnmpPercentage
}

// GetSnmpPercentageOk returns a tuple with the SnmpPercentage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetSnmpPercentageOk() (*float64, bool) {
	if o == nil || o.SnmpPercentage == nil {
		return nil, false
	}
	return o.SnmpPercentage, true
}

// HasSnmpPercentage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasSnmpPercentage() bool {
	if o != nil && o.SnmpPercentage != nil {
		return true
	}

	return false
}

// SetSnmpPercentage gets a reference to the given float64 and assigns it to the SnmpPercentage field.
func (o *MonthlyUsageAttributionValues) SetSnmpPercentage(v float64) {
	o.SnmpPercentage = &v
}

// GetSnmpUsage returns the SnmpUsage field value if set, zero value otherwise.
func (o *MonthlyUsageAttributionValues) GetSnmpUsage() float64 {
	if o == nil || o.SnmpUsage == nil {
		var ret float64
		return ret
	}
	return *o.SnmpUsage
}

// GetSnmpUsageOk returns a tuple with the SnmpUsage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonthlyUsageAttributionValues) GetSnmpUsageOk() (*float64, bool) {
	if o == nil || o.SnmpUsage == nil {
		return nil, false
	}
	return o.SnmpUsage, true
}

// HasSnmpUsage returns a boolean if a field has been set.
func (o *MonthlyUsageAttributionValues) HasSnmpUsage() bool {
	if o != nil && o.SnmpUsage != nil {
		return true
	}

	return false
}

// SetSnmpUsage gets a reference to the given float64 and assigns it to the SnmpUsage field.
func (o *MonthlyUsageAttributionValues) SetSnmpUsage(v float64) {
	o.SnmpUsage = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o MonthlyUsageAttributionValues) MarshalJSON() ([]byte, error) {
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
	if o.CustomTimeseriesPercentage != nil {
		toSerialize["custom_timeseries_percentage"] = o.CustomTimeseriesPercentage
	}
	if o.CustomTimeseriesUsage != nil {
		toSerialize["custom_timeseries_usage"] = o.CustomTimeseriesUsage
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
	if o.FargatePercentage != nil {
		toSerialize["fargate_percentage"] = o.FargatePercentage
	}
	if o.FargateUsage != nil {
		toSerialize["fargate_usage"] = o.FargateUsage
	}
	if o.FunctionsPercentage != nil {
		toSerialize["functions_percentage"] = o.FunctionsPercentage
	}
	if o.FunctionsUsage != nil {
		toSerialize["functions_usage"] = o.FunctionsUsage
	}
	if o.IndexedLogsPercentage != nil {
		toSerialize["indexed_logs_percentage"] = o.IndexedLogsPercentage
	}
	if o.IndexedLogsUsage != nil {
		toSerialize["indexed_logs_usage"] = o.IndexedLogsUsage
	}
	if o.InfraHostPercentage != nil {
		toSerialize["infra_host_percentage"] = o.InfraHostPercentage
	}
	if o.InfraHostUsage != nil {
		toSerialize["infra_host_usage"] = o.InfraHostUsage
	}
	if o.InvocationsPercentage != nil {
		toSerialize["invocations_percentage"] = o.InvocationsPercentage
	}
	if o.InvocationsUsage != nil {
		toSerialize["invocations_usage"] = o.InvocationsUsage
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
	if o.ProfiledHostPercentage != nil {
		toSerialize["profiled_host_percentage"] = o.ProfiledHostPercentage
	}
	if o.ProfiledHostUsage != nil {
		toSerialize["profiled_host_usage"] = o.ProfiledHostUsage
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
func (o *MonthlyUsageAttributionValues) UnmarshalJSON(bytes []byte) (err error) {
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
		CustomTimeseriesPercentage       *float64 `json:"custom_timeseries_percentage,omitempty"`
		CustomTimeseriesUsage            *float64 `json:"custom_timeseries_usage,omitempty"`
		EstimatedIndexedLogsPercentage   *float64 `json:"estimated_indexed_logs_percentage,omitempty"`
		EstimatedIndexedLogsUsage        *float64 `json:"estimated_indexed_logs_usage,omitempty"`
		EstimatedIndexedSpansPercentage  *float64 `json:"estimated_indexed_spans_percentage,omitempty"`
		EstimatedIndexedSpansUsage       *float64 `json:"estimated_indexed_spans_usage,omitempty"`
		EstimatedIngestedSpansPercentage *float64 `json:"estimated_ingested_spans_percentage,omitempty"`
		EstimatedIngestedSpansUsage      *float64 `json:"estimated_ingested_spans_usage,omitempty"`
		FargatePercentage                *float64 `json:"fargate_percentage,omitempty"`
		FargateUsage                     *float64 `json:"fargate_usage,omitempty"`
		FunctionsPercentage              *float64 `json:"functions_percentage,omitempty"`
		FunctionsUsage                   *float64 `json:"functions_usage,omitempty"`
		IndexedLogsPercentage            *float64 `json:"indexed_logs_percentage,omitempty"`
		IndexedLogsUsage                 *float64 `json:"indexed_logs_usage,omitempty"`
		InfraHostPercentage              *float64 `json:"infra_host_percentage,omitempty"`
		InfraHostUsage                   *float64 `json:"infra_host_usage,omitempty"`
		InvocationsPercentage            *float64 `json:"invocations_percentage,omitempty"`
		InvocationsUsage                 *float64 `json:"invocations_usage,omitempty"`
		NpmHostPercentage                *float64 `json:"npm_host_percentage,omitempty"`
		NpmHostUsage                     *float64 `json:"npm_host_usage,omitempty"`
		ProfiledContainerPercentage      *float64 `json:"profiled_container_percentage,omitempty"`
		ProfiledContainerUsage           *float64 `json:"profiled_container_usage,omitempty"`
		ProfiledHostPercentage           *float64 `json:"profiled_host_percentage,omitempty"`
		ProfiledHostUsage                *float64 `json:"profiled_host_usage,omitempty"`
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
	o.CustomTimeseriesPercentage = all.CustomTimeseriesPercentage
	o.CustomTimeseriesUsage = all.CustomTimeseriesUsage
	o.EstimatedIndexedLogsPercentage = all.EstimatedIndexedLogsPercentage
	o.EstimatedIndexedLogsUsage = all.EstimatedIndexedLogsUsage
	o.EstimatedIndexedSpansPercentage = all.EstimatedIndexedSpansPercentage
	o.EstimatedIndexedSpansUsage = all.EstimatedIndexedSpansUsage
	o.EstimatedIngestedSpansPercentage = all.EstimatedIngestedSpansPercentage
	o.EstimatedIngestedSpansUsage = all.EstimatedIngestedSpansUsage
	o.FargatePercentage = all.FargatePercentage
	o.FargateUsage = all.FargateUsage
	o.FunctionsPercentage = all.FunctionsPercentage
	o.FunctionsUsage = all.FunctionsUsage
	o.IndexedLogsPercentage = all.IndexedLogsPercentage
	o.IndexedLogsUsage = all.IndexedLogsUsage
	o.InfraHostPercentage = all.InfraHostPercentage
	o.InfraHostUsage = all.InfraHostUsage
	o.InvocationsPercentage = all.InvocationsPercentage
	o.InvocationsUsage = all.InvocationsUsage
	o.NpmHostPercentage = all.NpmHostPercentage
	o.NpmHostUsage = all.NpmHostUsage
	o.ProfiledContainerPercentage = all.ProfiledContainerPercentage
	o.ProfiledContainerUsage = all.ProfiledContainerUsage
	o.ProfiledHostPercentage = all.ProfiledHostPercentage
	o.ProfiledHostUsage = all.ProfiledHostUsage
	o.SnmpPercentage = all.SnmpPercentage
	o.SnmpUsage = all.SnmpUsage
	return nil
}
