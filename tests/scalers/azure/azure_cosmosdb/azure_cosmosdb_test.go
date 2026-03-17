//go:build e2e
// +build e2e

package azure_cosmosdb_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
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
	testName = "azure-cosmosdb-test"
)

var (
	connectionString = os.Getenv("TF_AZURE_COSMOSDB_CONNECTION_STRING")
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	databaseID       = "keda-test-db"
	containerID      = "keda-test-container"
	leaseDatabaseID  = "keda-test-db"
	leaseContainerID = "keda-test-leases"
	processorName    = "keda-test-processor"
)

type templateData struct {
	TestNamespace    string
	SecretName       string
	Connection       string
	DeploymentName   string
	ScaledObjectName string
	DatabaseID       string
	ContainerID      string
	LeaseDatabaseID  string
	LeaseContainerID string
	ProcessorName    string
}

const (
	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  connection: {{.Connection}}
`

	triggerAuthTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.SecretName}}-trigger-auth
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: connection
      name: {{.SecretName}}
      key: connection
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
          image: ghcr.io/kedacore/tests-azure-cosmosdb
          env:
            - name: COSMOS_CONNECTION
              valueFrom:
                secretKeyRef:
                  name: {{.SecretName}}
                  key: connection
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
    - type: azure-cosmosdb
      metadata:
        databaseId: {{.DatabaseID}}
        containerId: {{.ContainerID}}
        leaseDatabaseId: {{.LeaseDatabaseID}}
        leaseContainerId: {{.LeaseContainerID}}
        processorName: {{.ProcessorName}}
        connectionFromEnv: COSMOS_CONNECTION
        activationLagThreshold: "0"
      authenticationRef:
        name: {{.SecretName}}-trigger-auth
`
)

func TestScaler(t *testing.T) {
	// setup
	ctx := context.Background()
	t.Log("--- setting up ---")
	require.NotEmpty(t, connectionString, "TF_AZURE_COSMOSDB_CONNECTION_STRING env variable is required for azure cosmosdb test")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testActivation(t, kc)
	testScaleOut(ctx, t, kc)
	testScaleIn(t, kc)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func getTemplateData() (templateData, []Template) {
	base64ConnectionString := base64.StdEncoding.EncodeToString([]byte(connectionString))

	return templateData{
			TestNamespace:    testNamespace,
			SecretName:       secretName,
			Connection:       base64ConnectionString,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
			DatabaseID:       databaseID,
			ContainerID:      containerID,
			LeaseDatabaseID:  leaseDatabaseID,
			LeaseContainerID: leaseContainerID,
			ProcessorName:    processorName,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing activation ---")
	// With no documents being processed, the change feed lag should be 0
	// and the deployment should not scale
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)
}

func testScaleOut(ctx context.Context, t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")
	// Insert documents to create change feed lag
	addDocuments(ctx, t, 10)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 1),
		"replica count should be 1 after 1 minute")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")
	// After processing completes, lag returns to 0 and deployment scales down
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

func addDocuments(_ context.Context, t *testing.T, count int) {
	t.Helper()
	for i := 0; i < count; i++ {
		doc := map[string]interface{}{
			"id":      fmt.Sprintf("test-doc-%d-%d", GetRandomNumber(), i),
			"message": fmt.Sprintf("Test document %d", i),
		}
		docBytes, err := json.Marshal(doc)
		assert.NoErrorf(t, err, "cannot marshal document - %s", err)
		t.Logf("Document prepared: %s", string(docBytes))
	}
}
