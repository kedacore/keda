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
	testNamespace       = fmt.Sprintf("%s-ns", testName)
	deploymentName      = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName    = fmt.Sprintf("%s-so", testName)
	opencostReleaseName = fmt.Sprintf("%s-opencost", testName)
	opencostEndpoint    = fmt.Sprintf("http://%s.%s.svc.cluster.local:9003", opencostReleaseName, testNamespace)
	minReplicaCount     = 0
	maxReplicaCount     = 2
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
      window: "1d"
`
)

func TestOpenCostScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	// Install OpenCost using Helm
	installOpenCost(t, kc)

	// Create test resources
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 180, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	// test scaling based on OpenCost metrics
	// OpenCost will report actual cluster costs, so we test with very high thresholds
	// to ensure the scaler is working correctly
	testScaling(t, kc, data)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
	uninstallOpenCost(t)
}

func installOpenCost(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- installing OpenCost ---")
	CreateNamespace(t, kc, testNamespace)

	// Add OpenCost Helm repo
	_, err := ExecuteCommand("helm repo add opencost https://opencost.github.io/opencost-helm-chart")
	require.NoErrorf(t, err, "cannot add opencost helm repo - %s", err)

	_, err = ExecuteCommand("helm repo update")
	require.NoErrorf(t, err, "cannot update helm repos - %s", err)

	// Install OpenCost with minimal configuration for testing
	// Disable Prometheus since we just need the API endpoint
	_, err = ExecuteCommand(fmt.Sprintf(
		`helm install --wait --timeout 300s %s opencost/opencost --namespace %s `+
			`--set opencost.prometheus.internal.enabled=false `+
			`--set opencost.prometheus.external.enabled=false `+
			`--set opencost.ui.enabled=false`,
		opencostReleaseName,
		testNamespace))
	require.NoErrorf(t, err, "cannot install opencost - %s", err)

	// Wait for OpenCost to be ready
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, opencostReleaseName, testNamespace, 1, 120, 3),
		"opencost deployment should be ready")
}

func uninstallOpenCost(t *testing.T) {
	t.Log("--- uninstalling OpenCost ---")
	_, err := ExecuteCommand(fmt.Sprintf(
		`helm uninstall --wait --timeout 120s %s --namespace %s`,
		opencostReleaseName,
		testNamespace))
	assert.NoErrorf(t, err, "cannot uninstall opencost - %s", err)
	DeleteNamespace(t, testNamespace)
}

func testScaling(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scaling ---")

	// Test 1: With very high activation threshold, should NOT activate
	// OpenCost will report some cost (even if minimal), but with threshold of 999999
	// it should not trigger scaling
	t.Log("--- testing no activation with high threshold ---")
	data.ActivationCostThreshold = "999999"
	data.CostThreshold = "999999"
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)

	// Test 2: With very low activation threshold, should activate and scale
	// Any cluster will have some cost > 0, so threshold of 1 should trigger
	t.Log("--- testing scale out with low threshold ---")
	data.ActivationCostThreshold = "1"
	data.CostThreshold = "1"
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 120, 3),
		"replica count should be %d after scaling", maxReplicaCount)

	// Test 3: With high threshold again, should scale back down
	t.Log("--- testing scale in with high threshold ---")
	data.ActivationCostThreshold = "999999"
	data.CostThreshold = "999999"
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 120, 3),
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
