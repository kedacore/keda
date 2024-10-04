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

func TestTemporalParseMetadata(t *testing.T) {
	for _, testData := range testTemporalMetadata {
		metadata := &scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata}
		_, err := parseTemporalMetadata(metadata, logger)

		if err != nil && !testData.isError {
			t.Error("Expected success but got err", err)
		}
		if err == nil && testData.isError {
			t.Error("Expected error but got success")
		}
	}
}

func TestTemporalGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range temporalMetricIdentifiers {
		metadata, err := parseTemporalMetadata(&scalersconfig.ScalerConfig{
			TriggerMetadata: testData.metadataTestData.metadata,
			TriggerIndex:    testData.triggerIndex,
		}, logger)

		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockScaler := temporalScaler{
			metadata: metadata,
		}
		metricSpec := mockScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name

		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestParseTemporalMetadata(t *testing.T) {
	cases := []struct {
		name        string
		metadata    map[string]string
		wantMeta    *temporalMetadata
		authParams  map[string]string
		resolvedEnv map[string]string
		wantErr     bool
	}{
		{
			name: "empty queue name",
			metadata: map[string]string{
				"endpoint":  "test:7233",
				"namespace": "default",
			},
			wantMeta: &temporalMetadata{
				Endpoint:                  "test:7233",
				Namespace:                 "default",
				QueueName:                 "",
				TargetQueueSize:           5,
				ActivationTargetQueueSize: 0,
				AllActive:                 true,
				Unversioned:               true,
			},
			wantErr: true,
		},
		{
			name: "empty namespace",
			metadata: map[string]string{
				"endpoint":  "test:7233",
				"queueName": "testxx",
			},
			wantMeta: &temporalMetadata{
				Endpoint:                  "test:7233",
				Namespace:                 "default",
				QueueName:                 "testxx",
				TargetQueueSize:           5,
				ActivationTargetQueueSize: 0,
				AllActive:                 true,
				Unversioned:               true,
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
				Endpoint:                  "test:7233",
				Namespace:                 "default",
				QueueName:                 "testxx",
				TargetQueueSize:           5,
				ActivationTargetQueueSize: 12,
				AllActive:                 true,
				Unversioned:               true,
			},
			wantErr: false,
		},
		{
			name: "apiKey should not be empty",
			metadata: map[string]string{
				"endpoint":  "test:7233",
				"namespace": "default",
				"queueName": "testxx",
			},
			wantMeta: &temporalMetadata{
				Endpoint:                  "test:7233",
				Namespace:                 "default",
				QueueName:                 "testxx",
				TargetQueueSize:           5,
				ActivationTargetQueueSize: 0,
				AllActive:                 true,
				Unversioned:               true,
				APIKey:                    "test01",
			},
			authParams: map[string]string{
				"apiKey": "test01",
			},
			wantErr: false,
		},
		{
			name: "queue type should not be empty",
			metadata: map[string]string{
				"endpoint":   "test:7233",
				"namespace":  "default",
				"queueName":  "testxx",
				"queueTypes": "workflow,activity",
			},
			wantMeta: &temporalMetadata{
				Endpoint:                  "test:7233",
				Namespace:                 "default",
				QueueName:                 "testxx",
				TargetQueueSize:           5,
				ActivationTargetQueueSize: 0,
				AllActive:                 true,
				Unversioned:               true,
				QueueTypes:                []string{"workflow", "activity"},
			},
			wantErr: false,
		},
		{
			name: "read config from env",
			resolvedEnv: map[string]string{
				"endpoint":  "test:7233",
				"namespace": "default",
				"queueName": "testxx",
			},
			metadata: map[string]string{
				"endpointFromEnv":  "endpoint",
				"namespaceFromEnv": "namespace",
				"queueNameFromEnv": "queueName",
			},
			wantMeta: &temporalMetadata{
				Endpoint:                  "test:7233",
				Namespace:                 "default",
				QueueName:                 "testxx",
				TargetQueueSize:           5,
				ActivationTargetQueueSize: 0,
				AllActive:                 true,
				Unversioned:               true,
				APIKey:                    "test01",
			},
			authParams: map[string]string{
				"apiKey": "test01",
			},
			wantErr: false,
		},
	}

	for _, testCase := range cases {
		c := testCase
		t.Run(c.name, func(t *testing.T) {
			config := &scalersconfig.ScalerConfig{
				TriggerMetadata: c.metadata,
				AuthParams:      c.authParams,
				ResolvedEnv:     c.resolvedEnv,
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

func TestTemporalDefaultQueueTypes(t *testing.T) {
	metadata, err := parseTemporalMetadata(&scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{
			"endpoint": "localhost:7233", "queueName": "testcc",
		},
	}, logger)

	assert.NoError(t, err, "error should be nil")
	assert.Empty(t, metadata.QueueTypes, "queueTypes should be empty")

	assert.Len(t, getQueueTypes(metadata.QueueTypes), 3, "all queue types should be there")

	metadata.QueueTypes = []string{"workflow"}
	assert.Len(t, getQueueTypes(metadata.QueueTypes), 1, "only one type should be there")
}
