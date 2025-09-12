//go:build e2e
// +build e2e

package azure_log_analytics_test

import (
	"encoding/base64"
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
	testName = "azure-log-analytics-test"
)

var (
	logAnalyticsWorkspaceID = os.Getenv("TF_AZURE_LOG_ANALYTICS_WORKSPACE_ID")
	azureADClientID         = os.Getenv("TF_AZURE_SP_APP_ID")
	azureADSecret           = os.Getenv("AZURE_SP_KEY")
	azureADTenantID         = os.Getenv("TF_AZURE_SP_TENANT")
	testNamespace           = fmt.Sprintf("%s-ns", testName)
	secretName              = fmt.Sprintf("%s-secret", testName)
	deploymentName          = fmt.Sprintf("%s-deployment", testName)
	triggerAuthName         = fmt.Sprintf("%s-ta", testName)
	scaledObjectName        = fmt.Sprintf("%s-so", testName)
)

type templateData struct {
	TestNamespace           string
	SecretName              string
	DeploymentName          string
	TriggerAuthName         string
	ScaledObjectName        string
	AzureADClientID         string
	AzureADSecret           string
	AzureADTenantID         string
	LogAnalyticsWorkspaceID string
	QueryX, QueryY          int
}

const (
	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  clientSecret: {{.AzureADSecret}}
`

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
  secretTargetRef:
    - parameter: clientSecret
      name: {{.SecretName}}
      key: clientSecret
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
        clientId: {{.AzureADClientID}}
        tenantId: {{.AzureADTenantID}}
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
	require.NotEmpty(t, azureADClientID, "TF_AZURE_SP_APP_ID env variable is required for deployment bus tests")
	require.NotEmpty(t, azureADSecret, "AZURE_SP_KEY env variable is required for deployment bus tests")
	require.NotEmpty(t, azureADTenantID, "TF_AZURE_SP_TENANT env variable is required for deployment bus tests")

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
	base64ClientSecret := base64.StdEncoding.EncodeToString([]byte(azureADSecret))

	return templateData{
			TestNamespace:           testNamespace,
			SecretName:              secretName,
			DeploymentName:          deploymentName,
			TriggerAuthName:         triggerAuthName,
			ScaledObjectName:        scaledObjectName,
			AzureADClientID:         azureADClientID,
			AzureADSecret:           base64ClientSecret,
			AzureADTenantID:         azureADTenantID,
			LogAnalyticsWorkspaceID: logAnalyticsWorkspaceID,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
		}
}
