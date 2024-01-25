/*
Copyright 2024 The KEDA Authors

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
	"strconv"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc

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
	done := make(chan interface{})
	go func() {
		defer GinkgoRecover()
		cfg, err = testEnv.Start()
		close(done)
	}()
	Eventually(done).WithTimeout(time.Minute).Should(BeClosed())
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
		Scheme: scheme,
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    webhookInstallOptions.LocalServingHost,
			Port:    webhookInstallOptions.LocalServingPort,
			CertDir: webhookInstallOptions.LocalServingCertDir,
		}),
		LeaderElection: false,
		Metrics: server.Options{
			BindAddress: "0",
		},
	})
	Expect(err).NotTo(HaveOccurred())

	err = (&CloudEventSource{}).SetupWebhookWithManager(mgr)
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

var _ = It("validate cloudeventsource when event type is not support", func() {
	namespaceName := "nscloudeventnotsupport"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	spec := createCloudEventSourceSpecWithExcludeEventType("keda.scaledobject.ready.v1.test")
	ces := createCloudEventSource("nsccesexcludenotsupport", namespaceName, spec)
	Eventually(func() error {
		return k8sClient.Create(context.Background(), ces)
	}).Should(HaveOccurred())

	spec = createCloudEventSourceSpecWithIncludeEventType("keda.scaledobject.ready.v1.test")
	ces = createCloudEventSource("nsccesincludenotsupport", namespaceName, spec)
	Eventually(func() error {
		return k8sClient.Create(context.Background(), ces)
	}).Should(HaveOccurred())
})

var _ = It("validate cloudeventsource when event type is support", func() {
	namespaceName := "cloudeventtestns"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	for k, eventType := range AllEventTypes {
		spec := createCloudEventSourceSpecWithExcludeEventType(eventType)
		ces := createCloudEventSource("cloudeventexclude"+strconv.Itoa(k), namespaceName, spec)
		Eventually(func() error {
			return k8sClient.Create(context.Background(), ces)
		}).ShouldNot(HaveOccurred())
	}

	for k, eventType := range AllEventTypes {
		spec := createCloudEventSourceSpecWithIncludeEventType(eventType)
		ces := createCloudEventSource("cloudeventinclude"+strconv.Itoa(k), namespaceName, spec)
		Eventually(func() error {
			return k8sClient.Create(context.Background(), ces)
		}).ShouldNot(HaveOccurred())
	}
})

// -------------------------------------------------------------------------- //
// ----------------------------- HELP FUNCTIONS ----------------------------- //
// -------------------------------------------------------------------------- //

func createNamespace(name string) *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
}

func createCloudEventSourceSpecWithExcludeEventType(eventtype string) CloudEventSourceSpec {
	return CloudEventSourceSpec{
		EventSubscription: EventSubscription{
			ExcludedEventTypes: []string{eventtype},
		},
	}
}

func createCloudEventSourceSpecWithIncludeEventType(eventtype string) CloudEventSourceSpec {
	return CloudEventSourceSpec{
		EventSubscription: EventSubscription{
			IncludedEventTypes: []string{eventtype},
		},
	}
}

func createCloudEventSource(name string, namespace string, spec CloudEventSourceSpec) *CloudEventSource {
	return &CloudEventSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "CloudEventSource",
			APIVersion: "eventing.keda.sh",
		},
		Spec: spec,
	}
}
