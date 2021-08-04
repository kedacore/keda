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

type workloadIsActiveTestData struct {
	metadata map[string]string
	podCount int
	active   bool
}

var isActiveWorkloadTestDataset = []workloadIsActiveTestData{
	// "podSelector": "app=demo", "namespace": "test"
	{parseWorkloadMetadataTestDataset[0].metadata, 0, false},
	{parseWorkloadMetadataTestDataset[0].metadata, 1, false},
	{parseWorkloadMetadataTestDataset[0].metadata, 15, false},
	// "podSelector": "app=demo"
	{parseWorkloadMetadataTestDataset[1].metadata, 0, false},
	{parseWorkloadMetadataTestDataset[1].metadata, 1, true},
	{parseWorkloadMetadataTestDataset[1].metadata, 15, true},
}

func TestWorkloadIsActive(t *testing.T) {
	for _, testData := range isActiveWorkloadTestDataset {
		s, _ := NewKubernetesWorkloadScaler(
			fake.NewFakeClient(createPodlist(testData.podCount)),
			&ScalerConfig{
				TriggerMetadata:   testData.metadata,
				AuthParams:        map[string]string{},
				GlobalHTTPTimeout: 1000 * time.Millisecond,
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
	metadata map[string]string
	name     string
}

var getMetricSpecForScalingTestDataset = []workloadGetMetricSpecForScalingTestData{
	// "podSelector": "app=demo", "namespace": "test"
	{parseWorkloadMetadataTestDataset[0].metadata, "workload-test-app=demo"},
	// "podSelector": "app=demo", "namespace": ""
	{parseWorkloadMetadataTestDataset[1].metadata, "workload--app=demo"},
	// "podSelector": "app=demo"
	{parseWorkloadMetadataTestDataset[2].metadata, "workload--app=demo"},
	// "podSelector": "app=demo", "namespace": ""
	{parseWorkloadMetadataTestDataset[3].metadata, "workload-test-appin-demo1-demo2-"},
	// "podSelector": "app=demo"
	{parseWorkloadMetadataTestDataset[4].metadata, "workload-test-appin-demo1-demo2--deployin-deploy1-deploy2-"},
}

func TestWorkloadGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range getMetricSpecForScalingTestDataset {
		s, _ := NewKubernetesWorkloadScaler(
			fake.NewFakeClient(),
			&ScalerConfig{
				TriggerMetadata:   testData.metadata,
				AuthParams:        map[string]string{},
				GlobalHTTPTimeout: 1000 * time.Millisecond,
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
