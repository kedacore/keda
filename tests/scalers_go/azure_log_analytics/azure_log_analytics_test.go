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
var _ = godotenv.Load("../../.env")

const (
	testName = "azure-log-analytics-test"
)

var (
	logAnalyticsWorkspaceID = os.Getenv("AZURE_LOG_ANALYTICS_WORKSPACE_ID")
	azureADClientID         = os.Getenv("AZURE_SP_APP_ID")
	azureADSecret           = os.Getenv("AZURE_SP_KEY")
	azureADTenantID         = os.Getenv("AZURE_SP_TENANT")
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

type templateValues map[string]string

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
          image: nginx
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
      authenticationRef:
        name: {{.TriggerAuthName}}
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	require.NotEmpty(t, logAnalyticsWorkspaceID, "AZURE_LOG_ANALYTICS_WORKSPACE_ID env variable is required for deployment bus tests")
	require.NotEmpty(t, azureADClientID, "AZURE_SP_APP_ID env variable is required for deployment bus tests")
	require.NotEmpty(t, azureADSecret, "AZURE_SP_KEY env variable is required for deployment bus tests")
	require.NotEmpty(t, azureADTenantID, "AZURE_SP_TENANT env variable is required for deployment bus tests")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testScaleUp(t, kc, data)
	testScaleDown(t, kc, data)

	// cleanup
	templates["triggerAuthTemplate"] = triggerAuthTemplate
	templates["scaledObjectTemplate"] = scaledObjectTemplate
	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
}

func testScaleUp(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale up ---")
	data.QueryX = 10
	data.QueryY = 1

	KubectlApplyWithTemplate(t, data, "triggerAuthTemplate", triggerAuthTemplate)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 1),
		"replica count should be 2 after 1 minute")
}

func testScaleDown(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale down ---")
	data.QueryX = 0

	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

func getTemplateData() (templateData, templateValues) {
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
		}, templateValues{
			"secretTemplate":     secretTemplate,
			"deploymentTemplate": deploymentTemplate}
}
