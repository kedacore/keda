//go:build e2e
// +build e2e

package subresource_scale_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "subresource-scale-test"
)

var (
	testNamespace           = fmt.Sprintf("%s-ns", testName)
	argoNamespace           = "argo-rollouts"
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
		cleanupArgo(t)
	})
	setupArgo(t, kc)

	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	assert.True(t, waitForArgoRolloutReplicaCount(t, argoRolloutName, testNamespace, 0),
		"replica count should be 0 after 1 minute")

	// test scaling
	testScaleOut(t, kc)
	testScaleIn(t, kc)
}

func setupArgo(t *testing.T, kc *kubernetes.Clientset) {
	CreateNamespace(t, kc, argoNamespace)
	cmdWithNamespace := fmt.Sprintf("kubectl apply -n %s -f https://github.com/argoproj/argo-rollouts/releases/latest/download/install.yaml",
		argoNamespace)
	_, err := ExecuteCommand(cmdWithNamespace)

	require.NoErrorf(t, err, "cannot install argo resources - %s", err)
}

func cleanupArgo(t *testing.T) {
	cmdWithNamespace := fmt.Sprintf("kubectl delete -n %s -f https://github.com/argoproj/argo-rollouts/releases/latest/download/install.yaml",
		argoNamespace)
	_, err := ExecuteCommand(cmdWithNamespace)

	assert.NoErrorf(t, err, "cannot delete argo resources - %s", err)
	DeleteNamespace(t, argoNamespace)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")

	// scale monitored deployment to 5 replicas
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 5, testNamespace)
	assert.True(t, waitForArgoRolloutReplicaCount(t, argoRolloutName, testNamespace, 5),
		"replica count should be 5 after 1 minute")

	// scale monitored deployment to 10 replicas
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 10, testNamespace)
	assert.True(t, waitForArgoRolloutReplicaCount(t, argoRolloutName, testNamespace, 10),
		"replica count should be 10 after 1 minute")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	// scale monitored deployment to 5 replicas
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 5, testNamespace)
	assert.True(t, waitForArgoRolloutReplicaCount(t, argoRolloutName, testNamespace, 5),
		"replica count should be 5 after 1 minute")

	// scale monitored deployment to 0 replicas
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 0, testNamespace)
	assert.True(t, waitForArgoRolloutReplicaCount(t, argoRolloutName, testNamespace, 0),
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

func waitForArgoRolloutReplicaCount(t *testing.T, name, namespace string, target int) bool {
	for i := 0; i < 60; i++ {
		kctlGetCmd := fmt.Sprintf(`kubectl get rollouts.argoproj.io/%s -n %s -o jsonpath="{.spec.replicas}"`, argoRolloutName, namespace)
		output, err := ExecuteCommand(kctlGetCmd)

		assert.NoErrorf(t, err, "cannot get rollout info - %s", err)

		unqoutedOutput := strings.ReplaceAll(string(output), "\"", "")
		replicas, err := strconv.ParseInt(unqoutedOutput, 10, 64)
		assert.NoErrorf(t, err, "cannot convert rollout count to int - %s", err)

		t.Logf("Waiting for rollout replicas to hit target. Deployment - %s, Current  - %d, Target - %d",
			name, replicas, target)

		if replicas == int64(target) {
			return true
		}

		time.Sleep(time.Second)
	}

	return false
}
