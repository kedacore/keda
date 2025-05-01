//go:build e2e
// +build e2e

package kubernetes_workload_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "kubernetes-workload-test"
)

var (
	testNamespace           = fmt.Sprintf("%s-ns", testName)
	monitoredDeploymentName = "monitored-deployment"
	sutDeploymentName       = "sut-deployment"
	scaledObjectName        = fmt.Sprintf("%s-so", testName)
)

type templateData struct {
	TestNamespace           string
	MonitoredDeploymentName string
	SutDeploymentName       string
	ScaledObjectName        string
}

const (
	monitoredDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.MonitoredDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    deploy: workload-test
spec:
  replicas: 0
  selector:
    matchLabels:
      pod: workload-test
  template:
    metadata:
      labels:
        pod: workload-test
    spec:
      containers:
        - name: nginx
          image: 'ghcr.io/nginx/nginx-unprivileged:1.26'`

	sutDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.SutDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    deploy: workload-sut
spec:
  replicas: 0
  selector:
    matchLabels:
      pod: workload-sut
  template:
    metadata:
      labels:
        pod: workload-sut
    spec:
      containers:
      - name: nginx
        image: 'ghcr.io/nginx/nginx-unprivileged:1.26'`

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.SutDeploymentName}}
  pollingInterval: 5
  cooldownPeriod: 5
  minReplicaCount: 0
  maxReplicaCount: 10
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 5
  triggers:
  - type: kubernetes-workload
    metadata:
      podSelector: 'pod=workload-test'
      value: '1'
      activationValue: '3'`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, monitoredDeploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, sutDeploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testActivation(t, kc)
	testScaleOut(t, kc)
	testScaleIn(t, kc)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset) {
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 2, testNamespace)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, sutDeploymentName, testNamespace, 0, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	// scale monitored deployment to 5 replicas
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 5, testNamespace)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, sutDeploymentName, testNamespace, 5, 60, 2),
		"replica count should be 5 after 1 minute")

	// scale monitored deployment to 10 replicas
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 10, testNamespace)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, sutDeploymentName, testNamespace, 10, 60, 2),
		"replica count should be 10 after 1 minute")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	// scale monitored deployment to 5 replicas
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 5, testNamespace)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, sutDeploymentName, testNamespace, 5, 60, 2),
		"replica count should be 5 after 1 minute")

	// scale monitored deployment to 0 replicas
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 0, testNamespace)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, sutDeploymentName, testNamespace, 0, 60, 2),
		"replica count should be 0 after 1 minute")
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:           testNamespace,
			MonitoredDeploymentName: monitoredDeploymentName,
			SutDeploymentName:       sutDeploymentName,
			ScaledObjectName:        scaledObjectName,
		}, []Template{
			{Name: "monitoredDeploymentTemplate", Config: monitoredDeploymentTemplate},
			{Name: "sutDeploymentTemplate", Config: sutDeploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
