/*
Copyright 2026 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resolver

import (
	"context"
	"errors"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

// GVKForRef derives the target GroupVersionKind from the ref, applying
// KEDA's documented defaults (apps/v1 Deployment) when unset.
func GVKForRef(ref *kedav1alpha1.ScaleTarget) (schema.GroupVersionKind, error) {
	apiVersion := ref.APIVersion
	if apiVersion == "" {
		apiVersion = "apps/v1"
	}
	kind := ref.Kind
	if kind == "" {
		kind = "Deployment"
	}
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return schema.GroupVersionKind{}, fmt.Errorf("invalid scaleTargetRef.apiVersion %q: %w", apiVersion, err)
	}
	// ParseGroupVersion never rejects garbage (no "/" just means empty
	// group), so enforce the Kubernetes version grammar here: versions
	// are always v<major>[alpha|beta<minor>]. Fail fast with a clear
	// error instead of a RESTMapper discovery failure downstream.
	if len(gv.Version) < 2 || gv.Version[0] != 'v' || gv.Version[1] < '0' || gv.Version[1] > '9' {
		return schema.GroupVersionKind{}, fmt.Errorf("invalid scaleTargetRef.apiVersion %q: version must be v<major>[alpha|beta<minor>]", apiVersion)
	}
	return gv.WithKind(kind), nil
}

// Resolution failures are typed so callers fail closed (hold replicas,
// Degraded condition) instead of actuating on ambiguity.
var (
	// ErrNoScaleTargetSelector means no name, prefix, or labels were given.
	ErrNoScaleTargetSelector = errors.New("scaleTargetRef needs at least one of name, namePrefix, labelSelector")
	// ErrNoScaleTargetMatch means the selectors matched zero objects.
	ErrNoScaleTargetMatch = errors.New("scaleTargetRef selectors matched zero objects")
	// ErrAmbiguousScaleTargetMatch means the selectors matched several
	// objects. Scaling any one of them would be a guess.
	ErrAmbiguousScaleTargetMatch = errors.New("scaleTargetRef selectors matched several objects")
)

// effectiveLabelSelector treats an explicitly empty selector as absent:
// admission requires non-empty matchers, and an empty selector would
// otherwise match every object of the kind.
func effectiveLabelSelector(ref *kedav1alpha1.ScaleTarget) *metav1.LabelSelector {
	if ref == nil || ref.LabelSelector == nil {
		return nil
	}
	if len(ref.LabelSelector.MatchLabels) == 0 && len(ref.LabelSelector.MatchExpressions) == 0 {
		return nil
	}
	return ref.LabelSelector
}

// ResolveScaleTargetName maps a ScaleTarget to exactly one object name.
// A fixed name alone behaves exactly as before (no listing). Otherwise the
// name prefix and label selector are ANDed and exactly one survivor is
// required. Name and prefix comparisons are literal (no globs).
func ResolveScaleTargetName(ctx context.Context, c client.Client, namespace string, gvk schema.GroupVersionKind, ref *kedav1alpha1.ScaleTarget) (string, error) {
	if ref == nil {
		return "", ErrNoScaleTargetSelector
	}
	labelSelector := effectiveLabelSelector(ref)
	if ref.NamePrefix == "" && labelSelector == nil {
		if ref.Name == "" {
			return "", ErrNoScaleTargetSelector
		}
		return ref.Name, nil
	}

	var sel labels.Selector
	if labelSelector != nil {
		s, err := metav1.LabelSelectorAsSelector(labelSelector)
		if err != nil {
			return "", fmt.Errorf("invalid scaleTargetRef.labelSelector: %w", err)
		}
		sel = s
	} else {
		sel = labels.Everything()
	}

	objs, err := listTargetObjects(ctx, c, namespace, gvk, sel)
	if err != nil {
		return "", err
	}

	return matchScaleTargetObjects(objs, ref)
}

// matchScaleTargetObjects applies a ref's name, prefix and label selector
// to already-listed objects in memory and requires exactly one survivor.
// It mirrors the server-side filtering of ResolveScaleTargetName for
// callers that list each GVK once and match many refs against the result.
func matchScaleTargetObjects(objs []metav1.Object, ref *kedav1alpha1.ScaleTarget) (string, error) {
	var sel labels.Selector
	if labelSelector := effectiveLabelSelector(ref); labelSelector != nil {
		s, err := metav1.LabelSelectorAsSelector(labelSelector)
		if err != nil {
			return "", fmt.Errorf("invalid scaleTargetRef.labelSelector: %w", err)
		}
		sel = s
	}

	matched := make([]string, 0, 1)
	for _, obj := range objs {
		name := obj.GetName()
		if ref.Name != "" && name != ref.Name {
			continue
		}
		if ref.NamePrefix != "" && !strings.HasPrefix(name, ref.NamePrefix) {
			continue
		}
		if sel != nil && !sel.Matches(labels.Set(obj.GetLabels())) {
			continue
		}
		matched = append(matched, name)
	}

	switch len(matched) {
	case 0:
		return "", ErrNoScaleTargetMatch
	case 1:
		return matched[0], nil
	default:
		shown := matched
		suffix := ""
		if len(shown) > 3 {
			shown = shown[:3]
			suffix = ", ..."
		}
		return "", fmt.Errorf("%w: %d objects (%s%s)", ErrAmbiguousScaleTargetMatch, len(matched), strings.Join(shown, ", "), suffix)
	}
}

// listTargetObjects lists workloads of the given kind in the namespace,
// filtered by the selector when non-nil. Built-in workload kinds use typed
// lists served by the informer cache; anything else falls back to an
// unstructured list (uncached API read). A typed-list failure also falls
// back to unstructured, so resolution never depends on cache coverage.
func listTargetObjects(ctx context.Context, c client.Client, namespace string, gvk schema.GroupVersionKind, sel labels.Selector) ([]metav1.Object, error) {
	listOpts := &client.ListOptions{Namespace: namespace, LabelSelector: sel}
	switch {
	case gvk.Group == "apps" && gvk.Kind == "Deployment":
		l := &appsv1.DeploymentList{}
		if err := c.List(ctx, l, listOpts); err == nil {
			objs := make([]metav1.Object, 0, len(l.Items))
			for i := range l.Items {
				objs = append(objs, &l.Items[i])
			}
			return objs, nil
		}
	case gvk.Group == "apps" && gvk.Kind == "StatefulSet":
		l := &appsv1.StatefulSetList{}
		if err := c.List(ctx, l, listOpts); err == nil {
			objs := make([]metav1.Object, 0, len(l.Items))
			for i := range l.Items {
				objs = append(objs, &l.Items[i])
			}
			return objs, nil
		}
	case gvk.Group == "apps" && gvk.Kind == "ReplicaSet":
		l := &appsv1.ReplicaSetList{}
		if err := c.List(ctx, l, listOpts); err == nil {
			objs := make([]metav1.Object, 0, len(l.Items))
			for i := range l.Items {
				objs = append(objs, &l.Items[i])
			}
			return objs, nil
		}
	}

	l := &unstructured.UnstructuredList{}
	l.SetGroupVersionKind(gvk)
	if err := c.List(ctx, l, listOpts); err != nil {
		return nil, fmt.Errorf("listing scale targets: %w", err)
	}
	objs := make([]metav1.Object, 0, len(l.Items))
	for i := range l.Items {
		objs = append(objs, &l.Items[i])
	}
	return objs, nil
}

// ScaleTargetUsesSelectors reports whether the ref matches targets by
// name prefix or label selector instead of a fixed name.
func ScaleTargetUsesSelectors(ref *kedav1alpha1.ScaleTarget) bool {
	return ref != nil && (ref.NamePrefix != "" || ref.LabelSelector != nil)
}

// resolveNeighborTarget resolves one neighbor ref against a per-GVK
// cache: each workload kind is listed once per duplicate check no matter
// how many selector-based neighbors share it, and every neighbor's name,
// prefix and label selector filter that list in memory.
func resolveNeighborTarget(ctx context.Context, c client.Client, listed map[schema.GroupVersionKind][]metav1.Object, namespace string, gvk schema.GroupVersionKind, ref *kedav1alpha1.ScaleTarget) (string, error) {
	items, ok := listed[gvk]
	if !ok {
		var err error
		items, err = listTargetObjects(ctx, c, namespace, gvk, nil)
		if err != nil {
			return "", err
		}
		listed[gvk] = items
	}
	return matchScaleTargetObjects(items, ref)
}

// FindConflictingScaledObject returns the name of another ScaledObject in
// the same namespace that resolves to the same workload when the caller
// must hold, or "" when the caller wins or nothing conflicts. Only
// selector-involved refs can converge after admission: fixed-name
// duplicates are rejected at admission, so fixed refs short-circuit with
// no API calls.
//
// When several ScaledObjects converge on one workload the winner is
// deterministic: a fixed-name ref always beats a selector-based ref, and
// two selector-based refs defer to the oldest ScaledObject (creation
// timestamp, then name). Exactly one side holds, so the pair can never
// deadlock with both sides holding while both HPAs actuate.
func FindConflictingScaledObject(ctx context.Context, c client.Client, self *kedav1alpha1.ScaledObject, targetName string, selfGVK schema.GroupVersionKind) (string, error) {
	if self == nil || !ScaleTargetUsesSelectors(self.Spec.ScaleTargetRef) {
		return "", nil
	}
	soList := &kedav1alpha1.ScaledObjectList{}
	if err := c.List(ctx, soList, &client.ListOptions{Namespace: self.Namespace}); err != nil {
		return "", fmt.Errorf("listing ScaledObjects: %w", err)
	}
	listed := make(map[schema.GroupVersionKind][]metav1.Object)
	for i := range soList.Items {
		other := &soList.Items[i]
		if other.Name == self.Name || other.Spec.ScaleTargetRef == nil {
			continue
		}
		otherGVK, err := GVKForRef(other.Spec.ScaleTargetRef)
		if err != nil || otherGVK != selfGVK {
			continue
		}
		otherUsesSelectors := ScaleTargetUsesSelectors(other.Spec.ScaleTargetRef)
		otherTargetName := other.Spec.ScaleTargetRef.Name
		if otherTargetName == "" || otherUsesSelectors {
			resolved, err := resolveNeighborTarget(ctx, c, listed, self.Namespace, otherGVK, other.Spec.ScaleTargetRef)
			if err != nil {
				continue
			}
			otherTargetName = resolved
		}
		if otherTargetName != targetName {
			continue
		}
		if !otherUsesSelectors {
			return other.Name, nil
		}
		if other.CreationTimestamp.Time.Before(self.CreationTimestamp.Time) ||
			(other.CreationTimestamp.Time.Equal(self.CreationTimestamp.Time) && other.Name < self.Name) {
			return other.Name, nil
		}
	}
	return "", nil
}
