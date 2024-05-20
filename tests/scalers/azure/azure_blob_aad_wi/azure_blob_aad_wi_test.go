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

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
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
	t.Log("--- setting up ---")
	require.NotEmpty(t, connectionString, "TF_AZURE_STORAGE_CONNECTION_STRING env variable is required for azure blob test")

	containerURL := createContainer(t)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testActivation(t, kc, containerURL)
	testScaleOut(t, kc, containerURL)
	testScaleIn(t, kc, containerURL)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
	cleanupContainer(t, containerURL)
}

func createContainer(t *testing.T) azblob.ContainerURL {
	// Create Blob Container
	credential, endpoint, err := azure.ParseAzureStorageBlobConnection(
		context.Background(), kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		connectionString, "", "")
	assert.NoErrorf(t, err, "cannot parse storage connection string - %s", err)

	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})
	serviceURL := azblob.NewServiceURL(*endpoint, p)
	containerURL := serviceURL.NewContainerURL(containerName)

	_, err = containerURL.Create(context.Background(), azblob.Metadata{}, azblob.PublicAccessContainer)
	assert.NoErrorf(t, err, "cannot create blob container - %s", err)

	domains := strings.Split(endpoint.Hostname(), ".")
	accountName = domains[0]
	return containerURL
}

func getTemplateData() (templateData, []Template) {
	base64ConnectionString := base64.StdEncoding.EncodeToString([]byte(connectionString))

	return templateData{
			TestNamespace:    testNamespace,
			SecretName:       secretName,
			Connection:       base64ConnectionString,
			DeploymentName:   deploymentName,
			TriggerAuthName:  triggerAuthName,
			ScaledObjectName: scaledObjectName,
			ContainerName:    containerName,
			AccountName:      accountName,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, containerURL azblob.ContainerURL) {
	t.Log("--- testing activation ---")
	addFiles(t, containerURL, 4)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, containerURL azblob.ContainerURL) {
	t.Log("--- testing scale out ---")
	addFiles(t, containerURL, 10)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 1),
		"replica count should be 2 after 1 minute")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, containerURL azblob.ContainerURL) {
	t.Log("--- testing scale in ---")

	for i := 0; i < 10; i++ {
		blobName := fmt.Sprintf("blob-%d", i)
		blobURL := containerURL.NewBlockBlobURL(blobName)

		_, err := blobURL.Delete(context.Background(), azblob.DeleteSnapshotsOptionInclude,
			azblob.BlobAccessConditions{})

		assert.NoErrorf(t, err, "cannot delete blob - %s", err)
	}

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

func addFiles(t *testing.T, containerURL azblob.ContainerURL, count int) {
	data := "Hello World!"

	for i := 0; i < count; i++ {
		blobName := fmt.Sprintf("blob-%d", i)
		blobURL := containerURL.NewBlockBlobURL(blobName)

		_, err := blobURL.Upload(context.Background(), strings.NewReader(data),
			azblob.BlobHTTPHeaders{ContentType: "text/plain"}, azblob.Metadata{}, azblob.BlobAccessConditions{},
			azblob.DefaultAccessTier, nil, azblob.ClientProvidedKeyOptions{}, azblob.ImmutabilityPolicyOptions{})

		assert.NoErrorf(t, err, "cannot upload blob - %s", err)
	}
}

func cleanupContainer(t *testing.T, containerURL azblob.ContainerURL) {
	t.Log("--- cleaning up ---")
	_, err := containerURL.Delete(context.Background(), azblob.ContainerAccessConditions{})
	assert.NoErrorf(t, err, "cannot delete storage container - %s", err)
}
