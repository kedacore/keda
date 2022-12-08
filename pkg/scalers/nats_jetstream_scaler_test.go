package scalers

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	scalerIndex      int
	name             string
}

var testNATSJetStreamMetadata = []parseNATSJetStreamMetadataTestData{
	// All good localhost.
	{map[string]string{"natsServerMonitoringEndpoint": "localhost:8222", "account": "$G", "stream": "mystream", "consumer": "pull_consumer", "useHttps": "false"}, map[string]string{}, false},
	// All good url.
	{map[string]string{"natsServerMonitoringEndpoint": "nats.nats:8222", "account": "$G", "stream": "mystream", "consumer": "pull_consumer", "useHttps": "true"}, map[string]string{}, false},
	// nothing passed
	{map[string]string{}, map[string]string{}, true},
	// Missing account name, should fail
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
}

var natsJetStreamMetricIdentifiers = []natsJetStreamMetricIdentifier{
	{&testNATSJetStreamMetadata[0], 0, "s0-nats-jetstream-mystream"},
	{&testNATSJetStreamMetadata[0], 1, "s1-nats-jetstream-mystream"},
}

func TestNATSJetStreamParseMetadata(t *testing.T) {
	for _, testData := range testNATSJetStreamMetadata {
		_, err := parseNATSJetStreamMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
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
		meta, err := parseNATSJetStreamMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ScalerIndex: testData.scalerIndex})
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
				testNATSJetStreamGoodMetadata, map[string]string{}, false},
			0, "s0-nats-jetstream-mystream",
		},
		&jetStreamEndpointResponse{
			Accounts: []accountDetail{{Name: "$G",
				Streams: []*streamDetail{{Name: "mystream",
					Consumers: []consumerDetail{{Name: "pull_consumer"}},
				}},
			}},
		}, false, false},
	{
		"All Good - messages waiting (active)",
		&natsJetStreamMetricIdentifier{
			&parseNATSJetStreamMetadataTestData{
				testNATSJetStreamGoodMetadata, map[string]string{}, false},
			0, "s0-nats-jetstream-mystream",
		},
		&jetStreamEndpointResponse{
			Accounts: []accountDetail{{Name: "$G",
				Streams: []*streamDetail{{Name: "mystream",
					Consumers: []consumerDetail{{Name: "pull_consumer", NumPending: 100}},
				}},
			}},
		}, true, false},
	{
		"Not Active - Bad consumer name uses stream last sequence",
		&natsJetStreamMetricIdentifier{
			&parseNATSJetStreamMetadataTestData{
				testNATSJetStreamGoodMetadata, map[string]string{}, false},
			0, "s0-nats-jetstream-mystream",
		},
		&jetStreamEndpointResponse{
			Accounts: []accountDetail{{Name: "$G",
				Streams: []*streamDetail{{Name: "mystream", State: streamState{LastSequence: 1},
					Consumers: []consumerDetail{{Name: "pull_consumer_bad", NumPending: 100}},
				}},
			}},
		}, false, false},
	{
		"Fail - Non-matching stream name",
		&natsJetStreamMetricIdentifier{
			&parseNATSJetStreamMetadataTestData{
				testNATSJetStreamGoodMetadata, map[string]string{}, false},
			0, "s0-nats-jetstream-mystream",
		},
		&jetStreamEndpointResponse{
			Accounts: []accountDetail{{Name: "$G",
				Streams: []*streamDetail{{Name: "mystreamBad", State: streamState{LastSequence: 1},
					Consumers: []consumerDetail{{Name: "pull_consumer", NumPending: 100}},
				}},
			}},
		}, false, true},
	{
		"Fail - Unresolvable nats endpoint from config",
		&natsJetStreamMetricIdentifier{
			&parseNATSJetStreamMetadataTestData{
				map[string]string{"natsServerMonitoringEndpoint": "asdf32423fdsafdasdf:8222", "account": "$G", "stream": "mystream", "consumer": "pull_consumer", "activationLagThreshold": "10"}, map[string]string{}, false},
			0, "s0-nats-jetstream-mystream",
		},
		&jetStreamEndpointResponse{
			Accounts: []accountDetail{{Name: "$G",
				Streams: []*streamDetail{{Name: "mystream",
					Consumers: []consumerDetail{{Name: "pull_consumer", NumPending: 100}},
				}},
			}},
		}, false, true},
	{
		"All Good - messages waiting (clustered)",
		&natsJetStreamMetricIdentifier{
			&parseNATSJetStreamMetadataTestData{
				testNATSJetStreamGoodMetadata, map[string]string{}, false},
			0, "s0-nats-jetstream-mystream",
		},
		&jetStreamEndpointResponse{
			MetaCluster: metaCluster{ClusterSize: 3},
			Accounts: []accountDetail{{Name: "$G",
				Streams: []*streamDetail{{Name: "mystream",
					Consumers: []consumerDetail{{Name: "pull_consumer", NumPending: 100, Cluster: consumerCluster{Leader: "leader"}}},
				}},
			}},
		}, true, false},
	{
		"Fail - Bad leader name (clustered)",
		&natsJetStreamMetricIdentifier{
			&parseNATSJetStreamMetadataTestData{
				testNATSJetStreamGoodMetadata, map[string]string{}, false},
			0, "s0-nats-jetstream-mystream",
		},
		&jetStreamEndpointResponse{
			MetaCluster: metaCluster{ClusterSize: 3},
			Accounts: []accountDetail{{Name: "$G",
				Streams: []*streamDetail{{Name: "mystream",
					Consumers: []consumerDetail{{Name: "pull_consumer", NumPending: 100, Cluster: consumerCluster{Leader: "leaderBad!!!!"}}},
				}},
			}},
		}, false, true},
	{
		"Not Active - consumer missing - connected to node without consumer info (clustered)",
		&natsJetStreamMetricIdentifier{
			&parseNATSJetStreamMetadataTestData{
				testNATSJetStreamGoodMetadata, map[string]string{}, false},
			0, "s0-nats-jetstream-mystream",
		},
		&jetStreamEndpointResponse{
			MetaCluster: metaCluster{ClusterSize: 3},
			Accounts: []accountDetail{{Name: "$G",
				Streams: []*streamDetail{{Name: "mystream"}},
			}},
		}, false, false},
}

func TestNATSJetStreamIsActive(t *testing.T) {
	for _, mockResponse := range testNATSJetStreamMockResponses {
		mockResponseJSON, err := json.Marshal(mockResponse.data)
		if err != nil {
			t.Fatal("Could not parse mock response struct:", err)
		}

		srv := natsMockHTTPJetStreamServer(t, mockResponseJSON)
		defer srv.Close()

		ctx := context.Background()
		meta, err := parseNATSJetStreamMetadata(&ScalerConfig{TriggerMetadata: mockResponse.metadata.metadataTestData.metadata, ScalerIndex: mockResponse.metadata.scalerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}

		mockJetStreamScaler := natsJetStreamScaler{
			stream:     nil,
			metadata:   meta,
			httpClient: http.DefaultClient,
			logger:     InitializeLogger(&ScalerConfig{TriggerMetadata: mockResponse.metadata.metadataTestData.metadata, ScalerIndex: mockResponse.metadata.scalerIndex}, "nats_jetstream_scaler"),
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
	_, err := NewNATSJetStreamScaler(&ScalerConfig{TriggerMetadata: testNATSJetStreamGoodMetadata, ScalerIndex: 0})
	if err != nil {
		t.Error("Expected success for New NATS JetStream Scaler but got error", err)
	}

	// Fail - Empty account
	_, err = NewNATSJetStreamScaler(&ScalerConfig{TriggerMetadata: map[string]string{"natsServerMonitoringEndpoint": "localhost:8222", "account": ""}})
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

		tr := http.DefaultTransport.(*http.Transport).Clone()
		srv := natsMockHTTPJetStreamServer(t, mockResponseJSON)
		defer func() {
			srv.Close()
			http.DefaultTransport = tr
		}()

		ctx := context.Background()
		meta, err := parseNATSJetStreamMetadata(&ScalerConfig{TriggerMetadata: mockResponse.metadata.metadataTestData.metadata, ScalerIndex: mockResponse.metadata.scalerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}

		mockJetStreamScaler := natsJetStreamScaler{
			stream:     nil,
			metadata:   meta,
			httpClient: http.DefaultClient,
			logger:     InitializeLogger(&ScalerConfig{TriggerMetadata: mockResponse.metadata.metadataTestData.metadata, ScalerIndex: mockResponse.metadata.scalerIndex}, "nats_jetstream_scaler"),
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

func natsMockHTTPJetStreamServer(t *testing.T, mockResponseJSON []byte) *httptest.Server {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	// redirect leader.localhost for the clustered test
	http.DefaultTransport.(*http.Transport).DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		if addr == "leader.localhost:8222" {
			addr = "127.0.0.1:8222"
		}
		return dialer.DialContext(ctx, network, addr)
	}

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/jsz":
			w.WriteHeader(http.StatusOK)
			_, err := w.Write(mockResponseJSON)
			if err != nil {
				t.Fatal("Could not write to the http server connection:", err)
			}
		case "/varz":
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"cluster": {"urls": ["leader.localhost.nats.svc:8222"]}}`))
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

	return srv
}

func TestNATSJetStreamgetNATSJetstreamMonitoringData(t *testing.T) {
	tr := http.DefaultTransport.(*http.Transport).Clone()

	invalidJSONServer := natsMockHTTPJetStreamServer(t, []byte(`{invalidJSON}`))
	defer func() {
		invalidJSONServer.Close()
		http.DefaultTransport = tr
	}()

	ctx := context.Background()
	meta, err := parseNATSJetStreamMetadata(&ScalerConfig{TriggerMetadata: testNATSJetStreamGoodMetadata, ScalerIndex: 0})
	if err != nil {
		t.Fatal("Could not parse metadata:", err)
	}

	mockJetStreamScaler := natsJetStreamScaler{
		stream:     nil,
		metadata:   meta,
		httpClient: http.DefaultClient,
		logger:     InitializeLogger(&ScalerConfig{TriggerMetadata: testNATSJetStreamGoodMetadata, ScalerIndex: 0}, "nats_jetstream_scaler"),
	}

	err = mockJetStreamScaler.getNATSJetstreamMonitoringData(ctx, mockJetStreamScaler.metadata.monitoringURL)
	if err == nil {
		t.Error("Expected error for bad JSON monitoring data but got success")
	}
}

func TestNATSJetStreamGetNATSJetstreamNodeURL(t *testing.T) {
	invalidJSONServer := natsMockHTTPJetStreamServer(t, []byte(`{invalidJSON}`))
	defer invalidJSONServer.Close()

	meta, err := parseNATSJetStreamMetadata(&ScalerConfig{TriggerMetadata: testNATSJetStreamGoodMetadata, ScalerIndex: 0})
	if err != nil {
		t.Fatal("Could not parse metadata:", err)
	}

	mockJetStreamScaler := natsJetStreamScaler{
		stream:     nil,
		metadata:   meta,
		httpClient: http.DefaultClient,
		logger:     InitializeLogger(&ScalerConfig{TriggerMetadata: testNATSJetStreamGoodMetadata, ScalerIndex: 0}, "nats_jetstream_scaler"),
	}

	mockJetStreamScaler.metadata.monitoringURL = "234234:::::34234234;;;;really_bad_URL;;/"

	_, err = mockJetStreamScaler.getNATSJetStreamMonitoringNodeURL("leader")
	if err == nil {
		t.Error("Expected error for parsing monitoring node URL but got success")
	}
}

func TestNATSJetStreamGetNATSJetstreamServerURL(t *testing.T) {
	invalidJSONServer := natsMockHTTPJetStreamServer(t, []byte(`{invalidJSON}`))
	defer invalidJSONServer.Close()

	meta, err := parseNATSJetStreamMetadata(&ScalerConfig{TriggerMetadata: testNATSJetStreamGoodMetadata, ScalerIndex: 0})
	if err != nil {
		t.Fatal("Could not parse metadata:", err)
	}

	mockJetStreamScaler := natsJetStreamScaler{
		stream:     nil,
		metadata:   meta,
		httpClient: http.DefaultClient,
		logger:     InitializeLogger(&ScalerConfig{TriggerMetadata: testNATSJetStreamGoodMetadata, ScalerIndex: 0}, "nats_jetstream_scaler"),
	}

	mockJetStreamScaler.metadata.monitoringURL = "234234:::::34234234;;;;really_bad_URL;;/"

	_, err = mockJetStreamScaler.getNATSJetStreamMonitoringServerURL()
	if err == nil {
		t.Error("Expected error for parsing monitoring server URL but got success")
	}
}

func TestInvalidateNATSJetStreamCachedMonitoringData(t *testing.T) {
	meta, err := parseNATSJetStreamMetadata(&ScalerConfig{TriggerMetadata: testNATSJetStreamGoodMetadata, ScalerIndex: 0})
	if err != nil {
		t.Fatal("Could not parse metadata:", err)
	}

	mockJetStreamScaler := natsJetStreamScaler{
		stream:     nil,
		metadata:   meta,
		httpClient: http.DefaultClient,
		logger:     InitializeLogger(&ScalerConfig{TriggerMetadata: testNATSJetStreamGoodMetadata, ScalerIndex: 0}, "nats_jetstream_scaler"),
	}

	mockJetStreamScaler.invalidateNATSJetStreamCachedMonitoringData()
}

func TestNATSJetStreamClose(t *testing.T) {
	mockJetStreamScaler, err := NewNATSJetStreamScaler(&ScalerConfig{TriggerMetadata: testNATSJetStreamGoodMetadata, ScalerIndex: 0})
	if err != nil {
		t.Error("Expected success for New NATS JetStream Scaler but got error", err)
	}

	ctx := context.Background()
	jsClose := mockJetStreamScaler.Close(ctx)

	if jsClose != nil {
		t.Error("Expected success for NATS JetStream Scaler Close but got error", err)
	}
}
