//go:build e2e
// +build e2e

package azure_blob_aad_wi_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName = "azure-blob-aad-wi-test"
)

var (
	connectionString = os.Getenv("TF_AZURE_STORAGE_CONNECTION_STRING")
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	triggerAuthName  = fmt.Sprintf("%s-ta", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	containerName    = fmt.Sprintf("container-%d", GetRandomNumber())
	accountName      = ""
)

type templateData struct {
	TestNamespace    string
	SecretName       string
	Connection       string
	DeploymentName   string
	TriggerAuthName  string
	ScaledObjectName string
	ContainerName    string
	AccountName      string
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
      nodeSelector:
        kubernetes.io/os: linux
      containers:
        - name: {{.DeploymentName}}
          image: slurplk/blob-consumer:latest
          env:
            - name: FUNCTIONS_WORKER_RUNTIME
              value: dotnet
            - name: AzureFunctionsWebHost__hostid
              value: {{.DeploymentName}}
            - name: AzureWebJobsStorage
              valueFrom:
                secretKeyRef:
                  name: {{.SecretName}}
                  key: AzureWebJobsStorage
            - name: TEST_STORAGE_CONNECTION_STRING
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
  pollingInterval: 10
  minReplicaCount: 0
  maxReplicaCount: 2
  cooldownPeriod: 10
  triggers:
    - type: azure-blob
      metadata:
        blobContainerName: {{.ContainerName}}
        blobCount: '1'
        activationBlobCount: '5'
        accountName: {{.AccountName}}
      authenticationRef:
        name: {{.TriggerAuthName}}
`
)

func TestScaler(t *testing.T) {
	// setup
	ctx := context.Background()
	t.Log("--- setting up ---")
	require.NotEmpty(t, connectionString, "TF_AZURE_STORAGE_CONNECTION_STRING env variable is required for azure blob test")

	blobClient, err := azblob.NewClientFromConnectionString(connectionString, nil)
	assert.NoErrorf(t, err, "cannot create the queue client - %s", err)
	_, err = blobClient.CreateContainer(ctx, containerName, nil)
	assert.NoErrorf(t, err, "cannot create the container - %s", err)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testActivation(ctx, t, kc, blobClient)
	testScaleOut(ctx, t, kc, blobClient)
	testScaleIn(ctx, t, kc, blobClient)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
	_, err = blobClient.DeleteContainer(ctx, containerName, nil)
	assert.NoErrorf(t, err, "cannot delete the container - %s", err)
}

func getTemplateData() (templateData, []Template) {
	base64ConnectionString := base64.StdEncoding.EncodeToString([]byte(connectionString))

	return templateData{
			TestNamespace:    testNamespace,
			SecretName:       secretName,
			Connection:       base64ConnectionString,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
			ContainerName:    containerName,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testActivation(ctx context.Context, t *testing.T, kc *kubernetes.Clientset, blobClient *azblob.Client) {
	t.Log("--- testing activation ---")
	addFiles(ctx, t, blobClient, 4)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)
}

func testScaleOut(ctx context.Context, t *testing.T, kc *kubernetes.Clientset, blobClient *azblob.Client) {
	t.Log("--- testing scale out ---")
	addFiles(ctx, t, blobClient, 10)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 1),
		"replica count should be 2 after 1 minute")
}

func testScaleIn(ctx context.Context, t *testing.T, kc *kubernetes.Clientset, blobClient *azblob.Client) {
	t.Log("--- testing scale in ---")

	for i := 0; i < 10; i++ {
		blobName := fmt.Sprintf("blob-%d", i)
		_, err := blobClient.DeleteBlob(ctx, containerName, blobName, nil)
		assert.NoErrorf(t, err, "cannot delete blob - %s", err)
	}

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

func addFiles(ctx context.Context, t *testing.T, blobClient *azblob.Client, count int) {
	data := "Hello World!"

	for i := 0; i < count; i++ {
		blobName := fmt.Sprintf("blob-%d", i)
		_, err := blobClient.UploadStream(ctx, containerName, blobName, strings.NewReader(data), nil)
		assert.NoErrorf(t, err, "cannot upload blob - %s", err)
	}
}
