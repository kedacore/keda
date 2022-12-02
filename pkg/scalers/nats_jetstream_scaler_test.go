package scalers

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type parseNATSJetStreamMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

type natsJetStreamMetricIdentifier struct {
	metadataTestData *parseNATSJetStreamMetadataTestData
	scalerIndex      int
	name             string
}

var testNATSJetStreamMetadata = []parseNATSJetStreamMetadataTestData{
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
	// All good.
	{map[string]string{"natsServerMonitoringEndpoint": "nats.nats:8222", "account": "$G", "stream": "mystream", "consumer": "pull_consumer", "useHttps": "true"}, map[string]string{}, false},
	// All good + activationLagThreshold
	{map[string]string{"natsServerMonitoringEndpoint": "nats.nats:8222", "account": "$G", "stream": "mystream", "consumer": "pull_consumer", "activationLagThreshold": "10"}, map[string]string{}, false},
	// natsServerMonitoringEndpoint is defined in authParams
	{map[string]string{"account": "$G", "stream": "mystream", "consumer": "pull_consumer"}, map[string]string{"natsServerMonitoringEndpoint": "nats.nats:8222"}, false},
	// Missing nats server monitoring endpoint , should fail
	{map[string]string{"account": "$G", "stream": "mystream", "consumer": "pull_consumer"}, map[string]string{"natsServerMonitoringEndpoint": ""}, true},
	// Misconfigured https, should fail
	{map[string]string{"natsServerMonitoringEndpoint": "nats.nats:8222", "account": "$G", "stream": "mystream", "consumer": "pull_consumer", "useHttps": "error"}, map[string]string{}, true},
}

var natsJetStreamMetricIdentifiers = []natsJetStreamMetricIdentifier{
	{&testNATSJetStreamMetadata[5], 0, "s0-nats-jetstream-mystream"},
	{&testNATSJetStreamMetadata[5], 1, "s1-nats-jetstream-mystream"},
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
		mockStanScaler := natsJetStreamScaler{
			stream:     nil,
			metadata:   meta,
			httpClient: http.DefaultClient,
		}

		metricSpec := mockStanScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestGetNATSJetStreamEndpointHTTPS(t *testing.T) {
	endpoint := getNATSJetStreamEndpoint(true, "nats.nats:8222", "$G")

	assert.True(t, strings.HasPrefix(endpoint, "https:"))
}

func TestGetNATSJetStreamEndpointHTTP(t *testing.T) {
	endpoint := getNATSJetStreamEndpoint(false, "nats.nats:8222", "$G")

	assert.True(t, strings.HasPrefix(endpoint, "http:"))
}
