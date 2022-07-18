//go:build e2e
// +build e2e

package azure_data_explorer_test

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
	testName = "azure-data-explorer-test"
)

var (
	dataExplorerDB       = os.Getenv("AZURE_DATA_EXPLORER_DB")
	dataExplorerEndpoint = os.Getenv("AZURE_DATA_EXPLORER_ENDPOINT")
	azureADClientID      = os.Getenv("AZURE_SP_APP_ID")
	azureADSecret        = os.Getenv("AZURE_SP_KEY")
	azureADTenantID      = os.Getenv("AZURE_SP_TENANT")
	testNamespace        = fmt.Sprintf("%s-ns", testName)
	secretName           = fmt.Sprintf("%s-secret", testName)
	deploymentName       = fmt.Sprintf("%s-deployment", testName)
	triggerAuthName      = fmt.Sprintf("%s-ta", testName)
	scaledObjectName     = fmt.Sprintf("%s-so", testName)
	scaleInReplicaCount  = 0
	scaleInMetricValue   = 0
	scaleOutReplicaCount = 4
	scaleOutMetricValue  = 18
)

type templateData struct {
	TestNamespace        string
	SecretName           string
	DeploymentName       string
	TriggerAuthName      string
	ScaledObjectName     string
	AzureADClientID      string
	AzureADSecret        string
	AzureADTenantID      string
	DataExplorerDB       string
	DataExplorerEndpoint string
	ScaleReplicaCount    int
	ScaleMetricValue     int
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
  replicas: {{.ScaleReplicaCount}}
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
  cooldownPeriod: 10
  minReplicaCount: 0
  maxReplicaCount: 10
  pollingInterval: 30
  triggers:
    - type: azure-data-explorer
      metadata:
        databaseName: {{.DataExplorerDB}}
        endpoint: {{.DataExplorerEndpoint}}
        clientId: {{.AzureADClientID}}
        tenantId: {{.AzureADTenantID}}
        query: print result = {{.ScaleMetricValue}}
        threshold: "5"
      authenticationRef:
        name: {{.TriggerAuthName}}
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	require.NotEmpty(t, dataExplorerDB, "AZURE_DATA_EXPLORER_DB env variable is required for deployment bus tests")
	require.NotEmpty(t, dataExplorerEndpoint, "AZURE_DATA_EXPLORER_ENDPOINT env variable is required for deployment bus tests")
	require.NotEmpty(t, azureADClientID, "AZURE_SP_APP_ID env variable is required for deployment bus tests")
	require.NotEmpty(t, azureADSecret, "AZURE_SP_KEY env variable is required for deployment bus tests")
	require.NotEmpty(t, azureADTenantID, "AZURE_SP_TENANT env variable is required for deployment bus tests")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, scaleInReplicaCount, 60, 1),
		"replica count should be %d after a minute", scaleInReplicaCount)

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
	data.ScaleMetricValue = scaleOutMetricValue

	KubectlApplyWithTemplate(t, data, "triggerAuthTemplate", triggerAuthTemplate)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, scaleOutReplicaCount, 60, 1),
		"replica count should be %d after a minute", scaleOutReplicaCount)
}

func testScaleDown(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale down ---")
	data.ScaleMetricValue = scaleInMetricValue

	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, scaleInReplicaCount, 60, 1),
		"replica count should be %d after a minute", scaleInReplicaCount)
}

func getTemplateData() (templateData, templateValues) {
	base64ClientSecret := base64.StdEncoding.EncodeToString([]byte(azureADSecret))

	return templateData{
			TestNamespace:        testNamespace,
			SecretName:           secretName,
			DeploymentName:       deploymentName,
			TriggerAuthName:      triggerAuthName,
			ScaledObjectName:     scaledObjectName,
			AzureADClientID:      azureADClientID,
			AzureADSecret:        base64ClientSecret,
			AzureADTenantID:      azureADTenantID,
			DataExplorerDB:       dataExplorerDB,
			DataExplorerEndpoint: dataExplorerEndpoint,
			ScaleReplicaCount:    scaleInReplicaCount,
		}, templateValues{
			"secretTemplate":     secretTemplate,
			"deploymentTemplate": deploymentTemplate}
}
