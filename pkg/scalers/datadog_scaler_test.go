package scalers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
)

type datadogQueries struct {
	input   string
	output  bool
	isError bool
}

type datadogMetricIdentifier struct {
	metadataTestData *datadogAuthMetadataTestData
	triggerIndex     int
	name             string
}

type datadogAuthMetadataTestData struct {
	metricType v2.MetricTargetType
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

func assertEqual(t *testing.T, a interface{}, b interface{}) {
	if a == b {
		return
	}
	t.Errorf("%v != %v", a, b)
}

func TestMaxFloatFromSlice(t *testing.T) {
	input := []float64{1.0, 2.0, 3.0, 4.0}
	expectedOutput := float64(4.0)

	output := MaxFloatFromSlice(input)

	assertEqual(t, output, expectedOutput)
}

func TestAvgFloatFromSlice(t *testing.T) {
	input := []float64{1.0, 2.0, 3.0, 4.0}
	expectedOutput := float64(2.5)

	output := AvgFloatFromSlice(input)

	assertEqual(t, output, expectedOutput)
}

var testParseQueries = []datadogQueries{
	{"", false, true},
	// All properly formed
	{"avg:system.cpu.user{*}.rollup(sum, 30)", true, false},
	{"sum:system.cpu.user{*}.rollup(30)", true, false},
	{"avg:system.cpu.user{automatic-restart:false,bosh_address:192.168.101.12}.rollup(avg, 30)", true, false},
	{"top(per_second(abs(sum:http.requests{service:myapp,dc:us-west-2}.rollup(max, 2))), 5, 'mean', 'desc')", true, false},
	{"system.cpu.user{*}.rollup(sum, 30)", true, false},
	{"min:system.cpu.user{*}", true, false},
	// Multi-query
	{"avg:system.cpu.user{*}.rollup(sum, 30),sum:system.cpu.user{*}.rollup(30)", true, false},

	// Missing filter
	{"min:system.cpu.user", false, true},

	// Find out last point with value
	{"sum:trace.express.request.hits{*}.as_rate()/avg:kubernetes.cpu.requests{*}.rollup(10)", true, false},
}

func TestDatadogScalerParseQueries(t *testing.T) {
	for _, testData := range testParseQueries {
		output, err := parseDatadogQuery(testData.input)

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}

		if output != testData.output {
			t.Errorf("Expected %v, got %v", testData.output, output)
		}
	}
}

var testDatadogMetadata = []datadogAuthMetadataTestData{
	{"", map[string]string{}, map[string]string{}, true},

	// all properly formed
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "metricUnavailableValue": "1.5", "type": "average", "age": "60", "timeWindowOffset": "30", "lastAvailablePointOffset": "1"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, false},
	// Multi-query all properly formed
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count(),sum:trace.redis.command.hits{env:none,service:redis}.as_count()/2", "queryValue": "7", "queryAggregator": "average", "metricUnavailableValue": "1.5", "type": "average", "age": "60"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, false},
	// default age
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "type": "average"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, false},
	// default timeWindowOffset
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "metricUnavailableValue": "1.5", "type": "average", "age": "60", "lastAvailablePointOffset": "1"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, false},
	// default lastAvailablePointOffset
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "metricUnavailableValue": "1.5", "type": "average", "age": "60", "timeWindowOffset": "30"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, false},
	// default type
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "age": "60"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, false},
	// wrong type
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "type": "invalid", "age": "60"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, true},
	// both metadata type and trigger type
	{v2.AverageValueMetricType, map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "type": "average", "age": "60"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, true},
	// missing query
	{"", map[string]string{"queryValue": "7", "type": "average", "age": "60"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, true},
	// missing queryValue
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "type": "average", "age": "60"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, true},
	// wrong query value type
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "notanint", "type": "average", "age": "60"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, true},
	// wrong queryAggregator value
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "notanint", "queryAggegrator": "1.0", "type": "average", "age": "60"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, true},
	// wrong activation query value type
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "1", "activationQueryValue": "notanint", "type": "average", "age": "60"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, true},
	// malformed query
	{"", map[string]string{"query": "sum:trace.redis.command.hits", "queryValue": "7", "type": "average", "age": "60"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, true},
	// wrong unavailableMetricValue type
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "metricUnavailableValue": "notafloat", "type": "average", "age": "60"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, true},
	// success api/app keys
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, false},
	// default datadogSite
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey"}, false},
	// missing apiKey
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7"}, map[string]string{"appKey": "appKey"}, true},
	// missing appKey
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7"}, map[string]string{"apiKey": "apiKey"}, true},
	// invalid query missing {
	{"", map[string]string{"query": "sum:trace.redis.command.hits.as_count()", "queryValue": "7"}, map[string]string{}, true},
}

func TestDatadogScalerAuthParams(t *testing.T) {
	for _, testData := range testDatadogMetadata {
		_, err := parseDatadogMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams, MetricType: testData.metricType}, logr.Discard())

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
		meta, err := parseDatadogMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.metadataTestData.authParams, TriggerIndex: testData.triggerIndex, MetricType: testData.metadataTestData.metricType}, logr.Discard())
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
