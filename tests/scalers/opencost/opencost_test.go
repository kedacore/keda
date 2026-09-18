//go:build e2e
// +build e2e

package opencost_test

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
	testName = "opencost-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	opencostEndpoint = fmt.Sprintf("http://opencost.%s.svc.cluster.local:9003", testNamespace)
	minReplicaCount  = 0
	maxReplicaCount  = 2
)

type templateData struct {
	TestNamespace           string
	DeploymentName          string
	ScaledObjectName        string
	OpenCostEndpoint        string
	MinReplicaCount         int
	MaxReplicaCount         int
	CostThreshold           string
	ActivationCostThreshold string
}

const (
	deploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: {{.DeploymentName}}
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    matchLabels:
      app: {{.DeploymentName}}
  replicas: 0
  template:
    metadata:
      labels:
        app: {{.DeploymentName}}
    spec:
      containers:
      - name: nginx
        image: nginxinc/nginx-unprivileged
        ports:
        - containerPort: 8080
`

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  pollingInterval: 5
  cooldownPeriod: 10
  triggers:
  - type: opencost
    metadata:
      serverAddress: "{{.OpenCostEndpoint}}"
      costThreshold: "{{.CostThreshold}}"
      activationCostThreshold: "{{.ActivationCostThreshold}}"
      costType: "totalCost"
      aggregate: "namespace"
      window: "3d"
`
)

func TestOpenCostScaler(t *testing.T) {
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateNamespace(t, kc, testNamespace)

	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
		uninstallOpenCost(t)
		uninstallPrometheus(t)
		DeleteNamespace(t, testNamespace)
	})

	installPrometheus(t)
	installOpenCost(t)

	// deploy the target workload and initial ScaledObject
	KubectlApplyMultipleWithTemplate(t, data, templates)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 1 minute", minReplicaCount)

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc, data)
}

func installPrometheus(t *testing.T) {
	t.Log("--- installing prometheus ---")
	_, err := ExecuteCommand("helm repo add prometheus-community https://prometheus-community.github.io/helm-charts")
	require.NoErrorf(t, err, "cannot add prometheus helm repo - %s", err)
	_, err = ExecuteCommand("helm repo update prometheus-community")
	require.NoErrorf(t, err, "cannot update helm repo - %s", err)
	_, err = ExecuteCommand(fmt.Sprintf(
		"helm install prometheus prometheus-community/prometheus "+
			"--namespace %s "+
			"--set server.persistentVolume.enabled=false "+
			"--set alertmanager.enabled=false "+
			"--set kube-state-metrics.enabled=false "+
			"--set prometheus-node-exporter.enabled=false "+
			"--set prometheus-pushgateway.enabled=false "+
			"--wait --timeout 5m",
		testNamespace))
	require.NoErrorf(t, err, "cannot install prometheus - %s", err)
	t.Log("--- prometheus installed ---")
}

func uninstallPrometheus(t *testing.T) {
	_, err := ExecuteCommand(fmt.Sprintf("helm uninstall prometheus --namespace %s", testNamespace))
	assert.NoErrorf(t, err, "cannot uninstall prometheus - %s", err)
}

func installOpenCost(t *testing.T) {
	t.Log("--- installing opencost ---")
	_, err := ExecuteCommand("helm repo add opencost https://opencost.github.io/opencost-helm-chart")
	require.NoErrorf(t, err, "cannot add opencost helm repo - %s", err)
	_, err = ExecuteCommand("helm repo update opencost")
	require.NoErrorf(t, err, "cannot update helm repo - %s", err)
	_, err = ExecuteCommand(fmt.Sprintf(
		"helm install opencost opencost/opencost "+
			"--namespace %s "+
			"--set opencost.exporter.defaultClusterId=e2e-test "+
			"--set opencost.prometheus.internal.serviceName=prometheus-server "+
			"--set opencost.prometheus.internal.namespaceName=%s "+
			"--wait --timeout 5m",
		testNamespace, testNamespace))
	require.NoErrorf(t, err, "cannot install opencost - %s", err)
	t.Log("--- opencost installed ---")
}

func uninstallOpenCost(t *testing.T) {
	_, err := ExecuteCommand(fmt.Sprintf("helm uninstall opencost --namespace %s", testNamespace))
	assert.NoErrorf(t, err, "cannot uninstall opencost - %s", err)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing no activation with very high threshold ---")
	data.ActivationCostThreshold = "999999"
	data.CostThreshold = "999999"
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out with low threshold ---")
	// any running cluster has > $0.001/day in total namespace costs
	data.ActivationCostThreshold = "0.001"
	data.CostThreshold = "0.001"
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 180, 3),
		"replica count should be %d after scaling out", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in with high threshold ---")
	data.ActivationCostThreshold = "999999"
	data.CostThreshold = "999999"
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 180, 3),
		"replica count should be %d after scaling in", minReplicaCount)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:           testNamespace,
			DeploymentName:          deploymentName,
			ScaledObjectName:        scaledObjectName,
			OpenCostEndpoint:        opencostEndpoint,
			MinReplicaCount:         minReplicaCount,
			MaxReplicaCount:         maxReplicaCount,
			CostThreshold:           "999999",
			ActivationCostThreshold: "999999",
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
