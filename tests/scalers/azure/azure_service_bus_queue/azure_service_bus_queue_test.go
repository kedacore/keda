//go:build e2e
// +build e2e

package azure_service_bus_queue_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/admin"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName = "azure-service-bus-queue-test"
)

var (
	connectionString = os.Getenv("TF_AZURE_SERVICE_BUS_CONNECTION_STRING")
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	triggerAuthName  = fmt.Sprintf("%s-ta", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	queueName        = fmt.Sprintf("queue-%d", GetRandomNumber())
)

type templateData struct {
	TestNamespace    string
	SecretName       string
	Connection       string
	DeploymentName   string
	TriggerAuthName  string
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
type: Opaque
data:
  connection: {{.Connection}}
`

	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
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
        - name: nginx
          image: ghcr.io/nginx/nginx-unprivileged:1.26
`

	triggerAuthTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: connection
      name: {{.SecretName}}
      key: connection
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
  pollingInterval: 5
  cooldownPeriod: 10
  minReplicaCount: 0
  maxReplicaCount: 1
  triggers:
  - type: azure-servicebus
    metadata:
      queueName: {{.QueueName}}
      activationMessageCount: "5"
    authenticationRef:
      name: {{.TriggerAuthName}}
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	require.NotEmpty(t, connectionString, "TF_AZURE_SERVICE_BUS_CONNECTION_STRING env variable is required for service bus tests")

	client, adminClient := setupServiceBusQueue(t)

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testActivation(t, kc, client)
	testScaleOut(t, kc, client)
	testScaleIn(t, kc, adminClient)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
	cleanupServiceBusQueue(t, adminClient)
}

func setupServiceBusQueue(t *testing.T) (*azservicebus.Client, *admin.Client) {
	adminClient, err := admin.NewClientFromConnectionString(connectionString, nil)
	assert.NoErrorf(t, err, "cannot connect to service bus namespace - %s", err)

	// Delete the queue if already exists
	_, _ = adminClient.DeleteQueue(context.Background(), queueName, nil)

	_, err = adminClient.CreateQueue(context.Background(), queueName, nil)
	assert.NoErrorf(t, err, "cannot create the queue - %s", err)

	client, err := azservicebus.NewClientFromConnectionString(connectionString, nil)
	assert.NoErrorf(t, err, "cannot connect to service bus namespace - %s", err)

	return client, adminClient
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
			QueueName:        queueName,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, client *azservicebus.Client) {
	t.Log("--- testing activation ---")
	addMessages(t, client, 3)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, client *azservicebus.Client) {
	t.Log("--- testing scale out ---")
	addMessages(t, client, 10)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 1),
		"replica count should be 1 after 1 minute")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, adminClient *admin.Client) {
	t.Log("--- testing scale in ---")

	_, err := adminClient.DeleteQueue(context.Background(), queueName, nil)
	assert.NoErrorf(t, err, "cannot delete the queue - %s", err)
	_, err = adminClient.CreateQueue(context.Background(), queueName, nil)
	assert.NoErrorf(t, err, "cannot create the queue - %s", err)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

func addMessages(t *testing.T, client *azservicebus.Client, count int) {
	sender, err := client.NewSender(queueName, nil)
	assert.NoErrorf(t, err, "cannot create the sender - %s", err)
	for i := 0; i < count; i++ {
		msg := fmt.Sprintf("Message - %d", i)
		_ = sender.SendMessage(context.Background(), &azservicebus.Message{
			Body: []byte(msg),
		}, nil)
	}
}

func cleanupServiceBusQueue(t *testing.T, adminClient *admin.Client) {
	t.Log("--- cleaning up ---")
	_, err := adminClient.DeleteQueue(context.Background(), queueName, nil)
	assert.NoErrorf(t, err, "cannot delete service bus queue - %s", err)
}
