package scalers

import (
	"context"
	"testing"
)

type datadogMetricIdentifier struct {
	metadataTestData *datadogAuthMetadataTestData
	scalerIndex      int
	name             string
}

type datadogAuthMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

var testDatadogMetadata = []datadogAuthMetadataTestData{
	{map[string]string{}, map[string]string{}, true},

	// all properly formed
	{map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "metricName": "redis-hits", "type": "average", "age": "60"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "ddSite": "ddSite"}, false},
	// default age
	{map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "metricName": "redis-hits", "type": "average"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "ddSite": "ddSite"}, false},
	// default type
	{map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "metricName": "redis-hits", "age": "60"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "ddSite": "ddSite"}, false},
	// missing query
	{map[string]string{"queryValue": "7", "metricName": "redis-hits", "type": "average", "age": "60"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "ddSite": "ddSite"}, true},
	// missing queryValue
	{map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "metricName": "redis-hits", "type": "average", "age": "60"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "ddSite": "ddSite"}, true},
	// missing metricName
	{map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "type": "average", "age": "60"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "ddSite": "ddSite"}, true},
	// wrong query value type
	{map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "notanint", "metricName": "redis-hits", "type": "average", "age": "60"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "ddSite": "ddSite"}, true},

	// success api/app keys
	{map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "metricName": "redis-hits"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "ddSite": "ddSite"}, false},
	// default ddSite
	{map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "metricName": "redis-hits"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey"}, false},
	// missing apiKey
	{map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "metricName": "redis-hits"}, map[string]string{"appKey": "appKey"}, true},
	// missing appKey
	{map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "metricName": "redis-hits"}, map[string]string{"apiKey": "apiKey"}, true},
}

func TestDatadogScalerAuthParams(t *testing.T) {
	for _, testData := range testDatadogMetadata {
		_, err := parseDatadogMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

var datadogMetricIdentifiers = []datadogMetricIdentifier{
	{&testDatadogMetadata[1], 0, "s0-datadog-redis-hits"},
	{&testDatadogMetadata[1], 1, "s1-datadog-redis-hits"},
}

func TestDatadogGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range datadogMetricIdentifiers {
		meta, err := parseDatadogMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.metadataTestData.authParams, ScalerIndex: testData.scalerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockDatadogScaler := datadogScaler{
			metadata:  meta,
			apiClient: nil,
		}

		metricSpec := mockDatadogScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
