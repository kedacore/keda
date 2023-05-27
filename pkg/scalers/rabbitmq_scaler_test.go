package scalers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	host = "myHostSecret"
)

type parseRabbitMQMetadataTestData struct {
	metadata   map[string]string
	isError    bool
	authParams map[string]string
}

type parseRabbitMQAuthParamTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
	enableTLS  bool
}

type rabbitMQMetricIdentifier struct {
	metadataTestData *parseRabbitMQMetadataTestData
	index            int
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
	{map[string]string{"queueLength": "10", "hostFromEnv": host}, true, map[string]string{"host": host}},
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
	// queueLength and mode
	{map[string]string{"queueLength": "10", "mode": "QueueLength", "queueName": "sample", "host": "https://"}, true, map[string]string{}},
	// queueLength and value
	{map[string]string{"queueLength": "10", "value": "20", "queueName": "sample", "host": "https://"}, true, map[string]string{}},
	// queueLength and mode and value
	{map[string]string{"queueLength": "10", "mode": "QueueLength", "value": "20", "queueName": "sample", "host": "https://"}, true, map[string]string{}},
	// only mode
	{map[string]string{"mode": "QueueLength", "queueName": "sample", "host": "https://"}, true, map[string]string{}},
	// only value
	{map[string]string{"value": "20", "queueName": "sample", "host": "https://"}, true, map[string]string{}},
	// mode and value
	{map[string]string{"mode": "QueueLength", "value": "20", "queueName": "sample", "host": "https://"}, false, map[string]string{}},
	// invalid mode
	{map[string]string{"mode": "Feelings", "value": "20", "queueName": "sample", "host": "https://"}, true, map[string]string{}},
	// invalid value
	{map[string]string{"mode": "QueueLength", "value": "lots", "queueName": "sample", "host": "https://"}, true, map[string]string{}},
	// queue length amqp
	{map[string]string{"mode": "QueueLength", "value": "20", "queueName": "sample", "host": "amqps://"}, false, map[string]string{}},
	// message rate amqp
	{map[string]string{"mode": "MessageRate", "value": "1000", "queueName": "sample", "host": "amqps://"}, true, map[string]string{}},
	// message rate amqp
	{map[string]string{"mode": "MessageRate", "value": "1000", "queueName": "sample", "host": "amqp://"}, true, map[string]string{}},
	// message rate amqp
	{map[string]string{"mode": "MessageRate", "value": "1000", "queueName": "sample", "host": "http://"}, false, map[string]string{}},
	// message rate amqp
	{map[string]string{"mode": "MessageRate", "value": "1000", "queueName": "sample", "host": "https://"}, false, map[string]string{}},
	// amqp host and useRegex
	{map[string]string{"queueName": "sample", "host": "amqps://", "useRegex": "true"}, true, map[string]string{}},
	// http host and useRegex
	{map[string]string{"queueName": "sample", "host": "http://", "useRegex": "true"}, false, map[string]string{}},
	// message rate and useRegex
	{map[string]string{"mode": "MessageRate", "value": "1000", "queueName": "sample", "host": "http://", "useRegex": "true"}, false, map[string]string{}},
	// queue length and useRegex
	{map[string]string{"mode": "QueueLength", "value": "1000", "queueName": "sample", "host": "http://", "useRegex": "true"}, false, map[string]string{}},
	// custom metric name
	{map[string]string{"mode": "QueueLength", "value": "1000", "queueName": "sample", "host": "http://", "useRegex": "true", "metricName": "host1-sample"}, false, map[string]string{}},
	// http valid timeout
	{map[string]string{"mode": "QueueLength", "value": "1000", "queueName": "sample", "host": "http://", "timeout": "1000"}, false, map[string]string{}},
	// http invalid timeout
	{map[string]string{"mode": "QueueLength", "value": "1000", "queueName": "sample", "host": "http://", "timeout": "-10"}, true, map[string]string{}},
	// http wrong timeout
	{map[string]string{"mode": "QueueLength", "value": "1000", "queueName": "sample", "host": "http://", "timeout": "error"}, true, map[string]string{}},
	// amqp timeout
	{map[string]string{"mode": "QueueLength", "value": "1000", "queueName": "sample", "host": "amqp://", "timeout": "10"}, true, map[string]string{}},
	// valid pageSize
	{map[string]string{"mode": "MessageRate", "value": "1000", "queueName": "sample", "host": "http://", "useRegex": "true", "pageSize": "100"}, false, map[string]string{}},
	// pageSize less than 1
	{map[string]string{"mode": "MessageRate", "value": "1000", "queueName": "sample", "host": "http://", "useRegex": "true", "pageSize": "-1"}, true, map[string]string{}},
	// invalid pageSize
	{map[string]string{"mode": "MessageRate", "value": "1000", "queueName": "sample", "host": "http://", "useRegex": "true", "pageSize": "a"}, true, map[string]string{}},
	// activationValue passed
	{map[string]string{"activationValue": "10", "queueLength": "20", "queueName": "sample", "hostFromEnv": host}, false, map[string]string{}},
	// malformed activationValue
	{map[string]string{"activationValue": "AA", "queueLength": "10", "queueName": "sample", "hostFromEnv": host}, true, map[string]string{}},
	// http and excludeUnacknowledged
	{map[string]string{"mode": "QueueLength", "value": "1000", "queueName": "sample", "host": "http://", "useRegex": "true", "excludeUnacknowledged": "true"}, false, map[string]string{}},
	// amqp and excludeUnacknowledged
	{map[string]string{"mode": "QueueLength", "value": "1000", "queueName": "sample", "host": "amqp://", "useRegex": "true", "excludeUnacknowledged": "true"}, true, map[string]string{}},
	// unsafeSsl true
	{map[string]string{"queueName": "sample", "host": "https://", "unsafeSsl": "true"}, false, map[string]string{}},
	// unsafeSsl wrong input
	{map[string]string{"queueName": "sample", "host": "https://", "unsafeSsl": "random"}, true, map[string]string{}},
}

var testRabbitMQAuthParamData = []parseRabbitMQAuthParamTestData{
	{map[string]string{"queueName": "sample", "hostFromEnv": host}, map[string]string{"tls": "enable", "ca": "caaa", "cert": "ceert", "key": "keey"}, false, true},
	// success, TLS cert/key and assumed public CA
	{map[string]string{"queueName": "sample", "hostFromEnv": host}, map[string]string{"tls": "enable", "cert": "ceert", "key": "keey"}, false, true},
	// success, TLS cert/key + key password and assumed public CA
	{map[string]string{"queueName": "sample", "hostFromEnv": host}, map[string]string{"tls": "enable", "cert": "ceert", "key": "keey", "keyPassword": "keeyPassword"}, false, true},
	// success, TLS CA only
	{map[string]string{"queueName": "sample", "hostFromEnv": host}, map[string]string{"tls": "enable", "ca": "caaa"}, false, true},
	// failure, TLS missing cert
	{map[string]string{"queueName": "sample", "hostFromEnv": host}, map[string]string{"tls": "enable", "ca": "caaa", "key": "kee"}, true, true},
	// failure, TLS missing key
	{map[string]string{"queueName": "sample", "hostFromEnv": host}, map[string]string{"tls": "enable", "ca": "caaa", "cert": "ceert"}, true, true},
	// failure, TLS invalid
	{map[string]string{"queueName": "sample", "hostFromEnv": host}, map[string]string{"tls": "yes", "ca": "caaa", "cert": "ceert", "key": "kee"}, true, true},
}
var rabbitMQMetricIdentifiers = []rabbitMQMetricIdentifier{
	{&testRabbitMQMetadata[1], 0, "s0-rabbitmq-sample"},
	{&testRabbitMQMetadata[7], 1, "s1-rabbitmq-namespace-2Fname"},
	{&testRabbitMQMetadata[31], 2, "s2-rabbitmq-host1-sample"},
}

func TestRabbitMQParseMetadata(t *testing.T) {
	for idx, testData := range testRabbitMQMetadata {
		meta, err := parseRabbitMQMetadata(&ScalerConfig{ResolvedEnv: sampleRabbitMqResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error but got success in test case %d", idx)
		}
		if val, ok := testData.metadata["unsafeSsl"]; ok && err == nil {
			boolVal, err := strconv.ParseBool(val)
			if err != nil && !testData.isError {
				t.Errorf("Expect error but got success in test case %d", idx)
			}
			if boolVal != meta.unsafeSsl {
				t.Errorf("Expect %t but got %t in test case %d", boolVal, meta.unsafeSsl, idx)
			}
		}
	}
}

func TestRabbitMQParseAuthParamData(t *testing.T) {
	for _, testData := range testRabbitMQAuthParamData {
		metadata, err := parseRabbitMQMetadata(&ScalerConfig{ResolvedEnv: sampleRabbitMqResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
		if metadata != nil && metadata.enableTLS != testData.enableTLS {
			t.Errorf("Expected enableTLS to be set to %v but got %v\n", testData.enableTLS, metadata.enableTLS)
		}
		if metadata != nil && metadata.enableTLS {
			if metadata.ca != testData.authParams["ca"] {
				t.Errorf("Expected ca to be set to %v but got %v\n", testData.authParams["ca"], metadata.enableTLS)
			}
			if metadata.cert != testData.authParams["cert"] {
				t.Errorf("Expected cert to be set to %v but got %v\n", testData.authParams["cert"], metadata.cert)
			}
			if metadata.key != testData.authParams["key"] {
				t.Errorf("Expected key to be set to %v but got %v\n", testData.authParams["key"], metadata.key)
			}
			if metadata.keyPassword != testData.authParams["keyPassword"] {
				t.Errorf("Expected key to be set to %v but got %v\n", testData.authParams["keyPassword"], metadata.key)
			}
		}
	}
}

var testDefaultQueueLength = []parseRabbitMQMetadataTestData{
	// use default queueLength
	{map[string]string{"queueName": "sample", "hostFromEnv": host}, false, map[string]string{}},
	// use default queueLength with includeUnacked
	{map[string]string{"queueName": "sample", "hostFromEnv": host, "protocol": "http"}, false, map[string]string{}},
}

func TestParseDefaultQueueLength(t *testing.T) {
	for _, testData := range testDefaultQueueLength {
		metadata, err := parseRabbitMQMetadata(&ScalerConfig{ResolvedEnv: sampleRabbitMqResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		switch {
		case err != nil && !testData.isError:
			t.Error("Expected success but got error", err)
		case testData.isError && err == nil:
			t.Error("Expected error but got success")
		case metadata.value != defaultRabbitMQQueueLength:
			t.Error("Expected default queueLength =", defaultRabbitMQQueueLength, "but got", metadata.value)
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
	{`{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"queueLength": "10"}, ""},
	{`{"messages": 1, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"queueLength": "10"}, ""},
	{`{"messages": 1, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"queueLength": "10"}, ""},
	{`{"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"}`, http.StatusOK, false, map[string]string{"queueLength": "10"}, ""},
	{`{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 1.4}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"queueLength": "10"}, ""},
	{`{"messages": 1, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 1.4}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"queueLength": "10"}, ""},
	{`{"messages": 1, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 1.4}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"queueLength": "10"}, ""},
	{`{"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 1.4}}, "name": "evaluate_trials"}`, http.StatusOK, false, map[string]string{"queueLength": "10"}, ""},
	// mode QueueLength
	{`{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"value": "100", "mode": "QueueLength"}, ""},
	{`{"messages": 1, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"value": "100", "mode": "QueueLength"}, ""},
	{`{"messages": 1, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"value": "100", "mode": "QueueLength"}, ""},
	{`{"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"}`, http.StatusOK, false, map[string]string{"value": "100", "mode": "QueueLength"}, ""},
	{`{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 1.4}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"value": "100", "mode": "QueueLength"}, ""},
	{`{"messages": 1, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 1.4}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"value": "100", "mode": "QueueLength"}, ""},
	{`{"messages": 1, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 1.4}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"value": "100", "mode": "QueueLength"}, ""},
	{`{"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 1.4}}, "name": "evaluate_trials"}`, http.StatusOK, false, map[string]string{"value": "100", "mode": "QueueLength"}, ""},
	// mode MessageRate
	{`{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"value": "100", "mode": "MessageRate"}, ""},
	{`{"messages": 1, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"value": "100", "mode": "MessageRate"}, ""},
	{`{"messages": 1, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"value": "100", "mode": "MessageRate"}, ""},
	{`{"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"}`, http.StatusOK, false, map[string]string{"value": "100", "mode": "MessageRate"}, ""},
	{`{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 1.4}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"value": "100", "mode": "MessageRate"}, ""},
	{`{"messages": 1, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 1.4}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"value": "100", "mode": "MessageRate"}, ""},
	{`{"messages": 1, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 1.4}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"value": "100", "mode": "MessageRate"}, ""},
	{`{"messages": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 1.4}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"value": "100", "mode": "MessageRate"}, ""},
	// error response
	{`Password is incorrect`, http.StatusUnauthorized, false, nil, ""},
}

var vhostPaths = []string{"/myhost", "", "/", "//", rabbitRootVhostPath}

var testQueueInfoTestDataSingleVhost = []getQueueInfoTestData{
	{`{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 1.4}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"hostFromEnv": "plainHost", "vhostName": "myhost"}, "/myhost"},
	{`{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 1.4}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"hostFromEnv": "plainHost", "vhostName": "/"}, "//"},
	{`{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 1.4}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"hostFromEnv": "plainHost", "vhostName": ""}, ""},
	{`{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"hostFromEnv": "plainHost", "vhostName": "myhost"}, "/myhost"},
	{`{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"hostFromEnv": "plainHost", "vhostName": "/"}, rabbitRootVhostPath},
	{`{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"}`, http.StatusOK, true, map[string]string{"hostFromEnv": "plainHost", "vhostName": ""}, "/"},
}

func TestGetQueueInfo(t *testing.T) {
	allTestData := []getQueueInfoTestData{}
	for _, testData := range testQueueInfoTestData {
		for _, vhostPath := range vhostPaths {
			testData := testData
			testData.vhostPath = vhostPath
			allTestData = append(allTestData, testData)
		}
	}
	allTestData = append(allTestData, testQueueInfoTestDataSingleVhost...)

	for _, testData := range allTestData {
		testData := testData

		var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expectedPath := fmt.Sprintf("/api/queues%s/evaluate_trials", getExpectedVhost(testData.vhostPath))
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
		_, active, err := s.GetMetricsAndActivity(ctx, "Metric")

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

var testRegexQueueInfoTestData = []getQueueInfoTestData{
	// sum queue length
	{`{"items":[{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum"}, ""},
	{`{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum"}, ""},
	{`{"items":[{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum"}, ""},
	{`{"items":[{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum"}, ""},
	{`{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, false, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum"}, ""},
	{`{"items":[]}`, http.StatusOK, false, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum"}, ""},
	// sum queue length + ignoreUnacknowledged
	{`{"items":[{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum", "excludeUnacknowledged": "true"}, ""},
	{`{"items":[{"messages": 0, "messages_ready": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum", "excludeUnacknowledged": "true"}, ""},
	{`{"items":[{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_ready": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum", "excludeUnacknowledged": "true"}, ""},
	{`{"items":[{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum"}, ""},
	{`{"items":[{"messages": 0, "messages_ready": 0, "messages_unacknowledged": 0, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_ready": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, false, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum", "excludeUnacknowledged": "true"}, ""},
	{`{"items":[]}`, http.StatusOK, false, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum"}, ""},
	// max queue length
	{`{"items":[{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "max"}, ""},
	{`{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "max"}, ""},
	{`{"items":[{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "max"}, ""},
	{`{"items":[{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "max"}, ""},
	{`{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, false, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "max"}, ""},
	{`{"items":[]}`, http.StatusOK, false, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "max"}, ""},
	// max queue length + excludeUnacknowledged
	{`{"items":[{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "max", "excludeUnacknowledged": "true"}, ""},
	{`{"items":[{"messages": 0, "messages_ready": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "max", "excludeUnacknowledged": "true"}, ""},
	{`{"items":[{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_ready": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "max", "excludeUnacknowledged": "true"}, ""},
	{`{"items":[{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "max"}, ""},
	{`{"items":[{"messages": 0, "messages_ready": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_ready": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, false, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "max", "excludeUnacknowledged": "true"}, ""},
	{`{"items":[]}`, http.StatusOK, false, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "max"}, ""},
	// avg queue length
	{`{"items":[{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "avg"}, ""},
	{`{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "avg"}, ""},
	{`{"items":[{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "avg"}, ""},
	{`{"items":[{"messages": 4, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "avg"}, ""},
	{`{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, false, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "avg"}, ""},
	{`{"items":[]}`, http.StatusOK, false, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "avg"}, ""},
	// avg queue length + excludeUnacknowledged
	{`{"items":[{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "avg", "excludeUnacknowledged": "true"}, ""},
	{`{"items":[{"messages": 0, "messages_ready": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "avg", "excludeUnacknowledged": "true"}, ""},
	{`{"items":[{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_ready": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "avg", "excludeUnacknowledged": "true"}, ""},
	{`{"items":[{"messages": 4, "messages_ready": 3, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "avg"}, ""},
	{`{"items":[{"messages": 0, "messages_ready": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_ready": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, false, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "avg", "excludeUnacknowledged": "true"}, ""},
	{`{"items":[]}`, http.StatusOK, false, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "avg"}, ""},
	// sum message rate
	{`{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "sum"}, ""},
	{`{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "sum"}, ""},
	{`{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "sum"}, ""},
	{`{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "sum"}, ""},
	{`{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, false, map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "sum"}, ""},
	{`{"items":[]}`, http.StatusOK, false, map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "sum"}, ""},
	// max message rate
	{`{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "max"}, ""},
	{`{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "max"}, ""},
	{`{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "max"}, ""},
	{`{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "max"}, ""},
	{`{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, false, map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "max"}, ""},
	{`{"items":[]}`, http.StatusOK, false, map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "max"}, ""},
	// avg message rate
	{`{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "avg"}, ""},
	{`{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "avg"}, ""},
	{`{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "avg"}, ""},
	{`{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 4}}, "name": "evaluate_trial2"}]}`, http.StatusOK, true, map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "avg"}, ""},
	{`{"items":[{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trials"},{"messages": 0, "messages_unacknowledged": 1, "message_stats": {"publish_details": {"rate": 0}}, "name": "evaluate_trial2"}]}`, http.StatusOK, false, map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "avg"}, ""},
	{`{"items":[]}`, http.StatusOK, false, map[string]string{"mode": "MessageRate", "value": "1000", "useRegex": "true", "operation": "avg"}, ""},
}

var vhostPathsForRegex = []string{"", "/test-vh", rabbitRootVhostPath}

func TestGetQueueInfoWithRegex(t *testing.T) {
	allTestData := []getQueueInfoTestData{}
	for _, testData := range testRegexQueueInfoTestData {
		for _, vhostPath := range vhostPathsForRegex {
			testData := testData
			testData.vhostPath = vhostPath
			allTestData = append(allTestData, testData)
		}
	}

	for _, testData := range allTestData {
		var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expectedPath := fmt.Sprintf("/api/queues%s?page=1&use_regex=true&pagination=false&name=%%5Eevaluate_trials%%24&page_size=100", getExpectedVhost(testData.vhostPath))
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
			"queueName":   "^evaluate_trials$",
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
		_, active, err := s.GetMetricsAndActivity(ctx, "Metric")

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

type getRegexPageSizeTestData struct {
	queueInfo getQueueInfoTestData
	pageSize  int
}

var testRegexPageSizeTestData = []getRegexPageSizeTestData{
	{testRegexQueueInfoTestData[0], 100},
	{testRegexQueueInfoTestData[0], 200},
	{testRegexQueueInfoTestData[0], 500},
}

func TestGetPageSizeWithRegex(t *testing.T) {
	allTestData := []getRegexPageSizeTestData{}
	for _, testData := range testRegexPageSizeTestData {
		for _, vhostPath := range vhostPathsForRegex {
			testData := testData
			testData.queueInfo.vhostPath = vhostPath
			allTestData = append(allTestData, testData)
		}
	}

	for _, testData := range allTestData {
		var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expectedPath := fmt.Sprintf("/api/queues%s?page=1&use_regex=true&pagination=false&name=%%5Eevaluate_trials%%24&page_size=%d", getExpectedVhost(testData.queueInfo.vhostPath), testData.pageSize)
			if r.RequestURI != expectedPath {
				t.Error("Expect request path to =", expectedPath, "but it is", r.RequestURI)
			}

			w.WriteHeader(testData.queueInfo.responseStatus)
			_, err := w.Write([]byte(testData.queueInfo.response))
			if err != nil {
				t.Error("Expect request path to =", testData.queueInfo.response, "but it is", err)
			}
		}))

		resolvedEnv := map[string]string{host: fmt.Sprintf("%s%s", apiStub.URL, testData.queueInfo.vhostPath), "plainHost": apiStub.URL}

		metadata := map[string]string{
			"queueName":   "^evaluate_trials$",
			"hostFromEnv": host,
			"protocol":    "http",
			"useRegex":    "true",
			"pageSize":    fmt.Sprint(testData.pageSize),
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
		_, active, err := s.GetMetricsAndActivity(ctx, "Metric")

		if err != nil {
			t.Error("Expect success", err)
		}

		if !active {
			t.Error("Expect to be active")
		}
	}
}

func TestRabbitMQGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range rabbitMQMetricIdentifiers {
		meta, err := parseRabbitMQMetadata(&ScalerConfig{ResolvedEnv: sampleRabbitMqResolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: nil, ScalerIndex: testData.index})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockRabbitMQScaler := rabbitMQScaler{
			metadata:   meta,
			connection: nil,
			channel:    nil,
			httpClient: http.DefaultClient,
		}

		metricSpec := mockRabbitMQScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName, "wanted:", testData.name)
		}
	}
}

type rabbitMQErrorTestData struct {
	err     error
	message string
}

var anonymizeRabbitMQErrorTestData = []rabbitMQErrorTestData{
	{fmt.Errorf("https://user1:password1@domain.com"), "error inspecting rabbitMQ: https://user:password@domain.com"},
	{fmt.Errorf("https://fdasr345_-:password1@domain.com"), "error inspecting rabbitMQ: https://user:password@domain.com"},
	{fmt.Errorf("https://user1:fdasr345_-@domain.com"), "error inspecting rabbitMQ: https://user:password@domain.com"},
	{fmt.Errorf("https://fdakls_dsa:password1@domain.com"), "error inspecting rabbitMQ: https://user:password@domain.com"},
	{fmt.Errorf("fdasr345_-:password1@domain.com"), "error inspecting rabbitMQ: user:password@domain.com"},
	{fmt.Errorf("this user1:password1@domain.com fails"), "error inspecting rabbitMQ: this user:password@domain.com fails"},
	{fmt.Errorf("this https://user1:password1@domain.com fails also"), "error inspecting rabbitMQ: this https://user:password@domain.com fails also"},
	{fmt.Errorf("nothing to replace here"), "error inspecting rabbitMQ: nothing to replace here"},
	{fmt.Errorf("the queue https://user1:fdasr345_-@domain.com/api/virtual is unavailable"), "error inspecting rabbitMQ: the queue https://user:password@domain.com/api/virtual is unavailable"},
}

func TestRabbitMQAnonymizeRabbitMQError(t *testing.T) {
	metadata := map[string]string{
		"queueName":   "evaluate_trials",
		"hostFromEnv": host,
		"protocol":    "http",
	}
	meta, err := parseRabbitMQMetadata(&ScalerConfig{ResolvedEnv: sampleRabbitMqResolvedEnv, TriggerMetadata: metadata, AuthParams: nil})

	if err != nil {
		t.Fatalf("Error parsing metadata (%s)", err)
	}

	s := &rabbitMQScaler{
		metadata:   meta,
		httpClient: nil,
	}
	for _, testData := range anonymizeRabbitMQErrorTestData {
		err := s.anonymizeRabbitMQError(testData.err)
		assert.Equal(t, fmt.Sprint(err), testData.message)
	}
}

type getQueueInfoNavigationTestData struct {
	response string
	isError  bool
}

var testRegexQueueInfoNavigationTestData = []getQueueInfoNavigationTestData{
	// sum queue length
	{`{"items":[], "filtered_count": 250, "page": 1, "page_count": 3}`, true},
	{`{"items":[], "filtered_count": 250, "page": 1, "page_count": 1}`, false},
}

func TestRegexQueueMissingError(t *testing.T) {
	for _, testData := range testRegexQueueInfoNavigationTestData {
		var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expectedPath := "/api/queues/%2F?page=1&use_regex=true&pagination=false&name=evaluate_trials&page_size=100"
			if r.RequestURI != expectedPath {
				t.Error("Expect request path to =", expectedPath, "but it is", r.RequestURI)
			}

			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(testData.response))
			if err != nil {
				t.Error("Expect request path to =", testData.response, "but it is", err)
			}
		}))

		resolvedEnv := map[string]string{host: apiStub.URL, "plainHost": apiStub.URL}

		metadata := map[string]string{
			"queueName":   "evaluate_trials",
			"hostFromEnv": host,
			"protocol":    "http",
			"useRegex":    "true",
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
		_, _, err = s.GetMetricsAndActivity(ctx, "Metric")
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func getExpectedVhost(vhostPath string) string {
	switch vhostPath {
	case "":
		return rabbitRootVhostPath
	case "/":
		return rabbitRootVhostPath
	case "//":
		return rabbitRootVhostPath
	case rabbitRootVhostPath:
		return rabbitRootVhostPath
	default:
		return vhostPath
	}
}
