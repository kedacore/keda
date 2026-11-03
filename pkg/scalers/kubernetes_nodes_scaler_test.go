package scalers

import (
	"context"
	"fmt"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type nodesMetadataTestData struct {
	metadata  map[string]string
	namespace string
	isError   bool
}

var parseNodesMetadataTestDataset = []nodesMetadataTestData{
	{map[string]string{"value": "1"}, "test", false},
	{map[string]string{"value": "1", "nodeSelector": "role=worker"}, "test", false},
	{map[string]string{"value": "1", "activationValue": "1"}, "test", false},
	{map[string]string{"levels": "1,3,5"}, "test", false},
	{map[string]string{"value": "1", "levels": "1,3"}, "test", false},
	{map[string]string{"oddOnly": "true"}, "test", false},
	{map[string]string{"value": "1", "levels": "1,3", "oddOnly": "true"}, "test", true},
	{map[string]string{}, "test", false},
	{map[string]string{"value": "0"}, "test", true},
	{map[string]string{"value": "a"}, "test", true},
	{map[string]string{"value": "NaN"}, "test", true},
	{map[string]string{"value": "Inf"}, "test", true},
	{map[string]string{"value": "-Inf"}, "test", true},
	{map[string]string{"value": "1", "activationValue": "NaN"}, "test", true},
	{map[string]string{"value": "1", "activationValue": "Inf"}, "test", true},
	{map[string]string{"value": "1", "nodeSelector": ":::bad"}, "test", true},
	{map[string]string{"levels": "3,1"}, "test", true},
	{map[string]string{"levels": "0,3"}, "test", true},
	{map[string]string{"levels": "1,1,3"}, "test", true},
	{map[string]string{"levels": "a"}, "test", true},
}

func TestParseNodesMetadata(t *testing.T) {
	for i, testData := range parseNodesMetadataTestDataset {
		_, err := NewKubernetesNodesScaler(
			fake.NewClientBuilder().Build(),
			&scalersconfig.ScalerConfig{
				TriggerMetadata:         testData.metadata,
				ScalableObjectNamespace: testData.namespace,
			},
		)
		if err != nil && !testData.isError {
			t.Errorf("case %d (%v): expected success but got error: %v", i, testData.metadata, err)
		}
		if testData.isError && err == nil {
			t.Errorf("case %d (%v): expected error but got success", i, testData.metadata)
		}
	}
}

func readyNode(name string, labels map[string]string, ready bool) *v1.Node {
	status := v1.ConditionFalse
	if ready {
		status = v1.ConditionTrue
	}
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels},
		Status: v1.NodeStatus{
			Conditions: []v1.NodeCondition{{Type: v1.NodeReady, Status: status}},
		},
	}
}

// cordonedNode is a Ready-state node that carries spec.unschedulable
// (cordoned/draining). The scaler must never count it.
func cordonedNode(name string, labels map[string]string, ready bool) *v1.Node {
	node := readyNode(name, labels, ready)
	node.Spec.Unschedulable = true
	return node
}

func TestNodesCount(t *testing.T) {
	scaler, err := NewKubernetesNodesScaler(
		fake.NewClientBuilder().WithObjects(
			readyNode("n1", map[string]string{"role": "worker"}, true),
			readyNode("n2", map[string]string{"role": "worker"}, true),
			readyNode("n3", map[string]string{"role": "storage"}, false),
		).Build(),
		&scalersconfig.ScalerConfig{
			TriggerMetadata:         map[string]string{"value": "1"},
			ScalableObjectNamespace: "test",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	metrics, active, err := scaler.GetMetricsAndActivity(context.Background(), "s0-nodes")
	if err != nil {
		t.Fatal(err)
	}
	if len(metrics) != 1 || metrics[0].Value.Value() != 2 {
		t.Errorf("expected count 2 (unready excluded), got %v", metrics)
	}
	if !active {
		t.Error("expected active with 2 nodes")
	}
}

func TestNodesMetricSpec(t *testing.T) {
	scaler, err := NewKubernetesNodesScaler(
		fake.NewClientBuilder().Build(),
		&scalersconfig.ScalerConfig{
			TriggerMetadata:         map[string]string{"value": "2", "nodeSelector": "role=worker"},
			ScalableObjectNamespace: "test",
			TriggerIndex:            1,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	specs := scaler.GetMetricSpecForScaling(context.Background())
	if len(specs) != 1 || specs[0].External == nil {
		t.Fatalf("expected one external metric spec, got %v", specs)
	}
	// NormalizeString keeps "=" (house convention, same as queue-name
	// scalers): the name is opaque to both producer and HPA.
	if got := specs[0].External.Metric.Name; got != "s1-nodes-role=worker" {
		t.Errorf("metric name = %q, want s1-nodes-role=worker", got)
	}
	if got := specs[0].External.Target.AverageValue.Value(); got != 2 {
		t.Errorf("metric target = %d, want 2", got)
	}
}

func TestNodesActivationBoundary(t *testing.T) {
	newScaler := func(t *testing.T, objects ...client.Object) Scaler {
		t.Helper()
		s, err := NewKubernetesNodesScaler(
			fake.NewClientBuilder().WithObjects(objects...).Build(),
			&scalersconfig.ScalerConfig{
				TriggerMetadata:         map[string]string{"value": "1", "activationValue": "1"},
				ScalableObjectNamespace: "test",
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	// Count above activation: active.
	_, active, err := newScaler(t, readyNode("n1", nil, true), readyNode("n2", nil, true)).
		GetMetricsAndActivity(context.Background(), "s0-nodes")
	if err != nil || !active {
		t.Errorf("count 2 vs activation 1: active = %v, err = %v", active, err)
	}
	// Count equal to activation: strictly greater required, so inactive.
	_, active, err = newScaler(t, readyNode("n1", nil, true)).
		GetMetricsAndActivity(context.Background(), "s0-nodes")
	if err != nil || active {
		t.Errorf("count 1 vs activation 1: active = %v, err = %v (want inactive)", active, err)
	}
	// Zero nodes: inactive, metric 0 (HPA holds min, never negative).
	metrics, active, err := newScaler(t).
		GetMetricsAndActivity(context.Background(), "s0-nodes")
	if err != nil || active {
		t.Errorf("count 0: active = %v, err = %v (want inactive)", active, err)
	}
	if len(metrics) != 1 || metrics[0].Value.Value() != 0 {
		t.Errorf("count 0: metric = %v (want single zero value)", metrics)
	}
}

func TestNodesSelector(t *testing.T) {
	scaler, err := NewKubernetesNodesScaler(
		fake.NewClientBuilder().WithObjects(
			readyNode("n1", map[string]string{"role": "worker"}, true),
			readyNode("n2", map[string]string{"role": "storage"}, true),
		).Build(),
		&scalersconfig.ScalerConfig{
			TriggerMetadata:         map[string]string{"value": "1", "nodeSelector": "role=worker"},
			ScalableObjectNamespace: "test",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	metrics, _, err := scaler.GetMetricsAndActivity(context.Background(), "s0-nodes")
	if err != nil {
		t.Fatal(err)
	}
	if len(metrics) != 1 || metrics[0].Value.Value() != 1 {
		t.Errorf("expected count 1 (selector), got %v", metrics)
	}
}

func TestNodesExcludesUnschedulable(t *testing.T) {
	scaler, err := NewKubernetesNodesScaler(
		fake.NewClientBuilder().WithObjects(
			readyNode("n1", nil, true),
			cordonedNode("n2", nil, true),
			readyNode("n3", nil, false),
		).Build(),
		&scalersconfig.ScalerConfig{
			TriggerMetadata:         map[string]string{"value": "1"},
			ScalableObjectNamespace: "test",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	metrics, _, err := scaler.GetMetricsAndActivity(context.Background(), "s0-nodes")
	if err != nil {
		t.Fatal(err)
	}
	if len(metrics) != 1 || metrics[0].Value.Value() != 1 {
		t.Errorf("expected count 1 (cordoned + unready excluded), got %v", metrics)
	}
}

func TestNodesLevels(t *testing.T) {
	// nodes 0..6 with levels 1,3,5 emit 0,1,1,3,3,5,5.
	want := map[int]int64{0: 0, 1: 1, 2: 1, 3: 3, 4: 3, 5: 5, 6: 5}
	for nodes, stepped := range want {
		var objects []client.Object
		for i := 0; i < nodes; i++ {
			objects = append(objects, readyNode(fmt.Sprintf("n%d", i), nil, true))
		}
		s, err := NewKubernetesNodesScaler(
			fake.NewClientBuilder().WithObjects(objects...).Build(),
			&scalersconfig.ScalerConfig{
				TriggerMetadata:         map[string]string{"value": "1", "levels": "1,3,5"},
				ScalableObjectNamespace: "test",
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		metrics, _, err := s.GetMetricsAndActivity(context.Background(), "s0-nodes")
		if err != nil {
			t.Fatal(err)
		}
		if len(metrics) != 1 || metrics[0].Value.Value() != stepped {
			t.Errorf("nodes %d: metric = %v, want %d", nodes, metrics, stepped)
		}
	}
}

func TestNodesLevelsCap(t *testing.T) {
	// levels 1,3 caps: 5 nodes still emit 3.
	var objects []client.Object
	for i := 0; i < 5; i++ {
		objects = append(objects, readyNode(fmt.Sprintf("n%d", i), nil, true))
	}
	s, err := NewKubernetesNodesScaler(
		fake.NewClientBuilder().WithObjects(objects...).Build(),
		&scalersconfig.ScalerConfig{
			TriggerMetadata:         map[string]string{"value": "1", "levels": "1,3"},
			ScalableObjectNamespace: "test",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	metrics, _, err := s.GetMetricsAndActivity(context.Background(), "s0-nodes")
	if err != nil {
		t.Fatal(err)
	}
	if len(metrics) != 1 || metrics[0].Value.Value() != 3 {
		t.Errorf("levels cap: metric = %v, want 3", metrics)
	}
}

func TestNodesLevelsWithValue(t *testing.T) {
	// Value 2 with levels 1,3: 5 nodes -> raw ceil(5/2)=3 -> stepped 3,
	// metric carries stepped*value = 6 so HPA aims ceil(6/2)=3.
	var objects []client.Object
	for i := 0; i < 5; i++ {
		objects = append(objects, readyNode(fmt.Sprintf("n%d", i), nil, true))
	}
	s, err := NewKubernetesNodesScaler(
		fake.NewClientBuilder().WithObjects(objects...).Build(),
		&scalersconfig.ScalerConfig{
			TriggerMetadata:         map[string]string{"value": "2", "levels": "1,3"},
			ScalableObjectNamespace: "test",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	metrics, _, err := s.GetMetricsAndActivity(context.Background(), "s0-nodes")
	if err != nil {
		t.Fatal(err)
	}
	if len(metrics) != 1 || metrics[0].Value.Value() != 6 {
		t.Errorf("levels+value: metric = %v, want 6", metrics)
	}
}

func TestNodesFractionalValue(t *testing.T) {
	// 21 nodes at value 0.7 is exactly 30 replicas, but binary float
	// division lands just above 30, so a bare Ceil would emit 31 and
	// over-scale by one.
	var objects []client.Object
	for i := 0; i < 21; i++ {
		objects = append(objects, readyNode(fmt.Sprintf("n%d", i), nil, true))
	}
	s, err := NewKubernetesNodesScaler(
		fake.NewClientBuilder().WithObjects(objects...).Build(),
		&scalersconfig.ScalerConfig{
			TriggerMetadata:         map[string]string{"value": "0.7"},
			ScalableObjectNamespace: "test",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	metrics, _, err := s.GetMetricsAndActivity(context.Background(), "s0-nodes")
	if err != nil {
		t.Fatal(err)
	}
	if len(metrics) != 1 || metrics[0].Value.MilliValue() != 21000 {
		t.Errorf("fractional value: metric = %v, want 21000m", metrics)
	}
}

func TestNodesOddOnly(t *testing.T) {
	// Nodes 0..7 emit 0,1,1,3,3,5,5,7: unbounded, always odd.
	want := map[int]int64{0: 0, 1: 1, 2: 1, 3: 3, 4: 3, 5: 5, 6: 5, 7: 7}
	for nodes, stepped := range want {
		var objects []client.Object
		for i := 0; i < nodes; i++ {
			objects = append(objects, readyNode(fmt.Sprintf("n%d", i), nil, true))
		}
		s, err := NewKubernetesNodesScaler(
			fake.NewClientBuilder().WithObjects(objects...).Build(),
			&scalersconfig.ScalerConfig{
				TriggerMetadata:         map[string]string{"value": "1", "oddOnly": "true"},
				ScalableObjectNamespace: "test",
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		metrics, _, err := s.GetMetricsAndActivity(context.Background(), "s0-nodes")
		if err != nil {
			t.Fatal(err)
		}
		if len(metrics) != 1 || metrics[0].Value.Value() != stepped {
			t.Errorf("nodes %d: metric = %v, want %d", nodes, metrics, stepped)
		}
	}
}
