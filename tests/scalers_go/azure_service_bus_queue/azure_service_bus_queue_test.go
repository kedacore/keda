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
	"github.com/stretchr/testify/require"

	. "github.com/kedacore/keda/v2/tests"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "azure-service-bus-queue-test"
)

var (
	connectionString = os.Getenv("AZURE_SERVICE_BUS_CONNECTION_STRING")
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	triggerAuthName  = fmt.Sprintf("%s-ta", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	queueName        = fmt.Sprintf("%s-queue", testName)
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
    authenticationRef:
      name: {{.TriggerAuthName}}
`
)

func TestSetup(t *testing.T) {
	require.NotEmpty(t, connectionString, "AZURE_SERVICE_BUS_CONNECTION_STRING env variable is required for service bus tests")

	// Connect to service bus namespace.
	sbNamespace, err := servicebus.NewNamespace(servicebus.NamespaceWithConnectionString(connectionString))
	require.NoErrorf(t, err, "cannot connect to service bus namespace - %s", err)

	sbQueueManager := sbNamespace.NewQueueManager()
	sbQueues, err := sbQueueManager.List(context.Background())
	require.NoErrorf(t, err, "cannot fetch queue list for service bus namespace - %s", err)

	// Delete service bus queue if already exists.
	for _, queue := range sbQueues {
		if queue.Name == queueName {
			t.Log("Service Bus Queue already exists. Deleting.")
			err := sbQueueManager.Delete(context.Background(), queueName)
			require.NoErrorf(t, err, "cannot delete existing service bus queue - %s", err)
		}
	}

	// Create service bus queue.
	_, err = sbQueueManager.Put(context.Background(), queueName)
	require.NoErrorf(t, err, "cannot create service bus queue - %s", err)

	Kc = GetKubernetesClient(t)
	CreateNamespace(t, Kc, testNamespace)

	base64ConnectionString := base64.StdEncoding.EncodeToString([]byte(connectionString))

	data := templateData{
		TestNamespace:    testNamespace,
		SecretName:       secretName,
		Connection:       base64ConnectionString,
		DeploymentName:   deploymentName,
		TriggerAuthName:  triggerAuthName,
		ScaledObjectName: scaledObjectName,
		QueueName:        queueName,
	}

	// Create kubernetes resources
	KubectlApplyMultipleWithTemplate(t, data, secretTemplate, deploymentTemplate, triggerAuthTemplate, scaledObjectTemplate)

	require.True(t, WaitForDeploymentReplicaCount(t, Kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after a minute")
}

func TestScaleUp(t *testing.T) {
	sbNamespace, err := servicebus.NewNamespace(servicebus.NamespaceWithConnectionString(connectionString))
	require.NoErrorf(t, err, "cannot connect to service bus namespace - %s", err)

	sbQueue, err := sbNamespace.NewQueue(queueName)
	require.NoErrorf(t, err, "cannot create client for queue - %s", err)

	for i := 0; i < 5; i++ {
		_ = sbQueue.Send(context.Background(), servicebus.NewMessageFromString(fmt.Sprintf("Message - %d", i)))
	}

	require.True(t, WaitForDeploymentReplicaCount(t, Kc, deploymentName, testNamespace, 1, 60, 1),
		"replica count should be 1 after 1 minute")
}

func TestScaleDown(t *testing.T) {
	var messageHandlerFunc servicebus.HandlerFunc = func(ctx context.Context, msg *servicebus.Message) error {
		return msg.Complete(ctx)
	}

	sbNamespace, err := servicebus.NewNamespace(servicebus.NamespaceWithConnectionString(connectionString))
	require.NoErrorf(t, err, "cannot connect to service bus namespace - %s", err)

	sbQueue, err := sbNamespace.NewQueue(queueName)
	require.NoErrorf(t, err, "cannot create client for queue - %s", err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = sbQueue.Receive(ctx, messageHandlerFunc)

	require.True(t, WaitForDeploymentReplicaCount(t, Kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

func TestCleanup(t *testing.T) {
	base64ConnectionString := base64.StdEncoding.EncodeToString([]byte(connectionString))

	data := templateData{
		TestNamespace:    testNamespace,
		SecretName:       secretName,
		Connection:       base64ConnectionString,
		DeploymentName:   deploymentName,
		TriggerAuthName:  triggerAuthName,
		ScaledObjectName: scaledObjectName,
		QueueName:        queueName,
	}

	// Delete kubernetes resources
	KubectlDeleteMultipleWithTemplate(t, data, secretTemplate, deploymentTemplate, triggerAuthTemplate, scaledObjectTemplate)

	Kc = GetKubernetesClient(t)
	DeleteNamespace(t, Kc, testNamespace)

	// Delete service bus queue.
	sbNamespace, err := servicebus.NewNamespace(servicebus.NamespaceWithConnectionString(connectionString))
	require.NoErrorf(t, err, "cannot connect to service bus namespace - %s", err)

	sbQueueManager := sbNamespace.NewQueueManager()
	err = sbQueueManager.Delete(context.Background(), queueName)
	require.NoErrorf(t, err, "cannot delete existing service bus queue - %s", err)
}
