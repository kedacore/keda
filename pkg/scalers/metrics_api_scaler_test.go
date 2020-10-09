package scalers

import (
	"testing"
)

type metricsAPIMetadataTestData struct {
	metadata    map[string]string
	raisesError bool
}

var validMetricAPIMetadata = map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric", "targetValue": "42"}

var testMetricsAPIMetadata = []metricsAPIMetadataTestData{
	// No metadata
	{metadata: map[string]string{}, raisesError: true},
	// OK
	{metadata: map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric", "targetValue": "42"}, raisesError: false},
	// Target not an int
	{metadata: map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric", "targetValue": "aa"}, raisesError: true},
	// Missing metric name
	{metadata: map[string]string{"url": "http://dummy:1230/api/v1/", "targetValue": "aa"}, raisesError: true},
	// Missing url
	{metadata: map[string]string{"valueLocation": "metric", "targetValue": "aa"}, raisesError: true},
	// Missing targetValue
	{metadata: map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric"}, raisesError: true},
}

type metricAPIAuthMetadataTestData struct {
	authParams map[string]string
	isError    bool
}

var testMetricsAPIAuthMetadata = []metricAPIAuthMetadataTestData{
	// success TLS
	{map[string]string{"authMode": "tlsAuth", "ca": "caaa", "cert": "ceert", "key": "keey"}, false},
	// fail TLS, ca not given
	{map[string]string{"authMode": "tlsAuth", "cert": "ceert", "key": "keey"}, true},
	// fail TLS, key not given
	{map[string]string{"authMode": "tlsAuth", "ca": "caaa", "cert": "ceert"}, true},
	// fail TLS, cert not given
	{map[string]string{"authMode": "tlsAuth", "ca": "caaa", "key": "keey"}, true},
	// success apiKeyAuth default
	{map[string]string{"authMode": "apiKeyAuth", "apiKey": "apiikey"}, false},
	// success apiKeyAuth as query param
	{map[string]string{"authMode": "apiKeyAuth", "apiKey": "apiikey", "method": "query"}, false},
	// success apiKeyAuth with headers and custom key name
	{map[string]string{"authMode": "apiKeyAuth", "apiKey": "apiikey", "method": "header", "keyParamName": "custom"}, false},
	// success apiKeyAuth with query param and custom key name
	{map[string]string{"authMode": "apiKeyAuth", "apiKey": "apiikey", "method": "query", "keyParamName": "custom"}, false},
	// fail apiKeyAuth with no api key
	{map[string]string{"authMode": "apiKeyAuth"}, true},
	// success basicAuth
	{map[string]string{"authMode": "basicAuth", "username": "user", "password": "pass"}, false},
	// fail basicAuth with no username
	{map[string]string{"authMode": "basicAuth"}, true},
}

func TestParseMetricsAPIMetadata(t *testing.T) {
	for _, testData := range testMetricsAPIMetadata {
		_, err := metricsAPIMetadata(testData.metadata, map[string]string{})
		if err != nil && !testData.raisesError {
			t.Error("Expected success but got error", err)
		}
		if err == nil && testData.raisesError {
			t.Error("Expected error but got success")
		}
	}
}

func TestGetValueFromResponse(t *testing.T) {
	d := []byte(`{"components":[{"id": "82328e93e", "tasks": 32}],"count":2.43}`)
	v, err := GetValueFromResponse(d, "components.0.tasks")
	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if v != 32 {
		t.Errorf("Expected %d got %d", 32, v)
	}

	v, err = GetValueFromResponse(d, "count")
	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if v != 2 {
		t.Errorf("Expected %d got %d", 2, v)
	}
}

func TestMetricAPIScalerAuthParams(t *testing.T) {
	for _, testData := range testMetricsAPIAuthMetadata {
		meta, err := metricsAPIMetadata(validMetricAPIMetadata, testData.authParams)

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}

		if err == nil {
			if (meta.enableAPIKeyAuth && !(testData.authParams["authMode"] == "apiKeyAuth")) ||
				(meta.enableBaseAuth && !(testData.authParams["authMode"] == "basicAuth")) ||
				(meta.enableTLS && !(testData.authParams["authMode"] == "tlsAuth")) {
				t.Error("wrong auth mode detected")
			}
		}
	}
}
