// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// UsageAttributionSupportedMetrics Supported fields for usage attribution requests (valid requests contain one or more metrics, or `*` for all).
type UsageAttributionSupportedMetrics string

// List of UsageAttributionSupportedMetrics.
const (
	USAGEATTRIBUTIONSUPPORTEDMETRICS_CUSTOM_TIMESERIES_USAGE             UsageAttributionSupportedMetrics = "custom_timeseries_usage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_CONTAINER_USAGE                     UsageAttributionSupportedMetrics = "container_usage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_SNMP_PERCENTAGE                     UsageAttributionSupportedMetrics = "snmp_percentage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_APM_HOST_USAGE                      UsageAttributionSupportedMetrics = "apm_host_usage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_BROWSER_USAGE                       UsageAttributionSupportedMetrics = "browser_usage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_NPM_HOST_PERCENTAGE                 UsageAttributionSupportedMetrics = "npm_host_percentage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_INFRA_HOST_USAGE                    UsageAttributionSupportedMetrics = "infra_host_usage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_CUSTOM_TIMESERIES_PERCENTAGE        UsageAttributionSupportedMetrics = "custom_timeseries_percentage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_CONTAINER_PERCENTAGE                UsageAttributionSupportedMetrics = "container_percentage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_API_USAGE                           UsageAttributionSupportedMetrics = "api_usage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_APM_HOST_PERCENTAGE                 UsageAttributionSupportedMetrics = "apm_host_percentage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_INFRA_HOST_PERCENTAGE               UsageAttributionSupportedMetrics = "infra_host_percentage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_SNMP_USAGE                          UsageAttributionSupportedMetrics = "snmp_usage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_BROWSER_PERCENTAGE                  UsageAttributionSupportedMetrics = "browser_percentage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_API_PERCENTAGE                      UsageAttributionSupportedMetrics = "api_percentage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_NPM_HOST_USAGE                      UsageAttributionSupportedMetrics = "npm_host_usage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_LAMBDA_FUNCTIONS_USAGE              UsageAttributionSupportedMetrics = "lambda_functions_usage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_LAMBDA_FUNCTIONS_PERCENTAGE         UsageAttributionSupportedMetrics = "lambda_functions_percentage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_LAMBDA_INVOCATIONS_USAGE            UsageAttributionSupportedMetrics = "lambda_invocations_usage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_LAMBDA_INVOCATIONS_PERCENTAGE       UsageAttributionSupportedMetrics = "lambda_invocations_percentage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_FARGATE_USAGE                       UsageAttributionSupportedMetrics = "fargate_usage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_FARGATE_PERCENTAGE                  UsageAttributionSupportedMetrics = "fargate_percentage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_PROFILED_HOST_USAGE                 UsageAttributionSupportedMetrics = "profiled_host_usage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_PROFILED_HOST_PERCENTAGE            UsageAttributionSupportedMetrics = "profiled_host_percentage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_PROFILED_CONTAINER_USAGE            UsageAttributionSupportedMetrics = "profiled_container_usage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_PROFILED_CONTAINER_PERCENTAGE       UsageAttributionSupportedMetrics = "profiled_container_percentage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_DBM_HOSTS_USAGE                     UsageAttributionSupportedMetrics = "dbm_hosts_usage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_DBM_HOSTS_PERCENTAGE                UsageAttributionSupportedMetrics = "dbm_hosts_percentage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_DBM_QUERIES_USAGE                   UsageAttributionSupportedMetrics = "dbm_queries_usage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_DBM_QUERIES_PERCENTAGE              UsageAttributionSupportedMetrics = "dbm_queries_percentage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_ESTIMATED_INDEXED_LOGS_USAGE        UsageAttributionSupportedMetrics = "estimated_indexed_logs_usage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_ESTIMATED_INDEXED_LOGS_PERCENTAGE   UsageAttributionSupportedMetrics = "estimated_indexed_logs_percentage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_APPSEC_USAGE                        UsageAttributionSupportedMetrics = "appsec_usage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_APPSEC_PERCENTAGE                   UsageAttributionSupportedMetrics = "appsec_percentage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_ESTIMATED_INDEXED_SPANS_USAGE       UsageAttributionSupportedMetrics = "estimated_indexed_spans_usage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_ESTIMATED_INDEXED_SPANS_PERCENTAGE  UsageAttributionSupportedMetrics = "estimated_indexed_spans_percentage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_ESTIMATED_INGESTED_SPANS_USAGE      UsageAttributionSupportedMetrics = "estimated_ingested_spans_usage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_ESTIMATED_INGESTED_SPANS_PERCENTAGE UsageAttributionSupportedMetrics = "estimated_ingested_spans_percentage"
	USAGEATTRIBUTIONSUPPORTEDMETRICS_ALL                                 UsageAttributionSupportedMetrics = "*"
)

var allowedUsageAttributionSupportedMetricsEnumValues = []UsageAttributionSupportedMetrics{
	USAGEATTRIBUTIONSUPPORTEDMETRICS_CUSTOM_TIMESERIES_USAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_CONTAINER_USAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_SNMP_PERCENTAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_APM_HOST_USAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_BROWSER_USAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_NPM_HOST_PERCENTAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_INFRA_HOST_USAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_CUSTOM_TIMESERIES_PERCENTAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_CONTAINER_PERCENTAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_API_USAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_APM_HOST_PERCENTAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_INFRA_HOST_PERCENTAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_SNMP_USAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_BROWSER_PERCENTAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_API_PERCENTAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_NPM_HOST_USAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_LAMBDA_FUNCTIONS_USAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_LAMBDA_FUNCTIONS_PERCENTAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_LAMBDA_INVOCATIONS_USAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_LAMBDA_INVOCATIONS_PERCENTAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_FARGATE_USAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_FARGATE_PERCENTAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_PROFILED_HOST_USAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_PROFILED_HOST_PERCENTAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_PROFILED_CONTAINER_USAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_PROFILED_CONTAINER_PERCENTAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_DBM_HOSTS_USAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_DBM_HOSTS_PERCENTAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_DBM_QUERIES_USAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_DBM_QUERIES_PERCENTAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_ESTIMATED_INDEXED_LOGS_USAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_ESTIMATED_INDEXED_LOGS_PERCENTAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_APPSEC_USAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_APPSEC_PERCENTAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_ESTIMATED_INDEXED_SPANS_USAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_ESTIMATED_INDEXED_SPANS_PERCENTAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_ESTIMATED_INGESTED_SPANS_USAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_ESTIMATED_INGESTED_SPANS_PERCENTAGE,
	USAGEATTRIBUTIONSUPPORTEDMETRICS_ALL,
}

// GetAllowedValues reeturns the list of possible values.
func (v *UsageAttributionSupportedMetrics) GetAllowedValues() []UsageAttributionSupportedMetrics {
	return allowedUsageAttributionSupportedMetricsEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *UsageAttributionSupportedMetrics) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = UsageAttributionSupportedMetrics(value)
	return nil
}

// NewUsageAttributionSupportedMetricsFromValue returns a pointer to a valid UsageAttributionSupportedMetrics
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewUsageAttributionSupportedMetricsFromValue(v string) (*UsageAttributionSupportedMetrics, error) {
	ev := UsageAttributionSupportedMetrics(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for UsageAttributionSupportedMetrics: valid values are %v", v, allowedUsageAttributionSupportedMetricsEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v UsageAttributionSupportedMetrics) IsValid() bool {
	for _, existing := range allowedUsageAttributionSupportedMetricsEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to UsageAttributionSupportedMetrics value.
func (v UsageAttributionSupportedMetrics) Ptr() *UsageAttributionSupportedMetrics {
	return &v
}

// NullableUsageAttributionSupportedMetrics handles when a null is used for UsageAttributionSupportedMetrics.
type NullableUsageAttributionSupportedMetrics struct {
	value *UsageAttributionSupportedMetrics
	isSet bool
}

// Get returns the associated value.
func (v NullableUsageAttributionSupportedMetrics) Get() *UsageAttributionSupportedMetrics {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableUsageAttributionSupportedMetrics) Set(val *UsageAttributionSupportedMetrics) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableUsageAttributionSupportedMetrics) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableUsageAttributionSupportedMetrics) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableUsageAttributionSupportedMetrics initializes the struct as if Set has been called.
func NewNullableUsageAttributionSupportedMetrics(val *UsageAttributionSupportedMetrics) *NullableUsageAttributionSupportedMetrics {
	return &NullableUsageAttributionSupportedMetrics{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableUsageAttributionSupportedMetrics) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableUsageAttributionSupportedMetrics) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
