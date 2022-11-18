// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
)

// MonthlyUsageAttributionSupportedMetrics Supported metrics for monthly usage attribution requests.
type MonthlyUsageAttributionSupportedMetrics string

// List of MonthlyUsageAttributionSupportedMetrics.
const (
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_API_USAGE                           MonthlyUsageAttributionSupportedMetrics = "api_usage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_API_PERCENTAGE                      MonthlyUsageAttributionSupportedMetrics = "api_percentage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_APM_HOST_USAGE                      MonthlyUsageAttributionSupportedMetrics = "apm_host_usage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_APM_HOST_PERCENTAGE                 MonthlyUsageAttributionSupportedMetrics = "apm_host_percentage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_APPSEC_USAGE                        MonthlyUsageAttributionSupportedMetrics = "appsec_usage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_APPSEC_PERCENTAGE                   MonthlyUsageAttributionSupportedMetrics = "appsec_percentage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_BROWSER_USAGE                       MonthlyUsageAttributionSupportedMetrics = "browser_usage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_BROWSER_PERCENTAGE                  MonthlyUsageAttributionSupportedMetrics = "browser_percentage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_CONTAINER_USAGE                     MonthlyUsageAttributionSupportedMetrics = "container_usage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_CONTAINER_PERCENTAGE                MonthlyUsageAttributionSupportedMetrics = "container_percentage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_CSPM_CONTAINERS_PERCENTAGE          MonthlyUsageAttributionSupportedMetrics = "cspm_containers_percentage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_CSPM_CONTAINERS_USAGE               MonthlyUsageAttributionSupportedMetrics = "cspm_containers_usage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_CSPM_HOSTS_PERCENTAGE               MonthlyUsageAttributionSupportedMetrics = "cspm_hosts_percentage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_CSPM_HOSTS_USAGE                    MonthlyUsageAttributionSupportedMetrics = "cspm_hosts_usage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_CUSTOM_TIMESERIES_USAGE             MonthlyUsageAttributionSupportedMetrics = "custom_timeseries_usage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_CUSTOM_TIMESERIES_PERCENTAGE        MonthlyUsageAttributionSupportedMetrics = "custom_timeseries_percentage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_CWS_CONTAINERS_PERCENTAGE           MonthlyUsageAttributionSupportedMetrics = "cws_containers_percentage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_CWS_CONTAINERS_USAGE                MonthlyUsageAttributionSupportedMetrics = "cws_containers_usage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_CWS_HOSTS_PERCENTAGE                MonthlyUsageAttributionSupportedMetrics = "cws_hosts_percentage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_CWS_HOSTS_USAGE                     MonthlyUsageAttributionSupportedMetrics = "cws_hosts_usage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_DBM_HOSTS_PERCENTAGE                MonthlyUsageAttributionSupportedMetrics = "dbm_hosts_percentage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_DBM_HOSTS_USAGE                     MonthlyUsageAttributionSupportedMetrics = "dbm_hosts_usage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_DBM_QUERIES_PERCENTAGE              MonthlyUsageAttributionSupportedMetrics = "dbm_queries_percentage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_DBM_QUERIES_USAGE                   MonthlyUsageAttributionSupportedMetrics = "dbm_queries_usage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_ESTIMATED_INDEXED_LOGS_USAGE        MonthlyUsageAttributionSupportedMetrics = "estimated_indexed_logs_usage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_ESTIMATED_INDEXED_LOGS_PERCENTAGE   MonthlyUsageAttributionSupportedMetrics = "estimated_indexed_logs_percentage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_ESTIMATED_INDEXED_SPANS_USAGE       MonthlyUsageAttributionSupportedMetrics = "estimated_indexed_spans_usage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_ESTIMATED_INDEXED_SPANS_PERCENTAGE  MonthlyUsageAttributionSupportedMetrics = "estimated_indexed_spans_percentage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_ESTIMATED_INGESTED_SPANS_USAGE      MonthlyUsageAttributionSupportedMetrics = "estimated_ingested_spans_usage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_ESTIMATED_INGESTED_SPANS_PERCENTAGE MonthlyUsageAttributionSupportedMetrics = "estimated_ingested_spans_percentage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_FARGATE_USAGE                       MonthlyUsageAttributionSupportedMetrics = "fargate_usage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_FARGATE_PERCENTAGE                  MonthlyUsageAttributionSupportedMetrics = "fargate_percentage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_FUNCTIONS_USAGE                     MonthlyUsageAttributionSupportedMetrics = "functions_usage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_FUNCTIONS_PERCENTAGE                MonthlyUsageAttributionSupportedMetrics = "functions_percentage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_INDEXED_LOGS_USAGE                  MonthlyUsageAttributionSupportedMetrics = "indexed_logs_usage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_INDEXED_LOGS_PERCENTAGE             MonthlyUsageAttributionSupportedMetrics = "indexed_logs_percentage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_INFRA_HOST_USAGE                    MonthlyUsageAttributionSupportedMetrics = "infra_host_usage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_INFRA_HOST_PERCENTAGE               MonthlyUsageAttributionSupportedMetrics = "infra_host_percentage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_INVOCATIONS_USAGE                   MonthlyUsageAttributionSupportedMetrics = "invocations_usage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_INVOCATIONS_PERCENTAGE              MonthlyUsageAttributionSupportedMetrics = "invocations_percentage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_NPM_HOST_USAGE                      MonthlyUsageAttributionSupportedMetrics = "npm_host_usage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_NPM_HOST_PERCENTAGE                 MonthlyUsageAttributionSupportedMetrics = "npm_host_percentage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_PROFILED_CONTAINER_USAGE            MonthlyUsageAttributionSupportedMetrics = "profiled_container_usage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_PROFILED_CONTAINER_PERCENTAGE       MonthlyUsageAttributionSupportedMetrics = "profiled_container_percentage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_PROFILED_HOST_USAGE                 MonthlyUsageAttributionSupportedMetrics = "profiled_host_usage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_PROFILED_HOST_PERCENTAGE            MonthlyUsageAttributionSupportedMetrics = "profiled_host_percentage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_SNMP_USAGE                          MonthlyUsageAttributionSupportedMetrics = "snmp_usage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_SNMP_PERCENTAGE                     MonthlyUsageAttributionSupportedMetrics = "snmp_percentage"
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_ALL                                 MonthlyUsageAttributionSupportedMetrics = "*"
)

var allowedMonthlyUsageAttributionSupportedMetricsEnumValues = []MonthlyUsageAttributionSupportedMetrics{
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_API_USAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_API_PERCENTAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_APM_HOST_USAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_APM_HOST_PERCENTAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_APPSEC_USAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_APPSEC_PERCENTAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_BROWSER_USAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_BROWSER_PERCENTAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_CONTAINER_USAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_CONTAINER_PERCENTAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_CSPM_CONTAINERS_PERCENTAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_CSPM_CONTAINERS_USAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_CSPM_HOSTS_PERCENTAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_CSPM_HOSTS_USAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_CUSTOM_TIMESERIES_USAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_CUSTOM_TIMESERIES_PERCENTAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_CWS_CONTAINERS_PERCENTAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_CWS_CONTAINERS_USAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_CWS_HOSTS_PERCENTAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_CWS_HOSTS_USAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_DBM_HOSTS_PERCENTAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_DBM_HOSTS_USAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_DBM_QUERIES_PERCENTAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_DBM_QUERIES_USAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_ESTIMATED_INDEXED_LOGS_USAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_ESTIMATED_INDEXED_LOGS_PERCENTAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_ESTIMATED_INDEXED_SPANS_USAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_ESTIMATED_INDEXED_SPANS_PERCENTAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_ESTIMATED_INGESTED_SPANS_USAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_ESTIMATED_INGESTED_SPANS_PERCENTAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_FARGATE_USAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_FARGATE_PERCENTAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_FUNCTIONS_USAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_FUNCTIONS_PERCENTAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_INDEXED_LOGS_USAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_INDEXED_LOGS_PERCENTAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_INFRA_HOST_USAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_INFRA_HOST_PERCENTAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_INVOCATIONS_USAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_INVOCATIONS_PERCENTAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_NPM_HOST_USAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_NPM_HOST_PERCENTAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_PROFILED_CONTAINER_USAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_PROFILED_CONTAINER_PERCENTAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_PROFILED_HOST_USAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_PROFILED_HOST_PERCENTAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_SNMP_USAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_SNMP_PERCENTAGE,
	MONTHLYUSAGEATTRIBUTIONSUPPORTEDMETRICS_ALL,
}

// GetAllowedValues reeturns the list of possible values.
func (v *MonthlyUsageAttributionSupportedMetrics) GetAllowedValues() []MonthlyUsageAttributionSupportedMetrics {
	return allowedMonthlyUsageAttributionSupportedMetricsEnumValues
}

// UnmarshalJSON deserializes the given payload.
func (v *MonthlyUsageAttributionSupportedMetrics) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	*v = MonthlyUsageAttributionSupportedMetrics(value)
	return nil
}

// NewMonthlyUsageAttributionSupportedMetricsFromValue returns a pointer to a valid MonthlyUsageAttributionSupportedMetrics
// for the value passed as argument, or an error if the value passed is not allowed by the enum.
func NewMonthlyUsageAttributionSupportedMetricsFromValue(v string) (*MonthlyUsageAttributionSupportedMetrics, error) {
	ev := MonthlyUsageAttributionSupportedMetrics(v)
	if ev.IsValid() {
		return &ev, nil
	}
	return nil, fmt.Errorf("invalid value '%v' for MonthlyUsageAttributionSupportedMetrics: valid values are %v", v, allowedMonthlyUsageAttributionSupportedMetricsEnumValues)
}

// IsValid return true if the value is valid for the enum, false otherwise.
func (v MonthlyUsageAttributionSupportedMetrics) IsValid() bool {
	for _, existing := range allowedMonthlyUsageAttributionSupportedMetricsEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to MonthlyUsageAttributionSupportedMetrics value.
func (v MonthlyUsageAttributionSupportedMetrics) Ptr() *MonthlyUsageAttributionSupportedMetrics {
	return &v
}

// NullableMonthlyUsageAttributionSupportedMetrics handles when a null is used for MonthlyUsageAttributionSupportedMetrics.
type NullableMonthlyUsageAttributionSupportedMetrics struct {
	value *MonthlyUsageAttributionSupportedMetrics
	isSet bool
}

// Get returns the associated value.
func (v NullableMonthlyUsageAttributionSupportedMetrics) Get() *MonthlyUsageAttributionSupportedMetrics {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableMonthlyUsageAttributionSupportedMetrics) Set(val *MonthlyUsageAttributionSupportedMetrics) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableMonthlyUsageAttributionSupportedMetrics) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag.
func (v *NullableMonthlyUsageAttributionSupportedMetrics) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableMonthlyUsageAttributionSupportedMetrics initializes the struct as if Set has been called.
func NewNullableMonthlyUsageAttributionSupportedMetrics(val *MonthlyUsageAttributionSupportedMetrics) *NullableMonthlyUsageAttributionSupportedMetrics {
	return &NullableMonthlyUsageAttributionSupportedMetrics{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableMonthlyUsageAttributionSupportedMetrics) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableMonthlyUsageAttributionSupportedMetrics) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
