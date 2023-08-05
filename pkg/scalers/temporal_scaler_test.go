package scalers

import (
	"context"
	"testing"
)

var (
	temporalEndpoint     = "localhost:7233"
	temporalNamespace    = "v2"
	temporalWorkflowName = "SayHello"
)

type parseTemporalMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

type temporalMetricIdentifier struct {
	metadataTestData *parseTemporalMetadataTestData
	scalerIndex      int
	name             string
}

var testTemporalMetadata = []parseTemporalMetadataTestData{
	// nothing passed
	{map[string]string{}, true},
	// Missing workflow, should fail
	{map[string]string{"endpoint": temporalEndpoint, "namespace": temporalNamespace}, true},
	// Missing namespace, should success
	{map[string]string{"endpoint": temporalEndpoint, "workflowName": temporalWorkflowName}, false},
	// Missing endpoint, should fail
	{map[string]string{"workflowName": temporalWorkflowName, "namespace": temporalNamespace}, true},
	// All good.
	{map[string]string{"endpoint": temporalEndpoint, "workflowName": temporalWorkflowName, "namespace": temporalNamespace}, false},
	// All good + activationLagThreshold
	{map[string]string{"endpoint": temporalEndpoint, "workflowName": temporalWorkflowName, "namespace": temporalNamespace, "activationTargetQueueSize": "10"}, false},
}

var temporalMetricIdentifiers = []temporalMetricIdentifier{
	{&testTemporalMetadata[4], 0, "s0-temporal-v2-SayHello"},
	{&testTemporalMetadata[4], 1, "s1-temporal-v2-SayHello"},
}

func TestTemporalParseMetadata(t *testing.T) {
	for _, testData := range testTemporalMetadata {
		_, err := parseTemporalMetadata(&ScalerConfig{TriggerMetadata: testData.metadata})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		} else if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestTemporalGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range temporalMetricIdentifiers {
		ctx := context.Background()
		meta, err := parseTemporalMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ScalerIndex: testData.scalerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockTemporalScaler := temporalWorkflowScaler{
			metadata: meta,
		}

		metricSpec := mockTemporalScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
