//go:build e2e
// +build e2e

package subresource_scale_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "subresource-scale-test"
)

var (
	testNamespace           = fmt.Sprintf("%s-ns", testName)
	monitoredDeploymentName = fmt.Sprintf("%s-monitored", testName)
	argoRolloutName         = fmt.Sprintf("%s-rollout", testName)
	scaledObjectName        = fmt.Sprintf("%s-so", testName)
)

type templateData struct {
	TestNamespace           string
	MonitoredDeploymentName string
	ArgoRolloutName         string
	ScaledObjectName        string
}

const (
	monitoredDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.MonitoredDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.MonitoredDeploymentName}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: {{.MonitoredDeploymentName}}
  template:
    metadata:
      labels:
        app: {{.MonitoredDeploymentName}}
    spec:
      containers:
        - name: nginx
          image: 'ghcr.io/nginx/nginx-unprivileged:1.26'`

	argoRolloutTemplate = `apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: {{.ArgoRolloutName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.ArgoRolloutName}}
spec:
  replicas: 0
  strategy:
    canary:
      steps:
        - setWeight: 50
        - pause: {duration: 10}
  selector:
    matchLabels:
      app: {{.ArgoRolloutName}}
  template:
    metadata:
      labels:
        app: {{.ArgoRolloutName}}
    spec:
      containers:
        - name: nginx
          image: ghcr.io/nginx/nginx-unprivileged:1.26
`

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    apiVersion: argoproj.io/v1alpha1
    kind: Rollout
    name: {{.ArgoRolloutName}}
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
      podSelector: 'app={{.MonitoredDeploymentName}}'
      value: '1'
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		// cleanup
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	assert.True(t, WaitForArgoRolloutReplicaReadyCount(t, kc, argoRolloutName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testScaleOut(t, kc)
	testScaleIn(t, kc)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")

	// scale monitored deployment to 5 replicas
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 5, testNamespace)
	assert.True(t, WaitForArgoRolloutReplicaReadyCount(t, kc, argoRolloutName, testNamespace, 5, 60, 1),
		"replica count should be 5 after 1 minute")

	// scale monitored deployment to 10 replicas
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 10, testNamespace)
	assert.True(t, WaitForArgoRolloutReplicaReadyCount(t, kc, argoRolloutName, testNamespace, 10, 60, 1),
		"replica count should be 10 after 1 minute")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	// scale monitored deployment to 5 replicas
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 5, testNamespace)
	assert.True(t, WaitForArgoRolloutReplicaReadyCount(t, kc, argoRolloutName, testNamespace, 5, 60, 1),
		"replica count should be 5 after 1 minute")

	// scale monitored deployment to 0 replicas
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 0, testNamespace)
	assert.True(t, WaitForArgoRolloutReplicaReadyCount(t, kc, argoRolloutName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:           testNamespace,
			MonitoredDeploymentName: monitoredDeploymentName,
			ArgoRolloutName:         argoRolloutName,
			ScaledObjectName:        scaledObjectName,
		}, []Template{
			{Name: "monitoredDeploymentTemplate", Config: monitoredDeploymentTemplate},
			{Name: "argoRolloutTemplate", Config: argoRolloutTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
