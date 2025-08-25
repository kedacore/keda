package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
)

// Test host URLs for validation
const (
	testValidMQQueueURL     = "https://qmtest.qm2.eu-gb.mq.appdomain.cloud/ibmmq/rest/v2/admin/action/qmgr/QM1/mqsc"
	testInvalidMQQueueURL   = "testInvalidURL.com"
	defaultTargetQueueDepth = 20
)

// Test data struct used for TestIBMMQParseMetadata
type parseIBMMQMetadataTestData struct {
	metadata   map[string]string
	isError    bool
	authParams map[string]string
}

var sampleIBMMQResolvedEnv = map[string]string{
	username: "ibmmquser",
	password: "ibmmqpass",
}

// Test metric identifier with test MQ data and it's name
type IBMMQMetricIdentifier struct {
	metadataTestData *parseIBMMQMetadataTestData
	triggerIndex     int
	name             string
}

// Setting metric identifier mock name
var IBMMQMetricIdentifiers = []IBMMQMetricIdentifier{
	{&testIBMMQMetadata[1], 0, "s0-ibmmq-testQueue"},
	{&testIBMMQMetadata[1], 1, "s1-ibmmq-testQueue"},
}

// Test cases for TestIBMMQParseMetadata test
var testIBMMQMetadata = []parseIBMMQMetadataTestData{
	// Nothing passed
	{map[string]string{}, true, map[string]string{}},
	// Properly formed metadata
	{map[string]string{"host": testValidMQQueueURL, "queueName": "testQueue", "queueDepth": "10"}, false, map[string]string{"username": "testUsername", "password": "Pass123"}},
	// Properly formed metadata with 2 queues
	{map[string]string{"host": testValidMQQueueURL, "queueName": "testQueue1, testQueue2", "queueDepth": "10"}, false, map[string]string{"username": "testUsername", "password": "Pass123"}},
	// Properly formed metadata with 2 queues with param queueNames
	{map[string]string{"host": testValidMQQueueURL, "queueNames": "testQueue1, testQueue2", "queueDepth": "10"}, false, map[string]string{"username": "testUsername", "password": "Pass123"}},
	// Invalid operation
	{map[string]string{"host": testValidMQQueueURL, "queueName": "testQueue1, testQueue2", "operation": "test", "queueDepth": "10"}, true, map[string]string{"username": "testUsername", "password": "Pass123"}},
	// Invalid queueDepth using a string
	{map[string]string{"host": testValidMQQueueURL, "queueName": "testQueue", "queueDepth": "AA"}, true, map[string]string{"username": "testUsername", "password": "Pass123"}},
	// Invalid activationQueueDepth using a string
	{map[string]string{"host": testValidMQQueueURL, "queueName": "testQueue", "queueDepth": "1", "activationQueueDepth": "AA"}, true, map[string]string{"username": "testUsername", "password": "Pass123"}},
	// No host provided
	{map[string]string{"queueName": "testQueue", "queueDepth": "10"}, true, map[string]string{"username": "testUsername", "password": "Pass123"}},
	// Missing queueName
	{map[string]string{"host": testValidMQQueueURL, "queueDepth": "10"}, true, map[string]string{"username": "testUsername", "password": "Pass123"}},
	// Invalid URL
	{map[string]string{"host": testInvalidMQQueueURL, "queueName": "testQueue", "queueDepth": "10"}, true, map[string]string{"username": "testUsername", "password": "Pass123"}},
	// Properly formed authParams Basic Auth
	{map[string]string{"host": testValidMQQueueURL, "queueName": "testQueue", "queueDepth": "10"}, false, map[string]string{"username": "testUsername", "password": "Pass123"}},
	// Properly formed authParams Basic Auth and TLS
	{map[string]string{"host": testValidMQQueueURL, "queueName": "testQueue", "queueDepth": "10"}, false, map[string]string{"username": "testUsername", "password": "Pass123", "ca": "cavalue", "cert": "certvalue", "key": "keyvalue"}},
	// No key provided
	{map[string]string{"host": testValidMQQueueURL, "queueName": "testQueue", "queueDepth": "10"}, true, map[string]string{"username": "testUsername", "password": "Pass123", "ca": "cavalue", "cert": "certvalue"}},
	// No username provided
	{map[string]string{"host": testValidMQQueueURL, "queueName": "testQueue", "queueDepth": "10"}, true, map[string]string{"password": "Pass123"}},
	// No password provided
	{map[string]string{"host": testValidMQQueueURL, "queueName": "testQueue", "queueDepth": "10"}, true, map[string]string{"username": "testUsername"}},
	// Wrong input unsafeSsl
	{map[string]string{"host": testValidMQQueueURL, "queueName": "testQueue", "queueDepth": "10", "unsafeSsl": "random"}, true, map[string]string{"username": "testUsername", "password": "Pass123"}},
}

// Test MQ Connection metadata is parsed correctly
// should error on missing required field
// and verify that the password field is handled correctly.
func TestIBMMQParseMetadata(t *testing.T) {
	for _, testData := range testIBMMQMetadata {
		metadata, err := parseIBMMQMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleIBMMQResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
			fmt.Println(testData)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
			fmt.Println(testData)
		}
		if metadata.Password != "" && metadata.Password != testData.authParams["password"] {
			t.Error("Expected password from configuration but found something else: ", metadata.Password)
			fmt.Println(testData)
		}
	}
}

// Test case for TestParseDefaultQueueDepth test
var testDefaultQueueDepth = []parseIBMMQMetadataTestData{
	{map[string]string{"host": testValidMQQueueURL, "queueName": "testQueue"}, false, map[string]string{"username": "testUsername", "password": "Pass123"}},
}

// Test that DefaultQueueDepth is set when queueDepth is not provided
func TestParseDefaultQueueDepth(t *testing.T) {
	for _, testData := range testDefaultQueueDepth {
		metadata, err := parseIBMMQMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleIBMMQResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		switch {
		case err != nil && !testData.isError:
			t.Error("Expected success but got error", err)
		case testData.isError && err == nil:
			t.Error("Expected error but got success")
		case metadata.QueueDepth != defaultTargetQueueDepth:
			t.Error("Expected default queueDepth =", defaultTargetQueueDepth, "but got", metadata.QueueDepth)
		}
	}
}

// Create a scaler and check if metrics method is available
func TestIBMMQGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range IBMMQMetricIdentifiers {
		metadata, err := parseIBMMQMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleIBMMQResolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.metadataTestData.authParams, TriggerIndex: testData.triggerIndex})

		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockIBMMQScaler := ibmmqScaler{
			metadata: metadata,
		}
		metricSpec := mockIBMMQScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name

		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

type queueDepthResultTestData struct {
	name           string
	bodyStr        string
	responseStatus int
	expectedValue  int64
	isError        bool
}

var testQueueDepthResults = []queueDepthResultTestData{
	{
		name: "valid response queue exists",
		bodyStr: `{
		"commandResponse": [{
			"completionCode": 0,
			"reasonCode": 0,
			"parameters": {
			"curdepth": 10,
			"type": "QLOCAL",
			"queue": "DEV.QUEUE.1"
			}
		}],
		"overallReasonCode": 0,
		"overallCompletionCode": 0
		}`,
		responseStatus: http.StatusOK,
		expectedValue:  10,
		isError:        false,
	},
	{
		name: "invalid response queue not found",
		bodyStr: `{
		"commandResponse": [{
			"completionCode": 2,
			"reasonCode": 2085,
			"message": ["AMQ8147E: IBM MQ object FAKE.QUEUE not found."]
		}],
		"overallReasonCode": 3008,
		"overallCompletionCode": 2
		}`,
		responseStatus: http.StatusOK,
		expectedValue:  0,
		isError:        true,
	},
	{
		name: "invalid response failed to parse commandResponse from REST call",
		bodyStr: `{
		"error": [{
			"msgId": "MQWB0009E",
			"action": "Resubmit the request with a valid queue manager name.",
			"completionCode": 2,
			"reasonCode": 2058,
			"type": "rest",
			"message": "MQWB0009E: Could not query the queue manager 'testqmgR'.",
			"explanation": "The REST API was invoked specifying a queue manager name which cannot be located."}]
		}`,
		responseStatus: http.StatusNotFound,
		expectedValue:  0,
		isError:        true,
	},
}

func TestIBMMQScalerGetQueueDepthViaHTTP(t *testing.T) {
	for _, testData := range testQueueDepthResults {
		t.Run(testData.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(testData.responseStatus)

				var body any
				if err := json.Unmarshal([]byte(testData.bodyStr), &body); err != nil {
					t.Fatal(err)
				}
				if err := json.NewEncoder(writer).Encode(body); err != nil {
					t.Fatal(err)
				}
			}))
			defer server.Close()

			scaler := ibmmqScaler{
				metadata: ibmmqMetadata{
					Host:      server.URL,
					QueueName: []string{"TEST.QUEUE"},
					Operation: "max",
				},
				httpClient: server.Client(),
			}

			value, err := scaler.getQueueDepthViaHTTP(context.Background())
			assert.Equal(t, testData.expectedValue, value)

			if testData.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
