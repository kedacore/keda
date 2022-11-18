// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// UsageSummaryDateOrg Global hourly report of all data billed by Datadog for a given organization.
type UsageSummaryDateOrg struct {
	// Shows the 99th percentile of all agent hosts over all hours in the current date for the given org.
	AgentHostTop99p *int64 `json:"agent_host_top99p,omitempty"`
	// Shows the 99th percentile of all Azure app services using APM over all hours in the current date for the given org.
	ApmAzureAppServiceHostTop99p *int64 `json:"apm_azure_app_service_host_top99p,omitempty"`
	// Shows the 99th percentile of all distinct APM hosts over all hours in the current date for the given org.
	ApmHostTop99p *int64 `json:"apm_host_top99p,omitempty"`
	// Shows the sum of all audit logs lines indexed over all hours in the current date for the given org.
	AuditLogsLinesIndexedSum *int64 `json:"audit_logs_lines_indexed_sum,omitempty"`
	// The average profiled task count for Fargate Profiling.
	AvgProfiledFargateTasks *int64 `json:"avg_profiled_fargate_tasks,omitempty"`
	// Shows the 99th percentile of all AWS hosts over all hours in the current date for the given org.
	AwsHostTop99p *int64 `json:"aws_host_top99p,omitempty"`
	// Shows the sum of all AWS Lambda invocations over all hours in the current date for the given org.
	AwsLambdaFuncCount *int64 `json:"aws_lambda_func_count,omitempty"`
	// Shows the sum of all AWS Lambda invocations over all hours in the current date for the given org.
	AwsLambdaInvocationsSum *int64 `json:"aws_lambda_invocations_sum,omitempty"`
	// Shows the 99th percentile of all Azure app services over all hours in the current date for the given org.
	AzureAppServiceTop99p *int64 `json:"azure_app_service_top99p,omitempty"`
	// Shows the sum of all log bytes ingested over all hours in the current date for the given org.
	BillableIngestedBytesSum *int64 `json:"billable_ingested_bytes_sum,omitempty"`
	// Shows the sum of all browser lite sessions over all hours in the current date for the given org.
	BrowserRumLiteSessionCountSum *int64 `json:"browser_rum_lite_session_count_sum,omitempty"`
	// Shows the sum of all browser replay sessions over all hours in the current date for the given org.
	BrowserRumReplaySessionCountSum *int64 `json:"browser_rum_replay_session_count_sum,omitempty"`
	// Shows the sum of all browser RUM units over all hours in the current date for the given org.
	BrowserRumUnitsSum *int64 `json:"browser_rum_units_sum,omitempty"`
	// Shows the sum of all CI pipeline indexed spans over all hours in the current date for the given org.
	CiPipelineIndexedSpansSum *int64 `json:"ci_pipeline_indexed_spans_sum,omitempty"`
	// Shows the sum of all CI test indexed spans over all hours in the current date for the given org.
	CiTestIndexedSpansSum *int64 `json:"ci_test_indexed_spans_sum,omitempty"`
	// Shows the high-water mark of all CI visibility pipeline committers over all hours in the current date for the given org.
	CiVisibilityPipelineCommittersHwm *int64 `json:"ci_visibility_pipeline_committers_hwm,omitempty"`
	// Shows the high-water mark of all CI visibility test committers over all hours in the current date for the given org.
	CiVisibilityTestCommittersHwm *int64 `json:"ci_visibility_test_committers_hwm,omitempty"`
	// Shows the average of all distinct containers over all hours in the current date for the given org.
	ContainerAvg *int64 `json:"container_avg,omitempty"`
	// Shows the high-water mark of all distinct containers over all hours in the current date for the given org.
	ContainerHwm *int64 `json:"container_hwm,omitempty"`
	// Shows the 99th percentile of all Cloud Security Posture Management Azure app services hosts over all hours in the current date for the given org.
	CspmAasHostTop99p *int64 `json:"cspm_aas_host_top99p,omitempty"`
	// Shows the 99th percentile of all Cloud Security Posture Management Azure hosts over all hours in the current date for the given org.
	CspmAzureHostTop99p *int64 `json:"cspm_azure_host_top99p,omitempty"`
	// Shows the average number of Cloud Security Posture Management containers over all hours in the current date for the given org.
	CspmContainerAvg *int64 `json:"cspm_container_avg,omitempty"`
	// Shows the high-water mark of Cloud Security Posture Management containers over all hours in the current date for the given org.
	CspmContainerHwm *int64 `json:"cspm_container_hwm,omitempty"`
	// Shows the 99th percentile of all Cloud Security Posture Management hosts over all hours in the current date for the given org.
	CspmHostTop99p *int64 `json:"cspm_host_top99p,omitempty"`
	// Shows the average number of distinct custom metrics over all hours in the current date for the given org.
	CustomTsAvg *int64 `json:"custom_ts_avg,omitempty"`
	// Shows the average of all distinct Cloud Workload Security containers over all hours in the current date for the given org.
	CwsContainerCountAvg *int64 `json:"cws_container_count_avg,omitempty"`
	// Shows the 99th percentile of all Cloud Workload Security hosts over all hours in the current date for the given org.
	CwsHostTop99p *int64 `json:"cws_host_top99p,omitempty"`
	// Shows the 99th percentile of all Database Monitoring hosts over all hours in the current month for the given org.
	DbmHostTop99pSum *int64 `json:"dbm_host_top99p_sum,omitempty"`
	// Shows the average of all distinct Database Monitoring normalized queries over all hours in the current month for the given org.
	DbmQueriesAvgSum *int64 `json:"dbm_queries_avg_sum,omitempty"`
	// The average task count for Fargate.
	FargateTasksCountAvg *int64 `json:"fargate_tasks_count_avg,omitempty"`
	// Shows the high-water mark of all Fargate tasks over all hours in the current date for the given org.
	FargateTasksCountHwm *int64 `json:"fargate_tasks_count_hwm,omitempty"`
	// Shows the 99th percentile of all GCP hosts over all hours in the current date for the given org.
	GcpHostTop99p *int64 `json:"gcp_host_top99p,omitempty"`
	// Shows the 99th percentile of all Heroku dynos over all hours in the current date for the given org.
	HerokuHostTop99p *int64 `json:"heroku_host_top99p,omitempty"`
	// The organization id.
	Id *string `json:"id,omitempty"`
	// Shows the high-water mark of incident management monthly active users over all hours in the current date for the given org.
	IncidentManagementMonthlyActiveUsersHwm *int64 `json:"incident_management_monthly_active_users_hwm,omitempty"`
	// Shows the sum of all log events indexed over all hours in the current date for the given org.
	IndexedEventsCountSum *int64 `json:"indexed_events_count_sum,omitempty"`
	// Shows the 99th percentile of all distinct infrastructure hosts over all hours in the current date for the given org.
	InfraHostTop99p *int64 `json:"infra_host_top99p,omitempty"`
	// Shows the sum of all log bytes ingested over all hours in the current date for the given org.
	IngestedEventsBytesSum *int64 `json:"ingested_events_bytes_sum,omitempty"`
	// Shows the sum of all IoT devices over all hours in the current date for the given org.
	IotDeviceAggSum *int64 `json:"iot_device_agg_sum,omitempty"`
	// Shows the 99th percentile of all IoT devices over all hours in the current date for the given org.
	IotDeviceTop99pSum *int64 `json:"iot_device_top99p_sum,omitempty"`
	// Shows the sum of all mobile lite sessions over all hours in the current date for the given org.
	MobileRumLiteSessionCountSum *int64 `json:"mobile_rum_lite_session_count_sum,omitempty"`
	// Shows the sum of all mobile RUM Sessions on Android over all hours in the current date for the given org.
	MobileRumSessionCountAndroidSum *int64 `json:"mobile_rum_session_count_android_sum,omitempty"`
	// Shows the sum of all mobile RUM Sessions on iOS over all hours in the current date for the given org.
	MobileRumSessionCountIosSum *int64 `json:"mobile_rum_session_count_ios_sum,omitempty"`
	// Shows the sum of all mobile RUM Sessions on React Native over all hours in the current date for the given org.
	MobileRumSessionCountReactnativeSum *int64 `json:"mobile_rum_session_count_reactnative_sum,omitempty"`
	// Shows the sum of all mobile RUM Sessions over all hours in the current date for the given org.
	MobileRumSessionCountSum *int64 `json:"mobile_rum_session_count_sum,omitempty"`
	// Shows the sum of all mobile RUM units over all hours in the current date for the given org.
	MobileRumUnitsSum *int64 `json:"mobile_rum_units_sum,omitempty"`
	// The organization name.
	Name *string `json:"name,omitempty"`
	// Shows the sum of all Network flows indexed over all hours in the current date for the given org.
	NetflowIndexedEventsCountSum *int64 `json:"netflow_indexed_events_count_sum,omitempty"`
	// Shows the 99th percentile of all distinct Networks hosts over all hours in the current date for the given org.
	NpmHostTop99p *int64 `json:"npm_host_top99p,omitempty"`
	// Sum of all observability pipelines bytes processed over all hours in the current date for the given org.
	ObservabilityPipelinesBytesProcessedSum *int64 `json:"observability_pipelines_bytes_processed_sum,omitempty"`
	// Sum of all online archived events over all hours in the current date for the given org.
	OnlineArchiveEventsCountSum *int64 `json:"online_archive_events_count_sum,omitempty"`
	// Shows the 99th percentile of all hosts reported by the Datadog exporter for the OpenTelemetry Collector over all hours in the current date for the given org.
	OpentelemetryHostTop99p *int64 `json:"opentelemetry_host_top99p,omitempty"`
	// Shows the 99th percentile of all profiled hosts over all hours in the current date for the given org.
	ProfilingHostTop99p *int64 `json:"profiling_host_top99p,omitempty"`
	// The organization public id.
	PublicId *string `json:"public_id,omitempty"`
	// Shows the sum of all mobile sessions and all browser lite and legacy sessions over all hours in the current date for the given org.
	RumBrowserAndMobileSessionCount *int64 `json:"rum_browser_and_mobile_session_count,omitempty"`
	// Shows the sum of all browser RUM Lite Sessions over all hours in the current date for the given org.
	RumSessionCountSum *int64 `json:"rum_session_count_sum,omitempty"`
	// Shows the sum of RUM Sessions (browser and mobile) over all hours in the current date for the given org.
	RumTotalSessionCountSum *int64 `json:"rum_total_session_count_sum,omitempty"`
	// Shows the sum of all browser and mobile RUM units over all hours in the current date for the given org.
	RumUnitsSum *int64 `json:"rum_units_sum,omitempty"`
	// Shows the sum of all bytes scanned of logs usage by the Sensitive Data Scanner over all hours in the current month for the given org.
	SdsLogsScannedBytesSum *int64 `json:"sds_logs_scanned_bytes_sum,omitempty"`
	// Shows the sum of all bytes scanned across all usage types by the Sensitive Data Scanner over all hours in the current month for the given org.
	SdsTotalScannedBytesSum *int64 `json:"sds_total_scanned_bytes_sum,omitempty"`
	// Shows the sum of all Synthetic browser tests over all hours in the current date for the given org.
	SyntheticsBrowserCheckCallsCountSum *int64 `json:"synthetics_browser_check_calls_count_sum,omitempty"`
	// Shows the sum of all Synthetic API tests over all hours in the current date for the given org.
	SyntheticsCheckCallsCountSum *int64 `json:"synthetics_check_calls_count_sum,omitempty"`
	// Shows the sum of all Indexed Spans indexed over all hours in the current date for the given org.
	TraceSearchIndexedEventsCountSum *int64 `json:"trace_search_indexed_events_count_sum,omitempty"`
	// Shows the sum of all ingested APM span bytes over all hours in the current date for the given org.
	TwolIngestedEventsBytesSum *int64 `json:"twol_ingested_events_bytes_sum,omitempty"`
	// Shows the 99th percentile of all vSphere hosts over all hours in the current date for the given org.
	VsphereHostTop99p *int64 `json:"vsphere_host_top99p,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:-`
	AdditionalProperties map[string]interface{}
}

// NewUsageSummaryDateOrg instantiates a new UsageSummaryDateOrg object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageSummaryDateOrg() *UsageSummaryDateOrg {
	this := UsageSummaryDateOrg{}
	return &this
}

// NewUsageSummaryDateOrgWithDefaults instantiates a new UsageSummaryDateOrg object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageSummaryDateOrgWithDefaults() *UsageSummaryDateOrg {
	this := UsageSummaryDateOrg{}
	return &this
}

// GetAgentHostTop99p returns the AgentHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetAgentHostTop99p() int64 {
	if o == nil || o.AgentHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.AgentHostTop99p
}

// GetAgentHostTop99pOk returns a tuple with the AgentHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetAgentHostTop99pOk() (*int64, bool) {
	if o == nil || o.AgentHostTop99p == nil {
		return nil, false
	}
	return o.AgentHostTop99p, true
}

// HasAgentHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasAgentHostTop99p() bool {
	if o != nil && o.AgentHostTop99p != nil {
		return true
	}

	return false
}

// SetAgentHostTop99p gets a reference to the given int64 and assigns it to the AgentHostTop99p field.
func (o *UsageSummaryDateOrg) SetAgentHostTop99p(v int64) {
	o.AgentHostTop99p = &v
}

// GetApmAzureAppServiceHostTop99p returns the ApmAzureAppServiceHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetApmAzureAppServiceHostTop99p() int64 {
	if o == nil || o.ApmAzureAppServiceHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.ApmAzureAppServiceHostTop99p
}

// GetApmAzureAppServiceHostTop99pOk returns a tuple with the ApmAzureAppServiceHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetApmAzureAppServiceHostTop99pOk() (*int64, bool) {
	if o == nil || o.ApmAzureAppServiceHostTop99p == nil {
		return nil, false
	}
	return o.ApmAzureAppServiceHostTop99p, true
}

// HasApmAzureAppServiceHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasApmAzureAppServiceHostTop99p() bool {
	if o != nil && o.ApmAzureAppServiceHostTop99p != nil {
		return true
	}

	return false
}

// SetApmAzureAppServiceHostTop99p gets a reference to the given int64 and assigns it to the ApmAzureAppServiceHostTop99p field.
func (o *UsageSummaryDateOrg) SetApmAzureAppServiceHostTop99p(v int64) {
	o.ApmAzureAppServiceHostTop99p = &v
}

// GetApmHostTop99p returns the ApmHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetApmHostTop99p() int64 {
	if o == nil || o.ApmHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.ApmHostTop99p
}

// GetApmHostTop99pOk returns a tuple with the ApmHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetApmHostTop99pOk() (*int64, bool) {
	if o == nil || o.ApmHostTop99p == nil {
		return nil, false
	}
	return o.ApmHostTop99p, true
}

// HasApmHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasApmHostTop99p() bool {
	if o != nil && o.ApmHostTop99p != nil {
		return true
	}

	return false
}

// SetApmHostTop99p gets a reference to the given int64 and assigns it to the ApmHostTop99p field.
func (o *UsageSummaryDateOrg) SetApmHostTop99p(v int64) {
	o.ApmHostTop99p = &v
}

// GetAuditLogsLinesIndexedSum returns the AuditLogsLinesIndexedSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetAuditLogsLinesIndexedSum() int64 {
	if o == nil || o.AuditLogsLinesIndexedSum == nil {
		var ret int64
		return ret
	}
	return *o.AuditLogsLinesIndexedSum
}

// GetAuditLogsLinesIndexedSumOk returns a tuple with the AuditLogsLinesIndexedSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetAuditLogsLinesIndexedSumOk() (*int64, bool) {
	if o == nil || o.AuditLogsLinesIndexedSum == nil {
		return nil, false
	}
	return o.AuditLogsLinesIndexedSum, true
}

// HasAuditLogsLinesIndexedSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasAuditLogsLinesIndexedSum() bool {
	if o != nil && o.AuditLogsLinesIndexedSum != nil {
		return true
	}

	return false
}

// SetAuditLogsLinesIndexedSum gets a reference to the given int64 and assigns it to the AuditLogsLinesIndexedSum field.
func (o *UsageSummaryDateOrg) SetAuditLogsLinesIndexedSum(v int64) {
	o.AuditLogsLinesIndexedSum = &v
}

// GetAvgProfiledFargateTasks returns the AvgProfiledFargateTasks field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetAvgProfiledFargateTasks() int64 {
	if o == nil || o.AvgProfiledFargateTasks == nil {
		var ret int64
		return ret
	}
	return *o.AvgProfiledFargateTasks
}

// GetAvgProfiledFargateTasksOk returns a tuple with the AvgProfiledFargateTasks field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetAvgProfiledFargateTasksOk() (*int64, bool) {
	if o == nil || o.AvgProfiledFargateTasks == nil {
		return nil, false
	}
	return o.AvgProfiledFargateTasks, true
}

// HasAvgProfiledFargateTasks returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasAvgProfiledFargateTasks() bool {
	if o != nil && o.AvgProfiledFargateTasks != nil {
		return true
	}

	return false
}

// SetAvgProfiledFargateTasks gets a reference to the given int64 and assigns it to the AvgProfiledFargateTasks field.
func (o *UsageSummaryDateOrg) SetAvgProfiledFargateTasks(v int64) {
	o.AvgProfiledFargateTasks = &v
}

// GetAwsHostTop99p returns the AwsHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetAwsHostTop99p() int64 {
	if o == nil || o.AwsHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.AwsHostTop99p
}

// GetAwsHostTop99pOk returns a tuple with the AwsHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetAwsHostTop99pOk() (*int64, bool) {
	if o == nil || o.AwsHostTop99p == nil {
		return nil, false
	}
	return o.AwsHostTop99p, true
}

// HasAwsHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasAwsHostTop99p() bool {
	if o != nil && o.AwsHostTop99p != nil {
		return true
	}

	return false
}

// SetAwsHostTop99p gets a reference to the given int64 and assigns it to the AwsHostTop99p field.
func (o *UsageSummaryDateOrg) SetAwsHostTop99p(v int64) {
	o.AwsHostTop99p = &v
}

// GetAwsLambdaFuncCount returns the AwsLambdaFuncCount field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetAwsLambdaFuncCount() int64 {
	if o == nil || o.AwsLambdaFuncCount == nil {
		var ret int64
		return ret
	}
	return *o.AwsLambdaFuncCount
}

// GetAwsLambdaFuncCountOk returns a tuple with the AwsLambdaFuncCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetAwsLambdaFuncCountOk() (*int64, bool) {
	if o == nil || o.AwsLambdaFuncCount == nil {
		return nil, false
	}
	return o.AwsLambdaFuncCount, true
}

// HasAwsLambdaFuncCount returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasAwsLambdaFuncCount() bool {
	if o != nil && o.AwsLambdaFuncCount != nil {
		return true
	}

	return false
}

// SetAwsLambdaFuncCount gets a reference to the given int64 and assigns it to the AwsLambdaFuncCount field.
func (o *UsageSummaryDateOrg) SetAwsLambdaFuncCount(v int64) {
	o.AwsLambdaFuncCount = &v
}

// GetAwsLambdaInvocationsSum returns the AwsLambdaInvocationsSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetAwsLambdaInvocationsSum() int64 {
	if o == nil || o.AwsLambdaInvocationsSum == nil {
		var ret int64
		return ret
	}
	return *o.AwsLambdaInvocationsSum
}

// GetAwsLambdaInvocationsSumOk returns a tuple with the AwsLambdaInvocationsSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetAwsLambdaInvocationsSumOk() (*int64, bool) {
	if o == nil || o.AwsLambdaInvocationsSum == nil {
		return nil, false
	}
	return o.AwsLambdaInvocationsSum, true
}

// HasAwsLambdaInvocationsSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasAwsLambdaInvocationsSum() bool {
	if o != nil && o.AwsLambdaInvocationsSum != nil {
		return true
	}

	return false
}

// SetAwsLambdaInvocationsSum gets a reference to the given int64 and assigns it to the AwsLambdaInvocationsSum field.
func (o *UsageSummaryDateOrg) SetAwsLambdaInvocationsSum(v int64) {
	o.AwsLambdaInvocationsSum = &v
}

// GetAzureAppServiceTop99p returns the AzureAppServiceTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetAzureAppServiceTop99p() int64 {
	if o == nil || o.AzureAppServiceTop99p == nil {
		var ret int64
		return ret
	}
	return *o.AzureAppServiceTop99p
}

// GetAzureAppServiceTop99pOk returns a tuple with the AzureAppServiceTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetAzureAppServiceTop99pOk() (*int64, bool) {
	if o == nil || o.AzureAppServiceTop99p == nil {
		return nil, false
	}
	return o.AzureAppServiceTop99p, true
}

// HasAzureAppServiceTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasAzureAppServiceTop99p() bool {
	if o != nil && o.AzureAppServiceTop99p != nil {
		return true
	}

	return false
}

// SetAzureAppServiceTop99p gets a reference to the given int64 and assigns it to the AzureAppServiceTop99p field.
func (o *UsageSummaryDateOrg) SetAzureAppServiceTop99p(v int64) {
	o.AzureAppServiceTop99p = &v
}

// GetBillableIngestedBytesSum returns the BillableIngestedBytesSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetBillableIngestedBytesSum() int64 {
	if o == nil || o.BillableIngestedBytesSum == nil {
		var ret int64
		return ret
	}
	return *o.BillableIngestedBytesSum
}

// GetBillableIngestedBytesSumOk returns a tuple with the BillableIngestedBytesSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetBillableIngestedBytesSumOk() (*int64, bool) {
	if o == nil || o.BillableIngestedBytesSum == nil {
		return nil, false
	}
	return o.BillableIngestedBytesSum, true
}

// HasBillableIngestedBytesSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasBillableIngestedBytesSum() bool {
	if o != nil && o.BillableIngestedBytesSum != nil {
		return true
	}

	return false
}

// SetBillableIngestedBytesSum gets a reference to the given int64 and assigns it to the BillableIngestedBytesSum field.
func (o *UsageSummaryDateOrg) SetBillableIngestedBytesSum(v int64) {
	o.BillableIngestedBytesSum = &v
}

// GetBrowserRumLiteSessionCountSum returns the BrowserRumLiteSessionCountSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetBrowserRumLiteSessionCountSum() int64 {
	if o == nil || o.BrowserRumLiteSessionCountSum == nil {
		var ret int64
		return ret
	}
	return *o.BrowserRumLiteSessionCountSum
}

// GetBrowserRumLiteSessionCountSumOk returns a tuple with the BrowserRumLiteSessionCountSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetBrowserRumLiteSessionCountSumOk() (*int64, bool) {
	if o == nil || o.BrowserRumLiteSessionCountSum == nil {
		return nil, false
	}
	return o.BrowserRumLiteSessionCountSum, true
}

// HasBrowserRumLiteSessionCountSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasBrowserRumLiteSessionCountSum() bool {
	if o != nil && o.BrowserRumLiteSessionCountSum != nil {
		return true
	}

	return false
}

// SetBrowserRumLiteSessionCountSum gets a reference to the given int64 and assigns it to the BrowserRumLiteSessionCountSum field.
func (o *UsageSummaryDateOrg) SetBrowserRumLiteSessionCountSum(v int64) {
	o.BrowserRumLiteSessionCountSum = &v
}

// GetBrowserRumReplaySessionCountSum returns the BrowserRumReplaySessionCountSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetBrowserRumReplaySessionCountSum() int64 {
	if o == nil || o.BrowserRumReplaySessionCountSum == nil {
		var ret int64
		return ret
	}
	return *o.BrowserRumReplaySessionCountSum
}

// GetBrowserRumReplaySessionCountSumOk returns a tuple with the BrowserRumReplaySessionCountSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetBrowserRumReplaySessionCountSumOk() (*int64, bool) {
	if o == nil || o.BrowserRumReplaySessionCountSum == nil {
		return nil, false
	}
	return o.BrowserRumReplaySessionCountSum, true
}

// HasBrowserRumReplaySessionCountSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasBrowserRumReplaySessionCountSum() bool {
	if o != nil && o.BrowserRumReplaySessionCountSum != nil {
		return true
	}

	return false
}

// SetBrowserRumReplaySessionCountSum gets a reference to the given int64 and assigns it to the BrowserRumReplaySessionCountSum field.
func (o *UsageSummaryDateOrg) SetBrowserRumReplaySessionCountSum(v int64) {
	o.BrowserRumReplaySessionCountSum = &v
}

// GetBrowserRumUnitsSum returns the BrowserRumUnitsSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetBrowserRumUnitsSum() int64 {
	if o == nil || o.BrowserRumUnitsSum == nil {
		var ret int64
		return ret
	}
	return *o.BrowserRumUnitsSum
}

// GetBrowserRumUnitsSumOk returns a tuple with the BrowserRumUnitsSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetBrowserRumUnitsSumOk() (*int64, bool) {
	if o == nil || o.BrowserRumUnitsSum == nil {
		return nil, false
	}
	return o.BrowserRumUnitsSum, true
}

// HasBrowserRumUnitsSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasBrowserRumUnitsSum() bool {
	if o != nil && o.BrowserRumUnitsSum != nil {
		return true
	}

	return false
}

// SetBrowserRumUnitsSum gets a reference to the given int64 and assigns it to the BrowserRumUnitsSum field.
func (o *UsageSummaryDateOrg) SetBrowserRumUnitsSum(v int64) {
	o.BrowserRumUnitsSum = &v
}

// GetCiPipelineIndexedSpansSum returns the CiPipelineIndexedSpansSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetCiPipelineIndexedSpansSum() int64 {
	if o == nil || o.CiPipelineIndexedSpansSum == nil {
		var ret int64
		return ret
	}
	return *o.CiPipelineIndexedSpansSum
}

// GetCiPipelineIndexedSpansSumOk returns a tuple with the CiPipelineIndexedSpansSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetCiPipelineIndexedSpansSumOk() (*int64, bool) {
	if o == nil || o.CiPipelineIndexedSpansSum == nil {
		return nil, false
	}
	return o.CiPipelineIndexedSpansSum, true
}

// HasCiPipelineIndexedSpansSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasCiPipelineIndexedSpansSum() bool {
	if o != nil && o.CiPipelineIndexedSpansSum != nil {
		return true
	}

	return false
}

// SetCiPipelineIndexedSpansSum gets a reference to the given int64 and assigns it to the CiPipelineIndexedSpansSum field.
func (o *UsageSummaryDateOrg) SetCiPipelineIndexedSpansSum(v int64) {
	o.CiPipelineIndexedSpansSum = &v
}

// GetCiTestIndexedSpansSum returns the CiTestIndexedSpansSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetCiTestIndexedSpansSum() int64 {
	if o == nil || o.CiTestIndexedSpansSum == nil {
		var ret int64
		return ret
	}
	return *o.CiTestIndexedSpansSum
}

// GetCiTestIndexedSpansSumOk returns a tuple with the CiTestIndexedSpansSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetCiTestIndexedSpansSumOk() (*int64, bool) {
	if o == nil || o.CiTestIndexedSpansSum == nil {
		return nil, false
	}
	return o.CiTestIndexedSpansSum, true
}

// HasCiTestIndexedSpansSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasCiTestIndexedSpansSum() bool {
	if o != nil && o.CiTestIndexedSpansSum != nil {
		return true
	}

	return false
}

// SetCiTestIndexedSpansSum gets a reference to the given int64 and assigns it to the CiTestIndexedSpansSum field.
func (o *UsageSummaryDateOrg) SetCiTestIndexedSpansSum(v int64) {
	o.CiTestIndexedSpansSum = &v
}

// GetCiVisibilityPipelineCommittersHwm returns the CiVisibilityPipelineCommittersHwm field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetCiVisibilityPipelineCommittersHwm() int64 {
	if o == nil || o.CiVisibilityPipelineCommittersHwm == nil {
		var ret int64
		return ret
	}
	return *o.CiVisibilityPipelineCommittersHwm
}

// GetCiVisibilityPipelineCommittersHwmOk returns a tuple with the CiVisibilityPipelineCommittersHwm field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetCiVisibilityPipelineCommittersHwmOk() (*int64, bool) {
	if o == nil || o.CiVisibilityPipelineCommittersHwm == nil {
		return nil, false
	}
	return o.CiVisibilityPipelineCommittersHwm, true
}

// HasCiVisibilityPipelineCommittersHwm returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasCiVisibilityPipelineCommittersHwm() bool {
	if o != nil && o.CiVisibilityPipelineCommittersHwm != nil {
		return true
	}

	return false
}

// SetCiVisibilityPipelineCommittersHwm gets a reference to the given int64 and assigns it to the CiVisibilityPipelineCommittersHwm field.
func (o *UsageSummaryDateOrg) SetCiVisibilityPipelineCommittersHwm(v int64) {
	o.CiVisibilityPipelineCommittersHwm = &v
}

// GetCiVisibilityTestCommittersHwm returns the CiVisibilityTestCommittersHwm field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetCiVisibilityTestCommittersHwm() int64 {
	if o == nil || o.CiVisibilityTestCommittersHwm == nil {
		var ret int64
		return ret
	}
	return *o.CiVisibilityTestCommittersHwm
}

// GetCiVisibilityTestCommittersHwmOk returns a tuple with the CiVisibilityTestCommittersHwm field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetCiVisibilityTestCommittersHwmOk() (*int64, bool) {
	if o == nil || o.CiVisibilityTestCommittersHwm == nil {
		return nil, false
	}
	return o.CiVisibilityTestCommittersHwm, true
}

// HasCiVisibilityTestCommittersHwm returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasCiVisibilityTestCommittersHwm() bool {
	if o != nil && o.CiVisibilityTestCommittersHwm != nil {
		return true
	}

	return false
}

// SetCiVisibilityTestCommittersHwm gets a reference to the given int64 and assigns it to the CiVisibilityTestCommittersHwm field.
func (o *UsageSummaryDateOrg) SetCiVisibilityTestCommittersHwm(v int64) {
	o.CiVisibilityTestCommittersHwm = &v
}

// GetContainerAvg returns the ContainerAvg field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetContainerAvg() int64 {
	if o == nil || o.ContainerAvg == nil {
		var ret int64
		return ret
	}
	return *o.ContainerAvg
}

// GetContainerAvgOk returns a tuple with the ContainerAvg field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetContainerAvgOk() (*int64, bool) {
	if o == nil || o.ContainerAvg == nil {
		return nil, false
	}
	return o.ContainerAvg, true
}

// HasContainerAvg returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasContainerAvg() bool {
	if o != nil && o.ContainerAvg != nil {
		return true
	}

	return false
}

// SetContainerAvg gets a reference to the given int64 and assigns it to the ContainerAvg field.
func (o *UsageSummaryDateOrg) SetContainerAvg(v int64) {
	o.ContainerAvg = &v
}

// GetContainerHwm returns the ContainerHwm field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetContainerHwm() int64 {
	if o == nil || o.ContainerHwm == nil {
		var ret int64
		return ret
	}
	return *o.ContainerHwm
}

// GetContainerHwmOk returns a tuple with the ContainerHwm field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetContainerHwmOk() (*int64, bool) {
	if o == nil || o.ContainerHwm == nil {
		return nil, false
	}
	return o.ContainerHwm, true
}

// HasContainerHwm returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasContainerHwm() bool {
	if o != nil && o.ContainerHwm != nil {
		return true
	}

	return false
}

// SetContainerHwm gets a reference to the given int64 and assigns it to the ContainerHwm field.
func (o *UsageSummaryDateOrg) SetContainerHwm(v int64) {
	o.ContainerHwm = &v
}

// GetCspmAasHostTop99p returns the CspmAasHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetCspmAasHostTop99p() int64 {
	if o == nil || o.CspmAasHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.CspmAasHostTop99p
}

// GetCspmAasHostTop99pOk returns a tuple with the CspmAasHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetCspmAasHostTop99pOk() (*int64, bool) {
	if o == nil || o.CspmAasHostTop99p == nil {
		return nil, false
	}
	return o.CspmAasHostTop99p, true
}

// HasCspmAasHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasCspmAasHostTop99p() bool {
	if o != nil && o.CspmAasHostTop99p != nil {
		return true
	}

	return false
}

// SetCspmAasHostTop99p gets a reference to the given int64 and assigns it to the CspmAasHostTop99p field.
func (o *UsageSummaryDateOrg) SetCspmAasHostTop99p(v int64) {
	o.CspmAasHostTop99p = &v
}

// GetCspmAzureHostTop99p returns the CspmAzureHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetCspmAzureHostTop99p() int64 {
	if o == nil || o.CspmAzureHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.CspmAzureHostTop99p
}

// GetCspmAzureHostTop99pOk returns a tuple with the CspmAzureHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetCspmAzureHostTop99pOk() (*int64, bool) {
	if o == nil || o.CspmAzureHostTop99p == nil {
		return nil, false
	}
	return o.CspmAzureHostTop99p, true
}

// HasCspmAzureHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasCspmAzureHostTop99p() bool {
	if o != nil && o.CspmAzureHostTop99p != nil {
		return true
	}

	return false
}

// SetCspmAzureHostTop99p gets a reference to the given int64 and assigns it to the CspmAzureHostTop99p field.
func (o *UsageSummaryDateOrg) SetCspmAzureHostTop99p(v int64) {
	o.CspmAzureHostTop99p = &v
}

// GetCspmContainerAvg returns the CspmContainerAvg field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetCspmContainerAvg() int64 {
	if o == nil || o.CspmContainerAvg == nil {
		var ret int64
		return ret
	}
	return *o.CspmContainerAvg
}

// GetCspmContainerAvgOk returns a tuple with the CspmContainerAvg field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetCspmContainerAvgOk() (*int64, bool) {
	if o == nil || o.CspmContainerAvg == nil {
		return nil, false
	}
	return o.CspmContainerAvg, true
}

// HasCspmContainerAvg returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasCspmContainerAvg() bool {
	if o != nil && o.CspmContainerAvg != nil {
		return true
	}

	return false
}

// SetCspmContainerAvg gets a reference to the given int64 and assigns it to the CspmContainerAvg field.
func (o *UsageSummaryDateOrg) SetCspmContainerAvg(v int64) {
	o.CspmContainerAvg = &v
}

// GetCspmContainerHwm returns the CspmContainerHwm field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetCspmContainerHwm() int64 {
	if o == nil || o.CspmContainerHwm == nil {
		var ret int64
		return ret
	}
	return *o.CspmContainerHwm
}

// GetCspmContainerHwmOk returns a tuple with the CspmContainerHwm field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetCspmContainerHwmOk() (*int64, bool) {
	if o == nil || o.CspmContainerHwm == nil {
		return nil, false
	}
	return o.CspmContainerHwm, true
}

// HasCspmContainerHwm returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasCspmContainerHwm() bool {
	if o != nil && o.CspmContainerHwm != nil {
		return true
	}

	return false
}

// SetCspmContainerHwm gets a reference to the given int64 and assigns it to the CspmContainerHwm field.
func (o *UsageSummaryDateOrg) SetCspmContainerHwm(v int64) {
	o.CspmContainerHwm = &v
}

// GetCspmHostTop99p returns the CspmHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetCspmHostTop99p() int64 {
	if o == nil || o.CspmHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.CspmHostTop99p
}

// GetCspmHostTop99pOk returns a tuple with the CspmHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetCspmHostTop99pOk() (*int64, bool) {
	if o == nil || o.CspmHostTop99p == nil {
		return nil, false
	}
	return o.CspmHostTop99p, true
}

// HasCspmHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasCspmHostTop99p() bool {
	if o != nil && o.CspmHostTop99p != nil {
		return true
	}

	return false
}

// SetCspmHostTop99p gets a reference to the given int64 and assigns it to the CspmHostTop99p field.
func (o *UsageSummaryDateOrg) SetCspmHostTop99p(v int64) {
	o.CspmHostTop99p = &v
}

// GetCustomTsAvg returns the CustomTsAvg field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetCustomTsAvg() int64 {
	if o == nil || o.CustomTsAvg == nil {
		var ret int64
		return ret
	}
	return *o.CustomTsAvg
}

// GetCustomTsAvgOk returns a tuple with the CustomTsAvg field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetCustomTsAvgOk() (*int64, bool) {
	if o == nil || o.CustomTsAvg == nil {
		return nil, false
	}
	return o.CustomTsAvg, true
}

// HasCustomTsAvg returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasCustomTsAvg() bool {
	if o != nil && o.CustomTsAvg != nil {
		return true
	}

	return false
}

// SetCustomTsAvg gets a reference to the given int64 and assigns it to the CustomTsAvg field.
func (o *UsageSummaryDateOrg) SetCustomTsAvg(v int64) {
	o.CustomTsAvg = &v
}

// GetCwsContainerCountAvg returns the CwsContainerCountAvg field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetCwsContainerCountAvg() int64 {
	if o == nil || o.CwsContainerCountAvg == nil {
		var ret int64
		return ret
	}
	return *o.CwsContainerCountAvg
}

// GetCwsContainerCountAvgOk returns a tuple with the CwsContainerCountAvg field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetCwsContainerCountAvgOk() (*int64, bool) {
	if o == nil || o.CwsContainerCountAvg == nil {
		return nil, false
	}
	return o.CwsContainerCountAvg, true
}

// HasCwsContainerCountAvg returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasCwsContainerCountAvg() bool {
	if o != nil && o.CwsContainerCountAvg != nil {
		return true
	}

	return false
}

// SetCwsContainerCountAvg gets a reference to the given int64 and assigns it to the CwsContainerCountAvg field.
func (o *UsageSummaryDateOrg) SetCwsContainerCountAvg(v int64) {
	o.CwsContainerCountAvg = &v
}

// GetCwsHostTop99p returns the CwsHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetCwsHostTop99p() int64 {
	if o == nil || o.CwsHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.CwsHostTop99p
}

// GetCwsHostTop99pOk returns a tuple with the CwsHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetCwsHostTop99pOk() (*int64, bool) {
	if o == nil || o.CwsHostTop99p == nil {
		return nil, false
	}
	return o.CwsHostTop99p, true
}

// HasCwsHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasCwsHostTop99p() bool {
	if o != nil && o.CwsHostTop99p != nil {
		return true
	}

	return false
}

// SetCwsHostTop99p gets a reference to the given int64 and assigns it to the CwsHostTop99p field.
func (o *UsageSummaryDateOrg) SetCwsHostTop99p(v int64) {
	o.CwsHostTop99p = &v
}

// GetDbmHostTop99pSum returns the DbmHostTop99pSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetDbmHostTop99pSum() int64 {
	if o == nil || o.DbmHostTop99pSum == nil {
		var ret int64
		return ret
	}
	return *o.DbmHostTop99pSum
}

// GetDbmHostTop99pSumOk returns a tuple with the DbmHostTop99pSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetDbmHostTop99pSumOk() (*int64, bool) {
	if o == nil || o.DbmHostTop99pSum == nil {
		return nil, false
	}
	return o.DbmHostTop99pSum, true
}

// HasDbmHostTop99pSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasDbmHostTop99pSum() bool {
	if o != nil && o.DbmHostTop99pSum != nil {
		return true
	}

	return false
}

// SetDbmHostTop99pSum gets a reference to the given int64 and assigns it to the DbmHostTop99pSum field.
func (o *UsageSummaryDateOrg) SetDbmHostTop99pSum(v int64) {
	o.DbmHostTop99pSum = &v
}

// GetDbmQueriesAvgSum returns the DbmQueriesAvgSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetDbmQueriesAvgSum() int64 {
	if o == nil || o.DbmQueriesAvgSum == nil {
		var ret int64
		return ret
	}
	return *o.DbmQueriesAvgSum
}

// GetDbmQueriesAvgSumOk returns a tuple with the DbmQueriesAvgSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetDbmQueriesAvgSumOk() (*int64, bool) {
	if o == nil || o.DbmQueriesAvgSum == nil {
		return nil, false
	}
	return o.DbmQueriesAvgSum, true
}

// HasDbmQueriesAvgSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasDbmQueriesAvgSum() bool {
	if o != nil && o.DbmQueriesAvgSum != nil {
		return true
	}

	return false
}

// SetDbmQueriesAvgSum gets a reference to the given int64 and assigns it to the DbmQueriesAvgSum field.
func (o *UsageSummaryDateOrg) SetDbmQueriesAvgSum(v int64) {
	o.DbmQueriesAvgSum = &v
}

// GetFargateTasksCountAvg returns the FargateTasksCountAvg field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetFargateTasksCountAvg() int64 {
	if o == nil || o.FargateTasksCountAvg == nil {
		var ret int64
		return ret
	}
	return *o.FargateTasksCountAvg
}

// GetFargateTasksCountAvgOk returns a tuple with the FargateTasksCountAvg field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetFargateTasksCountAvgOk() (*int64, bool) {
	if o == nil || o.FargateTasksCountAvg == nil {
		return nil, false
	}
	return o.FargateTasksCountAvg, true
}

// HasFargateTasksCountAvg returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasFargateTasksCountAvg() bool {
	if o != nil && o.FargateTasksCountAvg != nil {
		return true
	}

	return false
}

// SetFargateTasksCountAvg gets a reference to the given int64 and assigns it to the FargateTasksCountAvg field.
func (o *UsageSummaryDateOrg) SetFargateTasksCountAvg(v int64) {
	o.FargateTasksCountAvg = &v
}

// GetFargateTasksCountHwm returns the FargateTasksCountHwm field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetFargateTasksCountHwm() int64 {
	if o == nil || o.FargateTasksCountHwm == nil {
		var ret int64
		return ret
	}
	return *o.FargateTasksCountHwm
}

// GetFargateTasksCountHwmOk returns a tuple with the FargateTasksCountHwm field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetFargateTasksCountHwmOk() (*int64, bool) {
	if o == nil || o.FargateTasksCountHwm == nil {
		return nil, false
	}
	return o.FargateTasksCountHwm, true
}

// HasFargateTasksCountHwm returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasFargateTasksCountHwm() bool {
	if o != nil && o.FargateTasksCountHwm != nil {
		return true
	}

	return false
}

// SetFargateTasksCountHwm gets a reference to the given int64 and assigns it to the FargateTasksCountHwm field.
func (o *UsageSummaryDateOrg) SetFargateTasksCountHwm(v int64) {
	o.FargateTasksCountHwm = &v
}

// GetGcpHostTop99p returns the GcpHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetGcpHostTop99p() int64 {
	if o == nil || o.GcpHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.GcpHostTop99p
}

// GetGcpHostTop99pOk returns a tuple with the GcpHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetGcpHostTop99pOk() (*int64, bool) {
	if o == nil || o.GcpHostTop99p == nil {
		return nil, false
	}
	return o.GcpHostTop99p, true
}

// HasGcpHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasGcpHostTop99p() bool {
	if o != nil && o.GcpHostTop99p != nil {
		return true
	}

	return false
}

// SetGcpHostTop99p gets a reference to the given int64 and assigns it to the GcpHostTop99p field.
func (o *UsageSummaryDateOrg) SetGcpHostTop99p(v int64) {
	o.GcpHostTop99p = &v
}

// GetHerokuHostTop99p returns the HerokuHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetHerokuHostTop99p() int64 {
	if o == nil || o.HerokuHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.HerokuHostTop99p
}

// GetHerokuHostTop99pOk returns a tuple with the HerokuHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetHerokuHostTop99pOk() (*int64, bool) {
	if o == nil || o.HerokuHostTop99p == nil {
		return nil, false
	}
	return o.HerokuHostTop99p, true
}

// HasHerokuHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasHerokuHostTop99p() bool {
	if o != nil && o.HerokuHostTop99p != nil {
		return true
	}

	return false
}

// SetHerokuHostTop99p gets a reference to the given int64 and assigns it to the HerokuHostTop99p field.
func (o *UsageSummaryDateOrg) SetHerokuHostTop99p(v int64) {
	o.HerokuHostTop99p = &v
}

// GetId returns the Id field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetId() string {
	if o == nil || o.Id == nil {
		var ret string
		return ret
	}
	return *o.Id
}

// GetIdOk returns a tuple with the Id field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetIdOk() (*string, bool) {
	if o == nil || o.Id == nil {
		return nil, false
	}
	return o.Id, true
}

// HasId returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasId() bool {
	if o != nil && o.Id != nil {
		return true
	}

	return false
}

// SetId gets a reference to the given string and assigns it to the Id field.
func (o *UsageSummaryDateOrg) SetId(v string) {
	o.Id = &v
}

// GetIncidentManagementMonthlyActiveUsersHwm returns the IncidentManagementMonthlyActiveUsersHwm field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetIncidentManagementMonthlyActiveUsersHwm() int64 {
	if o == nil || o.IncidentManagementMonthlyActiveUsersHwm == nil {
		var ret int64
		return ret
	}
	return *o.IncidentManagementMonthlyActiveUsersHwm
}

// GetIncidentManagementMonthlyActiveUsersHwmOk returns a tuple with the IncidentManagementMonthlyActiveUsersHwm field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetIncidentManagementMonthlyActiveUsersHwmOk() (*int64, bool) {
	if o == nil || o.IncidentManagementMonthlyActiveUsersHwm == nil {
		return nil, false
	}
	return o.IncidentManagementMonthlyActiveUsersHwm, true
}

// HasIncidentManagementMonthlyActiveUsersHwm returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasIncidentManagementMonthlyActiveUsersHwm() bool {
	if o != nil && o.IncidentManagementMonthlyActiveUsersHwm != nil {
		return true
	}

	return false
}

// SetIncidentManagementMonthlyActiveUsersHwm gets a reference to the given int64 and assigns it to the IncidentManagementMonthlyActiveUsersHwm field.
func (o *UsageSummaryDateOrg) SetIncidentManagementMonthlyActiveUsersHwm(v int64) {
	o.IncidentManagementMonthlyActiveUsersHwm = &v
}

// GetIndexedEventsCountSum returns the IndexedEventsCountSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetIndexedEventsCountSum() int64 {
	if o == nil || o.IndexedEventsCountSum == nil {
		var ret int64
		return ret
	}
	return *o.IndexedEventsCountSum
}

// GetIndexedEventsCountSumOk returns a tuple with the IndexedEventsCountSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetIndexedEventsCountSumOk() (*int64, bool) {
	if o == nil || o.IndexedEventsCountSum == nil {
		return nil, false
	}
	return o.IndexedEventsCountSum, true
}

// HasIndexedEventsCountSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasIndexedEventsCountSum() bool {
	if o != nil && o.IndexedEventsCountSum != nil {
		return true
	}

	return false
}

// SetIndexedEventsCountSum gets a reference to the given int64 and assigns it to the IndexedEventsCountSum field.
func (o *UsageSummaryDateOrg) SetIndexedEventsCountSum(v int64) {
	o.IndexedEventsCountSum = &v
}

// GetInfraHostTop99p returns the InfraHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetInfraHostTop99p() int64 {
	if o == nil || o.InfraHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.InfraHostTop99p
}

// GetInfraHostTop99pOk returns a tuple with the InfraHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetInfraHostTop99pOk() (*int64, bool) {
	if o == nil || o.InfraHostTop99p == nil {
		return nil, false
	}
	return o.InfraHostTop99p, true
}

// HasInfraHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasInfraHostTop99p() bool {
	if o != nil && o.InfraHostTop99p != nil {
		return true
	}

	return false
}

// SetInfraHostTop99p gets a reference to the given int64 and assigns it to the InfraHostTop99p field.
func (o *UsageSummaryDateOrg) SetInfraHostTop99p(v int64) {
	o.InfraHostTop99p = &v
}

// GetIngestedEventsBytesSum returns the IngestedEventsBytesSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetIngestedEventsBytesSum() int64 {
	if o == nil || o.IngestedEventsBytesSum == nil {
		var ret int64
		return ret
	}
	return *o.IngestedEventsBytesSum
}

// GetIngestedEventsBytesSumOk returns a tuple with the IngestedEventsBytesSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetIngestedEventsBytesSumOk() (*int64, bool) {
	if o == nil || o.IngestedEventsBytesSum == nil {
		return nil, false
	}
	return o.IngestedEventsBytesSum, true
}

// HasIngestedEventsBytesSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasIngestedEventsBytesSum() bool {
	if o != nil && o.IngestedEventsBytesSum != nil {
		return true
	}

	return false
}

// SetIngestedEventsBytesSum gets a reference to the given int64 and assigns it to the IngestedEventsBytesSum field.
func (o *UsageSummaryDateOrg) SetIngestedEventsBytesSum(v int64) {
	o.IngestedEventsBytesSum = &v
}

// GetIotDeviceAggSum returns the IotDeviceAggSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetIotDeviceAggSum() int64 {
	if o == nil || o.IotDeviceAggSum == nil {
		var ret int64
		return ret
	}
	return *o.IotDeviceAggSum
}

// GetIotDeviceAggSumOk returns a tuple with the IotDeviceAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetIotDeviceAggSumOk() (*int64, bool) {
	if o == nil || o.IotDeviceAggSum == nil {
		return nil, false
	}
	return o.IotDeviceAggSum, true
}

// HasIotDeviceAggSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasIotDeviceAggSum() bool {
	if o != nil && o.IotDeviceAggSum != nil {
		return true
	}

	return false
}

// SetIotDeviceAggSum gets a reference to the given int64 and assigns it to the IotDeviceAggSum field.
func (o *UsageSummaryDateOrg) SetIotDeviceAggSum(v int64) {
	o.IotDeviceAggSum = &v
}

// GetIotDeviceTop99pSum returns the IotDeviceTop99pSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetIotDeviceTop99pSum() int64 {
	if o == nil || o.IotDeviceTop99pSum == nil {
		var ret int64
		return ret
	}
	return *o.IotDeviceTop99pSum
}

// GetIotDeviceTop99pSumOk returns a tuple with the IotDeviceTop99pSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetIotDeviceTop99pSumOk() (*int64, bool) {
	if o == nil || o.IotDeviceTop99pSum == nil {
		return nil, false
	}
	return o.IotDeviceTop99pSum, true
}

// HasIotDeviceTop99pSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasIotDeviceTop99pSum() bool {
	if o != nil && o.IotDeviceTop99pSum != nil {
		return true
	}

	return false
}

// SetIotDeviceTop99pSum gets a reference to the given int64 and assigns it to the IotDeviceTop99pSum field.
func (o *UsageSummaryDateOrg) SetIotDeviceTop99pSum(v int64) {
	o.IotDeviceTop99pSum = &v
}

// GetMobileRumLiteSessionCountSum returns the MobileRumLiteSessionCountSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetMobileRumLiteSessionCountSum() int64 {
	if o == nil || o.MobileRumLiteSessionCountSum == nil {
		var ret int64
		return ret
	}
	return *o.MobileRumLiteSessionCountSum
}

// GetMobileRumLiteSessionCountSumOk returns a tuple with the MobileRumLiteSessionCountSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetMobileRumLiteSessionCountSumOk() (*int64, bool) {
	if o == nil || o.MobileRumLiteSessionCountSum == nil {
		return nil, false
	}
	return o.MobileRumLiteSessionCountSum, true
}

// HasMobileRumLiteSessionCountSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasMobileRumLiteSessionCountSum() bool {
	if o != nil && o.MobileRumLiteSessionCountSum != nil {
		return true
	}

	return false
}

// SetMobileRumLiteSessionCountSum gets a reference to the given int64 and assigns it to the MobileRumLiteSessionCountSum field.
func (o *UsageSummaryDateOrg) SetMobileRumLiteSessionCountSum(v int64) {
	o.MobileRumLiteSessionCountSum = &v
}

// GetMobileRumSessionCountAndroidSum returns the MobileRumSessionCountAndroidSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetMobileRumSessionCountAndroidSum() int64 {
	if o == nil || o.MobileRumSessionCountAndroidSum == nil {
		var ret int64
		return ret
	}
	return *o.MobileRumSessionCountAndroidSum
}

// GetMobileRumSessionCountAndroidSumOk returns a tuple with the MobileRumSessionCountAndroidSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetMobileRumSessionCountAndroidSumOk() (*int64, bool) {
	if o == nil || o.MobileRumSessionCountAndroidSum == nil {
		return nil, false
	}
	return o.MobileRumSessionCountAndroidSum, true
}

// HasMobileRumSessionCountAndroidSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasMobileRumSessionCountAndroidSum() bool {
	if o != nil && o.MobileRumSessionCountAndroidSum != nil {
		return true
	}

	return false
}

// SetMobileRumSessionCountAndroidSum gets a reference to the given int64 and assigns it to the MobileRumSessionCountAndroidSum field.
func (o *UsageSummaryDateOrg) SetMobileRumSessionCountAndroidSum(v int64) {
	o.MobileRumSessionCountAndroidSum = &v
}

// GetMobileRumSessionCountIosSum returns the MobileRumSessionCountIosSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetMobileRumSessionCountIosSum() int64 {
	if o == nil || o.MobileRumSessionCountIosSum == nil {
		var ret int64
		return ret
	}
	return *o.MobileRumSessionCountIosSum
}

// GetMobileRumSessionCountIosSumOk returns a tuple with the MobileRumSessionCountIosSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetMobileRumSessionCountIosSumOk() (*int64, bool) {
	if o == nil || o.MobileRumSessionCountIosSum == nil {
		return nil, false
	}
	return o.MobileRumSessionCountIosSum, true
}

// HasMobileRumSessionCountIosSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasMobileRumSessionCountIosSum() bool {
	if o != nil && o.MobileRumSessionCountIosSum != nil {
		return true
	}

	return false
}

// SetMobileRumSessionCountIosSum gets a reference to the given int64 and assigns it to the MobileRumSessionCountIosSum field.
func (o *UsageSummaryDateOrg) SetMobileRumSessionCountIosSum(v int64) {
	o.MobileRumSessionCountIosSum = &v
}

// GetMobileRumSessionCountReactnativeSum returns the MobileRumSessionCountReactnativeSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetMobileRumSessionCountReactnativeSum() int64 {
	if o == nil || o.MobileRumSessionCountReactnativeSum == nil {
		var ret int64
		return ret
	}
	return *o.MobileRumSessionCountReactnativeSum
}

// GetMobileRumSessionCountReactnativeSumOk returns a tuple with the MobileRumSessionCountReactnativeSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetMobileRumSessionCountReactnativeSumOk() (*int64, bool) {
	if o == nil || o.MobileRumSessionCountReactnativeSum == nil {
		return nil, false
	}
	return o.MobileRumSessionCountReactnativeSum, true
}

// HasMobileRumSessionCountReactnativeSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasMobileRumSessionCountReactnativeSum() bool {
	if o != nil && o.MobileRumSessionCountReactnativeSum != nil {
		return true
	}

	return false
}

// SetMobileRumSessionCountReactnativeSum gets a reference to the given int64 and assigns it to the MobileRumSessionCountReactnativeSum field.
func (o *UsageSummaryDateOrg) SetMobileRumSessionCountReactnativeSum(v int64) {
	o.MobileRumSessionCountReactnativeSum = &v
}

// GetMobileRumSessionCountSum returns the MobileRumSessionCountSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetMobileRumSessionCountSum() int64 {
	if o == nil || o.MobileRumSessionCountSum == nil {
		var ret int64
		return ret
	}
	return *o.MobileRumSessionCountSum
}

// GetMobileRumSessionCountSumOk returns a tuple with the MobileRumSessionCountSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetMobileRumSessionCountSumOk() (*int64, bool) {
	if o == nil || o.MobileRumSessionCountSum == nil {
		return nil, false
	}
	return o.MobileRumSessionCountSum, true
}

// HasMobileRumSessionCountSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasMobileRumSessionCountSum() bool {
	if o != nil && o.MobileRumSessionCountSum != nil {
		return true
	}

	return false
}

// SetMobileRumSessionCountSum gets a reference to the given int64 and assigns it to the MobileRumSessionCountSum field.
func (o *UsageSummaryDateOrg) SetMobileRumSessionCountSum(v int64) {
	o.MobileRumSessionCountSum = &v
}

// GetMobileRumUnitsSum returns the MobileRumUnitsSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetMobileRumUnitsSum() int64 {
	if o == nil || o.MobileRumUnitsSum == nil {
		var ret int64
		return ret
	}
	return *o.MobileRumUnitsSum
}

// GetMobileRumUnitsSumOk returns a tuple with the MobileRumUnitsSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetMobileRumUnitsSumOk() (*int64, bool) {
	if o == nil || o.MobileRumUnitsSum == nil {
		return nil, false
	}
	return o.MobileRumUnitsSum, true
}

// HasMobileRumUnitsSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasMobileRumUnitsSum() bool {
	if o != nil && o.MobileRumUnitsSum != nil {
		return true
	}

	return false
}

// SetMobileRumUnitsSum gets a reference to the given int64 and assigns it to the MobileRumUnitsSum field.
func (o *UsageSummaryDateOrg) SetMobileRumUnitsSum(v int64) {
	o.MobileRumUnitsSum = &v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasName() bool {
	if o != nil && o.Name != nil {
		return true
	}

	return false
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *UsageSummaryDateOrg) SetName(v string) {
	o.Name = &v
}

// GetNetflowIndexedEventsCountSum returns the NetflowIndexedEventsCountSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetNetflowIndexedEventsCountSum() int64 {
	if o == nil || o.NetflowIndexedEventsCountSum == nil {
		var ret int64
		return ret
	}
	return *o.NetflowIndexedEventsCountSum
}

// GetNetflowIndexedEventsCountSumOk returns a tuple with the NetflowIndexedEventsCountSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetNetflowIndexedEventsCountSumOk() (*int64, bool) {
	if o == nil || o.NetflowIndexedEventsCountSum == nil {
		return nil, false
	}
	return o.NetflowIndexedEventsCountSum, true
}

// HasNetflowIndexedEventsCountSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasNetflowIndexedEventsCountSum() bool {
	if o != nil && o.NetflowIndexedEventsCountSum != nil {
		return true
	}

	return false
}

// SetNetflowIndexedEventsCountSum gets a reference to the given int64 and assigns it to the NetflowIndexedEventsCountSum field.
func (o *UsageSummaryDateOrg) SetNetflowIndexedEventsCountSum(v int64) {
	o.NetflowIndexedEventsCountSum = &v
}

// GetNpmHostTop99p returns the NpmHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetNpmHostTop99p() int64 {
	if o == nil || o.NpmHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.NpmHostTop99p
}

// GetNpmHostTop99pOk returns a tuple with the NpmHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetNpmHostTop99pOk() (*int64, bool) {
	if o == nil || o.NpmHostTop99p == nil {
		return nil, false
	}
	return o.NpmHostTop99p, true
}

// HasNpmHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasNpmHostTop99p() bool {
	if o != nil && o.NpmHostTop99p != nil {
		return true
	}

	return false
}

// SetNpmHostTop99p gets a reference to the given int64 and assigns it to the NpmHostTop99p field.
func (o *UsageSummaryDateOrg) SetNpmHostTop99p(v int64) {
	o.NpmHostTop99p = &v
}

// GetObservabilityPipelinesBytesProcessedSum returns the ObservabilityPipelinesBytesProcessedSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetObservabilityPipelinesBytesProcessedSum() int64 {
	if o == nil || o.ObservabilityPipelinesBytesProcessedSum == nil {
		var ret int64
		return ret
	}
	return *o.ObservabilityPipelinesBytesProcessedSum
}

// GetObservabilityPipelinesBytesProcessedSumOk returns a tuple with the ObservabilityPipelinesBytesProcessedSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetObservabilityPipelinesBytesProcessedSumOk() (*int64, bool) {
	if o == nil || o.ObservabilityPipelinesBytesProcessedSum == nil {
		return nil, false
	}
	return o.ObservabilityPipelinesBytesProcessedSum, true
}

// HasObservabilityPipelinesBytesProcessedSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasObservabilityPipelinesBytesProcessedSum() bool {
	if o != nil && o.ObservabilityPipelinesBytesProcessedSum != nil {
		return true
	}

	return false
}

// SetObservabilityPipelinesBytesProcessedSum gets a reference to the given int64 and assigns it to the ObservabilityPipelinesBytesProcessedSum field.
func (o *UsageSummaryDateOrg) SetObservabilityPipelinesBytesProcessedSum(v int64) {
	o.ObservabilityPipelinesBytesProcessedSum = &v
}

// GetOnlineArchiveEventsCountSum returns the OnlineArchiveEventsCountSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetOnlineArchiveEventsCountSum() int64 {
	if o == nil || o.OnlineArchiveEventsCountSum == nil {
		var ret int64
		return ret
	}
	return *o.OnlineArchiveEventsCountSum
}

// GetOnlineArchiveEventsCountSumOk returns a tuple with the OnlineArchiveEventsCountSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetOnlineArchiveEventsCountSumOk() (*int64, bool) {
	if o == nil || o.OnlineArchiveEventsCountSum == nil {
		return nil, false
	}
	return o.OnlineArchiveEventsCountSum, true
}

// HasOnlineArchiveEventsCountSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasOnlineArchiveEventsCountSum() bool {
	if o != nil && o.OnlineArchiveEventsCountSum != nil {
		return true
	}

	return false
}

// SetOnlineArchiveEventsCountSum gets a reference to the given int64 and assigns it to the OnlineArchiveEventsCountSum field.
func (o *UsageSummaryDateOrg) SetOnlineArchiveEventsCountSum(v int64) {
	o.OnlineArchiveEventsCountSum = &v
}

// GetOpentelemetryHostTop99p returns the OpentelemetryHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetOpentelemetryHostTop99p() int64 {
	if o == nil || o.OpentelemetryHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.OpentelemetryHostTop99p
}

// GetOpentelemetryHostTop99pOk returns a tuple with the OpentelemetryHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetOpentelemetryHostTop99pOk() (*int64, bool) {
	if o == nil || o.OpentelemetryHostTop99p == nil {
		return nil, false
	}
	return o.OpentelemetryHostTop99p, true
}

// HasOpentelemetryHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasOpentelemetryHostTop99p() bool {
	if o != nil && o.OpentelemetryHostTop99p != nil {
		return true
	}

	return false
}

// SetOpentelemetryHostTop99p gets a reference to the given int64 and assigns it to the OpentelemetryHostTop99p field.
func (o *UsageSummaryDateOrg) SetOpentelemetryHostTop99p(v int64) {
	o.OpentelemetryHostTop99p = &v
}

// GetProfilingHostTop99p returns the ProfilingHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetProfilingHostTop99p() int64 {
	if o == nil || o.ProfilingHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.ProfilingHostTop99p
}

// GetProfilingHostTop99pOk returns a tuple with the ProfilingHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetProfilingHostTop99pOk() (*int64, bool) {
	if o == nil || o.ProfilingHostTop99p == nil {
		return nil, false
	}
	return o.ProfilingHostTop99p, true
}

// HasProfilingHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasProfilingHostTop99p() bool {
	if o != nil && o.ProfilingHostTop99p != nil {
		return true
	}

	return false
}

// SetProfilingHostTop99p gets a reference to the given int64 and assigns it to the ProfilingHostTop99p field.
func (o *UsageSummaryDateOrg) SetProfilingHostTop99p(v int64) {
	o.ProfilingHostTop99p = &v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasPublicId() bool {
	if o != nil && o.PublicId != nil {
		return true
	}

	return false
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *UsageSummaryDateOrg) SetPublicId(v string) {
	o.PublicId = &v
}

// GetRumBrowserAndMobileSessionCount returns the RumBrowserAndMobileSessionCount field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetRumBrowserAndMobileSessionCount() int64 {
	if o == nil || o.RumBrowserAndMobileSessionCount == nil {
		var ret int64
		return ret
	}
	return *o.RumBrowserAndMobileSessionCount
}

// GetRumBrowserAndMobileSessionCountOk returns a tuple with the RumBrowserAndMobileSessionCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetRumBrowserAndMobileSessionCountOk() (*int64, bool) {
	if o == nil || o.RumBrowserAndMobileSessionCount == nil {
		return nil, false
	}
	return o.RumBrowserAndMobileSessionCount, true
}

// HasRumBrowserAndMobileSessionCount returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasRumBrowserAndMobileSessionCount() bool {
	if o != nil && o.RumBrowserAndMobileSessionCount != nil {
		return true
	}

	return false
}

// SetRumBrowserAndMobileSessionCount gets a reference to the given int64 and assigns it to the RumBrowserAndMobileSessionCount field.
func (o *UsageSummaryDateOrg) SetRumBrowserAndMobileSessionCount(v int64) {
	o.RumBrowserAndMobileSessionCount = &v
}

// GetRumSessionCountSum returns the RumSessionCountSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetRumSessionCountSum() int64 {
	if o == nil || o.RumSessionCountSum == nil {
		var ret int64
		return ret
	}
	return *o.RumSessionCountSum
}

// GetRumSessionCountSumOk returns a tuple with the RumSessionCountSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetRumSessionCountSumOk() (*int64, bool) {
	if o == nil || o.RumSessionCountSum == nil {
		return nil, false
	}
	return o.RumSessionCountSum, true
}

// HasRumSessionCountSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasRumSessionCountSum() bool {
	if o != nil && o.RumSessionCountSum != nil {
		return true
	}

	return false
}

// SetRumSessionCountSum gets a reference to the given int64 and assigns it to the RumSessionCountSum field.
func (o *UsageSummaryDateOrg) SetRumSessionCountSum(v int64) {
	o.RumSessionCountSum = &v
}

// GetRumTotalSessionCountSum returns the RumTotalSessionCountSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetRumTotalSessionCountSum() int64 {
	if o == nil || o.RumTotalSessionCountSum == nil {
		var ret int64
		return ret
	}
	return *o.RumTotalSessionCountSum
}

// GetRumTotalSessionCountSumOk returns a tuple with the RumTotalSessionCountSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetRumTotalSessionCountSumOk() (*int64, bool) {
	if o == nil || o.RumTotalSessionCountSum == nil {
		return nil, false
	}
	return o.RumTotalSessionCountSum, true
}

// HasRumTotalSessionCountSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasRumTotalSessionCountSum() bool {
	if o != nil && o.RumTotalSessionCountSum != nil {
		return true
	}

	return false
}

// SetRumTotalSessionCountSum gets a reference to the given int64 and assigns it to the RumTotalSessionCountSum field.
func (o *UsageSummaryDateOrg) SetRumTotalSessionCountSum(v int64) {
	o.RumTotalSessionCountSum = &v
}

// GetRumUnitsSum returns the RumUnitsSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetRumUnitsSum() int64 {
	if o == nil || o.RumUnitsSum == nil {
		var ret int64
		return ret
	}
	return *o.RumUnitsSum
}

// GetRumUnitsSumOk returns a tuple with the RumUnitsSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetRumUnitsSumOk() (*int64, bool) {
	if o == nil || o.RumUnitsSum == nil {
		return nil, false
	}
	return o.RumUnitsSum, true
}

// HasRumUnitsSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasRumUnitsSum() bool {
	if o != nil && o.RumUnitsSum != nil {
		return true
	}

	return false
}

// SetRumUnitsSum gets a reference to the given int64 and assigns it to the RumUnitsSum field.
func (o *UsageSummaryDateOrg) SetRumUnitsSum(v int64) {
	o.RumUnitsSum = &v
}

// GetSdsLogsScannedBytesSum returns the SdsLogsScannedBytesSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetSdsLogsScannedBytesSum() int64 {
	if o == nil || o.SdsLogsScannedBytesSum == nil {
		var ret int64
		return ret
	}
	return *o.SdsLogsScannedBytesSum
}

// GetSdsLogsScannedBytesSumOk returns a tuple with the SdsLogsScannedBytesSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetSdsLogsScannedBytesSumOk() (*int64, bool) {
	if o == nil || o.SdsLogsScannedBytesSum == nil {
		return nil, false
	}
	return o.SdsLogsScannedBytesSum, true
}

// HasSdsLogsScannedBytesSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasSdsLogsScannedBytesSum() bool {
	if o != nil && o.SdsLogsScannedBytesSum != nil {
		return true
	}

	return false
}

// SetSdsLogsScannedBytesSum gets a reference to the given int64 and assigns it to the SdsLogsScannedBytesSum field.
func (o *UsageSummaryDateOrg) SetSdsLogsScannedBytesSum(v int64) {
	o.SdsLogsScannedBytesSum = &v
}

// GetSdsTotalScannedBytesSum returns the SdsTotalScannedBytesSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetSdsTotalScannedBytesSum() int64 {
	if o == nil || o.SdsTotalScannedBytesSum == nil {
		var ret int64
		return ret
	}
	return *o.SdsTotalScannedBytesSum
}

// GetSdsTotalScannedBytesSumOk returns a tuple with the SdsTotalScannedBytesSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetSdsTotalScannedBytesSumOk() (*int64, bool) {
	if o == nil || o.SdsTotalScannedBytesSum == nil {
		return nil, false
	}
	return o.SdsTotalScannedBytesSum, true
}

// HasSdsTotalScannedBytesSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasSdsTotalScannedBytesSum() bool {
	if o != nil && o.SdsTotalScannedBytesSum != nil {
		return true
	}

	return false
}

// SetSdsTotalScannedBytesSum gets a reference to the given int64 and assigns it to the SdsTotalScannedBytesSum field.
func (o *UsageSummaryDateOrg) SetSdsTotalScannedBytesSum(v int64) {
	o.SdsTotalScannedBytesSum = &v
}

// GetSyntheticsBrowserCheckCallsCountSum returns the SyntheticsBrowserCheckCallsCountSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetSyntheticsBrowserCheckCallsCountSum() int64 {
	if o == nil || o.SyntheticsBrowserCheckCallsCountSum == nil {
		var ret int64
		return ret
	}
	return *o.SyntheticsBrowserCheckCallsCountSum
}

// GetSyntheticsBrowserCheckCallsCountSumOk returns a tuple with the SyntheticsBrowserCheckCallsCountSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetSyntheticsBrowserCheckCallsCountSumOk() (*int64, bool) {
	if o == nil || o.SyntheticsBrowserCheckCallsCountSum == nil {
		return nil, false
	}
	return o.SyntheticsBrowserCheckCallsCountSum, true
}

// HasSyntheticsBrowserCheckCallsCountSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasSyntheticsBrowserCheckCallsCountSum() bool {
	if o != nil && o.SyntheticsBrowserCheckCallsCountSum != nil {
		return true
	}

	return false
}

// SetSyntheticsBrowserCheckCallsCountSum gets a reference to the given int64 and assigns it to the SyntheticsBrowserCheckCallsCountSum field.
func (o *UsageSummaryDateOrg) SetSyntheticsBrowserCheckCallsCountSum(v int64) {
	o.SyntheticsBrowserCheckCallsCountSum = &v
}

// GetSyntheticsCheckCallsCountSum returns the SyntheticsCheckCallsCountSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetSyntheticsCheckCallsCountSum() int64 {
	if o == nil || o.SyntheticsCheckCallsCountSum == nil {
		var ret int64
		return ret
	}
	return *o.SyntheticsCheckCallsCountSum
}

// GetSyntheticsCheckCallsCountSumOk returns a tuple with the SyntheticsCheckCallsCountSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetSyntheticsCheckCallsCountSumOk() (*int64, bool) {
	if o == nil || o.SyntheticsCheckCallsCountSum == nil {
		return nil, false
	}
	return o.SyntheticsCheckCallsCountSum, true
}

// HasSyntheticsCheckCallsCountSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasSyntheticsCheckCallsCountSum() bool {
	if o != nil && o.SyntheticsCheckCallsCountSum != nil {
		return true
	}

	return false
}

// SetSyntheticsCheckCallsCountSum gets a reference to the given int64 and assigns it to the SyntheticsCheckCallsCountSum field.
func (o *UsageSummaryDateOrg) SetSyntheticsCheckCallsCountSum(v int64) {
	o.SyntheticsCheckCallsCountSum = &v
}

// GetTraceSearchIndexedEventsCountSum returns the TraceSearchIndexedEventsCountSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetTraceSearchIndexedEventsCountSum() int64 {
	if o == nil || o.TraceSearchIndexedEventsCountSum == nil {
		var ret int64
		return ret
	}
	return *o.TraceSearchIndexedEventsCountSum
}

// GetTraceSearchIndexedEventsCountSumOk returns a tuple with the TraceSearchIndexedEventsCountSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetTraceSearchIndexedEventsCountSumOk() (*int64, bool) {
	if o == nil || o.TraceSearchIndexedEventsCountSum == nil {
		return nil, false
	}
	return o.TraceSearchIndexedEventsCountSum, true
}

// HasTraceSearchIndexedEventsCountSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasTraceSearchIndexedEventsCountSum() bool {
	if o != nil && o.TraceSearchIndexedEventsCountSum != nil {
		return true
	}

	return false
}

// SetTraceSearchIndexedEventsCountSum gets a reference to the given int64 and assigns it to the TraceSearchIndexedEventsCountSum field.
func (o *UsageSummaryDateOrg) SetTraceSearchIndexedEventsCountSum(v int64) {
	o.TraceSearchIndexedEventsCountSum = &v
}

// GetTwolIngestedEventsBytesSum returns the TwolIngestedEventsBytesSum field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetTwolIngestedEventsBytesSum() int64 {
	if o == nil || o.TwolIngestedEventsBytesSum == nil {
		var ret int64
		return ret
	}
	return *o.TwolIngestedEventsBytesSum
}

// GetTwolIngestedEventsBytesSumOk returns a tuple with the TwolIngestedEventsBytesSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetTwolIngestedEventsBytesSumOk() (*int64, bool) {
	if o == nil || o.TwolIngestedEventsBytesSum == nil {
		return nil, false
	}
	return o.TwolIngestedEventsBytesSum, true
}

// HasTwolIngestedEventsBytesSum returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasTwolIngestedEventsBytesSum() bool {
	if o != nil && o.TwolIngestedEventsBytesSum != nil {
		return true
	}

	return false
}

// SetTwolIngestedEventsBytesSum gets a reference to the given int64 and assigns it to the TwolIngestedEventsBytesSum field.
func (o *UsageSummaryDateOrg) SetTwolIngestedEventsBytesSum(v int64) {
	o.TwolIngestedEventsBytesSum = &v
}

// GetVsphereHostTop99p returns the VsphereHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDateOrg) GetVsphereHostTop99p() int64 {
	if o == nil || o.VsphereHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.VsphereHostTop99p
}

// GetVsphereHostTop99pOk returns a tuple with the VsphereHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDateOrg) GetVsphereHostTop99pOk() (*int64, bool) {
	if o == nil || o.VsphereHostTop99p == nil {
		return nil, false
	}
	return o.VsphereHostTop99p, true
}

// HasVsphereHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDateOrg) HasVsphereHostTop99p() bool {
	if o != nil && o.VsphereHostTop99p != nil {
		return true
	}

	return false
}

// SetVsphereHostTop99p gets a reference to the given int64 and assigns it to the VsphereHostTop99p field.
func (o *UsageSummaryDateOrg) SetVsphereHostTop99p(v int64) {
	o.VsphereHostTop99p = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageSummaryDateOrg) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AgentHostTop99p != nil {
		toSerialize["agent_host_top99p"] = o.AgentHostTop99p
	}
	if o.ApmAzureAppServiceHostTop99p != nil {
		toSerialize["apm_azure_app_service_host_top99p"] = o.ApmAzureAppServiceHostTop99p
	}
	if o.ApmHostTop99p != nil {
		toSerialize["apm_host_top99p"] = o.ApmHostTop99p
	}
	if o.AuditLogsLinesIndexedSum != nil {
		toSerialize["audit_logs_lines_indexed_sum"] = o.AuditLogsLinesIndexedSum
	}
	if o.AvgProfiledFargateTasks != nil {
		toSerialize["avg_profiled_fargate_tasks"] = o.AvgProfiledFargateTasks
	}
	if o.AwsHostTop99p != nil {
		toSerialize["aws_host_top99p"] = o.AwsHostTop99p
	}
	if o.AwsLambdaFuncCount != nil {
		toSerialize["aws_lambda_func_count"] = o.AwsLambdaFuncCount
	}
	if o.AwsLambdaInvocationsSum != nil {
		toSerialize["aws_lambda_invocations_sum"] = o.AwsLambdaInvocationsSum
	}
	if o.AzureAppServiceTop99p != nil {
		toSerialize["azure_app_service_top99p"] = o.AzureAppServiceTop99p
	}
	if o.BillableIngestedBytesSum != nil {
		toSerialize["billable_ingested_bytes_sum"] = o.BillableIngestedBytesSum
	}
	if o.BrowserRumLiteSessionCountSum != nil {
		toSerialize["browser_rum_lite_session_count_sum"] = o.BrowserRumLiteSessionCountSum
	}
	if o.BrowserRumReplaySessionCountSum != nil {
		toSerialize["browser_rum_replay_session_count_sum"] = o.BrowserRumReplaySessionCountSum
	}
	if o.BrowserRumUnitsSum != nil {
		toSerialize["browser_rum_units_sum"] = o.BrowserRumUnitsSum
	}
	if o.CiPipelineIndexedSpansSum != nil {
		toSerialize["ci_pipeline_indexed_spans_sum"] = o.CiPipelineIndexedSpansSum
	}
	if o.CiTestIndexedSpansSum != nil {
		toSerialize["ci_test_indexed_spans_sum"] = o.CiTestIndexedSpansSum
	}
	if o.CiVisibilityPipelineCommittersHwm != nil {
		toSerialize["ci_visibility_pipeline_committers_hwm"] = o.CiVisibilityPipelineCommittersHwm
	}
	if o.CiVisibilityTestCommittersHwm != nil {
		toSerialize["ci_visibility_test_committers_hwm"] = o.CiVisibilityTestCommittersHwm
	}
	if o.ContainerAvg != nil {
		toSerialize["container_avg"] = o.ContainerAvg
	}
	if o.ContainerHwm != nil {
		toSerialize["container_hwm"] = o.ContainerHwm
	}
	if o.CspmAasHostTop99p != nil {
		toSerialize["cspm_aas_host_top99p"] = o.CspmAasHostTop99p
	}
	if o.CspmAzureHostTop99p != nil {
		toSerialize["cspm_azure_host_top99p"] = o.CspmAzureHostTop99p
	}
	if o.CspmContainerAvg != nil {
		toSerialize["cspm_container_avg"] = o.CspmContainerAvg
	}
	if o.CspmContainerHwm != nil {
		toSerialize["cspm_container_hwm"] = o.CspmContainerHwm
	}
	if o.CspmHostTop99p != nil {
		toSerialize["cspm_host_top99p"] = o.CspmHostTop99p
	}
	if o.CustomTsAvg != nil {
		toSerialize["custom_ts_avg"] = o.CustomTsAvg
	}
	if o.CwsContainerCountAvg != nil {
		toSerialize["cws_container_count_avg"] = o.CwsContainerCountAvg
	}
	if o.CwsHostTop99p != nil {
		toSerialize["cws_host_top99p"] = o.CwsHostTop99p
	}
	if o.DbmHostTop99pSum != nil {
		toSerialize["dbm_host_top99p_sum"] = o.DbmHostTop99pSum
	}
	if o.DbmQueriesAvgSum != nil {
		toSerialize["dbm_queries_avg_sum"] = o.DbmQueriesAvgSum
	}
	if o.FargateTasksCountAvg != nil {
		toSerialize["fargate_tasks_count_avg"] = o.FargateTasksCountAvg
	}
	if o.FargateTasksCountHwm != nil {
		toSerialize["fargate_tasks_count_hwm"] = o.FargateTasksCountHwm
	}
	if o.GcpHostTop99p != nil {
		toSerialize["gcp_host_top99p"] = o.GcpHostTop99p
	}
	if o.HerokuHostTop99p != nil {
		toSerialize["heroku_host_top99p"] = o.HerokuHostTop99p
	}
	if o.Id != nil {
		toSerialize["id"] = o.Id
	}
	if o.IncidentManagementMonthlyActiveUsersHwm != nil {
		toSerialize["incident_management_monthly_active_users_hwm"] = o.IncidentManagementMonthlyActiveUsersHwm
	}
	if o.IndexedEventsCountSum != nil {
		toSerialize["indexed_events_count_sum"] = o.IndexedEventsCountSum
	}
	if o.InfraHostTop99p != nil {
		toSerialize["infra_host_top99p"] = o.InfraHostTop99p
	}
	if o.IngestedEventsBytesSum != nil {
		toSerialize["ingested_events_bytes_sum"] = o.IngestedEventsBytesSum
	}
	if o.IotDeviceAggSum != nil {
		toSerialize["iot_device_agg_sum"] = o.IotDeviceAggSum
	}
	if o.IotDeviceTop99pSum != nil {
		toSerialize["iot_device_top99p_sum"] = o.IotDeviceTop99pSum
	}
	if o.MobileRumLiteSessionCountSum != nil {
		toSerialize["mobile_rum_lite_session_count_sum"] = o.MobileRumLiteSessionCountSum
	}
	if o.MobileRumSessionCountAndroidSum != nil {
		toSerialize["mobile_rum_session_count_android_sum"] = o.MobileRumSessionCountAndroidSum
	}
	if o.MobileRumSessionCountIosSum != nil {
		toSerialize["mobile_rum_session_count_ios_sum"] = o.MobileRumSessionCountIosSum
	}
	if o.MobileRumSessionCountReactnativeSum != nil {
		toSerialize["mobile_rum_session_count_reactnative_sum"] = o.MobileRumSessionCountReactnativeSum
	}
	if o.MobileRumSessionCountSum != nil {
		toSerialize["mobile_rum_session_count_sum"] = o.MobileRumSessionCountSum
	}
	if o.MobileRumUnitsSum != nil {
		toSerialize["mobile_rum_units_sum"] = o.MobileRumUnitsSum
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.NetflowIndexedEventsCountSum != nil {
		toSerialize["netflow_indexed_events_count_sum"] = o.NetflowIndexedEventsCountSum
	}
	if o.NpmHostTop99p != nil {
		toSerialize["npm_host_top99p"] = o.NpmHostTop99p
	}
	if o.ObservabilityPipelinesBytesProcessedSum != nil {
		toSerialize["observability_pipelines_bytes_processed_sum"] = o.ObservabilityPipelinesBytesProcessedSum
	}
	if o.OnlineArchiveEventsCountSum != nil {
		toSerialize["online_archive_events_count_sum"] = o.OnlineArchiveEventsCountSum
	}
	if o.OpentelemetryHostTop99p != nil {
		toSerialize["opentelemetry_host_top99p"] = o.OpentelemetryHostTop99p
	}
	if o.ProfilingHostTop99p != nil {
		toSerialize["profiling_host_top99p"] = o.ProfilingHostTop99p
	}
	if o.PublicId != nil {
		toSerialize["public_id"] = o.PublicId
	}
	if o.RumBrowserAndMobileSessionCount != nil {
		toSerialize["rum_browser_and_mobile_session_count"] = o.RumBrowserAndMobileSessionCount
	}
	if o.RumSessionCountSum != nil {
		toSerialize["rum_session_count_sum"] = o.RumSessionCountSum
	}
	if o.RumTotalSessionCountSum != nil {
		toSerialize["rum_total_session_count_sum"] = o.RumTotalSessionCountSum
	}
	if o.RumUnitsSum != nil {
		toSerialize["rum_units_sum"] = o.RumUnitsSum
	}
	if o.SdsLogsScannedBytesSum != nil {
		toSerialize["sds_logs_scanned_bytes_sum"] = o.SdsLogsScannedBytesSum
	}
	if o.SdsTotalScannedBytesSum != nil {
		toSerialize["sds_total_scanned_bytes_sum"] = o.SdsTotalScannedBytesSum
	}
	if o.SyntheticsBrowserCheckCallsCountSum != nil {
		toSerialize["synthetics_browser_check_calls_count_sum"] = o.SyntheticsBrowserCheckCallsCountSum
	}
	if o.SyntheticsCheckCallsCountSum != nil {
		toSerialize["synthetics_check_calls_count_sum"] = o.SyntheticsCheckCallsCountSum
	}
	if o.TraceSearchIndexedEventsCountSum != nil {
		toSerialize["trace_search_indexed_events_count_sum"] = o.TraceSearchIndexedEventsCountSum
	}
	if o.TwolIngestedEventsBytesSum != nil {
		toSerialize["twol_ingested_events_bytes_sum"] = o.TwolIngestedEventsBytesSum
	}
	if o.VsphereHostTop99p != nil {
		toSerialize["vsphere_host_top99p"] = o.VsphereHostTop99p
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UsageSummaryDateOrg) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		AgentHostTop99p                         *int64  `json:"agent_host_top99p,omitempty"`
		ApmAzureAppServiceHostTop99p            *int64  `json:"apm_azure_app_service_host_top99p,omitempty"`
		ApmHostTop99p                           *int64  `json:"apm_host_top99p,omitempty"`
		AuditLogsLinesIndexedSum                *int64  `json:"audit_logs_lines_indexed_sum,omitempty"`
		AvgProfiledFargateTasks                 *int64  `json:"avg_profiled_fargate_tasks,omitempty"`
		AwsHostTop99p                           *int64  `json:"aws_host_top99p,omitempty"`
		AwsLambdaFuncCount                      *int64  `json:"aws_lambda_func_count,omitempty"`
		AwsLambdaInvocationsSum                 *int64  `json:"aws_lambda_invocations_sum,omitempty"`
		AzureAppServiceTop99p                   *int64  `json:"azure_app_service_top99p,omitempty"`
		BillableIngestedBytesSum                *int64  `json:"billable_ingested_bytes_sum,omitempty"`
		BrowserRumLiteSessionCountSum           *int64  `json:"browser_rum_lite_session_count_sum,omitempty"`
		BrowserRumReplaySessionCountSum         *int64  `json:"browser_rum_replay_session_count_sum,omitempty"`
		BrowserRumUnitsSum                      *int64  `json:"browser_rum_units_sum,omitempty"`
		CiPipelineIndexedSpansSum               *int64  `json:"ci_pipeline_indexed_spans_sum,omitempty"`
		CiTestIndexedSpansSum                   *int64  `json:"ci_test_indexed_spans_sum,omitempty"`
		CiVisibilityPipelineCommittersHwm       *int64  `json:"ci_visibility_pipeline_committers_hwm,omitempty"`
		CiVisibilityTestCommittersHwm           *int64  `json:"ci_visibility_test_committers_hwm,omitempty"`
		ContainerAvg                            *int64  `json:"container_avg,omitempty"`
		ContainerHwm                            *int64  `json:"container_hwm,omitempty"`
		CspmAasHostTop99p                       *int64  `json:"cspm_aas_host_top99p,omitempty"`
		CspmAzureHostTop99p                     *int64  `json:"cspm_azure_host_top99p,omitempty"`
		CspmContainerAvg                        *int64  `json:"cspm_container_avg,omitempty"`
		CspmContainerHwm                        *int64  `json:"cspm_container_hwm,omitempty"`
		CspmHostTop99p                          *int64  `json:"cspm_host_top99p,omitempty"`
		CustomTsAvg                             *int64  `json:"custom_ts_avg,omitempty"`
		CwsContainerCountAvg                    *int64  `json:"cws_container_count_avg,omitempty"`
		CwsHostTop99p                           *int64  `json:"cws_host_top99p,omitempty"`
		DbmHostTop99pSum                        *int64  `json:"dbm_host_top99p_sum,omitempty"`
		DbmQueriesAvgSum                        *int64  `json:"dbm_queries_avg_sum,omitempty"`
		FargateTasksCountAvg                    *int64  `json:"fargate_tasks_count_avg,omitempty"`
		FargateTasksCountHwm                    *int64  `json:"fargate_tasks_count_hwm,omitempty"`
		GcpHostTop99p                           *int64  `json:"gcp_host_top99p,omitempty"`
		HerokuHostTop99p                        *int64  `json:"heroku_host_top99p,omitempty"`
		Id                                      *string `json:"id,omitempty"`
		IncidentManagementMonthlyActiveUsersHwm *int64  `json:"incident_management_monthly_active_users_hwm,omitempty"`
		IndexedEventsCountSum                   *int64  `json:"indexed_events_count_sum,omitempty"`
		InfraHostTop99p                         *int64  `json:"infra_host_top99p,omitempty"`
		IngestedEventsBytesSum                  *int64  `json:"ingested_events_bytes_sum,omitempty"`
		IotDeviceAggSum                         *int64  `json:"iot_device_agg_sum,omitempty"`
		IotDeviceTop99pSum                      *int64  `json:"iot_device_top99p_sum,omitempty"`
		MobileRumLiteSessionCountSum            *int64  `json:"mobile_rum_lite_session_count_sum,omitempty"`
		MobileRumSessionCountAndroidSum         *int64  `json:"mobile_rum_session_count_android_sum,omitempty"`
		MobileRumSessionCountIosSum             *int64  `json:"mobile_rum_session_count_ios_sum,omitempty"`
		MobileRumSessionCountReactnativeSum     *int64  `json:"mobile_rum_session_count_reactnative_sum,omitempty"`
		MobileRumSessionCountSum                *int64  `json:"mobile_rum_session_count_sum,omitempty"`
		MobileRumUnitsSum                       *int64  `json:"mobile_rum_units_sum,omitempty"`
		Name                                    *string `json:"name,omitempty"`
		NetflowIndexedEventsCountSum            *int64  `json:"netflow_indexed_events_count_sum,omitempty"`
		NpmHostTop99p                           *int64  `json:"npm_host_top99p,omitempty"`
		ObservabilityPipelinesBytesProcessedSum *int64  `json:"observability_pipelines_bytes_processed_sum,omitempty"`
		OnlineArchiveEventsCountSum             *int64  `json:"online_archive_events_count_sum,omitempty"`
		OpentelemetryHostTop99p                 *int64  `json:"opentelemetry_host_top99p,omitempty"`
		ProfilingHostTop99p                     *int64  `json:"profiling_host_top99p,omitempty"`
		PublicId                                *string `json:"public_id,omitempty"`
		RumBrowserAndMobileSessionCount         *int64  `json:"rum_browser_and_mobile_session_count,omitempty"`
		RumSessionCountSum                      *int64  `json:"rum_session_count_sum,omitempty"`
		RumTotalSessionCountSum                 *int64  `json:"rum_total_session_count_sum,omitempty"`
		RumUnitsSum                             *int64  `json:"rum_units_sum,omitempty"`
		SdsLogsScannedBytesSum                  *int64  `json:"sds_logs_scanned_bytes_sum,omitempty"`
		SdsTotalScannedBytesSum                 *int64  `json:"sds_total_scanned_bytes_sum,omitempty"`
		SyntheticsBrowserCheckCallsCountSum     *int64  `json:"synthetics_browser_check_calls_count_sum,omitempty"`
		SyntheticsCheckCallsCountSum            *int64  `json:"synthetics_check_calls_count_sum,omitempty"`
		TraceSearchIndexedEventsCountSum        *int64  `json:"trace_search_indexed_events_count_sum,omitempty"`
		TwolIngestedEventsBytesSum              *int64  `json:"twol_ingested_events_bytes_sum,omitempty"`
		VsphereHostTop99p                       *int64  `json:"vsphere_host_top99p,omitempty"`
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
	o.AgentHostTop99p = all.AgentHostTop99p
	o.ApmAzureAppServiceHostTop99p = all.ApmAzureAppServiceHostTop99p
	o.ApmHostTop99p = all.ApmHostTop99p
	o.AuditLogsLinesIndexedSum = all.AuditLogsLinesIndexedSum
	o.AvgProfiledFargateTasks = all.AvgProfiledFargateTasks
	o.AwsHostTop99p = all.AwsHostTop99p
	o.AwsLambdaFuncCount = all.AwsLambdaFuncCount
	o.AwsLambdaInvocationsSum = all.AwsLambdaInvocationsSum
	o.AzureAppServiceTop99p = all.AzureAppServiceTop99p
	o.BillableIngestedBytesSum = all.BillableIngestedBytesSum
	o.BrowserRumLiteSessionCountSum = all.BrowserRumLiteSessionCountSum
	o.BrowserRumReplaySessionCountSum = all.BrowserRumReplaySessionCountSum
	o.BrowserRumUnitsSum = all.BrowserRumUnitsSum
	o.CiPipelineIndexedSpansSum = all.CiPipelineIndexedSpansSum
	o.CiTestIndexedSpansSum = all.CiTestIndexedSpansSum
	o.CiVisibilityPipelineCommittersHwm = all.CiVisibilityPipelineCommittersHwm
	o.CiVisibilityTestCommittersHwm = all.CiVisibilityTestCommittersHwm
	o.ContainerAvg = all.ContainerAvg
	o.ContainerHwm = all.ContainerHwm
	o.CspmAasHostTop99p = all.CspmAasHostTop99p
	o.CspmAzureHostTop99p = all.CspmAzureHostTop99p
	o.CspmContainerAvg = all.CspmContainerAvg
	o.CspmContainerHwm = all.CspmContainerHwm
	o.CspmHostTop99p = all.CspmHostTop99p
	o.CustomTsAvg = all.CustomTsAvg
	o.CwsContainerCountAvg = all.CwsContainerCountAvg
	o.CwsHostTop99p = all.CwsHostTop99p
	o.DbmHostTop99pSum = all.DbmHostTop99pSum
	o.DbmQueriesAvgSum = all.DbmQueriesAvgSum
	o.FargateTasksCountAvg = all.FargateTasksCountAvg
	o.FargateTasksCountHwm = all.FargateTasksCountHwm
	o.GcpHostTop99p = all.GcpHostTop99p
	o.HerokuHostTop99p = all.HerokuHostTop99p
	o.Id = all.Id
	o.IncidentManagementMonthlyActiveUsersHwm = all.IncidentManagementMonthlyActiveUsersHwm
	o.IndexedEventsCountSum = all.IndexedEventsCountSum
	o.InfraHostTop99p = all.InfraHostTop99p
	o.IngestedEventsBytesSum = all.IngestedEventsBytesSum
	o.IotDeviceAggSum = all.IotDeviceAggSum
	o.IotDeviceTop99pSum = all.IotDeviceTop99pSum
	o.MobileRumLiteSessionCountSum = all.MobileRumLiteSessionCountSum
	o.MobileRumSessionCountAndroidSum = all.MobileRumSessionCountAndroidSum
	o.MobileRumSessionCountIosSum = all.MobileRumSessionCountIosSum
	o.MobileRumSessionCountReactnativeSum = all.MobileRumSessionCountReactnativeSum
	o.MobileRumSessionCountSum = all.MobileRumSessionCountSum
	o.MobileRumUnitsSum = all.MobileRumUnitsSum
	o.Name = all.Name
	o.NetflowIndexedEventsCountSum = all.NetflowIndexedEventsCountSum
	o.NpmHostTop99p = all.NpmHostTop99p
	o.ObservabilityPipelinesBytesProcessedSum = all.ObservabilityPipelinesBytesProcessedSum
	o.OnlineArchiveEventsCountSum = all.OnlineArchiveEventsCountSum
	o.OpentelemetryHostTop99p = all.OpentelemetryHostTop99p
	o.ProfilingHostTop99p = all.ProfilingHostTop99p
	o.PublicId = all.PublicId
	o.RumBrowserAndMobileSessionCount = all.RumBrowserAndMobileSessionCount
	o.RumSessionCountSum = all.RumSessionCountSum
	o.RumTotalSessionCountSum = all.RumTotalSessionCountSum
	o.RumUnitsSum = all.RumUnitsSum
	o.SdsLogsScannedBytesSum = all.SdsLogsScannedBytesSum
	o.SdsTotalScannedBytesSum = all.SdsTotalScannedBytesSum
	o.SyntheticsBrowserCheckCallsCountSum = all.SyntheticsBrowserCheckCallsCountSum
	o.SyntheticsCheckCallsCountSum = all.SyntheticsCheckCallsCountSum
	o.TraceSearchIndexedEventsCountSum = all.TraceSearchIndexedEventsCountSum
	o.TwolIngestedEventsBytesSum = all.TwolIngestedEventsBytesSum
	o.VsphereHostTop99p = all.VsphereHostTop99p
	return nil
}
