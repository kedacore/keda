//go:build e2e
// +build e2e

package azure_data_explorer_aad_wi_test

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
	testName = "azure-data-explorer-add-wi-test"
)

var (
	dataExplorerDB        = os.Getenv("TF_AZURE_DATA_EXPLORER_DB")
	dataExplorerEndpoint  = os.Getenv("TF_AZURE_DATA_EXPLORER_ENDPOINT")
	testNamespace         = fmt.Sprintf("%s-ns", testName)
	deploymentName        = fmt.Sprintf("%s-deployment", testName)
	triggerAuthName       = fmt.Sprintf("%s-ta", testName)
	scaledObjectName      = fmt.Sprintf("%s-so", testName)
	scaleInReplicaCount   = 0
	scaleInMetricValue    = 0
	scaleOutReplicaCount  = 4
	scaleOutMetricValue   = 30
	activationMetricValue = 3
)

type templateData struct {
	TestNamespace        string
	DeploymentName       string
	TriggerAuthName      string
	ScaledObjectName     string
	DataExplorerDB       string
	DataExplorerEndpoint string
	ScaleReplicaCount    int
	ScaleMetricValue     int
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
  cooldownPeriod: 10
  minReplicaCount: 0
  maxReplicaCount: 4
  pollingInterval: 30
  triggers:
    - type: azure-data-explorer
      metadata:
        databaseName: {{.DataExplorerDB}}
        endpoint: {{.DataExplorerEndpoint}}
        query: print result = {{.ScaleMetricValue}}
        threshold: "5"
        activationThreshold: "3"
      authenticationRef:
        name: {{.TriggerAuthName}}
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	require.NotEmpty(t, dataExplorerDB, "TF_AZURE_DATA_EXPLORER_DB env variable is required for deployment bus tests")
	require.NotEmpty(t, dataExplorerEndpoint, "TF_AZURE_DATA_EXPLORER_ENDPOINT env variable is required for deployment bus tests")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, scaleInReplicaCount, 60, 1),
		"replica count should be %d after 1 minute", scaleInReplicaCount)

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
	data.ScaleMetricValue = activationMetricValue

	KubectlApplyWithTemplate(t, data, "triggerAuthTemplate", triggerAuthTemplate)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, scaleInReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	data.ScaleMetricValue = scaleOutMetricValue

	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, scaleOutReplicaCount, 60, 1),
		"replica count should be %d after 1 minute", scaleOutReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in ---")
	data.ScaleMetricValue = scaleInMetricValue

	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, scaleInReplicaCount, 60, 1),
		"replica count should be %d after 1 minute", scaleInReplicaCount)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:        testNamespace,
			DeploymentName:       deploymentName,
			TriggerAuthName:      triggerAuthName,
			ScaledObjectName:     scaledObjectName,
			DataExplorerDB:       dataExplorerDB,
			DataExplorerEndpoint: dataExplorerEndpoint,
			ScaleReplicaCount:    scaleInReplicaCount,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
		}
}
