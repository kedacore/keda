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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	err = k8sClient.Create(context.Background(), so)
	Expect(err).To(HaveOccurred())
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

	err = k8sClient.Create(context.Background(), so)
	Expect(err).To(HaveOccurred())
})

var _ = It("shouldn't validate the so creation When the fallback are configured and the scaler is either CPU or memory.", func() {
	namespaceName := "wrong-fallback-cpu-memory"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, true)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", true, map[string]string{}, "")
	so.Spec.Fallback = &Fallback{
		FailureThreshold: 3,
		Replicas:         6,
	}
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), so)
	Expect(err).To(HaveOccurred())
})

var _ = It("shouldn't validate the so creation when there is another unmanaged hpa and so has transfer-hpa-ownership activated", func() {

	hpaName := "test-unmanaged-hpa-ownership"
	namespaceName := "unmanaged-hpa-ownership"
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

var _ = It("shouldn't validate the so creation when hpa has shared-ownership unactivated", func() {

	hpaName := "test-hpa-disabled-validation-by-hpa-ownership"
	namespaceName := "hpa-ownership"
	namespace := createNamespace(namespaceName)
	hpa := createHpa(hpaName, namespaceName, workloadName, "apps/v1", "Deployment", nil)
	hpa.ObjectMeta.Annotations = map[string]string{ValidationsHpaOwnershipAnnotation: "false"}
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
	so := createScaledObject(soName, namespaceName, workloadName, "custom-api", "custom-kind", false, map[string]string{}, "")
	hpa := createHpa(hpaName, namespaceName, workloadName, "custom-api", "custom-kind", nil)

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
	so.ObjectMeta.Finalizers = append(so.ObjectMeta.Finalizers, "finalizer")

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

	so.ObjectMeta.Finalizers = []string{}
	Eventually(func() error {
		return k8sClient.Update(context.Background(), so)
	}).ShouldNot(HaveOccurred())
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
			ScaleTargetRef: &ScaleTarget{
				Name:       targetName,
				APIVersion: targetAPI,
				Kind:       targetKind,
			},
			IdleReplicaCount: ptr.To[int32](1),
			MinReplicaCount:  ptr.To[int32](5),
			MaxReplicaCount:  ptr.To[int32](10),
			Triggers:         triggers,
			Advanced:         advancedConfig,
		},
	}
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
			MinReplicaCount: ptr.To[int32](minReplicas),
			MaxReplicaCount: ptr.To[int32](maxReplicas),
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
