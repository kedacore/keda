package scalers

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseNATSJetStreamMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

type parseNATSJetStreamMockResponsesTestData struct {
	name     string
	metadata *natsJetStreamMetricIdentifier
	data     *jetStreamEndpointResponse
	isActive bool
	isError  bool
}

type natsJetStreamMetricIdentifier struct {
	metadataTestData *parseNATSJetStreamMetadataTestData
	triggerIndex     int
	name             string
}

var testNATSJetStreamMetadata = []parseNATSJetStreamMetadataTestData{
	// All good localhost.
	{map[string]string{"natsServerMonitoringEndpoint": "localhost:8222", "account": "$G", "accountID": "$G", "stream": "mystream", "consumer": "pull_consumer", "useHttps": "false"}, map[string]string{}, false},
	// All good url.
	{map[string]string{"natsServerMonitoringEndpoint": "nats.nats:8222", "account": "$G", "stream": "mystream", "consumer": "pull_consumer", "useHttps": "true"}, map[string]string{}, false},
	// All good uses ID over name
	{map[string]string{"natsServerMonitoringEndpoint": "localhost:8222", "accountID": "$G", "stream": "mystream", "consumer": "pull_consumer", "useHttps": "false"}, map[string]string{}, false},
	// nothing passed
	{map[string]string{}, map[string]string{}, true},
	// Missing account name and ID, should fail
	{map[string]string{"natsServerMonitoringEndpoint": "nats.nats:8222", "stream": "mystream", "consumer": "pull_consumer"}, map[string]string{}, true},
	// Missing stream name, should fail
	{map[string]string{"natsServerMonitoringEndpoint": "nats.nats:8222", "account": "$G", "consumer": "pull_consumer"}, map[string]string{}, true},
	// Missing consumer name should fail
	{map[string]string{"natsServerMonitoringEndpoint": "nats.nats:8222", "account": "$G", "stream": "mystream"}, map[string]string{}, true},
	// Missing nats server monitoring endpoint, should fail
	{map[string]string{"account": "$G", "stream": "mystream"}, map[string]string{}, true},
	// All good + activationLagThreshold
	{map[string]string{"natsServerMonitoringEndpoint": "nats.nats:8222", "account": "$G", "stream": "mystream", "consumer": "pull_consumer", "activationLagThreshold": "10"}, map[string]string{}, false},
	// Misconfigured activationLagThreshold
	{map[string]string{"natsServerMonitoringEndpoint": "nats.nats:8222", "account": "$G", "stream": "mystream", "consumer": "pull_consumer", "activationLagThreshold": "Y"}, map[string]string{}, true},
	// natsServerMonitoringEndpoint is defined in authParams
	{map[string]string{"account": "$G", "stream": "mystream", "consumer": "pull_consumer"}, map[string]string{"natsServerMonitoringEndpoint": "nats.nats:8222"}, false},
	// Missing nats server monitoring endpoint , should fail
	{map[string]string{"account": "$G", "stream": "mystream", "consumer": "pull_consumer"}, map[string]string{"natsServerMonitoringEndpoint": ""}, true},
	// Misconfigured https, should fail
	{map[string]string{"natsServerMonitoringEndpoint": "nats.nats:8222", "account": "$G", "stream": "mystream", "consumer": "pull_consumer", "useHttps": "error"}, map[string]string{}, true},
	// All good + lagThreshold
	{map[string]string{"account": "$G", "stream": "mystream", "consumer": "pull_consumer", jetStreamLagThresholdMetricName: "6"}, map[string]string{"natsServerMonitoringEndpoint": "nats.nats:8222"}, false},
	// Misconfigured lag threshold
	{map[string]string{"account": "$G", "stream": "mystream", "consumer": "pull_consumer", jetStreamLagThresholdMetricName: "Y"}, map[string]string{"natsServerMonitoringEndpoint": "nats.nats:8222"}, true},
	// All good + account from authParams
	{map[string]string{"stream": "mystream", "consumer": "pull_consumer"}, map[string]string{"account": "$G", "natsServerMonitoringEndpoint": "nats.nats:8222"}, false},
	// Misconfigured account
	{map[string]string{"stream": "mystream", "consumer": "pull_consumer"}, map[string]string{"natsServerMonitoringEndpoint": "nats.nats:8222"}, true},
}

var natsJetStreamMetricIdentifiers = []natsJetStreamMetricIdentifier{
	{&testNATSJetStreamMetadata[0], 0, "s0-nats-jetstream-mystream"},
	{&testNATSJetStreamMetadata[0], 1, "s1-nats-jetstream-mystream"},
}

func TestNATSJetStreamParseMetadata(t *testing.T) {
	for _, testData := range testNATSJetStreamMetadata {
		_, err := parseNATSJetStreamMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		} else if testData.isError && err == nil {
			t.Error("Expected error but got success" + testData.authParams["natsServerMonitoringEndpoint"] + "foo")
		}
	}
}

func TestNATSJetStreamGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range natsJetStreamMetricIdentifiers {
		ctx := context.Background()
		meta, err := parseNATSJetStreamMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockJetStreamScaler := natsJetStreamScaler{
			stream:     nil,
			metadata:   meta,
			httpClient: http.DefaultClient,
		}

		metricSpec := mockJetStreamScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestGetNATSJetStreamEndpointHTTPS(t *testing.T) {
	endpoint := getNATSJetStreamMonitoringURL(true, "nats.nats:8222", "$G")

	assert.True(t, strings.HasPrefix(endpoint, "https:"))
}

func TestGetNATSJetStreamEndpointHTTP(t *testing.T) {
	endpoint := getNATSJetStreamMonitoringURL(false, "nats.nats:8222", "$G")

	assert.True(t, strings.HasPrefix(endpoint, "http:"))
}

var testNATSJetStreamGoodMetadata = map[string]string{"natsServerMonitoringEndpoint": "localhost:8222", "account": "$G", "stream": "mystream", "consumer": "pull_consumer", "useHttps": "false", "activationLagThreshold": "10"}

var testNATSJetStreamMockResponses = []parseNATSJetStreamMockResponsesTestData{
	{
		"All Good - no messages waiting (not active)",
		&natsJetStreamMetricIdentifier{
			&parseNATSJetStreamMetadataTestData{
				testNATSJetStreamGoodMetadata, map[string]string{}, false,
			},
			0, "s0-nats-jetstream-mystream",
		},
		&jetStreamEndpointResponse{
			Accounts: []accountDetail{{
				Name: "$G",
				Streams: []*streamDetail{{
					Name:      "mystream",
					Consumers: []consumerDetail{{Name: "pull_consumer"}},
				}},
			}},
		}, false, false,
	},
	{
		"All Good - messages waiting (active)",
		&natsJetStreamMetricIdentifier{
			&parseNATSJetStreamMetadataTestData{
				testNATSJetStreamGoodMetadata, map[string]string{}, false,
			},
			0, "s0-nats-jetstream-mystream",
		},
		&jetStreamEndpointResponse{
			Accounts: []accountDetail{{
				Name: "$G",
				Streams: []*streamDetail{{
					Name:      "mystream",
					Consumers: []consumerDetail{{Name: "pull_consumer", NumPending: 100}},
				}},
			}},
		}, true, false,
	},
	{
		"Fail - stream exists but configured consumer missing",
		&natsJetStreamMetricIdentifier{
			&parseNATSJetStreamMetadataTestData{
				testNATSJetStreamGoodMetadata, map[string]string{}, false,
			},
			0, "s0-nats-jetstream-mystream",
		},
		&jetStreamEndpointResponse{
			Accounts: []accountDetail{{
				Name: "$G",
				Streams: []*streamDetail{{
					Name: "mystream", State: streamState{LastSequence: 1000},
					Consumers: []consumerDetail{{Name: "pull_consumer_bad", NumPending: 100}},
				}},
			}},
		}, false, true,
	},
	{
		"Fail - Non-matching stream name",
		&natsJetStreamMetricIdentifier{
			&parseNATSJetStreamMetadataTestData{
				testNATSJetStreamGoodMetadata, map[string]string{}, false,
			},
			0, "s0-nats-jetstream-mystream",
		},
		&jetStreamEndpointResponse{
			Accounts: []accountDetail{{
				Name: "$G",
				Streams: []*streamDetail{{
					Name: "mystreamBad", State: streamState{LastSequence: 1},
					Consumers: []consumerDetail{{Name: "pull_consumer", NumPending: 100}},
				}},
			}},
		}, false, true,
	},
	{
		"Fail - Unresolvable nats endpoint from config",
		&natsJetStreamMetricIdentifier{
			&parseNATSJetStreamMetadataTestData{
				map[string]string{"natsServerMonitoringEndpoint": "asdf32423fdsafdasdf:8222", "account": "$G", "stream": "mystream", "consumer": "pull_consumer", "activationLagThreshold": "10"}, map[string]string{}, false,
			},
			0, "s0-nats-jetstream-mystream",
		},
		&jetStreamEndpointResponse{
			Accounts: []accountDetail{{
				Name: "$G",
				Streams: []*streamDetail{{
					Name:      "mystream",
					Consumers: []consumerDetail{{Name: "pull_consumer", NumPending: 100}},
				}},
			}},
		}, false, true,
	},
	{
		"All Good - messages waiting (clustered)",
		&natsJetStreamMetricIdentifier{
			&parseNATSJetStreamMetadataTestData{
				testNATSJetStreamGoodMetadata, map[string]string{}, false,
			},
			0, "s0-nats-jetstream-mystream",
		},
		&jetStreamEndpointResponse{
			MetaCluster: metaCluster{ClusterSize: 3},
			Accounts: []accountDetail{{
				Name: "$G",
				Streams: []*streamDetail{{
					Name:      "mystream",
					Consumers: []consumerDetail{{Name: "pull_consumer", NumPending: 100, Cluster: consumerCluster{Leader: "leader"}}},
				}},
			}},
		}, true, false,
	},
	{
		"Not Active - consumer missing - connected to node without consumer info (clustered)",
		&natsJetStreamMetricIdentifier{
			&parseNATSJetStreamMetadataTestData{
				testNATSJetStreamGoodMetadata, map[string]string{}, false,
			},
			0, "s0-nats-jetstream-mystream",
		},
		&jetStreamEndpointResponse{
			MetaCluster: metaCluster{ClusterSize: 3},
			Accounts: []accountDetail{{
				Name:    "$G",
				Streams: []*streamDetail{{Name: "mystream"}},
			}},
		}, false, true,
	},
}

var testNATSJetStreamServerMockResponses = map[string][]byte{
	"localhost:8222": []byte(`{"server_name": "leader", "cluster": {"urls": ["leader.localhost.nats.svc:8222","not-leader-1.localhost.nats.svc:8222", "not-leader-2.localhost.nats.svc:8222"]}}`),
}

func TestNATSJetStreamIsActive(t *testing.T) {
	for _, mockResponse := range testNATSJetStreamMockResponses {
		mockResponseJSON, err := json.Marshal(mockResponse.data)
		if err != nil {
			t.Fatal("Could not parse mock response struct:", err)
		}

		client, srv := natsMockHTTPJetStreamServer(t, mockResponseJSON)

		ctx := context.Background()
		meta, err := parseNATSJetStreamMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: mockResponse.metadata.metadataTestData.metadata, TriggerIndex: mockResponse.metadata.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}

		mockJetStreamScaler := natsJetStreamScaler{
			stream:     nil,
			metadata:   meta,
			httpClient: client,
			logger:     InitializeLogger(&scalersconfig.ScalerConfig{TriggerMetadata: mockResponse.metadata.metadataTestData.metadata, TriggerIndex: mockResponse.metadata.triggerIndex}, "nats_jetstream_scaler"),
		}

		_, isActive, err := mockJetStreamScaler.GetMetricsAndActivity(ctx, "metric_name")
		if err != nil && !mockResponse.isError {
			t.Errorf("Expected success for '%s' but got error %s", mockResponse.name, err)
		} else if mockResponse.isError && err == nil {
			t.Errorf("Expected error for '%s' but got success %s", mockResponse.name, mockResponse.metadata.metadataTestData.authParams["natsServerMonitoringEndpoint"])
		}

		if isActive != mockResponse.isActive {
			t.Errorf("Expected '%s' 'isActive=%s', got '%s'", mockResponse.name, strconv.FormatBool(mockResponse.isActive), strconv.FormatBool(isActive))
		}
		srv.Close()
	}
}

func TestNewNATSJetStreamScaler(t *testing.T) {
	// All Good
	_, err := NewNATSJetStreamScaler(&scalersconfig.ScalerConfig{TriggerMetadata: testNATSJetStreamGoodMetadata, TriggerIndex: 0})
	if err != nil {
		t.Error("Expected success for New NATS JetStream Scaler but got error", err)
	}

	// Fail - Empty account
	_, err = NewNATSJetStreamScaler(&scalersconfig.ScalerConfig{TriggerMetadata: map[string]string{"natsServerMonitoringEndpoint": "localhost:8222", "account": ""}})
	if err == nil {
		t.Error("Expected error for parsing monitoring leader URL but got success")
	}
}

func TestNATSJetStreamGetMetrics(t *testing.T) {
	for _, mockResponse := range testNATSJetStreamMockResponses {
		mockResponseJSON, err := json.Marshal(mockResponse.data)
		if err != nil {
			t.Fatal("Could not parse mock response struct:", err)
		}

		client, srv := natsMockHTTPJetStreamServer(t, mockResponseJSON)

		ctx := context.Background()
		meta, err := parseNATSJetStreamMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: mockResponse.metadata.metadataTestData.metadata, TriggerIndex: mockResponse.metadata.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}

		mockJetStreamScaler := natsJetStreamScaler{
			stream:     nil,
			metadata:   meta,
			httpClient: client,
			logger:     InitializeLogger(&scalersconfig.ScalerConfig{TriggerMetadata: mockResponse.metadata.metadataTestData.metadata, TriggerIndex: mockResponse.metadata.triggerIndex}, "nats_jetstream_scaler"),
		}

		_, _, err = mockJetStreamScaler.GetMetricsAndActivity(ctx, "metric_name")

		if err != nil && !mockResponse.isError {
			t.Errorf("Expected success for '%s' but got error %s", mockResponse.name, err)
		} else if mockResponse.isError && err == nil {
			t.Errorf("Expected error for '%s' but got success %s", mockResponse.name, mockResponse.metadata.metadataTestData.authParams["natsServerMonitoringEndpoint"])
		}
		srv.Close()
	}
}

func natsMockHTTPJetStreamServer(t *testing.T, mockResponseJSON []byte) (*http.Client, *httptest.Server) {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	// redirect leader.localhost for the clustered test
	client := &http.Client{
		Transport: &http.Transport{},
	}
	client.Transport.(*http.Transport).DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		if strings.HasSuffix(addr, ".localhost:8222") {
			addr = "127.0.0.1:8222"
		}
		return dialer.DialContext(ctx, network, addr)
	}

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/jsz":
			w.WriteHeader(http.StatusOK)
			// if requesting from specific node and not a leader, which indicate clustered test
			// send empty response
			if strings.HasSuffix(r.Host, ".localhost:8222") && r.Host != "leader.localhost:8222" {
				mockResponseJSON, _ = json.Marshal(&jetStreamEndpointResponse{})
			}
			_, err := w.Write(mockResponseJSON)
			if err != nil {
				t.Fatal("Could not write to the http server connection:", err)
			}
		case "/varz":
			w.WriteHeader(http.StatusOK)
			res, ok := testNATSJetStreamServerMockResponses[r.Host]
			if !ok {
				// if given host is not a specific node (e.g. loadbalancer)
				// get response from random node
				for _, v := range testNATSJetStreamServerMockResponses {
					res = v
					break
				}
			}
			_, err := w.Write(res)
			if err != nil {
				t.Fatal("Could not write to the http server connection:", err)
			}
		default:
			t.Errorf("Expected to request '/jsz or /varz', got: %s", r.URL.Path)
		}
	}))

	l, _ := net.Listen("tcp", "127.0.0.1:8222")
	srv.Listener = l
	srv.Start()

	return client, srv
}

func TestNATSJetStreamgetNATSJetstreamMonitoringData(t *testing.T) {
	client, invalidJSONServer := natsMockHTTPJetStreamServer(t, []byte(`{invalidJSON}`))
	defer func() {
		invalidJSONServer.Close()
	}()

	ctx := context.Background()
	meta, err := parseNATSJetStreamMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testNATSJetStreamGoodMetadata, TriggerIndex: 0})
	if err != nil {
		t.Fatal("Could not parse metadata:", err)
	}

	mockJetStreamScaler := natsJetStreamScaler{
		stream:     nil,
		metadata:   meta,
		httpClient: client,
		logger:     InitializeLogger(&scalersconfig.ScalerConfig{TriggerMetadata: testNATSJetStreamGoodMetadata, TriggerIndex: 0}, "nats_jetstream_scaler"),
	}

	err = mockJetStreamScaler.getNATSJetstreamMonitoringData(ctx, mockJetStreamScaler.metadata.monitoringURL)
	if err == nil {
		t.Error("Expected error for bad JSON monitoring data but got success")
	}
}

func TestNATSJetStreamGetNATSJetstreamNodeURL(t *testing.T) {
	client, invalidJSONServer := natsMockHTTPJetStreamServer(t, []byte(`{invalidJSON}`))
	defer invalidJSONServer.Close()

	meta, err := parseNATSJetStreamMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testNATSJetStreamGoodMetadata, TriggerIndex: 0})
	if err != nil {
		t.Fatal("Could not parse metadata:", err)
	}

	mockJetStreamScaler := natsJetStreamScaler{
		stream:     nil,
		metadata:   meta,
		httpClient: client,
		logger:     InitializeLogger(&scalersconfig.ScalerConfig{TriggerMetadata: testNATSJetStreamGoodMetadata, TriggerIndex: 0}, "nats_jetstream_scaler"),
	}

	mockJetStreamScaler.metadata.monitoringURL = "234234:::::34234234;;;;really_bad_URL;;/"

	_, err = mockJetStreamScaler.getNATSJetStreamMonitoringNodeURL("leader")
	if err == nil {
		t.Error("Expected error for parsing monitoring node URL but got success")
	}
}

func TestNATSJetStreamGetNATSJetstreamServerURL(t *testing.T) {
	client, invalidJSONServer := natsMockHTTPJetStreamServer(t, []byte(`{invalidJSON}`))
	defer invalidJSONServer.Close()

	meta, err := parseNATSJetStreamMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testNATSJetStreamGoodMetadata, TriggerIndex: 0})
	if err != nil {
		t.Fatal("Could not parse metadata:", err)
	}

	mockJetStreamScaler := natsJetStreamScaler{
		stream:     nil,
		metadata:   meta,
		httpClient: client,
		logger:     InitializeLogger(&scalersconfig.ScalerConfig{TriggerMetadata: testNATSJetStreamGoodMetadata, TriggerIndex: 0}, "nats_jetstream_scaler"),
	}

	mockJetStreamScaler.metadata.monitoringURL = "234234:::::34234234;;;;really_bad_URL;;/"

	_, err = mockJetStreamScaler.getNATSJetStreamMonitoringServerURL("")
	if err == nil {
		t.Error("Expected error for parsing monitoring server URL but got success")
	}
}

func TestNATSJetStreamGetMaxMsgLag(t *testing.T) {
	meta, err := parseNATSJetStreamMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testNATSJetStreamGoodMetadata, TriggerIndex: 0})
	if err != nil {
		t.Fatal("Could not parse metadata:", err)
	}

	t.Run("consumer present returns pending+ackPending", func(t *testing.T) {
		scaler := natsJetStreamScaler{
			metadata: meta,
			stream: &streamDetail{
				Name:  "mystream",
				State: streamState{LastSequence: 9999},
				Consumers: []consumerDetail{{
					Name:          "pull_consumer",
					NumPending:    7,
					NumAckPending: 3,
				}},
			},
		}

		lag, err := scaler.getMaxMsgLag()
		assert.NoError(t, err)
		assert.Equal(t, int64(10), lag)
	})

	t.Run("stream exists but consumer missing returns error and zero lag", func(t *testing.T) {
		scaler := natsJetStreamScaler{
			metadata: meta,
			stream: &streamDetail{
				Name:  "mystream",
				State: streamState{LastSequence: 9999},
				Consumers: []consumerDetail{{
					Name:       "some_other_consumer",
					NumPending: 100,
				}},
			},
		}

		lag, err := scaler.getMaxMsgLag()
		assert.Error(t, err)
		assert.Equal(t, int64(0), lag, "must not fall back to stream LastSequence")
		assert.Contains(t, err.Error(), "pull_consumer")
		assert.Contains(t, err.Error(), "mystream")
	})
}

func TestNATSJetStreamClose(t *testing.T) {
	mockJetStreamScaler, err := NewNATSJetStreamScaler(&scalersconfig.ScalerConfig{TriggerMetadata: testNATSJetStreamGoodMetadata, TriggerIndex: 0})
	if err != nil {
		t.Error("Expected success for New NATS JetStream Scaler but got error", err)
	}

	ctx := context.Background()
	jsClose := mockJetStreamScaler.Close(ctx)

	if jsClose != nil {
		t.Error("Expected success for NATS JetStream Scaler Close but got error", err)
	}
}

func TestGetNATSJetStreamMonitoringURL(t *testing.T) {
	tests := []struct {
		name               string
		useHTTPS           bool
		natsServerEndpoint string
		accountID          string
		expectedURL        string
	}{
		{
			name:               "HTTP scheme",
			useHTTPS:           false,
			natsServerEndpoint: "nats.nats:8222",
			accountID:          "$G",
			expectedURL:        "http://nats.nats:8222/jsz?acc=%24G&config=true&consumers=true",
		},
		{
			name:               "HTTPS scheme",
			useHTTPS:           true,
			natsServerEndpoint: "nats.nats:8222",
			accountID:          "$G",
			expectedURL:        "https://nats.nats:8222/jsz?acc=%24G&config=true&consumers=true",
		},
		{
			name:               "Account ID with ampersand is URL-encoded and cannot inject parameters",
			useHTTPS:           false,
			natsServerEndpoint: "nats.nats:8222",
			accountID:          "myaccount&consumers=false",
			expectedURL:        "http://nats.nats:8222/jsz?acc=myaccount%26consumers%3Dfalse&config=true&consumers=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := getNATSJetStreamMonitoringURL(tt.useHTTPS, tt.natsServerEndpoint, tt.accountID)
			assert.Equal(t, tt.expectedURL, endpoint)

			parsedURL, err := url.Parse(endpoint)
			assert.NoError(t, err)
			q := parsedURL.Query()
			assert.Equal(t, "true", q.Get("consumers"), "consumers parameter must always be true")
			assert.Equal(t, "true", q.Get("config"), "config parameter must always be true")
			assert.Equal(t, tt.accountID, q.Get("acc"), "acc parameter must equal the raw account ID")
		})
	}
}

func TestNATSJetStreamMonitoringNodeURLEncoding(t *testing.T) {
	meta, err := parseNATSJetStreamMetadata(&scalersconfig.ScalerConfig{
		TriggerMetadata: testNATSJetStreamGoodMetadata,
		TriggerIndex:    0,
	})
	if err != nil {
		t.Fatal("Could not parse metadata:", err)
	}

	scaler := natsJetStreamScaler{
		metadata: meta,
		logger:   InitializeLogger(&scalersconfig.ScalerConfig{TriggerMetadata: testNATSJetStreamGoodMetadata}, "nats_jetstream_scaler"),
	}

	nodeURL, err := scaler.getNATSJetStreamMonitoringNodeURL("leader.nats.svc")
	assert.NoError(t, err)

	parsedURL, err := url.Parse(nodeURL)
	assert.NoError(t, err)
	q := parsedURL.Query()
	assert.Equal(t, "true", q.Get("consumers"), "consumers parameter must always be true after node URL construction")
	assert.Equal(t, "true", q.Get("config"), "config parameter must always be true after node URL construction")
	assert.Equal(t, "$G", q.Get("acc"), "acc parameter must be preserved correctly in node URL")
	assert.Equal(t, "leader.nats.svc:8222", parsedURL.Host, "node hostname must be set correctly")
}

func TestNATSJetStreamMonitoringNodeURLByNodeEncoding(t *testing.T) {
	meta, err := parseNATSJetStreamMetadata(&scalersconfig.ScalerConfig{
		TriggerMetadata: testNATSJetStreamGoodMetadata,
		TriggerIndex:    0,
	})
	if err != nil {
		t.Fatal("Could not parse metadata:", err)
	}

	scaler := natsJetStreamScaler{
		metadata: meta,
		logger:   InitializeLogger(&scalersconfig.ScalerConfig{TriggerMetadata: testNATSJetStreamGoodMetadata}, "nats_jetstream_scaler"),
	}

	nodeURL, err := scaler.getNATSJetStreamMonitoringNodeURLByNode("leader")
	assert.NoError(t, err)

	parsedURL, err := url.Parse(nodeURL)
	assert.NoError(t, err)
	q := parsedURL.Query()
	assert.Equal(t, "true", q.Get("consumers"), "consumers parameter must always be true after node-by-node URL construction")
	assert.Equal(t, "true", q.Get("config"), "config parameter must always be true after node-by-node URL construction")
	assert.Equal(t, "$G", q.Get("acc"), "acc parameter must be preserved correctly in node-by-node URL")
	assert.True(t, strings.HasPrefix(parsedURL.Host, "leader."), "node-by-node hostname must be prefixed with node name")
}
