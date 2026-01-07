package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	datadog "github.com/DataDog/datadog-api-client-go/api/v1/datadog"
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
	// Test case with different datadogNamespace and datadogMetricNamespace to ensure the correct namespace is used in URL
	{"", map[string]string{"useClusterAgentProxy": "true", "datadogMetricName": "test-metric", "datadogMetricNamespace": "application-metrics", "targetValue": "10"}, map[string]string{"token": "test-token", "datadogNamespace": "datadog-system", "datadogMetricsService": "datadog-cluster-agent-metrics-api", "datadogMetricsServicePort": "8443", "authMode": "bearer"}, false},
	// Test case with custom service name and port to verify URL building
	{"", map[string]string{"useClusterAgentProxy": "true", "datadogMetricName": "custom-metric", "datadogMetricNamespace": "prod-metrics", "targetValue": "5"}, map[string]string{"token": "test-token", "datadogNamespace": "monitoring", "datadogMetricsService": "custom-datadog-service", "datadogMetricsServicePort": "9443", "authMode": "bearer"}, false},
	// valid timeout values for cluster agent
	{"", map[string]string{"useClusterAgentProxy": "true", "datadogMetricName": "nginx-hits", "datadogMetricNamespace": "default", "targetValue": "2", "timeout": "30s"}, map[string]string{"token": "token", "datadogNamespace": "datadog", "datadogMetricsService": "datadog-cluster-agent-metrics-api", "authMode": "bearer"}, false},
	{"", map[string]string{"useClusterAgentProxy": "true", "datadogMetricName": "nginx-hits", "datadogMetricNamespace": "default", "targetValue": "2", "timeout": "1m"}, map[string]string{"token": "token", "datadogNamespace": "datadog", "datadogMetricsService": "datadog-cluster-agent-metrics-api", "authMode": "bearer"}, false},
	{"", map[string]string{"useClusterAgentProxy": "true", "datadogMetricName": "nginx-hits", "datadogMetricNamespace": "default", "targetValue": "2", "timeout": "2m30s"}, map[string]string{"token": "token", "datadogNamespace": "datadog", "datadogMetricsService": "datadog-cluster-agent-metrics-api", "authMode": "bearer"}, false},
	{"", map[string]string{"useClusterAgentProxy": "true", "datadogMetricName": "nginx-hits", "datadogMetricNamespace": "default", "targetValue": "2", "timeout": "500ms"}, map[string]string{"token": "token", "datadogNamespace": "datadog", "datadogMetricsService": "datadog-cluster-agent-metrics-api", "authMode": "bearer"}, false},
	{"", map[string]string{"useClusterAgentProxy": "true", "datadogMetricName": "nginx-hits", "datadogMetricNamespace": "default", "targetValue": "2", "timeout": "0s"}, map[string]string{"token": "token", "datadogNamespace": "datadog", "datadogMetricsService": "datadog-cluster-agent-metrics-api", "authMode": "bearer"}, false},
	{"", map[string]string{"useClusterAgentProxy": "true", "datadogMetricName": "nginx-hits", "datadogMetricNamespace": "default", "targetValue": "2", "timeout": "30"}, map[string]string{"token": "token", "datadogNamespace": "datadog", "datadogMetricsService": "datadog-cluster-agent-metrics-api", "authMode": "bearer"}, false},
	{"", map[string]string{"useClusterAgentProxy": "true", "datadogMetricName": "nginx-hits", "datadogMetricNamespace": "default", "targetValue": "2", "timeout": ""}, map[string]string{"token": "token", "datadogNamespace": "datadog", "datadogMetricsService": "datadog-cluster-agent-metrics-api", "authMode": "bearer"}, false},
	// invalid timeout values for cluster agent
	{"", map[string]string{"useClusterAgentProxy": "true", "datadogMetricName": "nginx-hits", "datadogMetricNamespace": "default", "targetValue": "2", "timeout": "invalid"}, map[string]string{"token": "token", "datadogNamespace": "datadog", "datadogMetricsService": "datadog-cluster-agent-metrics-api", "authMode": "bearer"}, true},
	{"", map[string]string{"useClusterAgentProxy": "true", "datadogMetricName": "nginx-hits", "datadogMetricNamespace": "default", "targetValue": "2", "timeout": "-10s"}, map[string]string{"token": "token", "datadogNamespace": "datadog", "datadogMetricsService": "datadog-cluster-agent-metrics-api", "authMode": "bearer"}, true},
}

var testDatadogAPIMetadata = []datadogAuthMetadataTestData{
	{"", map[string]string{}, map[string]string{}, true},

	// all properly formed
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "metricUnavailableValue": "1.5", "type": "average", "age": "60", "timeWindowOffset": "30", "lastAvailablePointOffset": "1"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, false},
	// valid timeout values
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "timeout": "30s"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, false},
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "timeout": "1m"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, false},
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "timeout": "2m30s"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, false},
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "timeout": "500ms"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, false},
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "timeout": "0s"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, false},
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "timeout": "30"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, false},
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "timeout": ""}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, false},
	// invalid timeout values
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "timeout": "invalid"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, true},
	{"", map[string]string{"query": "sum:trace.redis.command.hits{env:none,service:redis}.as_count()", "queryValue": "7", "timeout": "-5s"}, map[string]string{"apiKey": "apiKey", "appKey": "appKey", "datadogSite": "datadogSite"}, true},
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

// Helper function to create metadata and validate
func createAndValidateMetadata(testData *datadogAuthMetadataTestData, useClusterAgent bool, triggerIndex int) (*datadogMetadata, error) {
	config := &scalersconfig.ScalerConfig{
		TriggerMetadata: testData.metadata,
		AuthParams:      testData.authParams,
		MetricType:      testData.metricType,
		TriggerIndex:    triggerIndex,
	}

	meta := &datadogMetadata{}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing metadata: %w", err)
	}

	meta.TriggerIndex = config.TriggerIndex

	if meta.Timeout == 0 {
		meta.Timeout = config.GlobalHTTPTimeout
	}

	var err error
	if useClusterAgent {
		err = validateClusterAgentMetadata(meta, config, logr.Discard())
	} else {
		err = validateAPIMetadata(meta, config, logr.Discard())
	}

	if err != nil {
		return nil, err
	}

	return meta, nil
}

func TestDatadogScalerAPIAuthParams(t *testing.T) {
	for idx, testData := range testDatadogAPIMetadata {
		_, err := createAndValidateMetadata(&testData, false, 0)

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
		meta, err := createAndValidateMetadata(&testData, true, 0)

		if err != nil && !testData.isError {
			t.Errorf("Expected success but got error: %s for test case %d", err, idx)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error but got success for test case %d", idx)
		}

		// Additional validation for URL building when we have valid metadata
		// This validates that datadogNamespace is used correctly in URL building (issue #6769)
		if !testData.isError && meta != nil {
			datadogNamespace := testData.authParams["datadogNamespace"]
			datadogMetricNamespace := testData.metadata["datadogMetricNamespace"]

			if datadogNamespace != "" && datadogMetricNamespace != "" {
				// Verify that the URL contains the service namespace (datadogNamespace), not the metric namespace
				if !strings.Contains(meta.DatadogMetricServiceURL, datadogNamespace) {
					t.Errorf("Test case %d: DatadogMetricServiceURL should contain datadogNamespace '%s', but got %s", idx, datadogNamespace, meta.DatadogMetricServiceURL)
				}
				// When namespaces are different, ensure metric namespace is NOT used in the service URL
				if datadogNamespace != datadogMetricNamespace {
					datadogMetricsService := testData.authParams["datadogMetricsService"]
					datadogMetricsServicePort := testData.authParams["datadogMetricsServicePort"]

					incorrectURL := fmt.Sprintf("https://%s.%s:%s/apis/external.metrics.k8s.io/v1beta1",
						datadogMetricsService, datadogMetricNamespace, datadogMetricsServicePort)

					if meta.DatadogMetricServiceURL == incorrectURL {
						t.Errorf("Test case %d: Bug detected - DatadogMetricServiceURL incorrectly uses datadogMetricNamespace instead of datadogNamespace. Got %s", idx, meta.DatadogMetricServiceURL)
					}
				}
			}
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
	for idx, testData := range datadogMetricIdentifiers {
		useClusterAgent := testData.typeOfScaler == clusterAgentType
		meta, err := createAndValidateMetadata(testData.metadataTestData, useClusterAgent, testData.triggerIndex)
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
			t.Errorf("Wrong External metric source name:%s for test case %d, expected: %s", metricName, idx, testData.name)
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

func TestDatadogMetadataValidateUseFiller(t *testing.T) {
	testCases := []struct {
		name                   string
		metricUnavailableValue string
		useClusterAgent        bool
		expectedUseFiller      bool
		expectedFillValue      float64
	}{
		// API metadata tests
		{"API: Not configured", "", false, false, 0},
		{"API: Explicitly set to 0", "0", false, true, 0},
		{"API: Positive value", "1.5", false, true, 1.5},
		{"API: Negative value", "-1.0", false, true, -1},
		{"API: Small positive value", "0.1", false, true, 0.1},

		// Cluster Agent metadata tests
		{"ClusterAgent: Not configured", "", true, false, 0},
		{"ClusterAgent: Explicitly set to 0", "0", true, true, 0},
		{"ClusterAgent: Positive value", "1.5", true, true, 1.5},
		{"ClusterAgent: Negative value", "-1.0", true, true, -1},
		{"ClusterAgent: Small positive value", "0.1", true, true, 0.1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testData := &datadogAuthMetadataTestData{
				metadata:   map[string]string{},
				authParams: map[string]string{},
				isError:    false,
			}

			// Set required metadata based on mode
			if tc.useClusterAgent {
				// Cluster Agent mode requires different metadata
				testData.authParams["datadogMetricsService"] = "datadog-metrics-service"
				testData.authParams["datadogNamespace"] = "default"
				testData.metadata["datadogMetricName"] = "test-metric"
				testData.metadata["datadogMetricNamespace"] = "test-namespace"
				testData.metadata["queryValue"] = "7"
			} else {
				// API mode requires query and credentials
				testData.metadata["query"] = "sum:trace.redis.command.hits{env:none,service:redis}.as_count()"
				testData.metadata["queryValue"] = "7"
				testData.authParams["apiKey"] = "apiKey"
				testData.authParams["appKey"] = "appKey"
				testData.authParams["datadogSite"] = "datadogSite"
			}

			if tc.metricUnavailableValue != "" {
				testData.metadata["metricUnavailableValue"] = tc.metricUnavailableValue
			}

			meta, err := createAndValidateMetadata(testData, tc.useClusterAgent, 0)
			if err != nil {
				t.Errorf("Validate() unexpected error = %v", err)
			}
			if meta.UseFiller != tc.expectedUseFiller {
				t.Errorf("UseFiller = %v, want %v (metricUnavailableValue = %q)",
					meta.UseFiller, tc.expectedUseFiller, tc.metricUnavailableValue)
			}

			// Check FillValue based on whether it should be set
			if tc.expectedUseFiller {
				if meta.FillValue == nil {
					t.Errorf("FillValue is nil, want %v (metricUnavailableValue = %q)",
						tc.expectedFillValue, tc.metricUnavailableValue)
				} else if *meta.FillValue != tc.expectedFillValue {
					t.Errorf("FillValue = %v, want %v (metricUnavailableValue = %q)",
						*meta.FillValue, tc.expectedFillValue, tc.metricUnavailableValue)
				}
			} else {
				if meta.FillValue != nil {
					t.Errorf("FillValue = %v, want nil (metricUnavailableValue = %q)",
						*meta.FillValue, tc.metricUnavailableValue)
				}
			}
		})
	}
}

func TestDatadogGetQueryResultHandles422NoData(t *testing.T) {
	testCases := []struct {
		name        string
		body        string
		useFiller   bool
		fillValue   float64
		expectValue float64
		expectErr   bool
	}{
		{
			name:      "no data without filler returns error",
			body:      `{"errors":["No data points found within the given time window"]}`,
			expectErr: true,
		},
		{
			name:        "no data with filler",
			body:        `{"errors":["No datapoints found for query"]}`,
			useFiller:   true,
			fillValue:   1.5,
			expectValue: 1.5,
		},
		{
			name:      "no data from errors string without filler returns error",
			body:      `{"errors":"No data points found within the given time window"}`,
			expectErr: true,
		},
		{
			name:      "no data from error field without filler returns error",
			body:      `{"error":"No data points found for query"}`,
			expectErr: true,
		},
		{
			name:      "unprocessable error remains fatal",
			body:      `{"errors":["Invalid query"]}`,
			expectErr: true,
		},
		{
			name:      "mixed errors remain fatal",
			body:      `{"errors":["No data points found within the given time window","Invalid query"]}`,
			expectErr: true,
		},
		{
			name:      "invalid json remains fatal",
			body:      `no data points found within the given time window`,
			expectErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/query" {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnprocessableEntity)
				_ = json.NewEncoder(w).Encode(json.RawMessage(testCase.body))
			}))
			defer server.Close()

			configuration := datadog.NewConfiguration()
			configuration.Servers = datadog.ServerConfigurations{{URL: server.URL}}
			apiClient := datadog.NewAPIClient(configuration)

			meta := &datadogMetadata{
				APIKey:      "apiKey",
				AppKey:      "appKey",
				DatadogSite: "datadoghq.com",
				Query:       "avg:system.cpu.user{*}",
				Age:         90,
			}
			if testCase.useFiller {
				meta.UseFiller = true
				meta.FillValue = &testCase.fillValue
			}

			scaler := &datadogScaler{
				metadata:  meta,
				apiClient: apiClient,
				logger:    logr.Discard(),
			}

			value, err := scaler.getQueryResult(context.Background())
			if testCase.expectErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if value != testCase.expectValue {
				t.Fatalf("expected %v, got %v", testCase.expectValue, value)
			}
		})
	}
}

func TestDatadogGetMetricValueHandles422NoData(t *testing.T) {
	testCases := []struct {
		name        string
		body        string
		useFiller   bool
		fillValue   float64
		expectValue float64
		expectErr   bool
	}{
		{
			name:      "no data without filler returns error",
			body:      `{"errors":["No data points found within the given time window"]}`,
			expectErr: true,
		},
		{
			name:        "no data with filler",
			body:        `{"errors":["No datapoints found for query"]}`,
			useFiller:   true,
			fillValue:   1.5,
			expectValue: 1.5,
		},
		{
			name:      "unprocessable error remains fatal",
			body:      `{"errors":["Invalid query"]}`,
			expectErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnprocessableEntity)
				_ = json.NewEncoder(w).Encode(json.RawMessage(testCase.body))
			}))
			defer server.Close()

			meta := &datadogMetadata{
				DatadogMetricServiceURL: server.URL,
			}
			if testCase.useFiller {
				meta.UseFiller = true
				meta.FillValue = &testCase.fillValue
			}

			scaler := &datadogScaler{
				metadata:             meta,
				httpClient:           server.Client(),
				logger:               logr.Discard(),
				useClusterAgentProxy: true,
			}

			req, _ := http.NewRequest("GET", server.URL, nil)
			value, err := scaler.getDatadogMetricValue(req)

			if testCase.expectErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if value != testCase.expectValue {
				t.Fatalf("expected %v, got %v", testCase.expectValue, value)
			}
		})
	}
}
