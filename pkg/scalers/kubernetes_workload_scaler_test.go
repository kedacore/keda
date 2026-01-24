package scalers

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type workloadMetadataTestData struct {
	metadata  map[string]string
	namespace string
	isError   bool
}

var parseWorkloadMetadataTestDataset = []workloadMetadataTestData{
	{map[string]string{"value": "1", "podSelector": "app=demo"}, "test", false},
	{map[string]string{"value": "1", "podSelector": "app=demo"}, "default", false},
	{map[string]string{"value": "1", "podSelector": "app in (demo1, demo2)"}, "test", false},
	{map[string]string{"value": "1", "podSelector": "app in (demo1, demo2),deploy in (deploy1, deploy2)"}, "test", false},
	{map[string]string{"podSelector": "app=demo"}, "test", true},
	{map[string]string{"podSelector": "app=demo"}, "default", true},
	{map[string]string{"value": "1"}, "test", true},
	{map[string]string{"value": "1"}, "default", true},
	{map[string]string{"value": "a", "podSelector": "app=demo"}, "test", true},
	{map[string]string{"value": "a", "podSelector": "app=demo"}, "default", true},
	{map[string]string{"value": "0", "podSelector": "app=demo"}, "test", true},
	{map[string]string{"value": "0", "podSelector": "app=demo"}, "default", true},
	{map[string]string{"value": "1", "activationValue": "aa", "podSelector": "app=demo"}, "test", true},
	{map[string]string{"value": "1", "podSelector": "app=demo", "namespace": "default"}, "test", false},
}

func TestParseWorkloadMetadata(t *testing.T) {
	for _, testData := range parseWorkloadMetadataTestDataset {
		_, err := NewKubernetesWorkloadScaler(
			fake.NewClientBuilder().Build(),
			&scalersconfig.ScalerConfig{
				TriggerMetadata:         testData.metadata,
				ScalableObjectNamespace: testData.namespace,
			},
		)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

type workloadIsActiveTestData struct {
	metadata  map[string]string
	namespace string
	podCount  int
	active    bool
}

var isActiveWorkloadTestDataset = []workloadIsActiveTestData{
	// "metadata": {"value": "1", "podSelector": "app=demo"}, "namespace": "test"
	{parseWorkloadMetadataTestDataset[0].metadata, parseWorkloadMetadataTestDataset[0].namespace, 0, false},
	{parseWorkloadMetadataTestDataset[0].metadata, parseWorkloadMetadataTestDataset[0].namespace, 1, false},
	{parseWorkloadMetadataTestDataset[0].metadata, parseWorkloadMetadataTestDataset[0].namespace, 15, false},
	// "metadata": {"value": "1", "podSelector": "app=demo"}, "namespace": "default"
	{parseWorkloadMetadataTestDataset[1].metadata, parseWorkloadMetadataTestDataset[1].namespace, 0, false},
	{parseWorkloadMetadataTestDataset[1].metadata, parseWorkloadMetadataTestDataset[1].namespace, 1, true},
	{parseWorkloadMetadataTestDataset[1].metadata, parseWorkloadMetadataTestDataset[1].namespace, 15, true},
	// "metadata": {"value": "1", "podSelector": "app=demo", "namespace": "default"}, "namespace": "test"
	{parseWorkloadMetadataTestDataset[len(parseWorkloadMetadataTestDataset)-1].metadata, parseWorkloadMetadataTestDataset[len(parseWorkloadMetadataTestDataset)-1].namespace, 0, false},
	{parseWorkloadMetadataTestDataset[len(parseWorkloadMetadataTestDataset)-1].metadata, parseWorkloadMetadataTestDataset[len(parseWorkloadMetadataTestDataset)-1].namespace, 1, true},
	{parseWorkloadMetadataTestDataset[len(parseWorkloadMetadataTestDataset)-1].metadata, parseWorkloadMetadataTestDataset[len(parseWorkloadMetadataTestDataset)-1].namespace, 15, true},
}

func TestWorkloadIsActive(t *testing.T) {
	for _, testData := range isActiveWorkloadTestDataset {
		s, err := NewKubernetesWorkloadScaler(
			fake.NewClientBuilder().WithRuntimeObjects(createPodlist(testData.podCount, "default")).Build(),
			&scalersconfig.ScalerConfig{
				TriggerMetadata:         testData.metadata,
				AuthParams:              map[string]string{},
				GlobalHTTPTimeout:       1000 * time.Millisecond,
				ScalableObjectNamespace: testData.namespace,
			},
		)
		if err != nil {
			t.Error("Error creating scaler", err)
			continue
		}
		_, isActive, _ := s.GetMetricsAndActivity(context.TODO(), "Metric")
		if testData.active && !isActive {
			t.Error("Expected active but got inactive")
		}
		if !testData.active && isActive {
			t.Error("Expected inactive but got active")
		}
	}
}

type workloadGetMetricSpecForScalingTestData struct {
	metadata     map[string]string
	namespace    string
	triggerIndex int
	name         string
}

var getMetricSpecForScalingTestDataset = []workloadGetMetricSpecForScalingTestData{
	// "podSelector": "app=demo", "namespace": "test"
	{parseWorkloadMetadataTestDataset[0].metadata, parseWorkloadMetadataTestDataset[0].namespace, 0, "s0-workload-test"},
	// "podSelector": "app=demo", "namespace": "default"
	{parseWorkloadMetadataTestDataset[1].metadata, parseWorkloadMetadataTestDataset[1].namespace, 1, "s1-workload-default"},
	// "podSelector": "app in (demo1, demo2)", "namespace": "test"
	{parseWorkloadMetadataTestDataset[2].metadata, parseWorkloadMetadataTestDataset[2].namespace, 2, "s2-workload-test"},
	// "podSelector": "app in (demo1, demo2),deploy in (deploy1, deploy2)", "namespace": "test"
	{parseWorkloadMetadataTestDataset[3].metadata, parseWorkloadMetadataTestDataset[3].namespace, 3, "s3-workload-test"},
}

func TestWorkloadGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range getMetricSpecForScalingTestDataset {
		s, err := NewKubernetesWorkloadScaler(
			fake.NewClientBuilder().Build(),
			&scalersconfig.ScalerConfig{
				TriggerMetadata:         testData.metadata,
				AuthParams:              map[string]string{},
				GlobalHTTPTimeout:       1000 * time.Millisecond,
				ScalableObjectNamespace: testData.namespace,
				TriggerIndex:            testData.triggerIndex,
			},
		)
		if err != nil {
			t.Error("Error creating scaler", err)
			continue
		}
		metric := s.GetMetricSpecForScaling(context.Background())

		if metric[0].External.Metric.Name != testData.name {
			t.Errorf("Expected '%s' as metric name and got '%s'", testData.name, metric[0].External.Metric.Name)
		}
	}
}

func createPodlist(count int, namespace string) *v1.PodList {
	list := &v1.PodList{}
	for i := 0; i < count; i++ {
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        fmt.Sprintf("demo-pod-v%d", i),
				Namespace:   namespace,
				Annotations: map[string]string{},
				Labels: map[string]string{
					"app": "demo",
				},
			},
		}
		list.Items = append(list.Items, *pod)
	}
	return list
}

func TestWorkloadPhase(t *testing.T) {
	phases := map[v1.PodPhase]bool{
		v1.PodRunning:   true,
		v1.PodSucceeded: false,
		v1.PodFailed:    false,
		v1.PodUnknown:   true,
		v1.PodPending:   true,
	}
	for phase, active := range phases {
		list := &v1.PodList{}
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        strings.ToLower(fmt.Sprintf("phase-%s", phase)),
				Namespace:   "default",
				Annotations: map[string]string{},
				Labels: map[string]string{
					"app": "testphases",
				},
			},
			Status: v1.PodStatus{
				Phase: phase,
			},
		}
		list.Items = append(list.Items, *pod)
		s, err := NewKubernetesWorkloadScaler(
			fake.NewClientBuilder().WithRuntimeObjects(list).Build(),
			&scalersconfig.ScalerConfig{
				TriggerMetadata: map[string]string{
					"podSelector": "app=testphases",
					"value":       "1",
				},
				AuthParams:              map[string]string{},
				GlobalHTTPTimeout:       1000 * time.Millisecond,
				ScalableObjectNamespace: "default",
			},
		)
		if err != nil {
			t.Errorf("Failed to create test scaler -- %v", err)
		}
		_, isActive, err := s.GetMetricsAndActivity(context.TODO(), "Metric")
		if err != nil {
			t.Errorf("Failed to count active -- %v", err)
		}
		if active && !isActive {
			t.Errorf("Expected active for phase %s but got inactive", phase)
		}
		if !active && isActive {
			t.Errorf("Expected inactive for phase %s but got active", phase)
		}
	}
}
