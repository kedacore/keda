package scalers

import (
	"net/http"
	"strings"
	"testing"
)

type parsePrometheusMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

type prometheusMetricIdentifier struct {
	metadataTestData *parsePrometheusMetadataTestData
	name             string
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

var prometheusMetricIdentifiers = []prometheusMetricIdentifier{
	{&testPromMetadata[1], "prometheus-http---localhost-9090-http_requests_total"},
}

type prometheusAuthMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

var testPrometheusAuthMetadata = []prometheusAuthMetadataTestData{
	// success TLS
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "disableScaleToZero": "true", "authModes": "tls"}, map[string]string{"ca": "caaa", "cert": "ceert", "key": "keey"}, false},
	// TLS, ca is optional
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "disableScaleToZero": "true", "authModes": "tls"}, map[string]string{"cert": "ceert", "key": "keey"}, false},
	// fail TLS, key not given
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "disableScaleToZero": "true", "authModes": "tls"}, map[string]string{"ca": "caaa", "cert": "ceert"}, true},
	// fail TLS, cert not given
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "disableScaleToZero": "true", "authModes": "tls"}, map[string]string{"ca": "caaa", "key": "keey"}, true},
	// success bearer default
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "disableScaleToZero": "true", "authModes": "bearer"}, map[string]string{"bearerToken": "tooooken"}, false},
	// fail bearerAuth with no token
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "disableScaleToZero": "true", "authModes": "bearer"}, map[string]string{}, true},
	// success basicAuth
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "disableScaleToZero": "true", "authModes": "basic"}, map[string]string{"username": "user", "password": "pass"}, false},
	// fail basicAuth with no username
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "disableScaleToZero": "true", "authModes": "basic"}, map[string]string{}, true},

	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "disableScaleToZero": "true", "authModes": "tls, basic"}, map[string]string{"ca": "caaa", "cert": "ceert", "key": "keey", "username": "user", "password": "pass"}, false},

	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "disableScaleToZero": "true", "authModes": "tls,basic"}, map[string]string{"username": "user", "password": "pass"}, true},
}

func TestPrometheusParseMetadata(t *testing.T) {
	for _, testData := range testPromMetadata {
		_, err := parsePrometheusMetadata(&ScalerConfig{TriggerMetadata: testData.metadata})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestPrometheusGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range prometheusMetricIdentifiers {
		meta, err := parsePrometheusMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockPrometheusScaler := prometheusScaler{
			metadata:   meta,
			httpClient: http.DefaultClient,
		}

		metricSpec := mockPrometheusScaler.GetMetricSpecForScaling()
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestPrometheusScalerAuthParams(t *testing.T) {
	for _, testData := range testPrometheusAuthMetadata {
		meta, err := parsePrometheusMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}

		if err == nil {
			if (meta.enableBearerAuth && !strings.Contains(testData.metadata["authModes"], "bearer")) ||
				(meta.enableBasicAuth && !strings.Contains(testData.metadata["authModes"], "basic")) ||
				(meta.enableTLS && !strings.Contains(testData.metadata["authModes"], "tls")) {
				t.Error("wrong auth mode detected")
			}
		}
	}
}
