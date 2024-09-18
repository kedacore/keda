package scalers

import (
	"context"
	"fmt"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"

	"github.com/go-logr/logr"
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

// A complete valid metadata example for reference
var validCPUMemoryMetadata = map[string]string{
	"type":            "Utilization",
	"value":           "50",
	"activationValue": "40",
}
var validContainerCPUMemoryMetadata = map[string]string{
	"type":          "Utilization",
	"value":         "50",
	"containerName": "foo",
}

var testCPUMemoryMetadata = []parseCPUMemoryMetadataTestData{
	{"", map[string]string{}, true},
	{"", validCPUMemoryMetadata, false},
	{"", validContainerCPUMemoryMetadata, false},
	{"", map[string]string{"type": "Utilization", "value": "50"}, false},
	{v2.UtilizationMetricType, map[string]string{"value": "50"}, false},
	{"", map[string]string{"type": "AverageValue", "value": "50"}, false},
	{v2.AverageValueMetricType, map[string]string{"value": "50"}, false},
	{"", map[string]string{"type": "AverageValue", "value": "50", "activationValue": "40"}, false},
	{"", map[string]string{"type": "Value", "value": "50"}, true},
	{v2.ValueMetricType, map[string]string{"value": "50"}, true},
	{"", map[string]string{"type": "AverageValue"}, true},
	{"", map[string]string{"type": "xxx", "value": "50"}, true},
}

func TestCPUMemoryParseMetadata(t *testing.T) {
	for _, testData := range testCPUMemoryMetadata {
		config := &scalersconfig.ScalerConfig{
			TriggerMetadata: testData.metadata,
			MetricType:      testData.metricType,
		}
		_, err := parseResourceMetadata(config, logr.Discard())
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestGetMetricSpecForScaling(t *testing.T) {
	// Using trigger.metadata.type field for type
	config := &scalersconfig.ScalerConfig{
		TriggerMetadata: validCPUMemoryMetadata,
	}
	kubeClient := fake.NewFakeClient()
	scaler, _ := NewCPUMemoryScaler(v1.ResourceCPU, config, kubeClient)
	metricSpec := scaler.GetMetricSpecForScaling(context.Background())

	assert.Equal(t, metricSpec[0].Type, v2.ResourceMetricSourceType)
	assert.Equal(t, metricSpec[0].Resource.Name, v1.ResourceCPU)
	assert.Equal(t, metricSpec[0].Resource.Target.Type, v2.UtilizationMetricType)

	// Using trigger.metricType field for type
	config = &scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{"value": "50"},
		MetricType:      v2.UtilizationMetricType,
	}
	scaler, _ = NewCPUMemoryScaler(v1.ResourceCPU, config, kubeClient)
	metricSpec = scaler.GetMetricSpecForScaling(context.Background())

	assert.Equal(t, metricSpec[0].Type, v2.ResourceMetricSourceType)
	assert.Equal(t, metricSpec[0].Resource.Name, v1.ResourceCPU)
	assert.Equal(t, metricSpec[0].Resource.Target.Type, v2.UtilizationMetricType)
}

func TestGetContainerMetricSpecForScaling(t *testing.T) {
	// Using trigger.metadata.type field for type
	config := &scalersconfig.ScalerConfig{
		TriggerMetadata: validContainerCPUMemoryMetadata,
	}
	kubeClient := fake.NewFakeClient()
	scaler, _ := NewCPUMemoryScaler(v1.ResourceCPU, config, kubeClient)
	metricSpec := scaler.GetMetricSpecForScaling(context.Background())

	assert.Equal(t, metricSpec[0].Type, v2.ContainerResourceMetricSourceType)
	assert.Equal(t, metricSpec[0].ContainerResource.Name, v1.ResourceCPU)
	assert.Equal(t, metricSpec[0].ContainerResource.Target.Type, v2.UtilizationMetricType)
	assert.Equal(t, metricSpec[0].ContainerResource.Container, validContainerCPUMemoryMetadata["containerName"])

	// Using trigger.metricType field for type
	config = &scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{"value": "50", "containerName": "bar"},
		MetricType:      v2.UtilizationMetricType,
	}
	scaler, _ = NewCPUMemoryScaler(v1.ResourceCPU, config, kubeClient)
	metricSpec = scaler.GetMetricSpecForScaling(context.Background())

	assert.Equal(t, metricSpec[0].Type, v2.ContainerResourceMetricSourceType)
	assert.Equal(t, metricSpec[0].ContainerResource.Name, v1.ResourceCPU)
	assert.Equal(t, metricSpec[0].ContainerResource.Target.Type, v2.UtilizationMetricType)
	assert.Equal(t, metricSpec[0].ContainerResource.Container, "bar")
}

func createScaledObject() *kedav1alpha1.ScaledObject {
	maxReplicas := int32(3)
	minReplicas := int32(0)
	pollingInterval := int32(10)
	return &kedav1alpha1.ScaledObject{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "keda.sh/v1alpha1",
			Kind:       "ScaledObject",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-name",
			Namespace: "test-namespace",
		},
		Spec: kedav1alpha1.ScaledObjectSpec{
			MaxReplicaCount: &maxReplicas,
			MinReplicaCount: &minReplicas,
			PollingInterval: &pollingInterval,
			ScaleTargetRef: &kedav1alpha1.ScaleTarget{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       "test-deployment",
			},
			Triggers: []kedav1alpha1.ScaleTriggers{
				{
					Type: "cpu",
					Metadata: map[string]string{
						"activationValue": "500",
						"value":           "800",
					},
					MetricType: v2.UtilizationMetricType,
				},
			},
		},
		Status: kedav1alpha1.ScaledObjectStatus{
			HpaName: "keda-hpa-test-name",
		},
	}
}

func createHPAWithAverageUtilization(averageUtilization int32) (*v2.HorizontalPodAutoscaler, error) {
	minReplicas := int32(1)
	averageValue, err := resource.ParseQuantity("800m")
	if err != nil {
		fmt.Errorf("Error parsing quantity: %s", err)
		return nil, err
	}

	return &v2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keda-hpa-test-name",
			Namespace: "test-namespace",
		},
		Spec: v2.HorizontalPodAutoscalerSpec{
			MaxReplicas: 3,
			MinReplicas: &minReplicas,
			Metrics: []v2.MetricSpec{

				{
					Type: v2.ResourceMetricSourceType,
					Resource: &v2.ResourceMetricSource{
						Name: v1.ResourceCPU,
						Target: v2.MetricTarget{
							AverageUtilization: &averageUtilization,
							Type:               v2.UtilizationMetricType,
						},
					},
				},
			},
		},
		Status: v2.HorizontalPodAutoscalerStatus{
			CurrentMetrics: []v2.MetricStatus{
				{
					Type: v2.ResourceMetricSourceType,
					Resource: &v2.ResourceMetricStatus{
						Name: v1.ResourceCPU,
						Current: v2.MetricValueStatus{
							AverageUtilization: &averageUtilization,
							AverageValue:       &averageValue,
						},
					},
				},
			},
		},
	}, nil
}

func TestGetMetricsAndActivity_IsActive(t *testing.T) {
	config := &scalersconfig.ScalerConfig{
		TriggerMetadata:         validCPUMemoryMetadata,
		ScalableObjectType:      "ScaledObject",
		ScalableObjectName:      "test-name",
		ScalableObjectNamespace: "test-namespace",
	}

	hpa, err := createHPAWithAverageUtilization(50)
	if err != nil {
		t.Errorf("Error creating HPA: %s", err)
	}

	kubeClient := fake.NewClientBuilder().WithObjects(hpa, createScaledObject()).Build()
	scaler, _ := NewCPUMemoryScaler(v1.ResourceCPU, config, kubeClient)

	_, isActive, _ := scaler.GetMetricsAndActivity(context.Background(), "cpu")
	assert.Equal(t, isActive, true)
}

func TestGetMetricsAndActivity_IsNotActive(t *testing.T) {
	config := &scalersconfig.ScalerConfig{
		TriggerMetadata:         validCPUMemoryMetadata,
		ScalableObjectType:      "ScaledObject",
		ScalableObjectName:      "test-name",
		ScalableObjectNamespace: "test-namespace",
	}

	hpa, err := createHPAWithAverageUtilization(30)
	if err != nil {
		t.Errorf("Error creating HPA: %s", err)
	}

	kubeClient := fake.NewClientBuilder().WithRuntimeObjects(hpa, createScaledObject()).Build()
	scaler, _ := NewCPUMemoryScaler(v1.ResourceCPU, config, kubeClient)

	_, isActive, _ := scaler.GetMetricsAndActivity(context.Background(), "cpu")
	assert.Equal(t, isActive, false)
}
