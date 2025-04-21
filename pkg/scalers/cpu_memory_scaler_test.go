package scalers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseCPUMemoryMetadataTestData struct {
	metricType v2.MetricTargetType
	metadata   map[string]string
	isError    bool
}

var validCPUMemoryMetadata = map[string]string{
	"value": "50",
}
var validContainerCPUMemoryMetadata = map[string]string{
	"value":         "50",
	"containerName": "foo",
}

var testCPUMemoryMetadata = []parseCPUMemoryMetadataTestData{
	{"", map[string]string{}, true},
	{v2.UtilizationMetricType, map[string]string{"value": "50"}, false},
	{v2.AverageValueMetricType, map[string]string{"value": "50"}, false},
	{v2.ValueMetricType, map[string]string{"value": "50"}, true},
	{"", map[string]string{"value": ""}, true},
	{"", map[string]string{}, true},
}

func TestCPUMemoryParseMetadata(t *testing.T) {
	for i, testData := range testCPUMemoryMetadata {
		config := &scalersconfig.ScalerConfig{
			TriggerMetadata: testData.metadata,
			MetricType:      testData.metricType,
		}
		_, err := parseResourceMetadata(config)
		if err != nil && !testData.isError {
			t.Errorf("Test case %d: Expected success but got error: %v", i, err)
		}
		if testData.isError && err == nil {
			t.Errorf("Test case %d: Expected error but got success", i)
		}
	}
}

func TestGetMetricSpecForScaling(t *testing.T) {
	config := &scalersconfig.ScalerConfig{
		TriggerMetadata: validCPUMemoryMetadata,
		MetricType:      v2.UtilizationMetricType,
	}
	scaler, _ := NewCPUMemoryScaler(v1.ResourceCPU, config)
	metricSpec := scaler.GetMetricSpecForScaling(context.Background())

	assert.Equal(t, metricSpec[0].Type, v2.ResourceMetricSourceType)
	assert.Equal(t, metricSpec[0].Resource.Name, v1.ResourceCPU)
	assert.Equal(t, metricSpec[0].Resource.Target.Type, v2.UtilizationMetricType)
}

func TestGetContainerMetricSpecForScaling(t *testing.T) {
	config := &scalersconfig.ScalerConfig{
		TriggerMetadata: validContainerCPUMemoryMetadata,
		MetricType:      v2.UtilizationMetricType,
	}
	scaler, _ := NewCPUMemoryScaler(v1.ResourceCPU, config)
	metricSpec := scaler.GetMetricSpecForScaling(context.Background())

	assert.Equal(t, metricSpec[0].Type, v2.ContainerResourceMetricSourceType)
	assert.Equal(t, metricSpec[0].ContainerResource.Name, v1.ResourceCPU)
	assert.Equal(t, metricSpec[0].ContainerResource.Target.Type, v2.UtilizationMetricType)
	assert.Equal(t, metricSpec[0].ContainerResource.Container, validContainerCPUMemoryMetadata["containerName"])
}
