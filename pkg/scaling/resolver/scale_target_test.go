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
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

var deploymentGVK = schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}

func generatedDeployment(namespace, name string, labels map[string]string) *appsv1.Deployment {
	merged := map[string]string{"team": "events"}
	for k, v := range labels {
		merged[k] = v
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    merged,
		},
	}
}

func TestResolveScaleTargetName(t *testing.T) {
	scheme := clientgoscheme.Scheme
	objs := []runtime.Object{
		generatedDeployment("argo-events", "ingress-sensor-abc12", map[string]string{"events.argoproj.io/sensor-name": "ingress"}),
		generatedDeployment("argo-events", "other-sensor-zz99", map[string]string{"events.argoproj.io/sensor-name": "other"}),
	}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objs...).Build()
	ctx := context.Background()
	ref := func() *kedav1alpha1.ScaleTarget {
		return &kedav1alpha1.ScaleTarget{APIVersion: "apps/v1", Kind: "Deployment"}
	}

	t.Run("fixed name behaves as before", func(t *testing.T) {
		r := ref()
		r.Name = "ingress-sensor-abc12"
		name, err := ResolveScaleTargetName(ctx, cl, "argo-events", deploymentGVK, r)
		if err != nil {
			t.Fatal(err)
		}
		if name != "ingress-sensor-abc12" {
			t.Fatalf("ResolveScaleTargetName() = %q, want %q", name, "ingress-sensor-abc12")
		}
	})

	t.Run("no selector at all errors", func(t *testing.T) {
		if _, err := ResolveScaleTargetName(ctx, cl, "argo-events", deploymentGVK, ref()); !errors.Is(err, ErrNoScaleTargetSelector) {
			t.Fatalf("ResolveScaleTargetName() error = %v, want %v", err, ErrNoScaleTargetSelector)
		}
	})

	t.Run("nil ref errors", func(t *testing.T) {
		if _, err := ResolveScaleTargetName(ctx, cl, "argo-events", deploymentGVK, nil); !errors.Is(err, ErrNoScaleTargetSelector) {
			t.Fatalf("ResolveScaleTargetName() error = %v, want %v", err, ErrNoScaleTargetSelector)
		}
	})

	t.Run("fixed name plus prefix", func(t *testing.T) {
		r := ref()
		r.Name = "ingress-sensor-abc12"
		r.NamePrefix = "ingress-"
		name, err := ResolveScaleTargetName(ctx, cl, "argo-events", deploymentGVK, r)
		if err != nil {
			t.Fatal(err)
		}
		if name != "ingress-sensor-abc12" {
			t.Fatalf("ResolveScaleTargetName() = %q, want %q", name, "ingress-sensor-abc12")
		}
	})

	t.Run("fixed name contradicting prefix matches zero", func(t *testing.T) {
		r := ref()
		r.Name = "other-sensor-zz99"
		r.NamePrefix = "ingress-"
		if _, err := ResolveScaleTargetName(ctx, cl, "argo-events", deploymentGVK, r); !errors.Is(err, ErrNoScaleTargetMatch) {
			t.Fatalf("ResolveScaleTargetName() error = %v, want %v", err, ErrNoScaleTargetMatch)
		}
	})

	t.Run("prefix resolves generated name", func(t *testing.T) {
		r := ref()
		r.NamePrefix = "ingress-sensor-"
		name, err := ResolveScaleTargetName(ctx, cl, "argo-events", deploymentGVK, r)
		if err != nil {
			t.Fatal(err)
		}
		if name != "ingress-sensor-abc12" {
			t.Fatalf("ResolveScaleTargetName() = %q, want %q", name, "ingress-sensor-abc12")
		}
	})

	t.Run("labels narrow to one", func(t *testing.T) {
		r := ref()
		r.LabelSelector = &metav1.LabelSelector{MatchLabels: map[string]string{
			"events.argoproj.io/sensor-name": "other",
		}}
		name, err := ResolveScaleTargetName(ctx, cl, "argo-events", deploymentGVK, r)
		if err != nil {
			t.Fatal(err)
		}
		if name != "other-sensor-zz99" {
			t.Fatalf("ResolveScaleTargetName() = %q, want %q", name, "other-sensor-zz99")
		}
	})

	t.Run("prefix plus labels", func(t *testing.T) {
		r := ref()
		r.NamePrefix = "ingress-"
		r.LabelSelector = &metav1.LabelSelector{MatchLabels: map[string]string{
			"events.argoproj.io/sensor-name": "ingress",
		}}
		name, err := ResolveScaleTargetName(ctx, cl, "argo-events", deploymentGVK, r)
		if err != nil {
			t.Fatal(err)
		}
		if name != "ingress-sensor-abc12" {
			t.Fatalf("ResolveScaleTargetName() = %q, want %q", name, "ingress-sensor-abc12")
		}
	})

	t.Run("contradictory selectors match zero", func(t *testing.T) {
		r := ref()
		r.NamePrefix = "ingress-"
		r.LabelSelector = &metav1.LabelSelector{MatchLabels: map[string]string{
			"events.argoproj.io/sensor-name": "other",
		}}
		if _, err := ResolveScaleTargetName(ctx, cl, "argo-events", deploymentGVK, r); !errors.Is(err, ErrNoScaleTargetMatch) {
			t.Fatalf("ResolveScaleTargetName() error = %v, want %v", err, ErrNoScaleTargetMatch)
		}
	})

	t.Run("broad selector fails closed on ambiguity", func(t *testing.T) {
		r := ref()
		r.LabelSelector = &metav1.LabelSelector{MatchLabels: map[string]string{
			"team": "events",
		}}
		if _, err := ResolveScaleTargetName(ctx, cl, "argo-events", deploymentGVK, r); !errors.Is(err, ErrAmbiguousScaleTargetMatch) {
			t.Fatalf("ResolveScaleTargetName() error = %v, want %v", err, ErrAmbiguousScaleTargetMatch)
		}
	})
}

func TestGVKForRef(t *testing.T) {
	t.Run("defaults to appsv1 deployment", func(t *testing.T) {
		gvk, err := GVKForRef(&kedav1alpha1.ScaleTarget{})
		if err != nil {
			t.Fatal(err)
		}
		if gvk != deploymentGVK {
			t.Fatalf("GVKForRef() = %v, want %v", gvk, deploymentGVK)
		}
	})

	t.Run("explicit ref passes through", func(t *testing.T) {
		gvk, err := GVKForRef(&kedav1alpha1.ScaleTarget{APIVersion: "autoscaling/v2", Kind: "HorizontalPodAutoscaler"})
		if err != nil {
			t.Fatal(err)
		}
		if gvk.Group != "autoscaling" || gvk.Version != "v2" || gvk.Kind != "HorizontalPodAutoscaler" {
			t.Fatalf("unexpected gvk %v", gvk)
		}
	})

	t.Run("invalid apiVersion errors", func(t *testing.T) {
		if _, err := GVKForRef(&kedav1alpha1.ScaleTarget{APIVersion: ":::bad"}); err == nil {
			t.Fatal("want error for invalid apiVersion")
		}
	})
}

func TestScaleTargetUsesSelectors(t *testing.T) {
	if ScaleTargetUsesSelectors(nil) {
		t.Fatal("ScaleTargetUsesSelectors(nil) = true, want false")
	}
	if ScaleTargetUsesSelectors(&kedav1alpha1.ScaleTarget{}) {
		t.Fatal("ScaleTargetUsesSelectors(empty) = true, want false")
	}
	if ScaleTargetUsesSelectors(&kedav1alpha1.ScaleTarget{Name: "fixed"}) {
		t.Fatal("ScaleTargetUsesSelectors(fixed) = true, want false")
	}
	if !ScaleTargetUsesSelectors(&kedav1alpha1.ScaleTarget{NamePrefix: "sensor-"}) {
		t.Fatal("ScaleTargetUsesSelectors(prefix) = false, want true")
	}
	if !ScaleTargetUsesSelectors(&kedav1alpha1.ScaleTarget{LabelSelector: &metav1.LabelSelector{}}) {
		t.Fatal("ScaleTargetUsesSelectors(selector) = false, want true")
	}
}

func conflictScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	if err := kedav1alpha1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	return s
}

func conflictSO(name string, created metav1.Time, ref *kedav1alpha1.ScaleTarget) *kedav1alpha1.ScaledObject {
	return &kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "ns", CreationTimestamp: created},
		Spec:       kedav1alpha1.ScaledObjectSpec{ScaleTargetRef: ref},
	}
}

func TestFindConflictingScaledObject(t *testing.T) {
	older := metav1.NewTime(time.Now().Add(-time.Hour))
	newer := metav1.NewTime(time.Now())
	ctx := context.Background()

	selectorRef := func() *kedav1alpha1.ScaleTarget {
		return &kedav1alpha1.ScaleTarget{
			APIVersion:    "apps/v1",
			Kind:          "Deployment",
			LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"team": "events"}},
		}
	}
	fixedRef := func(name string) *kedav1alpha1.ScaleTarget {
		return &kedav1alpha1.ScaleTarget{APIVersion: "apps/v1", Kind: "Deployment", Name: name}
	}
	target := func(name string) *appsv1.Deployment {
		return &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: name, Labels: map[string]string{"team": "events"}}}
	}

	t.Run("fixed self short-circuits", func(t *testing.T) {
		cl := fake.NewClientBuilder().WithScheme(conflictScheme(t)).Build()
		self := conflictSO("self", newer, fixedRef("target-a"))
		if got, err := FindConflictingScaledObject(ctx, cl, self, "target-a", deploymentGVK); err != nil || got != "" {
			t.Fatalf("FindConflictingScaledObject() = %q, %v; want %q, nil", got, err, "")
		}
	})

	t.Run("no conflict when alone", func(t *testing.T) {
		self := conflictSO("self", newer, selectorRef())
		cl := fake.NewClientBuilder().WithScheme(conflictScheme(t)).WithRuntimeObjects(self, target("target-a")).Build()
		if got, err := FindConflictingScaledObject(ctx, cl, self, "target-a", deploymentGVK); err != nil || got != "" {
			t.Fatalf("FindConflictingScaledObject() = %q, %v; want %q, nil", got, err, "")
		}
	})

	t.Run("older selector wins over newer selector", func(t *testing.T) {
		self := conflictSO("self", newer, selectorRef())
		other := conflictSO("other", older, selectorRef())
		cl := fake.NewClientBuilder().WithScheme(conflictScheme(t)).WithRuntimeObjects(self, other, target("target-a")).Build()
		if got, err := FindConflictingScaledObject(ctx, cl, self, "target-a", deploymentGVK); err != nil || got != "other" {
			t.Fatalf("FindConflictingScaledObject() = %q, %v; want %q, nil", got, err, "other")
		}
	})

	t.Run("newer selector holds against older selector", func(t *testing.T) {
		self := conflictSO("self", older, selectorRef())
		other := conflictSO("other", newer, selectorRef())
		cl := fake.NewClientBuilder().WithScheme(conflictScheme(t)).WithRuntimeObjects(self, other, target("target-a")).Build()
		if got, err := FindConflictingScaledObject(ctx, cl, self, "target-a", deploymentGVK); err != nil || got != "" {
			t.Fatalf("FindConflictingScaledObject() = %q, %v; want %q, nil", got, err, "")
		}
	})

	t.Run("equal timestamps break ties by name", func(t *testing.T) {
		self := conflictSO("zzz-self", older, selectorRef())
		other := conflictSO("aaa-other", older, selectorRef())
		cl := fake.NewClientBuilder().WithScheme(conflictScheme(t)).WithRuntimeObjects(self, other, target("target-a")).Build()
		if got, err := FindConflictingScaledObject(ctx, cl, self, "target-a", deploymentGVK); err != nil || got != "aaa-other" {
			t.Fatalf("FindConflictingScaledObject() = %q, %v; want %q, nil", got, err, "aaa-other")
		}
	})

	t.Run("fixed other always beats selector self", func(t *testing.T) {
		for _, otherTime := range []metav1.Time{older, newer} {
			self := conflictSO("self", newer, selectorRef())
			other := conflictSO("other", otherTime, fixedRef("target-a"))
			cl := fake.NewClientBuilder().WithScheme(conflictScheme(t)).WithRuntimeObjects(self, other, target("target-a")).Build()
			if got, err := FindConflictingScaledObject(ctx, cl, self, "target-a", deploymentGVK); err != nil || got != "other" {
				t.Fatalf("otherTime %v: FindConflictingScaledObject() = %q, %v; want %q, nil", otherTime, got, err, "other")
			}
		}
	})

	t.Run("neighbor with name and selectors resolves before comparing", func(t *testing.T) {
		// The neighbor names target-a but its label selector matches only
		// target-b, so it resolves to target-b: no conflict with target-a.
		self := conflictSO("self", newer, selectorRef())
		otherRef := &kedav1alpha1.ScaleTarget{
			APIVersion:    "apps/v1",
			Kind:          "Deployment",
			Name:          "target-a",
			LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"team": "other"}},
		}
		other := conflictSO("other", older, otherRef)
		objs := []runtime.Object{self, other,
			target("target-a"),
			&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "target-b", Labels: map[string]string{"team": "other"}}},
		}
		cl := fake.NewClientBuilder().WithScheme(conflictScheme(t)).WithRuntimeObjects(objs...).Build()
		if got, err := FindConflictingScaledObject(ctx, cl, self, "target-a", deploymentGVK); err != nil || got != "" {
			t.Fatalf("FindConflictingScaledObject() = %q, %v; want %q, nil", got, err, "")
		}
	})

	t.Run("two selector neighbors share one listing without conflict", func(t *testing.T) {
		self := conflictSO("self", newer, selectorRef())
		other := conflictSO("other", older, &kedav1alpha1.ScaleTarget{
			APIVersion:    "apps/v1",
			Kind:          "Deployment",
			LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"team": "other"}},
		})
		objs := []runtime.Object{self, other,
			target("target-a"),
			&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "target-b", Labels: map[string]string{"team": "other"}}},
		}
		cl := fake.NewClientBuilder().WithScheme(conflictScheme(t)).WithRuntimeObjects(objs...).Build()
		if got, err := FindConflictingScaledObject(ctx, cl, self, "target-a", deploymentGVK); err != nil || got != "" {
			t.Fatalf("FindConflictingScaledObject() = %q, %v; want %q, nil", got, err, "")
		}
	})

	t.Run("unresolvable and foreign neighbors are skipped", func(t *testing.T) {
		self := conflictSO("self", newer, selectorRef())
		ghost := conflictSO("ghost", older, &kedav1alpha1.ScaleTarget{
			APIVersion:    "apps/v1",
			Kind:          "Deployment",
			LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"team": "nobody"}},
		})
		foreign := conflictSO("foreign", older, &kedav1alpha1.ScaleTarget{
			APIVersion:    "apps/v1",
			Kind:          "StatefulSet",
			LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"team": "events"}},
		})
		cl := fake.NewClientBuilder().WithScheme(conflictScheme(t)).WithRuntimeObjects(self, ghost, foreign, target("target-a")).Build()
		if got, err := FindConflictingScaledObject(ctx, cl, self, "target-a", deploymentGVK); err != nil || got != "" {
			t.Fatalf("FindConflictingScaledObject() = %q, %v; want %q, nil", got, err, "")
		}
	})
}
