package scalers

import (
	"context"
	"encoding/json"
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
	description    string
	response       string
	responseStatus int
	isActive       bool
	extraMetadata  map[string]string
	vhostPath      string
}

func (r queueInfo) toString() string {
	b, _ := json.Marshal(r)
	return string(b)
}

func (r regexQueueInfo) toString() string {
	b, _ := json.Marshal(r)
	return string(b)
}

func mockResponse(name string, total int, ready int, unacked int, rate float64) queueInfo {
	return queueInfo{
		Name:                   name,
		Messages:               total,
		MessagesReady:          ready,
		MessagesUnacknowledged: unacked,
		MessageStat: messageStat{
			publishDetail{
				Rate: rate,
			},
		},
	}
}

func evaluateTrialsMockResponse(total int, ready int, unacked int, rate float64) queueInfo {
	return mockResponse("evaluate_trials", total, ready, unacked, rate)
}

func evaluateTrials2MockResponse(total int, ready int, unacked int, rate float64) queueInfo {
	return mockResponse("evaluate_trials2", total, ready, unacked, rate)
}

var testQueueInfoTestData = []getQueueInfoTestData{
	// queueLength
	{
		description:    "Legacy queueLength: 10 - 10R messages - expect active",
		response:       evaluateTrialsMockResponse(10, 10, 0, 0).toString(),
		responseStatus: http.StatusOK,
		isActive:       true,
		extraMetadata:  map[string]string{"queueLength": "10"},
		vhostPath:      "",
	},
	{
		description:    "Legacy queueLength: 10 - 10R/0U messages - expect no active",
		response:       evaluateTrialsMockResponse(10, 10, 0, 0).toString(),
		responseStatus: http.StatusOK,
		isActive:       false,
		extraMetadata:  map[string]string{"queueLength": "10", "activationValue": "10"},
		vhostPath:      "",
	},
	// mode QueueLength
	{
		description:    "QueueLength: 10 - 10R/0U messages - expect no active",
		response:       evaluateTrialsMockResponse(10, 10, 0, 0).toString(),
		responseStatus: http.StatusOK,
		isActive:       false,
		extraMetadata:  map[string]string{"value": "10", "mode": "QueueLength", "activationValue": "10"},
		vhostPath:      "",
	},
	{
		description:    "QueueLength: 11 - 11R/0U messages - expect active + trigger",
		response:       evaluateTrialsMockResponse(11, 11, 0, 0).toString(),
		responseStatus: http.StatusOK,
		isActive:       true,
		extraMetadata:  map[string]string{"value": "10", "mode": "QueueLength", "activationValue": "10"},
		vhostPath:      "",
	},
	// mode MessageRate
	{
		description:    "QueueLength: 10 - 10R/0U messages - rate 1.5 - expect no active",
		response:       evaluateTrialsMockResponse(10, 10, 0, 1.5).toString(),
		responseStatus: http.StatusOK,
		isActive:       false,
		extraMetadata:  map[string]string{"value": "1.2", "mode": "MessageRate", "activationValue": "2"},
		vhostPath:      "",
	},
	{
		description:    "QueueLength: 11 - 11R/0U messages - rate 2.5 - expect active",
		response:       evaluateTrialsMockResponse(11, 11, 0, 2.5).toString(),
		responseStatus: http.StatusOK,
		isActive:       true,
		extraMetadata:  map[string]string{"value": "1.2", "mode": "MessageRate", "activationValue": "2"},
		vhostPath:      "",
	},
	{
		description: "avg rate (5) with activationValue 4 should be active",
		response: queueInfo{
			Name:                   "evaluate_trials",
			Messages:               0,
			MessagesReady:          0,
			MessagesUnacknowledged: 0,
			MessageStat: messageStat{
				publishDetail{
					Rate:        0,
					AverageRate: 5,
					Average:     0,
				},
			},
		}.toString(),
		responseStatus: http.StatusOK,
		isActive:       true,
		extraMetadata: map[string]string{
			"mode":                  "MessageRate",
			"messageRatesAge":       "5",
			"messageRatesIncrement": "1",
			"activationValue":       "4",
			"value":                 "99",
		},
		vhostPath: "",
	},
	{
		description: "avg rate (5) with activationValue 5 should not be active",
		response: queueInfo{
			Name:                   "evaluate_trials",
			Messages:               0,
			MessagesReady:          0,
			MessagesUnacknowledged: 0,
			MessageStat: messageStat{
				publishDetail{
					Rate:        0,
					AverageRate: 5,
					Average:     0,
				},
			},
		}.toString(),
		responseStatus: http.StatusOK,
		isActive:       false,
		extraMetadata: map[string]string{
			"mode":                  "MessageRate",
			"messageRatesAge":       "5",
			"messageRatesIncrement": "1",
			"activationValue":       "5",
			"value":                 "99",
		},
		vhostPath: "",
	},
	{"Authentication issue", `Password is incorrect`, http.StatusUnauthorized, false, nil, ""},
}

var vhostPaths = []string{"/myhost", "", "/", "//", rabbitRootVhostPath}

var testQueueInfoTestDataSingleVhost = []getQueueInfoTestData{
	{"vhostName: myhost - vhostPath: /myhost", evaluateTrialsMockResponse(1, 1, 0, 0).toString(), http.StatusOK, true,
		map[string]string{"hostFromEnv": "plainHost", "vhostName": "myhost"}, "/myhost"},
	{"vhostName: / - vhostPath: //", evaluateTrialsMockResponse(1, 1, 0, 0).toString(), http.StatusOK, true,
		map[string]string{"hostFromEnv": "plainHost", "vhostName": "/"}, "//"},
	{"vhostName: \"\" - vhostPath: \"\"", evaluateTrialsMockResponse(1, 1, 0, 0).toString(), http.StatusOK, true,
		map[string]string{"hostFromEnv": "plainHost", "vhostName": ""}, ""},
	{"vhostName: myhost - vhostPath: /myhost", evaluateTrialsMockResponse(1, 1, 0, 0).toString(), http.StatusOK, true,
		map[string]string{"hostFromEnv": "plainHost", "vhostName": "myhost"}, "/myhost"},
	{"vhostName: / - vhostPath: " + rabbitRootVhostPath, evaluateTrialsMockResponse(1, 1, 0, 0).toString(), http.StatusOK, true,
		map[string]string{"hostFromEnv": "plainHost", "vhostName": "/"}, rabbitRootVhostPath},
	{"vhostName: \"\" - vhostPath: /", evaluateTrialsMockResponse(1, 1, 0, 0).toString(), http.StatusOK, true,
		map[string]string{"hostFromEnv": "plainHost", "vhostName": ""}, "/"},
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
			if r.URL.Path != expectedPath {
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
					t.Errorf("Expect to be active: %v", testData.description)
				} else {
					t.Errorf("Expect to not be active: %v", testData.description)
				}
			}
		} else if !strings.Contains(err.Error(), testData.response) {
			t.Error("Expect error to be like '", testData.response, "' but it's '", err, "'")
		}
	}
}

var _testRegexQueueInfoTestDataLegacy = []getQueueInfoTestData{{
	"Legacy queueLength sum (0,1) + (4,1) with activationValue 6 should not be active",
	regexQueueInfo{
		Queues: []queueInfo{
			evaluateTrialsMockResponse(1, 0, 1, 0),
			evaluateTrials2MockResponse(5, 4, 1, 0),
		},
	}.toString(),
	http.StatusOK, false, map[string]string{"queueLength": "10", "useRegex": "true", "operation": "sum", "activationValue": "6"}, "",
}}

// sum queue length
var _testRegexQueueInfoTestDataSum = []getQueueInfoTestData{
	{
		"sum (0,1) + (4,1) with activationValue 6 should not be active",
		regexQueueInfo{
			Queues: []queueInfo{
				evaluateTrialsMockResponse(1, 0, 1, 0),
				evaluateTrials2MockResponse(5, 4, 1, 0),
			},
		}.toString(),
		http.StatusOK, false, map[string]string{"mode": "QueueLength", "value": "10", "useRegex": "true", "operation": "sum", "activationValue": "6"}, "",
	},
	{
		"sum (0,1) + (4,1) should be active",
		regexQueueInfo{
			Queues: []queueInfo{
				evaluateTrialsMockResponse(1, 0, 1, 0),
				evaluateTrials2MockResponse(5, 4, 1, 0),
			},
		}.toString(),
		http.StatusOK, true, map[string]string{"mode": "QueueLength", "value": "10", "useRegex": "true", "operation": "sum"}, "",
	},
	{
		"sum (4,1) + (0,1) should be active",
		regexQueueInfo{
			Queues: []queueInfo{
				evaluateTrialsMockResponse(5, 4, 1, 0),
				evaluateTrials2MockResponse(1, 0, 1, 0),
			},
		}.toString(),
		http.StatusOK, true, map[string]string{"mode": "QueueLength", "value": "10", "useRegex": "true", "operation": "sum"}, "",
	},
	{
		"sum (4,1) should  be active",
		regexQueueInfo{
			Queues: []queueInfo{evaluateTrials2MockResponse(5, 4, 1, 0)},
		}.toString(),
		http.StatusOK, true, map[string]string{"mode": "QueueLength", "value": "10", "useRegex": "true", "operation": "sum"}, "",
	},
	// sum queue length + ignoreUnacknowledged
	{
		"sum (0,1) + (4,1) with activationValue 5 and excludeUnacknowledged should not be active",
		regexQueueInfo{
			Queues: []queueInfo{
				evaluateTrialsMockResponse(1, 0, 1, 0),
				evaluateTrials2MockResponse(5, 4, 1, 0),
			},
		}.toString(),
		http.StatusOK, false, map[string]string{
			"queueLength":           "10",
			"useRegex":              "true",
			"operation":             "sum",
			"excludeUnacknowledged": "true",
			"activationValue":       "5",
		},
		"",
	},
	{
		"sum (0,1) + (4,1) with activationValue 5 should be active",
		regexQueueInfo{
			Queues: []queueInfo{
				evaluateTrialsMockResponse(1, 0, 1, 0),
				evaluateTrials2MockResponse(5, 4, 1, 0),
			},
		}.toString(),
		http.StatusOK, true, map[string]string{
			"queueLength":           "10",
			"useRegex":              "true",
			"operation":             "sum",
			"excludeUnacknowledged": "false",
			// "activationValue": "5",
			"activationValue": "4",
		},
		"",
	},
}

var _testRegexQueueInfoTestDataMax = []getQueueInfoTestData{
	// max queue length
	{
		"max (0,1), (4,1) with activationValue 3 should be active",
		regexQueueInfo{
			Queues: []queueInfo{
				evaluateTrialsMockResponse(1, 0, 1, 0),
				evaluateTrials2MockResponse(5, 4, 1, 0),
			},
		}.toString(),
		http.StatusOK, true, map[string]string{
			"queueLength":     "4",
			"useRegex":        "true",
			"operation":       "max",
			"activationValue": "3",
		},
		"",
	},
	{
		"max (0,1), (4,1) with activationValue 4 should be active",
		regexQueueInfo{
			Queues: []queueInfo{
				evaluateTrialsMockResponse(1, 0, 1, 0),
				evaluateTrials2MockResponse(5, 4, 1, 0),
			},
		}.toString(),
		http.StatusOK, true, map[string]string{
			"queueLength":     "4",
			"useRegex":        "true",
			"operation":       "max",
			"activationValue": "4",
		},
		"",
	},
	// max queue length + ignore unacknowledged
	{
		"max (0,1), (4,1) with activationValue 5 and exclude unack should not be active",
		regexQueueInfo{
			Queues: []queueInfo{
				evaluateTrialsMockResponse(1, 0, 1, 0),
				evaluateTrials2MockResponse(5, 4, 1, 0),
			},
		}.toString(),
		http.StatusOK, false, map[string]string{
			"queueLength":           "4",
			"useRegex":              "true",
			"operation":             "max",
			"activationValue":       "5",
			"excludeUnacknowledged": "true",
		},
		"",
	},
	{
		"max () should not be active",
		regexQueueInfo{
			Queues: []queueInfo{},
		}.toString(),
		http.StatusOK, false, map[string]string{
			"queueLength":           "4",
			"useRegex":              "true",
			"operation":             "max",
			"excludeUnacknowledged": "true",
		},
		"",
	},
}

var _testRegexQueueInfoTestDataAvg = []getQueueInfoTestData{
	// avg queue length
	{
		"avg (0,1), (4,1) with activationValue 3 and should not be active",
		regexQueueInfo{
			Queues: []queueInfo{
				evaluateTrialsMockResponse(1, 0, 1, 0),
				evaluateTrials2MockResponse(5, 4, 1, 0),
			},
		}.toString(),
		http.StatusOK, false, map[string]string{
			"queueLength":           "4",
			"useRegex":              "true",
			"operation":             "avg",
			"activationValue":       "3",
			"excludeUnacknowledged": "true",
		},
		"",
	},
	{
		"avg (2,1), (4,1) with activationValue 2, excluding unack, should be active",
		regexQueueInfo{
			Queues: []queueInfo{
				evaluateTrialsMockResponse(3, 2, 1, 0),
				evaluateTrials2MockResponse(5, 4, 1, 0),
			},
		}.toString(),
		http.StatusOK, true, map[string]string{
			"queueLength":           "4",
			"useRegex":              "true",
			"operation":             "avg",
			"activationValue":       "2",
			"excludeUnacknowledged": "true",
		},
		"",
	},
	{
		"avg () should not be active",
		regexQueueInfo{
			Queues: []queueInfo{},
		}.toString(),
		http.StatusOK, false, map[string]string{
			"mode": "QueueLength", "value": "4",
			"useRegex":  "true",
			"operation": "avg",
		},
		"",
	},
	// avg queue length + ignore unacknowledged
	{
		"avg (0,1), (4,1) with activationValue 2 and should not be active",
		regexQueueInfo{
			Queues: []queueInfo{
				evaluateTrialsMockResponse(4, 0, 1, 0),
				evaluateTrials2MockResponse(4, 4, 1, 0),
			},
		}.toString(),
		http.StatusOK, false, map[string]string{
			"queueLength":           "4",
			"useRegex":              "true",
			"operation":             "avg",
			"activationValue":       "2",
			"excludeUnacknowledged": "true",
		},
		"",
	},
}

var _testRegexQueueInfoTestDataSumRate = []getQueueInfoTestData{
	{
		"sum (1.2) + (1.8) with activationValue 6 should not be active",
		regexQueueInfo{
			Queues: []queueInfo{
				evaluateTrialsMockResponse(0, 0, 0, 1.2),
				evaluateTrials2MockResponse(0, 0, 0, 1.8),
			},
		}.toString(),
		http.StatusOK, false, map[string]string{"mode": "MessageRate", "value": "10", "useRegex": "true", "operation": "sum", "activationValue": "3"}, "",
	},
	{
		"sum (1.2) + (1.8) should be active",
		regexQueueInfo{
			Queues: []queueInfo{
				evaluateTrialsMockResponse(0, 0, 0, 1.2),
				evaluateTrials2MockResponse(0, 0, 0, 1.8),
			},
		}.toString(),
		http.StatusOK, true, map[string]string{"mode": "MessageRate", "value": "10", "useRegex": "true", "operation": "sum", "activationValue": "2.9"}, "",
	},
	{
		"sum (1.8) + (1.2) with unset activationValue should be active",
		regexQueueInfo{
			Queues: []queueInfo{
				evaluateTrialsMockResponse(0, 0, 0, 1.8),
				evaluateTrials2MockResponse(0, 0, 0, 1.2),
			},
		}.toString(),
		http.StatusOK, true, map[string]string{"mode": "MessageRate", "value": "10", "useRegex": "true", "operation": "sum"}, "",
	},
	{
		"sum (3) should  be active",
		regexQueueInfo{
			Queues: []queueInfo{evaluateTrials2MockResponse(0, 0, 0, 3)},
		}.toString(),
		http.StatusOK, true, map[string]string{"mode": "MessageRate", "value": "2.9", "useRegex": "true", "operation": "sum"}, "",
	},
	{
		"sum (1.2) + (1.8) with activationValue 5 should not be active",
		regexQueueInfo{
			Queues: []queueInfo{
				evaluateTrialsMockResponse(0, 0, 0, 1.2),
				evaluateTrials2MockResponse(0, 0, 0, 1.8),
			},
		}.toString(),
		http.StatusOK, false, map[string]string{
			"mode":                  "MessageRate",
			"useRegex":              "true",
			"operation":             "sum",
			"excludeUnacknowledged": "true",
			"activationValue":       "5",
			"value":                 "999",
		},
		"",
	},
}

var _testRegexQueueInfoTestDataMaxRate = []getQueueInfoTestData{
	{
		"max (1.2), (1.8) with activationValue 1.7 should be active",
		regexQueueInfo{
			Queues: []queueInfo{
				evaluateTrialsMockResponse(0, 0, 0, 1.2),
				evaluateTrials2MockResponse(0, 0, 0, 1.8),
			},
		}.toString(),
		http.StatusOK, true, map[string]string{
			"mode":            "MessageRate",
			"useRegex":        "true",
			"operation":       "max",
			"activationValue": "1.7",
			"value":           "99",
		},
		"",
	},
	{
		"max (1.2), (1.8) with activationValue 3.1 should not be active",
		regexQueueInfo{
			Queues: []queueInfo{
				evaluateTrialsMockResponse(0, 0, 0, 1.2),
				evaluateTrials2MockResponse(0, 0, 0, 1.8),
			},
		}.toString(),
		http.StatusOK, false, map[string]string{
			"mode":            "MessageRate",
			"useRegex":        "true",
			"operation":       "max",
			"activationValue": "3.1",
			"value":           "99",
		},
		"",
	},
	{
		"max () should not be active",
		regexQueueInfo{
			Queues: []queueInfo{},
		}.toString(),
		http.StatusOK, false, map[string]string{
			"mode":                  "MessageRate",
			"useRegex":              "true",
			"operation":             "max",
			"excludeUnacknowledged": "true",
			"value":                 "99",
		},
		"",
	},
}

var _testRegexQueueInfoTestDataAvgRate = []getQueueInfoTestData{
	{
		"avg (1.2), (1.8) with activationValue 1.5 and should not be active",
		regexQueueInfo{
			Queues: []queueInfo{
				evaluateTrialsMockResponse(0, 0, 0, 1.2),
				evaluateTrials2MockResponse(0, 0, 0, 1.8),
			},
		}.toString(),
		http.StatusOK, false, map[string]string{
			"mode":                  "MessageRate",
			"useRegex":              "true",
			"operation":             "avg",
			"activationValue":       "1.5",
			"excludeUnacknowledged": "true",
			"value":                 "99",
		},
		"",
	},
	{
		"avg (1.2), (1.8) with activationValue 1 and should be active",
		regexQueueInfo{
			Queues: []queueInfo{
				evaluateTrialsMockResponse(0, 0, 0, 1.2),
				evaluateTrials2MockResponse(0, 0, 0, 1.8),
			},
		}.toString(),
		http.StatusOK, true, map[string]string{
			"mode":                  "MessageRate",
			"useRegex":              "true",
			"operation":             "avg",
			"activationValue":       "1",
			"excludeUnacknowledged": "true",
			"value":                 "99",
		},
		"",
	},
	{
		"avg () should not be active",
		regexQueueInfo{
			Queues: []queueInfo{},
		}.toString(),
		http.StatusOK, false, map[string]string{
			"mode":      "MessageRate",
			"value":     "4",
			"useRegex":  "true",
			"operation": "avg",
		},
		"",
	},
}

var _testQueueInfoTestAvgRate = []getQueueInfoTestData{
	{
		description: "avg rate (10) with activationValue 11 should not be active",
		response: regexQueueInfo{
			Queues: []queueInfo{
				{
					Name:                   "evaluate_trials",
					Messages:               0,
					MessagesReady:          0,
					MessagesUnacknowledged: 0,
					MessageStat: messageStat{
						publishDetail{
							Rate:        0,
							AverageRate: 10,
							Average:     0,
						},
					},
				},
				{
					Name: "evaluate_trials",
					MessageStat: messageStat{
						publishDetail{
							AverageRate: 10,
						},
					},
				},
			},
		}.toString(),
		responseStatus: http.StatusOK,
		isActive:       true,
		extraMetadata: map[string]string{
			"useRegex":              "true",
			"mode":                  "MessageRate",
			"messageRatesAge":       "5",
			"messageRatesIncrement": "1",
			"activationValue":       "11",
			"value":                 "99",
		},
		vhostPath: "",
	},
	{
		description: "avg rate (10) with activationValue 9 should be active",
		response: regexQueueInfo{
			Queues: []queueInfo{
				{
					Name:                   "evaluate_trials",
					Messages:               0,
					MessagesReady:          0,
					MessagesUnacknowledged: 0,
					MessageStat: messageStat{
						publishDetail{
							Rate:        0,
							AverageRate: 10,
							Average:     0,
						},
					},
				},
				{
					Name: "evaluate_trials",
					MessageStat: messageStat{
						publishDetail{
							AverageRate: 10,
						},
					},
				},
			},
		}.toString(),
		responseStatus: http.StatusOK,
		isActive:       true,
		extraMetadata: map[string]string{
			"useRegex":              "true",
			"mode":                  "MessageRate",
			"messageRatesAge":       "5",
			"messageRatesIncrement": "1",
			"activationValue":       "9",
			"value":                 "99",
		},
		vhostPath: "",
	},
}

func getTestRegexQueueInfoTestData() []getQueueInfoTestData {
	var testRegexQueueInfoTestData = []getQueueInfoTestData{}
	for _, testgroup := range [][]getQueueInfoTestData{
		_testRegexQueueInfoTestDataLegacy,
		_testRegexQueueInfoTestDataSum,
		_testRegexQueueInfoTestDataMax,
		_testRegexQueueInfoTestDataAvg,
		_testRegexQueueInfoTestDataSumRate,
		_testRegexQueueInfoTestDataMaxRate,
		_testRegexQueueInfoTestDataAvgRate,
		_testQueueInfoTestAvgRate,
	} {
		testRegexQueueInfoTestData = append(testRegexQueueInfoTestData, testgroup...)
	}
	return testRegexQueueInfoTestData
}

var vhostPathsForRegex = []string{"", "/test-vh", rabbitRootVhostPath}

func TestGetQueueInfoWithRegex(t *testing.T) {
	allTestData := []getQueueInfoTestData{}
	for _, testData := range getTestRegexQueueInfoTestData() {
		for _, vhostPath := range vhostPathsForRegex {
			testData := testData
			testData.vhostPath = vhostPath
			allTestData = append(allTestData, testData)
		}
	}

	for _, testData := range allTestData {
		var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expectedValues := map[string][]string{
				"name":       {"^evaluate_trials$"},
				"page":       {"1"},
				"page_size":  {"100"},
				"pagination": {"false"},
				"use_regex":  {"true"},
			}

			val, ok := testData.extraMetadata["messageRatesAge"]
			if ok {
				expectedValues["msg_rates_age"] = []string{val}
			}
			val, ok = testData.extraMetadata["messageRatesIncrement"]
			if ok {
				expectedValues["msg_rates_incr"] = []string{val}
			}
			expectedJSON, _ := json.Marshal(expectedValues)
			haveJSON, _ := json.Marshal(r.URL.Query())
			if string(expectedJSON) != string(haveJSON) {
				t.Error("Expect request query to =", expectedValues, "but it is", r.URL.Query())
			}

			expectedPath := fmt.Sprintf("/api/queues%s", getExpectedVhost(testData.vhostPath))
			if r.URL.Path != expectedPath {
				t.Error("Expect request path to =", expectedPath, "but it is", r.URL.Path)
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
			t.Errorf("Expect success for %v", testData.description)
		}

		ctx := context.TODO()
		_, active, err := s.GetMetricsAndActivity(ctx, "Metric")

		if testData.responseStatus == http.StatusOK {
			if err != nil {
				t.Error("Expect success", err)
			}

			if active != testData.isActive {
				if testData.isActive {
					t.Errorf("Expect to be active: %v", testData.description)
				} else {
					t.Errorf("Expect to not be active: %v", testData.description)
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

func TestGetPageSizeWithRegex(t *testing.T) {
	allTestData := []getRegexPageSizeTestData{}
	regexTestData := getTestRegexQueueInfoTestData()

	var testRegexPageSizeTestData = []getRegexPageSizeTestData{
		{regexTestData[0], 100},
		{regexTestData[0], 200},
		{regexTestData[0], 500},
	}
	for _, testData := range testRegexPageSizeTestData {
		for _, vhostPath := range vhostPathsForRegex {
			testData := testData
			testData.queueInfo.vhostPath = vhostPath
			allTestData = append(allTestData, testData)
		}
	}

	for _, testData := range allTestData {
		var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expectedValues := map[string][]string{
				"name":       {"^evaluate_trials$"},
				"page":       {"1"},
				"page_size":  {fmt.Sprint(testData.pageSize)},
				"pagination": {"false"},
				"use_regex":  {"true"},
			}
			expectedJSON, _ := json.Marshal(expectedValues)
			haveJSON, _ := json.Marshal(r.URL.Query())
			if string(expectedJSON) != string(haveJSON) {
				t.Error("Expect request query to =", expectedValues, "but it is", r.URL.Query())
			}

			expectedPath := fmt.Sprintf("/api/queues%s", getExpectedVhost(testData.queueInfo.vhostPath))
			if r.URL.Path != expectedPath {
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
			expectedValues := map[string][]string{
				"name":       {"evaluate_trials"},
				"page":       {"1"},
				"page_size":  {"100"},
				"pagination": {"false"},
				"use_regex":  {"true"},
			}

			expectedJSON, _ := json.Marshal(expectedValues)
			haveJSON, _ := json.Marshal(r.URL.Query())

			if string(expectedJSON) != string(haveJSON) {
				t.Error("Expect request query to =", expectedValues, "but it is", r.URL.Query())
			}

			expectedPath := "/api/queues/%2F"
			if r.URL.Path != expectedPath {
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
