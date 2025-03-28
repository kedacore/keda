package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parsePrometheusMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

type prometheusMetricIdentifier struct {
	metadataTestData *parsePrometheusMetadataTestData
	triggerIndex     int
	name             string
}

var testPromMetadata = []parsePrometheusMetadataTestData{
	{map[string]string{}, true},
	// all properly formed
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up"}, false},
	// all properly formed, with namespace
	{map[string]string{"serverAddress": "http://localhost:9090", "threshold": "100", "query": "up", "namespace": "foo"}, false},
	// all properly formed, with ignoreNullValues
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "ignoreNullValues": "false"}, false},
	// all properly formed, with activationThreshold
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "activationThreshold": "50"}, false},
	// missing serverAddress
	{map[string]string{"serverAddress": "", "metricName": "http_requests_total", "threshold": "100", "query": "up"}, true},
	// missing threshold
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "query": "up"}, true},
	// malformed threshold
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "one", "query": "up"}, true},
	// malformed threshold
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "activationThreshold": "one", "query": "up"}, true},
	// missing query
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": ""}, true},
	// ignoreNullValues with wrong value
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "ignoreNullValues": "xxxx"}, true},
	// unsafeSsl
	{map[string]string{"serverAddress": "https://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "unsafeSsl": "true"}, false},
	// customHeaders
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "customHeaders": "key1=value1,key2=value2"}, false},
	// customHeaders with wrong format
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "customHeaders": "key1=value1,key2"}, true},
	// queryParameters
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "queryParameters": "key1=value1,key2=value2"}, false},
	// queryParameters with wrong format
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "queryParameters": "key1=value1,key2"}, true},
	// valid custom http client timeout
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "timeout": "1000"}, false},
	// invalid - negative - custom http client timeout
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "timeout": "-1"}, true},
	// invalid - not a number - custom http client timeout with milliseconds
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "timeout": "a"}, true},
}

var prometheusMetricIdentifiers = []prometheusMetricIdentifier{
	{&testPromMetadata[1], 0, "s0-prometheus"},
	{&testPromMetadata[1], 1, "s1-prometheus"},
}

type prometheusAuthMetadataTestData struct {
	metadata            map[string]string
	authParams          map[string]string
	podIdentityProvider kedav1alpha1.PodIdentityProvider
	isError             bool
}

var testPrometheusAuthMetadata = []prometheusAuthMetadataTestData{
	// success TLS
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "tls"}, map[string]string{"ca": "caaa", "cert": "ceert", "key": "keey"}, "", false},
	// TLS, ca is optional
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "tls"}, map[string]string{"cert": "ceert", "key": "keey"}, "", false},
	// fail TLS, key not given
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "tls"}, map[string]string{"ca": "caaa", "cert": "ceert"}, "", true},
	// fail TLS, cert not given
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "tls"}, map[string]string{"ca": "caaa", "key": "keey"}, "", true},
	// success bearer default
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "bearer"}, map[string]string{"bearerToken": "tooooken"}, "", false},
	// fail bearerAuth with no token
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "bearer"}, map[string]string{}, "", true},
	// success basicAuth
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "basic"}, map[string]string{"username": "user", "password": "pass"}, "", false},
	// fail basicAuth with no username
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "basic"}, map[string]string{}, "", true},

	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "tls, basic"}, map[string]string{"ca": "caaa", "cert": "ceert", "key": "keey", "username": "user", "password": "pass"}, "", false},

	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "tls,basic"}, map[string]string{"username": "user", "password": "pass"}, "", true},
	// success custom auth
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "custom"}, map[string]string{"customAuthHeader": "header", "customAuthValue": "value"}, "", false},
	// fail custom auth with no customAuthHeader
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "custom"}, map[string]string{"customAuthHeader": ""}, "", true},
	// fail custom auth with no customAuthValue
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "custom"}, map[string]string{"customAuthValue": ""}, "", true},
	// success custom auth with newlines in customAuthHeader
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "custom"}, map[string]string{"customAuthHeader": "header\n", "customAuthValue": "value\n"}, "", false},

	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "tls,basic"}, map[string]string{"username": "user", "password": "pass"}, "", true},
	// pod identity and other auth modes enabled together
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up", "authModes": "basic"}, map[string]string{"username": "user", "password": "pass"}, "azure-workload", true},
	// azure workload identity
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up"}, nil, "azure-workload", false},
	// azure pod identity
	{map[string]string{"serverAddress": "http://localhost:9090", "metricName": "http_requests_total", "threshold": "100", "query": "up"}, nil, "azure", false},
}

func TestPrometheusParseMetadata(t *testing.T) {
	for _, testData := range testPromMetadata {
		_, err := parsePrometheusMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata})
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
		meta, err := parsePrometheusMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, TriggerIndex: testData.triggerIndex})
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
		meta, err := parsePrometheusMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams, PodIdentity: kedav1alpha1.AuthPodIdentity{Provider: testData.podIdentityProvider}})

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}

		if err == nil {
			if !meta.PrometheusAuth.Disabled() {
				if (meta.PrometheusAuth.EnabledBearerAuth() && !strings.Contains(testData.metadata["authModes"], "bearer")) ||
					(meta.PrometheusAuth.EnabledBasicAuth() && !strings.Contains(testData.metadata["authModes"], "basic")) ||
					(meta.PrometheusAuth.EnabledTLS() && !strings.Contains(testData.metadata["authModes"], "tls")) ||
					(meta.PrometheusAuth.EnabledCustomAuth() && !strings.Contains(testData.metadata["authModes"], "custom")) {
					t.Error("wrong auth mode detected")
				}
			}
		}
	}
}

type prometheusPromQueryResultTestData struct {
	name             string
	bodyStr          string
	responseStatus   int
	expectedValue    float64
	isError          bool
	ignoreNullValues bool
	unsafeSsl        bool
}

var testPromQueryResult = []prometheusPromQueryResultTestData{
	{
		name:             "no results",
		bodyStr:          `{}`,
		responseStatus:   http.StatusOK,
		expectedValue:    0,
		isError:          false,
		ignoreNullValues: true,
		unsafeSsl:        false,
	},
	{
		name:             "no values",
		bodyStr:          `{"data":{"result":[]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    0,
		isError:          false,
		ignoreNullValues: true,
		unsafeSsl:        true,
	},
	{
		name:             "no values but shouldn't ignore",
		bodyStr:          `{"data":{"result":[]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    -1,
		isError:          true,
		ignoreNullValues: false,
		unsafeSsl:        false,
	},
	{
		name:             "value is empty list",
		bodyStr:          `{"data":{"result":[{"value": []}]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    0,
		isError:          false,
		ignoreNullValues: true,
		unsafeSsl:        true,
	},
	{
		name:             "value is empty list but shouldn't ignore",
		bodyStr:          `{"data":{"result":[{"value": []}]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    -1,
		isError:          true,
		ignoreNullValues: false,
		unsafeSsl:        false,
	},
	{
		name:             "valid value",
		bodyStr:          `{"data":{"result":[{"value": ["1", "2"]}]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    2,
		isError:          false,
		ignoreNullValues: true,
		unsafeSsl:        true,
	},
	{
		name:             "not enough values",
		bodyStr:          `{"data":{"result":[{"value": ["1"]}]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    -1,
		isError:          true,
		ignoreNullValues: true,
		unsafeSsl:        true,
	},
	{
		name:             "multiple results",
		bodyStr:          `{"data":{"result":[{},{}]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    -1,
		isError:          true,
		ignoreNullValues: true,
		unsafeSsl:        true,
	},
	{
		name:             "error status response",
		bodyStr:          `{}`,
		responseStatus:   http.StatusBadRequest,
		expectedValue:    -1,
		isError:          true,
		ignoreNullValues: true,
		unsafeSsl:        true,
	},
	{
		name:             "+Inf",
		bodyStr:          `{"data":{"result":[{"value": ["1", "+Inf"]}]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    0,
		isError:          false,
		ignoreNullValues: true,
		unsafeSsl:        true,
	},
	{
		name:             "+Inf but shouldn't ignore ",
		bodyStr:          `{"data":{"result":[{"value": ["1", "+Inf"]}]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    -1,
		isError:          true,
		ignoreNullValues: false,
		unsafeSsl:        true,
	},
	{
		name:             "-Inf",
		bodyStr:          `{"data":{"result":[{"value": ["1", "-Inf"]}]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    0,
		isError:          false,
		ignoreNullValues: true,
		unsafeSsl:        true,
	},
	{
		name:             "-Inf but shouldn't ignore ",
		bodyStr:          `{"data":{"result":[{"value": ["1", "-Inf"]}]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    -1,
		isError:          true,
		ignoreNullValues: false,
		unsafeSsl:        true,
	},
}

func TestPrometheusScalerExecutePromQuery(t *testing.T) {
	for _, testData := range testPromQueryResult {
		t.Run(testData.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(testData.responseStatus)

				if _, err := writer.Write([]byte(testData.bodyStr)); err != nil {
					t.Fatal(err)
				}
			}))

			scaler := prometheusScaler{
				metadata: &prometheusMetadata{
					ServerAddress:    server.URL,
					IgnoreNullValues: testData.ignoreNullValues,
					UnsafeSSL:        testData.unsafeSsl,
				},
				httpClient: http.DefaultClient,
				logger:     logr.Discard(),
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

func TestPrometheusScalerCustomHeaders(t *testing.T) {
	testData := prometheusPromQueryResultTestData{
		name:             "no values",
		bodyStr:          `{"data":{"result":[]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    0,
		isError:          false,
		ignoreNullValues: true,
	}
	customHeadersValue := map[string]string{
		"X-Client-Id":          "cid",
		"X-Tenant-Id":          "tid",
		"X-Organization-Token": "oid",
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		for headerName, headerValue := range customHeadersValue {
			reqHeader := request.Header.Get(headerName)
			assert.Equal(t, reqHeader, headerValue)
		}

		writer.WriteHeader(testData.responseStatus)
		if _, err := writer.Write([]byte(testData.bodyStr)); err != nil {
			t.Fatal(err)
		}
	}))

	scaler := prometheusScaler{
		metadata: &prometheusMetadata{
			ServerAddress:    server.URL,
			CustomHeaders:    customHeadersValue,
			IgnoreNullValues: testData.ignoreNullValues,
		},
		httpClient: http.DefaultClient,
	}

	_, err := scaler.ExecutePromQuery(context.TODO())

	assert.NoError(t, err)
}

func TestPrometheusScalerExecutePromQueryParameters(t *testing.T) {
	testData := prometheusPromQueryResultTestData{
		name:             "no values",
		bodyStr:          `{"data":{"result":[]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    0,
		isError:          false,
		ignoreNullValues: true,
	}
	queryParametersValue := map[string]string{
		"first":  "foo",
		"second": "bar",
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		queryParameter := request.URL.Query()
		time := time.Now().UTC().Format(time.RFC3339)
		require.Equal(t, queryParameter.Get("time"), time)

		for queryParameterName, queryParameterValue := range queryParametersValue {
			require.Equal(t, queryParameter.Get(queryParameterName), queryParameterValue)
		}

		expectedPath := "/api/v1/query"
		require.Equal(t, request.URL.Path, expectedPath)

		writer.WriteHeader(testData.responseStatus)
		if _, err := writer.Write([]byte(testData.bodyStr)); err != nil {
			t.Fatal(err)
		}
	}))
	scaler := prometheusScaler{
		metadata: &prometheusMetadata{
			ServerAddress:    server.URL,
			QueryParameters:  queryParametersValue,
			IgnoreNullValues: testData.ignoreNullValues,
		},
		httpClient: http.DefaultClient,
	}
	_, err := scaler.ExecutePromQuery(context.TODO())

	assert.NoError(t, err)
}

func TestPrometheusScaler_ExecutePromQuery_WithGCPNativeAuthentication(t *testing.T) {
	fakeGoogleOAuthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"token_type": "Bearer", "access_token": "fake_access_token"}`)
	}))
	defer fakeGoogleOAuthServer.Close()

	fakeGCPCredsJSON, err := json.Marshal(map[string]string{
		"type": "service_account",
		"private_key": `-----BEGIN RSA PRIVATE KEY-----
MIIBOwIBAAJBAOfgBHLEOcXo2X+8SSzF1rEsTewRzZIOZAak4XRULY+dBd1bsGBM
+dOb9a65cJbDuL3zmTZnfAxjmh2ueNTZvOcCAwEAAQJBAMwwibpG8llF48KInCfB
UH5U9YmdY9nqskrnh2JZfoWnpBbGxtqg0vbdmvEL2bcbeUnudF25mPpoONw1F6G6
5IECIQD0ouUBttDMacs5XqQppYCb8eAmiMkJxwgtJfPb9iGm0wIhAPKlXzNgIsMP
v3sqXcOO3tNjEohptOpEyLWyCt3Htm0dAiB9w/CvfOjC7fCIQdtrfaYshaCSrueL
m0Lc0xIXFuYd+QIgZ9DpkomnVd3/BytxQqJ2I+tXmpXfmfwkA9lRXOJ94uECIQC8
IisErx3ap2o99Zn+Yotv/TGZkS+lfMLdbcOBr8a57Q==
-----END RSA PRIVATE KEY-----`,
		"token_uri": fakeGoogleOAuthServer.URL,
	})
	require.NoError(t, err)

	fakeGCPCredsPath := filepath.Join(t.TempDir(), "fake_application_default_credentials.json")

	f, err := os.Create(fakeGCPCredsPath)
	require.NoError(t, err)
	_, err = f.Write(fakeGCPCredsJSON)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	newFakeServer := func(t *testing.T) *httptest.Server {
		t.Helper()
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v1/projects/my-fake-project/location/global/prometheus/api/v1/query", r.URL.Path)
			assert.True(t, r.URL.Query().Has("time"))
			assert.Equal(t, "sum(rate(http_requests_total{instance=\"my-instance\"}[5m]))", r.URL.Query().Get("query"))

			if !assert.Equal(t, "Bearer fake_access_token", r.Header.Get("Authorization")) {
				w.WriteHeader(http.StatusUnauthorized)
			}

			assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"status": "success",
				"data": map[string]any{
					"resultType": "vector",
					"result": []map[string]any{
						{"metric": map[string]string{}, "value": []any{1686063687, "777"}},
					},
				},
			}))
		}))
	}

	tests := map[string]struct {
		config func(*testing.T, *scalersconfig.ScalerConfig) *scalersconfig.ScalerConfig
	}{
		"using GCP workload identity": {
			config: func(t *testing.T, config *scalersconfig.ScalerConfig) *scalersconfig.ScalerConfig {
				t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", fakeGCPCredsPath)
				config.PodIdentity = kedav1alpha1.AuthPodIdentity{
					Provider: kedav1alpha1.PodIdentityProviderGCP,
				}
				return config
			},
		},

		"with Google app credentials on auth params": {
			config: func(t *testing.T, config *scalersconfig.ScalerConfig) *scalersconfig.ScalerConfig {
				config.AuthParams = map[string]string{
					"GoogleApplicationCredentials": string(fakeGCPCredsJSON),
				}
				return config
			},
		},

		"with Google app credentials on envs": {
			config: func(t *testing.T, config *scalersconfig.ScalerConfig) *scalersconfig.ScalerConfig {
				config.TriggerMetadata["credentialsFromEnv"] = "GCP_APP_CREDENTIALS"
				config.ResolvedEnv = map[string]string{
					"GCP_APP_CREDENTIALS": string(fakeGCPCredsJSON),
				}
				return config
			},
		},

		"with Google app credentials file on auth params": {
			config: func(t *testing.T, config *scalersconfig.ScalerConfig) *scalersconfig.ScalerConfig {
				config.TriggerMetadata["credentialsFromEnvFile"] = "GCP_APP_CREDENTIALS"
				config.ResolvedEnv = map[string]string{
					"GCP_APP_CREDENTIALS": fakeGCPCredsPath,
				}
				return config
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			server := newFakeServer(t)
			defer server.Close()

			baseConfig := &scalersconfig.ScalerConfig{
				TriggerMetadata: map[string]string{
					"serverAddress": server.URL + "/v1/projects/my-fake-project/location/global/prometheus",
					"query":         "sum(rate(http_requests_total{instance=\"my-instance\"}[5m]))",
					"threshold":     "100",
				},
			}

			require.NotNil(t, tt.config, "you must provide a config generator func")
			config := tt.config(t, baseConfig)

			scaler, err := NewPrometheusScaler(config)
			require.NoError(t, err)

			s, ok := scaler.(*prometheusScaler)
			require.True(t, ok, "Scaler must be a Prometheus Scaler")
			_, ok = s.httpClient.Transport.(*oauth2.Transport)
			require.True(t, ok, "HTTP transport must be Google OAuth2")

			got, err := s.ExecutePromQuery(context.TODO())
			require.NoError(t, err)
			assert.Equal(t, float64(777), got)
		})
	}
}
