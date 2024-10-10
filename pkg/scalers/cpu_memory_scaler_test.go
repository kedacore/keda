package scalers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	metrics "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseCPUMemoryMetadataTestData struct {
	metricType v2.MetricTargetType
	metadata   map[string]string
	isError    bool
}

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

var testCPUMemoryMetadataActivationPresent = parseCPUMemoryMetadataTestData{
	metricType: v2.AverageValueMetricType,
	metadata:   map[string]string{"type": "AverageValue", "value": "50", "activationValue": "40"},
}

var testCPUMemoryMetadataActivationNotPresent = parseCPUMemoryMetadataTestData{
	metricType: v2.AverageValueMetricType,
	metadata:   map[string]string{"type": "AverageValue", "value": "50"},
}

var selectLabels = map[string]string{
	"app": "test-deployment",
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
			ScaleTargetGVKR: &kedav1alpha1.GroupVersionKindResource{
				Group:    "apps",
				Version:  "v1",
				Resource: "deployments",
				Kind:     "Deployment",
			},
		},
	}
}

func createDeployment() *appsv1.Deployment {
	replicas := int32(1)
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "test-namespace",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selectLabels,
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: selectLabels,
			},
		},
	}
	return deployment
}

func createPod(cpuRequest string) *v1.Pod {
	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment-1",
			Namespace: "test-namespace",
			Labels:    selectLabels,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: "test-container",
					Resources: v1.ResourceRequirements{
						Limits: v1.ResourceList{
							v1.ResourceCPU: resource.MustParse("600m"),
						},
						Requests: v1.ResourceList{
							v1.ResourceCPU: resource.MustParse(cpuRequest),
						},
					},
				},
			},
		},
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
		},
	}

	return pod
}

func createPodMetrics(cpuUsage string) *metrics.PodMetrics {
	err := metrics.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil
	}

	cpuQuantity, _ := resource.ParseQuantity(cpuUsage)
	return &metrics.PodMetrics{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "metrics.k8s.io/v1beta1",
			Kind:       "PodMetrics",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment-1",
			Namespace: "test-namespace",
			Labels:    selectLabels,
		},
		Containers: []metrics.ContainerMetrics{
			{
				Name: "test-container",
				Usage: v1.ResourceList{
					v1.ResourceCPU: cpuQuantity,
				},
			},
		},
	}
}

type mockPodMetricsesGetter struct {
}

func (m *mockPodMetricsesGetter) PodMetricses(namespace string) metricsv1beta1.PodMetricsInterface {
	return &mockPodMetricsInterface{}
}

type mockPodMetricsInterface struct {
	metricsv1beta1.PodMetricsExpansion
}

func (m *mockPodMetricsInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (*metrics.PodMetrics, error) {
	return nil, nil
}

func (m *mockPodMetricsInterface) List(ctx context.Context, opts metav1.ListOptions) (*metrics.PodMetricsList, error) {
	return &metrics.PodMetricsList{
		Items: []metrics.PodMetrics{
			*createPodMetrics("500m"),
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "metrics.k8s.io/v1beta1",
			Kind:       "PodMetricsList",
		},
	}, nil
}

func (m *mockPodMetricsInterface) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return nil, nil
}

func getMockMetricsClient() metricsv1beta1.PodMetricsesGetter {
	return &mockPodMetricsesGetter{}
}

func TestCPUMemoryParseMetadata(t *testing.T) {
	logger := logr.Discard()
	for i, testData := range testCPUMemoryMetadata {
		config := &scalersconfig.ScalerConfig{
			TriggerMetadata: testData.metadata,
			MetricType:      testData.metricType,
		}
		_, err := parseResourceMetadata(config, logger, fake.NewFakeClient())
		if err != nil && !testData.isError {
			t.Errorf("Test case %d: Expected success but got error: %v", i, err)
		}
		if testData.isError && err == nil {
			t.Errorf("Test case %d: Expected error but got success", i)
		}
	}

	// Test activation value present
	config := &scalersconfig.ScalerConfig{
		TriggerMetadata: testCPUMemoryMetadataActivationPresent.metadata,
		MetricType:      testCPUMemoryMetadataActivationPresent.metricType,
	}

	metadata, err := parseResourceMetadata(config, logger, fake.NewFakeClient())
	if err != nil {
		t.Errorf("Test case activation value present: Expected success but got error: %v", err)
	}
	if !metadata.ActivationAverageValue.Equal(resource.MustParse("40")) {
		t.Errorf("Test case activation value present: Expected activation value 40 but got %v", metadata.ActivationAverageValue)
	}

	// Test activation value not present
	config = &scalersconfig.ScalerConfig{
		TriggerMetadata: testCPUMemoryMetadataActivationNotPresent.metadata,
		MetricType:      testCPUMemoryMetadataActivationNotPresent.metricType,
	}

	metadata, err = parseResourceMetadata(config, logger, fake.NewFakeClient())
	if err != nil {
		t.Errorf("Test case activation value not present: Expected success but got error: %v", err)
	}
	if !metadata.ActivationAverageValue.Equal(resource.MustParse("0")) {
		t.Errorf("Test case activation value not present: Expected activation value 0 but got %v", metadata.ActivationAverageValue)
	}
}

func TestGetMetricSpecForScaling(t *testing.T) {
	// Using trigger.metadata.type field for type
	config := &scalersconfig.ScalerConfig{
		TriggerMetadata: validCPUMemoryMetadata,
	}
	kubeClient := fake.NewFakeClient()
	metricsClient := getMockMetricsClient()
	scaler, _ := NewCPUMemoryScaler(v1.ResourceCPU, config, kubeClient, metricsClient)
	metricSpec := scaler.GetMetricSpecForScaling(context.Background())

	assert.Equal(t, metricSpec[0].Type, v2.ResourceMetricSourceType)
	assert.Equal(t, metricSpec[0].Resource.Name, v1.ResourceCPU)
	assert.Equal(t, metricSpec[0].Resource.Target.Type, v2.UtilizationMetricType)

	// Using trigger.metricType field for type
	config = &scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{"value": "50"},
		MetricType:      v2.UtilizationMetricType,
	}
	scaler, _ = NewCPUMemoryScaler(v1.ResourceMemory, config, kubeClient, metricsClient)
	metricSpec = scaler.GetMetricSpecForScaling(context.Background())

	assert.Equal(t, metricSpec[0].Type, v2.ResourceMetricSourceType)
	assert.Equal(t, metricSpec[0].Resource.Name, v1.ResourceMemory)
	assert.Equal(t, metricSpec[0].Resource.Target.Type, v2.UtilizationMetricType)
}

func TestGetContainerMetricSpecForScaling(t *testing.T) {
	// Using trigger.metadata.type field for type
	config := &scalersconfig.ScalerConfig{
		TriggerMetadata: validContainerCPUMemoryMetadata,
	}
	kubeClient := fake.NewFakeClient()
	metricsClient := getMockMetricsClient()
	scaler, _ := NewCPUMemoryScaler(v1.ResourceCPU, config, kubeClient, metricsClient)
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
	scaler, _ = NewCPUMemoryScaler(v1.ResourceCPU, config, kubeClient, metricsClient)
	metricSpec = scaler.GetMetricSpecForScaling(context.Background())

	assert.Equal(t, metricSpec[0].Type, v2.ContainerResourceMetricSourceType)
	assert.Equal(t, metricSpec[0].ContainerResource.Name, v1.ResourceCPU)
	assert.Equal(t, metricSpec[0].ContainerResource.Target.Type, v2.UtilizationMetricType)
	assert.Equal(t, metricSpec[0].ContainerResource.Container, "bar")
}

func TestGetMetricsAndActivity_IsActive(t *testing.T) {
	config := &scalersconfig.ScalerConfig{
		TriggerMetadata:         validCPUMemoryMetadata,
		ScalableObjectType:      "ScaledObject",
		ScalableObjectName:      "test-name",
		ScalableObjectNamespace: "test-namespace",
	}

	deployment := createDeployment()
	pod := createPod("400m")

	err := kedav1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		t.Errorf("Error adding to scheme: %s", err)
		return
	}

	kubeClient := fake.NewClientBuilder().WithObjects(deployment, pod, createPodMetrics("500m"), createScaledObject()).WithScheme(scheme.Scheme).Build()
	metricsClient := getMockMetricsClient()
	scaler, _ := NewCPUMemoryScaler(v1.ResourceCPU, config, kubeClient, metricsClient)

	_, isActive, _ := scaler.GetMetricsAndActivity(context.Background(), "cpu")
	assert.Equal(t, true, isActive)
}

func TestGetMetricsAndActivity_IsNotActive(t *testing.T) {
	config := &scalersconfig.ScalerConfig{
		TriggerMetadata:         validCPUMemoryMetadata,
		ScalableObjectType:      "ScaledObject",
		ScalableObjectName:      "test-name",
		ScalableObjectNamespace: "test-namespace",
	}

	deployment := createDeployment()
	pod := createPod("2")

	err := kedav1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		t.Errorf("Error adding to scheme: %s", err)
		return
	}

	kubeClient := fake.NewClientBuilder().WithObjects(deployment, pod, createScaledObject()).WithScheme(scheme.Scheme).Build()
	metricsClient := getMockMetricsClient()
	scaler, _ := NewCPUMemoryScaler(v1.ResourceCPU, config, kubeClient, metricsClient)

	_, isActive, _ := scaler.GetMetricsAndActivity(context.Background(), "cpu")
	assert.Equal(t, isActive, false)
}
