// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// UsageBillableSummaryKeys Response with aggregated usage types.
type UsageBillableSummaryKeys struct {
	// Response with properties for each aggregated usage type.
	ApmFargateAverage *UsageBillableSummaryBody `json:"apm_fargate_average,omitempty"`
	// Response with properties for each aggregated usage type.
	ApmFargateSum *UsageBillableSummaryBody `json:"apm_fargate_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	ApmHostSum *UsageBillableSummaryBody `json:"apm_host_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	ApmHostTop99p *UsageBillableSummaryBody `json:"apm_host_top99p,omitempty"`
	// Response with properties for each aggregated usage type.
	ApmProfilerHostSum *UsageBillableSummaryBody `json:"apm_profiler_host_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	ApmProfilerHostTop99p *UsageBillableSummaryBody `json:"apm_profiler_host_top99p,omitempty"`
	// Response with properties for each aggregated usage type.
	ApmTraceSearchSum *UsageBillableSummaryBody `json:"apm_trace_search_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	ApplicationSecurityHostSum *UsageBillableSummaryBody `json:"application_security_host_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	CiPipelineIndexedSpansSum *UsageBillableSummaryBody `json:"ci_pipeline_indexed_spans_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	CiPipelineMaximum *UsageBillableSummaryBody `json:"ci_pipeline_maximum,omitempty"`
	// Response with properties for each aggregated usage type.
	CiPipelineSum *UsageBillableSummaryBody `json:"ci_pipeline_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	CiTestIndexedSpansSum *UsageBillableSummaryBody `json:"ci_test_indexed_spans_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	CiTestingMaximum *UsageBillableSummaryBody `json:"ci_testing_maximum,omitempty"`
	// Response with properties for each aggregated usage type.
	CiTestingSum *UsageBillableSummaryBody `json:"ci_testing_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	CspmContainerSum *UsageBillableSummaryBody `json:"cspm_container_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	CspmHostSum *UsageBillableSummaryBody `json:"cspm_host_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	CspmHostTop99p *UsageBillableSummaryBody `json:"cspm_host_top99p,omitempty"`
	// Response with properties for each aggregated usage type.
	CustomEventSum *UsageBillableSummaryBody `json:"custom_event_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	CwsContainerSum *UsageBillableSummaryBody `json:"cws_container_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	CwsHostSum *UsageBillableSummaryBody `json:"cws_host_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	CwsHostTop99p *UsageBillableSummaryBody `json:"cws_host_top99p,omitempty"`
	// Response with properties for each aggregated usage type.
	DbmHostSum *UsageBillableSummaryBody `json:"dbm_host_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	DbmHostTop99p *UsageBillableSummaryBody `json:"dbm_host_top99p,omitempty"`
	// Response with properties for each aggregated usage type.
	DbmNormalizedQueriesAverage *UsageBillableSummaryBody `json:"dbm_normalized_queries_average,omitempty"`
	// Response with properties for each aggregated usage type.
	DbmNormalizedQueriesSum *UsageBillableSummaryBody `json:"dbm_normalized_queries_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	FargateContainerApmAndProfilerAverage *UsageBillableSummaryBody `json:"fargate_container_apm_and_profiler_average,omitempty"`
	// Response with properties for each aggregated usage type.
	FargateContainerApmAndProfilerSum *UsageBillableSummaryBody `json:"fargate_container_apm_and_profiler_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	FargateContainerAverage *UsageBillableSummaryBody `json:"fargate_container_average,omitempty"`
	// Response with properties for each aggregated usage type.
	FargateContainerProfilerAverage *UsageBillableSummaryBody `json:"fargate_container_profiler_average,omitempty"`
	// Response with properties for each aggregated usage type.
	FargateContainerProfilerSum *UsageBillableSummaryBody `json:"fargate_container_profiler_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	FargateContainerSum *UsageBillableSummaryBody `json:"fargate_container_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	IncidentManagementMaximum *UsageBillableSummaryBody `json:"incident_management_maximum,omitempty"`
	// Response with properties for each aggregated usage type.
	IncidentManagementSum *UsageBillableSummaryBody `json:"incident_management_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	InfraAndApmHostSum *UsageBillableSummaryBody `json:"infra_and_apm_host_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	InfraAndApmHostTop99p *UsageBillableSummaryBody `json:"infra_and_apm_host_top99p,omitempty"`
	// Response with properties for each aggregated usage type.
	InfraContainerSum *UsageBillableSummaryBody `json:"infra_container_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	InfraHostSum *UsageBillableSummaryBody `json:"infra_host_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	InfraHostTop99p *UsageBillableSummaryBody `json:"infra_host_top99p,omitempty"`
	// Response with properties for each aggregated usage type.
	IngestedSpansSum *UsageBillableSummaryBody `json:"ingested_spans_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	IngestedTimeseriesAverage *UsageBillableSummaryBody `json:"ingested_timeseries_average,omitempty"`
	// Response with properties for each aggregated usage type.
	IngestedTimeseriesSum *UsageBillableSummaryBody `json:"ingested_timeseries_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	IotSum *UsageBillableSummaryBody `json:"iot_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	IotTop99p *UsageBillableSummaryBody `json:"iot_top99p,omitempty"`
	// Response with properties for each aggregated usage type.
	LambdaFunctionAverage *UsageBillableSummaryBody `json:"lambda_function_average,omitempty"`
	// Response with properties for each aggregated usage type.
	LambdaFunctionSum *UsageBillableSummaryBody `json:"lambda_function_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	LogsIndexed15daySum *UsageBillableSummaryBody `json:"logs_indexed_15day_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	LogsIndexed180daySum *UsageBillableSummaryBody `json:"logs_indexed_180day_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	LogsIndexed30daySum *UsageBillableSummaryBody `json:"logs_indexed_30day_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	LogsIndexed360daySum *UsageBillableSummaryBody `json:"logs_indexed_360day_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	LogsIndexed3daySum *UsageBillableSummaryBody `json:"logs_indexed_3day_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	LogsIndexed45daySum *UsageBillableSummaryBody `json:"logs_indexed_45day_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	LogsIndexed60daySum *UsageBillableSummaryBody `json:"logs_indexed_60day_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	LogsIndexed7daySum *UsageBillableSummaryBody `json:"logs_indexed_7day_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	LogsIndexed90daySum *UsageBillableSummaryBody `json:"logs_indexed_90day_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	LogsIndexedCustomRetentionSum *UsageBillableSummaryBody `json:"logs_indexed_custom_retention_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	LogsIndexedSum *UsageBillableSummaryBody `json:"logs_indexed_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	LogsIngestedSum *UsageBillableSummaryBody `json:"logs_ingested_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	NetworkDeviceSum *UsageBillableSummaryBody `json:"network_device_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	NetworkDeviceTop99p *UsageBillableSummaryBody `json:"network_device_top99p,omitempty"`
	// Response with properties for each aggregated usage type.
	NpmFlowSum *UsageBillableSummaryBody `json:"npm_flow_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	NpmHostSum *UsageBillableSummaryBody `json:"npm_host_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	NpmHostTop99p *UsageBillableSummaryBody `json:"npm_host_top99p,omitempty"`
	// Response with properties for each aggregated usage type.
	ObservabilityPipelineSum *UsageBillableSummaryBody `json:"observability_pipeline_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	OnlineArchiveSum *UsageBillableSummaryBody `json:"online_archive_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	ProfContainerSum *UsageBillableSummaryBody `json:"prof_container_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	ProfHostSum *UsageBillableSummaryBody `json:"prof_host_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	ProfHostTop99p *UsageBillableSummaryBody `json:"prof_host_top99p,omitempty"`
	// Response with properties for each aggregated usage type.
	RumLiteSum *UsageBillableSummaryBody `json:"rum_lite_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	RumReplaySum *UsageBillableSummaryBody `json:"rum_replay_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	RumSum *UsageBillableSummaryBody `json:"rum_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	RumUnitsSum *UsageBillableSummaryBody `json:"rum_units_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	SensitiveDataScannerSum *UsageBillableSummaryBody `json:"sensitive_data_scanner_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	ServerlessInvocationSum *UsageBillableSummaryBody `json:"serverless_invocation_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	SiemSum *UsageBillableSummaryBody `json:"siem_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	StandardTimeseriesAverage *UsageBillableSummaryBody `json:"standard_timeseries_average,omitempty"`
	// Response with properties for each aggregated usage type.
	SyntheticsApiTestsSum *UsageBillableSummaryBody `json:"synthetics_api_tests_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	SyntheticsBrowserChecksSum *UsageBillableSummaryBody `json:"synthetics_browser_checks_sum,omitempty"`
	// Response with properties for each aggregated usage type.
	TimeseriesAverage *UsageBillableSummaryBody `json:"timeseries_average,omitempty"`
	// Response with properties for each aggregated usage type.
	TimeseriesSum *UsageBillableSummaryBody `json:"timeseries_sum,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageBillableSummaryKeys instantiates a new UsageBillableSummaryKeys object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageBillableSummaryKeys() *UsageBillableSummaryKeys {
	this := UsageBillableSummaryKeys{}
	return &this
}

// NewUsageBillableSummaryKeysWithDefaults instantiates a new UsageBillableSummaryKeys object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageBillableSummaryKeysWithDefaults() *UsageBillableSummaryKeys {
	this := UsageBillableSummaryKeys{}
	return &this
}

// GetApmFargateAverage returns the ApmFargateAverage field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetApmFargateAverage() UsageBillableSummaryBody {
	if o == nil || o.ApmFargateAverage == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.ApmFargateAverage
}

// GetApmFargateAverageOk returns a tuple with the ApmFargateAverage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetApmFargateAverageOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.ApmFargateAverage == nil {
		return nil, false
	}
	return o.ApmFargateAverage, true
}

// HasApmFargateAverage returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasApmFargateAverage() bool {
	if o != nil && o.ApmFargateAverage != nil {
		return true
	}

	return false
}

// SetApmFargateAverage gets a reference to the given UsageBillableSummaryBody and assigns it to the ApmFargateAverage field.
func (o *UsageBillableSummaryKeys) SetApmFargateAverage(v UsageBillableSummaryBody) {
	o.ApmFargateAverage = &v
}

// GetApmFargateSum returns the ApmFargateSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetApmFargateSum() UsageBillableSummaryBody {
	if o == nil || o.ApmFargateSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.ApmFargateSum
}

// GetApmFargateSumOk returns a tuple with the ApmFargateSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetApmFargateSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.ApmFargateSum == nil {
		return nil, false
	}
	return o.ApmFargateSum, true
}

// HasApmFargateSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasApmFargateSum() bool {
	if o != nil && o.ApmFargateSum != nil {
		return true
	}

	return false
}

// SetApmFargateSum gets a reference to the given UsageBillableSummaryBody and assigns it to the ApmFargateSum field.
func (o *UsageBillableSummaryKeys) SetApmFargateSum(v UsageBillableSummaryBody) {
	o.ApmFargateSum = &v
}

// GetApmHostSum returns the ApmHostSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetApmHostSum() UsageBillableSummaryBody {
	if o == nil || o.ApmHostSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.ApmHostSum
}

// GetApmHostSumOk returns a tuple with the ApmHostSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetApmHostSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.ApmHostSum == nil {
		return nil, false
	}
	return o.ApmHostSum, true
}

// HasApmHostSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasApmHostSum() bool {
	if o != nil && o.ApmHostSum != nil {
		return true
	}

	return false
}

// SetApmHostSum gets a reference to the given UsageBillableSummaryBody and assigns it to the ApmHostSum field.
func (o *UsageBillableSummaryKeys) SetApmHostSum(v UsageBillableSummaryBody) {
	o.ApmHostSum = &v
}

// GetApmHostTop99p returns the ApmHostTop99p field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetApmHostTop99p() UsageBillableSummaryBody {
	if o == nil || o.ApmHostTop99p == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.ApmHostTop99p
}

// GetApmHostTop99pOk returns a tuple with the ApmHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetApmHostTop99pOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.ApmHostTop99p == nil {
		return nil, false
	}
	return o.ApmHostTop99p, true
}

// HasApmHostTop99p returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasApmHostTop99p() bool {
	if o != nil && o.ApmHostTop99p != nil {
		return true
	}

	return false
}

// SetApmHostTop99p gets a reference to the given UsageBillableSummaryBody and assigns it to the ApmHostTop99p field.
func (o *UsageBillableSummaryKeys) SetApmHostTop99p(v UsageBillableSummaryBody) {
	o.ApmHostTop99p = &v
}

// GetApmProfilerHostSum returns the ApmProfilerHostSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetApmProfilerHostSum() UsageBillableSummaryBody {
	if o == nil || o.ApmProfilerHostSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.ApmProfilerHostSum
}

// GetApmProfilerHostSumOk returns a tuple with the ApmProfilerHostSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetApmProfilerHostSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.ApmProfilerHostSum == nil {
		return nil, false
	}
	return o.ApmProfilerHostSum, true
}

// HasApmProfilerHostSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasApmProfilerHostSum() bool {
	if o != nil && o.ApmProfilerHostSum != nil {
		return true
	}

	return false
}

// SetApmProfilerHostSum gets a reference to the given UsageBillableSummaryBody and assigns it to the ApmProfilerHostSum field.
func (o *UsageBillableSummaryKeys) SetApmProfilerHostSum(v UsageBillableSummaryBody) {
	o.ApmProfilerHostSum = &v
}

// GetApmProfilerHostTop99p returns the ApmProfilerHostTop99p field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetApmProfilerHostTop99p() UsageBillableSummaryBody {
	if o == nil || o.ApmProfilerHostTop99p == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.ApmProfilerHostTop99p
}

// GetApmProfilerHostTop99pOk returns a tuple with the ApmProfilerHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetApmProfilerHostTop99pOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.ApmProfilerHostTop99p == nil {
		return nil, false
	}
	return o.ApmProfilerHostTop99p, true
}

// HasApmProfilerHostTop99p returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasApmProfilerHostTop99p() bool {
	if o != nil && o.ApmProfilerHostTop99p != nil {
		return true
	}

	return false
}

// SetApmProfilerHostTop99p gets a reference to the given UsageBillableSummaryBody and assigns it to the ApmProfilerHostTop99p field.
func (o *UsageBillableSummaryKeys) SetApmProfilerHostTop99p(v UsageBillableSummaryBody) {
	o.ApmProfilerHostTop99p = &v
}

// GetApmTraceSearchSum returns the ApmTraceSearchSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetApmTraceSearchSum() UsageBillableSummaryBody {
	if o == nil || o.ApmTraceSearchSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.ApmTraceSearchSum
}

// GetApmTraceSearchSumOk returns a tuple with the ApmTraceSearchSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetApmTraceSearchSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.ApmTraceSearchSum == nil {
		return nil, false
	}
	return o.ApmTraceSearchSum, true
}

// HasApmTraceSearchSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasApmTraceSearchSum() bool {
	if o != nil && o.ApmTraceSearchSum != nil {
		return true
	}

	return false
}

// SetApmTraceSearchSum gets a reference to the given UsageBillableSummaryBody and assigns it to the ApmTraceSearchSum field.
func (o *UsageBillableSummaryKeys) SetApmTraceSearchSum(v UsageBillableSummaryBody) {
	o.ApmTraceSearchSum = &v
}

// GetApplicationSecurityHostSum returns the ApplicationSecurityHostSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetApplicationSecurityHostSum() UsageBillableSummaryBody {
	if o == nil || o.ApplicationSecurityHostSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.ApplicationSecurityHostSum
}

// GetApplicationSecurityHostSumOk returns a tuple with the ApplicationSecurityHostSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetApplicationSecurityHostSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.ApplicationSecurityHostSum == nil {
		return nil, false
	}
	return o.ApplicationSecurityHostSum, true
}

// HasApplicationSecurityHostSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasApplicationSecurityHostSum() bool {
	if o != nil && o.ApplicationSecurityHostSum != nil {
		return true
	}

	return false
}

// SetApplicationSecurityHostSum gets a reference to the given UsageBillableSummaryBody and assigns it to the ApplicationSecurityHostSum field.
func (o *UsageBillableSummaryKeys) SetApplicationSecurityHostSum(v UsageBillableSummaryBody) {
	o.ApplicationSecurityHostSum = &v
}

// GetCiPipelineIndexedSpansSum returns the CiPipelineIndexedSpansSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetCiPipelineIndexedSpansSum() UsageBillableSummaryBody {
	if o == nil || o.CiPipelineIndexedSpansSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.CiPipelineIndexedSpansSum
}

// GetCiPipelineIndexedSpansSumOk returns a tuple with the CiPipelineIndexedSpansSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetCiPipelineIndexedSpansSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.CiPipelineIndexedSpansSum == nil {
		return nil, false
	}
	return o.CiPipelineIndexedSpansSum, true
}

// HasCiPipelineIndexedSpansSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasCiPipelineIndexedSpansSum() bool {
	if o != nil && o.CiPipelineIndexedSpansSum != nil {
		return true
	}

	return false
}

// SetCiPipelineIndexedSpansSum gets a reference to the given UsageBillableSummaryBody and assigns it to the CiPipelineIndexedSpansSum field.
func (o *UsageBillableSummaryKeys) SetCiPipelineIndexedSpansSum(v UsageBillableSummaryBody) {
	o.CiPipelineIndexedSpansSum = &v
}

// GetCiPipelineMaximum returns the CiPipelineMaximum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetCiPipelineMaximum() UsageBillableSummaryBody {
	if o == nil || o.CiPipelineMaximum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.CiPipelineMaximum
}

// GetCiPipelineMaximumOk returns a tuple with the CiPipelineMaximum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetCiPipelineMaximumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.CiPipelineMaximum == nil {
		return nil, false
	}
	return o.CiPipelineMaximum, true
}

// HasCiPipelineMaximum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasCiPipelineMaximum() bool {
	if o != nil && o.CiPipelineMaximum != nil {
		return true
	}

	return false
}

// SetCiPipelineMaximum gets a reference to the given UsageBillableSummaryBody and assigns it to the CiPipelineMaximum field.
func (o *UsageBillableSummaryKeys) SetCiPipelineMaximum(v UsageBillableSummaryBody) {
	o.CiPipelineMaximum = &v
}

// GetCiPipelineSum returns the CiPipelineSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetCiPipelineSum() UsageBillableSummaryBody {
	if o == nil || o.CiPipelineSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.CiPipelineSum
}

// GetCiPipelineSumOk returns a tuple with the CiPipelineSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetCiPipelineSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.CiPipelineSum == nil {
		return nil, false
	}
	return o.CiPipelineSum, true
}

// HasCiPipelineSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasCiPipelineSum() bool {
	if o != nil && o.CiPipelineSum != nil {
		return true
	}

	return false
}

// SetCiPipelineSum gets a reference to the given UsageBillableSummaryBody and assigns it to the CiPipelineSum field.
func (o *UsageBillableSummaryKeys) SetCiPipelineSum(v UsageBillableSummaryBody) {
	o.CiPipelineSum = &v
}

// GetCiTestIndexedSpansSum returns the CiTestIndexedSpansSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetCiTestIndexedSpansSum() UsageBillableSummaryBody {
	if o == nil || o.CiTestIndexedSpansSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.CiTestIndexedSpansSum
}

// GetCiTestIndexedSpansSumOk returns a tuple with the CiTestIndexedSpansSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetCiTestIndexedSpansSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.CiTestIndexedSpansSum == nil {
		return nil, false
	}
	return o.CiTestIndexedSpansSum, true
}

// HasCiTestIndexedSpansSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasCiTestIndexedSpansSum() bool {
	if o != nil && o.CiTestIndexedSpansSum != nil {
		return true
	}

	return false
}

// SetCiTestIndexedSpansSum gets a reference to the given UsageBillableSummaryBody and assigns it to the CiTestIndexedSpansSum field.
func (o *UsageBillableSummaryKeys) SetCiTestIndexedSpansSum(v UsageBillableSummaryBody) {
	o.CiTestIndexedSpansSum = &v
}

// GetCiTestingMaximum returns the CiTestingMaximum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetCiTestingMaximum() UsageBillableSummaryBody {
	if o == nil || o.CiTestingMaximum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.CiTestingMaximum
}

// GetCiTestingMaximumOk returns a tuple with the CiTestingMaximum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetCiTestingMaximumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.CiTestingMaximum == nil {
		return nil, false
	}
	return o.CiTestingMaximum, true
}

// HasCiTestingMaximum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasCiTestingMaximum() bool {
	if o != nil && o.CiTestingMaximum != nil {
		return true
	}

	return false
}

// SetCiTestingMaximum gets a reference to the given UsageBillableSummaryBody and assigns it to the CiTestingMaximum field.
func (o *UsageBillableSummaryKeys) SetCiTestingMaximum(v UsageBillableSummaryBody) {
	o.CiTestingMaximum = &v
}

// GetCiTestingSum returns the CiTestingSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetCiTestingSum() UsageBillableSummaryBody {
	if o == nil || o.CiTestingSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.CiTestingSum
}

// GetCiTestingSumOk returns a tuple with the CiTestingSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetCiTestingSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.CiTestingSum == nil {
		return nil, false
	}
	return o.CiTestingSum, true
}

// HasCiTestingSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasCiTestingSum() bool {
	if o != nil && o.CiTestingSum != nil {
		return true
	}

	return false
}

// SetCiTestingSum gets a reference to the given UsageBillableSummaryBody and assigns it to the CiTestingSum field.
func (o *UsageBillableSummaryKeys) SetCiTestingSum(v UsageBillableSummaryBody) {
	o.CiTestingSum = &v
}

// GetCspmContainerSum returns the CspmContainerSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetCspmContainerSum() UsageBillableSummaryBody {
	if o == nil || o.CspmContainerSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.CspmContainerSum
}

// GetCspmContainerSumOk returns a tuple with the CspmContainerSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetCspmContainerSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.CspmContainerSum == nil {
		return nil, false
	}
	return o.CspmContainerSum, true
}

// HasCspmContainerSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasCspmContainerSum() bool {
	if o != nil && o.CspmContainerSum != nil {
		return true
	}

	return false
}

// SetCspmContainerSum gets a reference to the given UsageBillableSummaryBody and assigns it to the CspmContainerSum field.
func (o *UsageBillableSummaryKeys) SetCspmContainerSum(v UsageBillableSummaryBody) {
	o.CspmContainerSum = &v
}

// GetCspmHostSum returns the CspmHostSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetCspmHostSum() UsageBillableSummaryBody {
	if o == nil || o.CspmHostSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.CspmHostSum
}

// GetCspmHostSumOk returns a tuple with the CspmHostSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetCspmHostSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.CspmHostSum == nil {
		return nil, false
	}
	return o.CspmHostSum, true
}

// HasCspmHostSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasCspmHostSum() bool {
	if o != nil && o.CspmHostSum != nil {
		return true
	}

	return false
}

// SetCspmHostSum gets a reference to the given UsageBillableSummaryBody and assigns it to the CspmHostSum field.
func (o *UsageBillableSummaryKeys) SetCspmHostSum(v UsageBillableSummaryBody) {
	o.CspmHostSum = &v
}

// GetCspmHostTop99p returns the CspmHostTop99p field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetCspmHostTop99p() UsageBillableSummaryBody {
	if o == nil || o.CspmHostTop99p == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.CspmHostTop99p
}

// GetCspmHostTop99pOk returns a tuple with the CspmHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetCspmHostTop99pOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.CspmHostTop99p == nil {
		return nil, false
	}
	return o.CspmHostTop99p, true
}

// HasCspmHostTop99p returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasCspmHostTop99p() bool {
	if o != nil && o.CspmHostTop99p != nil {
		return true
	}

	return false
}

// SetCspmHostTop99p gets a reference to the given UsageBillableSummaryBody and assigns it to the CspmHostTop99p field.
func (o *UsageBillableSummaryKeys) SetCspmHostTop99p(v UsageBillableSummaryBody) {
	o.CspmHostTop99p = &v
}

// GetCustomEventSum returns the CustomEventSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetCustomEventSum() UsageBillableSummaryBody {
	if o == nil || o.CustomEventSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.CustomEventSum
}

// GetCustomEventSumOk returns a tuple with the CustomEventSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetCustomEventSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.CustomEventSum == nil {
		return nil, false
	}
	return o.CustomEventSum, true
}

// HasCustomEventSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasCustomEventSum() bool {
	if o != nil && o.CustomEventSum != nil {
		return true
	}

	return false
}

// SetCustomEventSum gets a reference to the given UsageBillableSummaryBody and assigns it to the CustomEventSum field.
func (o *UsageBillableSummaryKeys) SetCustomEventSum(v UsageBillableSummaryBody) {
	o.CustomEventSum = &v
}

// GetCwsContainerSum returns the CwsContainerSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetCwsContainerSum() UsageBillableSummaryBody {
	if o == nil || o.CwsContainerSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.CwsContainerSum
}

// GetCwsContainerSumOk returns a tuple with the CwsContainerSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetCwsContainerSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.CwsContainerSum == nil {
		return nil, false
	}
	return o.CwsContainerSum, true
}

// HasCwsContainerSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasCwsContainerSum() bool {
	if o != nil && o.CwsContainerSum != nil {
		return true
	}

	return false
}

// SetCwsContainerSum gets a reference to the given UsageBillableSummaryBody and assigns it to the CwsContainerSum field.
func (o *UsageBillableSummaryKeys) SetCwsContainerSum(v UsageBillableSummaryBody) {
	o.CwsContainerSum = &v
}

// GetCwsHostSum returns the CwsHostSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetCwsHostSum() UsageBillableSummaryBody {
	if o == nil || o.CwsHostSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.CwsHostSum
}

// GetCwsHostSumOk returns a tuple with the CwsHostSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetCwsHostSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.CwsHostSum == nil {
		return nil, false
	}
	return o.CwsHostSum, true
}

// HasCwsHostSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasCwsHostSum() bool {
	if o != nil && o.CwsHostSum != nil {
		return true
	}

	return false
}

// SetCwsHostSum gets a reference to the given UsageBillableSummaryBody and assigns it to the CwsHostSum field.
func (o *UsageBillableSummaryKeys) SetCwsHostSum(v UsageBillableSummaryBody) {
	o.CwsHostSum = &v
}

// GetCwsHostTop99p returns the CwsHostTop99p field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetCwsHostTop99p() UsageBillableSummaryBody {
	if o == nil || o.CwsHostTop99p == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.CwsHostTop99p
}

// GetCwsHostTop99pOk returns a tuple with the CwsHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetCwsHostTop99pOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.CwsHostTop99p == nil {
		return nil, false
	}
	return o.CwsHostTop99p, true
}

// HasCwsHostTop99p returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasCwsHostTop99p() bool {
	if o != nil && o.CwsHostTop99p != nil {
		return true
	}

	return false
}

// SetCwsHostTop99p gets a reference to the given UsageBillableSummaryBody and assigns it to the CwsHostTop99p field.
func (o *UsageBillableSummaryKeys) SetCwsHostTop99p(v UsageBillableSummaryBody) {
	o.CwsHostTop99p = &v
}

// GetDbmHostSum returns the DbmHostSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetDbmHostSum() UsageBillableSummaryBody {
	if o == nil || o.DbmHostSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.DbmHostSum
}

// GetDbmHostSumOk returns a tuple with the DbmHostSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetDbmHostSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.DbmHostSum == nil {
		return nil, false
	}
	return o.DbmHostSum, true
}

// HasDbmHostSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasDbmHostSum() bool {
	if o != nil && o.DbmHostSum != nil {
		return true
	}

	return false
}

// SetDbmHostSum gets a reference to the given UsageBillableSummaryBody and assigns it to the DbmHostSum field.
func (o *UsageBillableSummaryKeys) SetDbmHostSum(v UsageBillableSummaryBody) {
	o.DbmHostSum = &v
}

// GetDbmHostTop99p returns the DbmHostTop99p field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetDbmHostTop99p() UsageBillableSummaryBody {
	if o == nil || o.DbmHostTop99p == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.DbmHostTop99p
}

// GetDbmHostTop99pOk returns a tuple with the DbmHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetDbmHostTop99pOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.DbmHostTop99p == nil {
		return nil, false
	}
	return o.DbmHostTop99p, true
}

// HasDbmHostTop99p returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasDbmHostTop99p() bool {
	if o != nil && o.DbmHostTop99p != nil {
		return true
	}

	return false
}

// SetDbmHostTop99p gets a reference to the given UsageBillableSummaryBody and assigns it to the DbmHostTop99p field.
func (o *UsageBillableSummaryKeys) SetDbmHostTop99p(v UsageBillableSummaryBody) {
	o.DbmHostTop99p = &v
}

// GetDbmNormalizedQueriesAverage returns the DbmNormalizedQueriesAverage field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetDbmNormalizedQueriesAverage() UsageBillableSummaryBody {
	if o == nil || o.DbmNormalizedQueriesAverage == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.DbmNormalizedQueriesAverage
}

// GetDbmNormalizedQueriesAverageOk returns a tuple with the DbmNormalizedQueriesAverage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetDbmNormalizedQueriesAverageOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.DbmNormalizedQueriesAverage == nil {
		return nil, false
	}
	return o.DbmNormalizedQueriesAverage, true
}

// HasDbmNormalizedQueriesAverage returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasDbmNormalizedQueriesAverage() bool {
	if o != nil && o.DbmNormalizedQueriesAverage != nil {
		return true
	}

	return false
}

// SetDbmNormalizedQueriesAverage gets a reference to the given UsageBillableSummaryBody and assigns it to the DbmNormalizedQueriesAverage field.
func (o *UsageBillableSummaryKeys) SetDbmNormalizedQueriesAverage(v UsageBillableSummaryBody) {
	o.DbmNormalizedQueriesAverage = &v
}

// GetDbmNormalizedQueriesSum returns the DbmNormalizedQueriesSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetDbmNormalizedQueriesSum() UsageBillableSummaryBody {
	if o == nil || o.DbmNormalizedQueriesSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.DbmNormalizedQueriesSum
}

// GetDbmNormalizedQueriesSumOk returns a tuple with the DbmNormalizedQueriesSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetDbmNormalizedQueriesSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.DbmNormalizedQueriesSum == nil {
		return nil, false
	}
	return o.DbmNormalizedQueriesSum, true
}

// HasDbmNormalizedQueriesSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasDbmNormalizedQueriesSum() bool {
	if o != nil && o.DbmNormalizedQueriesSum != nil {
		return true
	}

	return false
}

// SetDbmNormalizedQueriesSum gets a reference to the given UsageBillableSummaryBody and assigns it to the DbmNormalizedQueriesSum field.
func (o *UsageBillableSummaryKeys) SetDbmNormalizedQueriesSum(v UsageBillableSummaryBody) {
	o.DbmNormalizedQueriesSum = &v
}

// GetFargateContainerApmAndProfilerAverage returns the FargateContainerApmAndProfilerAverage field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetFargateContainerApmAndProfilerAverage() UsageBillableSummaryBody {
	if o == nil || o.FargateContainerApmAndProfilerAverage == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.FargateContainerApmAndProfilerAverage
}

// GetFargateContainerApmAndProfilerAverageOk returns a tuple with the FargateContainerApmAndProfilerAverage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetFargateContainerApmAndProfilerAverageOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.FargateContainerApmAndProfilerAverage == nil {
		return nil, false
	}
	return o.FargateContainerApmAndProfilerAverage, true
}

// HasFargateContainerApmAndProfilerAverage returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasFargateContainerApmAndProfilerAverage() bool {
	if o != nil && o.FargateContainerApmAndProfilerAverage != nil {
		return true
	}

	return false
}

// SetFargateContainerApmAndProfilerAverage gets a reference to the given UsageBillableSummaryBody and assigns it to the FargateContainerApmAndProfilerAverage field.
func (o *UsageBillableSummaryKeys) SetFargateContainerApmAndProfilerAverage(v UsageBillableSummaryBody) {
	o.FargateContainerApmAndProfilerAverage = &v
}

// GetFargateContainerApmAndProfilerSum returns the FargateContainerApmAndProfilerSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetFargateContainerApmAndProfilerSum() UsageBillableSummaryBody {
	if o == nil || o.FargateContainerApmAndProfilerSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.FargateContainerApmAndProfilerSum
}

// GetFargateContainerApmAndProfilerSumOk returns a tuple with the FargateContainerApmAndProfilerSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetFargateContainerApmAndProfilerSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.FargateContainerApmAndProfilerSum == nil {
		return nil, false
	}
	return o.FargateContainerApmAndProfilerSum, true
}

// HasFargateContainerApmAndProfilerSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasFargateContainerApmAndProfilerSum() bool {
	if o != nil && o.FargateContainerApmAndProfilerSum != nil {
		return true
	}

	return false
}

// SetFargateContainerApmAndProfilerSum gets a reference to the given UsageBillableSummaryBody and assigns it to the FargateContainerApmAndProfilerSum field.
func (o *UsageBillableSummaryKeys) SetFargateContainerApmAndProfilerSum(v UsageBillableSummaryBody) {
	o.FargateContainerApmAndProfilerSum = &v
}

// GetFargateContainerAverage returns the FargateContainerAverage field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetFargateContainerAverage() UsageBillableSummaryBody {
	if o == nil || o.FargateContainerAverage == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.FargateContainerAverage
}

// GetFargateContainerAverageOk returns a tuple with the FargateContainerAverage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetFargateContainerAverageOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.FargateContainerAverage == nil {
		return nil, false
	}
	return o.FargateContainerAverage, true
}

// HasFargateContainerAverage returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasFargateContainerAverage() bool {
	if o != nil && o.FargateContainerAverage != nil {
		return true
	}

	return false
}

// SetFargateContainerAverage gets a reference to the given UsageBillableSummaryBody and assigns it to the FargateContainerAverage field.
func (o *UsageBillableSummaryKeys) SetFargateContainerAverage(v UsageBillableSummaryBody) {
	o.FargateContainerAverage = &v
}

// GetFargateContainerProfilerAverage returns the FargateContainerProfilerAverage field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetFargateContainerProfilerAverage() UsageBillableSummaryBody {
	if o == nil || o.FargateContainerProfilerAverage == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.FargateContainerProfilerAverage
}

// GetFargateContainerProfilerAverageOk returns a tuple with the FargateContainerProfilerAverage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetFargateContainerProfilerAverageOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.FargateContainerProfilerAverage == nil {
		return nil, false
	}
	return o.FargateContainerProfilerAverage, true
}

// HasFargateContainerProfilerAverage returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasFargateContainerProfilerAverage() bool {
	if o != nil && o.FargateContainerProfilerAverage != nil {
		return true
	}

	return false
}

// SetFargateContainerProfilerAverage gets a reference to the given UsageBillableSummaryBody and assigns it to the FargateContainerProfilerAverage field.
func (o *UsageBillableSummaryKeys) SetFargateContainerProfilerAverage(v UsageBillableSummaryBody) {
	o.FargateContainerProfilerAverage = &v
}

// GetFargateContainerProfilerSum returns the FargateContainerProfilerSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetFargateContainerProfilerSum() UsageBillableSummaryBody {
	if o == nil || o.FargateContainerProfilerSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.FargateContainerProfilerSum
}

// GetFargateContainerProfilerSumOk returns a tuple with the FargateContainerProfilerSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetFargateContainerProfilerSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.FargateContainerProfilerSum == nil {
		return nil, false
	}
	return o.FargateContainerProfilerSum, true
}

// HasFargateContainerProfilerSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasFargateContainerProfilerSum() bool {
	if o != nil && o.FargateContainerProfilerSum != nil {
		return true
	}

	return false
}

// SetFargateContainerProfilerSum gets a reference to the given UsageBillableSummaryBody and assigns it to the FargateContainerProfilerSum field.
func (o *UsageBillableSummaryKeys) SetFargateContainerProfilerSum(v UsageBillableSummaryBody) {
	o.FargateContainerProfilerSum = &v
}

// GetFargateContainerSum returns the FargateContainerSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetFargateContainerSum() UsageBillableSummaryBody {
	if o == nil || o.FargateContainerSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.FargateContainerSum
}

// GetFargateContainerSumOk returns a tuple with the FargateContainerSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetFargateContainerSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.FargateContainerSum == nil {
		return nil, false
	}
	return o.FargateContainerSum, true
}

// HasFargateContainerSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasFargateContainerSum() bool {
	if o != nil && o.FargateContainerSum != nil {
		return true
	}

	return false
}

// SetFargateContainerSum gets a reference to the given UsageBillableSummaryBody and assigns it to the FargateContainerSum field.
func (o *UsageBillableSummaryKeys) SetFargateContainerSum(v UsageBillableSummaryBody) {
	o.FargateContainerSum = &v
}

// GetIncidentManagementMaximum returns the IncidentManagementMaximum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetIncidentManagementMaximum() UsageBillableSummaryBody {
	if o == nil || o.IncidentManagementMaximum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.IncidentManagementMaximum
}

// GetIncidentManagementMaximumOk returns a tuple with the IncidentManagementMaximum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetIncidentManagementMaximumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.IncidentManagementMaximum == nil {
		return nil, false
	}
	return o.IncidentManagementMaximum, true
}

// HasIncidentManagementMaximum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasIncidentManagementMaximum() bool {
	if o != nil && o.IncidentManagementMaximum != nil {
		return true
	}

	return false
}

// SetIncidentManagementMaximum gets a reference to the given UsageBillableSummaryBody and assigns it to the IncidentManagementMaximum field.
func (o *UsageBillableSummaryKeys) SetIncidentManagementMaximum(v UsageBillableSummaryBody) {
	o.IncidentManagementMaximum = &v
}

// GetIncidentManagementSum returns the IncidentManagementSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetIncidentManagementSum() UsageBillableSummaryBody {
	if o == nil || o.IncidentManagementSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.IncidentManagementSum
}

// GetIncidentManagementSumOk returns a tuple with the IncidentManagementSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetIncidentManagementSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.IncidentManagementSum == nil {
		return nil, false
	}
	return o.IncidentManagementSum, true
}

// HasIncidentManagementSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasIncidentManagementSum() bool {
	if o != nil && o.IncidentManagementSum != nil {
		return true
	}

	return false
}

// SetIncidentManagementSum gets a reference to the given UsageBillableSummaryBody and assigns it to the IncidentManagementSum field.
func (o *UsageBillableSummaryKeys) SetIncidentManagementSum(v UsageBillableSummaryBody) {
	o.IncidentManagementSum = &v
}

// GetInfraAndApmHostSum returns the InfraAndApmHostSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetInfraAndApmHostSum() UsageBillableSummaryBody {
	if o == nil || o.InfraAndApmHostSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.InfraAndApmHostSum
}

// GetInfraAndApmHostSumOk returns a tuple with the InfraAndApmHostSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetInfraAndApmHostSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.InfraAndApmHostSum == nil {
		return nil, false
	}
	return o.InfraAndApmHostSum, true
}

// HasInfraAndApmHostSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasInfraAndApmHostSum() bool {
	if o != nil && o.InfraAndApmHostSum != nil {
		return true
	}

	return false
}

// SetInfraAndApmHostSum gets a reference to the given UsageBillableSummaryBody and assigns it to the InfraAndApmHostSum field.
func (o *UsageBillableSummaryKeys) SetInfraAndApmHostSum(v UsageBillableSummaryBody) {
	o.InfraAndApmHostSum = &v
}

// GetInfraAndApmHostTop99p returns the InfraAndApmHostTop99p field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetInfraAndApmHostTop99p() UsageBillableSummaryBody {
	if o == nil || o.InfraAndApmHostTop99p == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.InfraAndApmHostTop99p
}

// GetInfraAndApmHostTop99pOk returns a tuple with the InfraAndApmHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetInfraAndApmHostTop99pOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.InfraAndApmHostTop99p == nil {
		return nil, false
	}
	return o.InfraAndApmHostTop99p, true
}

// HasInfraAndApmHostTop99p returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasInfraAndApmHostTop99p() bool {
	if o != nil && o.InfraAndApmHostTop99p != nil {
		return true
	}

	return false
}

// SetInfraAndApmHostTop99p gets a reference to the given UsageBillableSummaryBody and assigns it to the InfraAndApmHostTop99p field.
func (o *UsageBillableSummaryKeys) SetInfraAndApmHostTop99p(v UsageBillableSummaryBody) {
	o.InfraAndApmHostTop99p = &v
}

// GetInfraContainerSum returns the InfraContainerSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetInfraContainerSum() UsageBillableSummaryBody {
	if o == nil || o.InfraContainerSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.InfraContainerSum
}

// GetInfraContainerSumOk returns a tuple with the InfraContainerSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetInfraContainerSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.InfraContainerSum == nil {
		return nil, false
	}
	return o.InfraContainerSum, true
}

// HasInfraContainerSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasInfraContainerSum() bool {
	if o != nil && o.InfraContainerSum != nil {
		return true
	}

	return false
}

// SetInfraContainerSum gets a reference to the given UsageBillableSummaryBody and assigns it to the InfraContainerSum field.
func (o *UsageBillableSummaryKeys) SetInfraContainerSum(v UsageBillableSummaryBody) {
	o.InfraContainerSum = &v
}

// GetInfraHostSum returns the InfraHostSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetInfraHostSum() UsageBillableSummaryBody {
	if o == nil || o.InfraHostSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.InfraHostSum
}

// GetInfraHostSumOk returns a tuple with the InfraHostSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetInfraHostSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.InfraHostSum == nil {
		return nil, false
	}
	return o.InfraHostSum, true
}

// HasInfraHostSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasInfraHostSum() bool {
	if o != nil && o.InfraHostSum != nil {
		return true
	}

	return false
}

// SetInfraHostSum gets a reference to the given UsageBillableSummaryBody and assigns it to the InfraHostSum field.
func (o *UsageBillableSummaryKeys) SetInfraHostSum(v UsageBillableSummaryBody) {
	o.InfraHostSum = &v
}

// GetInfraHostTop99p returns the InfraHostTop99p field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetInfraHostTop99p() UsageBillableSummaryBody {
	if o == nil || o.InfraHostTop99p == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.InfraHostTop99p
}

// GetInfraHostTop99pOk returns a tuple with the InfraHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetInfraHostTop99pOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.InfraHostTop99p == nil {
		return nil, false
	}
	return o.InfraHostTop99p, true
}

// HasInfraHostTop99p returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasInfraHostTop99p() bool {
	if o != nil && o.InfraHostTop99p != nil {
		return true
	}

	return false
}

// SetInfraHostTop99p gets a reference to the given UsageBillableSummaryBody and assigns it to the InfraHostTop99p field.
func (o *UsageBillableSummaryKeys) SetInfraHostTop99p(v UsageBillableSummaryBody) {
	o.InfraHostTop99p = &v
}

// GetIngestedSpansSum returns the IngestedSpansSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetIngestedSpansSum() UsageBillableSummaryBody {
	if o == nil || o.IngestedSpansSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.IngestedSpansSum
}

// GetIngestedSpansSumOk returns a tuple with the IngestedSpansSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetIngestedSpansSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.IngestedSpansSum == nil {
		return nil, false
	}
	return o.IngestedSpansSum, true
}

// HasIngestedSpansSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasIngestedSpansSum() bool {
	if o != nil && o.IngestedSpansSum != nil {
		return true
	}

	return false
}

// SetIngestedSpansSum gets a reference to the given UsageBillableSummaryBody and assigns it to the IngestedSpansSum field.
func (o *UsageBillableSummaryKeys) SetIngestedSpansSum(v UsageBillableSummaryBody) {
	o.IngestedSpansSum = &v
}

// GetIngestedTimeseriesAverage returns the IngestedTimeseriesAverage field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetIngestedTimeseriesAverage() UsageBillableSummaryBody {
	if o == nil || o.IngestedTimeseriesAverage == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.IngestedTimeseriesAverage
}

// GetIngestedTimeseriesAverageOk returns a tuple with the IngestedTimeseriesAverage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetIngestedTimeseriesAverageOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.IngestedTimeseriesAverage == nil {
		return nil, false
	}
	return o.IngestedTimeseriesAverage, true
}

// HasIngestedTimeseriesAverage returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasIngestedTimeseriesAverage() bool {
	if o != nil && o.IngestedTimeseriesAverage != nil {
		return true
	}

	return false
}

// SetIngestedTimeseriesAverage gets a reference to the given UsageBillableSummaryBody and assigns it to the IngestedTimeseriesAverage field.
func (o *UsageBillableSummaryKeys) SetIngestedTimeseriesAverage(v UsageBillableSummaryBody) {
	o.IngestedTimeseriesAverage = &v
}

// GetIngestedTimeseriesSum returns the IngestedTimeseriesSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetIngestedTimeseriesSum() UsageBillableSummaryBody {
	if o == nil || o.IngestedTimeseriesSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.IngestedTimeseriesSum
}

// GetIngestedTimeseriesSumOk returns a tuple with the IngestedTimeseriesSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetIngestedTimeseriesSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.IngestedTimeseriesSum == nil {
		return nil, false
	}
	return o.IngestedTimeseriesSum, true
}

// HasIngestedTimeseriesSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasIngestedTimeseriesSum() bool {
	if o != nil && o.IngestedTimeseriesSum != nil {
		return true
	}

	return false
}

// SetIngestedTimeseriesSum gets a reference to the given UsageBillableSummaryBody and assigns it to the IngestedTimeseriesSum field.
func (o *UsageBillableSummaryKeys) SetIngestedTimeseriesSum(v UsageBillableSummaryBody) {
	o.IngestedTimeseriesSum = &v
}

// GetIotSum returns the IotSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetIotSum() UsageBillableSummaryBody {
	if o == nil || o.IotSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.IotSum
}

// GetIotSumOk returns a tuple with the IotSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetIotSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.IotSum == nil {
		return nil, false
	}
	return o.IotSum, true
}

// HasIotSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasIotSum() bool {
	if o != nil && o.IotSum != nil {
		return true
	}

	return false
}

// SetIotSum gets a reference to the given UsageBillableSummaryBody and assigns it to the IotSum field.
func (o *UsageBillableSummaryKeys) SetIotSum(v UsageBillableSummaryBody) {
	o.IotSum = &v
}

// GetIotTop99p returns the IotTop99p field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetIotTop99p() UsageBillableSummaryBody {
	if o == nil || o.IotTop99p == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.IotTop99p
}

// GetIotTop99pOk returns a tuple with the IotTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetIotTop99pOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.IotTop99p == nil {
		return nil, false
	}
	return o.IotTop99p, true
}

// HasIotTop99p returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasIotTop99p() bool {
	if o != nil && o.IotTop99p != nil {
		return true
	}

	return false
}

// SetIotTop99p gets a reference to the given UsageBillableSummaryBody and assigns it to the IotTop99p field.
func (o *UsageBillableSummaryKeys) SetIotTop99p(v UsageBillableSummaryBody) {
	o.IotTop99p = &v
}

// GetLambdaFunctionAverage returns the LambdaFunctionAverage field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetLambdaFunctionAverage() UsageBillableSummaryBody {
	if o == nil || o.LambdaFunctionAverage == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.LambdaFunctionAverage
}

// GetLambdaFunctionAverageOk returns a tuple with the LambdaFunctionAverage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetLambdaFunctionAverageOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.LambdaFunctionAverage == nil {
		return nil, false
	}
	return o.LambdaFunctionAverage, true
}

// HasLambdaFunctionAverage returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasLambdaFunctionAverage() bool {
	if o != nil && o.LambdaFunctionAverage != nil {
		return true
	}

	return false
}

// SetLambdaFunctionAverage gets a reference to the given UsageBillableSummaryBody and assigns it to the LambdaFunctionAverage field.
func (o *UsageBillableSummaryKeys) SetLambdaFunctionAverage(v UsageBillableSummaryBody) {
	o.LambdaFunctionAverage = &v
}

// GetLambdaFunctionSum returns the LambdaFunctionSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetLambdaFunctionSum() UsageBillableSummaryBody {
	if o == nil || o.LambdaFunctionSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.LambdaFunctionSum
}

// GetLambdaFunctionSumOk returns a tuple with the LambdaFunctionSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetLambdaFunctionSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.LambdaFunctionSum == nil {
		return nil, false
	}
	return o.LambdaFunctionSum, true
}

// HasLambdaFunctionSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasLambdaFunctionSum() bool {
	if o != nil && o.LambdaFunctionSum != nil {
		return true
	}

	return false
}

// SetLambdaFunctionSum gets a reference to the given UsageBillableSummaryBody and assigns it to the LambdaFunctionSum field.
func (o *UsageBillableSummaryKeys) SetLambdaFunctionSum(v UsageBillableSummaryBody) {
	o.LambdaFunctionSum = &v
}

// GetLogsIndexed15daySum returns the LogsIndexed15daySum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetLogsIndexed15daySum() UsageBillableSummaryBody {
	if o == nil || o.LogsIndexed15daySum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.LogsIndexed15daySum
}

// GetLogsIndexed15daySumOk returns a tuple with the LogsIndexed15daySum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetLogsIndexed15daySumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.LogsIndexed15daySum == nil {
		return nil, false
	}
	return o.LogsIndexed15daySum, true
}

// HasLogsIndexed15daySum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasLogsIndexed15daySum() bool {
	if o != nil && o.LogsIndexed15daySum != nil {
		return true
	}

	return false
}

// SetLogsIndexed15daySum gets a reference to the given UsageBillableSummaryBody and assigns it to the LogsIndexed15daySum field.
func (o *UsageBillableSummaryKeys) SetLogsIndexed15daySum(v UsageBillableSummaryBody) {
	o.LogsIndexed15daySum = &v
}

// GetLogsIndexed180daySum returns the LogsIndexed180daySum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetLogsIndexed180daySum() UsageBillableSummaryBody {
	if o == nil || o.LogsIndexed180daySum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.LogsIndexed180daySum
}

// GetLogsIndexed180daySumOk returns a tuple with the LogsIndexed180daySum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetLogsIndexed180daySumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.LogsIndexed180daySum == nil {
		return nil, false
	}
	return o.LogsIndexed180daySum, true
}

// HasLogsIndexed180daySum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasLogsIndexed180daySum() bool {
	if o != nil && o.LogsIndexed180daySum != nil {
		return true
	}

	return false
}

// SetLogsIndexed180daySum gets a reference to the given UsageBillableSummaryBody and assigns it to the LogsIndexed180daySum field.
func (o *UsageBillableSummaryKeys) SetLogsIndexed180daySum(v UsageBillableSummaryBody) {
	o.LogsIndexed180daySum = &v
}

// GetLogsIndexed30daySum returns the LogsIndexed30daySum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetLogsIndexed30daySum() UsageBillableSummaryBody {
	if o == nil || o.LogsIndexed30daySum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.LogsIndexed30daySum
}

// GetLogsIndexed30daySumOk returns a tuple with the LogsIndexed30daySum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetLogsIndexed30daySumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.LogsIndexed30daySum == nil {
		return nil, false
	}
	return o.LogsIndexed30daySum, true
}

// HasLogsIndexed30daySum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasLogsIndexed30daySum() bool {
	if o != nil && o.LogsIndexed30daySum != nil {
		return true
	}

	return false
}

// SetLogsIndexed30daySum gets a reference to the given UsageBillableSummaryBody and assigns it to the LogsIndexed30daySum field.
func (o *UsageBillableSummaryKeys) SetLogsIndexed30daySum(v UsageBillableSummaryBody) {
	o.LogsIndexed30daySum = &v
}

// GetLogsIndexed360daySum returns the LogsIndexed360daySum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetLogsIndexed360daySum() UsageBillableSummaryBody {
	if o == nil || o.LogsIndexed360daySum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.LogsIndexed360daySum
}

// GetLogsIndexed360daySumOk returns a tuple with the LogsIndexed360daySum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetLogsIndexed360daySumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.LogsIndexed360daySum == nil {
		return nil, false
	}
	return o.LogsIndexed360daySum, true
}

// HasLogsIndexed360daySum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasLogsIndexed360daySum() bool {
	if o != nil && o.LogsIndexed360daySum != nil {
		return true
	}

	return false
}

// SetLogsIndexed360daySum gets a reference to the given UsageBillableSummaryBody and assigns it to the LogsIndexed360daySum field.
func (o *UsageBillableSummaryKeys) SetLogsIndexed360daySum(v UsageBillableSummaryBody) {
	o.LogsIndexed360daySum = &v
}

// GetLogsIndexed3daySum returns the LogsIndexed3daySum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetLogsIndexed3daySum() UsageBillableSummaryBody {
	if o == nil || o.LogsIndexed3daySum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.LogsIndexed3daySum
}

// GetLogsIndexed3daySumOk returns a tuple with the LogsIndexed3daySum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetLogsIndexed3daySumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.LogsIndexed3daySum == nil {
		return nil, false
	}
	return o.LogsIndexed3daySum, true
}

// HasLogsIndexed3daySum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasLogsIndexed3daySum() bool {
	if o != nil && o.LogsIndexed3daySum != nil {
		return true
	}

	return false
}

// SetLogsIndexed3daySum gets a reference to the given UsageBillableSummaryBody and assigns it to the LogsIndexed3daySum field.
func (o *UsageBillableSummaryKeys) SetLogsIndexed3daySum(v UsageBillableSummaryBody) {
	o.LogsIndexed3daySum = &v
}

// GetLogsIndexed45daySum returns the LogsIndexed45daySum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetLogsIndexed45daySum() UsageBillableSummaryBody {
	if o == nil || o.LogsIndexed45daySum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.LogsIndexed45daySum
}

// GetLogsIndexed45daySumOk returns a tuple with the LogsIndexed45daySum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetLogsIndexed45daySumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.LogsIndexed45daySum == nil {
		return nil, false
	}
	return o.LogsIndexed45daySum, true
}

// HasLogsIndexed45daySum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasLogsIndexed45daySum() bool {
	if o != nil && o.LogsIndexed45daySum != nil {
		return true
	}

	return false
}

// SetLogsIndexed45daySum gets a reference to the given UsageBillableSummaryBody and assigns it to the LogsIndexed45daySum field.
func (o *UsageBillableSummaryKeys) SetLogsIndexed45daySum(v UsageBillableSummaryBody) {
	o.LogsIndexed45daySum = &v
}

// GetLogsIndexed60daySum returns the LogsIndexed60daySum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetLogsIndexed60daySum() UsageBillableSummaryBody {
	if o == nil || o.LogsIndexed60daySum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.LogsIndexed60daySum
}

// GetLogsIndexed60daySumOk returns a tuple with the LogsIndexed60daySum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetLogsIndexed60daySumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.LogsIndexed60daySum == nil {
		return nil, false
	}
	return o.LogsIndexed60daySum, true
}

// HasLogsIndexed60daySum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasLogsIndexed60daySum() bool {
	if o != nil && o.LogsIndexed60daySum != nil {
		return true
	}

	return false
}

// SetLogsIndexed60daySum gets a reference to the given UsageBillableSummaryBody and assigns it to the LogsIndexed60daySum field.
func (o *UsageBillableSummaryKeys) SetLogsIndexed60daySum(v UsageBillableSummaryBody) {
	o.LogsIndexed60daySum = &v
}

// GetLogsIndexed7daySum returns the LogsIndexed7daySum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetLogsIndexed7daySum() UsageBillableSummaryBody {
	if o == nil || o.LogsIndexed7daySum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.LogsIndexed7daySum
}

// GetLogsIndexed7daySumOk returns a tuple with the LogsIndexed7daySum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetLogsIndexed7daySumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.LogsIndexed7daySum == nil {
		return nil, false
	}
	return o.LogsIndexed7daySum, true
}

// HasLogsIndexed7daySum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasLogsIndexed7daySum() bool {
	if o != nil && o.LogsIndexed7daySum != nil {
		return true
	}

	return false
}

// SetLogsIndexed7daySum gets a reference to the given UsageBillableSummaryBody and assigns it to the LogsIndexed7daySum field.
func (o *UsageBillableSummaryKeys) SetLogsIndexed7daySum(v UsageBillableSummaryBody) {
	o.LogsIndexed7daySum = &v
}

// GetLogsIndexed90daySum returns the LogsIndexed90daySum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetLogsIndexed90daySum() UsageBillableSummaryBody {
	if o == nil || o.LogsIndexed90daySum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.LogsIndexed90daySum
}

// GetLogsIndexed90daySumOk returns a tuple with the LogsIndexed90daySum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetLogsIndexed90daySumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.LogsIndexed90daySum == nil {
		return nil, false
	}
	return o.LogsIndexed90daySum, true
}

// HasLogsIndexed90daySum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasLogsIndexed90daySum() bool {
	if o != nil && o.LogsIndexed90daySum != nil {
		return true
	}

	return false
}

// SetLogsIndexed90daySum gets a reference to the given UsageBillableSummaryBody and assigns it to the LogsIndexed90daySum field.
func (o *UsageBillableSummaryKeys) SetLogsIndexed90daySum(v UsageBillableSummaryBody) {
	o.LogsIndexed90daySum = &v
}

// GetLogsIndexedCustomRetentionSum returns the LogsIndexedCustomRetentionSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetLogsIndexedCustomRetentionSum() UsageBillableSummaryBody {
	if o == nil || o.LogsIndexedCustomRetentionSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.LogsIndexedCustomRetentionSum
}

// GetLogsIndexedCustomRetentionSumOk returns a tuple with the LogsIndexedCustomRetentionSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetLogsIndexedCustomRetentionSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.LogsIndexedCustomRetentionSum == nil {
		return nil, false
	}
	return o.LogsIndexedCustomRetentionSum, true
}

// HasLogsIndexedCustomRetentionSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasLogsIndexedCustomRetentionSum() bool {
	if o != nil && o.LogsIndexedCustomRetentionSum != nil {
		return true
	}

	return false
}

// SetLogsIndexedCustomRetentionSum gets a reference to the given UsageBillableSummaryBody and assigns it to the LogsIndexedCustomRetentionSum field.
func (o *UsageBillableSummaryKeys) SetLogsIndexedCustomRetentionSum(v UsageBillableSummaryBody) {
	o.LogsIndexedCustomRetentionSum = &v
}

// GetLogsIndexedSum returns the LogsIndexedSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetLogsIndexedSum() UsageBillableSummaryBody {
	if o == nil || o.LogsIndexedSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.LogsIndexedSum
}

// GetLogsIndexedSumOk returns a tuple with the LogsIndexedSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetLogsIndexedSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.LogsIndexedSum == nil {
		return nil, false
	}
	return o.LogsIndexedSum, true
}

// HasLogsIndexedSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasLogsIndexedSum() bool {
	if o != nil && o.LogsIndexedSum != nil {
		return true
	}

	return false
}

// SetLogsIndexedSum gets a reference to the given UsageBillableSummaryBody and assigns it to the LogsIndexedSum field.
func (o *UsageBillableSummaryKeys) SetLogsIndexedSum(v UsageBillableSummaryBody) {
	o.LogsIndexedSum = &v
}

// GetLogsIngestedSum returns the LogsIngestedSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetLogsIngestedSum() UsageBillableSummaryBody {
	if o == nil || o.LogsIngestedSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.LogsIngestedSum
}

// GetLogsIngestedSumOk returns a tuple with the LogsIngestedSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetLogsIngestedSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.LogsIngestedSum == nil {
		return nil, false
	}
	return o.LogsIngestedSum, true
}

// HasLogsIngestedSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasLogsIngestedSum() bool {
	if o != nil && o.LogsIngestedSum != nil {
		return true
	}

	return false
}

// SetLogsIngestedSum gets a reference to the given UsageBillableSummaryBody and assigns it to the LogsIngestedSum field.
func (o *UsageBillableSummaryKeys) SetLogsIngestedSum(v UsageBillableSummaryBody) {
	o.LogsIngestedSum = &v
}

// GetNetworkDeviceSum returns the NetworkDeviceSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetNetworkDeviceSum() UsageBillableSummaryBody {
	if o == nil || o.NetworkDeviceSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.NetworkDeviceSum
}

// GetNetworkDeviceSumOk returns a tuple with the NetworkDeviceSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetNetworkDeviceSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.NetworkDeviceSum == nil {
		return nil, false
	}
	return o.NetworkDeviceSum, true
}

// HasNetworkDeviceSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasNetworkDeviceSum() bool {
	if o != nil && o.NetworkDeviceSum != nil {
		return true
	}

	return false
}

// SetNetworkDeviceSum gets a reference to the given UsageBillableSummaryBody and assigns it to the NetworkDeviceSum field.
func (o *UsageBillableSummaryKeys) SetNetworkDeviceSum(v UsageBillableSummaryBody) {
	o.NetworkDeviceSum = &v
}

// GetNetworkDeviceTop99p returns the NetworkDeviceTop99p field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetNetworkDeviceTop99p() UsageBillableSummaryBody {
	if o == nil || o.NetworkDeviceTop99p == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.NetworkDeviceTop99p
}

// GetNetworkDeviceTop99pOk returns a tuple with the NetworkDeviceTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetNetworkDeviceTop99pOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.NetworkDeviceTop99p == nil {
		return nil, false
	}
	return o.NetworkDeviceTop99p, true
}

// HasNetworkDeviceTop99p returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasNetworkDeviceTop99p() bool {
	if o != nil && o.NetworkDeviceTop99p != nil {
		return true
	}

	return false
}

// SetNetworkDeviceTop99p gets a reference to the given UsageBillableSummaryBody and assigns it to the NetworkDeviceTop99p field.
func (o *UsageBillableSummaryKeys) SetNetworkDeviceTop99p(v UsageBillableSummaryBody) {
	o.NetworkDeviceTop99p = &v
}

// GetNpmFlowSum returns the NpmFlowSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetNpmFlowSum() UsageBillableSummaryBody {
	if o == nil || o.NpmFlowSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.NpmFlowSum
}

// GetNpmFlowSumOk returns a tuple with the NpmFlowSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetNpmFlowSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.NpmFlowSum == nil {
		return nil, false
	}
	return o.NpmFlowSum, true
}

// HasNpmFlowSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasNpmFlowSum() bool {
	if o != nil && o.NpmFlowSum != nil {
		return true
	}

	return false
}

// SetNpmFlowSum gets a reference to the given UsageBillableSummaryBody and assigns it to the NpmFlowSum field.
func (o *UsageBillableSummaryKeys) SetNpmFlowSum(v UsageBillableSummaryBody) {
	o.NpmFlowSum = &v
}

// GetNpmHostSum returns the NpmHostSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetNpmHostSum() UsageBillableSummaryBody {
	if o == nil || o.NpmHostSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.NpmHostSum
}

// GetNpmHostSumOk returns a tuple with the NpmHostSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetNpmHostSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.NpmHostSum == nil {
		return nil, false
	}
	return o.NpmHostSum, true
}

// HasNpmHostSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasNpmHostSum() bool {
	if o != nil && o.NpmHostSum != nil {
		return true
	}

	return false
}

// SetNpmHostSum gets a reference to the given UsageBillableSummaryBody and assigns it to the NpmHostSum field.
func (o *UsageBillableSummaryKeys) SetNpmHostSum(v UsageBillableSummaryBody) {
	o.NpmHostSum = &v
}

// GetNpmHostTop99p returns the NpmHostTop99p field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetNpmHostTop99p() UsageBillableSummaryBody {
	if o == nil || o.NpmHostTop99p == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.NpmHostTop99p
}

// GetNpmHostTop99pOk returns a tuple with the NpmHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetNpmHostTop99pOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.NpmHostTop99p == nil {
		return nil, false
	}
	return o.NpmHostTop99p, true
}

// HasNpmHostTop99p returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasNpmHostTop99p() bool {
	if o != nil && o.NpmHostTop99p != nil {
		return true
	}

	return false
}

// SetNpmHostTop99p gets a reference to the given UsageBillableSummaryBody and assigns it to the NpmHostTop99p field.
func (o *UsageBillableSummaryKeys) SetNpmHostTop99p(v UsageBillableSummaryBody) {
	o.NpmHostTop99p = &v
}

// GetObservabilityPipelineSum returns the ObservabilityPipelineSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetObservabilityPipelineSum() UsageBillableSummaryBody {
	if o == nil || o.ObservabilityPipelineSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.ObservabilityPipelineSum
}

// GetObservabilityPipelineSumOk returns a tuple with the ObservabilityPipelineSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetObservabilityPipelineSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.ObservabilityPipelineSum == nil {
		return nil, false
	}
	return o.ObservabilityPipelineSum, true
}

// HasObservabilityPipelineSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasObservabilityPipelineSum() bool {
	if o != nil && o.ObservabilityPipelineSum != nil {
		return true
	}

	return false
}

// SetObservabilityPipelineSum gets a reference to the given UsageBillableSummaryBody and assigns it to the ObservabilityPipelineSum field.
func (o *UsageBillableSummaryKeys) SetObservabilityPipelineSum(v UsageBillableSummaryBody) {
	o.ObservabilityPipelineSum = &v
}

// GetOnlineArchiveSum returns the OnlineArchiveSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetOnlineArchiveSum() UsageBillableSummaryBody {
	if o == nil || o.OnlineArchiveSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.OnlineArchiveSum
}

// GetOnlineArchiveSumOk returns a tuple with the OnlineArchiveSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetOnlineArchiveSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.OnlineArchiveSum == nil {
		return nil, false
	}
	return o.OnlineArchiveSum, true
}

// HasOnlineArchiveSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasOnlineArchiveSum() bool {
	if o != nil && o.OnlineArchiveSum != nil {
		return true
	}

	return false
}

// SetOnlineArchiveSum gets a reference to the given UsageBillableSummaryBody and assigns it to the OnlineArchiveSum field.
func (o *UsageBillableSummaryKeys) SetOnlineArchiveSum(v UsageBillableSummaryBody) {
	o.OnlineArchiveSum = &v
}

// GetProfContainerSum returns the ProfContainerSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetProfContainerSum() UsageBillableSummaryBody {
	if o == nil || o.ProfContainerSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.ProfContainerSum
}

// GetProfContainerSumOk returns a tuple with the ProfContainerSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetProfContainerSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.ProfContainerSum == nil {
		return nil, false
	}
	return o.ProfContainerSum, true
}

// HasProfContainerSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasProfContainerSum() bool {
	if o != nil && o.ProfContainerSum != nil {
		return true
	}

	return false
}

// SetProfContainerSum gets a reference to the given UsageBillableSummaryBody and assigns it to the ProfContainerSum field.
func (o *UsageBillableSummaryKeys) SetProfContainerSum(v UsageBillableSummaryBody) {
	o.ProfContainerSum = &v
}

// GetProfHostSum returns the ProfHostSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetProfHostSum() UsageBillableSummaryBody {
	if o == nil || o.ProfHostSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.ProfHostSum
}

// GetProfHostSumOk returns a tuple with the ProfHostSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetProfHostSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.ProfHostSum == nil {
		return nil, false
	}
	return o.ProfHostSum, true
}

// HasProfHostSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasProfHostSum() bool {
	if o != nil && o.ProfHostSum != nil {
		return true
	}

	return false
}

// SetProfHostSum gets a reference to the given UsageBillableSummaryBody and assigns it to the ProfHostSum field.
func (o *UsageBillableSummaryKeys) SetProfHostSum(v UsageBillableSummaryBody) {
	o.ProfHostSum = &v
}

// GetProfHostTop99p returns the ProfHostTop99p field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetProfHostTop99p() UsageBillableSummaryBody {
	if o == nil || o.ProfHostTop99p == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.ProfHostTop99p
}

// GetProfHostTop99pOk returns a tuple with the ProfHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetProfHostTop99pOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.ProfHostTop99p == nil {
		return nil, false
	}
	return o.ProfHostTop99p, true
}

// HasProfHostTop99p returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasProfHostTop99p() bool {
	if o != nil && o.ProfHostTop99p != nil {
		return true
	}

	return false
}

// SetProfHostTop99p gets a reference to the given UsageBillableSummaryBody and assigns it to the ProfHostTop99p field.
func (o *UsageBillableSummaryKeys) SetProfHostTop99p(v UsageBillableSummaryBody) {
	o.ProfHostTop99p = &v
}

// GetRumLiteSum returns the RumLiteSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetRumLiteSum() UsageBillableSummaryBody {
	if o == nil || o.RumLiteSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.RumLiteSum
}

// GetRumLiteSumOk returns a tuple with the RumLiteSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetRumLiteSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.RumLiteSum == nil {
		return nil, false
	}
	return o.RumLiteSum, true
}

// HasRumLiteSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasRumLiteSum() bool {
	if o != nil && o.RumLiteSum != nil {
		return true
	}

	return false
}

// SetRumLiteSum gets a reference to the given UsageBillableSummaryBody and assigns it to the RumLiteSum field.
func (o *UsageBillableSummaryKeys) SetRumLiteSum(v UsageBillableSummaryBody) {
	o.RumLiteSum = &v
}

// GetRumReplaySum returns the RumReplaySum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetRumReplaySum() UsageBillableSummaryBody {
	if o == nil || o.RumReplaySum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.RumReplaySum
}

// GetRumReplaySumOk returns a tuple with the RumReplaySum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetRumReplaySumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.RumReplaySum == nil {
		return nil, false
	}
	return o.RumReplaySum, true
}

// HasRumReplaySum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasRumReplaySum() bool {
	if o != nil && o.RumReplaySum != nil {
		return true
	}

	return false
}

// SetRumReplaySum gets a reference to the given UsageBillableSummaryBody and assigns it to the RumReplaySum field.
func (o *UsageBillableSummaryKeys) SetRumReplaySum(v UsageBillableSummaryBody) {
	o.RumReplaySum = &v
}

// GetRumSum returns the RumSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetRumSum() UsageBillableSummaryBody {
	if o == nil || o.RumSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.RumSum
}

// GetRumSumOk returns a tuple with the RumSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetRumSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.RumSum == nil {
		return nil, false
	}
	return o.RumSum, true
}

// HasRumSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasRumSum() bool {
	if o != nil && o.RumSum != nil {
		return true
	}

	return false
}

// SetRumSum gets a reference to the given UsageBillableSummaryBody and assigns it to the RumSum field.
func (o *UsageBillableSummaryKeys) SetRumSum(v UsageBillableSummaryBody) {
	o.RumSum = &v
}

// GetRumUnitsSum returns the RumUnitsSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetRumUnitsSum() UsageBillableSummaryBody {
	if o == nil || o.RumUnitsSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.RumUnitsSum
}

// GetRumUnitsSumOk returns a tuple with the RumUnitsSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetRumUnitsSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.RumUnitsSum == nil {
		return nil, false
	}
	return o.RumUnitsSum, true
}

// HasRumUnitsSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasRumUnitsSum() bool {
	if o != nil && o.RumUnitsSum != nil {
		return true
	}

	return false
}

// SetRumUnitsSum gets a reference to the given UsageBillableSummaryBody and assigns it to the RumUnitsSum field.
func (o *UsageBillableSummaryKeys) SetRumUnitsSum(v UsageBillableSummaryBody) {
	o.RumUnitsSum = &v
}

// GetSensitiveDataScannerSum returns the SensitiveDataScannerSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetSensitiveDataScannerSum() UsageBillableSummaryBody {
	if o == nil || o.SensitiveDataScannerSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.SensitiveDataScannerSum
}

// GetSensitiveDataScannerSumOk returns a tuple with the SensitiveDataScannerSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetSensitiveDataScannerSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.SensitiveDataScannerSum == nil {
		return nil, false
	}
	return o.SensitiveDataScannerSum, true
}

// HasSensitiveDataScannerSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasSensitiveDataScannerSum() bool {
	if o != nil && o.SensitiveDataScannerSum != nil {
		return true
	}

	return false
}

// SetSensitiveDataScannerSum gets a reference to the given UsageBillableSummaryBody and assigns it to the SensitiveDataScannerSum field.
func (o *UsageBillableSummaryKeys) SetSensitiveDataScannerSum(v UsageBillableSummaryBody) {
	o.SensitiveDataScannerSum = &v
}

// GetServerlessInvocationSum returns the ServerlessInvocationSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetServerlessInvocationSum() UsageBillableSummaryBody {
	if o == nil || o.ServerlessInvocationSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.ServerlessInvocationSum
}

// GetServerlessInvocationSumOk returns a tuple with the ServerlessInvocationSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetServerlessInvocationSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.ServerlessInvocationSum == nil {
		return nil, false
	}
	return o.ServerlessInvocationSum, true
}

// HasServerlessInvocationSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasServerlessInvocationSum() bool {
	if o != nil && o.ServerlessInvocationSum != nil {
		return true
	}

	return false
}

// SetServerlessInvocationSum gets a reference to the given UsageBillableSummaryBody and assigns it to the ServerlessInvocationSum field.
func (o *UsageBillableSummaryKeys) SetServerlessInvocationSum(v UsageBillableSummaryBody) {
	o.ServerlessInvocationSum = &v
}

// GetSiemSum returns the SiemSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetSiemSum() UsageBillableSummaryBody {
	if o == nil || o.SiemSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.SiemSum
}

// GetSiemSumOk returns a tuple with the SiemSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetSiemSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.SiemSum == nil {
		return nil, false
	}
	return o.SiemSum, true
}

// HasSiemSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasSiemSum() bool {
	if o != nil && o.SiemSum != nil {
		return true
	}

	return false
}

// SetSiemSum gets a reference to the given UsageBillableSummaryBody and assigns it to the SiemSum field.
func (o *UsageBillableSummaryKeys) SetSiemSum(v UsageBillableSummaryBody) {
	o.SiemSum = &v
}

// GetStandardTimeseriesAverage returns the StandardTimeseriesAverage field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetStandardTimeseriesAverage() UsageBillableSummaryBody {
	if o == nil || o.StandardTimeseriesAverage == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.StandardTimeseriesAverage
}

// GetStandardTimeseriesAverageOk returns a tuple with the StandardTimeseriesAverage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetStandardTimeseriesAverageOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.StandardTimeseriesAverage == nil {
		return nil, false
	}
	return o.StandardTimeseriesAverage, true
}

// HasStandardTimeseriesAverage returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasStandardTimeseriesAverage() bool {
	if o != nil && o.StandardTimeseriesAverage != nil {
		return true
	}

	return false
}

// SetStandardTimeseriesAverage gets a reference to the given UsageBillableSummaryBody and assigns it to the StandardTimeseriesAverage field.
func (o *UsageBillableSummaryKeys) SetStandardTimeseriesAverage(v UsageBillableSummaryBody) {
	o.StandardTimeseriesAverage = &v
}

// GetSyntheticsApiTestsSum returns the SyntheticsApiTestsSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetSyntheticsApiTestsSum() UsageBillableSummaryBody {
	if o == nil || o.SyntheticsApiTestsSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.SyntheticsApiTestsSum
}

// GetSyntheticsApiTestsSumOk returns a tuple with the SyntheticsApiTestsSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetSyntheticsApiTestsSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.SyntheticsApiTestsSum == nil {
		return nil, false
	}
	return o.SyntheticsApiTestsSum, true
}

// HasSyntheticsApiTestsSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasSyntheticsApiTestsSum() bool {
	if o != nil && o.SyntheticsApiTestsSum != nil {
		return true
	}

	return false
}

// SetSyntheticsApiTestsSum gets a reference to the given UsageBillableSummaryBody and assigns it to the SyntheticsApiTestsSum field.
func (o *UsageBillableSummaryKeys) SetSyntheticsApiTestsSum(v UsageBillableSummaryBody) {
	o.SyntheticsApiTestsSum = &v
}

// GetSyntheticsBrowserChecksSum returns the SyntheticsBrowserChecksSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetSyntheticsBrowserChecksSum() UsageBillableSummaryBody {
	if o == nil || o.SyntheticsBrowserChecksSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.SyntheticsBrowserChecksSum
}

// GetSyntheticsBrowserChecksSumOk returns a tuple with the SyntheticsBrowserChecksSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetSyntheticsBrowserChecksSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.SyntheticsBrowserChecksSum == nil {
		return nil, false
	}
	return o.SyntheticsBrowserChecksSum, true
}

// HasSyntheticsBrowserChecksSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasSyntheticsBrowserChecksSum() bool {
	if o != nil && o.SyntheticsBrowserChecksSum != nil {
		return true
	}

	return false
}

// SetSyntheticsBrowserChecksSum gets a reference to the given UsageBillableSummaryBody and assigns it to the SyntheticsBrowserChecksSum field.
func (o *UsageBillableSummaryKeys) SetSyntheticsBrowserChecksSum(v UsageBillableSummaryBody) {
	o.SyntheticsBrowserChecksSum = &v
}

// GetTimeseriesAverage returns the TimeseriesAverage field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetTimeseriesAverage() UsageBillableSummaryBody {
	if o == nil || o.TimeseriesAverage == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.TimeseriesAverage
}

// GetTimeseriesAverageOk returns a tuple with the TimeseriesAverage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetTimeseriesAverageOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.TimeseriesAverage == nil {
		return nil, false
	}
	return o.TimeseriesAverage, true
}

// HasTimeseriesAverage returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasTimeseriesAverage() bool {
	if o != nil && o.TimeseriesAverage != nil {
		return true
	}

	return false
}

// SetTimeseriesAverage gets a reference to the given UsageBillableSummaryBody and assigns it to the TimeseriesAverage field.
func (o *UsageBillableSummaryKeys) SetTimeseriesAverage(v UsageBillableSummaryBody) {
	o.TimeseriesAverage = &v
}

// GetTimeseriesSum returns the TimeseriesSum field value if set, zero value otherwise.
func (o *UsageBillableSummaryKeys) GetTimeseriesSum() UsageBillableSummaryBody {
	if o == nil || o.TimeseriesSum == nil {
		var ret UsageBillableSummaryBody
		return ret
	}
	return *o.TimeseriesSum
}

// GetTimeseriesSumOk returns a tuple with the TimeseriesSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageBillableSummaryKeys) GetTimeseriesSumOk() (*UsageBillableSummaryBody, bool) {
	if o == nil || o.TimeseriesSum == nil {
		return nil, false
	}
	return o.TimeseriesSum, true
}

// HasTimeseriesSum returns a boolean if a field has been set.
func (o *UsageBillableSummaryKeys) HasTimeseriesSum() bool {
	if o != nil && o.TimeseriesSum != nil {
		return true
	}

	return false
}

// SetTimeseriesSum gets a reference to the given UsageBillableSummaryBody and assigns it to the TimeseriesSum field.
func (o *UsageBillableSummaryKeys) SetTimeseriesSum(v UsageBillableSummaryBody) {
	o.TimeseriesSum = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageBillableSummaryKeys) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.ApmFargateAverage != nil {
		toSerialize["apm_fargate_average"] = o.ApmFargateAverage
	}
	if o.ApmFargateSum != nil {
		toSerialize["apm_fargate_sum"] = o.ApmFargateSum
	}
	if o.ApmHostSum != nil {
		toSerialize["apm_host_sum"] = o.ApmHostSum
	}
	if o.ApmHostTop99p != nil {
		toSerialize["apm_host_top99p"] = o.ApmHostTop99p
	}
	if o.ApmProfilerHostSum != nil {
		toSerialize["apm_profiler_host_sum"] = o.ApmProfilerHostSum
	}
	if o.ApmProfilerHostTop99p != nil {
		toSerialize["apm_profiler_host_top99p"] = o.ApmProfilerHostTop99p
	}
	if o.ApmTraceSearchSum != nil {
		toSerialize["apm_trace_search_sum"] = o.ApmTraceSearchSum
	}
	if o.ApplicationSecurityHostSum != nil {
		toSerialize["application_security_host_sum"] = o.ApplicationSecurityHostSum
	}
	if o.CiPipelineIndexedSpansSum != nil {
		toSerialize["ci_pipeline_indexed_spans_sum"] = o.CiPipelineIndexedSpansSum
	}
	if o.CiPipelineMaximum != nil {
		toSerialize["ci_pipeline_maximum"] = o.CiPipelineMaximum
	}
	if o.CiPipelineSum != nil {
		toSerialize["ci_pipeline_sum"] = o.CiPipelineSum
	}
	if o.CiTestIndexedSpansSum != nil {
		toSerialize["ci_test_indexed_spans_sum"] = o.CiTestIndexedSpansSum
	}
	if o.CiTestingMaximum != nil {
		toSerialize["ci_testing_maximum"] = o.CiTestingMaximum
	}
	if o.CiTestingSum != nil {
		toSerialize["ci_testing_sum"] = o.CiTestingSum
	}
	if o.CspmContainerSum != nil {
		toSerialize["cspm_container_sum"] = o.CspmContainerSum
	}
	if o.CspmHostSum != nil {
		toSerialize["cspm_host_sum"] = o.CspmHostSum
	}
	if o.CspmHostTop99p != nil {
		toSerialize["cspm_host_top99p"] = o.CspmHostTop99p
	}
	if o.CustomEventSum != nil {
		toSerialize["custom_event_sum"] = o.CustomEventSum
	}
	if o.CwsContainerSum != nil {
		toSerialize["cws_container_sum"] = o.CwsContainerSum
	}
	if o.CwsHostSum != nil {
		toSerialize["cws_host_sum"] = o.CwsHostSum
	}
	if o.CwsHostTop99p != nil {
		toSerialize["cws_host_top99p"] = o.CwsHostTop99p
	}
	if o.DbmHostSum != nil {
		toSerialize["dbm_host_sum"] = o.DbmHostSum
	}
	if o.DbmHostTop99p != nil {
		toSerialize["dbm_host_top99p"] = o.DbmHostTop99p
	}
	if o.DbmNormalizedQueriesAverage != nil {
		toSerialize["dbm_normalized_queries_average"] = o.DbmNormalizedQueriesAverage
	}
	if o.DbmNormalizedQueriesSum != nil {
		toSerialize["dbm_normalized_queries_sum"] = o.DbmNormalizedQueriesSum
	}
	if o.FargateContainerApmAndProfilerAverage != nil {
		toSerialize["fargate_container_apm_and_profiler_average"] = o.FargateContainerApmAndProfilerAverage
	}
	if o.FargateContainerApmAndProfilerSum != nil {
		toSerialize["fargate_container_apm_and_profiler_sum"] = o.FargateContainerApmAndProfilerSum
	}
	if o.FargateContainerAverage != nil {
		toSerialize["fargate_container_average"] = o.FargateContainerAverage
	}
	if o.FargateContainerProfilerAverage != nil {
		toSerialize["fargate_container_profiler_average"] = o.FargateContainerProfilerAverage
	}
	if o.FargateContainerProfilerSum != nil {
		toSerialize["fargate_container_profiler_sum"] = o.FargateContainerProfilerSum
	}
	if o.FargateContainerSum != nil {
		toSerialize["fargate_container_sum"] = o.FargateContainerSum
	}
	if o.IncidentManagementMaximum != nil {
		toSerialize["incident_management_maximum"] = o.IncidentManagementMaximum
	}
	if o.IncidentManagementSum != nil {
		toSerialize["incident_management_sum"] = o.IncidentManagementSum
	}
	if o.InfraAndApmHostSum != nil {
		toSerialize["infra_and_apm_host_sum"] = o.InfraAndApmHostSum
	}
	if o.InfraAndApmHostTop99p != nil {
		toSerialize["infra_and_apm_host_top99p"] = o.InfraAndApmHostTop99p
	}
	if o.InfraContainerSum != nil {
		toSerialize["infra_container_sum"] = o.InfraContainerSum
	}
	if o.InfraHostSum != nil {
		toSerialize["infra_host_sum"] = o.InfraHostSum
	}
	if o.InfraHostTop99p != nil {
		toSerialize["infra_host_top99p"] = o.InfraHostTop99p
	}
	if o.IngestedSpansSum != nil {
		toSerialize["ingested_spans_sum"] = o.IngestedSpansSum
	}
	if o.IngestedTimeseriesAverage != nil {
		toSerialize["ingested_timeseries_average"] = o.IngestedTimeseriesAverage
	}
	if o.IngestedTimeseriesSum != nil {
		toSerialize["ingested_timeseries_sum"] = o.IngestedTimeseriesSum
	}
	if o.IotSum != nil {
		toSerialize["iot_sum"] = o.IotSum
	}
	if o.IotTop99p != nil {
		toSerialize["iot_top99p"] = o.IotTop99p
	}
	if o.LambdaFunctionAverage != nil {
		toSerialize["lambda_function_average"] = o.LambdaFunctionAverage
	}
	if o.LambdaFunctionSum != nil {
		toSerialize["lambda_function_sum"] = o.LambdaFunctionSum
	}
	if o.LogsIndexed15daySum != nil {
		toSerialize["logs_indexed_15day_sum"] = o.LogsIndexed15daySum
	}
	if o.LogsIndexed180daySum != nil {
		toSerialize["logs_indexed_180day_sum"] = o.LogsIndexed180daySum
	}
	if o.LogsIndexed30daySum != nil {
		toSerialize["logs_indexed_30day_sum"] = o.LogsIndexed30daySum
	}
	if o.LogsIndexed360daySum != nil {
		toSerialize["logs_indexed_360day_sum"] = o.LogsIndexed360daySum
	}
	if o.LogsIndexed3daySum != nil {
		toSerialize["logs_indexed_3day_sum"] = o.LogsIndexed3daySum
	}
	if o.LogsIndexed45daySum != nil {
		toSerialize["logs_indexed_45day_sum"] = o.LogsIndexed45daySum
	}
	if o.LogsIndexed60daySum != nil {
		toSerialize["logs_indexed_60day_sum"] = o.LogsIndexed60daySum
	}
	if o.LogsIndexed7daySum != nil {
		toSerialize["logs_indexed_7day_sum"] = o.LogsIndexed7daySum
	}
	if o.LogsIndexed90daySum != nil {
		toSerialize["logs_indexed_90day_sum"] = o.LogsIndexed90daySum
	}
	if o.LogsIndexedCustomRetentionSum != nil {
		toSerialize["logs_indexed_custom_retention_sum"] = o.LogsIndexedCustomRetentionSum
	}
	if o.LogsIndexedSum != nil {
		toSerialize["logs_indexed_sum"] = o.LogsIndexedSum
	}
	if o.LogsIngestedSum != nil {
		toSerialize["logs_ingested_sum"] = o.LogsIngestedSum
	}
	if o.NetworkDeviceSum != nil {
		toSerialize["network_device_sum"] = o.NetworkDeviceSum
	}
	if o.NetworkDeviceTop99p != nil {
		toSerialize["network_device_top99p"] = o.NetworkDeviceTop99p
	}
	if o.NpmFlowSum != nil {
		toSerialize["npm_flow_sum"] = o.NpmFlowSum
	}
	if o.NpmHostSum != nil {
		toSerialize["npm_host_sum"] = o.NpmHostSum
	}
	if o.NpmHostTop99p != nil {
		toSerialize["npm_host_top99p"] = o.NpmHostTop99p
	}
	if o.ObservabilityPipelineSum != nil {
		toSerialize["observability_pipeline_sum"] = o.ObservabilityPipelineSum
	}
	if o.OnlineArchiveSum != nil {
		toSerialize["online_archive_sum"] = o.OnlineArchiveSum
	}
	if o.ProfContainerSum != nil {
		toSerialize["prof_container_sum"] = o.ProfContainerSum
	}
	if o.ProfHostSum != nil {
		toSerialize["prof_host_sum"] = o.ProfHostSum
	}
	if o.ProfHostTop99p != nil {
		toSerialize["prof_host_top99p"] = o.ProfHostTop99p
	}
	if o.RumLiteSum != nil {
		toSerialize["rum_lite_sum"] = o.RumLiteSum
	}
	if o.RumReplaySum != nil {
		toSerialize["rum_replay_sum"] = o.RumReplaySum
	}
	if o.RumSum != nil {
		toSerialize["rum_sum"] = o.RumSum
	}
	if o.RumUnitsSum != nil {
		toSerialize["rum_units_sum"] = o.RumUnitsSum
	}
	if o.SensitiveDataScannerSum != nil {
		toSerialize["sensitive_data_scanner_sum"] = o.SensitiveDataScannerSum
	}
	if o.ServerlessInvocationSum != nil {
		toSerialize["serverless_invocation_sum"] = o.ServerlessInvocationSum
	}
	if o.SiemSum != nil {
		toSerialize["siem_sum"] = o.SiemSum
	}
	if o.StandardTimeseriesAverage != nil {
		toSerialize["standard_timeseries_average"] = o.StandardTimeseriesAverage
	}
	if o.SyntheticsApiTestsSum != nil {
		toSerialize["synthetics_api_tests_sum"] = o.SyntheticsApiTestsSum
	}
	if o.SyntheticsBrowserChecksSum != nil {
		toSerialize["synthetics_browser_checks_sum"] = o.SyntheticsBrowserChecksSum
	}
	if o.TimeseriesAverage != nil {
		toSerialize["timeseries_average"] = o.TimeseriesAverage
	}
	if o.TimeseriesSum != nil {
		toSerialize["timeseries_sum"] = o.TimeseriesSum
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UsageBillableSummaryKeys) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		ApmFargateAverage                     *UsageBillableSummaryBody `json:"apm_fargate_average,omitempty"`
		ApmFargateSum                         *UsageBillableSummaryBody `json:"apm_fargate_sum,omitempty"`
		ApmHostSum                            *UsageBillableSummaryBody `json:"apm_host_sum,omitempty"`
		ApmHostTop99p                         *UsageBillableSummaryBody `json:"apm_host_top99p,omitempty"`
		ApmProfilerHostSum                    *UsageBillableSummaryBody `json:"apm_profiler_host_sum,omitempty"`
		ApmProfilerHostTop99p                 *UsageBillableSummaryBody `json:"apm_profiler_host_top99p,omitempty"`
		ApmTraceSearchSum                     *UsageBillableSummaryBody `json:"apm_trace_search_sum,omitempty"`
		ApplicationSecurityHostSum            *UsageBillableSummaryBody `json:"application_security_host_sum,omitempty"`
		CiPipelineIndexedSpansSum             *UsageBillableSummaryBody `json:"ci_pipeline_indexed_spans_sum,omitempty"`
		CiPipelineMaximum                     *UsageBillableSummaryBody `json:"ci_pipeline_maximum,omitempty"`
		CiPipelineSum                         *UsageBillableSummaryBody `json:"ci_pipeline_sum,omitempty"`
		CiTestIndexedSpansSum                 *UsageBillableSummaryBody `json:"ci_test_indexed_spans_sum,omitempty"`
		CiTestingMaximum                      *UsageBillableSummaryBody `json:"ci_testing_maximum,omitempty"`
		CiTestingSum                          *UsageBillableSummaryBody `json:"ci_testing_sum,omitempty"`
		CspmContainerSum                      *UsageBillableSummaryBody `json:"cspm_container_sum,omitempty"`
		CspmHostSum                           *UsageBillableSummaryBody `json:"cspm_host_sum,omitempty"`
		CspmHostTop99p                        *UsageBillableSummaryBody `json:"cspm_host_top99p,omitempty"`
		CustomEventSum                        *UsageBillableSummaryBody `json:"custom_event_sum,omitempty"`
		CwsContainerSum                       *UsageBillableSummaryBody `json:"cws_container_sum,omitempty"`
		CwsHostSum                            *UsageBillableSummaryBody `json:"cws_host_sum,omitempty"`
		CwsHostTop99p                         *UsageBillableSummaryBody `json:"cws_host_top99p,omitempty"`
		DbmHostSum                            *UsageBillableSummaryBody `json:"dbm_host_sum,omitempty"`
		DbmHostTop99p                         *UsageBillableSummaryBody `json:"dbm_host_top99p,omitempty"`
		DbmNormalizedQueriesAverage           *UsageBillableSummaryBody `json:"dbm_normalized_queries_average,omitempty"`
		DbmNormalizedQueriesSum               *UsageBillableSummaryBody `json:"dbm_normalized_queries_sum,omitempty"`
		FargateContainerApmAndProfilerAverage *UsageBillableSummaryBody `json:"fargate_container_apm_and_profiler_average,omitempty"`
		FargateContainerApmAndProfilerSum     *UsageBillableSummaryBody `json:"fargate_container_apm_and_profiler_sum,omitempty"`
		FargateContainerAverage               *UsageBillableSummaryBody `json:"fargate_container_average,omitempty"`
		FargateContainerProfilerAverage       *UsageBillableSummaryBody `json:"fargate_container_profiler_average,omitempty"`
		FargateContainerProfilerSum           *UsageBillableSummaryBody `json:"fargate_container_profiler_sum,omitempty"`
		FargateContainerSum                   *UsageBillableSummaryBody `json:"fargate_container_sum,omitempty"`
		IncidentManagementMaximum             *UsageBillableSummaryBody `json:"incident_management_maximum,omitempty"`
		IncidentManagementSum                 *UsageBillableSummaryBody `json:"incident_management_sum,omitempty"`
		InfraAndApmHostSum                    *UsageBillableSummaryBody `json:"infra_and_apm_host_sum,omitempty"`
		InfraAndApmHostTop99p                 *UsageBillableSummaryBody `json:"infra_and_apm_host_top99p,omitempty"`
		InfraContainerSum                     *UsageBillableSummaryBody `json:"infra_container_sum,omitempty"`
		InfraHostSum                          *UsageBillableSummaryBody `json:"infra_host_sum,omitempty"`
		InfraHostTop99p                       *UsageBillableSummaryBody `json:"infra_host_top99p,omitempty"`
		IngestedSpansSum                      *UsageBillableSummaryBody `json:"ingested_spans_sum,omitempty"`
		IngestedTimeseriesAverage             *UsageBillableSummaryBody `json:"ingested_timeseries_average,omitempty"`
		IngestedTimeseriesSum                 *UsageBillableSummaryBody `json:"ingested_timeseries_sum,omitempty"`
		IotSum                                *UsageBillableSummaryBody `json:"iot_sum,omitempty"`
		IotTop99p                             *UsageBillableSummaryBody `json:"iot_top99p,omitempty"`
		LambdaFunctionAverage                 *UsageBillableSummaryBody `json:"lambda_function_average,omitempty"`
		LambdaFunctionSum                     *UsageBillableSummaryBody `json:"lambda_function_sum,omitempty"`
		LogsIndexed15daySum                   *UsageBillableSummaryBody `json:"logs_indexed_15day_sum,omitempty"`
		LogsIndexed180daySum                  *UsageBillableSummaryBody `json:"logs_indexed_180day_sum,omitempty"`
		LogsIndexed30daySum                   *UsageBillableSummaryBody `json:"logs_indexed_30day_sum,omitempty"`
		LogsIndexed360daySum                  *UsageBillableSummaryBody `json:"logs_indexed_360day_sum,omitempty"`
		LogsIndexed3daySum                    *UsageBillableSummaryBody `json:"logs_indexed_3day_sum,omitempty"`
		LogsIndexed45daySum                   *UsageBillableSummaryBody `json:"logs_indexed_45day_sum,omitempty"`
		LogsIndexed60daySum                   *UsageBillableSummaryBody `json:"logs_indexed_60day_sum,omitempty"`
		LogsIndexed7daySum                    *UsageBillableSummaryBody `json:"logs_indexed_7day_sum,omitempty"`
		LogsIndexed90daySum                   *UsageBillableSummaryBody `json:"logs_indexed_90day_sum,omitempty"`
		LogsIndexedCustomRetentionSum         *UsageBillableSummaryBody `json:"logs_indexed_custom_retention_sum,omitempty"`
		LogsIndexedSum                        *UsageBillableSummaryBody `json:"logs_indexed_sum,omitempty"`
		LogsIngestedSum                       *UsageBillableSummaryBody `json:"logs_ingested_sum,omitempty"`
		NetworkDeviceSum                      *UsageBillableSummaryBody `json:"network_device_sum,omitempty"`
		NetworkDeviceTop99p                   *UsageBillableSummaryBody `json:"network_device_top99p,omitempty"`
		NpmFlowSum                            *UsageBillableSummaryBody `json:"npm_flow_sum,omitempty"`
		NpmHostSum                            *UsageBillableSummaryBody `json:"npm_host_sum,omitempty"`
		NpmHostTop99p                         *UsageBillableSummaryBody `json:"npm_host_top99p,omitempty"`
		ObservabilityPipelineSum              *UsageBillableSummaryBody `json:"observability_pipeline_sum,omitempty"`
		OnlineArchiveSum                      *UsageBillableSummaryBody `json:"online_archive_sum,omitempty"`
		ProfContainerSum                      *UsageBillableSummaryBody `json:"prof_container_sum,omitempty"`
		ProfHostSum                           *UsageBillableSummaryBody `json:"prof_host_sum,omitempty"`
		ProfHostTop99p                        *UsageBillableSummaryBody `json:"prof_host_top99p,omitempty"`
		RumLiteSum                            *UsageBillableSummaryBody `json:"rum_lite_sum,omitempty"`
		RumReplaySum                          *UsageBillableSummaryBody `json:"rum_replay_sum,omitempty"`
		RumSum                                *UsageBillableSummaryBody `json:"rum_sum,omitempty"`
		RumUnitsSum                           *UsageBillableSummaryBody `json:"rum_units_sum,omitempty"`
		SensitiveDataScannerSum               *UsageBillableSummaryBody `json:"sensitive_data_scanner_sum,omitempty"`
		ServerlessInvocationSum               *UsageBillableSummaryBody `json:"serverless_invocation_sum,omitempty"`
		SiemSum                               *UsageBillableSummaryBody `json:"siem_sum,omitempty"`
		StandardTimeseriesAverage             *UsageBillableSummaryBody `json:"standard_timeseries_average,omitempty"`
		SyntheticsApiTestsSum                 *UsageBillableSummaryBody `json:"synthetics_api_tests_sum,omitempty"`
		SyntheticsBrowserChecksSum            *UsageBillableSummaryBody `json:"synthetics_browser_checks_sum,omitempty"`
		TimeseriesAverage                     *UsageBillableSummaryBody `json:"timeseries_average,omitempty"`
		TimeseriesSum                         *UsageBillableSummaryBody `json:"timeseries_sum,omitempty"`
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
	if all.ApmFargateAverage != nil && all.ApmFargateAverage.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ApmFargateAverage = all.ApmFargateAverage
	if all.ApmFargateSum != nil && all.ApmFargateSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ApmFargateSum = all.ApmFargateSum
	if all.ApmHostSum != nil && all.ApmHostSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ApmHostSum = all.ApmHostSum
	if all.ApmHostTop99p != nil && all.ApmHostTop99p.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ApmHostTop99p = all.ApmHostTop99p
	if all.ApmProfilerHostSum != nil && all.ApmProfilerHostSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ApmProfilerHostSum = all.ApmProfilerHostSum
	if all.ApmProfilerHostTop99p != nil && all.ApmProfilerHostTop99p.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ApmProfilerHostTop99p = all.ApmProfilerHostTop99p
	if all.ApmTraceSearchSum != nil && all.ApmTraceSearchSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ApmTraceSearchSum = all.ApmTraceSearchSum
	if all.ApplicationSecurityHostSum != nil && all.ApplicationSecurityHostSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ApplicationSecurityHostSum = all.ApplicationSecurityHostSum
	if all.CiPipelineIndexedSpansSum != nil && all.CiPipelineIndexedSpansSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.CiPipelineIndexedSpansSum = all.CiPipelineIndexedSpansSum
	if all.CiPipelineMaximum != nil && all.CiPipelineMaximum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.CiPipelineMaximum = all.CiPipelineMaximum
	if all.CiPipelineSum != nil && all.CiPipelineSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.CiPipelineSum = all.CiPipelineSum
	if all.CiTestIndexedSpansSum != nil && all.CiTestIndexedSpansSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.CiTestIndexedSpansSum = all.CiTestIndexedSpansSum
	if all.CiTestingMaximum != nil && all.CiTestingMaximum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.CiTestingMaximum = all.CiTestingMaximum
	if all.CiTestingSum != nil && all.CiTestingSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.CiTestingSum = all.CiTestingSum
	if all.CspmContainerSum != nil && all.CspmContainerSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.CspmContainerSum = all.CspmContainerSum
	if all.CspmHostSum != nil && all.CspmHostSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.CspmHostSum = all.CspmHostSum
	if all.CspmHostTop99p != nil && all.CspmHostTop99p.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.CspmHostTop99p = all.CspmHostTop99p
	if all.CustomEventSum != nil && all.CustomEventSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.CustomEventSum = all.CustomEventSum
	if all.CwsContainerSum != nil && all.CwsContainerSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.CwsContainerSum = all.CwsContainerSum
	if all.CwsHostSum != nil && all.CwsHostSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.CwsHostSum = all.CwsHostSum
	if all.CwsHostTop99p != nil && all.CwsHostTop99p.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.CwsHostTop99p = all.CwsHostTop99p
	if all.DbmHostSum != nil && all.DbmHostSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.DbmHostSum = all.DbmHostSum
	if all.DbmHostTop99p != nil && all.DbmHostTop99p.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.DbmHostTop99p = all.DbmHostTop99p
	if all.DbmNormalizedQueriesAverage != nil && all.DbmNormalizedQueriesAverage.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.DbmNormalizedQueriesAverage = all.DbmNormalizedQueriesAverage
	if all.DbmNormalizedQueriesSum != nil && all.DbmNormalizedQueriesSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.DbmNormalizedQueriesSum = all.DbmNormalizedQueriesSum
	if all.FargateContainerApmAndProfilerAverage != nil && all.FargateContainerApmAndProfilerAverage.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.FargateContainerApmAndProfilerAverage = all.FargateContainerApmAndProfilerAverage
	if all.FargateContainerApmAndProfilerSum != nil && all.FargateContainerApmAndProfilerSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.FargateContainerApmAndProfilerSum = all.FargateContainerApmAndProfilerSum
	if all.FargateContainerAverage != nil && all.FargateContainerAverage.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.FargateContainerAverage = all.FargateContainerAverage
	if all.FargateContainerProfilerAverage != nil && all.FargateContainerProfilerAverage.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.FargateContainerProfilerAverage = all.FargateContainerProfilerAverage
	if all.FargateContainerProfilerSum != nil && all.FargateContainerProfilerSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.FargateContainerProfilerSum = all.FargateContainerProfilerSum
	if all.FargateContainerSum != nil && all.FargateContainerSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.FargateContainerSum = all.FargateContainerSum
	if all.IncidentManagementMaximum != nil && all.IncidentManagementMaximum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.IncidentManagementMaximum = all.IncidentManagementMaximum
	if all.IncidentManagementSum != nil && all.IncidentManagementSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.IncidentManagementSum = all.IncidentManagementSum
	if all.InfraAndApmHostSum != nil && all.InfraAndApmHostSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.InfraAndApmHostSum = all.InfraAndApmHostSum
	if all.InfraAndApmHostTop99p != nil && all.InfraAndApmHostTop99p.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.InfraAndApmHostTop99p = all.InfraAndApmHostTop99p
	if all.InfraContainerSum != nil && all.InfraContainerSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.InfraContainerSum = all.InfraContainerSum
	if all.InfraHostSum != nil && all.InfraHostSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.InfraHostSum = all.InfraHostSum
	if all.InfraHostTop99p != nil && all.InfraHostTop99p.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.InfraHostTop99p = all.InfraHostTop99p
	if all.IngestedSpansSum != nil && all.IngestedSpansSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.IngestedSpansSum = all.IngestedSpansSum
	if all.IngestedTimeseriesAverage != nil && all.IngestedTimeseriesAverage.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.IngestedTimeseriesAverage = all.IngestedTimeseriesAverage
	if all.IngestedTimeseriesSum != nil && all.IngestedTimeseriesSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.IngestedTimeseriesSum = all.IngestedTimeseriesSum
	if all.IotSum != nil && all.IotSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.IotSum = all.IotSum
	if all.IotTop99p != nil && all.IotTop99p.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.IotTop99p = all.IotTop99p
	if all.LambdaFunctionAverage != nil && all.LambdaFunctionAverage.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.LambdaFunctionAverage = all.LambdaFunctionAverage
	if all.LambdaFunctionSum != nil && all.LambdaFunctionSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.LambdaFunctionSum = all.LambdaFunctionSum
	if all.LogsIndexed15daySum != nil && all.LogsIndexed15daySum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.LogsIndexed15daySum = all.LogsIndexed15daySum
	if all.LogsIndexed180daySum != nil && all.LogsIndexed180daySum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.LogsIndexed180daySum = all.LogsIndexed180daySum
	if all.LogsIndexed30daySum != nil && all.LogsIndexed30daySum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.LogsIndexed30daySum = all.LogsIndexed30daySum
	if all.LogsIndexed360daySum != nil && all.LogsIndexed360daySum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.LogsIndexed360daySum = all.LogsIndexed360daySum
	if all.LogsIndexed3daySum != nil && all.LogsIndexed3daySum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.LogsIndexed3daySum = all.LogsIndexed3daySum
	if all.LogsIndexed45daySum != nil && all.LogsIndexed45daySum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.LogsIndexed45daySum = all.LogsIndexed45daySum
	if all.LogsIndexed60daySum != nil && all.LogsIndexed60daySum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.LogsIndexed60daySum = all.LogsIndexed60daySum
	if all.LogsIndexed7daySum != nil && all.LogsIndexed7daySum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.LogsIndexed7daySum = all.LogsIndexed7daySum
	if all.LogsIndexed90daySum != nil && all.LogsIndexed90daySum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.LogsIndexed90daySum = all.LogsIndexed90daySum
	if all.LogsIndexedCustomRetentionSum != nil && all.LogsIndexedCustomRetentionSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.LogsIndexedCustomRetentionSum = all.LogsIndexedCustomRetentionSum
	if all.LogsIndexedSum != nil && all.LogsIndexedSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.LogsIndexedSum = all.LogsIndexedSum
	if all.LogsIngestedSum != nil && all.LogsIngestedSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.LogsIngestedSum = all.LogsIngestedSum
	if all.NetworkDeviceSum != nil && all.NetworkDeviceSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.NetworkDeviceSum = all.NetworkDeviceSum
	if all.NetworkDeviceTop99p != nil && all.NetworkDeviceTop99p.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.NetworkDeviceTop99p = all.NetworkDeviceTop99p
	if all.NpmFlowSum != nil && all.NpmFlowSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.NpmFlowSum = all.NpmFlowSum
	if all.NpmHostSum != nil && all.NpmHostSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.NpmHostSum = all.NpmHostSum
	if all.NpmHostTop99p != nil && all.NpmHostTop99p.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.NpmHostTop99p = all.NpmHostTop99p
	if all.ObservabilityPipelineSum != nil && all.ObservabilityPipelineSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ObservabilityPipelineSum = all.ObservabilityPipelineSum
	if all.OnlineArchiveSum != nil && all.OnlineArchiveSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.OnlineArchiveSum = all.OnlineArchiveSum
	if all.ProfContainerSum != nil && all.ProfContainerSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ProfContainerSum = all.ProfContainerSum
	if all.ProfHostSum != nil && all.ProfHostSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ProfHostSum = all.ProfHostSum
	if all.ProfHostTop99p != nil && all.ProfHostTop99p.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ProfHostTop99p = all.ProfHostTop99p
	if all.RumLiteSum != nil && all.RumLiteSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.RumLiteSum = all.RumLiteSum
	if all.RumReplaySum != nil && all.RumReplaySum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.RumReplaySum = all.RumReplaySum
	if all.RumSum != nil && all.RumSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.RumSum = all.RumSum
	if all.RumUnitsSum != nil && all.RumUnitsSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.RumUnitsSum = all.RumUnitsSum
	if all.SensitiveDataScannerSum != nil && all.SensitiveDataScannerSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.SensitiveDataScannerSum = all.SensitiveDataScannerSum
	if all.ServerlessInvocationSum != nil && all.ServerlessInvocationSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ServerlessInvocationSum = all.ServerlessInvocationSum
	if all.SiemSum != nil && all.SiemSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.SiemSum = all.SiemSum
	if all.StandardTimeseriesAverage != nil && all.StandardTimeseriesAverage.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.StandardTimeseriesAverage = all.StandardTimeseriesAverage
	if all.SyntheticsApiTestsSum != nil && all.SyntheticsApiTestsSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.SyntheticsApiTestsSum = all.SyntheticsApiTestsSum
	if all.SyntheticsBrowserChecksSum != nil && all.SyntheticsBrowserChecksSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.SyntheticsBrowserChecksSum = all.SyntheticsBrowserChecksSum
	if all.TimeseriesAverage != nil && all.TimeseriesAverage.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.TimeseriesAverage = all.TimeseriesAverage
	if all.TimeseriesSum != nil && all.TimeseriesSum.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.TimeseriesSum = all.TimeseriesSum
	return nil
}
