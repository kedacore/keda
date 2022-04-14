package scalers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type parsePrometheusMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

type prometheusMetricIdentifier struct {
	metadataTestData *parsePrometheusMetadataTestData
	scalerIndex      int
	name             string
}

var testPromMetadata = []parsePrometheusMetadataTestData{
	{map[string]string{}, true},
	// all properly formed
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up"}, false},
	// all properly formed, with namespace
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "namespace": "foo"}, false},
	// missing serverAddress
	{map[string]string{"serverAddress": "", "metricName": "http_requests_total", "threshold": "100", "query": "up"}, true},
	// missing metricName
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "", "threshold": "100", "query": "up"}, true},
	// missing threshold
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "query": "up"}, true},
	// malformed threshold
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "one", "query": "up"}, true},
	// missing query
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": ""}, true},
}

var prometheusMetricIdentifiers = []prometheusMetricIdentifier{
	{&testPromMetadata[1], 0, "s0-prometheus-http_requests_total"},
	{&testPromMetadata[1], 1, "s1-prometheus-http_requests_total"},
}

type prometheusAuthMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

var testPrometheusAuthMetadata = []prometheusAuthMetadataTestData{
	// success TLS
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "tls"}, map[string]string{"ca": "caaa", "cert": "ceert", "key": "keey"}, false},
	// TLS, ca is optional
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "tls"}, map[string]string{"cert": "ceert", "key": "keey"}, false},
	// fail TLS, key not given
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "tls"}, map[string]string{"ca": "caaa", "cert": "ceert"}, true},
	// fail TLS, cert not given
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "tls"}, map[string]string{"ca": "caaa", "key": "keey"}, true},
	// success bearer default
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "bearer"}, map[string]string{"bearerToken": "tooooken"}, false},
	// fail bearerAuth with no token
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "bearer"}, map[string]string{}, true},
	// success basicAuth
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "basic"}, map[string]string{"username": "user", "password": "pass"}, false},
	// fail basicAuth with no username
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "basic"}, map[string]string{}, true},

	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "tls, basic"}, map[string]string{"ca": "caaa", "cert": "ceert", "key": "keey", "username": "user", "password": "pass"}, false},

	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "tls,basic"}, map[string]string{"username": "user", "password": "pass"}, true},
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
		meta, err := parsePrometheusMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ScalerIndex: testData.scalerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockPrometheusScaler := prometheusScaler{
			metadata:   meta,
			httpClient: http.DefaultClient,
		}

		metricSpec := mockPrometheusScaler.GetMetricSpecForScaling(context.Background())
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
			if (meta.prometheusAuth.EnableBearerAuth && !strings.Contains(testData.metadata["authModes"], "bearer")) ||
				(meta.prometheusAuth.EnableBasicAuth && !strings.Contains(testData.metadata["authModes"], "basic")) ||
				(meta.prometheusAuth.EnableTLS && !strings.Contains(testData.metadata["authModes"], "tls")) {
				t.Error("wrong auth mode detected")
			}
		}
	}
}

type prometheusQromQueryResultTestData struct {
	name           string
	bodyStr        string
	responseStatus int
	expectedValue  float64
	isError        bool
}

var testPromQueryResult = []prometheusQromQueryResultTestData{
	{
		name:           "no results",
		bodyStr:        `{}`,
		responseStatus: http.StatusOK,
		expectedValue:  0,
		isError:        false,
	},
	{
		name:           "no values",
		bodyStr:        `{"data":{"result":[]}}`,
		responseStatus: http.StatusOK,
		expectedValue:  0,
		isError:        false,
	},
	{
		name:           "valid value",
		bodyStr:        `{"data":{"result":[{"value": ["1", "2"]}]}}`,
		responseStatus: http.StatusOK,
		expectedValue:  2,
		isError:        false,
	},
	{
		name:           "not enough values",
		bodyStr:        `{"data":{"result":[{"value": ["1"]}]}}`,
		responseStatus: http.StatusOK,
		expectedValue:  -1,
		isError:        true,
	},
	{
		name:           "multiple results",
		bodyStr:        `{"data":{"result":[{},{}]}}`,
		responseStatus: http.StatusOK,
		expectedValue:  -1,
		isError:        true,
	},
	{
		name:           "error status response",
		bodyStr:        `{}`,
		responseStatus: http.StatusBadRequest,
		expectedValue:  -1,
		isError:        true,
	},
}

func TestPrometheusScalerExecutePromQuery(t *testing.T) {
	for _, testData := range testPromQueryResult {
		t.Run(testData.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.WriteHeader(testData.responseStatus)

				if _, err := writer.Write([]byte(testData.bodyStr)); err != nil {
					t.Fatal(err)
				}
			}))

			scaler := prometheusScaler{
				metadata: &prometheusMetadata{
					serverAddress: server.URL,
				},
				httpClient: http.DefaultClient,
			}

			value, err := scaler.ExecutePromQuery(context.TODO())

			assert.Equal(t, testData.expectedValue, value)

			if testData.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPrometheusScalerCortexHeader(t *testing.T) {
	testData := prometheusQromQueryResultTestData{
		name:           "no values",
		bodyStr:        `{"data":{"result":[]}}`,
		responseStatus: http.StatusOK,
		expectedValue:  0,
		isError:        false,
	}
	cortexOrgValue := "my-org"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		reqHeader := request.Header.Get(promCortexHeaderKey)
		assert.Equal(t, reqHeader, cortexOrgValue)
		writer.WriteHeader(testData.responseStatus)
		if _, err := writer.Write([]byte(testData.bodyStr)); err != nil {
			t.Fatal(err)
		}
	}))

	scaler := prometheusScaler{
		metadata: &prometheusMetadata{
			serverAddress: server.URL,
			cortexOrgID:   cortexOrgValue,
		},
		httpClient: http.DefaultClient,
	}

	_, err := scaler.ExecutePromQuery(context.TODO())

	assert.NoError(t, err)
}
