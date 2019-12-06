package scalers

import (
	"testing"
)

const (
	testServerAddress = "myAddress"
)

var testPrometheusResolvedEnv = map[string]string{
	testServerAddress: "none",
}

type parsePrometheusMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

var testPromMetadata = []parsePrometheusMetadataTestData{
	{map[string]string{}, map[string]string{}, true},
	// all properly formed
	{map[string]string{"serverAddress": testServerAddress, "metricName": "http_requests_total", "threshold": "100", "query": "up", "disableScaleToZero": "true"}, map[string]string{}, false},
	// missing serverAddress
	{map[string]string{"serverAddress": "", "metricName": "http_requests_total", "threshold": "100", "query": "up", "disableScaleToZero": "true"}, map[string]string{}, true},
	// missing metricName
	{map[string]string{"serverAddress": testServerAddress, "metricName": "", "threshold": "100", "query": "up", "disableScaleToZero": "true"}, map[string]string{}, true},
	// malformed threshold
	{map[string]string{"serverAddress": testServerAddress, "metricName": "http_requests_total", "threshold": "one", "query": "up", "disableScaleToZero": "true"}, map[string]string{}, true},
	// missing query
	{map[string]string{"serverAddress": testServerAddress, "metricName": "http_requests_total", "threshold": "100", "query": "", "disableScaleToZero": "true"}, map[string]string{}, true},
	// all properly formed, default disableScaleToZero
	{map[string]string{"serverAddress": testServerAddress, "metricName": "http_requests_total", "threshold": "100", "query": "up"}, map[string]string{}, false},
	// metadata not sullpy serverAddress but authParams supply.
	{map[string]string{"serverAddress": "", "metricName": "http_requests_total", "threshold": "100", "query": "up", "disableScaleToZero": "true"}, map[string]string{
		"serverAddress": "none",
	}, false},
}

func TestPrometheusParseMetadata(t *testing.T) {
	for _, testData := range testPromMetadata {
		_, err := parsePrometheusMetadata(testPrometheusResolvedEnv, testData.metadata, testData.authParams)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}
