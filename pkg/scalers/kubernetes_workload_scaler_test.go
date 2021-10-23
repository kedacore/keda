package scalers

import (
	"context"
	"fmt"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
}

func TestParseWorkloadMetadata(t *testing.T) {
	for _, testData := range parseWorkloadMetadataTestDataset {
		_, err := parseWorkloadMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, Namespace: testData.namespace})
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
	// "podSelector": "app=demo", "namespace": "test"
	{parseWorkloadMetadataTestDataset[0].metadata, parseWorkloadMetadataTestDataset[0].namespace, 0, false},
	{parseWorkloadMetadataTestDataset[0].metadata, parseWorkloadMetadataTestDataset[0].namespace, 1, false},
	{parseWorkloadMetadataTestDataset[0].metadata, parseWorkloadMetadataTestDataset[0].namespace, 15, false},
	// "podSelector": "app=demo", "namespace": "default"
	{parseWorkloadMetadataTestDataset[1].metadata, parseWorkloadMetadataTestDataset[1].namespace, 0, false},
	{parseWorkloadMetadataTestDataset[1].metadata, parseWorkloadMetadataTestDataset[1].namespace, 1, true},
	{parseWorkloadMetadataTestDataset[1].metadata, parseWorkloadMetadataTestDataset[1].namespace, 15, true},
}

func TestWorkloadIsActive(t *testing.T) {
	for _, testData := range isActiveWorkloadTestDataset {
		s, _ := NewKubernetesWorkloadScaler(
			fake.NewFakeClient(createPodlist(testData.podCount)),
			&ScalerConfig{
				TriggerMetadata:   testData.metadata,
				AuthParams:        map[string]string{},
				GlobalHTTPTimeout: 1000 * time.Millisecond,
				Namespace:         testData.namespace,
			},
		)
		isActive, _ := s.IsActive(context.TODO())
		if testData.active && !isActive {
			t.Error("Expected active but got inactive")
		}
		if !testData.active && isActive {
			t.Error("Expected inactive but got active")
		}
	}
}

type workloadGetMetricSpecForScalingTestData struct {
	metadata    map[string]string
	namespace   string
	scalerIndex int
	name        string
}

var getMetricSpecForScalingTestDataset = []workloadGetMetricSpecForScalingTestData{
	// "podSelector": "app=demo", "namespace": "test"
	{parseWorkloadMetadataTestDataset[0].metadata, parseWorkloadMetadataTestDataset[0].namespace, 0, "s0-workload-test-app=demo"},
	// "podSelector": "app=demo", "namespace": "default"
	{parseWorkloadMetadataTestDataset[1].metadata, parseWorkloadMetadataTestDataset[1].namespace, 1, "s1-workload-default-app=demo"},
	// "podSelector": "app in (demo1, demo2)", "namespace": "test"
	{parseWorkloadMetadataTestDataset[2].metadata, parseWorkloadMetadataTestDataset[2].namespace, 2, "s2-workload-test-appin-demo1-demo2-"},
	// "podSelector": "app in (demo1, demo2),deploy in (deploy1, deploy2)", "namespace": "test"
	{parseWorkloadMetadataTestDataset[3].metadata, parseWorkloadMetadataTestDataset[3].namespace, 3, "s3-workload-test-appin-demo1-demo2--deployin-deploy1-deploy2-"},
}

func TestWorkloadGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range getMetricSpecForScalingTestDataset {
		s, _ := NewKubernetesWorkloadScaler(
			fake.NewFakeClient(),
			&ScalerConfig{
				TriggerMetadata:   testData.metadata,
				AuthParams:        map[string]string{},
				GlobalHTTPTimeout: 1000 * time.Millisecond,
				Namespace:         testData.namespace,
				ScalerIndex:       testData.scalerIndex,
			},
		)
		metric := s.GetMetricSpecForScaling()

		if metric[0].External.Metric.Name != testData.name {
			t.Errorf("Expected '%s' as metric name and got '%s'", testData.name, metric[0].External.Metric.Name)
		}
	}
}

func createPodlist(count int) *v1.PodList {
	list := &v1.PodList{}
	for i := 0; i < count; i++ {
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        fmt.Sprintf("demo-pod-v%d", i),
				Namespace:   "default",
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
