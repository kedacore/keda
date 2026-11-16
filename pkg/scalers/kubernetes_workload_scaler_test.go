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
	{map[string]string{"value": "1", "podSelector": "app=demo", "groupByNode": "true"}, "test", false},
	{map[string]string{"podSelector": "app=demo"}, "test", true},
	{map[string]string{"podSelector": "app=demo"}, "default", true},
	{map[string]string{"value": "1"}, "test", true},
	{map[string]string{"value": "1"}, "default", true},
	{map[string]string{"value": "a", "podSelector": "app=demo"}, "test", true},
	{map[string]string{"value": "a", "podSelector": "app=demo"}, "default", true},
	{map[string]string{"value": "0", "podSelector": "app=demo"}, "test", true},
	{map[string]string{"value": "0", "podSelector": "app=demo"}, "default", true},
	{map[string]string{"value": "1", "activationValue": "aa", "podSelector": "app=demo"}, "test", true},
	{map[string]string{"value": "1", "podSelector": "app=demo", "groupByNode": "invalid"}, "test", true},
	{map[string]string{"value": "1", "podSelector": "app=demo", "onlyReadyPods": "true"}, "test", false},
	{map[string]string{"value": "1", "podSelector": "app=demo", "onlyReadyPods": "false"}, "test", false},
	{map[string]string{"value": "1", "podSelector": "app=demo", "onlyReadyPods": "invalid"}, "test", true},
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
		s, err := NewKubernetesWorkloadScaler(
			fake.NewClientBuilder().WithRuntimeObjects(createPodlist(testData.podCount)).Build(),
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

func createPodlist(count int) *v1.PodList {
	list := &v1.PodList{}
	for i := range count {
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

func TestWorkloadGroupByNode(t *testing.T) {
	podList := &v1.PodList{
		Items: []v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "demo-pod-node-a-1",
					Namespace: "default",
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Spec: v1.PodSpec{
					NodeName: "node-a",
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "demo-pod-node-a-2",
					Namespace: "default",
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Spec: v1.PodSpec{
					NodeName: "node-a",
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "demo-pod-node-b",
					Namespace: "default",
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Spec: v1.PodSpec{
					NodeName: "node-b",
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "demo-pod-pending",
					Namespace: "default",
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodPending,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "demo-pod-failed",
					Namespace: "default",
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Spec: v1.PodSpec{
					NodeName: "node-c",
				},
				Status: v1.PodStatus{
					Phase: v1.PodFailed,
				},
			},
		},
	}

	testCases := []struct {
		name     string
		metadata map[string]string
		expected int64
	}{
		{
			name: "count all matching pods",
			metadata: map[string]string{
				"podSelector": "app=demo",
				"value":       "1",
			},
			expected: 4,
		},
		{
			name: "count unique nodes for scheduled matching pods",
			metadata: map[string]string{
				"podSelector": "app=demo",
				"value":       "1",
				"groupByNode": "true",
			},
			expected: 2,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s, err := NewKubernetesWorkloadScaler(
				fake.NewClientBuilder().WithRuntimeObjects(podList).Build(),
				&scalersconfig.ScalerConfig{
					TriggerMetadata:         testCase.metadata,
					ScalableObjectNamespace: "default",
				},
			)
			if err != nil {
				t.Fatalf("Error creating scaler: %v", err)
			}

			workloadScaler, ok := s.(*kubernetesWorkloadScaler)
			if !ok {
				t.Fatalf("Expected *kubernetesWorkloadScaler and got %T", s)
			}

			count, err := workloadScaler.getMetricValue(context.Background())
			if err != nil {
				t.Fatalf("Error getting metric value: %v", err)
			}

			if count != testCase.expected {
				t.Fatalf("Expected count %d and got %d", testCase.expected, count)
			}
		})
	}
}

func newWorkloadTestPod(name, nodeName string, phase v1.PodPhase, ready v1.ConditionStatus, terminating bool) v1.Pod {
	pod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Labels: map[string]string{
				"app": "demo",
			},
		},
		Spec: v1.PodSpec{
			NodeName: nodeName,
		},
		Status: v1.PodStatus{
			Phase: phase,
		},
	}
	if ready != "" {
		pod.Status.Conditions = []v1.PodCondition{{Type: v1.PodReady, Status: ready}}
	}
	if terminating {
		now := metav1.Now()
		pod.DeletionTimestamp = &now
		// the fake client refuses objects with a deletion timestamp but no finalizers
		pod.Finalizers = []string{"keda.sh/test"}
	}
	return pod
}

func TestWorkloadShouldCountPod(t *testing.T) {
	testCases := []struct {
		name             string
		pod              v1.Pod
		countAll         bool
		countIfReadyOnly bool
	}{
		{"running and ready", newWorkloadTestPod("p", "", v1.PodRunning, v1.ConditionTrue, false), true, true},
		{"running but not ready", newWorkloadTestPod("p", "", v1.PodRunning, v1.ConditionFalse, false), true, false},
		{"running without ready condition", newWorkloadTestPod("p", "", v1.PodRunning, "", false), true, false},
		{"pending", newWorkloadTestPod("p", "", v1.PodPending, "", false), true, false},
		{"unknown phase with stale ready condition", newWorkloadTestPod("p", "", v1.PodUnknown, v1.ConditionTrue, false), true, false},
		{"terminating but still ready", newWorkloadTestPod("p", "", v1.PodRunning, v1.ConditionTrue, true), true, false},
		{"failed", newWorkloadTestPod("p", "", v1.PodFailed, "", false), false, false},
		{"succeeded", newWorkloadTestPod("p", "", v1.PodSucceeded, "", false), false, false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := shouldCountPod(testCase.pod, false); got != testCase.countAll {
				t.Errorf("onlyReadyPods=false: expected %v and got %v", testCase.countAll, got)
			}
			if got := shouldCountPod(testCase.pod, true); got != testCase.countIfReadyOnly {
				t.Errorf("onlyReadyPods=true: expected %v and got %v", testCase.countIfReadyOnly, got)
			}
		})
	}
}

func TestWorkloadOnlyReadyPods(t *testing.T) {
	podList := &v1.PodList{
		Items: []v1.Pod{
			newWorkloadTestPod("ready", "node-a", v1.PodRunning, v1.ConditionTrue, false),
			newWorkloadTestPod("running-not-ready", "node-a", v1.PodRunning, v1.ConditionFalse, false),
			newWorkloadTestPod("pending", "", v1.PodPending, "", false),
			newWorkloadTestPod("terminating-ready", "node-b", v1.PodRunning, v1.ConditionTrue, true),
			newWorkloadTestPod("unknown", "node-c", v1.PodUnknown, v1.ConditionTrue, false),
			// Succeeded and Failed pods are never counted, even while terminating
			newWorkloadTestPod("failed-terminating", "node-d", v1.PodFailed, "", true),
			newWorkloadTestPod("succeeded", "node-d", v1.PodSucceeded, "", false),
		},
	}

	testCases := []struct {
		name          string
		onlyReadyPods string
		groupByNode   bool
		expected      int64
	}{
		{name: "not set counts every non-terminal pod", expected: 5},
		{name: "false counts every non-terminal pod", onlyReadyPods: "false", expected: 5},
		{name: "true counts only ready pods that are not terminating", onlyReadyPods: "true", expected: 1},
		{name: "group by node, not set", groupByNode: true, expected: 3},
		{name: "group by node, only ready pods", onlyReadyPods: "true", groupByNode: true, expected: 1},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			metadata := map[string]string{
				"podSelector": "app=demo",
				"value":       "1",
			}
			if testCase.onlyReadyPods != "" {
				metadata["onlyReadyPods"] = testCase.onlyReadyPods
			}
			if testCase.groupByNode {
				metadata["groupByNode"] = "true"
			}

			s, err := NewKubernetesWorkloadScaler(
				fake.NewClientBuilder().WithRuntimeObjects(podList).Build(),
				&scalersconfig.ScalerConfig{
					TriggerMetadata:         metadata,
					ScalableObjectNamespace: "default",
				},
			)
			if err != nil {
				t.Fatalf("Error creating scaler: %v", err)
			}

			workloadScaler, ok := s.(*kubernetesWorkloadScaler)
			if !ok {
				t.Fatalf("Expected *kubernetesWorkloadScaler and got %T", s)
			}

			count, err := workloadScaler.getMetricValue(context.Background())
			if err != nil {
				t.Fatalf("Error getting metric value: %v", err)
			}
			if count != testCase.expected {
				t.Fatalf("Expected count %d and got %d", testCase.expected, count)
			}
		})
	}
}

func TestWorkloadOnlyReadyPodsActivity(t *testing.T) {
	notReady := newWorkloadTestPod("running-not-ready", "", v1.PodRunning, v1.ConditionFalse, false)
	terminatingReady := newWorkloadTestPod("terminating-ready", "", v1.PodRunning, v1.ConditionTrue, true)
	ready := newWorkloadTestPod("ready", "", v1.PodRunning, v1.ConditionTrue, false)

	testCases := []struct {
		name   string
		pods   []v1.Pod
		active bool
	}{
		{name: "inactive when no matching pod is ready", pods: []v1.Pod{notReady, terminatingReady}, active: false},
		{name: "active when a ready pod exists", pods: []v1.Pod{notReady, terminatingReady, ready}, active: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s, err := NewKubernetesWorkloadScaler(
				fake.NewClientBuilder().WithRuntimeObjects(&v1.PodList{Items: testCase.pods}).Build(),
				&scalersconfig.ScalerConfig{
					TriggerMetadata: map[string]string{
						"podSelector":     "app=demo",
						"value":           "1",
						"activationValue": "0",
						"onlyReadyPods":   "true",
					},
					ScalableObjectNamespace: "default",
				},
			)
			if err != nil {
				t.Fatalf("Error creating scaler: %v", err)
			}

			_, isActive, err := s.GetMetricsAndActivity(context.Background(), "Metric")
			if err != nil {
				t.Fatalf("Error getting metrics: %v", err)
			}
			if isActive != testCase.active {
				t.Fatalf("Expected active=%v and got %v", testCase.active, isActive)
			}
		})
	}
}
