//go:build e2e
// +build e2e

package azure_service_bus_queue_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"
	"time"

	servicebus "github.com/Azure/azure-service-bus-go"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

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
          image: nginx:1.16.1
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

	sbQueueManager, sbQueue := setupServiceBusQueue(t)

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testActivation(t, kc, sbQueue)
	testScaleUp(t, kc, sbQueue)
	testScaleDown(t, kc, sbQueue)

	// cleanup
	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
	cleanupServiceBusQueue(t, sbQueueManager)
}

func setupServiceBusQueue(t *testing.T) (*servicebus.QueueManager, *servicebus.Queue) {
	// Connect to service bus namespace.
	sbNamespace, err := servicebus.NewNamespace(servicebus.NamespaceWithConnectionString(connectionString))
	assert.NoErrorf(t, err, "cannot connect to service bus namespace - %s", err)

	sbQueueManager := sbNamespace.NewQueueManager()

	createQueue(t, sbQueueManager)

	sbQueue, err := sbNamespace.NewQueue(queueName)
	assert.NoErrorf(t, err, "cannot create client for queue - %s", err)

	return sbQueueManager, sbQueue
}

func createQueue(t *testing.T, sbQueueManager *servicebus.QueueManager) {
	// delete queue if already exists
	sbQueues, err := sbQueueManager.List(context.Background())
	assert.NoErrorf(t, err, "cannot fetch queue list for service bus namespace - %s", err)

	for _, queue := range sbQueues {
		if queue.Name == queueName {
			t.Log("Service Bus Queue already exists. Deleting.")
			err := sbQueueManager.Delete(context.Background(), queueName)
			assert.NoErrorf(t, err, "cannot delete existing service bus queue - %s", err)
		}
	}

	// create queue
	_, err = sbQueueManager.Put(context.Background(), queueName)
	assert.NoErrorf(t, err, "cannot create service bus queue - %s", err)
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

func testActivation(t *testing.T, kc *kubernetes.Clientset, sbQueue *servicebus.Queue) {
	t.Log("--- testing activation ---")
	addMessages(sbQueue, 3)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)
}

func testScaleUp(t *testing.T, kc *kubernetes.Clientset, sbQueue *servicebus.Queue) {
	t.Log("--- testing scale up ---")
	addMessages(sbQueue, 10)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 1),
		"replica count should be 1 after 1 minute")
}

func testScaleDown(t *testing.T, kc *kubernetes.Clientset, sbQueue *servicebus.Queue) {
	t.Log("--- testing scale down ---")
	var messageHandlerFunc servicebus.HandlerFunc = func(ctx context.Context, msg *servicebus.Message) error {
		return msg.Complete(ctx)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = sbQueue.Receive(ctx, messageHandlerFunc)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

func addMessages(sbQueue *servicebus.Queue, count int) {
	for i := 0; i < count; i++ {
		msg := fmt.Sprintf("Message - %d", i)
		_ = sbQueue.Send(context.Background(), servicebus.NewMessageFromString(msg))
	}
}

func cleanupServiceBusQueue(t *testing.T, sbQueueManager *servicebus.QueueManager) {
	t.Log("--- cleaning up ---")
	err := sbQueueManager.Delete(context.Background(), queueName)
	assert.NoErrorf(t, err, "cannot delete service bus queue - %s", err)
}
