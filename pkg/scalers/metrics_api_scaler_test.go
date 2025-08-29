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

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
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
	// success with both apiKey and TLS authentication
	{map[string]string{"url": "http://dummy:1230/api/v1/", "valueLocation": "metric", "targetValue": "42", "authMode": "apiKey,tls"}, map[string]string{"apiKey": "apiikey", "ca": "caaa", "cert": "ceert", "key": "keey"}, false},
}

func TestParseMetricsAPIMetadata(t *testing.T) {
	for _, testData := range testMetricsAPIMetadata {
		_, err := parseMetricsAPIMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: map[string]string{}})
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
	triggerIndex     int
	name             string
}

var metricsAPIMetricIdentifiers = []metricsAPIMetricIdentifier{
	{metadataTestData: &testMetricsAPIMetadata[1], triggerIndex: 1, name: "s1-metric-api-metric-test"},
}

func TestMetricsAPIGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range metricsAPIMetricIdentifiers {
		s, err := NewMetricsAPIScaler(
			&scalersconfig.ScalerConfig{
				ResolvedEnv:       map[string]string{},
				TriggerMetadata:   testData.metadataTestData.metadata,
				AuthParams:        map[string]string{},
				GlobalHTTPTimeout: 3000 * time.Millisecond,
				TriggerIndex:      testData.triggerIndex,
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
	inputJSON := []byte(`{"components":[{"id": "82328e93e", "tasks": 32, "str": "64", "k":"1k","wrong":"NaN"}],"count":2.43}`)
	inputYAML := []byte(`{components: [{id: 82328e93e, tasks: 32, str: '64', k: 1k, wrong: NaN}], count: 2.43}`)
	inputPrometheus := []byte(`# HELP backend_queue_size Total number of items
	# TYPE backend_queue_size counter
	backend_queue_size{queueName="zero"} 0
	backend_queue_size{queueName="one"} 1
	backend_queue_size{queueName="two", instance="random"} 2
	backend_queue_size{queueName="two", instance="zero"} 20
	# HELP random_metric Random metric generate to include noise
	# TYPE random_metric counter
	random_metric 10`)
	// Ending the file without new line is intended to verify
	// https://github.com/kedacore/keda/issues/6559

	testCases := []struct {
		name      string
		input     []byte
		key       string
		format    APIFormat
		expectVal float64
		expectErr bool
	}{
		{name: "integer", input: inputJSON, key: "count", format: JSONFormat, expectVal: 2.43},
		{name: "string", input: inputJSON, key: "components.0.str", format: JSONFormat, expectVal: 64},
		{name: "{}.[].{}", input: inputJSON, key: "components.0.tasks", format: JSONFormat, expectVal: 32},
		{name: "invalid data", input: inputJSON, key: "components.0.wrong", format: JSONFormat, expectErr: true},

		{name: "integer", input: inputYAML, key: "count", format: YAMLFormat, expectVal: 2.43},
		{name: "string", input: inputYAML, key: "components.0.str", format: YAMLFormat, expectVal: 64},
		{name: "{}.[].{}", input: inputYAML, key: "components.0.tasks", format: YAMLFormat, expectVal: 32},
		{name: "invalid data", input: inputYAML, key: "components.0.wrong", format: YAMLFormat, expectErr: true},

		{name: "no labels", input: inputPrometheus, key: "random_metric", format: PrometheusFormat, expectVal: 10},
		{name: "one label", input: inputPrometheus, key: "backend_queue_size{queueName=\"one\"}", format: PrometheusFormat, expectVal: 1},
		{name: "multiple labels not queried", input: inputPrometheus, key: "backend_queue_size{queueName=\"two\"}", format: PrometheusFormat, expectVal: 2},
		{name: "multiple labels queried", input: inputPrometheus, key: "backend_queue_size{queueName=\"two\", instance=\"zero\"}", format: PrometheusFormat, expectVal: 20},
		{name: "invalid data", input: inputPrometheus, key: "backend_queue_size{invalid=test}", format: PrometheusFormat, expectErr: true},
	}

	for _, tc := range testCases {
		t.Run(string(tc.format)+": "+tc.name, func(t *testing.T) {
			v, err := GetValueFromResponse(tc.input, tc.key, tc.format)

			if tc.expectErr {
				assert.Error(t, err)
			}

			assert.EqualValues(t, tc.expectVal, v)
		})
	}
}

func TestMetricAPIScalerAuthParams(t *testing.T) {
	for _, testData := range testMetricsAPIAuthMetadata {
		_, err := parseMetricsAPIMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
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
		&scalersconfig.ScalerConfig{
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
		metadata:   &metricsAPIScalerMetadata{URL: "http://dummy:1230/api/v1/"},
		httpClient: &httpClient,
	}

	_, err := s.getMetricValue(context.TODO())

	assert.Equal(t, err.Error(), "/api/v1/: api returned 418")
}
