package scalers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
)

type parseLokiMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

var testLokiMetadata = []parseLokiMetadataTestData{
	{map[string]string{}, true},
	// all properly formed
	{map[string]string{"serverAddress": "http://localhost:3100", "threshold": "1", "query": "sum(rate({filename=\"/var/log/syslog\"}[1m])) by (level)"}, false},
	// all properly formed, with ignoreNullValues
	{map[string]string{"serverAddress": "http://localhost:3100", "threshold": "1", "query": "sum(rate({filename=\"/var/log/syslog\"}[1m])) by (level)", "ignoreNullValues": "false"}, false},
	// all properly formed, with activationThreshold
	{map[string]string{"serverAddress": "http://localhost:3100", "threshold": "1", "query": "sum(rate({filename=\"/var/log/syslog\"}[1m])) by (level)", "activationThreshold": "50"}, false},
	// missing serverAddress
	{map[string]string{"serverAddress": "", "threshold": "1", "query": "sum(rate({filename=\"/var/log/syslog\"}[1m])) by (level)"}, true},
	// missing threshold
	{map[string]string{"serverAddress": "http://localhost:3100", "query": "sum(rate({filename=\"/var/log/syslog\"}[1m])) by (level)"}, true},
	// malformed threshold
	{map[string]string{"serverAddress": "http://localhost:3100", "threshold": "one", "query": "sum(rate({filename=\"/var/log/syslog\"}[1m])) by (level)"}, true},
	// malformed threshold
	{map[string]string{"serverAddress": "http://localhost:3100", "threshold": "1", "activationThreshold": "one", "query": "sum(rate({filename=\"/var/log/syslog\"}[1m])) by (level)"}, true},
	// missing query
	{map[string]string{"serverAddress": "http://localhost:3100", "threshold": "1", "query": ""}, true},
	// ignoreNullValues with wrong value
	{map[string]string{"serverAddress": "http://localhost:3100", "threshold": "1", "query": "sum(rate({filename=\"/var/log/syslog\"}[1m])) by (level)", "ignoreNullValues": "xxxx"}, true},
	// with unsafeSsl
	{map[string]string{"serverAddress": "https://localhost:3100", "threshold": "1", "query": "sum(rate({filename=\"/var/log/syslog\"}[1m])) by (level)", "unsafeSsl": "true"}, false},
}

type lokiAuthMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

var testLokiAuthMetadata = []lokiAuthMetadataTestData{
	// success bearer default
	{map[string]string{"serverAddress": "http://localhost:3100", "threshold": "1", "query": "sum(rate({filename=\"/var/log/syslog\"}[1m])) by (level)", "authModes": "bearer"}, map[string]string{"bearerToken": "dummy-token"}, false},
	// fail bearerAuth with no token
	{map[string]string{"serverAddress": "http://localhost:3100", "threshold": "1", "query": "sum(rate({filename=\"/var/log/syslog\"}[1m])) by (level)", "authModes": "bearer"}, map[string]string{}, true},
	// success basicAuth
	{map[string]string{"serverAddress": "http://localhost:3100", "threshold": "1", "query": "sum(rate({filename=\"/var/log/syslog\"}[1m])) by (level)", "authModes": "basic"}, map[string]string{"username": "user", "password": "pass"}, false},
	// fail basicAuth with no username
	{map[string]string{"serverAddress": "http://localhost:3100", "threshold": "1", "query": "sum(rate({filename=\"/var/log/syslog\"}[1m])) by (level)", "authModes": "basic"}, map[string]string{}, true},
}

func TestLokiParseMetadata(t *testing.T) {
	for _, testData := range testLokiMetadata {
		_, err := parseLokiMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestLokiScalerAuthParams(t *testing.T) {
	for _, testData := range testLokiAuthMetadata {
		meta, err := parseLokiMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}

		if err == nil {
			if meta.Auth.EnableBasicAuth && !strings.Contains(testData.metadata["authModes"], "basic") {
				t.Error("wrong auth mode detected")
			}
		}
	}
}

type lokiQueryResultTestData struct {
	name             string
	bodyStr          string
	responseStatus   int
	expectedValue    float64
	isError          bool
	ignoreNullValues bool
	unsafeSsl        bool
}

var testLokiQueryResult = []lokiQueryResultTestData{
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
}

func TestLokiScalerExecuteLogQLQuery(t *testing.T) {
	for _, testData := range testLokiQueryResult {
		t.Run(testData.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(testData.responseStatus)
				if _, err := writer.Write([]byte(testData.bodyStr)); err != nil {
					t.Fatal(err)
				}
			}))

			scaler := lokiScaler{
				metadata: lokiMetadata{
					ServerAddress:    server.URL,
					IgnoreNullValues: testData.ignoreNullValues,
					UnsafeSsl:        testData.unsafeSsl,
				},
				httpClient: http.DefaultClient,
				logger:     logr.Discard(),
			}

			value, err := scaler.ExecuteLokiQuery(context.TODO())

			assert.Equal(t, testData.expectedValue, value)
			if testData.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLokiScalerTenantHeader(t *testing.T) {
	testData := lokiQueryResultTestData{
		name:             "no values",
		bodyStr:          `{"data":{"result":[]}}`,
		responseStatus:   http.StatusOK,
		expectedValue:    0,
		isError:          false,
		ignoreNullValues: true,
	}
	tenantName := "Tenant1"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		reqHeader := request.Header.Get(tenantNameHeaderKey)
		assert.Equal(t, reqHeader, tenantName)
		writer.WriteHeader(testData.responseStatus)
		if _, err := writer.Write([]byte(testData.bodyStr)); err != nil {
			t.Fatal(err)
		}
	}))

	scaler := lokiScaler{
		metadata: lokiMetadata{
			ServerAddress:    server.URL,
			TenantName:       tenantName,
			IgnoreNullValues: testData.ignoreNullValues,
		},
		httpClient: http.DefaultClient,
	}

	_, err := scaler.ExecuteLokiQuery(context.TODO())
	assert.NoError(t, err)
}
