package scalers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type metricsAPIMetadataTestData struct {
	metadata    map[string]string
	raisesError bool
}

var testMetricsAPIMetadata = []metricsAPIMetadataTestData{
	// No metadata
	{metadata: map[string]string{}, raisesError: true},
	// OK
	{metadata: map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric.test", "targetValue": "42"}, raisesError: false},
	// Target not an int
	{metadata: map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric", "targetValue": "aa"}, raisesError: true},
	// Activation target not an int
	{metadata: map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric", "targetValue": "1", "activationTargetValue": "aa"}, raisesError: true},
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
	// success bearerAuth default
	{map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric", "targetValue": "42", "authMode": "bearer"}, map[string]string{"token": "bearerTokenValue"}, false},
	// fail bearerAuth without token
	{map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric", "targetValue": "42", "authMode": "bearer"}, map[string]string{}, true},
	// success unsafeSsl true
	{map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric", "targetValue": "42", "unsafeSsl": "true"}, map[string]string{}, false},
	// success unsafeSsl false
	{map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric", "targetValue": "42", "unsafeSsl": "false"}, map[string]string{}, false},
	// failed unsafeSsl non bool
	{map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric", "targetValue": "42", "unsafeSsl": "yes"}, map[string]string{}, true},
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

type metricsAPIMetricIdentifier struct {
	metadataTestData *metricsAPIMetadataTestData
	scalerIndex      int
	name             string
}

var metricsAPIMetricIdentifiers = []metricsAPIMetricIdentifier{
	{metadataTestData: &testMetricsAPIMetadata[1], scalerIndex: 1, name: "s1-metric-api-metric-test"},
}

func TestMetricsAPIGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range metricsAPIMetricIdentifiers {
		s, err := NewMetricsAPIScaler(
			&ScalerConfig{
				ResolvedEnv:       map[string]string{},
				TriggerMetadata:   testData.metadataTestData.metadata,
				AuthParams:        map[string]string{},
				GlobalHTTPTimeout: 3000 * time.Millisecond,
				ScalerIndex:       testData.scalerIndex,
			},
		)
		if err != nil {
			t.Errorf("Error creating the Scaler")
		}

		metricSpec := s.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestGetValueFromResponse(t *testing.T) {
	d := []byte(`{"components":[{"id": "82328e93e", "tasks": 32, "str": "64", "k":"1k","wrong":"NaN"}],"count":2.43}`)
	v, err := GetValueFromResponse(d, "components.0.tasks")
	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if v != 32 {
		t.Errorf("Expected %d got %f", 32, v)
	}

	v, err = GetValueFromResponse(d, "count")
	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if v != 2.43 {
		t.Errorf("Expected %d got %f", 2, v)
	}

	v, err = GetValueFromResponse(d, "components.0.str")
	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if v != 64 {
		t.Errorf("Expected %d got %f", 64, v)
	}

	v, err = GetValueFromResponse(d, "components.0.k")
	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if v != 1000 {
		t.Errorf("Expected %d got %f", 1000, v)
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
				(meta.enableTLS && !(testData.metadata["authMode"] == "tls")) ||
				(meta.enableBearerAuth && !(testData.metadata["authMode"] == "bearer")) {
				t.Error("wrong auth mode detected")
			}
		}
	}
}

func TestBearerAuth(t *testing.T) {
	authentication := map[string]string{
		"token": "secure-token",
	}

	var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if val, ok := r.Header["Authorization"]; ok {
			if val[0] != fmt.Sprintf("Bearer %s", authentication["token"]) {
				t.Errorf("Authorization header malformed")
			}
		} else {
			t.Errorf("Authorization header not found")
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"components":[{"id": "82328e93e", "tasks": 32, "str": "64", "k":"1k","wrong":"NaN"}],"count":2.43}`))
	}))

	metadata := map[string]string{
		"url":           apiStub.URL,
		"valueLocation": "components.0.tasks",
		"targetValue":   "1",
		"authMode":      "bearer",
	}

	s, err := NewMetricsAPIScaler(
		&ScalerConfig{
			ResolvedEnv:       map[string]string{},
			TriggerMetadata:   metadata,
			AuthParams:        authentication,
			GlobalHTTPTimeout: 3000 * time.Millisecond,
		},
	)
	if err != nil {
		t.Errorf("Error creating the Scaler")
	}

	_, _, err = s.GetMetricsAndActivity(context.TODO(), "test-metric")
	if err != nil {
		t.Errorf("Error getting the metric")
	}
}

type MockHTTPRoundTripper struct {
	mock.Mock
}

func (m *MockHTTPRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	args := m.Called(request)
	resp := args.Get(0).(*http.Response)
	resp.Request = request
	return resp, args.Error(1)
}

func TestGetMetricValueErrorMessage(t *testing.T) {
	// mock roundtripper to return non-ok status code
	mockHTTPRoundTripper := MockHTTPRoundTripper{}
	mockHTTPRoundTripper.On("RoundTrip", mock.Anything).Return(&http.Response{StatusCode: http.StatusTeapot}, nil)

	httpClient := http.Client{Transport: &mockHTTPRoundTripper}
	s := metricsAPIScaler{
		metadata: &metricsAPIScalerMetadata{url: "http://dummy:1230/api/v1/"},
		client:   &httpClient,
	}

	_, err := s.getMetricValue(context.TODO())

	assert.Equal(t, err.Error(), "/api/v1/: api returned 418")
}
