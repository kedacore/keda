package scalers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	host = "myHostSecret"
)

type parseRabbitMQMetadataTestData struct {
	metadata   map[string]string
	isError    bool
	authParams map[string]string
}

type rabbitMQMetricIdentifier struct {
	metadataTestData *parseRabbitMQMetadataTestData
	name             string
}

var sampleRabbitMqResolvedEnv = map[string]string{
	host: "amqp://user:sercet@somehost.com:5236/vhost",
}

var testRabbitMQMetadata = []parseRabbitMQMetadataTestData{
	// nothing passed
	{map[string]string{}, true, map[string]string{}},
	// properly formed metadata
	{map[string]string{"queueLength": "10", "queueName": "sample", "hostFromEnv": host}, false, map[string]string{}},
	// malformed queueLength
	{map[string]string{"queueLength": "AA", "queueName": "sample", "hostFromEnv": host}, true, map[string]string{}},
	// missing host
	{map[string]string{"queueLength": "AA", "queueName": "sample"}, true, map[string]string{}},
	// missing queueName
	{map[string]string{"queueLength": "10", "hostFromEnv": host}, true, map[string]string{}},
	// host defined in authParams
	{map[string]string{"queueLength": "10"}, true, map[string]string{"host": host}},
	// properly formed metadata with http protocol
	{map[string]string{"queueLength": "10", "queueName": "sample", "host": host, "protocol": "http"}, false, map[string]string{}},
	// queue name with slashes
	{map[string]string{"queueLength": "10", "queueName": "namespace/name", "hostFromEnv": host}, false, map[string]string{}},
}

var rabbitMQMetricIdentifiers = []rabbitMQMetricIdentifier{
	{&testRabbitMQMetadata[1], "rabbitmq-sample"},
	{&testRabbitMQMetadata[7], "rabbitmq-namespace-name"},
}

func TestRabbitMQParseMetadata(t *testing.T) {
	for _, testData := range testRabbitMQMetadata {
		_, err := parseRabbitMQMetadata(&ScalerConfig{ResolvedEnv: sampleRabbitMqResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

var testDefaultQueueLength = []parseRabbitMQMetadataTestData{
	// use default queueLength
	{map[string]string{"queueName": "sample", "host": host}, false, map[string]string{}},
	// use default queueLength with includeUnacked
	{map[string]string{"queueName": "sample", "host": host, "protocol": "http"}, false, map[string]string{}},
}

func TestParseDefaultQueueLength(t *testing.T) {
	for _, testData := range testDefaultQueueLength {
		metadata, err := parseRabbitMQMetadata(&ScalerConfig{ResolvedEnv: sampleRabbitMqResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		switch {
		case err != nil && !testData.isError:
			t.Error("Expected success but got error", err)
		case testData.isError && err == nil:
			t.Error("Expected error but got success")
		case metadata.queueLength != defaultRabbitMQQueueLength:
			t.Error("Expected default queueLength =", defaultRabbitMQQueueLength, "but got", metadata.queueLength)
		}
	}
}

type getQueueInfoTestData struct {
	response       string
	responseStatus int
	isActive       bool
}

var testQueueInfoTestData = []getQueueInfoTestData{
	{`{"messages": 4, "messages_unacknowledged": 1, "name": "evaluate_trials"}`, http.StatusOK, true},
	{`{"messages": 1, "messages_unacknowledged": 1, "name": "evaluate_trials"}`, http.StatusOK, true},
	{`{"messages": 1, "messages_unacknowledged": 0, "name": "evaluate_trials"}`, http.StatusOK, true},
	{`{"messages": 0, "messages_unacknowledged": 0, "name": "evaluate_trials"}`, http.StatusOK, false},
	{`Password is incorrect`, http.StatusUnauthorized, false},
}

var vhostPathes = []string{"/myhost", "", "/", "//", "/%2F"}

func TestGetQueueInfo(t *testing.T) {
	for _, testData := range testQueueInfoTestData {
		testData := testData
		for _, vhostPath := range vhostPathes {
			expectedVhost := "myhost"

			if vhostPath != "/myhost" {
				expectedVhost = "%2F"
			}

			var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/api/queues/" + expectedVhost + "/evaluate_trials"
				if r.RequestURI != expectedPath {
					t.Error("Expect request path to =", expectedPath, "but it is", r.RequestURI)
				}

				w.WriteHeader(testData.responseStatus)
				_, err := w.Write([]byte(testData.response))
				if err != nil {
					t.Error("Expect request path to =", testData.response, "but it is", err)
				}
			}))

			resolvedEnv := map[string]string{host: fmt.Sprintf("%s%s", apiStub.URL, vhostPath)}

			metadata := map[string]string{
				"queueLength": "10",
				"queueName":   "evaluate_trials",
				"hostFromEnv": host,
				"protocol":    "http",
			}

			s, err := NewRabbitMQScaler(
				&ScalerConfig{
					ResolvedEnv:       resolvedEnv,
					TriggerMetadata:   metadata,
					AuthParams:        map[string]string{},
					GlobalHTTPTimeout: 300 * time.Millisecond,
				},
			)

			if err != nil {
				t.Error("Expect success", err)
			}

			ctx := context.TODO()
			active, err := s.IsActive(ctx)

			if testData.responseStatus == http.StatusOK {
				if err != nil {
					t.Error("Expect success", err)
				}

				if active != testData.isActive {
					if testData.isActive {
						t.Error("Expect to be active")
					} else {
						t.Error("Expect to not be active")
					}
				}
			} else if !strings.Contains(err.Error(), testData.response) {
				t.Error("Expect error to be like '", testData.response, "' but it's '", err, "'")
			}
		}
	}
}

func TestRabbitMQGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range rabbitMQMetricIdentifiers {
		meta, err := parseRabbitMQMetadata(&ScalerConfig{ResolvedEnv: map[string]string{"myHostSecret": "myHostSecret"}, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: nil})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockRabbitMQScaler := rabbitMQScaler{
			metadata:   meta,
			connection: nil,
			channel:    nil,
			httpClient: http.DefaultClient,
		}

		metricSpec := mockRabbitMQScaler.GetMetricSpecForScaling()
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName, "wanted:", testData.name)
		}
	}
}
