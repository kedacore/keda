package scalers

import (
	"strings"
	"testing"
)

type parseGraphiteMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

type graphiteMetricIdentifier struct {
	metadataTestData *parseGraphiteMetadataTestData
	name             string
}

var testGrapMetadata = []parseGraphiteMetadataTestData{
	{map[string]string{}, true},
	// all properly formed
	{map[string]string{"serverAddress": "http://localhost:81", "metricName": "request-count", "threshold": "100", "query": "stats.counters.http.hello-world.request.count.count", "queryTime": "-30Seconds"}, false},
	// missing serverAddress
	{map[string]string{"serverAddress": "", "metricName": "request-count", "threshold": "100", "query": "stats.counters.http.hello-world.request.count.count", "queryTime": "-30Seconds"}, true},
	// missing metricName
	{map[string]string{"serverAddress": "http://localhost:81", "metricName": "", "threshold": "100", "query": "stats.counters.http.hello-world.request.count.count", "queryTime": "-30Seconds"}, true},
	// malformed threshold
	{map[string]string{"serverAddress": "http://localhost:81", "metricName": "request-count", "threshold": "one", "query": "stats.counters.http.hello-world.request.count.count", "queryTime": "-30Seconds"}, true},
	// missing query
	{map[string]string{"serverAddress": "http://localhost:81", "metricName": "request-count", "threshold": "100", "query": "", "queryTime": "-30Seconds", "disableScaleToZero": "true"}, true},
	// missing queryTime
	{map[string]string{"serverAddress": "http://localhost:81", "metricName": "request-count", "threshold": "100", "query": "stats.counters.http.hello-world.request.count.count", "queryTime": ""}, true},
}

var graphiteMetricIdentifiers = []graphiteMetricIdentifier{
	{&testGrapMetadata[1], "graphite-request-count"},
}

type graphiteAuthMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

var testGraphiteAuthMetadata = []graphiteAuthMetadataTestData{
	// success basicAuth
	{map[string]string{"serverAddress": "http://localhost:81", "metricName": "request-count", "threshold": "100", "query": "stats.counters.http.hello-world.request.count.count", "queryTime": "-30Seconds", "authMode": "basic"}, map[string]string{"username": "user", "password": "pass"}, false},
	// fail basicAuth with no username
	{map[string]string{"serverAddress": "http://localhost:81", "metricName": "request-count", "threshold": "100", "query": "stats.counters.http.hello-world.request.count.count", "queryTime": "-30Seconds", "authMode": "basic"}, map[string]string{}, true},
	// fail if using non-basicAuth authMode
	{map[string]string{"serverAddress": "http://localhost:81", "metricName": "request-count", "threshold": "100", "query": "stats.counters.http.hello-world.request.count.count", "queryTime": "-30Seconds", "authMode": "tls"}, map[string]string{"username": "user"}, true},
}

func TestGraphiteParseMetadata(t *testing.T) {
	for _, testData := range testGrapMetadata {
		_, err := parseGraphiteMetadata(&ScalerConfig{TriggerMetadata: testData.metadata})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestGraphiteGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range graphiteMetricIdentifiers {
		meta, err := parseGraphiteMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockGraphiteScaler := graphiteScaler{
			metadata: meta,
		}

		metricSpec := mockGraphiteScaler.GetMetricSpecForScaling()
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestGraphiteScalerAuthParams(t *testing.T) {
	for _, testData := range testGraphiteAuthMetadata {
		meta, err := parseGraphiteMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}

		if err == nil {
			if meta.enableBasicAuth && !strings.Contains(testData.metadata["authMode"], "basic") {
				t.Error("wrong auth mode detected")
			}
		}
	}
}
