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
	"github.com/stretchr/testify/require"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
	. "github.com/kedacore/keda/v2/tests"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "azure-queue-test"
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
`
)

func TestSetup(t *testing.T) {
	require.NotEmpty(t, connectionString, "AZURE_STORAGE_CONNECTION_STRING env variable is required for service bus tests")

	// Create Queue
	httpClient := kedautil.CreateHTTPClient(DefaultHTTPTimeOut, false)
	credential, endpoint, err := azure.ParseAzureStorageQueueConnection(
		context.Background(), httpClient, kedav1alpha1.PodIdentityProviderNone, connectionString, "", "")
	require.NoErrorf(t, err, "cannot parse storage connection string - %s", err)

	p := azqueue.NewPipeline(credential, azqueue.PipelineOptions{})
	serviceURL := azqueue.NewServiceURL(*endpoint, p)
	queueURL := serviceURL.NewQueueURL(queueName)

	_, err = queueURL.Create(context.Background(), azqueue.Metadata{})
	require.NoErrorf(t, err, "cannot create storage queue - %s", err)

	// Create kubernetes resources
	Kc = GetKubernetesClient(t)
	CreateNamespace(t, Kc, testNamespace)

	base64ConnectionString := base64.StdEncoding.EncodeToString([]byte(connectionString))

	data := templateData{
		TestNamespace:    testNamespace,
		SecretName:       secretName,
		Connection:       base64ConnectionString,
		DeploymentName:   deploymentName,
		ScaledObjectName: scaledObjectName,
		QueueName:        queueName,
	}

	KubectlApplyMultipleWithTemplate(t, data, secretTemplate, deploymentTemplate, scaledObjectTemplate)

	require.True(t, WaitForDeploymentReplicaCount(t, Kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after a minute")
}

func TestScaleUp(t *testing.T) {
	// Create Queue
	httpClient := kedautil.CreateHTTPClient(DefaultHTTPTimeOut, false)
	credential, endpoint, err := azure.ParseAzureStorageQueueConnection(
		context.Background(), httpClient, kedav1alpha1.PodIdentityProviderNone, connectionString, "", "")
	require.NoErrorf(t, err, "cannot parse storage connection string - %s", err)

	p := azqueue.NewPipeline(credential, azqueue.PipelineOptions{})
	serviceURL := azqueue.NewServiceURL(*endpoint, p)
	queueURL := serviceURL.NewQueueURL(queueName)

	messageURL := queueURL.NewMessagesURL()

	for i := 0; i < 5; i++ {
		go func(t *testing.T, idx int) {
			for j := 0; j < 200; j++ {
				msg := fmt.Sprintf("Routine %d - Message - %d", idx, j)
				_, err := messageURL.Enqueue(context.Background(), msg, 0*time.Second, time.Hour)
				require.NoErrorf(t, err, "cannot enqueue message - %s", err)
			}
		}(t, i)
	}

	require.True(t, WaitForDeploymentReplicaCount(t, Kc, deploymentName, testNamespace, 1, 60, 1),
		"replica count should be 0 after a minute")
}

func TestScaleDown(t *testing.T) {
	// Create Queue
	httpClient := kedautil.CreateHTTPClient(DefaultHTTPTimeOut, false)
	credential, endpoint, err := azure.ParseAzureStorageQueueConnection(
		context.Background(), httpClient, kedav1alpha1.PodIdentityProviderNone, connectionString, "", "")
	require.NoErrorf(t, err, "cannot parse storage connection string - %s", err)

	p := azqueue.NewPipeline(credential, azqueue.PipelineOptions{})
	serviceURL := azqueue.NewServiceURL(*endpoint, p)
	queueURL := serviceURL.NewQueueURL(queueName)

	messageURL := queueURL.NewMessagesURL()

	// Clear queue
	// I am not sure if this is the best way to go about this. If someone can find a better way
	// please raise a PR.
	for {
		props, err := queueURL.GetProperties(context.Background())
		require.NoErrorf(t, err, "cannot fetch queue properties - %s", err)

		if props.ApproximateMessagesCount() > 0 {
			_, err := messageURL.Clear(context.Background())
			require.NoErrorf(t, err, "cannot clear storage queue - %s", err)
		} else {
			break
		}
	}

	require.True(t, WaitForDeploymentReplicaCount(t, Kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after a minute")
}

func TestCleanup(t *testing.T) {
	base64ConnectionString := base64.StdEncoding.EncodeToString([]byte(connectionString))

	data := templateData{
		TestNamespace:    testNamespace,
		SecretName:       secretName,
		Connection:       base64ConnectionString,
		DeploymentName:   deploymentName,
		ScaledObjectName: scaledObjectName,
		QueueName:        queueName,
	}

	// Delete kubernetes resources
	KubectlDeleteMultipleWithTemplate(t, data, secretTemplate, deploymentTemplate, scaledObjectTemplate)

	Kc = GetKubernetesClient(t)
	DeleteNamespace(t, Kc, testNamespace)

	// Create Queue
	httpClient := kedautil.CreateHTTPClient(DefaultHTTPTimeOut, false)
	credential, endpoint, err := azure.ParseAzureStorageQueueConnection(
		context.Background(), httpClient, kedav1alpha1.PodIdentityProviderNone, connectionString, "", "")
	require.NoErrorf(t, err, "cannot parse storage connection string - %s", err)

	p := azqueue.NewPipeline(credential, azqueue.PipelineOptions{})
	serviceURL := azqueue.NewServiceURL(*endpoint, p)
	queueURL := serviceURL.NewQueueURL(queueName)

	_, err = queueURL.Delete(context.Background())
	require.NoErrorf(t, err, "cannot create storage queue - %s", err)
}
