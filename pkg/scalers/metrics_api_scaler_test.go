package scalers

import (
	"testing"
)

type metricsAPIMetadataTestData struct {
	metadata    map[string]string
	raisesError bool
}

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
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

var testMetricsAPIAuthMetadata = []metricAPIAuthMetadataTestData{
	// success TLS
	{map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric", "targetValue": "42", "authMode": "tls"}, map[string]string{"ca": "caaa", "cert": "ceert", "key": "keey"}, false},
	// fail TLS, ca not given
	{map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric", "targetValue": "42", "authMode": "tls"}, map[string]string{"cert": "ceert", "key": "keey"}, true},
	// fail TLS, key not given
	{map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric", "targetValue": "42", "authMode": "tls"}, map[string]string{"ca": "caaa", "cert": "ceert"}, true},
	// fail TLS, cert not given
	{map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric", "targetValue": "42", "authMode": "tls"}, map[string]string{"ca": "caaa", "key": "keey"}, true},
	// success apiKeyAuth default
	{map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric", "targetValue": "42", "authMode": "apiKey"}, map[string]string{"apiKey": "apiikey"}, false},
	// success apiKeyAuth as query param
	{map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric", "targetValue": "42", "authMode": "apiKey"}, map[string]string{"apiKey": "apiikey", "method": "query"}, false},
	// success apiKeyAuth with headers and custom key name
	{map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric", "targetValue": "42", "authMode": "apiKey"}, map[string]string{"apiKey": "apiikey", "method": "header", "keyParamName": "custom"}, false},
	// success apiKeyAuth with query param and custom key name
	{map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric", "targetValue": "42", "authMode": "apiKey"}, map[string]string{"apiKey": "apiikey", "method": "query", "keyParamName": "custom"}, false},
	// fail apiKeyAuth with no api key
	{map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric", "targetValue": "42", "authMode": "apiKey"}, map[string]string{}, true},
	// success basicAuth
	{map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric", "targetValue": "42", "authMode": "basic"}, map[string]string{"username": "user", "password": "pass"}, false},
	// fail basicAuth with no username
	{map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric", "targetValue": "42", "authMode": "basic"}, map[string]string{}, true},
}

func TestParseMetricsAPIMetadata(t *testing.T) {
	for _, testData := range testMetricsAPIMetadata {
		_, err := parseMetricsAPIMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: map[string]string{}})
		if err != nil && !testData.raisesError {
			t.Error("Expected success but got error", err)
		}
		if err == nil && testData.raisesError {
			t.Error("Expected error but got success")
		}
	}
}

func TestGetValueFromResponse(t *testing.T) {
	d := []byte(`{"components":[{"id": "82328e93e", "tasks": 32, "str": "64", "k":"1k","wrong":"NaN"}],"count":2.43}`)
	v, err := GetValueFromResponse(d, "components.0.tasks")
	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if v.CmpInt64(32) != 0 {
		t.Errorf("Expected %d got %d", 32, v.AsDec())
	}

	v, err = GetValueFromResponse(d, "count")
	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if v.CmpInt64(2) != 0 {
		t.Errorf("Expected %d got %d", 2, v.AsDec())
	}

	v, err = GetValueFromResponse(d, "components.0.str")
	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if v.CmpInt64(64) != 0 {
		t.Errorf("Expected %d got %d", 64, v.AsDec())
	}

	v, err = GetValueFromResponse(d, "components.0.k")
	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if v.CmpInt64(1000) != 0 {
		t.Errorf("Expected %d got %d", 1000, v.AsDec())
	}

	_, err = GetValueFromResponse(d, "components.0.wrong")
	if err == nil {
		t.Error("Expected error but got success", err)
	}
}

func TestMetricAPIScalerAuthParams(t *testing.T) {
	for _, testData := range testMetricsAPIAuthMetadata {
		meta, err := parseMetricsAPIMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}

		if err == nil {
			if (meta.enableAPIKeyAuth && !(testData.metadata["authMode"] == "apiKey")) ||
				(meta.enableBaseAuth && !(testData.metadata["authMode"] == "basic")) ||
				(meta.enableTLS && !(testData.metadata["authMode"] == "tls")) {
				t.Error("wrong auth mode detected")
			}
		}
	}
}
