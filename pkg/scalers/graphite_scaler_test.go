package scalers

import (
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
	{map[string]string{"serverAddress": "http://localhost:81", "metricName": "request-count", "threshold": "100", "query": "stats.counters.http.hello-world.request.count.count", "queryTime": "-30Seconds", "disableScaleToZero": "true"}, false},
	// missing serverAddress
	{map[string]string{"serverAddress": "", "grapmetricName": "request-count", "threshold": "100", "query": "stats.counters.http.hello-world.request.count.count", "queryTime": "-30Seconds", "disableScaleToZero": "true"}, true},
	// missing metricName
	{map[string]string{"serverAddress": "http://localhost:81", "metricName": "", "threshold": "100", "query": "stats.counters.http.hello-world.request.count.count", "queryTime": "-30Seconds", "disableScaleToZero": "true"}, true},
	// malformed threshold
	{map[string]string{"serverAddress": "http://localhost:81", "metricName": "request-count", "threshold": "one", "query": "stats.counters.http.hello-world.request.count.count", "queryTime": "-30Seconds", "disableScaleToZero": "true"}, true},
	// missing query
	{map[string]string{"serverAddress": "http://localhost:81", "metricName": "request-count", "threshold": "100", "query": "", "queryTime": "-30Seconds", "disableScaleToZero": "true"}, true},
	// missing queryTime
	{map[string]string{"serverAddress": "http://localhost:81", "metricName": "request-count", "threshold": "100", "query": "stats.counters.http.hello-world.request.count.count", "queryTime": "", "disableScaleToZero": "true"}, true},
	// all properly formed, default disableScaleToZero
	{map[string]string{"serverAddress": "http://localhost:81", "metricName": "request-count", "threshold": "100", "queryTime": "-30Seconds", "query": "stats.counters.http.hello-world.request.count.count"}, false},
}

var graphiteMetricIdentifiers = []graphiteMetricIdentifier{
	{&testGrapMetadata[1], "graphite-http---localhost-81-request-count"},
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
