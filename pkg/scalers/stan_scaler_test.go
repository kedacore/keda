package scalers

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseStanMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

type stanMetricIdentifier struct {
	metadataTestData *parseStanMetadataTestData
	triggerIndex     int
	name             string
}

var testStanMetadata = []parseStanMetadataTestData{
	// nothing passed
	{map[string]string{}, map[string]string{}, true},
	// Missing subject name, should fail
	{map[string]string{"natsServerMonitoringEndpoint": "stan-nats-ss", "queueGroup": "grp1", "durableName": "ImDurable"}, map[string]string{}, true},
	// Missing durable name, should fail
	{map[string]string{"natsServerMonitoringEndpoint": "stan-nats-ss", "queueGroup": "grp1", "subject": "mySubject"}, map[string]string{}, true},
	// Missing nats server monitoring endpoint, should fail
	{map[string]string{"queueGroup": "grp1", "subject": "mySubject"}, map[string]string{}, true},
	// All good.
	{map[string]string{"natsServerMonitoringEndpoint": "stan-nats-ss", "queueGroup": "grp1", "durableName": "ImDurable", "subject": "mySubject", "useHttps": "true"}, map[string]string{}, false},
	// All good + activationLagThreshold
	{map[string]string{"natsServerMonitoringEndpoint": "stan-nats-ss", "queueGroup": "grp1", "durableName": "ImDurable", "subject": "mySubject", "activationLagThreshold": "10"}, map[string]string{}, false},
	// natsServerMonitoringEndpoint is defined in authParams
	{map[string]string{"queueGroup": "grp1", "durableName": "ImDurable", "subject": "mySubject"}, map[string]string{"natsServerMonitoringEndpoint": "stan-nats-ss"}, false},
	// Missing nats server monitoring endpoint , should fail
	{map[string]string{"queueGroup": "grp1", "durableName": "ImDurable", "subject": "mySubject"}, map[string]string{"natsServerMonitoringEndpoint": ""}, true},
	// Misconfigured https, should fail
	{map[string]string{"natsServerMonitoringEndpoint": "stan-nats-ss", "queueGroup": "grp1", "durableName": "ImDurable", "subject": "mySubject", "useHttps": "error"}, map[string]string{}, true},
}

var stanMetricIdentifiers = []stanMetricIdentifier{
	{&testStanMetadata[4], 0, "s0-stan-mySubject"},
	{&testStanMetadata[4], 1, "s1-stan-mySubject"},
}

func TestStanParseMetadata(t *testing.T) {
	for _, testData := range testStanMetadata {
		_, err := parseStanMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		} else if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestStanGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range stanMetricIdentifiers {
		ctx := context.Background()
		meta, err := parseStanMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockStanScaler := stanScaler{
			channelInfo: nil,
			metadata:    meta,
			httpClient:  http.DefaultClient,
		}

		metricSpec := mockStanScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestGetSTANChannelsEndpointHTTPS(t *testing.T) {
	endpoint := getSTANChannelsEndpoint(true, "stan-nats-ss")

	assert.True(t, strings.HasPrefix(endpoint, "https:"))
}

func TestGetSTANChannelsEndpointHTTP(t *testing.T) {
	endpoint := getSTANChannelsEndpoint(false, "stan-nats-ss")

	assert.True(t, strings.HasPrefix(endpoint, "http:"))
}
