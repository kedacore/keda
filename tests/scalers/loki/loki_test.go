//go:build e2e
// +build e2e

package loki_test

import (
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "loki-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	lokiServerName   = fmt.Sprintf("%s-server", testName)
	minReplicaCount  = 0
	maxReplicaCount  = 2
)

type templateData struct {
	TestNamespace    string
	DeploymentName   string
	ScaledObjectName string
	LokiServerName   string
	BackTick         string
	MinReplicaCount  int
	MaxReplicaCount  int
}

const (
	deploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: test-app
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: test-app
  template:
    metadata:
      labels:
        app: test-app
        type: keda-testing
    spec:
      containers:
      - name: nginx
        image: ghcr.io/nginx/nginx-unprivileged:1.26
        ports:
        - containerPort: 80
---
`

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  pollingInterval: 3
  cooldownPeriod:  1
  triggers:
  - type: loki
    metadata:
      serverAddress: http://loki.{{.TestNamespace}}.svc.cluster.local:3100
      threshold: '0.85'
      activationThreshold: '0.85'
      query: sum(rate({namespace="{{.TestNamespace}}"} |= {{.BackTick}}keda-scaler{{.BackTick}} != {{.BackTick}}level=info{{.BackTick}} [1m]))
`

	generateLowLevelLoadJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: generate-low-level-requests-job
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-hey
        name: test
        command: ["/bin/sh"]
        args: ["-c", "for i in $(seq 1 30);do echo \"keda-scaler $i\";sleep 1;done"]
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          capabilities:
            drop:
              - ALL
          seccompProfile:
            type: RuntimeDefault
      restartPolicy: Never
  activeDeadlineSeconds: 100
  backoffLimit: 2
  `

	generateLoadJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: generate-requests-job
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-hey
        name: test
        command: ["/bin/sh"]
        args: ["-c", "for i in $(seq 1 30);do echo \"keda-scaler $i\";echo \"keda-scaler $((i*2))\";sleep 1;done"]
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          capabilities:
            drop:
              - ALL
          seccompProfile:
            type: RuntimeDefault
      restartPolicy: Never
  activeDeadlineSeconds: 100
  backoffLimit: 2
`
)

// TestLokiScaler creates deployments - there are two deployments - both using the same image but one deployment
// is directly tied to the KEDA HPA while the other is isolated that can be used for metrics
// even when the KEDA deployment is at zero - the service points to both deployments
func TestLokiScaler(t *testing.T) {
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		uninstallLoki(t, testNamespace)
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	installLoki(t, kc, testNamespace)

	// Create kubernetes resources for testing
	KubectlApplyMultipleWithTemplate(t, data, templates)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	KubectlReplaceWithTemplate(t, data, "generateLowLevelLoadJobTemplate", generateLowLevelLoadJobTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlReplaceWithTemplate(t, data, "generateLoadJobTemplate", generateLoadJobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 5),
		"replica count should be %d after 5 minutes", minReplicaCount)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:    testNamespace,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
			LokiServerName:   lokiServerName,
			MinReplicaCount:  minReplicaCount,
			MaxReplicaCount:  maxReplicaCount,
			BackTick:         "`",
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func installLoki(t *testing.T, kc *kubernetes.Clientset, namespace string) {
	CreateNamespace(t, kc, namespace)
	_, err := ExecuteCommand("helm repo add grafana https://grafana.github.io/helm-charts")
	require.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand("helm repo update")
	require.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand(fmt.Sprintf("helm upgrade --install loki grafana/loki-stack --wait --namespace=%s", namespace))
	require.NoErrorf(t, err, "cannot execute command - %s", err)
}

func uninstallLoki(t *testing.T, namespace string) {
	_, err := ExecuteCommand(fmt.Sprintf("helm uninstall loki --wait --namespace=%s", namespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}
