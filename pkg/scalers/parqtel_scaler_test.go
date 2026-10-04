package scalers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseParqtelMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

var testParqtelMetadata = []parseParqtelMetadataTestData{
	// missing required fields
	{map[string]string{}, true},
	// all properly formed
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100"}, false},
	// with activationThreshold
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "activationThreshold": "50"}, false},
	// with ignoreNullValues
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "ignoreNullValues": "false"}, false},
	// missing serverAddress
	{map[string]string{"serverAddress": "", "query": "up", "threshold": "100"}, true},
	// missing threshold
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up"}, true},
	// malformed threshold
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "one"}, true},
	// malformed activationThreshold
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "activationThreshold": "one"}, true},
	// missing query
	{map[string]string{"serverAddress": "http://localhost:9090", "threshold": "100", "query": ""}, true},
	// ignoreNullValues with wrong value
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "ignoreNullValues": "xxxx"}, true},
	// unsafeSsl
	{map[string]string{"serverAddress": "https://localhost:9090", "query": "up", "threshold": "100", "unsafeSsl": "true"}, false},
	// customHeaders
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "customHeaders": "key1=value1,key2=value2"}, false},
	// customHeaders with wrong format
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "customHeaders": "key1=value1,key2"}, true},
	// queryParameters
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "queryParameters": "key1=value1,key2=value2"}, false},
	// queryParameters with wrong format
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "queryParameters": "key1=value1,key2"}, true},
	// valid custom http client timeout
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "timeout": "1000"}, false},
	// invalid - negative - custom http client timeout
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "timeout": "-1"}, true},
	// invalid - not a number - custom http client timeout
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "timeout": "a"}, true},
	// valid range query type
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "queryType": "range"}, false},
	// valid range query with step
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "queryType": "range", "rangeStep": "30s"}, false},
	// invalid range step
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "queryType": "range", "rangeStep": "abc"}, true},
	// unsupported queryType
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "queryType": "bogus"}, true},
	// unsupported resultAggregation
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "resultAggregation": "bogus"}, true},
	// valid resultAggregation
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "resultAggregation": "sum"}, false},
	// valid signal: logs
	{map[string]string{"serverAddress": "http://localhost:9090", "query": `{service="api"}`, "threshold": "100", "signal": "logs"}, false},
	// valid signal: traces
	{map[string]string{"serverAddress": "http://localhost:9090", "query": `service.name="api"`, "threshold": "100", "signal": "traces"}, false},
	// invalid signal
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "signal": "bogus"}, true},
}

func TestParqtelParseMetadata(t *testing.T) {
	for _, testData := range testParqtelMetadata {
		_, err := parseParqtelMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestParqtelGetMetricSpecForScaling(t *testing.T) {
	for _, triggerIndex := range []int{0, 1} {
		meta, err := parseParqtelMetadata(&scalersconfig.ScalerConfig{
			TriggerMetadata: map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100"},
			TriggerIndex:    triggerIndex,
		})
		require.NoError(t, err)

		scaler := parqtelScaler{metadata: meta, httpClient: http.DefaultClient}
		metricSpec := scaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name

		expected := "s" + string(rune('0'+triggerIndex)) + "-parqtel"
		assert.Equal(t, expected, metricName)
	}
}

type parqtelAuthMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

var testParqtelAuthMetadata = []parqtelAuthMetadataTestData{
	// success TLS
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "authModes": "tls"}, map[string]string{"ca": "caaa", "cert": "ceert", "key": "keey"}, false},
	// TLS, ca is optional
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "authModes": "tls"}, map[string]string{"cert": "ceert", "key": "keey"}, false},
	// fail TLS, key not given
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "authModes": "tls"}, map[string]string{"ca": "caaa", "cert": "ceert"}, true},
	// fail TLS, cert not given
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "authModes": "tls"}, map[string]string{"ca": "caaa", "key": "keey"}, true},
	// success bearer
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "authModes": "bearer"}, map[string]string{"bearerToken": "tooooken"}, false},
	// fail bearer with no token
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "authModes": "bearer"}, map[string]string{}, true},
	// success basic
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "authModes": "basic"}, map[string]string{"username": "user", "password": "pass"}, false},
	// fail basic with no username
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "authModes": "basic"}, map[string]string{}, true},
	// success custom auth
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "authModes": "custom"}, map[string]string{"customAuthHeader": "header", "customAuthValue": "value"}, false},
	// fail custom auth with no customAuthHeader
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "authModes": "custom"}, map[string]string{"customAuthValue": "value"}, true},
	// unsupported auth mode (oauth)
	{map[string]string{"serverAddress": "http://localhost:9090", "query": "up", "threshold": "100", "authModes": "oauth"}, map[string]string{"oauthTokenURI": "http://localhost:8080/token", "clientID": "client", "clientSecret": "secret"}, true},
}

func TestParqtelScalerAuthParams(t *testing.T) {
	for _, testData := range testParqtelAuthMetadata {
		_, err := parseParqtelMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

type parqtelQueryResultTestData struct {
	name              string
	bodyStr           string
	responseStatus    int
	expectedValue     float64
	isError           bool
	ignoreNullValues  bool
	resultAggregation string
}

var testParqtelQueryResult = []parqtelQueryResultTestData{
	{
		name:             "no results",
		bodyStr:          `{}`,
		responseStatus:   http.StatusOK,
		expectedValue:    0,
		isError:          false,
		ignoreNullValues: true,
	},
	{
		name:             "no values",
		bodyStr:          `{"data":{"result":[]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    0,
		isError:          false,
		ignoreNullValues: true,
	},
	{
		name:             "no values but shouldn't ignore",
		bodyStr:          `{"data":{"result":[]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    -1,
		isError:          true,
		ignoreNullValues: false,
	},
	{
		name:             "value is empty list",
		bodyStr:          `{"data":{"result":[{"value": []}]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    0,
		isError:          false,
		ignoreNullValues: true,
	},
	{
		name:             "value is empty list but shouldn't ignore",
		bodyStr:          `{"data":{"result":[{"value": []}]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    -1,
		isError:          true,
		ignoreNullValues: false,
	},
	{
		name:             "valid instant value",
		bodyStr:          `{"data":{"resultType":"vector","result":[{"value": [1686063687, "2"]}]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    2,
		isError:          false,
		ignoreNullValues: true,
	},
	{
		name:             "not enough values",
		bodyStr:          `{"data":{"result":[{"value": ["1"]}]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    0,
		isError:          false,
		ignoreNullValues: true,
	},
	{
		name:              "multiple results first",
		bodyStr:           `{"data":{"result":[{"value": [1, "10"]},{"value": [1, "20"]}]}}`,
		responseStatus:    http.StatusOK,
		expectedValue:     10,
		isError:           false,
		ignoreNullValues:  true,
		resultAggregation: "first",
	},
	{
		name:              "multiple results sum",
		bodyStr:           `{"data":{"result":[{"value": [1, "10"]},{"value": [1, "20"]}]}}`,
		responseStatus:    http.StatusOK,
		expectedValue:     30,
		isError:           false,
		ignoreNullValues:  true,
		resultAggregation: "sum",
	},
	{
		name:              "multiple results max",
		bodyStr:           `{"data":{"result":[{"value": [1, "10"]},{"value": [1, "20"]}]}}`,
		responseStatus:    http.StatusOK,
		expectedValue:     20,
		isError:           false,
		ignoreNullValues:  true,
		resultAggregation: "max",
	},
	{
		name:              "multiple results min",
		bodyStr:           `{"data":{"result":[{"value": [1, "10"]},{"value": [1, "20"]}]}}`,
		responseStatus:    http.StatusOK,
		expectedValue:     10,
		isError:           false,
		ignoreNullValues:  true,
		resultAggregation: "min",
	},
	{
		name:              "multiple results avg",
		bodyStr:           `{"data":{"result":[{"value": [1, "10"]},{"value": [1, "20"]}]}}`,
		responseStatus:    http.StatusOK,
		expectedValue:     15,
		isError:           false,
		ignoreNullValues:  true,
		resultAggregation: "avg",
	},
	{
		name:              "multiple results count",
		bodyStr:           `{"data":{"result":[{"value": [1, "10"]},{"value": [1, "20"]}]}}`,
		responseStatus:    http.StatusOK,
		expectedValue:     2,
		isError:           false,
		ignoreNullValues:  true,
		resultAggregation: "count",
	},
	{
		name:              "multiple results last",
		bodyStr:           `{"data":{"result":[{"value": [1, "10"]},{"value": [1, "20"]}]}}`,
		responseStatus:    http.StatusOK,
		expectedValue:     20,
		isError:           false,
		ignoreNullValues:  true,
		resultAggregation: "last",
	},
	{
		name:             "range matrix uses last sample",
		bodyStr:          `{"data":{"resultType":"matrix","result":[{"values": [[1, "5"], [2, "7"], [3, "9"]]}]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    9,
		isError:          false,
		ignoreNullValues: true,
	},
	{
		name:             "error status response",
		bodyStr:          `{}`,
		responseStatus:   http.StatusBadRequest,
		expectedValue:    -1,
		isError:          true,
		ignoreNullValues: true,
	},
	{
		name:             "query returned error status",
		bodyStr:          `{"status":"error","error":"boom"}`,
		responseStatus:   http.StatusOK,
		expectedValue:    -1,
		isError:          true,
		ignoreNullValues: true,
	},
	{
		name:             "+Inf",
		bodyStr:          `{"data":{"result":[{"value": ["1", "+Inf"]}]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    0,
		isError:          false,
		ignoreNullValues: true,
	},
	{
		name:             "+Inf but shouldn't ignore",
		bodyStr:          `{"data":{"result":[{"value": ["1", "+Inf"]}]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    -1,
		isError:          true,
		ignoreNullValues: false,
	},
	{
		name:             "NaN",
		bodyStr:          `{"data":{"result":[{"value": ["1", "NaN"]}]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    0,
		isError:          false,
		ignoreNullValues: true,
	},
	{
		name:             "NaN but shouldn't ignore",
		bodyStr:          `{"data":{"result":[{"value": ["1", "NaN"]}]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    -1,
		isError:          true,
		ignoreNullValues: false,
	},
}

func TestParqtelScalerExecuteQuery(t *testing.T) {
	for _, testData := range testParqtelQueryResult {
		t.Run(testData.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(testData.responseStatus)
				if _, err := writer.Write([]byte(testData.bodyStr)); err != nil { // nosemgrep: no-direct-write-to-responsewriter
					t.Fatal(err)
				}
			}))
			defer server.Close()

			scaler := parqtelScaler{
				metadata: &parqtelMetadata{
					ServerAddress:     server.URL,
					IgnoreNullValues:  testData.ignoreNullValues,
					ResultAggregation: testData.resultAggregation,
					ParqtelAuth:       &authentication.Config{},
				},
				httpClient: http.DefaultClient,
				logger:     logr.Discard(),
			}

			value, err := scaler.ExecuteQuery(context.Background())

			assert.Equal(t, testData.expectedValue, value)
			if testData.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParqtelScalerCustomHeaders(t *testing.T) {
	customHeadersValue := map[string]string{
		"X-Client-Id":          "cid",
		"X-Tenant-Id":          "tid",
		"X-Organization-Token": "oid",
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		for headerName, headerValue := range customHeadersValue {
			assert.Equal(t, headerValue, request.Header.Get(headerName))
		}
		writer.WriteHeader(http.StatusOK)
		if _, err := writer.Write([]byte(`{"data":{"result":[]}}`)); err != nil { // nosemgrep: no-direct-write-to-responsewriter
			t.Fatal(err)
		}
	}))
	defer server.Close()

	scaler := parqtelScaler{
		metadata: &parqtelMetadata{
			ServerAddress:    server.URL,
			CustomHeaders:    customHeadersValue,
			IgnoreNullValues: true,
			ParqtelAuth:      &authentication.Config{},
		},
		httpClient: http.DefaultClient,
	}

	_, err := scaler.ExecuteQuery(context.Background())
	assert.NoError(t, err)
}

func TestParqtelScalerQueryParameters(t *testing.T) {
	queryParametersValue := map[string]string{
		"first":  "foo",
		"second": "bar",
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		assert.Equal(t, "foo", query.Get("first"))
		assert.Equal(t, "bar", query.Get("second"))
		assert.Equal(t, "/api/v1/query", request.URL.Path)
		assert.NotEmpty(t, query.Get("query"))

		writer.WriteHeader(http.StatusOK)
		if _, err := writer.Write([]byte(`{"data":{"result":[]}}`)); err != nil { // nosemgrep: no-direct-write-to-responsewriter
			t.Fatal(err)
		}
	}))
	defer server.Close()

	scaler := parqtelScaler{
		metadata: &parqtelMetadata{
			ServerAddress:    server.URL,
			Query:            "up",
			QueryParameters:  queryParametersValue,
			IgnoreNullValues: true,
			ParqtelAuth:      &authentication.Config{},
		},
		httpClient: http.DefaultClient,
	}
	_, err := scaler.ExecuteQuery(context.Background())
	assert.NoError(t, err)
}

func TestParqtelScalerRangeQueryURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/api/v1/query_range", request.URL.Path)
		query := request.URL.Query()
		assert.Equal(t, "30s", query.Get("step"))
		assert.NotEmpty(t, query.Get("start"))
		assert.NotEmpty(t, query.Get("end"))
		assert.NotEmpty(t, query.Get("query"))

		writer.WriteHeader(http.StatusOK)
		if _, err := writer.Write([]byte(`{"data":{"resultType":"matrix","result":[{"values": [[1, "3"]]}]}}`)); err != nil { // nosemgrep: no-direct-write-to-responsewriter
			t.Fatal(err)
		}
	}))
	defer server.Close()

	scaler := parqtelScaler{
		metadata: &parqtelMetadata{
			ServerAddress:    server.URL,
			Query:            "up",
			QueryType:        "range",
			RangeStep:        "30s",
			IgnoreNullValues: true,
			ParqtelAuth:      &authentication.Config{},
		},
		httpClient: http.DefaultClient,
	}
	value, err := scaler.ExecuteQuery(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 3.0, value)
}

func TestParqtelScalerRangeInverted(t *testing.T) {
	// A range whose start is after its end must be rejected before any request is made.
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("request should not be made for an inverted range")
	}))
	defer server.Close()

	scaler := parqtelScaler{
		metadata: &parqtelMetadata{
			ServerAddress:    server.URL,
			Query:            "up",
			QueryType:        "range",
			RangeStart:       "2000000000", // year 2033
			RangeEnd:         "1000000000", // year 2001
			IgnoreNullValues: true,
			ParqtelAuth:      &authentication.Config{},
		},
		httpClient: http.DefaultClient,
	}
	_, err := scaler.ExecuteQuery(context.Background())
	assert.Error(t, err)
}

func TestParqtelScalerLogsQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v1/logs/count", request.URL.Path)
		query := request.URL.Query()
		assert.Equal(t, `{service="api"}`, query.Get("query"))
		assert.NotEmpty(t, query.Get("start"))
		assert.NotEmpty(t, query.Get("end"))

		writer.WriteHeader(http.StatusOK)
		if _, err := writer.Write([]byte(`{"status":"success","data":[{"start_ns":0,"end_ns":60,"count":5},{"start_ns":60,"end_ns":120,"count":7}]}`)); err != nil { // nosemgrep: no-direct-write-to-responsewriter
			t.Fatal(err)
		}
	}))
	defer server.Close()

	scaler := parqtelScaler{
		metadata: &parqtelMetadata{
			ServerAddress:     server.URL,
			Signal:            "logs",
			Query:             `{service="api"}`,
			ResultAggregation: "sum",
			IgnoreNullValues:  true,
			ParqtelAuth:       &authentication.Config{},
		},
		httpClient: http.DefaultClient,
		logger:     logr.Discard(),
	}
	value, err := scaler.ExecuteQuery(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 12.0, value) // 5 + 7 buckets summed
}

func TestParqtelScalerTracesQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v1/traces/search", request.URL.Path)
		query := request.URL.Query()
		assert.Equal(t, `service.name="api"`, query.Get("q"))
		assert.NotEmpty(t, query.Get("start"))
		assert.NotEmpty(t, query.Get("end"))

		writer.WriteHeader(http.StatusOK)
		if _, err := writer.Write([]byte(`{"status":"success","data":{"trace_id":"abc","spans":[],"total_spans_in_range":100,"spans_matched":42,"truncated":false}}`)); err != nil { // nosemgrep: no-direct-write-to-responsewriter
			t.Fatal(err)
		}
	}))
	defer server.Close()

	scaler := parqtelScaler{
		metadata: &parqtelMetadata{
			ServerAddress:    server.URL,
			Signal:           "traces",
			Query:            `service.name="api"`,
			IgnoreNullValues: true,
			ParqtelAuth:      &authentication.Config{},
		},
		httpClient: http.DefaultClient,
		logger:     logr.Discard(),
	}
	value, err := scaler.ExecuteQuery(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 42.0, value) // spans_matched
}

func TestParqtelParseStepDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		isError  bool
	}{
		{"60", 60 * time.Second, false},
		{"60s", 60 * time.Second, false},
		{"5m", 5 * time.Minute, false},
		{"1h", time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"2.5s", 2500 * time.Millisecond, false},
		{"", 0, true},
		{"0", 0, true},
		{"-5s", 0, true},
		{"abc", 0, true},
		{"60x", 0, true},
	}
	for _, tt := range tests {
		got, err := parseStepDuration(tt.input)
		if tt.isError {
			assert.Error(t, err, "input=%q", tt.input)
		} else {
			assert.NoError(t, err, "input=%q", tt.input)
			assert.Equal(t, tt.expected, got, "input=%q", tt.input)
		}
	}
}

func TestParqtelParseRangeBound(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	nowSecs := float64(now.Unix())
	tests := []struct {
		input    string
		expected float64
		isError  bool
	}{
		// absolute unix seconds
		{"1686063687", 1686063687, false},
		{"1686063687.5", 1686063687.5, false},
		// absolute RFC3339
		{"2023-06-05T12:00:00Z", float64(time.Date(2023, 6, 5, 12, 0, 0, 0, time.UTC).Unix()), false},
		// relative durations are offsets before now
		{"5m", nowSecs - 300, false},
		{"60s", nowSecs - 60, false},
		{"1h", nowSecs - 3600, false},
		// invalid
		{"", 0, true},
		{"not-a-time", 0, true},
	}
	for _, tt := range tests {
		got, err := parseRangeBound(tt.input, now)
		if tt.isError {
			assert.Error(t, err, "input=%q", tt.input)
		} else {
			assert.NoError(t, err, "input=%q", tt.input)
			assert.Equal(t, tt.expected, got, "input=%q", tt.input)
		}
	}
}

func TestParqtelGetMetricsAndActivity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"data":{"result":[{"value": [1, "42"]}]}}`)) // nosemgrep: no-direct-write-to-responsewriter
	}))
	defer server.Close()

	scaler := parqtelScaler{
		metadata: &parqtelMetadata{
			ServerAddress:       server.URL,
			IgnoreNullValues:    true,
			ActivationThreshold: 10,
			ParqtelAuth:         &authentication.Config{},
		},
		httpClient: http.DefaultClient,
		logger:     logr.Discard(),
	}

	values, active, err := scaler.GetMetricsAndActivity(context.Background(), "s0-parqtel")
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.Equal(t, "s0-parqtel", values[0].MetricName)
	assert.True(t, active, "value 42 > activationThreshold 10 should be active")
}
