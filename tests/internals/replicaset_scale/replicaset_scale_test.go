//go:build e2e
// +build e2e

package replicaset_scale_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "replicaset-scale-test"
)

var (
	testNamespace           = fmt.Sprintf("%s-ns", testName)
	monitoredDeploymentName = fmt.Sprintf("%s-monitored", testName)
	replicaSetName          = fmt.Sprintf("%s-rs", testName)
	scaledObjectName        = fmt.Sprintf("%s-so", testName)
)

type templateData struct {
	TestNamespace           string
	MonitoredDeploymentName string
	ReplicaSetName          string
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

	replicaSetTemplate = `apiVersion: apps/v1
kind: ReplicaSet
metadata:
  name: {{.ReplicaSetName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.ReplicaSetName}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: {{.ReplicaSetName}}
  template:
    metadata:
      labels:
        app: {{.ReplicaSetName}}
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
    apiVersion: apps/v1
    kind: ReplicaSet
    name: {{.ReplicaSetName}}
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

func TestReplicaSetScaling(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	assert.True(t, WaitForReplicaSetReplicaReadyCount(t, kc, replicaSetName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testScaleOut(t, kc)
	testScaleIn(t, kc)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")

	// scale monitored deployment to 5 replicas
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 5, testNamespace)
	assert.True(t, WaitForReplicaSetReplicaReadyCount(t, kc, replicaSetName, testNamespace, 5, 60, 1),
		"replica count should be 5 after 1 minute")

	// scale monitored deployment to 10 replicas
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 10, testNamespace)
	assert.True(t, WaitForReplicaSetReplicaReadyCount(t, kc, replicaSetName, testNamespace, 10, 60, 1),
		"replica count should be 10 after 1 minute")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	// scale monitored deployment to 5 replicas
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 5, testNamespace)
	assert.True(t, WaitForReplicaSetReplicaReadyCount(t, kc, replicaSetName, testNamespace, 5, 60, 1),
		"replica count should be 5 after 1 minute")

	// scale monitored deployment to 0 replicas
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 0, testNamespace)
	assert.True(t, WaitForReplicaSetReplicaReadyCount(t, kc, replicaSetName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:           testNamespace,
			MonitoredDeploymentName: monitoredDeploymentName,
			ReplicaSetName:          replicaSetName,
			ScaledObjectName:        scaledObjectName,
		}, []Template{
			{Name: "monitoredDeploymentTemplate", Config: monitoredDeploymentTemplate},
			{Name: "replicaSetTemplate", Config: replicaSetTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
