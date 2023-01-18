//go:build e2e
// +build e2e

package azure_workload_identity_user_assigned_test

import (
	"context"
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
	testName = "azure-workload-identity-user-assigned-test"
)

var (
	connectionString = os.Getenv("TF_AZURE_SERVICE_BUS_ALTERNATIVE_CONNECTION_STRING")
	azureADClientID  = os.Getenv("TF_AZURE_IDENTITY_2_APP_ID")
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	triggerAuthName  = fmt.Sprintf("%s-ta", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	queueName        = fmt.Sprintf("%s-queue", testName)
)

type templateData struct {
	TestNamespace       string
	DeploymentName      string
	TriggerAuthName     string
	IdentityID          string
	ScaledObjectName    string
	ServiceBusNamespace string
	QueueName           string
}

const (
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
  podIdentity:
    provider: azure-workload
`

	triggerAuthTemplateWithIdentityID = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: azure-workload
    identityId: {{.IdentityID}}
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
      namespace: {{.ServiceBusNamespace}}
      queueName: {{.QueueName}}
    authenticationRef:
      name: {{.TriggerAuthName}}
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	require.NotEmpty(t, connectionString, "TF_AZURE_SERVICE_BUS_ALTERNATIVE_CONNECTION_STRING env variable is required")
	require.NotEmpty(t, azureADClientID, "TF_AZURE_IDENTITY_2_APP_ID env variable is required for service bus tests")

	sbNamespace, sbQueueManager, sbQueue := setupServiceBusQueue(t)

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	data.ServiceBusNamespace = sbNamespace.Name

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testScaleUpWithIncorrectIdentity(t, kc, sbQueue)
	testScaleUpWithCorrectIdentity(t, kc, data)
	testScaleDown(t, kc, sbQueue)

	// cleanup
	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
	cleanupServiceBusQueue(t, sbQueueManager)
}

func setupServiceBusQueue(t *testing.T) (*servicebus.Namespace, *servicebus.QueueManager, *servicebus.Queue) {
	// Connect to service bus namespace.
	sbNamespace, err := servicebus.NewNamespace(servicebus.NamespaceWithConnectionString(connectionString))
	assert.NoErrorf(t, err, "cannot connect to service bus namespace - %s", err)

	sbQueueManager := sbNamespace.NewQueueManager()

	createQueue(t, sbQueueManager)

	sbQueue, err := sbNamespace.NewQueue(queueName)
	assert.NoErrorf(t, err, "cannot create client for queue - %s", err)

	return sbNamespace, sbQueueManager, sbQueue
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
	return templateData{
			TestNamespace:    testNamespace,
			DeploymentName:   deploymentName,
			TriggerAuthName:  triggerAuthName,
			IdentityID:       azureADClientID,
			ScaledObjectName: scaledObjectName,
			QueueName:        queueName,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testScaleUpWithIncorrectIdentity(t *testing.T, kc *kubernetes.Clientset, sbQueue *servicebus.Queue) {
	t.Log("--- testing scale up with incorrect identity ---")
	for i := 0; i < 5; i++ {
		msg := fmt.Sprintf("Message - %d", i)
		_ = sbQueue.Send(context.Background(), servicebus.NewMessageFromString(msg))
	}

	// scale up should fail as we are using the incorrect identity
	assert.True(t, WaitForDeploymentReplicaCountChange(t, kc, deploymentName, testNamespace, 30, 1) == 0,
		"replica count should be 0 after 1 minute")
}

func testScaleUpWithCorrectIdentity(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale up with correct identity ---")
	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlDeleteWithTemplate(t, data, "triggerAuthTemplate", triggerAuthTemplate)

	KubectlApplyWithTemplate(t, data, "triggerAuthTemplateWithIdentityID", triggerAuthTemplateWithIdentityID)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

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

func cleanupServiceBusQueue(t *testing.T, sbQueueManager *servicebus.QueueManager) {
	t.Log("--- cleaning up ---")
	err := sbQueueManager.Delete(context.Background(), queueName)
	assert.NoErrorf(t, err, "cannot delete service bus queue - %s", err)
}
