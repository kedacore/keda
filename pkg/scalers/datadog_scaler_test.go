package scalers

import (
	"context"
	"testing"
)

type datadogQueries struct {
	input   string
	age     int
	output  string
	isError bool
}

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

var testParseQueries = []datadogQueries{
	{"", 0, "", true},
	// All properly formed
	{"avg:system.cpu.user{*}.rollup(sum, 30)", 120, "avg:system.cpu.user{*}.rollup(sum, 30)", false},
	{"sum:system.cpu.user{*}.rollup(30)", 30, "sum:system.cpu.user{*}.rollup(30)", false},
	{"avg:system.cpu.user{automatic-restart:false,bosh_address:192.168.101.12}.rollup(avg, 30)", 120, "avg:system.cpu.user{automatic-restart:false,bosh_address:192.168.101.12}.rollup(avg, 30)", false},
	{"per_second(sum:system.cpu.user{*}.rollup(avg, 30))", 120, "per_second(sum:system.cpu.user{*}.rollup(avg, 30))", false},
	{"log10(sum:system.cpu.user{*}.rollup(avg, 30))", 120, "log10(sum:system.cpu.user{*}.rollup(avg, 30))", false},
	// Multiple functions
	{"top(per_second(abs(sum:http.requests{*}.rollup(max, 2))), 5, 'mean', 'desc')", 120, "top(per_second(abs(sum:http.requests{*}.rollup(max, 2))), 5, 'mean', 'desc')", false},

	// Default aggregator
	{"system.cpu.user{*}.rollup(sum, 30)", 120, "avg:system.cpu.user{*}.rollup(sum, 30)", false},

	// Default rollup
	{"min:system.cpu.user{*}", 120, "min:system.cpu.user{*}.rollup(avg, 24)", false},

	// Missing filter
	{"min:system.cpu.user", 120, "", true},

	// Malformed rollup
	{"min:system.cpu.user{*}.rollup(avg)", 120, "", true},

	// Malformed function wrapper -- missing end bracket
	{"per_second(sum:system.cpu.user{*}.rollup(avg, 30)", 120, "", true},
}

func TestDatadogScalerParseQueries(t *testing.T) {
	for _, testData := range testParseQueries {
		output, err := parseAndTransformDatadogQuery(testData.input, testData.age)

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}

		if output != testData.output {
			t.Errorf("Expected %s, got %s", testData.output, output)
		}
	}
}

var testDatadogMetadata = []datadogAuthMetadataTestData{
	{map[string]string{}, map[string]string{}, true},

	// all properly formed
	{map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "metricUnavailableValue": "1.5", "type": "average", "age": "60"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, false},
	// default age
	{map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "type": "average"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, false},
	// default type
	{map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "age": "60"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, false},
	// missing query
	{map[string]string{"queryValue": "7", "type": "average", "age": "60"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, true},
	// missing queryValue
	{map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "type": "average", "age": "60"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, true},
	// wrong query value type
	{map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "notanint", "type": "average", "age": "60"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, true},
	// malformed query
	{map[string]string{"query": "sum:trace.redis.command.hits", "queryValue": "7", "type": "average", "age": "60"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, true},
	// wrong unavailableMetricValue type
	{map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "metricUnavailableValue": "notafloat", "type": "average", "age": "60"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, true},

	// success api/app keys
	{map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, false},
	// default datadogSite
	{map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey"}, false},
	// missing apiKey
	{map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7"}, map[string]string{"appKey": "appKey"}, true},
	// missing appKey
	{map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7"}, map[string]string{"apiKey": "apiKey"}, true},
	// invalid query missing {
	{map[string]string{"query": "sum:trace.redis.command.hits.as_count()", "queryValue": "7"}, map[string]string{}, true},
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
	{&testDatadogMetadata[1], 0, "s0-datadog-sum-trace-redis-command-hits"},
	{&testDatadogMetadata[1], 1, "s1-datadog-sum-trace-redis-command-hits"},
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
