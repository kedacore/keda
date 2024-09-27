package scalers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	"github.com/stretchr/testify/assert"
)

var (
	temporalEndpoint  = "localhost:7233"
	temporalNamespace = "v2"
	temporalQueueName = "default"

	logger = logr.Discard()
)

type parseTemporalMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

type temporalMetricIdentifier struct {
	metadataTestData *parseTemporalMetadataTestData
	triggerIndex     int
	name             string
}

var testTemporalMetadata = []parseTemporalMetadataTestData{
	// nothing passed
	{map[string]string{}, true},
	// Missing queueName, should fail
	{map[string]string{"endpoint": temporalEndpoint, "namespace": temporalNamespace}, true},
	// Missing namespace, should success
	{map[string]string{"endpoint": temporalEndpoint, "queueName": temporalQueueName}, false},
	// Missing endpoint, should fail
	{map[string]string{"queueName": temporalQueueName, "namespace": temporalNamespace}, true},
	// All good.
	{map[string]string{"endpoint": temporalEndpoint, "queueName": temporalQueueName, "namespace": temporalNamespace}, false},
	// All good + activationLagThreshold
	{map[string]string{"endpoint": temporalEndpoint, "queueName": temporalQueueName, "namespace": temporalNamespace, "activationTargetQueueSize": "10"}, false},
}

var temporalMetricIdentifiers = []temporalMetricIdentifier{
	{&testTemporalMetadata[4], 0, "s0-temporal-v2-default"},
	{&testTemporalMetadata[4], 1, "s1-temporal-v2-default"},
}

func TestTemporalGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range temporalMetricIdentifiers {
		ctx := context.Background()
		meta, err := parseTemporalMetadata(&scalersconfig.ScalerConfig{
			TriggerMetadata: testData.metadataTestData.metadata,
			TriggerIndex:    testData.triggerIndex,
		}, logger)
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockTemporalScaler := temporalScaler{
			metadata: meta,
		}

		metricSpec := mockTemporalScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestParseTemporalMetadata(t *testing.T) {
	cases := []struct {
		name     string
		metadata map[string]string
		wantMeta *temporalMetadata
		wantErr  bool
	}{
		{
			name:     "empty metadata",
			wantMeta: nil,
			wantErr:  true,
		},
		{
			name: "empty queue name",
			metadata: map[string]string{
				"endpoint":  "test:7233",
				"namespace": "default",
			},
			wantMeta: nil,
			wantErr:  true,
		},
		{
			name: "empty namespace",
			metadata: map[string]string{
				"endpoint":  "test:7233",
				"queueName": "testxx",
			},
			wantMeta: &temporalMetadata{
				endpoint:        "test:7233",
				namespace:       "default",
				queueName:       "testxx",
				targetQueueSize: 5,
			},
			wantErr: false,
		},
		{
			name: "activationTargetQueueSize should not be 0",
			metadata: map[string]string{
				"endpoint":                  "test:7233",
				"namespace":                 "default",
				"queueName":                 "testxx",
				"activationTargetQueueSize": "12",
			},
			wantMeta: &temporalMetadata{
				endpoint:               "test:7233",
				namespace:              "default",
				queueName:              "testxx",
				targetQueueSize:        5,
				activationLagThreshold: 12,
			},
			wantErr: false,
		},
	}

	for _, testCase := range cases {
		c := testCase
		t.Run(c.name, func(t *testing.T) {
			config := &scalersconfig.ScalerConfig{
				TriggerMetadata: c.metadata,
			}
			meta, err := parseTemporalMetadata(config, logger)
			if c.wantErr == true && err != nil {
				t.Log("Expected error, got err")
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, c.wantMeta, meta)
		})
	}
}
