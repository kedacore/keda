package scalers

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
)

type parseTemporalMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

type temporalMetricIdentifier struct {
	metadataTestData *parseTemporalMetadataTestData
	scalerIndex      int
	name             string
}

var testTemporalMetadata = []parseTemporalMetadataTestData{
	{map[string]string{}, map[string]string{}, true},
	// all properly formed
	{map[string]string{"address": "http://localhost", "threshold": "100", "activationThreshold": "20"}, map[string]string{}, false},
	// missing address
	{map[string]string{"threshold": "100"}, map[string]string{}, true},
	// missing account
	{map[string]string{"address": "http://localhost", "threshold": "one"}, map[string]string{}, true},
	// malformed activationThreshold
	{map[string]string{"address": "http://localhost", "threshold": "100", "activationThreshold": "notanint"}, map[string]string{}, true},
	// missing threshold
}

var temporalMetricIdentifiers = []temporalMetricIdentifier{
	{&testTemporalMetadata[1], 0, "s0-temporal"},
	{&testTemporalMetadata[1], 1, "s1-temporal"},
}

func TestTemporalParseMetadata(t *testing.T) {
	for _, testData := range testTemporalMetadata {
		_, err := parseTemporalMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams}, logr.Discard())
		if err != nil && !testData.isError {
			fmt.Printf("X: %s", testData.metadata)
			t.Error("Expected success but got error", err)
		}
		// if testData.isError && err == nil {
		//	fmt.Printf("X: %s", testData.metadata)
		//	t.Error("Expected error but got success")
		//}
	}
}
func TestTemporalGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range temporalMetricIdentifiers {
		meta, err := parseTemporalMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.metadataTestData.authParams, ScalerIndex: testData.scalerIndex}, logr.Discard())
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockTemporalScaler := temporalScaler{
			metadata: meta,
		}

		metricSpec := mockTemporalScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
