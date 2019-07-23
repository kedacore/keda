package scalers

import (
	"testing"
)

type kubernetesEventMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

var testk8sEventMetadata = []kubernetesEventMetadataTestData{
	// properly formed metadata
	{map[string]string{"scaleDownPeriodSeconds": "10", "numberOfEvents": "5", "fieldSelector": ""}, false},
	// malformed scaleDownPeriodSeconds
	{map[string]string{"scaleDownPeriodSeconds": "AA", "numberOfEvents": "5"}, true},
	// malformed numberOfEvents
	{map[string]string{"scaleDownPeriodSeconds": "5", "numberOfEvents": "AA"}, true},
}

func TestKubernetesEventParseMetadata(t *testing.T) {
	for _, testData := range testk8sEventMetadata {
		_, err := parseKubernetesEventsMetadata(testData.metadata)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}
