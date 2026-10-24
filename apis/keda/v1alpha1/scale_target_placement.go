package v1alpha1

import (
	"strings"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// KubernetesNodesTriggerType is the scaler type that sets the workload
// replica count from the number of Ready nodes.
//
// The schema generator parses scalers_builder.go for string-literal case
// clauses, so the builder keeps its own "kubernetes-nodes" literal instead
// of referencing this constant.
const KubernetesNodesTriggerType = "kubernetes-nodes"

// UsesKubernetesNodesTrigger reports whether the ScaledObject has a
// kubernetes-nodes trigger.
func UsesKubernetesNodesTrigger(so *ScaledObject) bool {
	if so == nil {
		return false
	}
	for i := range so.Spec.Triggers {
		if so.Spec.Triggers[i].Type == KubernetesNodesTriggerType {
			return true
		}
	}
	return false
}

// TargetHasPlacementRules reports whether the workload's pod template carries
// pod anti-affinity or topology spread constraints. The kubernetes-nodes
// trigger only sets the replica count, so without these rules the scheduler
// may pile pods onto fewer nodes than the replica count suggests.
func TargetHasPlacementRules(obj *unstructured.Unstructured) bool {
	if obj == nil {
		return false
	}
	if antiAffinity, found, err := unstructured.NestedMap(obj.Object,
		"spec", "template", "spec", "affinity", "podAntiAffinity"); err == nil && found {
		if required, _, _ := unstructured.NestedSlice(antiAffinity, "requiredDuringSchedulingIgnoredDuringExecution"); len(required) > 0 {
			return true
		}
		if preferred, _, _ := unstructured.NestedSlice(antiAffinity, "preferredDuringSchedulingIgnoredDuringExecution"); len(preferred) > 0 {
			return true
		}
	}
	if spread, found, err := unstructured.NestedSlice(obj.Object,
		"spec", "template", "spec", "topologySpreadConstraints"); err == nil && found && len(spread) > 0 {
		return true
	}
	return false
}

// TargetHasPodTemplate reports whether the object carries a pod template at
// spec.template. Plain workloads (Deployments, StatefulSets, ReplicaSets,
// Rollouts) always do; operator custom resources store placement in their
// own schema and are exempt from pod-template checks here.
func TargetHasPodTemplate(obj *unstructured.Unstructured) bool {
	if obj == nil {
		return false
	}
	_, found, err := unstructured.NestedMap(obj.Object, "spec", "template")
	return err == nil && found
}

// IsPlacementFollowSafe reports whether a kubernetes-nodes ScaledObject may
// follow node count against the given target. Without pod anti-affinity or
// topology spread constraints the replica count is meaningless: pods pile
// onto fewer nodes than the count suggests. Targets without a pod template
// (operator custom resources) are exempt, as the operator owns placement.
// Fixed size (min==max) is exempt: the trigger is inert. Nil target fails
// open; callers handle missing targets on their own path.
func IsPlacementFollowSafe(so *ScaledObject, obj *unstructured.Unstructured) bool {
	if !UsesKubernetesNodesTrigger(so) {
		return true
	}
	if obj == nil || !TargetHasPodTemplate(obj) {
		return true
	}
	if TargetHasPlacementRules(obj) {
		return true
	}
	return *so.GetHPAMinReplicas() == so.GetHPAMaxReplicas()
}

// TargetStatefulSetOrder reports the podManagementPolicy of a plain
// StatefulSet target and whether the object is one. A missing policy means
// OrderedReady (the Kubernetes default). Operator CRs, Deployments and
// anything else never match and are exempt from order checks: the operator
// owns join/leave ordering behind its own scale subresource.
func TargetStatefulSetOrder(obj *unstructured.Unstructured) (policy string, isStatefulSet bool) {
	if obj == nil || obj.GetKind() != "StatefulSet" {
		return "", false
	}
	if apiVersion := obj.GetAPIVersion(); apiVersion != "" && !strings.HasPrefix(apiVersion, "apps/") {
		return "", false
	}
	policy, _, _ = unstructured.NestedString(obj.Object, "spec", "podManagementPolicy")
	if policy == "" {
		policy = "OrderedReady"
	}
	return policy, true
}

// IsOrderedReadyFollowSafe reports whether a kubernetes-nodes ScaledObject
// may follow node count against the given target. Plain StatefulSets with
// OrderedReady wedge on middle-node scale-down (a Pending lower ordinal
// blocks tail removal) or delete the healthy tail to make room, so they
// are unsafe with follow-down. Safe escapes: Parallel, scale-up-only (HPA
// scaleDown Disabled, including via the paused-scale-in annotation which
// the HPA builder turns into Disabled), or fixed size (min==max, no
// downscale possible). Everything else (Deployments, operator CRs) is
// safe by construction here. Nil target fails open; callers handle
// missing targets on their own path.
func IsOrderedReadyFollowSafe(so *ScaledObject, obj *unstructured.Unstructured) bool {
	if !UsesKubernetesNodesTrigger(so) {
		return true
	}
	policy, isStatefulSet := TargetStatefulSetOrder(obj)
	if !isStatefulSet {
		return true
	}
	if policy == "Parallel" {
		return true
	}
	if so.NeedToPauseScaleIn() {
		return true
	}
	if isScaleDownDisabled(so) {
		return true
	}
	return *so.GetHPAMinReplicas() == so.GetHPAMaxReplicas()
}

// isScaleDownDisabled reports whether the ScaledObject blocks HPA scale-down,
// either directly through the HPA behavior or through the paused-scale-in
// annotation, which the HPA builder turns into Disabled.
func isScaleDownDisabled(so *ScaledObject) bool {
	if so == nil || so.Spec.Advanced == nil {
		return false
	}
	hpaConfig := so.Spec.Advanced.HorizontalPodAutoscalerConfig
	if hpaConfig == nil || hpaConfig.Behavior == nil || hpaConfig.Behavior.ScaleDown == nil {
		return false
	}
	policy := hpaConfig.Behavior.ScaleDown.SelectPolicy
	return policy != nil && *policy == autoscalingv2.DisabledPolicySelect
}
