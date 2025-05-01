//go:build e2e
// +build e2e

package azure_log_analytics_aad_wi_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName = "azure-log-analytics-aad-wi-test"
)

var (
	logAnalyticsWorkspaceID = os.Getenv("TF_AZURE_LOG_ANALYTICS_WORKSPACE_ID")
	testNamespace           = fmt.Sprintf("%s-ns", testName)
	deploymentName          = fmt.Sprintf("%s-deployment", testName)
	triggerAuthName         = fmt.Sprintf("%s-ta", testName)
	scaledObjectName        = fmt.Sprintf("%s-so", testName)
)

type templateData struct {
	TestNamespace           string
	DeploymentName          string
	TriggerAuthName         string
	ScaledObjectName        string
	LogAnalyticsWorkspaceID string
	QueryX, QueryY          int
}

const (
	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: {{.DeploymentName}}
  template:
    metadata:
      labels:
        app: {{.DeploymentName}}
    spec:
      containers:
        - name: {{.DeploymentName}}
          image: ghcr.io/nginx/nginx-unprivileged:1.26
`

	triggerAuthTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: azure-workload
`

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  labels:
    deploymentName: {{.DeploymentName}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  cooldownPeriod: 5
  pollingInterval: 5
  maxReplicaCount: 2
  triggers:
    - type: azure-log-analytics
      metadata:
        workspaceId: {{.LogAnalyticsWorkspaceID}}
        query: "let x = {{.QueryX}}; let y = {{.QueryY}}; print MetricValue = x, Threshold = y;"
        threshold: "1"
        activationThreshold: "3"
      authenticationRef:
        name: {{.TriggerAuthName}}
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	require.NotEmpty(t, logAnalyticsWorkspaceID, "TF_AZURE_LOG_ANALYTICS_WORKSPACE_ID env variable is required for deployment bus tests")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc, data)

	// cleanup
	templates = append(templates, Template{Name: "triggerAuthTemplate", Config: triggerAuthTemplate})
	templates = append(templates, Template{Name: "scaledObjectTemplate", Config: scaledObjectTemplate})
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	data.QueryX = 2
	data.QueryY = 1

	KubectlApplyWithTemplate(t, data, "triggerAuthTemplate", triggerAuthTemplate)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	data.QueryX = 10
	data.QueryY = 1

	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 1),
		"replica count should be 2 after 1 minute")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in ---")
	data.QueryX = 0

	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:           testNamespace,
			DeploymentName:          deploymentName,
			TriggerAuthName:         triggerAuthName,
			ScaledObjectName:        scaledObjectName,
			LogAnalyticsWorkspaceID: logAnalyticsWorkspaceID,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
		}
}
