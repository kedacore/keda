package scalers

import (
	"testing"
)

type parsePrometheusMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

var testPromMetadata = []parsePrometheusMetadataTestData{
	{map[string]string{}, true},
	// all properly formed
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "disableScaleToZero": "true"}, false},
	// missing serverAddress
	{map[string]string{"serverAddress": "", "metricName": "http_requests_total", "threshold": "100", "query": "up", "disableScaleToZero": "true"}, true},
	// missing metricName
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "", "threshold": "100", "query": "up", "disableScaleToZero": "true"}, true},
	// malformed threshold
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "one", "query": "up", "disableScaleToZero": "true"}, true},
	// missing query
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "", "disableScaleToZero": "true"}, true},
	// all properly formed, default disableScaleToZero
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up"}, false},
}

func TestPrometheusParseMetadata(t *testing.T) {
	for _, testData := range testPromMetadata {
		_, err := parsePrometheusMetadata(testData.metadata, map[string]string{})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}
