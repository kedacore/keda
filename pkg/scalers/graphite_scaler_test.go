package scalers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseGraphiteMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

type graphiteMetricIdentifier struct {
	metadataTestData *parseGraphiteMetadataTestData
	triggerIndex     int
	name             string
}

var testGrapMetadata = []parseGraphiteMetadataTestData{
	{map[string]string{}, true},
	// all properly formed
	{map[string]string{"serverAddress": "http://localhost:81", "threshold": "100", "activationThreshold": "23", "query": "stats.counters.http.hello-world.request.count.count", "queryTime": "-30Seconds"}, false},
	// missing serverAddress
	{map[string]string{"serverAddress": "", "threshold": "100", "query": "stats.counters.http.hello-world.request.count.count", "queryTime": "-30Seconds"}, true},
	// malformed threshold
	{map[string]string{"serverAddress": "http://localhost:81", "threshold": "one", "query": "stats.counters.http.hello-world.request.count.count", "queryTime": "-30Seconds"}, true},
	// malformed activationThreshold
	{map[string]string{"serverAddress": "http://localhost:81", "threshold": "100", "activationThreshold": "one", "query": "stats.counters.http.hello-world.request.count.count", "queryTime": "-30Seconds"}, true},
	// missing query
	{map[string]string{"serverAddress": "http://localhost:81", "threshold": "100", "query": "", "queryTime": "-30Seconds"}, true},
	// missing queryTime
	{map[string]string{"serverAddress": "http://localhost:81", "threshold": "100", "query": "stats.counters.http.hello-world.request.count.count", "queryTime": ""}, true},
}

var graphiteMetricIdentifiers = []graphiteMetricIdentifier{
	{&testGrapMetadata[1], 0, "s0-graphite"},
	{&testGrapMetadata[1], 1, "s1-graphite"},
}

type graphiteAuthMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

var testGraphiteAuthMetadata = []graphiteAuthMetadataTestData{
	// success basicAuth
	{map[string]string{"serverAddress": "http://localhost:81", "threshold": "100", "query": "stats.counters.http.hello-world.request.count.count", "queryTime": "-30Seconds", "authMode": "basic"}, map[string]string{"username": "user", "password": "pass"}, false},
	// fail basicAuth with no username
	{map[string]string{"serverAddress": "http://localhost:81", "threshold": "100", "query": "stats.counters.http.hello-world.request.count.count", "queryTime": "-30Seconds", "authMode": "basic"}, map[string]string{}, true},
	// fail if using non-basicAuth authMode
	{map[string]string{"serverAddress": "http://localhost:81", "threshold": "100", "query": "stats.counters.http.hello-world.request.count.count", "queryTime": "-30Seconds", "authMode": "tls"}, map[string]string{"username": "user"}, true},
}

type grapQueryResultTestData struct {
	name           string
	bodyStr        string
	responseStatus int
	expectedValue  float64
	isError        bool
}

var testGrapQueryResults = []grapQueryResultTestData{
	{
		name:           "no results",
		bodyStr:        `[{"target":"sumSeries(metric)","tags":{"name":"metric","aggregatedBy":"sum"},"datapoints":[]}]`,
		responseStatus: http.StatusOK,
		expectedValue:  0,
		isError:        false,
	},
	{
		name:           "valid response, latest datapoint is non-null",
		bodyStr:        `[{"target":"sumSeries(metric)","tags":{"name":"metric","aggregatedBy":"sum"},"datapoints":[[1,10000000]]}]`,
		responseStatus: http.StatusOK,
		expectedValue:  1,
		isError:        false,
	},
	{
		name:           "valid response, latest datapoint is null",
		bodyStr:        `[{"target":"sumSeries(metric)","tags":{"name":"metric","aggregatedBy":"sum"},"datapoints":[[1,10000000],[null,10000010]]}]`,
		responseStatus: http.StatusOK,
		expectedValue:  1,
		isError:        false,
	},
	{
		name:           "invalid response, all datapoints are null",
		bodyStr:        `[{"target":"sumSeries(metric)","tags":{"name":"metric","aggregatedBy":"sum"},"datapoints":[[null,10000000],[null,10000010]]}]`,
		responseStatus: http.StatusOK,
		expectedValue:  -1,
		isError:        true,
	},
	{
		name:           "multiple results",
		bodyStr:        `[{"target":"sumSeries(metric1)","tags":{"name":"metric1","aggregatedBy":"sum"},"datapoints":[[1,1000000]]}, {"target":"sumSeries(metric2)","tags":{"name":"metric2","aggregatedBy":"sum"},"datapoints":[[1,1000000]]}]`,
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

func TestGraphiteParseMetadata(t *testing.T) {
	for _, testData := range testGrapMetadata {
		_, err := parseGraphiteMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestGraphiteGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range graphiteMetricIdentifiers {
		ctx := context.Background()
		meta, err := parseGraphiteMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockGraphiteScaler := graphiteScaler{
			metadata: meta,
		}

		metricSpec := mockGraphiteScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestGraphiteScalerAuthParams(t *testing.T) {
	for _, testData := range testGraphiteAuthMetadata {
		meta, err := parseGraphiteMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}

		if err == nil {
			if meta.enableBasicAuth && !strings.Contains(testData.metadata["authMode"], "basic") {
				t.Error("wrong auth mode detected")
			}
		}
	}
}

func TestGrapScalerExecuteGrapQuery(t *testing.T) {
	for _, testData := range testGrapQueryResults {
		t.Run(testData.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.WriteHeader(testData.responseStatus)

				if _, err := writer.Write([]byte(testData.bodyStr)); err != nil {
					t.Fatal(err)
				}
			}))

			scaler := graphiteScaler{
				metadata: &graphiteMetadata{
					ServerAddress: server.URL,
				},
				httpClient: http.DefaultClient,
			}

			value, err := scaler.executeGrapQuery(context.TODO())

			assert.Equal(t, testData.expectedValue, value)

			if testData.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
