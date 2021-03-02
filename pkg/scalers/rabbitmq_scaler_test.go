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
	// vhost passed
	{map[string]string{"vhostName": "myVhost", "queueName": "namespace/name", "hostFromEnv": host}, false, map[string]string{}},
	// vhost passed but empty
	{map[string]string{"vhostName": "", "queueName": "namespace/name", "hostFromEnv": host}, false, map[string]string{}},
	// protocol defined in authParams
	{map[string]string{"queueName": "sample", "hostFromEnv": host}, false, map[string]string{"protocol": "http"}},
	// auto protocol and a bad URL
	{map[string]string{"queueName": "sample", "host": "something://"}, true, map[string]string{}},
	// auto protocol and an HTTP URL
	{map[string]string{"queueName": "sample", "host": "http://"}, false, map[string]string{}},
	// auto protocol and an HTTPS URL
	{map[string]string{"queueName": "sample", "host": "https://"}, false, map[string]string{}},
	// publishRate number
	{map[string]string{rabbitPublishedPerSecondMetricName: "100", "queueName": "sample", "host": "https://"}, false, map[string]string{}},
	// publishRate not number
	{map[string]string{rabbitPublishedPerSecondMetricName: "AA", "queueName": "sample", "host": "https://"}, true, map[string]string{}},
	// publishRate http
	{map[string]string{rabbitPublishedPerSecondMetricName: "100", "queueName": "sample", "host": "http://"}, false, map[string]string{}},
	// publishRate amqp
	{map[string]string{rabbitPublishedPerSecondMetricName: "100", "queueName": "sample", "host": "amqp://"}, true, map[string]string{}},
	// publishRate amqps
	{map[string]string{rabbitPublishedPerSecondMetricName: "100", "queueName": "sample", "host": "amqps://"}, true, map[string]string{}},
	// publishRate and queueLength
	{map[string]string{rabbitPublishedPerSecondMetricName: "100", "queueLength": "10", "queueName": "sample", "host": "https://"}, true, map[string]string{}},
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
	{map[string]string{"queueName": "sample", "hostFromEnv": host}, false, map[string]string{}},
	// use default queueLength with includeUnacked
	{map[string]string{"queueName": "sample", "hostFromEnv": host, "protocol": "http"}, false, map[string]string{}},
	// use default queueLength with includeUnacked
	{map[string]string{"queueName": "sample", rabbitPublishedPerSecondMetricName: "100", "hostFromEnv": host, "protocol": "http"}, false, map[string]string{}},
}

func TestParseDefaultQueueLength(t *testing.T) {
	for _, testData := range testDefaultQueueLength {
		metadata, err := parseRabbitMQMetadata(&ScalerConfig{ResolvedEnv: sampleRabbitMqResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		switch {
		case err != nil && !testData.isError:
			t.Error("Expected success but got error", err)
		case testData.isError && err == nil:
			t.Error("Expected error but got success")
		case metadata.publishRate > 0 && metadata.queueLength != 0:
			t.Error("Expected default queueLength = 0 when publishRate is specified")
		case metadata.publishRate == 0 && metadata.queueLength != defaultRabbitMQQueueLength:
			t.Error("Expected default queueLength =", defaultRabbitMQQueueLength, "but got", metadata.queueLength)
		}
	}
}

type getQueueInfoTestData struct {
	response       string
	responseStatus int
	isActive       bool
	extraMetadata  map[string]string
	vhostPath      string
}

var testQueueInfoTestData = []getQueueInfoTestData{
	// queueLength
	{`{"messages": 4, "messages_unacknowledged": 1, "publish_details":{"rate":0}, "name": "evaluate_trials"}`, http.StatusOK, true, nil, ""},
	{`{"messages": 1, "messages_unacknowledged": 1, "publish_details":{"rate":0}, "name": "evaluate_trials"}`, http.StatusOK, true, nil, ""},
	{`{"messages": 1, "messages_unacknowledged": 0, "publish_details":{"rate":0}, "name": "evaluate_trials"}`, http.StatusOK, true, nil, ""},
	{`{"messages": 0, "messages_unacknowledged": 0, "publish_details":{"rate":0}, "name": "evaluate_trials"}`, http.StatusOK, false, nil, ""},
	{`{"messages": 4, "messages_unacknowledged": 1, "publish_details":{"rate":1743.2}, "name": "evaluate_trials"}`, http.StatusOK, true, nil, ""},
	{`{"messages": 1, "messages_unacknowledged": 1, "publish_details":{"rate":1743.2}, "name": "evaluate_trials"}`, http.StatusOK, true, nil, ""},
	{`{"messages": 1, "messages_unacknowledged": 0, "publish_details":{"rate":1743.2}, "name": "evaluate_trials"}`, http.StatusOK, true, nil, ""},
	{`{"messages": 0, "messages_unacknowledged": 0, "publish_details":{"rate":1743.2}, "name": "evaluate_trials"}`, http.StatusOK, false, nil, ""},
	// publishRate
	{`{"messages": 0, "messages_unacknowledged": 0, "publish_details":{"rate":1743.2}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{rabbitPublishedPerSecondMetricName: "100", "queueLength": "0"}, ""},
	{`{"messages": 0, "messages_unacknowledged": 0, "publish_details":{"rate":0}, "name": "evaluate_trials"}`, http.StatusOK, false, map[string]string{rabbitPublishedPerSecondMetricName: "100", "queueLength": "0"}, ""},
	{`{"messages": 1, "messages_unacknowledged": 1, "publish_details":{"rate":1743.2}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{rabbitPublishedPerSecondMetricName: "100", "queueLength": "0"}, ""},
	{`{"messages": 1, "messages_unacknowledged": 1, "publish_details":{"rate":0}, "name": "evaluate_trials"}`, http.StatusOK, false, map[string]string{rabbitPublishedPerSecondMetricName: "100", "queueLength": "0"}, ""},
	// error response
	{`Password is incorrect`, http.StatusUnauthorized, false, nil, ""},
}

var vhostPathes = []string{"/myhost", "", "/", "//", "/%2F"}

var testQueueInfoTestDataSingleVhost = []getQueueInfoTestData{
	{`{"messages": 4, "messages_unacknowledged": 1, "publish_details":{"rate":1743.2}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"hostFromEnv": "plainHost", "vhostName": "myhost"}, "/myhost"},
	{`{"messages": 4, "messages_unacknowledged": 1, "publish_details":{"rate":1743.2}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"hostFromEnv": "plainHost", "vhostName": "/"}, "/"},
	{`{"messages": 4, "messages_unacknowledged": 1, "publish_details":{"rate":1743.2}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"hostFromEnv": "plainHost", "vhostName": ""}, "/"},
	{`{"messages": 4, "messages_unacknowledged": 1, "publish_details":{"rate":0}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"hostFromEnv": "plainHost", "vhostName": "myhost"}, "/myhost"},
	{`{"messages": 4, "messages_unacknowledged": 1, "publish_details":{"rate":0}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"hostFromEnv": "plainHost", "vhostName": "/"}, "/"},
	{`{"messages": 4, "messages_unacknowledged": 1, "publish_details":{"rate":0}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"hostFromEnv": "plainHost", "vhostName": ""}, "/"},
}

func TestGetQueueInfo(t *testing.T) {
	allTestData := []getQueueInfoTestData{}
	for _, testData := range testQueueInfoTestData {
		for _, vhostPath := range vhostPathes {
			testData := testData
			testData.vhostPath = vhostPath
			allTestData = append(allTestData, testData)
		}
	}
	allTestData = append(allTestData, testQueueInfoTestDataSingleVhost...)

	for _, testData := range allTestData {
		testData := testData
		expectedVhost := "myhost"

		if testData.vhostPath != "/myhost" {
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

		resolvedEnv := map[string]string{host: fmt.Sprintf("%s%s", apiStub.URL, testData.vhostPath), "plainHost": apiStub.URL}

		metadata := map[string]string{
			"queueLength": "10",
			"queueName":   "evaluate_trials",
			"hostFromEnv": host,
			"protocol":    "http",
		}
		for k, v := range testData.extraMetadata {
			metadata[k] = v
		}

		s, err := NewRabbitMQScaler(
			&ScalerConfig{
				ResolvedEnv:       resolvedEnv,
				TriggerMetadata:   metadata,
				AuthParams:        map[string]string{},
				GlobalHTTPTimeout: 1000 * time.Millisecond,
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

func TestRabbitMQGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range rabbitMQMetricIdentifiers {
		meta, err := parseRabbitMQMetadata(&ScalerConfig{ResolvedEnv: sampleRabbitMqResolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: nil})
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
