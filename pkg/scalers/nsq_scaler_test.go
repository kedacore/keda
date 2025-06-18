package scalers

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
	v2 "k8s.io/api/autoscaling/v2"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type nsqMetadataTestData struct {
	metadata                   map[string]string
	numNSQLookupdHTTPAddresses int
	nsqLookupdHTTPAddresses    []string
	topic                      string
	channel                    string
	depthThreshold             int64
	activationDepthThreshold   int64
	useHTTPS                   bool
	unsafeSsl                  bool
	isError                    bool
	description                string
}

type nsqMetricIdentifier struct {
	metadataTestData *nsqMetadataTestData
	triggerIndex     int
	name             string
	metricType       string
}

var parseNSQMetadataTestDataset = []nsqMetadataTestData{
	{
		metadata:                   map[string]string{"nsqLookupdHTTPAddresses": "nsqlookupd-0:4161", "topic": "topic", "channel": "channel"},
		numNSQLookupdHTTPAddresses: 1,
		nsqLookupdHTTPAddresses:    []string{"nsqlookupd-0:4161"},
		topic:                      "topic",
		channel:                    "channel",
		depthThreshold:             10,
		activationDepthThreshold:   0,
		isError:                    false,
		description:                "Success",
	},
	{
		metadata:                   map[string]string{"nsqLookupdHTTPAddresses": "nsqlookupd-0:4161,nsqlookupd-1:4161", "topic": "topic", "channel": "channel"},
		numNSQLookupdHTTPAddresses: 2,
		nsqLookupdHTTPAddresses:    []string{"nsqlookupd-0:4161", "nsqlookupd-1:4161"},
		topic:                      "topic",
		channel:                    "channel",
		depthThreshold:             10,
		activationDepthThreshold:   0,
		isError:                    false,
		description:                "Success, multiple nsqlookupd addresses",
	},
	{
		metadata:                   map[string]string{"nsqLookupdHTTPAddresses": "nsqlookupd-0:4161", "topic": "topic", "channel": "channel", "depthThreshold": "100", "activationDepthThreshold": "1", "useHttps": "true", "unsafeSsl": "true"},
		numNSQLookupdHTTPAddresses: 1,
		nsqLookupdHTTPAddresses:    []string{"nsqlookupd-0:4161"},
		topic:                      "topic",
		channel:                    "channel",
		depthThreshold:             100,
		activationDepthThreshold:   1,
		useHTTPS:                   true,
		unsafeSsl:                  true,
		isError:                    false,
		description:                "Success - setting optional fields",
	},
	{
		metadata:    map[string]string{"topic": "topic", "channel": "channel"},
		isError:     true,
		description: "Error, no nsqlookupd addresses",
	},
	{
		metadata:    map[string]string{"nsqLookupdHTTPAddresses": "nsqlookupd-0:4161", "channel": "channel"},
		isError:     true,
		description: "Error, no topic",
	},
	{
		metadata:    map[string]string{"nsqLookupdHTTPAddresses": "nsqlookupd-0:4161", "topic": "topic"},
		isError:     true,
		description: "Error, no channel",
	},
	{
		metadata:    map[string]string{"nsqLookupdHTTPAddresses": "nsqlookupd-0:4161", "topic": "topic", "channel": "channel", "depthThreshold": "0"},
		isError:     true,
		description: "Error, depthThreshold is <=0",
	},
	{
		metadata:    map[string]string{"nsqLookupdHTTPAddresses": "nsqlookupd-0:4161", "topic": "topic", "channel": "channel", "activationDepthThreshold": "-1"},
		isError:     true,
		description: "Error, activationDepthThreshold is <0",
	},
}

var nsqMetricIdentifiers = []nsqMetricIdentifier{
	{&parseNSQMetadataTestDataset[0], 0, "s0-nsq-topic-channel", "Value"},
	{&parseNSQMetadataTestDataset[0], 1, "s1-nsq-topic-channel", "AverageValue"},
}

// Create mock handlers that return fixed responses
func createMockNSQdHandler(depth int64, statsError bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if statsError {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		response := fmt.Sprintf(`{"topics":[{"topic_name":"topic","channels":[{"channel_name":"channel","depth":%d}]}]}`, depth)
		http.ServeContent(w, r, "", time.Time{}, strings.NewReader(response))
	}
}

func createMockLookupdHandler(hostname, port string, lookupError bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if lookupError {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		response := fmt.Sprintf(`{"producers":[{"broadcast_address":"%s","http_port":%s}]}`, hostname, port)
		http.ServeContent(w, r, "", time.Time{}, strings.NewReader(response))
	}
}

func createMockNSQdDepthHandler(statsError, channelPaused bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if statsError {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		var response string
		if channelPaused {
			response = `{"topics":[{"topic_name":"topic", "depth":250, "channels":[{"channel_name":"channel", "depth":100, "paused":true}]}]}`
		} else {
			response = `{"topics":[{"topic_name":"topic", "depth":250, "channels":[{"channel_name":"channel", "depth":100}]}]}`
		}
		http.ServeContent(w, r, "", time.Time{}, strings.NewReader(response))
	}
}

func createMockLookupdDepthHandler(hostname, port string, lookupError, topicNotExist, producersNotExist bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if lookupError {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		var response string
		switch {
		case topicNotExist:
			response = `{"message": "TOPIC_NOT_FOUND"}`
		case producersNotExist:
			response = `{"producers":[]}`
		default:
			response = fmt.Sprintf(`{"producers":[{"broadcast_address":"%s","http_port":%s}]}`, hostname, port)
		}

		http.ServeContent(w, r, "", time.Time{}, strings.NewReader(response))
	}
}

func createMockServerWithResponse(statusCode int, response string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if statusCode != http.StatusOK {
			http.Error(w, "Internal Server Error", statusCode)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		http.ServeContent(w, r, "", time.Time{}, strings.NewReader(response))
	}
}

func TestNSQParseMetadata(t *testing.T) {
	for _, testData := range parseNSQMetadataTestDataset {
		config := scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata}

		meta, err := parseNSQMetadata(&config)
		if err != nil {
			if testData.isError {
				continue
			}
			t.Error("Expected success, got error", err, testData.description)
		}
		if err == nil && testData.isError {
			t.Error("Expected error, got success", testData.description)
		}

		assert.Equal(t, testData.numNSQLookupdHTTPAddresses, len(meta.NSQLookupdHTTPAddresses), testData.description)
		assert.Equal(t, testData.nsqLookupdHTTPAddresses, meta.NSQLookupdHTTPAddresses, testData.description)
		assert.Equal(t, testData.topic, meta.Topic, testData.description)
		assert.Equal(t, testData.channel, meta.Channel, testData.description)
		assert.Equal(t, testData.depthThreshold, meta.DepthThreshold, testData.description)
		assert.Equal(t, testData.activationDepthThreshold, meta.ActivationDepthThreshold, testData.description)
		assert.Equal(t, testData.useHTTPS, meta.UseHTTPS, testData.description)
		assert.Equal(t, testData.unsafeSsl, meta.UnsafeSSL, testData.description)
	}
}

func TestNSQGetMetricsAndActivity(t *testing.T) {
	type testCase struct {
		lookupError               bool
		statsError                bool
		expectedDepth             int64
		expectedActive            bool
		activationdDepthThreshold int64
	}
	testCases := []testCase{
		{
			lookupError: true,
		},
		{
			statsError: true,
		},
		{
			expectedDepth:  100,
			expectedActive: true,
		},
		{
			expectedDepth:  0,
			expectedActive: false,
		},
		{
			expectedDepth:             9,
			activationdDepthThreshold: 10,
			expectedActive:            false,
		},
	}
	for _, tc := range testCases {
		mockNSQdServer := httptest.NewServer(createMockNSQdHandler(tc.expectedDepth, tc.statsError))
		defer mockNSQdServer.Close()

		parsedNSQdURL, err := url.Parse(mockNSQdServer.URL)
		assert.Nil(t, err)

		mockNSQLookupdServer := httptest.NewServer(createMockLookupdHandler(parsedNSQdURL.Hostname(), parsedNSQdURL.Port(), tc.lookupError))
		defer mockNSQLookupdServer.Close()

		parsedNSQLookupdURL, err := url.Parse(mockNSQLookupdServer.URL)
		assert.Nil(t, err)

		nsqlookupdHost := net.JoinHostPort(parsedNSQLookupdURL.Hostname(), parsedNSQLookupdURL.Port())

		activationThreshold := fmt.Sprintf("%d", tc.activationdDepthThreshold)

		config := scalersconfig.ScalerConfig{TriggerMetadata: map[string]string{
			"nsqLookupdHTTPAddresses":  nsqlookupdHost,
			"topic":                    "topic",
			"channel":                  "channel",
			"activationDepthThreshold": activationThreshold,
		}}
		meta, err := parseNSQMetadata(&config)
		assert.Nil(t, err)

		s := nsqScaler{v2.AverageValueMetricType, meta, http.DefaultClient, "http", logr.Discard()}

		metricName := "s0-nsq-topic-channel"
		metrics, activity, err := s.GetMetricsAndActivity(context.Background(), metricName)

		if err != nil && (tc.lookupError || tc.statsError) {
			assert.Equal(t, 0, len(metrics))
			assert.False(t, activity)
			continue
		}

		assert.Nil(t, err)
		assert.Equal(t, 1, len(metrics))
		assert.Equal(t, metricName, metrics[0].MetricName)
		assert.Equal(t, tc.expectedDepth, metrics[0].Value.Value())
		if tc.expectedActive {
			assert.True(t, activity)
		} else {
			assert.False(t, activity)
		}
	}
}

func TestNSQGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range nsqMetricIdentifiers {
		meta, err := parseNSQMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}

		metricType := v2.MetricTargetType(testData.metricType)
		mockNSQScaler := nsqScaler{metricType, meta, nil, "http", logr.Discard()}

		metricSpec := mockNSQScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		assert.Equal(t, testData.name, metricName)
		assert.Equal(t, 1, len(metricSpec))
		assert.Equal(t, metricType, metricSpec[0].External.Target.Type)
		depthThreshold := meta.DepthThreshold
		if metricType == v2.AverageValueMetricType {
			assert.Equal(t, depthThreshold, metricSpec[0].External.Target.AverageValue.Value())
		} else {
			assert.Equal(t, depthThreshold, metricSpec[0].External.Target.Value.Value())
		}
	}
}

func TestNSQGetTopicChannelDepth(t *testing.T) {
	type testCase struct {
		lookupError       bool
		topicNotExist     bool
		producersNotExist bool
		statsError        bool
		channelPaused     bool
		expectedDepth     int64
		description       string
	}
	testCases := []testCase{
		{
			lookupError: true,
			description: "nsqlookupd call failed",
		},
		{
			topicNotExist: true,
			expectedDepth: 0,
			description:   "Topic does not exist",
		},
		{
			producersNotExist: true,
			expectedDepth:     0,
			description:       "No producers for topic",
		},
		{
			statsError:  true,
			description: "nsqd call failed",
		},
		{
			channelPaused: true,
			expectedDepth: 0,
			description:   "Channel is paused",
		},
		{
			expectedDepth: 100,
			description:   "successfully retrieved depth",
		},
	}

	for _, tc := range testCases {
		mockNSQdServer := httptest.NewServer(createMockNSQdDepthHandler(tc.statsError, tc.channelPaused))
		defer mockNSQdServer.Close()

		parsedNSQdURL, err := url.Parse(mockNSQdServer.URL)
		assert.Nil(t, err)

		mockNSQLookupdServer := httptest.NewServer(createMockLookupdDepthHandler(parsedNSQdURL.Hostname(), parsedNSQdURL.Port(), tc.lookupError, tc.topicNotExist, tc.producersNotExist))
		defer mockNSQLookupdServer.Close()

		parsedNSQLookupdURL, err := url.Parse(mockNSQLookupdServer.URL)
		assert.Nil(t, err)

		nsqLookupdHosts := []string{net.JoinHostPort(parsedNSQLookupdURL.Hostname(), parsedNSQLookupdURL.Port())}

		s := nsqScaler{httpClient: http.DefaultClient, scheme: "http", metadata: nsqMetadata{NSQLookupdHTTPAddresses: nsqLookupdHosts, Topic: "topic", Channel: "channel"}}

		depth, err := s.getTopicChannelDepth(context.Background())

		if err != nil && (tc.lookupError || tc.statsError) {
			continue
		}

		assert.Nil(t, err)
		assert.Equal(t, tc.expectedDepth, depth)
	}
}

func TestNSQGetTopicProducers(t *testing.T) {
	type statusAndResponse struct {
		status   int
		response string
	}
	type testCase struct {
		statusAndResponses []statusAndResponse
		expectedNSQdHosts  []string
		isError            bool
		description        string
	}
	testCases := []testCase{
		{
			statusAndResponses: []statusAndResponse{
				{http.StatusOK, `{"producers":[], "channels":[]}`},
			},
			expectedNSQdHosts: []string{},
			description:       "No producers or channels",
		},
		{
			statusAndResponses: []statusAndResponse{
				{http.StatusOK, `{"producers":[{"broadcast_address":"nsqd-0","http_port":4161}]}`},
			},
			expectedNSQdHosts: []string{"nsqd-0:4161"},
			description:       "Single nsqd host",
		},
		{
			statusAndResponses: []statusAndResponse{
				{http.StatusOK, `{"producers":[{"broadcast_address":"nsqd-0","http_port":4161}]}`},
				{http.StatusOK, `{"producers":[{"broadcast_address":"nsqd-1","http_port":4161}]}`},
				{http.StatusOK, `{"producers":[{"broadcast_address":"nsqd-2","http_port":8161}]}`},
			},
			expectedNSQdHosts: []string{"nsqd-0:4161", "nsqd-1:4161", "nsqd-2:8161"},
			description:       "Multiple nsqd hosts",
		},
		{
			statusAndResponses: []statusAndResponse{
				{http.StatusOK, `{"producers":[{"broadcast_address":"nsqd-0","http_port":4161}]}`},
				{http.StatusOK, `{"producers":[{"broadcast_address":"nsqd-0","http_port":4161}]}`},
			},
			expectedNSQdHosts: []string{"nsqd-0:4161"},
			description:       "De-dupe nsqd hosts",
		},
		{
			statusAndResponses: []statusAndResponse{
				{http.StatusOK, `{"producers":[{"broadcast_address":"nsqd-0","http_port":4161}]}`},
				{http.StatusInternalServerError, ""},
			},
			isError:     true,
			description: "At least one host responded with error",
		},
	}

	for _, tc := range testCases {
		callCount := atomic.NewInt32(0)
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			index := int(callCount.Load())
			callCount.Inc()

			statusResponse := tc.statusAndResponses[index]
			if statusResponse.status != http.StatusOK {
				http.Error(w, "Internal Server Error", statusResponse.status)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			http.ServeContent(w, r, "", time.Time{}, strings.NewReader(statusResponse.response))
		}))
		defer mockServer.Close()

		parsedURL, err := url.Parse(mockServer.URL)
		assert.Nil(t, err)

		var nsqLookupdHosts []string
		nsqLookupdHost := net.JoinHostPort(parsedURL.Hostname(), parsedURL.Port())
		for i := 0; i < len(tc.statusAndResponses); i++ {
			nsqLookupdHosts = append(nsqLookupdHosts, nsqLookupdHost)
		}

		s := nsqScaler{httpClient: http.DefaultClient, scheme: "http", metadata: nsqMetadata{NSQLookupdHTTPAddresses: nsqLookupdHosts}}

		nsqdHosts, err := s.getTopicProducers(context.Background(), "topic")

		if err != nil && tc.isError {
			continue
		}

		assert.Nil(t, err)
		assert.ElementsMatch(t, tc.expectedNSQdHosts, nsqdHosts)
	}
}

func TestNSQGetLookup(t *testing.T) {
	type testCase struct {
		serverStatus   int
		serverResponse string
		isError        bool
		description    string
	}
	testCases := []testCase{
		{
			serverStatus:   http.StatusNotFound,
			serverResponse: `{"message": "TOPIC_NOT_FOUND"}`,
			isError:        false,
			description:    "Topic does not exist",
		},
		{
			serverStatus:   http.StatusOK,
			serverResponse: `{"producers":[{"broadcast_address":"nsqd-0","http_port":4151}], "channels":[]}`,
			isError:        false,
			description:    "Channel does not exist",
		},
		{
			serverStatus:   http.StatusNotFound,
			serverResponse: `{"producers":[], "channels":["channel"]}`,
			isError:        false,
			description:    "No nsqd producers exist",
		},
		{
			serverStatus:   http.StatusOK,
			serverResponse: `{"producers":[{"broadcast_address":"nsqd-0", "http_port":4151}], "channels":["channel"]}`,
			isError:        false,
			description:    "Topic and channel exist with nsqd producers",
		},
		{
			serverStatus: http.StatusInternalServerError,
			isError:      true,
			description:  "Host responds with error",
		},
	}

	s := nsqScaler{httpClient: http.DefaultClient, scheme: "http"}
	for _, tc := range testCases {
		mockServer := httptest.NewServer(createMockServerWithResponse(tc.serverStatus, tc.serverResponse))
		defer mockServer.Close()

		parsedURL, err := url.Parse(mockServer.URL)
		assert.Nil(t, err)

		host := net.JoinHostPort(parsedURL.Hostname(), parsedURL.Port())

		resp, err := s.getLookup(context.Background(), host, "topic")

		if err != nil && tc.isError {
			continue
		}

		assert.Nil(t, err, tc.description)

		if tc.serverStatus != http.StatusNotFound {
			assert.NotNil(t, resp, tc.description)
		} else {
			assert.Nil(t, resp, tc.description)
		}
	}
}

func TestNSQAggregateDepth(t *testing.T) {
	type statusAndResponse struct {
		status   int
		response string
	}
	type testCase struct {
		statusAndResponses []statusAndResponse
		expectedDepth      int64
		isError            bool
		description        string
	}
	testCases := []testCase{
		{
			statusAndResponses: []statusAndResponse{
				{http.StatusOK, `{"topics":null}`},
			},
			expectedDepth: 0,
			isError:       false,
			description:   "Topic does not exist",
		},
		{
			statusAndResponses: []statusAndResponse{
				{http.StatusOK, `{"topics":[{"topic_name":"topic", "depth":250, "channels":[]}]}`},
			},
			expectedDepth: 250,
			isError:       false,
			description:   "Topic exists with no channels",
		},
		{
			statusAndResponses: []statusAndResponse{
				{http.StatusOK, `{"topics":[{"topic_name":"topic", "depth":250, "channels":[{"channel_name":"other_channel", "depth":100}]}]}`},
			},
			expectedDepth: 250,
			isError:       false,
			description:   "Topic exists with different channels",
		},
		{
			statusAndResponses: []statusAndResponse{
				{http.StatusOK, `{"topics":[{"topic_name":"topic", "depth":250, "channels":[{"channel_name":"channel", "depth":100}]}]}`},
			},
			expectedDepth: 100,
			isError:       false,
			description:   "Topic and channel exist",
		},
		{
			statusAndResponses: []statusAndResponse{
				{http.StatusOK, `{"topics":[{"topic_name":"topic", "depth":250, "channels":[{"channel_name":"channel", "depth":100, "paused":true}]}]}`},
			},
			expectedDepth: 0,
			isError:       false,
			description:   "Channel is paused",
		},
		{
			statusAndResponses: []statusAndResponse{
				{http.StatusOK, `{"topics":[{"topic_name":"topic", "depth":250, "channels":[{"channel_name":"channel", "depth":100}]}]}`},
				{http.StatusOK, `{"topics":[{"topic_name":"topic", "depth":250, "channels":[{"channel_name":"channel", "depth":50}]}]}`},
			},
			expectedDepth: 150,
			isError:       false,
			description:   "Sum multiple depth values",
		},
		{
			statusAndResponses: []statusAndResponse{
				{http.StatusOK, `{"topics":[{"topic_name":"topic", "depth":500, "channels":[]}]}`},
				{http.StatusOK, `{"topics":[{"topic_name":"topic", "depth":400, "channels":[{"channel_name":"other_channel", "depth":300}]}]}`},
				{http.StatusOK, `{"topics":[{"topic_name":"topic", "depth":200, "channels":[{"channel_name":"channel", "depth":100}]}]}`},
			},
			expectedDepth: 1000,
			isError:       false,
			description:   "Channel doesn't exist on all nsqd hosts",
		},
		{
			statusAndResponses: []statusAndResponse{
				{http.StatusOK, `{"topics":[{"topic_name":"topic", "depth":250, "channels":[{"channel_name":"channel", "depth":100}]}]}`},
				{http.StatusInternalServerError, ""},
			},
			expectedDepth: -1,
			isError:       true,
			description:   "At least one host responded with error",
		},
	}

	s := nsqScaler{httpClient: http.DefaultClient, scheme: "http"}
	for _, tc := range testCases {
		callCount := atomic.NewInt32(0)
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			index := int(callCount.Load())
			callCount.Inc()

			statusResponse := tc.statusAndResponses[index]
			if statusResponse.status != http.StatusOK {
				http.Error(w, "Internal Server Error", statusResponse.status)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			http.ServeContent(w, r, "", time.Time{}, strings.NewReader(statusResponse.response))
		}))
		defer mockServer.Close()

		parsedURL, err := url.Parse(mockServer.URL)
		assert.Nil(t, err)

		var nsqdHosts []string
		nsqdHost := net.JoinHostPort(parsedURL.Hostname(), parsedURL.Port())
		for i := 0; i < len(tc.statusAndResponses); i++ {
			nsqdHosts = append(nsqdHosts, nsqdHost)
		}

		depth, err := s.aggregateDepth(context.Background(), nsqdHosts, "topic", "channel")

		if err != nil && tc.isError {
			continue
		}

		assert.Nil(t, err, tc.description)
		assert.Equal(t, tc.expectedDepth, depth, tc.description)
	}
}

func TestNSQGetStats(t *testing.T) {
	type testCase struct {
		serverStatus   int
		serverResponse string
		isError        bool
		description    string
	}
	testCases := []testCase{
		{
			serverStatus:   http.StatusOK,
			serverResponse: `{"topics":null}`,
			isError:        false,
			description:    "Topic does not exist",
		},
		{
			serverStatus:   http.StatusOK,
			serverResponse: `{"topics":[{"topic_name":"topic", "depth":250, "channels":[]}]}`,
			isError:        false,
			description:    "Channel does not exist",
		},
		{
			serverStatus:   http.StatusOK,
			serverResponse: `{"topics":[{"topic_name":"topic", "depth":250, "channels":[{"channel_name":"channel", "depth":250}]}]}`,
			isError:        false,
			description:    "Topic and channel exist",
		},
		{
			serverStatus: http.StatusInternalServerError,
			isError:      true,
			description:  "Host responds with error",
		},
	}

	s := nsqScaler{httpClient: http.DefaultClient, scheme: "http"}
	for _, tc := range testCases {
		mockServer := httptest.NewServer(createMockServerWithResponse(tc.serverStatus, tc.serverResponse))
		defer mockServer.Close()

		parsedURL, err := url.Parse(mockServer.URL)
		assert.Nil(t, err)

		host := net.JoinHostPort(parsedURL.Hostname(), parsedURL.Port())
		resp, err := s.getStats(context.Background(), host, "topic")

		if err != nil && tc.isError {
			continue
		}

		assert.Nil(t, err, tc.description)
		assert.NotNil(t, resp, tc.description)
	}
}
