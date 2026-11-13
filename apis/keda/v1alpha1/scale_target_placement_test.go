package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"
)

func placementTarget() *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata":   map[string]interface{}{"name": "target", "namespace": "default"},
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"affinity":                  map[string]interface{}{},
					"topologySpreadConstraints": []interface{}{},
					"containers":                []interface{}{},
				},
			},
		},
	}}
}

func TestUsesKubernetesNodesTrigger(t *testing.T) {
	assert.False(t, UsesKubernetesNodesTrigger(nil))
	assert.False(t, UsesKubernetesNodesTrigger(&ScaledObject{}))
	assert.False(t, UsesKubernetesNodesTrigger(&ScaledObject{
		Spec: ScaledObjectSpec{Triggers: []ScaleTriggers{{Type: "cron"}}},
	}))
	assert.True(t, UsesKubernetesNodesTrigger(&ScaledObject{
		Spec: ScaledObjectSpec{Triggers: []ScaleTriggers{{Type: "cron"}, {Type: KubernetesNodesTriggerType}}},
	}))
}

func TestTargetHasPlacementRules(t *testing.T) {
	assert.False(t, TargetHasPlacementRules(nil))
	assert.False(t, TargetHasPlacementRules(&unstructured.Unstructured{}))

	bare := placementTarget()
	assert.False(t, TargetHasPlacementRules(bare))

	required := placementTarget()
	assert.NoError(t, unstructured.SetNestedSlice(required.Object,
		[]interface{}{map[string]interface{}{"labelSelector": map[string]interface{}{}}},
		"spec", "template", "spec", "affinity", "podAntiAffinity", "requiredDuringSchedulingIgnoredDuringExecution"))
	assert.True(t, TargetHasPlacementRules(required))

	preferred := placementTarget()
	assert.NoError(t, unstructured.SetNestedSlice(preferred.Object,
		[]interface{}{map[string]interface{}{"weight": int64(100)}},
		"spec", "template", "spec", "affinity", "podAntiAffinity", "preferredDuringSchedulingIgnoredDuringExecution"))
	assert.True(t, TargetHasPlacementRules(preferred))

	spread := placementTarget()
	assert.NoError(t, unstructured.SetNestedSlice(spread.Object,
		[]interface{}{map[string]interface{}{"maxSkew": int64(1)}},
		"spec", "template", "spec", "topologySpreadConstraints"))
	assert.True(t, TargetHasPlacementRules(spread))
}

func statefulTarget(t *testing.T, policy string) *unstructured.Unstructured {
	t.Helper()
	obj := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "StatefulSet",
		"metadata":   map[string]interface{}{"name": "sts", "namespace": "default"},
	}}
	if policy != "" {
		assert.NoError(t, unstructured.SetNestedField(obj.Object, policy, "spec", "podManagementPolicy"))
	}
	return obj
}

func nodesSO(configure func(*ScaledObject)) *ScaledObject {
	so := &ScaledObject{
		Spec: ScaledObjectSpec{
			MinReplicaCount: ptr.To[int32](1),
			MaxReplicaCount: ptr.To[int32](10),
			Triggers:        []ScaleTriggers{{Type: KubernetesNodesTriggerType}},
		},
	}
	if configure != nil {
		configure(so)
	}
	return so
}

func TestTargetStatefulSetOrder(t *testing.T) {
	_, isSTS := TargetStatefulSetOrder(nil)
	assert.False(t, isSTS)

	_, isSTS = TargetStatefulSetOrder(placementTarget())
	assert.False(t, isSTS)

	// Missing policy means OrderedReady (the Kubernetes default).
	policy, isSTS := TargetStatefulSetOrder(statefulTarget(t, ""))
	assert.True(t, isSTS)
	assert.Equal(t, "OrderedReady", policy)

	policy, isSTS = TargetStatefulSetOrder(statefulTarget(t, "Parallel"))
	assert.True(t, isSTS)
	assert.Equal(t, "Parallel", policy)
}

func TestIsOrderedReadyFollowSafe(t *testing.T) {
	// Non-nodes triggers are always safe.
	assert.True(t, IsOrderedReadyFollowSafe(
		&ScaledObject{Spec: ScaledObjectSpec{Triggers: []ScaleTriggers{{Type: "cron"}}}},
		statefulTarget(t, "")))

	// Non-StatefulSet targets (Deployments, operator CRs) are exempt.
	assert.True(t, IsOrderedReadyFollowSafe(nodesSO(nil), placementTarget()))

	// Parallel passes.
	assert.True(t, IsOrderedReadyFollowSafe(nodesSO(nil), statefulTarget(t, "Parallel")))

	// OrderedReady follow-down is unsafe.
	assert.False(t, IsOrderedReadyFollowSafe(nodesSO(nil), statefulTarget(t, "")))
	assert.False(t, IsOrderedReadyFollowSafe(nodesSO(nil), statefulTarget(t, "OrderedReady")))

	// Scale-up-only passes via HPA behavior.
	disabled := nodesSO(func(so *ScaledObject) {
		so.Spec.Advanced = &AdvancedConfig{
			HorizontalPodAutoscalerConfig: &HorizontalPodAutoscalerConfig{},
		}
	})
	assert.False(t, IsOrderedReadyFollowSafe(disabled, statefulTarget(t, "")))
	disabled.Spec.Advanced.HorizontalPodAutoscalerConfig.Behavior = &autoscalingv2.HorizontalPodAutoscalerBehavior{
		ScaleDown: &autoscalingv2.HPAScalingRules{SelectPolicy: ptr.To(autoscalingv2.DisabledPolicySelect)},
	}
	assert.True(t, IsOrderedReadyFollowSafe(disabled, statefulTarget(t, "")))

	// Fixed size passes (no downscale possible).
	fixed := nodesSO(func(so *ScaledObject) {
		*so.Spec.MinReplicaCount = 3
		*so.Spec.MaxReplicaCount = 3
	})
	assert.True(t, IsOrderedReadyFollowSafe(fixed, statefulTarget(t, "")))
}

func TestTargetHasPodTemplate(t *testing.T) {
	assert.False(t, TargetHasPodTemplate(nil))
	assert.False(t, TargetHasPodTemplate(&unstructured.Unstructured{}))
	assert.True(t, TargetHasPodTemplate(placementTarget()))

	// Operator-style custom resources carry no pod template.
	cr := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "example.io/v1",
		"kind":       "Cluster",
		"metadata":   map[string]interface{}{"name": "db", "namespace": "default"},
		"spec":       map[string]interface{}{"instances": int64(3)},
	}}
	assert.False(t, TargetHasPodTemplate(cr))
}

func TestIsPlacementFollowSafe(t *testing.T) {
	// Non-nodes triggers are always safe.
	assert.True(t, IsPlacementFollowSafe(
		&ScaledObject{Spec: ScaledObjectSpec{Triggers: []ScaleTriggers{{Type: "cron"}}}},
		placementTarget()))

	// Targets without a pod template (operator CRs) are exempt.
	cr := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "example.io/v1",
		"kind":       "Cluster",
		"metadata":   map[string]interface{}{"name": "db", "namespace": "default"},
		"spec":       map[string]interface{}{"instances": int64(3)},
	}}
	assert.True(t, IsPlacementFollowSafe(nodesSO(nil), cr))

	// Workloads with rules pass.
	affined := placementTarget()
	assert.NoError(t, unstructured.SetNestedSlice(affined.Object,
		[]interface{}{map[string]interface{}{"labelSelector": map[string]interface{}{}}},
		"spec", "template", "spec", "affinity", "podAntiAffinity", "requiredDuringSchedulingIgnoredDuringExecution"))
	assert.True(t, IsPlacementFollowSafe(nodesSO(nil), affined))

	// Bare workloads are unsafe.
	assert.False(t, IsPlacementFollowSafe(nodesSO(nil), placementTarget()))

	// Fixed size is exempt (trigger inert).
	fixed := nodesSO(func(so *ScaledObject) {
		*so.Spec.MinReplicaCount = 2
		*so.Spec.MaxReplicaCount = 2
	})
	assert.True(t, IsPlacementFollowSafe(fixed, placementTarget()))
}
