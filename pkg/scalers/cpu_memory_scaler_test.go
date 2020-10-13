package scalers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/api/core/v1"
)

type parseCPUMemoryMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

// A complete valid metadata example for reference
var validCPUMemoryMetadata = map[string]string{
	"type":  "Utilization",
	"value": "50",
}

var testCPUMemoryMetadata = []parseCPUMemoryMetadataTestData{
	{map[string]string{}, true},
	{validCPUMemoryMetadata, false},
	{map[string]string{"type": "Utilization", "value": "50"}, false},
	{map[string]string{"type": "Value", "value": "50"}, false},
	{map[string]string{"type": "AverageValue", "value": "50"}, false},
	{map[string]string{"type": "AverageValue"}, true},
	{map[string]string{"type": "xxx", "value": "50"}, true},
}

func TestCPUMemoryParseMetadata(t *testing.T) {
	for _, testData := range testCPUMemoryMetadata {
		config := &ScalerConfig{
			TriggerMetadata: testData.metadata,
		}
		_, err := parseResourceMetadata(config)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestGetMetricSpecForScaling(t *testing.T) {
	config := &ScalerConfig{
		TriggerMetadata: validCPUMemoryMetadata,
	}
	scaler, _ := NewCPUMemoryScaler(v1.ResourceCPU, config)
	metricSpec := scaler.GetMetricSpecForScaling()

	assert.Equal(t, metricSpec[0].Type, v2beta2.ResourceMetricSourceType)
	assert.Equal(t, metricSpec[0].Resource.Name, v1.ResourceCPU)
	assert.Equal(t, metricSpec[0].Resource.Target.Type, v2beta2.UtilizationMetricType)
}
