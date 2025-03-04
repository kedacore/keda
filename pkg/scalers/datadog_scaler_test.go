package scalers

import (
	"context"
	"slices"
	"testing"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type datadogQueries struct {
	input   string
	output  bool
	isError bool
}

type datadogScalerType int64

const (
	apiType datadogScalerType = iota
	clusterAgentType
)

type datadogMetricIdentifier struct {
	metadataTestData *datadogAuthMetadataTestData
	typeOfScaler     datadogScalerType
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

	output := slices.Max(input)

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

var testDatadogClusterAgentMetadata = []datadogAuthMetadataTestData{
	{"", map[string]string{}, map[string]string{}, true},

	// all properly formed
	{"", map[string]string{"useClusterAgentProxy": "true", "datadogMetricName": "nginx-hits", "datadogMetricNamespace": "default", "targetValue": "2", "type": "global"}, map[string]string{"token": "token", "datadogNamespace": "datadog", "datadogMetricsService": "datadog-cluster-agent-metrics-api", "datadogMetricsServicePort": "8080", "unsafeSsl": "true", "authMode": "bearer"}, false},
	// Default Datadog service name and port
	{"", map[string]string{"useClusterAgentProxy": "true", "datadogMetricName": "nginx-hits", "datadogMetricNamespace": "default", "targetValue": "2", "type": "global"}, map[string]string{"token": "token", "datadogNamespace": "datadog", "datadogMetricsService": "datadog-cluster-agent-metrics-api", "unsafeSsl": "true", "authMode": "bearer"}, false},

	// TODO: Fix this failed test case
	// both metadata type and trigger type
	{v2.AverageValueMetricType, map[string]string{"useClusterAgentProxy": "true", "datadogMetricName": "nginx-hits", "datadogMetricNamespace": "default", "targetValue": "2", "type": "global"}, map[string]string{"token": "token", "datadogNamespace": "datadog", "datadogMetricsService": "datadog-cluster-agent-metrics-api", "unsafeSsl": "true", "authMode": "bearer"}, true},
	// missing DatadogMetric name
	{"", map[string]string{"useClusterAgentProxy": "true", "datadogMetricNamespace": "default", "targetValue": "2", "type": "global"}, map[string]string{"token": "token", "datadogNamespace": "datadog", "datadogMetricsService": "datadog-cluster-agent-metrics-api", "unsafeSsl": "true", "authMode": "bearer"}, true},
	// missing DatadogMetric namespace
	{"", map[string]string{"useClusterAgentProxy": "true", "datadogMetricName": "nginx-hits", "targetValue": "2", "type": "global"}, map[string]string{"token": "token", "datadogNamespace": "datadog", "datadogMetricsService": "datadog-cluster-agent-metrics-api", "unsafeSsl": "true", "authMode": "bearer"}, true},
	// wrong port type
	{"", map[string]string{"useClusterAgentProxy": "true", "datadogMetricName": "nginx-hits", "datadogMetricNamespace": "default", "targetValue": "2", "type": "global"}, map[string]string{"token": "token", "datadogNamespace": "datadog", "datadogMetricsService": "datadog-cluster-agent-metrics-api", "datadogMetricsServicePort": "notanint", "unsafeSsl": "true", "authMode": "bearer"}, true},
	// wrong targetValue type
	{"", map[string]string{"useClusterAgentProxy": "true", "datadogMetricName": "nginx-hits", "datadogMetricNamespace": "default", "targetValue": "notanint", "type": "global"}, map[string]string{"token": "token", "datadogNamespace": "datadog", "datadogMetricsService": "datadog-cluster-agent-metrics-api", "datadogMetricsServicePort": "8080", "unsafeSsl": "true", "authMode": "bearer"}, true},
	// wrong type
	{"", map[string]string{"useClusterAgentProxy": "true", "datadogMetricName": "nginx-hits", "datadogMetricNamespace": "default", "targetValue": "2", "type": "notatype"}, map[string]string{"token": "token", "datadogNamespace": "datadog", "datadogMetricsService": "datadog-cluster-agent-metrics-api", "datadogMetricsServicePort": "8080", "unsafeSsl": "true", "authMode": "bearer"}, true},
}

var testDatadogAPIMetadata = []datadogAuthMetadataTestData{
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

func TestDatadogScalerAPIAuthParams(t *testing.T) {
	for idx, testData := range testDatadogAPIMetadata {
		_, err := parseDatadogAPIMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams, MetricType: testData.metricType}, logr.Discard())

		if err != nil && !testData.isError {
			t.Errorf("Expected success but got error: %s for test case %d", err, idx)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error but got success for test case %d", idx)
		}
	}
}

func TestDatadogScalerClusterAgentAuthParams(t *testing.T) {
	for idx, testData := range testDatadogClusterAgentMetadata {
		_, err := parseDatadogClusterAgentMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams, MetricType: testData.metricType}, logr.Discard())

		if err != nil && !testData.isError {
			t.Errorf("Expected success but got error: %s for test case %d", err, idx)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error but got success for test case %d", idx)
		}
	}
}

var datadogMetricIdentifiers = []datadogMetricIdentifier{
	{&testDatadogAPIMetadata[1], apiType, 0, "s0-datadog-sum-trace-redis-command-hits"},
	{&testDatadogAPIMetadata[1], apiType, 1, "s1-datadog-sum-trace-redis-command-hits"},
	{&testDatadogClusterAgentMetadata[1], clusterAgentType, 0, "datadogmetric@default:nginx-hits"},
}

// TODO: Need to check whether we need to rewrite this test case because vType is long deprecated
func TestDatadogGetMetricSpecForScaling(t *testing.T) {
	var err error
	var meta *datadogMetadata

	for idx, testData := range datadogMetricIdentifiers {
		if testData.typeOfScaler == apiType {
			meta, err = parseDatadogAPIMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.metadataTestData.authParams, TriggerIndex: testData.triggerIndex, MetricType: testData.metadataTestData.metricType}, logr.Discard())
		} else {
			meta, err = parseDatadogClusterAgentMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.metadataTestData.authParams, TriggerIndex: testData.triggerIndex, MetricType: testData.metadataTestData.metricType}, logr.Discard())
		}
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}

		mockDatadogScaler := datadogScaler{
			metadata:   meta,
			apiClient:  nil,
			httpClient: nil,
		}

		metricSpec := mockDatadogScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Errorf("Wrong External metric source name:%s for test case %d", metricName, idx)
		}
	}
}

func TestBuildClusterAgentURL(t *testing.T) {
	// Test valid inputs
	url := buildClusterAgentURL("datadogMetricsService", "datadogNamespace", 8080)
	if url != "https://datadogMetricsService.datadogNamespace:8080/apis/external.metrics.k8s.io/v1beta1" {
		t.Error("Expected https://datadogMetricsService.datadogNamespace:8080/apis/external.metrics.k8s.io/v1beta1, got ", url)
	}
}

func TestBuildMetricURL(t *testing.T) {
	// Test valid inputs
	url := buildMetricURL("https://localhost:8080/apis/datadoghq.com/v1alpha1", "datadogMetricNamespace", "datadogMetricName")
	if url != "https://localhost:8080/apis/datadoghq.com/v1alpha1/namespaces/datadogMetricNamespace/datadogMetricName" {
		t.Error("Expected https://localhost:8080/apis/datadoghq.com/v1alpha1/namespaces/datadogMetricNamespace/datadogMetricName, got ", url)
	}
}
