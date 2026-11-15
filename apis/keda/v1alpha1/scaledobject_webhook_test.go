/*
Copyright 2023 The KEDA Authors

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

package v1alpha1

import (
	"context"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = It("should validate the so creation when there isn't any hpa", func() {

	namespaceName := "valid"
	namespace := createNamespace(namespaceName)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("should validate the so creation when there are other SO for other workloads", func() {

	namespaceName := "valid-multiple-so"
	namespace := createNamespace(namespaceName)
	so1 := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")
	so2 := createScaledObject("other-so-name", namespaceName, "other-workload", "apps/v1", "Deployment", false, map[string]string{}, "")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), so1)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so2)
	}).ShouldNot(HaveOccurred())
})

var _ = It("should validate the so creation when another SO targets the same name with a different Kind", func() {
	// Regression coverage for the scaleTargetRefNameIdx field index: SOs
	// sharing a scaleTargetRef.Name are returned by the indexed List
	// together, and verifyScaledObjects must disambiguate them by GVK so a
	// Deployment "foo" and a StatefulSet "foo" can coexist in one namespace.
	namespaceName := "same-name-different-kind"
	sharedTargetName := "shared-target"
	namespace := createNamespace(namespaceName)
	soDeployment := createScaledObject("so-for-deployment", namespaceName, sharedTargetName, "apps/v1", "Deployment", false, map[string]string{}, "")
	soStatefulSet := createScaledObject("so-for-statefulset", namespaceName, sharedTargetName, "apps/v1", "StatefulSet", false, map[string]string{}, "")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), soDeployment)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), soStatefulSet)
	}).ShouldNot(HaveOccurred())
})

var _ = It("should admit two selector SOs resolving to different workloads", func() {
	// Regression coverage for selector-aware duplicate detection: name-less
	// SOs share the "" field-index value, so without resolution the second
	// creation would be falsely rejected as "already managed".
	namespaceName := "selector-distinct-targets"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), createLabeledDeployment(namespaceName, "ingress-sensor-abc12", map[string]string{"events.argoproj.io/sensor-name": "ingress"}, false, false))
	Expect(err).ToNot(HaveOccurred())
	err = k8sClient.Create(context.Background(), createLabeledDeployment(namespaceName, "other-sensor-zz99", map[string]string{"events.argoproj.io/sensor-name": "other"}, false, false))
	Expect(err).ToNot(HaveOccurred())

	so1 := createScaledObjectWithRef("so-ingress", namespaceName, &ScaleTarget{
		APIVersion: "apps/v1", Kind: "Deployment", NamePrefix: "ingress-sensor-",
		LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"events.argoproj.io/sensor-name": "ingress"}},
	}, false, map[string]string{}, "")
	Eventually(func() error {
		return k8sClient.Create(context.Background(), so1)
	}).ShouldNot(HaveOccurred())

	so2 := createScaledObjectWithRef("so-other", namespaceName, &ScaleTarget{
		APIVersion: "apps/v1", Kind: "Deployment", NamePrefix: "other-sensor-",
		LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"events.argoproj.io/sensor-name": "other"}},
	}, false, map[string]string{}, "")
	Eventually(func() error {
		return k8sClient.Create(context.Background(), so2)
	}).ShouldNot(HaveOccurred())
})

var _ = It("shouldn't admit a second selector SO resolving to the same workload", func() {
	namespaceName := "selector-same-target"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), createLabeledDeployment(namespaceName, "ingress-sensor-abc12", map[string]string{"events.argoproj.io/sensor-name": "ingress"}, false, false))
	Expect(err).ToNot(HaveOccurred())

	so1 := createScaledObjectWithRef("so-first", namespaceName, &ScaleTarget{
		APIVersion: "apps/v1", Kind: "Deployment", NamePrefix: "ingress-sensor-",
	}, false, map[string]string{}, "")
	Eventually(func() error {
		return k8sClient.Create(context.Background(), so1)
	}).ShouldNot(HaveOccurred())

	so2 := createScaledObjectWithRef("so-second", namespaceName, &ScaleTarget{
		APIVersion: "apps/v1", Kind: "Deployment",
		LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"events.argoproj.io/sensor-name": "ingress"}},
	}, false, map[string]string{}, "")
	Eventually(func() error {
		return k8sClient.Create(context.Background(), so2)
	}).Should(HaveOccurred())
})

var _ = It("shouldn't admit a fixed-name SO when a selector SO already resolves to that workload", func() {
	namespaceName := "selector-then-fixed"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), createLabeledDeployment(namespaceName, "ingress-sensor-abc12", map[string]string{"events.argoproj.io/sensor-name": "ingress"}, false, false))
	Expect(err).ToNot(HaveOccurred())

	selectorSo := createScaledObjectWithRef("so-selector", namespaceName, &ScaleTarget{
		APIVersion: "apps/v1", Kind: "Deployment", NamePrefix: "ingress-sensor-",
	}, false, map[string]string{}, "")
	Eventually(func() error {
		return k8sClient.Create(context.Background(), selectorSo)
	}).ShouldNot(HaveOccurred())

	fixedSo := createScaledObject("so-fixed", namespaceName, "ingress-sensor-abc12", "apps/v1", "Deployment", false, map[string]string{}, "")
	Eventually(func() error {
		return k8sClient.Create(context.Background(), fixedSo)
	}).Should(HaveOccurred())
})

var _ = It("shouldn't admit a selector SO whose selectors match several workloads", func() {
	// Ambiguity fails fast at admission: the scale loop could only hold on
	// it forever.
	namespaceName := "selector-ambiguous"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), createLabeledDeployment(namespaceName, "sensor-aaa11", map[string]string{"team": "events"}, false, false))
	Expect(err).ToNot(HaveOccurred())
	err = k8sClient.Create(context.Background(), createLabeledDeployment(namespaceName, "sensor-bbb22", map[string]string{"team": "events"}, false, false))
	Expect(err).ToNot(HaveOccurred())

	so := createScaledObjectWithRef(soName, namespaceName, &ScaleTarget{
		APIVersion: "apps/v1", Kind: "Deployment",
		LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"team": "events"}},
	}, false, map[string]string{}, "")
	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).Should(HaveOccurred())
})

var _ = It("shouldn't admit a selector SO when an unmanaged hpa targets the resolved workload", func() {
	hpaName := "test-unmanaged-hpa"
	namespaceName := "selector-unmanaged-hpa"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), createLabeledDeployment(namespaceName, "ingress-sensor-abc12", map[string]string{"events.argoproj.io/sensor-name": "ingress"}, false, false))
	Expect(err).ToNot(HaveOccurred())

	hpa := createHpa(hpaName, namespaceName, "ingress-sensor-abc12", "apps/v1", "Deployment", nil)
	err = k8sClient.Create(context.Background(), hpa)
	Expect(err).ToNot(HaveOccurred())

	so := createScaledObjectWithRef(soName, namespaceName, &ScaleTarget{
		APIVersion: "apps/v1", Kind: "Deployment", NamePrefix: "ingress-sensor-",
	}, false, map[string]string{}, "")
	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).Should(HaveOccurred())
})

var _ = It("should admit a cpu/memory SO with a selector resolving to a resourced deployment", func() {
	namespaceName := "selector-cpu-memory"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	deployment := createLabeledDeployment(namespaceName, "worker-abc12", map[string]string{"role": "worker"}, true, true)
	err = k8sClient.Create(context.Background(), deployment)
	Expect(err).ToNot(HaveOccurred())

	so := createScaledObjectWithRef(soName, namespaceName, &ScaleTarget{
		APIVersion: "apps/v1", Kind: "Deployment", NamePrefix: "worker-",
	}, true, map[string]string{}, "")
	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("should validate the so creation when there are other HPA for other workloads", func() {

	namespaceName := "valid-other-hpa"
	namespace := createNamespace(namespaceName)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")
	hpa := createHpa("other-hpa-name", namespaceName, "other-workload", "apps/v1", "Deployment", nil)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), hpa)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("should validate the so creation when it's own hpa is already generated", func() {

	hpaName := "test-so-hpa"
	namespaceName := "own-hpa"
	namespace := createNamespace(namespaceName)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")
	hpa := createHpa(hpaName, namespaceName, workloadName, "apps/v1", "Deployment", so)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), hpa)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("should validate the so update when it's own hpa is already generated", func() {

	hpaName := "test-so-hpa"
	namespaceName := "update-so"
	namespace := createNamespace(namespaceName)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")
	hpa := createHpa(hpaName, namespaceName, workloadName, "apps/v1", "Deployment", so)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), hpa)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), so)
	Expect(err).ToNot(HaveOccurred())

	so.Spec.MaxReplicaCount = ptr.To[int32](7)
	Eventually(func() error {
		return k8sClient.Update(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("shouldn't validate the so creation when there is another unmanaged hpa", func() {

	hpaName := "test-unmanaged-hpa"
	namespaceName := "unmanaged-hpa"
	namespace := createNamespace(namespaceName)
	hpa := createHpa(hpaName, namespaceName, workloadName, "apps/v1", "Deployment", nil)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), hpa)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).Should(HaveOccurred())
})

var _ = It("shouldn't validate the so creation when the replica counts are wrong", func() {
	namespaceName := "wrong-replica-count"
	namespace := createNamespace(namespaceName)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")
	so.Spec.MinReplicaCount = ptr.To[int32](10)
	so.Spec.MaxReplicaCount = ptr.To[int32](5)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).Should(HaveOccurred())
})

var _ = It("shouldn't validate the so creation when the fallback is wrong", func() {
	namespaceName := "wrong-fallback"
	namespace := createNamespace(namespaceName)

	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")
	so.Spec.Fallback = &Fallback{
		FailureThreshold: -1,
		Replicas:         -3,
	}

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).Should(HaveOccurred())
})

var _ = It("shouldn't validate the so creation when the fallback is configured and only cpu/memory triggers are used.", func() {
	namespaceName := "wrong-fallback-cpu-memory"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, true)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", true, map[string]string{}, "")
	so.Spec.Fallback = &Fallback{
		FailureThreshold: 3,
		Replicas:         6,
	}
	for index := range so.Spec.Triggers {
		so.Spec.Triggers[index].MetricType = "AverageValue"
	}
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).Should(HaveOccurred())
})

var _ = It("should validate the so creation when the fallback is configured, and at least one trigger (besides cpu/memory) is configured.", func() {
	namespaceName := "right-fallback-at-least-one-averagevalue"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, true)
	// Create ScaledObject with cpu and memory triggers.
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", true, map[string]string{}, "")

	triggers := []ScaleTriggers{
		{
			Type: "kubernetes-workload",
			Name: "workload_trig_1",
			Metadata: map[string]string{
				"podSelector": "pod=workload-test",
				"value":       "1",
			},
			MetricType: v2.ValueMetricType,
		},
		{
			Type: "kubernetes-workload",
			Name: "workload_trig_2",
			Metadata: map[string]string{
				"podSelector": "pod=workload-test",
				"value":       "1",
			},
			MetricType: v2.AverageValueMetricType,
		},
	}
	// Append other triggers to the SO, one of them with metricType=AverageValue.
	so.Spec.Triggers = append(so.Spec.Triggers, triggers...)
	so.Spec.Fallback = &Fallback{
		FailureThreshold: 3,
		Replicas:         6,
	}

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("should validate the so creation when the fallback is configured, and so uses ScalingModifiers.", func() {
	namespaceName := "right-fallback-scalingmodifier"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, true)
	// Create ScaledObject with cpu and memory triggers.
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", true, map[string]string{}, "")

	triggers := []ScaleTriggers{
		{
			Type: "kubernetes-workload",
			Name: "workload_trig_1",
			Metadata: map[string]string{
				"podSelector": "pod=workload-test",
				"value":       "1",
			},
			MetricType: v2.ValueMetricType,
		},
		{
			Type: "kubernetes-workload",
			Name: "workload_trig_2",
			Metadata: map[string]string{
				"podSelector": "pod=workload-test",
				"value":       "1",
			},
			MetricType: v2.ValueMetricType,
		},
	}
	// Append other triggers to the SO, none of them with metricType=AverageValue.
	so.Spec.Triggers = append(so.Spec.Triggers, triggers...)
	so.Spec.Fallback = &Fallback{
		FailureThreshold: 3,
		Replicas:         6,
	}
	so.Spec.Advanced.ScalingModifiers = ScalingModifiers{Target: "2", Formula: "workload_trig_1 + workload_trig_2", MetricType: v2.ValueMetricType}

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).Should(HaveOccurred())
})

var _ = It("should validate the so creation when there is another unmanaged hpa and so has transfer-hpa-ownership activated and the name is specified", func() {

	hpaName := "test-unmanaged-hpa-ownership-with-name"
	namespaceName := "unmanaged-hpa-ownership-with-name"
	namespace := createNamespace(namespaceName)
	hpa := createHpa(hpaName, namespaceName, workloadName, "apps/v1", "Deployment", nil)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{ScaledObjectTransferHpaOwnershipAnnotation: "true"}, hpaName)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), hpa)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("shouldn't validate the so creation when there is another unmanaged hpa and so has transfer-hpa-ownership activated but no name specified", func() {

	hpaName := "test-unmanaged-hpa-ownership-without-name"
	namespaceName := "unmanaged-hpa-ownership-without-name"
	namespace := createNamespace(namespaceName)
	hpa := createHpa(hpaName, namespaceName, workloadName, "apps/v1", "Deployment", nil)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{ScaledObjectTransferHpaOwnershipAnnotation: "true"}, "")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), hpa)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).Should(HaveOccurred())
})

var _ = It("shouldn't validate the so creation when hpa has shared-ownership unactivated", func() {

	hpaName := "test-hpa-disabled-validation-by-hpa-ownership"
	namespaceName := "hpa-ownership"
	namespace := createNamespace(namespaceName)
	hpa := createHpa(hpaName, namespaceName, workloadName, "apps/v1", "Deployment", nil)
	hpa.Annotations = map[string]string{ValidationsHpaOwnershipAnnotation: "false"}
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{ScaledObjectTransferHpaOwnershipAnnotation: "false"}, hpaName)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), hpa)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("shouldn't validate the so creation when there is another so", func() {

	so2Name := "test-so2"
	namespaceName := "managed-hpa"
	namespace := createNamespace(namespaceName)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")
	so2 := createScaledObject(so2Name, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), so2)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).Should(HaveOccurred())
})

var _ = It("shouldn't validate the so creation when there is another hpa with custom apis", func() {

	hpaName := "test-custom-hpa"
	namespaceName := "custom-apis"
	namespace := createNamespace(namespaceName)
	so := createScaledObject(soName, namespaceName, workloadName, "custom-api/v1", "custom-kind", false, map[string]string{}, "")
	hpa := createHpa(hpaName, namespaceName, workloadName, "custom-api/v1", "custom-kind", nil)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), hpa)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).Should(HaveOccurred())
})

var _ = It("should validate the so creation with cpu and memory when deployment has requests", func() {

	namespaceName := "deployment-has-requests"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, true)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", true, map[string]string{}, "")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("shouldn't validate the creation with cpu and memory when deployment is missing", func() {

	namespaceName := "deployment-missing"
	namespace := createNamespace(namespaceName)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", true, map[string]string{}, "")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).Should(HaveOccurred())
})

var _ = It("should validate the creation with cpu and memory when deployment is missing and dry-run is true", func() {

	namespaceName := "deployment-missing-dry-run"
	namespace := createNamespace(namespaceName)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", true, map[string]string{}, "")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so, client.DryRunAll)
	}).ShouldNot(HaveOccurred())
})

var _ = It("shouldn't validate the so creation with cpu and memory when deployment hasn't got memory request", func() {

	namespaceName := "deployment-no-memory-request"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, false)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", true, map[string]string{}, "")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).Should(HaveOccurred())
})

// This test checks whether the validation fails when the CPU and memory resource limits are missing in pod spec (in
// deployment) and there are CPU and memory triggers in ScaledObject. This is a test for already existing behavior.
// See github.com/kedacore/keda/issues/5348
var _ = It("shouldn't validate the SO creation with CPU and memory when deployment doesn't have CPU and memory", func() {
	namespaceName := "resource-default-limits-missing"
	namespace := createNamespace(namespaceName)
	deployment := createDeployment(namespaceName, false, false)
	scaledObject := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", true, map[string]string{}, "")

	// Create namespace
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	// Create deployment
	err = k8sClient.Create(context.Background(), deployment)
	Expect(err).ToNot(HaveOccurred())

	// Create scaled object, asynchronously
	Eventually(func() error {
		return k8sClient.Create(context.Background(), scaledObject)
	}).Should(HaveOccurred())
})

// This test checks whether the validation fails when the CPU and memory resource limits are missing in pod spec (in
// deployment), but there are default limits specified in LimitRange in the same namespace, and there are CPU and
// memory triggers in ScaledObject. This a test for newly added behavior after fixing the following issue.
// See github.com/kedacore/keda/issues/5348
var _ = It("should validate the SO creation with CPU and memory when deployment doesn't have CPU and memory, but LimitRange has the limits specified", func() {
	namespaceName := "resource-default-limits-in-limitrange"
	limitRangeName := "test-limit-range"
	cpuLimit := resource.NewMilliQuantity(100, resource.DecimalSI)
	memoryLimit := resource.NewMilliQuantity(100, resource.DecimalSI)
	namespace := createNamespace(namespaceName)
	deployment := createDeployment(namespaceName, false, false)
	limitRange := createLimitRange(limitRangeName, namespaceName, v1.LimitTypeContainer, cpuLimit, memoryLimit)
	scaledObject := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", true, map[string]string{}, "")

	// Create namespace
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	// Create limit range
	err = k8sClient.Create(context.Background(), limitRange)
	Expect(err).ToNot(HaveOccurred())

	// Create deployment
	err = k8sClient.Create(context.Background(), deployment)
	Expect(err).ToNot(HaveOccurred())

	// Create scaled object, asynchronously
	Eventually(func() error {
		return k8sClient.Create(context.Background(), scaledObject)
	}).ShouldNot(HaveOccurred())
})

var _ = It("shouldn't validate the so creation with cpu and memory when deployment hasn't got cpu request", func() {

	namespaceName := "deployment-no-cpu-request"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, false, true)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", true, map[string]string{}, "")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).Should(HaveOccurred())
})

var _ = It("should validate the so creation with cpu and memory when statefulset has requests", func() {

	namespaceName := "statefulset-has-requests"
	namespace := createNamespace(namespaceName)
	workload := createStatefulSet(namespaceName, true, true)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "StatefulSet", true, map[string]string{}, "")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("shouldn't validate the so creation with cpu and memory when statefulset hasn't got memory request", func() {

	namespaceName := "statefulset-no-memory-request"
	namespace := createNamespace(namespaceName)
	workload := createStatefulSet(namespaceName, true, false)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "StatefulSet", true, map[string]string{}, "")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).Should(HaveOccurred())
})

var _ = It("shouldn't validate the so creation with cpu and memory when statefulset hasn't got cpu request", func() {

	namespaceName := "statefulset-no-cpu-request"
	namespace := createNamespace(namespaceName)
	workload := createStatefulSet(namespaceName, false, true)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "StatefulSet", true, map[string]string{}, "")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).Should(HaveOccurred())
})

// The following tests cover cpu/memory requests declared at pod level (KEP-2837) instead of on the
// containers. The HPA uses the pod-level request as the utilization denominator, so such a workload
// is valid and must be accepted. See github.com/kedacore/keda/issues/8113
var _ = It("should validate the so creation with cpu and memory when deployment has pod-level requests", func() {

	namespaceName := "deployment-has-pod-level-requests"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, false, false)
	workload.Spec.Template.Spec.Resources = &v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
			v1.ResourceMemory: *resource.NewQuantity(100*1024*1024, resource.BinarySI),
		},
	}
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", true, map[string]string{}, "")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

// A trigger that sets containerName produces an HPA ContainerResource metric, whose denominator is
// the named container's own request. The HPA ignores pod-level requests in that case, so validation
// must keep rejecting a container that declares none.
var _ = It("shouldn't validate the so creation with cpu when the trigger sets containerName and only pod-level requests are declared", func() {

	namespaceName := "deployment-pod-level-requests-container-name"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, false, false)
	workload.Spec.Template.Spec.Resources = &v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU: *resource.NewMilliQuantity(100, resource.DecimalSI),
		},
	}
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")
	so.Spec.Triggers = []ScaleTriggers{
		{
			Type:       "cpu",
			MetricType: v2.UtilizationMetricType,
			Metadata: map[string]string{
				"value":         "10",
				"containerName": "test",
			},
		},
	}

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).Should(MatchError(ContainSubstring("the container test doesn't have the cpu request defined")))
})

var _ = It("should validate the so creation with cpu when the trigger sets containerName and that container declares requests", func() {

	namespaceName := "deployment-container-requests-container-name"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, true)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")
	so.Spec.Triggers = []ScaleTriggers{
		{
			Type:       "cpu",
			MetricType: v2.UtilizationMetricType,
			Metadata: map[string]string{
				"value":         "10",
				"containerName": "test",
			},
		},
	}

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

// A pod-level request only satisfies the trigger for the resource it declares, so a memory trigger
// must still be rejected when only cpu is declared at pod level.
var _ = It("shouldn't validate the so creation with cpu and memory when deployment has pod-level cpu request only", func() {

	namespaceName := "deployment-pod-level-cpu-request-only"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, false, false)
	workload.Spec.Template.Spec.Resources = &v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU: *resource.NewMilliQuantity(100, resource.DecimalSI),
		},
	}
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", true, map[string]string{}, "")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).Should(MatchError(ContainSubstring("the scaledobject has a memory trigger")))
})

// Regression guard: container-level requests must keep satisfying the check when the pod-level
// resources are present but declare nothing.
var _ = It("should validate the so creation with cpu and memory when deployment has container requests and empty pod-level resources", func() {

	namespaceName := "deployment-container-requests-empty-pod-level"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, true)
	workload.Spec.Template.Spec.Resources = &v1.ResourceRequirements{}
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", true, map[string]string{}, "")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("should validate the so creation with cpu and memory when statefulset has pod-level requests", func() {

	namespaceName := "statefulset-has-pod-level-requests"
	namespace := createNamespace(namespaceName)
	workload := createStatefulSet(namespaceName, false, false)
	workload.Spec.Template.Spec.Resources = &v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
			v1.ResourceMemory: *resource.NewQuantity(100*1024*1024, resource.BinarySI),
		},
	}
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "StatefulSet", true, map[string]string{}, "")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("should validate the so creation without cpu and memory when custom resources", func() {

	namespaceName := "crd-not-resources"
	namespace := createNamespace(namespaceName)
	so := createScaledObject(soName, namespaceName, workloadName, "custom-api", "StatefulSet", true, map[string]string{}, "")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("should validate so creation when all requirements are met for scaling to zero with cpu scaler", func() {
	namespaceName := "scale-to-zero-good"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, false)

	so := createScaledObjectSTZ(soName, namespaceName, workloadName, 0, 5, true)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("shouldn't validate so creation with cpu scaler requirements not being met for scaling to 0", func() {
	namespaceName := "scale-to-zero-min-replicas-bad"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, false)

	so := createScaledObjectSTZ(soName, namespaceName, workloadName, 0, 5, false)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())
	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).Should(HaveOccurred())
})

var _ = It("should validate so creation when min replicas is > 0 with only cpu scaler given", func() {
	namespaceName := "scale-to-zero-no-external-trigger-good"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, false)

	so := createScaledObjectSTZ(soName, namespaceName, workloadName, 1, 5, false)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())
	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())

})

var _ = It("should not validate ScaledObject creation when deployment only provides cpu resource limits", func() {

	namespaceName := "only-cpu-resource-limits-set"
	namespace := createNamespace(namespaceName)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	workload := createDeployment(namespaceName, false, false)
	workload.Spec.Template.Spec.Containers[0].Resources.Limits = v1.ResourceList{
		v1.ResourceCPU: *resource.NewMilliQuantity(100, resource.DecimalSI),
	}

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")
	so.Spec.Triggers = []ScaleTriggers{
		{
			Type: "cpu",
			Metadata: map[string]string{
				"value": "10",
			},
		},
	}

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("should not validate ScaledObject creation when deployment only provides memory resource limits", func() {

	namespaceName := "only-memory-resource-limits-set"
	namespace := createNamespace(namespaceName)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	workload := createDeployment(namespaceName, false, false)
	workload.Spec.Template.Spec.Containers[0].Resources.Limits = v1.ResourceList{
		v1.ResourceMemory: *resource.NewMilliQuantity(1024, resource.DecimalSI),
	}

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")
	so.Spec.Triggers = []ScaleTriggers{
		{
			Type: "memory",
			Metadata: map[string]string{
				"value": "512Mi",
			},
		},
	}

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("should validate the so update if it's removing the finalizer even if it's invalid", func() {

	namespaceName := "removing-finalizers"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, true)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", true, map[string]string{}, "")
	so.Finalizers = append(so.Finalizers, "finalizer")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())

	workload.Spec.Template.Spec.Containers[0].Resources.Requests = nil
	err = k8sClient.Update(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	so.Finalizers = []string{}
	Eventually(func() error {
		return k8sClient.Update(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("shouldn't validate the so creation when the name exceeds the label value limit", func() {

	namespaceName := "so-name-too-long"
	namespace := createNamespace(namespaceName)
	longName := strings.Repeat("a", maxK8sLabelValueLength+1)
	so := createScaledObject(longName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "short-hpa")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).Should(HaveOccurred())
})

var _ = It("should validate the so creation when the name is at the label value limit and a custom HPA name is set", func() {

	namespaceName := "so-name-max-with-custom-hpa"
	namespace := createNamespace(namespaceName)
	maxName := strings.Repeat("a", maxK8sLabelValueLength)
	so := createScaledObject(maxName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "short-hpa")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("shouldn't validate the so creation when no custom HPA name is set and the generated HPA name would exceed the label value limit", func() {

	namespaceName := "so-generated-hpa-too-long"
	namespace := createNamespace(namespaceName)
	// 55 chars: keda-hpa- prefix (9) + 55 = 64, overflows
	longName := strings.Repeat("a", maxK8sLabelValueLength-len("keda-hpa-")+1)
	so := createScaledObject(longName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).Should(HaveOccurred())
})

var _ = It("should validate the so creation when no custom HPA name is set and the generated HPA name fits within the label value limit", func() {

	namespaceName := "so-generated-hpa-max"
	namespace := createNamespace(namespaceName)
	// 54 chars: keda-hpa- prefix (9) + 54 = 63, fits
	maxName := strings.Repeat("a", maxK8sLabelValueLength-len("keda-hpa-"))
	so := createScaledObject(maxName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("shouldn't validate the so update when removing the custom HPA name would make the generated HPA name exceed the label value limit", func() {

	namespaceName := "so-update-remove-custom-hpa"
	namespace := createNamespace(namespaceName)
	// 60 chars: passes create with custom HPA name, but keda-hpa-<60> = 69 would overflow
	longName := strings.Repeat("a", 60)
	so := createScaledObject(longName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "short-hpa")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), so)
	Expect(err).ToNot(HaveOccurred())

	so.Spec.Advanced.HorizontalPodAutoscalerConfig = nil
	Eventually(func() error {
		return k8sClient.Update(context.Background(), so)
	}).Should(HaveOccurred())
})

var _ = It("shouldn't create so when stabilizationWindowSeconds exceeds 3600", func() {

	namespaceName := "fail-so-creation"
	namespace := createNamespace(namespaceName)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")
	so.Spec.Advanced.HorizontalPodAutoscalerConfig = &HorizontalPodAutoscalerConfig{
		Behavior: &v2.HorizontalPodAutoscalerBehavior{
			ScaleDown: &v2.HPAScalingRules{
				StabilizationWindowSeconds: ptr.To[int32](3700),
			},
		},
	}
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).
		WithTimeout(5 * time.Second).
		Should(HaveOccurred())
})

var _ = It("should validate empty triggers in ScaledObject", func() {

	namespaceName := "empty-triggers-set"
	namespace := createNamespace(namespaceName)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	workload := createDeployment(namespaceName, false, false)
	workload.Spec.Template.Spec.Containers[0].Resources.Limits = v1.ResourceList{
		v1.ResourceCPU: *resource.NewMilliQuantity(100, resource.DecimalSI),
	}

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")
	so.Spec.Triggers = []ScaleTriggers{}

	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).Should(HaveOccurred())
})

// ============================ SCALING MODIFIERS ============================ \\
// =========================================================================== \\

var _ = It("should validate the so creation with ScalingModifiers.Formula", func() {
	namespaceName := "scaling-modifiers-formula-good"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, false, false)

	sm := ScalingModifiers{Target: "2", Formula: "workload_trig + cron_trig"}

	triggers := []ScaleTriggers{
		{
			Type: "cron",
			Name: "cron_trig",
			Metadata: map[string]string{
				"timezone":        "UTC",
				"start":           "0 * * * *",
				"end":             "1 * * * *",
				"desiredReplicas": "1",
			},
		},
		{
			Type: "kubernetes-workload",
			Name: "workload_trig",
			Metadata: map[string]string{
				"podSelector": "pod=workload-test",
				"value":       "1",
			},
		},
	}

	so := createScaledObjectScalingModifiers(namespaceName, sm, triggers)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())
	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("shouldnt validate the so creation with scalingModifiers.Formula but no target", func() {
	namespaceName := "scaling-modifiers-formula-no-target-bad"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, false, false)

	sm := ScalingModifiers{Formula: "workload_trig + cron_trig"}

	triggers := []ScaleTriggers{
		{
			Type: "cron",
			Name: "cron_trig",
			Metadata: map[string]string{
				"timezone":        "UTC",
				"start":           "0 * * * *",
				"end":             "1 * * * *",
				"desiredReplicas": "1",
			},
		},
		{
			Type: "kubernetes-workload",
			Name: "workload_trig",
			Metadata: map[string]string{
				"podSelector": "pod=workload-test",
				"value":       "1",
			},
		},
	}

	so := createScaledObjectScalingModifiers(namespaceName, sm, triggers)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())
	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).Should(HaveOccurred())
})

var _ = It("shouldnt validate the so creation with ScalingModifiers when triggers dont have names", func() {
	namespaceName := "scaling-modifiers-triggers-no-names-bad"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, true)

	sm := ScalingModifiers{Formula: "workload_trig + cron_trig"}

	triggers := []ScaleTriggers{
		{
			Type: "cron",
			Metadata: map[string]string{
				"timezone":        "UTC",
				"start":           "0 * * * *",
				"end":             "1 * * * *",
				"desiredReplicas": "1",
			},
		},
		{
			Type: "kubernetes-workload",
			Metadata: map[string]string{
				"podSelector": "pod=workload-test",
				"value":       "1",
			},
		},
	}

	so := createScaledObjectScalingModifiers(namespaceName, sm, triggers)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())
	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).Should(HaveOccurred())
})

var _ = It("should validate the so creation with ScalingModifiers when formula triggers do have names but not all triggers", func() {
	namespaceName := "scaling-modifiers-specific-triggers-good"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, true)

	sm := ScalingModifiers{Target: "2", Formula: "workload_trig + 1"}

	triggers := []ScaleTriggers{
		{
			Type: "cron",
			Metadata: map[string]string{
				"timezone":        "UTC",
				"start":           "0 * * * *",
				"end":             "1 * * * *",
				"desiredReplicas": "1",
			},
		},
		{
			Type: "kubernetes-workload",
			Name: "workload_trig",
			Metadata: map[string]string{
				"podSelector": "pod=workload-test",
				"value":       "1",
			},
		},
	}

	so := createScaledObjectScalingModifiers(namespaceName, sm, triggers)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())
	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("should validate the so creation with ScalingModifiers when formula casts to float already", func() {
	namespaceName := "scaling-modifiers-cast-to-float-good"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, true)

	sm := ScalingModifiers{Target: "2", Formula: "float(workload_trig + 1)"}

	triggers := []ScaleTriggers{
		{
			Type: "kubernetes-workload",
			Name: "workload_trig",
			Metadata: map[string]string{
				"podSelector": "pod=workload-test",
				"value":       "1",
			},
		},
	}

	so := createScaledObjectScalingModifiers(namespaceName, sm, triggers)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())
	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("should validate the so creation with ScalingModifiers.Formula - casting float from ternary operator", func() {
	namespaceName := "scaling-modifiers-formula-casting-float-good"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, false, false)

	sm := ScalingModifiers{Target: "2", Formula: "float(workload_trig < 5 ? cron_trig + workload_trig : 5)"}

	triggers := []ScaleTriggers{
		{
			Type: "cron",
			Name: "cron_trig",
			Metadata: map[string]string{
				"timezone":        "UTC",
				"start":           "0 * * * *",
				"end":             "1 * * * *",
				"desiredReplicas": "1",
			},
		},
		{
			Type: "kubernetes-workload",
			Name: "workload_trig",
			Metadata: map[string]string{
				"podSelector": "pod=workload-test",
				"value":       "1",
			},
		},
	}

	so := createScaledObjectScalingModifiers(namespaceName, sm, triggers)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())
	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

// this test checks that internally, the casting to float happened successfully
// to override the return value of ternary operator '?' - because it tries to compile
// in webhook validator or during hpa setup and wouldnt compile without float return
// value
var _ = It("should validate the so creation with ScalingModifiers.Formula - ternary operator without casting float", func() {
	namespaceName := "scaling-modifiers-formula-ternary-no-casting-float-good"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, false, false)

	sm := ScalingModifiers{Target: "2", Formula: "workload_trig < 5 ? cron_trig + workload_trig : 5"}

	triggers := []ScaleTriggers{
		{
			Type: "cron",
			Name: "cron_trig",
			Metadata: map[string]string{
				"timezone":        "UTC",
				"start":           "0 * * * *",
				"end":             "1 * * * *",
				"desiredReplicas": "1",
			},
		},
		{
			Type: "kubernetes-workload",
			Name: "workload_trig",
			Metadata: map[string]string{
				"podSelector": "pod=workload-test",
				"value":       "1",
			},
		},
	}

	so := createScaledObjectScalingModifiers(namespaceName, sm, triggers)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())
	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("should validate the so creation with ScalingModifiers.Formula - count operator", func() {
	namespaceName := "scaling-modifiers-formula-count-function-good"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, false, false)

	sm := ScalingModifiers{Target: "2", Formula: "count([trig_one,trig_two],{#>1}) > 1 ? 5 : 0"}

	triggers := []ScaleTriggers{
		{
			Type: "cron",
			Name: "trig_one",
			Metadata: map[string]string{
				"timezone":        "UTC",
				"start":           "0 * * * *",
				"end":             "1 * * * *",
				"desiredReplicas": "1",
			},
		},
		{
			Type: "kubernetes-workload",
			Name: "trig_two",
			Metadata: map[string]string{
				"podSelector": "pod=workload-test",
				"value":       "1",
			},
		},
	}

	so := createScaledObjectScalingModifiers(namespaceName, sm, triggers)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())
	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("should validate the so creation with ScalingModifiers.Formula - complex ternary", func() {
	namespaceName := "scaling-modifiers-formula-complex-ternary-good"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, false, false)

	sm := ScalingModifiers{Target: "2", Formula: "float(trig_one < 2 ? trig_one+trig_two >= 2 ? 5 : 10 : 0)"}

	triggers := []ScaleTriggers{
		{
			Type: "cron",
			Name: "trig_one",
			Metadata: map[string]string{
				"timezone":        "UTC",
				"start":           "0 * * * *",
				"end":             "1 * * * *",
				"desiredReplicas": "1",
			},
		},
		{
			Type: "kubernetes-workload",
			Name: "trig_two",
			Metadata: map[string]string{
				"podSelector": "pod=workload-test",
				"value":       "1",
			},
		},
	}

	so := createScaledObjectScalingModifiers(namespaceName, sm, triggers)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())
	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("should validate the so creation with ScalingModifiers.Formula - double float cast", func() {
	namespaceName := "scaling-modifiers-formula-double-float-cast-good"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, false, false)

	sm := ScalingModifiers{Target: "2", Formula: "float(float(trig_two < 5 ? trig_one + trig_two : 5))"}
	triggers := []ScaleTriggers{
		{
			Type: "cron",
			Name: "trig_one",
			Metadata: map[string]string{
				"timezone":        "UTC",
				"start":           "0 * * * *",
				"end":             "1 * * * *",
				"desiredReplicas": "1",
			},
		},
		{
			Type: "kubernetes-workload",
			Name: "trig_two",
			Metadata: map[string]string{
				"podSelector": "pod=workload-test",
				"value":       "1",
			},
		},
	}

	so := createScaledObjectScalingModifiers(namespaceName, sm, triggers)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())
	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

// ======================== TARGET PLACEMENT DENY TESTS ========================

var _ = It("should deny a kubernetes-nodes ScaledObject whose workload has no placement rules", func() {
	namespaceName := "placement-deny-no-rules"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, false, false)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")
	so.Spec.Triggers = []ScaleTriggers{{Type: "kubernetes-nodes", Metadata: map[string]string{}}}

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	_, err = so.ValidateCreate(new(false))
	Expect(err).To(HaveOccurred())
	Expect(err.Error()).To(ContainSubstring("no pod anti-affinity or topology spread constraints"))
})

var _ = It("should NOT emit placement warning when the target workload has pod anti-affinity", func() {
	namespaceName := "placement-no-warning-with-rules"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, false, false)
	workload.Spec.Template.Spec.Affinity = &v1.Affinity{
		PodAntiAffinity: &v1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
				{
					LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"test": "test"}},
					TopologyKey:   "kubernetes.io/hostname",
				},
			},
		},
	}
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")
	so.Spec.Triggers = []ScaleTriggers{{Type: "kubernetes-nodes", Metadata: map[string]string{}}}

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	warnings, err := so.ValidateCreate(new(false))
	Expect(err).ToNot(HaveOccurred())
	Expect(warnings).ToNot(ContainElement(ContainSubstring("no pod anti-affinity or topology spread constraints")))
})

var _ = It("should NOT emit placement warning for triggers other than kubernetes-nodes", func() {
	namespaceName := "placement-no-warning-other-trigger"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, false, false)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	warnings, err := so.ValidateCreate(new(false))
	Expect(err).ToNot(HaveOccurred())
	Expect(warnings).ToNot(ContainElement(ContainSubstring("no pod anti-affinity or topology spread constraints")))
})

var _ = It("should reject a kubernetes-nodes ScaledObject with an empty labelSelector", func() {
	namespaceName := "order-reject-empty-selector"
	namespace := createNamespace(namespaceName)
	so := createScaledObjectWithRef(soName, namespaceName, &ScaleTarget{
		APIVersion:    "apps/v1",
		Kind:          "Deployment",
		LabelSelector: &metav1.LabelSelector{},
	}, false, map[string]string{}, "")
	so.Spec.Triggers = []ScaleTriggers{{Type: "kubernetes-nodes", Metadata: map[string]string{}}}

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	// The CEL rule on the CRD rejects the empty selector: it would match
	// every object of the kind.
	err = k8sClient.Create(context.Background(), so)
	Expect(err).To(HaveOccurred())
	Expect(err.Error()).To(ContainSubstring("non-empty labelSelector"))
})

var _ = It("should deny through the direct-client fallback when the cache misses a fresh OrderedReady target", func() {
	namespaceName := "order-direct-fallback"
	sts := createStatefulSetWithPolicy(namespaceName, "")
	sts.Spec.Template.Spec.Affinity = &v1.Affinity{
		PodAntiAffinity: &v1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
				{
					LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"test": "test"}},
					TopologyKey:   "kubernetes.io/hostname",
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	Expect(clientgoscheme.AddToScheme(scheme)).ToNot(HaveOccurred())
	Expect(AddToScheme(scheme)).ToNot(HaveOccurred())
	indexByRefName := func(obj client.Object) []string {
		switch o := obj.(type) {
		case *v2.HorizontalPodAutoscaler:
			return []string{o.Spec.ScaleTargetRef.Name}
		case *ScaledObject:
			if o.Spec.ScaleTargetRef == nil {
				return nil
			}
			return []string{o.Spec.ScaleTargetRef.Name}
		default:
			return nil
		}
	}
	// cachedCl simulates a stale informer cache: nothing observed yet.
	// directCl is the authoritative API state with the fresh target.
	cachedCl := fake.NewClientBuilder().WithScheme(scheme).
		WithIndex(&v2.HorizontalPodAutoscaler{}, scaleTargetRefNameIdx, indexByRefName).
		WithIndex(&ScaledObject{}, scaleTargetRefNameIdx, indexByRefName).
		Build()
	directCl := fake.NewClientBuilder().WithScheme(scheme).
		WithRuntimeObjects(sts).Build()

	oldKc, oldDirectClient, oldFallback := kc, directClient, cacheMissToDirectClient
	kc, directClient, cacheMissToDirectClient = cachedCl, directCl, true
	defer func() { kc, directClient, cacheMissToDirectClient = oldKc, oldDirectClient, oldFallback }()

	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "StatefulSet", false, map[string]string{}, "")
	so.Spec.Triggers = []ScaleTriggers{{Type: "kubernetes-nodes", Metadata: map[string]string{}}}

	// Without the typed-list plus Get fallback this fails open (stale
	// empty cache skips both checks); with it, the order check denies.
	_, err := so.ValidateCreate(new(false))
	Expect(err).To(HaveOccurred())
	Expect(err.Error()).To(ContainSubstring("OrderedReady"))
})

var _ = It("should admit a fixed-size kubernetes-nodes ScaledObject without placement rules", func() {
	namespaceName := "placement-allow-fixed"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, false, false)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")
	so.Spec.Triggers = []ScaleTriggers{{Type: "kubernetes-nodes", Metadata: map[string]string{}}}
	so.Spec.MinReplicaCount = ptr.To[int32](2)
	so.Spec.MaxReplicaCount = ptr.To[int32](2)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	_, err = so.ValidateCreate(new(false))
	Expect(err).ToNot(HaveOccurred())
})

// ======================== ORDEREDREADY FOLLOW-DOWN TESTS ========================
var _ = It("should deny a kubernetes-nodes ScaledObject on an OrderedReady StatefulSet", func() {
	namespaceName := "order-deny-orderedready"
	namespace := createNamespace(namespaceName)
	workload := createStatefulSetWithPolicy(namespaceName, "")
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "StatefulSet", false, map[string]string{}, "")
	so.Spec.Triggers = []ScaleTriggers{{Type: "kubernetes-nodes", Metadata: map[string]string{}}}

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	_, err = so.ValidateCreate(new(false))
	Expect(err).To(HaveOccurred())
	Expect(err.Error()).To(ContainSubstring("OrderedReady"))
})

var _ = It("should admit a kubernetes-nodes ScaledObject on a Parallel StatefulSet", func() {
	namespaceName := "order-allow-parallel"
	namespace := createNamespace(namespaceName)
	workload := createStatefulSetWithPolicy(namespaceName, string(appsv1.ParallelPodManagement))
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "StatefulSet", false, map[string]string{}, "")
	so.Spec.Triggers = []ScaleTriggers{{Type: "kubernetes-nodes", Metadata: map[string]string{}}}

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	_, err = so.ValidateCreate(new(false))
	Expect(err).ToNot(HaveOccurred())
})

var _ = It("should admit an OrderedReady StatefulSet with scale-up-only behavior", func() {
	namespaceName := "order-allow-scaleuponly"
	namespace := createNamespace(namespaceName)
	workload := createStatefulSetWithPolicy(namespaceName, "")
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "StatefulSet", false, map[string]string{}, "")
	so.Spec.Triggers = []ScaleTriggers{{Type: "kubernetes-nodes", Metadata: map[string]string{}}}
	disabled := v2.DisabledPolicySelect
	so.Spec.Advanced.HorizontalPodAutoscalerConfig = &HorizontalPodAutoscalerConfig{
		Behavior: &v2.HorizontalPodAutoscalerBehavior{
			ScaleDown: &v2.HPAScalingRules{SelectPolicy: &disabled},
		},
	}

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	_, err = so.ValidateCreate(new(false))
	Expect(err).ToNot(HaveOccurred())
})

var _ = It("should admit an OrderedReady StatefulSet with fixed size", func() {
	namespaceName := "order-allow-fixed"
	namespace := createNamespace(namespaceName)
	workload := createStatefulSetWithPolicy(namespaceName, "")
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "StatefulSet", false, map[string]string{}, "")
	so.Spec.Triggers = []ScaleTriggers{{Type: "kubernetes-nodes", Metadata: map[string]string{}}}
	so.Spec.MinReplicaCount = ptr.To[int32](3)
	so.Spec.MaxReplicaCount = ptr.To[int32](3)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	_, err = so.ValidateCreate(new(false))
	Expect(err).ToNot(HaveOccurred())
})

var _ = It("should fail open when the kubernetes-nodes target does not exist yet", func() {
	namespaceName := "order-failopen-missing"
	namespace := createNamespace(namespaceName)
	so := createScaledObject(soName, namespaceName, "never-created-sts", "apps/v1", "StatefulSet", false, map[string]string{}, "")
	so.Spec.Triggers = []ScaleTriggers{{Type: "kubernetes-nodes", Metadata: map[string]string{}}}

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	_, err = so.ValidateCreate(new(false))
	Expect(err).ToNot(HaveOccurred())
})

// ======================== DIRECT-CLIENT RECHECK TESTS ========================
var _ = It("should consult the direct client when the cached target lookup finds nothing", func() {
	namespaceName := "direct-recheck"
	// The workload carries the matched label on the Deployment object
	// itself so the selector resolves against it.
	workload := createDeployment(namespaceName, false, false)
	workload.Labels = map[string]string{"test": "test"}
	hpa := createHpa("keda-hpa-other", namespaceName, workloadName, "apps/v1", "Deployment", nil)

	scheme := runtime.NewScheme()
	Expect(clientgoscheme.AddToScheme(scheme)).ToNot(HaveOccurred())
	Expect(AddToScheme(scheme)).ToNot(HaveOccurred())
	hpaIndex := func(obj client.Object) []string {
		return []string{obj.(*v2.HorizontalPodAutoscaler).Spec.ScaleTargetRef.Name}
	}
	// cachedCl simulates a stale informer cache: the HPA is visible but
	// the workload is not. directCl is the authoritative API state.
	cachedCl := fake.NewClientBuilder().WithScheme(scheme).
		WithIndex(&v2.HorizontalPodAutoscaler{}, scaleTargetRefNameIdx, hpaIndex).
		WithRuntimeObjects(hpa).Build()
	directCl := fake.NewClientBuilder().WithScheme(scheme).
		WithRuntimeObjects(workload, hpa).Build()

	oldKc, oldDirectClient, oldFallback := kc, directClient, cacheMissToDirectClient
	kc, directClient, cacheMissToDirectClient = cachedCl, directCl, true
	defer func() { kc, directClient, cacheMissToDirectClient = oldKc, oldDirectClient, oldFallback }()

	so := createScaledObjectWithRef(soName, namespaceName, &ScaleTarget{
		APIVersion:    "apps/v1",
		Kind:          "Deployment",
		LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"test": "test"}},
	}, false, map[string]string{}, "")

	// The direct recheck finds the workload, so the HPA ownership check
	// runs and rejects the duplicate instead of skipping.
	_, err := verifyHpas(so, "create", false)
	Expect(err).To(HaveOccurred())
	Expect(err.Error()).To(ContainSubstring("already managed by the hpa"))

	// Without the fallback the stale zero-match skips the check.
	cacheMissToDirectClient = false
	_, err = verifyHpas(so, "create", false)
	Expect(err).ToNot(HaveOccurred())
})

// ======================== DUPLICATE-RACE PRECEDENCE TESTS ========================

var _ = It("should admit updates to the winning selector when a newer loser converged later", func() {
	namespaceName := "precedence-winner-update"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, false, false)
	workload.Labels = map[string]string{}
	winnerSO := createScaledObjectWithRef("aaa-winner", namespaceName, &ScaleTarget{
		APIVersion:    "apps/v1",
		Kind:          "Deployment",
		LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"winner": "yes"}},
	}, false, map[string]string{}, "")
	loserSO := createScaledObjectWithRef("zzz-loser", namespaceName, &ScaleTarget{
		APIVersion:    "apps/v1",
		Kind:          "Deployment",
		LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"loser": "yes"}},
	}, false, map[string]string{}, "")

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())
	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())
	// Winner first so it is older; neither selector matches yet.
	err = k8sClient.Create(context.Background(), winnerSO)
	Expect(err).ToNot(HaveOccurred())
	err = k8sClient.Create(context.Background(), loserSO)
	Expect(err).ToNot(HaveOccurred())

	// Converge both onto the workload with a label edit (no admission).
	workload.Labels = map[string]string{"winner": "yes", "loser": "yes"}
	err = k8sClient.Update(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	// Updating the winner must stay admitted even though the loser now
	// resolves to the same workload: the winner beats it at runtime.
	updated := winnerSO.DeepCopy()
	updated.Spec.PollingInterval = ptr.To[int32](31)
	_, err = updated.ValidateUpdate(winnerSO, new(false))
	Expect(err).ToNot(HaveOccurred())

	// Updating the loser is still rejected.
	updatedLoser := loserSO.DeepCopy()
	updatedLoser.Spec.PollingInterval = ptr.To[int32](31)
	_, err = updatedLoser.ValidateUpdate(loserSO, new(false))
	Expect(err).To(HaveOccurred())
	Expect(err.Error()).To(ContainSubstring("already managed by the ScaledObject"))
})

// ======================== POLLINGINTERVAL WARNING TESTS ========================

var _ = It("should emit warning when PollingInterval is set with minReplicaCount > 0 and idleReplicaCount not set", func() {
	namespaceName := "polling-interval-warning-idle-not-set"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, true)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")

	so.Spec.MinReplicaCount = ptr.To[int32](1)
	so.Spec.IdleReplicaCount = nil
	so.Spec.PollingInterval = ptr.To[int32](30)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	warnings, err := so.ValidateCreate(new(false))
	Expect(err).ToNot(HaveOccurred())
	Expect(warnings).To(ContainElement(ContainSubstring("PollingInterval is configured but is not relevant")))
})

var _ = It("should NOT emit warning when PollingInterval is set with minReplicaCount > 0 and idleReplicaCount > 0", func() {
	namespaceName := "polling-interval-no-warning-idle-greater-zero"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, true)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")

	so.Spec.MinReplicaCount = ptr.To[int32](2)
	so.Spec.IdleReplicaCount = ptr.To[int32](1)
	so.Spec.PollingInterval = ptr.To[int32](30)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	warnings, err := so.ValidateCreate(new(false))
	Expect(err).ToNot(HaveOccurred())
	Expect(warnings).ToNot(ContainElement(ContainSubstring("PollingInterval is configured but is not relevant")))
})

var _ = It("should NOT emit warning when PollingInterval is set with idleReplicaCount = 0", func() {
	namespaceName := "polling-interval-no-warning-idle-zero"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, true)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")

	so.Spec.MinReplicaCount = ptr.To[int32](2)
	so.Spec.IdleReplicaCount = ptr.To[int32](0)
	so.Spec.PollingInterval = ptr.To[int32](30)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	warnings, err := so.ValidateCreate(new(false))
	Expect(err).ToNot(HaveOccurred())
	Expect(warnings).ToNot(ContainElement(ContainSubstring("PollingInterval is configured but is not relevant")))
})

var _ = It("should NOT emit warning when PollingInterval is set with useCachedMetrics enabled", func() {
	namespaceName := "polling-interval-no-warning-cached-metrics"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, true)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")

	so.Spec.MinReplicaCount = ptr.To[int32](2)
	so.Spec.IdleReplicaCount = ptr.To[int32](1)
	so.Spec.PollingInterval = ptr.To[int32](30)
	so.Spec.Triggers = []ScaleTriggers{
		{
			Type:             "kubernetes-workload",
			UseCachedMetrics: true,
			Name:             "workload_trig_1",
			Metadata: map[string]string{
				"podSelector": "pod=workload-test",
				"value":       "1",
			},
			MetricType: v2.ValueMetricType,
		},
	}

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	warnings, err := so.ValidateCreate(new(false))
	Expect(err).ToNot(HaveOccurred())
	Expect(warnings).ToNot(ContainElement(ContainSubstring("PollingInterval is configured but is not relevant")))
})

var _ = It("should NOT emit warning when PollingInterval is set with scalingModifiers", func() {
	namespaceName := "polling-interval-no-warning-scaling-modifiers"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, true)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")

	so.Spec.MinReplicaCount = ptr.To[int32](2)
	so.Spec.IdleReplicaCount = nil
	so.Spec.PollingInterval = ptr.To[int32](30)
	so.Spec.Triggers = []ScaleTriggers{
		{
			Type: "kubernetes-workload",
			Name: "workload_trig",
			Metadata: map[string]string{
				"podSelector": "pod=workload-test",
				"value":       "1",
			},
		},
	}
	so.Spec.Advanced.ScalingModifiers = ScalingModifiers{Target: "2", Formula: "workload_trig + 1"}

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	warnings, err := so.ValidateCreate(new(false))
	Expect(err).ToNot(HaveOccurred())
	Expect(warnings).ToNot(ContainElement(ContainSubstring("PollingInterval is configured but is not relevant")))
})

var _ = It("should NOT emit warning when PollingInterval is not set", func() {
	namespaceName := "polling-interval-not-set"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, true)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")

	so.Spec.MinReplicaCount = ptr.To[int32](2)
	so.Spec.IdleReplicaCount = ptr.To[int32](1)
	so.Spec.PollingInterval = nil

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	warnings, err := so.ValidateCreate(new(false))
	Expect(err).ToNot(HaveOccurred())
	Expect(warnings).ToNot(ContainElement(ContainSubstring("PollingInterval is configured but is not relevant")))
})

// ======================== COOLDOWNPERIOD WARNING TESTS ========================

var _ = It("should emit warning when CooldownPeriod is set with minReplicaCount > 0 and idleReplicaCount not set", func() {
	namespaceName := "cooldown-warning-idle-not-set"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, true)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")

	so.Spec.MinReplicaCount = ptr.To[int32](1)
	so.Spec.IdleReplicaCount = nil
	so.Spec.CooldownPeriod = ptr.To[int32](300)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	warnings, err := so.ValidateCreate(new(false))
	Expect(err).ToNot(HaveOccurred())
	Expect(warnings).To(ContainElement(ContainSubstring("CooldownPeriod is configured but is not relevant")))
})

var _ = It("should NOT emit warning when CooldownPeriod is set with minReplicaCount > 0 and idleReplicaCount > 0", func() {
	namespaceName := "cooldown-no-warning-idle-greater-zero"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, true)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")

	so.Spec.MinReplicaCount = ptr.To[int32](2)
	so.Spec.IdleReplicaCount = ptr.To[int32](1)
	so.Spec.CooldownPeriod = ptr.To[int32](300)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	warnings, err := so.ValidateCreate(new(false))
	Expect(err).ToNot(HaveOccurred())
	Expect(warnings).ToNot(ContainElement(ContainSubstring("CooldownPeriod is configured but is not relevant")))
})

var _ = It("should NOT emit warning when CooldownPeriod is set with idleReplicaCount = 0", func() {
	namespaceName := "cooldown-no-warning-idle-zero"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, true)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")

	so.Spec.MinReplicaCount = ptr.To[int32](2)
	so.Spec.IdleReplicaCount = ptr.To[int32](0)
	so.Spec.CooldownPeriod = ptr.To[int32](300)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	warnings, err := so.ValidateCreate(new(false))
	Expect(err).ToNot(HaveOccurred())
	Expect(warnings).ToNot(ContainElement(ContainSubstring("CooldownPeriod is configured but is not relevant")))
})

var _ = It("should NOT emit warning when CooldownPeriod is not set", func() {
	namespaceName := "cooldown-not-set"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, true)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")

	so.Spec.MinReplicaCount = ptr.To[int32](2)
	so.Spec.IdleReplicaCount = ptr.To[int32](1)
	so.Spec.CooldownPeriod = nil

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	warnings, err := so.ValidateCreate(new(false))
	Expect(err).ToNot(HaveOccurred())
	Expect(warnings).ToNot(ContainElement(ContainSubstring("CooldownPeriod is configured but is not relevant")))
})

// ======================== COMBINED WARNING TESTS ========================

var _ = It("should emit both warnings when both PollingInterval and CooldownPeriod are misconfigured", func() {
	namespaceName := "both-warnings"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, true)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")

	so.Spec.MinReplicaCount = ptr.To[int32](2)
	so.Spec.IdleReplicaCount = nil
	so.Spec.PollingInterval = ptr.To[int32](30)
	so.Spec.CooldownPeriod = ptr.To[int32](300)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	warnings, err := so.ValidateCreate(new(false))
	Expect(err).ToNot(HaveOccurred())
	Expect(warnings).To(ContainElement(ContainSubstring("PollingInterval is configured but is not relevant")))
	Expect(warnings).To(ContainElement(ContainSubstring("CooldownPeriod is configured but is not relevant")))
})

var _ = It("should emit no warnings when both are properly configured with idleReplicaCount = 0", func() {
	namespaceName := "both-no-warnings-idle-zero"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, true)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")

	so.Spec.MinReplicaCount = ptr.To[int32](2)
	so.Spec.IdleReplicaCount = ptr.To[int32](0)
	so.Spec.PollingInterval = ptr.To[int32](30)
	so.Spec.CooldownPeriod = ptr.To[int32](300)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	warnings, err := so.ValidateCreate(new(false))
	Expect(err).ToNot(HaveOccurred())
	Expect(warnings).To(BeEmpty())
})

var _ = It("should validate the so creation with scalingModifiers fallback behavior", func() {
	namespaceName := "scaling-modifiers-fallback-valid"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, false, false)

	sm := ScalingModifiers{Target: "2", Formula: "trig_one ?? trig_two ?? 5"}
	triggers := []ScaleTriggers{
		{
			Type: "cron",
			Name: "trig_one",
			Metadata: map[string]string{
				"timezone":        "UTC",
				"start":           "0 * * * *",
				"end":             "1 * * * *",
				"desiredReplicas": "1",
			},
		},
		{
			Type: "kubernetes-workload",
			Name: "trig_two",
			Metadata: map[string]string{
				"podSelector": "pod=workload-test",
				"value":       "1",
			},
		},
	}

	so := createScaledObjectScalingModifiers(namespaceName, sm, triggers)
	so.Spec.Fallback = &Fallback{
		FailureThreshold: 3,
		Replicas:         5,
		Behavior:         FallbackBehaviorScalingModifiers,
	}

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())
	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).ShouldNot(HaveOccurred())
})

var _ = It("shouldn't validate the so creation with scalingModifiers fallback without formula", func() {
	namespaceName := "scaling-modifiers-fallback-no-formula"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, false, false)

	triggers := []ScaleTriggers{
		{
			Type: "cron",
			Metadata: map[string]string{
				"timezone":        "UTC",
				"start":           "0 * * * *",
				"end":             "1 * * * *",
				"desiredReplicas": "1",
			},
		},
	}

	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false, map[string]string{}, "")
	so.Spec.Triggers = triggers
	so.Spec.Fallback = &Fallback{
		FailureThreshold: 3,
		Replicas:         5,
		Behavior:         FallbackBehaviorScalingModifiers,
	}

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())
	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() error {
		return k8sClient.Create(context.Background(), so)
	}).Should(HaveOccurred())
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// -------------------------------------------------------------------------- //
// ----------------------------- HELP FUNCTIONS ----------------------------- //
// -------------------------------------------------------------------------- //

func createNamespace(name string) *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
}

func createScaledObject(name, namespace, targetName, targetAPI, targetKind string, hasCPUAndMemory bool, annotations map[string]string, hpaName string) *ScaledObject {
	return createScaledObjectWithRef(name, namespace, &ScaleTarget{
		Name:       targetName,
		APIVersion: targetAPI,
		Kind:       targetKind,
	}, hasCPUAndMemory, annotations, hpaName)
}

func createScaledObjectWithRef(name, namespace string, ref *ScaleTarget, hasCPUAndMemory bool, annotations map[string]string, hpaName string) *ScaledObject {
	triggers := []ScaleTriggers{
		{
			Type: "cron",
			Metadata: map[string]string{
				"timezone":        "UTC",
				"start":           "0 * * * *",
				"end":             "1 * * * *",
				"desiredReplicas": "1",
			},
		},
	}

	if hasCPUAndMemory {
		cpuTrigger := ScaleTriggers{
			Type: "cpu",
			Metadata: map[string]string{
				"value": "10",
			},
		}
		triggers = append(triggers, cpuTrigger)
		memoryTrigger := ScaleTriggers{
			Type: "memory",
			Metadata: map[string]string{
				"value": "10",
			},
		}
		triggers = append(triggers, memoryTrigger)
	}

	advancedConfig := &AdvancedConfig{}

	if hpaName != "" {
		advancedConfig.HorizontalPodAutoscalerConfig = &HorizontalPodAutoscalerConfig{
			Name: hpaName,
		}
	}

	return &ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			UID:         types.UID(name),
			Annotations: annotations,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "ScaledObject",
			APIVersion: "keda.sh",
		},
		Spec: ScaledObjectSpec{
			ScaleTargetRef:   ref,
			IdleReplicaCount: ptr.To[int32](1),
			MinReplicaCount:  ptr.To[int32](5),
			MaxReplicaCount:  ptr.To[int32](10),
			Triggers:         triggers,
			Advanced:         advancedConfig,
		},
	}
}

func createLabeledDeployment(namespace, name string, labels map[string]string, hasCPU, hasMemory bool) *appsv1.Deployment {
	deployment := createDeployment(namespace, hasCPU, hasMemory)
	deployment.Name = name
	deployment.Labels = labels
	return deployment
}

func createHpa(name, namespace, targetName, targetAPI, targetKind string, owner *ScaledObject) *v2.HorizontalPodAutoscaler {
	hpa := &v2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: v2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: v2.CrossVersionObjectReference{
				Name:       targetName,
				APIVersion: targetAPI,
				Kind:       targetKind,
			},
			MinReplicas: ptr.To[int32](5),
			MaxReplicas: 10,
			Metrics: []v2.MetricSpec{
				{
					Resource: &v2.ResourceMetricSource{
						Name: v1.ResourceCPU,
						Target: v2.MetricTarget{
							AverageUtilization: ptr.To[int32](30),
							Type:               v2.AverageValueMetricType,
						},
					},
					Type: v2.ResourceMetricSourceType,
				},
			},
		},
	}

	if owner != nil {
		hpa.OwnerReferences = append(hpa.OwnerReferences, metav1.OwnerReference{
			Kind:       owner.Kind,
			Name:       owner.Name,
			APIVersion: owner.APIVersion,
			UID:        owner.UID,
		})
	}

	return hpa
}

func createDeployment(namespace string, hasCPU, hasMemory bool) *appsv1.Deployment {
	cpu := 0
	if hasCPU {
		cpu = 100
	}
	memory := 0
	if hasMemory {
		memory = 100
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: workloadName, Namespace: namespace},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To[int32](1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"test": "test",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: workloadName,
					Labels: map[string]string{
						"test": "test",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "test",
							Image: "test",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(int64(cpu), resource.DecimalSI),
									v1.ResourceMemory: *resource.NewMilliQuantity(int64(memory), resource.DecimalSI),
								},
							},
						},
					},
				},
			},
		},
	}
}

func createStatefulSetWithPolicy(namespace, policy string) *appsv1.StatefulSet {
	sts := createStatefulSet(namespace, false, false)
	if policy != "" {
		sts.Spec.PodManagementPolicy = appsv1.PodManagementPolicyType(policy)
	}
	// Placement rules so order specs test order, not placement.
	sts.Spec.Template.Spec.Affinity = &v1.Affinity{
		PodAntiAffinity: &v1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
				{
					LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"test": "test"}},
					TopologyKey:   "kubernetes.io/hostname",
				},
			},
		},
	}
	return sts
}

func createStatefulSet(namespace string, hasCPU, hasMemory bool) *appsv1.StatefulSet {
	cpu := 0
	if hasCPU {
		cpu = 100
	}
	memory := 0
	if hasMemory {
		memory = 100
	}

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: workloadName, Namespace: namespace},
		Spec: appsv1.StatefulSetSpec{
			Replicas: ptr.To[int32](1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"test": "test",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: workloadName,
					Labels: map[string]string{
						"test": "test",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "test",
							Image: "test",
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(int64(cpu), resource.DecimalSI),
									v1.ResourceMemory: *resource.NewMilliQuantity(int64(memory), resource.DecimalSI),
								},
							},
						},
					},
				},
			},
		},
	}
}

func createScaledObjectSTZ(name string, namespace string, targetName string, minReplicas int32, maxReplicas int32, hasExternalTrigger bool) *ScaledObject {
	triggers := []ScaleTriggers{
		{
			Type: "cpu",
			Metadata: map[string]string{
				"value": "10",
			},
		},
	}

	if hasExternalTrigger {
		kubeWorkloadTrigger := ScaleTriggers{
			Type: "kubernetes-workload",
			Metadata: map[string]string{
				"podSelector": "pod=workload-test",
				"value":       "1",
			},
		}
		triggers = append(triggers, kubeWorkloadTrigger)
	}

	return &ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       types.UID(name),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "ScaledObject",
			APIVersion: "keda.sh",
		},
		Spec: ScaledObjectSpec{
			ScaleTargetRef: &ScaleTarget{
				Name: targetName,
			},
			MinReplicaCount: new(minReplicas),
			MaxReplicaCount: new(maxReplicas),
			CooldownPeriod:  ptr.To[int32](1),
			Triggers:        triggers,
		},
	}
}

func createScaledObjectScalingModifiers(namespace string, sm ScalingModifiers, triggers []ScaleTriggers) *ScaledObject {
	name := soName
	targetName := workloadName
	return &ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       types.UID(name),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "ScaledObject",
			APIVersion: "keda.sh",
		},
		Spec: ScaledObjectSpec{
			ScaleTargetRef: &ScaleTarget{
				Name: targetName,
			},
			MinReplicaCount: ptr.To[int32](0),
			MaxReplicaCount: ptr.To[int32](10),
			CooldownPeriod:  ptr.To[int32](1),
			Triggers:        triggers,
			Advanced: &AdvancedConfig{
				ScalingModifiers: sm,
			},
		},
	}
}

// createLimitRange creates a LimitRange resource in the specified namespace. The CPU and memory are pointers for easy
// use of constructor methods like NewQuantity, NewMilliQuantity, etc. directly at the caller.
func createLimitRange(name, namespace string, limitType v1.LimitType, cpu, memory *resource.Quantity) *v1.LimitRange {
	return &v1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.LimitRangeSpec{
			Limits: []v1.LimitRangeItem{
				{
					Type: limitType,
					Default: v1.ResourceList{
						v1.ResourceCPU:    *cpu,
						v1.ResourceMemory: *memory,
					},
				},
			},
		},
	}
}

func TestIncomingWinsDuplicateRace(t *testing.T) {
	older := metav1.NewTime(time.Now().Add(-time.Hour))
	newer := metav1.NewTime(time.Now())
	fixedRef := func() *ScaleTarget {
		return &ScaleTarget{APIVersion: "apps/v1", Kind: "Deployment", Name: "target"}
	}
	selectorRef := func() *ScaleTarget {
		return &ScaleTarget{
			APIVersion:    "apps/v1",
			Kind:          "Deployment",
			LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"team": "x"}},
		}
	}
	so := func(name string, created metav1.Time, ref *ScaleTarget) *ScaledObject {
		return &ScaledObject{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "ns", CreationTimestamp: created},
			Spec:       ScaledObjectSpec{ScaleTargetRef: ref},
		}
	}

	if incomingWinsDuplicateRace(so("new", newer, fixedRef()), so("old", older, selectorRef())) != true {
		t.Error("fixed incoming must beat selector other regardless of age")
	}
	if incomingWinsDuplicateRace(so("new", newer, selectorRef()), so("old", older, fixedRef())) != false {
		t.Error("selector incoming must lose to fixed other")
	}
	if incomingWinsDuplicateRace(so("a", newer, fixedRef()), so("b", older, fixedRef())) != false {
		t.Error("fixed-fixed duplicates must still be rejected")
	}
	if incomingWinsDuplicateRace(so("old", older, selectorRef()), so("new", newer, selectorRef())) != true {
		t.Error("older selector incoming must beat newer selector other")
	}
	if incomingWinsDuplicateRace(so("new", newer, selectorRef()), so("old", older, selectorRef())) != false {
		t.Error("newer selector incoming must lose to older selector other")
	}
	if incomingWinsDuplicateRace(so("aaa", older, selectorRef()), so("zzz", older, selectorRef())) != true {
		t.Error("equal timestamps must break ties by name")
	}
	if incomingWinsDuplicateRace(so("new", metav1.Time{}, selectorRef()), so("old", older, selectorRef())) != false {
		t.Error("create requests with zero timestamp must lose to stored objects")
	}
	if incomingWinsDuplicateRace(so("x", newer, nil), so("y", older, selectorRef())) != false {
		t.Error("nil incoming ref must lose")
	}
}
