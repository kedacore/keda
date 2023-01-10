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
	"crypto/tls"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	v2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc

const (
	workloadName = "deployment-name"
	soName       = "test-so"
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Webhook Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.Background())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: false,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "..", "..", "config", "webhooks")},
		},
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	scheme := runtime.NewScheme()
	err = AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = clientgoscheme.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = admissionv1beta1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// start webhook server using Manager
	webhookInstallOptions := &testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme,
		Host:               webhookInstallOptions.LocalServingHost,
		Port:               webhookInstallOptions.LocalServingPort,
		CertDir:            webhookInstallOptions.LocalServingCertDir,
		LeaderElection:     false,
		MetricsBindAddress: "0",
	})
	Expect(err).NotTo(HaveOccurred())

	err = (&ScaledObject{}).SetupWebhookWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:webhook

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()

	// wait for the webhook server to get ready
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", webhookInstallOptions.LocalServingHost, webhookInstallOptions.LocalServingPort)
	Eventually(func() error {
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return err
		}
		conn.Close()
		return nil
	}).Should(Succeed())

})

var _ = It("should validate the so creation when there isn't any hpa", func() {

	namespaceName := "valid"
	namespace := createNamespace(namespaceName)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), so)
	Expect(err).ToNot(HaveOccurred())
})

var _ = It("should validate the so creation when there are other SO for other workloads", func() {

	namespaceName := "valid-multiple-so"
	namespace := createNamespace(namespaceName)
	so1 := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false)
	so2 := createScaledObject("other-so-name", namespaceName, "other-workload", "apps/v1", "Deployment", false)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), so1)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), so2)
	Expect(err).ToNot(HaveOccurred())
})

var _ = It("should validate the so creation when there are other HPA for other workloads", func() {

	namespaceName := "valid-other-hpa"
	namespace := createNamespace(namespaceName)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false)
	hpa := createHpa("other-hpa-name", namespaceName, "other-workload", "apps/v1", "Deployment", nil)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), hpa)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), so)
	Expect(err).ToNot(HaveOccurred())
})

var _ = It("should validate the so creation when it's own hpa is already generated", func() {

	hpaName := "test-so-hpa"
	namespaceName := "own-hpa"
	namespace := createNamespace(namespaceName)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false)
	hpa := createHpa(hpaName, namespaceName, workloadName, "apps/v1", "Deployment", so)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), hpa)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), so)
	Expect(err).ToNot(HaveOccurred())
})

var _ = It("should validate the so update when it's own hpa is already generated", func() {

	hpaName := "test-so-hpa"
	namespaceName := "update-so"
	namespace := createNamespace(namespaceName)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false)
	hpa := createHpa(hpaName, namespaceName, workloadName, "apps/v1", "Deployment", so)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), hpa)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), so)
	Expect(err).ToNot(HaveOccurred())

	so.Spec.MaxReplicaCount = pointer.Int32(7)
	err = k8sClient.Update(context.Background(), so)
	Expect(err).ToNot(HaveOccurred())
})

var _ = It("shouldn't validate the so creation when there is another unmanaged hpa", func() {

	hpaName := "test-unmanaged-hpa"
	namespaceName := "unmanaged-hpa"
	namespace := createNamespace(namespaceName)
	hpa := createHpa(hpaName, namespaceName, workloadName, "apps/v1", "Deployment", nil)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), hpa)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), so)
	Expect(err).To(HaveOccurred())
})

var _ = It("shouldn't validate the so creation when there is another so", func() {

	so2Name := "test-so2"
	namespaceName := "managed-hpa"
	namespace := createNamespace(namespaceName)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", false)
	so2 := createScaledObject(so2Name, namespaceName, workloadName, "apps/v1", "Deployment", false)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), so2)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), so)
	Expect(err).To(HaveOccurred())
})

var _ = It("shouldn't validate the so creation when there is another hpa with custom apis", func() {

	hpaName := "test-custom-hpa"
	namespaceName := "custom-apis"
	namespace := createNamespace(namespaceName)
	so := createScaledObject(soName, namespaceName, workloadName, "custom-api", "custom-kind", false)
	hpa := createHpa(hpaName, namespaceName, workloadName, "custom-api", "custom-kind", nil)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), hpa)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), so)
	Expect(err).To(HaveOccurred())
})

var _ = It("should validate the so creation with cpu and memory when deployment has requests", func() {

	namespaceName := "deployment-has-requests"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, true)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", true)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), so)
	Expect(err).ToNot(HaveOccurred())
})

var _ = It("shouldn't validate the so creation with cpu and memory when deployment hasn't got memory request", func() {

	namespaceName := "deployment-no-memory-request"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, true, false)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", true)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), so)
	Expect(err).To(HaveOccurred())
})

var _ = It("shouldn't validate the so creation with cpu and memory when deployment hasn't got cpu request", func() {

	namespaceName := "deployment-no-cpu-request"
	namespace := createNamespace(namespaceName)
	workload := createDeployment(namespaceName, false, true)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "Deployment", true)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), so)
	Expect(err).To(HaveOccurred())
})

var _ = It("should validate the so creation with cpu and memory when statefulset has requests", func() {

	namespaceName := "statefulset-has-requests"
	namespace := createNamespace(namespaceName)
	workload := createStatefulSet(namespaceName, true, true)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "StatefulSet", true)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), so)
	Expect(err).ToNot(HaveOccurred())
})

var _ = It("shouldn't validate the so creation with cpu and memory when statefulset hasn't got memory request", func() {

	namespaceName := "statefulset-no-memory-request"
	namespace := createNamespace(namespaceName)
	workload := createStatefulSet(namespaceName, true, false)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "StatefulSet", true)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), so)
	Expect(err).To(HaveOccurred())
})

var _ = It("shouldn't validate the so creation with cpu and memory when statefulset hasn't got cpu request", func() {

	namespaceName := "statefulset-no-cpu-request"
	namespace := createNamespace(namespaceName)
	workload := createStatefulSet(namespaceName, false, true)
	so := createScaledObject(soName, namespaceName, workloadName, "apps/v1", "StatefulSet", true)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), workload)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), so)
	Expect(err).To(HaveOccurred())
})

var _ = It("should validate the so creation without cpu and memory when custom resources", func() {

	namespaceName := "crd-not-resources"
	namespace := createNamespace(namespaceName)
	so := createScaledObject(soName, namespaceName, workloadName, "custom-api", "StatefulSet", true)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.Create(context.Background(), so)
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func createNamespace(name string) *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
}

func createScaledObject(name, namespace, targetName, targetAPI, targetKind string, hasCPUAndMemory bool) *ScaledObject {
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
				Name:       targetName,
				APIVersion: targetAPI,
				Kind:       targetKind,
			},
			IdleReplicaCount: pointer.Int32(1),
			MinReplicaCount:  pointer.Int32(5),
			MaxReplicaCount:  pointer.Int32(10),
			Triggers:         triggers,
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
			MinReplicas: pointer.Int32(5),
			MaxReplicas: 10,
			Metrics: []v2.MetricSpec{
				{
					Resource: &v2.ResourceMetricSource{
						Name: v1.ResourceCPU,
						Target: v2.MetricTarget{
							AverageUtilization: pointer.Int32(30),
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
			Replicas: pointer.Int32(1),
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
			Replicas: pointer.Int32(1),
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
