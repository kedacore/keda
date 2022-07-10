//go:build e2e
// +build e2e

package azure_queue_restore_original_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"github.com/Azure/azure-storage-queue-go/azqueue"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "azure-queue-restore-test"
)

var (
	connectionString = os.Getenv("AZURE_STORAGE_CONNECTION_STRING")
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	queueName        = fmt.Sprintf("%s-queue", testName)
)

type templateData struct {
	TestNamespace    string
	SecretName       string
	Connection       string
	DeploymentName   string
	ScaledObjectName string
	QueueName        string
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
  AzureWebJobsStorage: {{.Connection}}
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
  replicas: 2
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
          image: ghcr.io/kedacore/tests-azure-queue
          resources:
          env:
            - name: FUNCTIONS_WORKER_RUNTIME
              value: node
            - name: AzureWebJobsStorage
              valueFrom:
                secretKeyRef:
                  name: {{.SecretName}}
                  key: AzureWebJobsStorage
`

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  advanced:
    restoreToOriginalReplicaCount: true
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  minReplicaCount: 0
  maxReplicaCount: 4
  cooldownPeriod: 10
  triggers:
    - type: azure-queue
      metadata:
        queueName: {{.QueueName}}
        connectionFromEnv: AzureWebJobsStorage
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	require.NotEmpty(t, connectionString, "AZURE_STORAGE_CONNECTION_STRING env variable is required for service bus tests")

	queueURL := createQueue(t)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaCount(t, kc, deploymentName, testNamespace, 2, 60, 1),
		"replica count should be 2 after a minute")

	// test scaling
	testScale(t, kc, data)
	testRestore(t, kc, data)

	// cleanup
	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
	cleanupQueue(t, queueURL)
}

func createQueue(t *testing.T) azqueue.QueueURL {
	// Create Queue
	httpClient := kedautil.CreateHTTPClient(DefaultHTTPTimeOut, false)
	credential, endpoint, err := azure.ParseAzureStorageQueueConnection(
		context.Background(), httpClient, kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		connectionString, "", "")
	assert.NoErrorf(t, err, "cannot parse storage connection string - %s", err)

	p := azqueue.NewPipeline(credential, azqueue.PipelineOptions{})
	serviceURL := azqueue.NewServiceURL(*endpoint, p)
	queueURL := serviceURL.NewQueueURL(queueName)

	_, err = queueURL.Create(context.Background(), azqueue.Metadata{})
	assert.NoErrorf(t, err, "cannot create storage queue - %s", err)

	return queueURL
}

func getTemplateData() (templateData, templateValues) {
	base64ConnectionString := base64.StdEncoding.EncodeToString([]byte(connectionString))

	return templateData{
			TestNamespace:    testNamespace,
			SecretName:       secretName,
			Connection:       base64ConnectionString,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
			QueueName:        queueName,
		}, templateValues{
			"secretTemplate":     secretTemplate,
			"deploymentTemplate": deploymentTemplate}
}

func testScale(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scaling ---")
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	assert.True(t, WaitForDeploymentReplicaCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after a minute")
}

func testRestore(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing restore ---")
	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	assert.True(t, WaitForDeploymentReplicaCount(t, kc, deploymentName, testNamespace, 2, 60, 1),
		"replica count should be 2 after a minute")
}

func cleanupQueue(t *testing.T, queueURL azqueue.QueueURL) {
	t.Log("--- cleaning up ---")
	_, err := queueURL.Delete(context.Background())
	assert.NoErrorf(t, err, "cannot create storage queue - %s", err)
}
