package scalers

import (
	"testing"
)

type workloadMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

var parseWorkloadMetadataTestDataset = []workloadMetadataTestData{
	{map[string]string{"value": "1", "podSelector": "app=demo", "namespace": "test"}, false},
	{map[string]string{"value": "1", "podSelector": "app=demo", "namespace": ""}, false},
	{map[string]string{"value": "1", "podSelector": "app=demo"}, false},
	{map[string]string{"value": "1", "podSelector": "app in (demo1, demo2)", "namespace": "test"}, false},
	{map[string]string{"value": "1", "podSelector": "app in (demo1, demo2),deploy in (deploy1, deploy2)", "namespace": "test"}, false},
	{map[string]string{"podSelector": "app=demo", "namespace": "test"}, true},
	{map[string]string{"podSelector": "app=demo", "namespace": ""}, true},
	{map[string]string{"podSelector": "app=demo"}, true},
	{map[string]string{"value": "1", "namespace": "test"}, true},
	{map[string]string{"value": "1", "namespace": ""}, true},
	{map[string]string{"value": "1"}, true},
	{map[string]string{"value": "a", "podSelector": "app=demo", "namespace": "test"}, true},
	{map[string]string{"value": "a", "podSelector": "app=demo", "namespace": ""}, true},
	{map[string]string{"value": "a", "podSelector": "app=demo"}, true},
	{map[string]string{"value": "0", "podSelector": "app=demo", "namespace": "test"}, true},
	{map[string]string{"value": "0", "podSelector": "app=demo", "namespace": ""}, true},
	{map[string]string{"value": "0", "podSelector": "app=demo"}, true},
}

func TestParseWorkloadMetadata(t *testing.T) {
	for _, testData := range parseWorkloadMetadataTestDataset {
		_, err := parseWorkloadMetadata(&ScalerConfig{TriggerMetadata: testData.metadata})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}
