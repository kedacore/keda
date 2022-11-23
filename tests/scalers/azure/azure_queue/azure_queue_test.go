//go:build e2e
// +build e2e

package azure_queue_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"
	"time"

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
var _ = godotenv.Load("../../../.env")

const (
	testName = "azure-queue-test"
)

var (
	connectionString = os.Getenv("TF_AZURE_STORAGE_CONNECTION_STRING")
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	queueName        = fmt.Sprintf("queue-%d", GetRandomNumber())
)

type templateData struct {
	TestNamespace    string
	SecretName       string
	Connection       string
	DeploymentName   string
	ScaledObjectName string
	QueueName        string
}

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
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  minReplicaCount: 0
  maxReplicaCount: 1
  cooldownPeriod: 10
  triggers:
    - type: azure-queue
      metadata:
        queueName: {{.QueueName}}
        connectionFromEnv: AzureWebJobsStorage
        activationQueueLength: "5"
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	require.NotEmpty(t, connectionString, "TF_AZURE_STORAGE_CONNECTION_STRING env variable is required for azure queue test")

	queueURL, messageURL := createQueue(t)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testActivation(t, kc, messageURL)
	testScaleOut(t, kc, messageURL)
	testScaleIn(t, kc, messageURL)

	// cleanup
	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
	cleanupQueue(t, queueURL)
}

func createQueue(t *testing.T) (azqueue.QueueURL, azqueue.MessagesURL) {
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

	messageURL := queueURL.NewMessagesURL()
	t.Logf("Queue %s created", queueName)
	return queueURL, messageURL
}

func getTemplateData() (templateData, []Template) {
	base64ConnectionString := base64.StdEncoding.EncodeToString([]byte(connectionString))

	return templateData{
			TestNamespace:    testNamespace,
			SecretName:       secretName,
			Connection:       base64ConnectionString,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
			QueueName:        queueName,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, messageURL azqueue.MessagesURL) {
	t.Log("--- testing activation ---")
	addMessages(t, messageURL, 3)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, messageURL azqueue.MessagesURL) {
	t.Log("--- testing scale out ---")
	addMessages(t, messageURL, 5)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 1),
		"replica count should be 1 after 1 minute")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, messageURL azqueue.MessagesURL) {
	t.Log("--- testing scale in ---")
	_, err := messageURL.Clear(context.Background())
	assert.NoErrorf(t, err, "cannot clear queue - %s", err)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

func addMessages(t *testing.T, messageURL azqueue.MessagesURL, count int) {
	for i := 0; i < count; i++ {
		msg := fmt.Sprintf("Message - %d", i)
		_, err := messageURL.Enqueue(context.Background(), msg, 0*time.Second, time.Hour)
		assert.NoErrorf(t, err, "cannot enqueue message - %s", err)
		t.Logf("Message queued")
	}
}

func cleanupQueue(t *testing.T, queueURL azqueue.QueueURL) {
	t.Log("--- cleaning up ---")
	_, err := queueURL.Delete(context.Background())
	assert.NoErrorf(t, err, "cannot delete storage queue - %s", err)
	t.Logf("Queue %s deleted", queueName)
}
