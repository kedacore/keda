//go:build e2e
// +build e2e

package azure_queue_aad_wi_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName = "azure-queue-aad-wi-test"
)

var (
	connectionString = os.Getenv("TF_AZURE_STORAGE_CONNECTION_STRING")
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	triggerAuthName  = fmt.Sprintf("%s-ta", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	accountName      = ""
	queueName        = fmt.Sprintf("queue-%d", GetRandomNumber())
)

type templateData struct {
	TestNamespace    string
	SecretName       string
	Connection       string
	DeploymentName   string
	TriggerAuthName  string
	ScaledObjectName string
	AccountName      string
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
          env:
            - name: FUNCTIONS_WORKER_RUNTIME
              value: node
            - name: AzureWebJobsStorage
              valueFrom:
                secretKeyRef:
                  name: {{.SecretName}}
                  key: AzureWebJobsStorage
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
        activationQueueLength: "5"
        accountName: {{.AccountName}}
      authenticationRef:
        name: {{.TriggerAuthName}}
`
)

func TestScaler(t *testing.T) {
	// setup
	ctx := context.Background()
	t.Log("--- setting up ---")
	require.NotEmpty(t, connectionString, "TF_AZURE_STORAGE_CONNECTION_STRING env variable is required for azure queue test")

	queueClient, err := azqueue.NewQueueClientFromConnectionString(connectionString, queueName, nil)
	assert.NoErrorf(t, err, "cannot create the queue client - %s", err)
	_, err = queueClient.Create(ctx, nil)
	assert.NoErrorf(t, err, "cannot create the queue - %s", err)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testActivation(ctx, t, kc, queueClient)
	testScaleOut(ctx, t, kc, queueClient)
	testScaleIn(ctx, t, kc, queueClient)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
	_, err = queueClient.Delete(ctx, nil)
	assert.NoErrorf(t, err, "cannot create the queue - %s", err)
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
func testActivation(ctx context.Context, t *testing.T, kc *kubernetes.Clientset, client *azqueue.QueueClient) {
	t.Log("--- testing activation ---")
	addMessages(ctx, t, client, 3)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)
}

func testScaleOut(ctx context.Context, t *testing.T, kc *kubernetes.Clientset, client *azqueue.QueueClient) {
	t.Log("--- testing scale out ---")
	addMessages(ctx, t, client, 5)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 1),
		"replica count should be 1 after 1 minute")
}

func testScaleIn(ctx context.Context, t *testing.T, kc *kubernetes.Clientset, client *azqueue.QueueClient) {
	t.Log("--- testing scale in ---")
	_, err := client.ClearMessages(ctx, nil)
	assert.NoErrorf(t, err, "cannot clear queue - %s", err)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

func addMessages(ctx context.Context, t *testing.T, client *azqueue.QueueClient, count int) {
	for i := 0; i < count; i++ {
		msg := fmt.Sprintf("Message - %d", i)
		_, err := client.EnqueueMessage(ctx, msg, nil)
		assert.NoErrorf(t, err, "cannot enqueue message - %s", err)
		t.Logf("Message queued")
	}
}
