package scalers

import (
	"context"
	"fmt"
	"testing"
)

type parseNewRelicMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

type newrelicMetricIdentifier struct {
	metadataTestData *parseNewRelicMetadataTestData
	scalerIndex      int
	name             string
}

var testNewRelicMetadata = []parseNewRelicMetadataTestData{
	{map[string]string{}, true},
	// all properly formed
	{map[string]string{"Account": "0", "metricName": "results", "threshold": "100", "QueryKey": "somekey", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, false},
	// all properly formed
	{map[string]string{"Account": "0", "Region": "EU", "metricName": "results", "threshold": "100", "QueryKey": "somekey", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, false},
	// Account as String
	{map[string]string{"Account": "ABC", "metricName": "results", "threshold": "100", "QueryKey": "somekey", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, true},
	// missing Account
	{map[string]string{"metricName": "results", "threshold": "100", "QueryKey": "somekey", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, true},
	// missing metricName
	{map[string]string{"Account": "0", "threshold": "100", "QueryKey": "somekey", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, true},
	// malformed threshold
	{map[string]string{"Account": "0", "metricName": "results", "threshold": "one", "QueryKey": "somekey", "nrql": "SELECT average(cpuUsedCores) as result FROM K8sContainerSample WHERE containerName='coredns'"}, true},
	// missing query
	{map[string]string{"Account": "0", "metricName": "results", "threshold": "100", "QueryKey": "somekey"}, true},
}

var newrelicMetricIdentifiers = []newrelicMetricIdentifier{
	{&testNewRelicMetadata[1], 0, "s0-newrelic-results"},
	{&testNewRelicMetadata[1], 1, "s1-newrelic-results"},
}

func TestNewRelicParseMetadata(t *testing.T) {
	for _, testData := range testNewRelicMetadata {
		_, err := parseNewRelicMetadata(&ScalerConfig{TriggerMetadata: testData.metadata})
		if err != nil && !testData.isError {
			fmt.Printf("X: %s", testData.metadata)
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			fmt.Printf("X: %s", testData.metadata)
			t.Error("Expected error but got success")
		}
	}
}
func TestNewRelicGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range newrelicMetricIdentifiers {
		meta, err := parseNewRelicMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ScalerIndex: testData.scalerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockNewRelicScaler := newrelicScaler{
			metadata: meta,
			nrClient: nil,
		}

		metricSpec := mockNewRelicScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

/*
type newrelicAuthMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
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
*/
