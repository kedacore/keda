package azure_service_bus_topic_test

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
	testName = "azure-service-bus-topic-test"
)

var (
	connectionString = os.Getenv("AZURE_SERVICE_BUS_CONNECTION_STRING")
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	triggerAuthName  = fmt.Sprintf("%s-ta", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	topicName        = fmt.Sprintf("%s-topic", testName)
	subscriptionName = fmt.Sprintf("%s-subscription", testName)
)

type templateData struct {
	TestNamespace    string
	SecretName       string
	Connection       string
	DeploymentName   string
	TriggerAuthName  string
	ScaledObjectName string
	TopicName        string
	SubscriptionName string
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
      topicName: {{.TopicName}}
      subscriptionName: {{.SubscriptionName}}
    authenticationRef:
      name: {{.TriggerAuthName}}
`
)

func TestSetup(t *testing.T) {
	require.NotEmpty(t, connectionString, "AZURE_SERVICE_BUS_CONNECTION_STRING env variable is required for service bus tests")

	// Connect to service bus namespace.
	sbNamespace, err := servicebus.NewNamespace(servicebus.NamespaceWithConnectionString(connectionString))
	require.NoErrorf(t, err, "cannot connect to service bus namespace - %s", err)

	sbTopicManager := sbNamespace.NewTopicManager()
	sbTopics, err := sbTopicManager.List(context.Background())
	require.NoErrorf(t, err, "cannot fetch topic list for service bus namespace - %s", err)

	// Delete service bus topic if already exists.
	for _, topic := range sbTopics {
		if topic.Name == topicName {
			t.Log("Service Bus Topic already exists. Deleting.")
			err := sbTopicManager.Delete(context.Background(), topicName)
			require.NoErrorf(t, err, "cannot delete existing service bus topic - %s", err)
		}
	}

	// Create service bus topic.
	_, err = sbTopicManager.Put(context.Background(), topicName)
	require.NoErrorf(t, err, "cannot create service bus topic - %s", err)

	// Create subscription within topic
	sbSubscriptionManager, err := sbNamespace.NewSubscriptionManager(topicName)
	require.NoErrorf(t, err, "cannot create subscription manager for topic - %s", err)

	_, err = sbSubscriptionManager.Put(context.Background(), subscriptionName)
	require.NoErrorf(t, err, "cannot create subscription for topic - %s", err)

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
		TopicName:        topicName,
		SubscriptionName: subscriptionName,
	}

	// Create kubernetes resources
	KubectlApplyMultipleWithTemplate(t, data, secretTemplate, deploymentTemplate, triggerAuthTemplate, scaledObjectTemplate)

	require.True(t, WaitForDeploymentReplicaCount(t, Kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after a minute")
}

func TestScaleUp(t *testing.T) {
	sbNamespace, err := servicebus.NewNamespace(servicebus.NamespaceWithConnectionString(connectionString))
	require.NoErrorf(t, err, "cannot connect to service bus namespace - %s", err)

	sbTopic, err := sbNamespace.NewTopic(topicName)
	require.NoErrorf(t, err, "cannot create for topic - %s", err)

	for i := 0; i < 5; i++ {
		_ = sbTopic.Send(context.Background(), servicebus.NewMessageFromString(fmt.Sprintf("Message - %d", i)))
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

	sbTopic, err := sbNamespace.NewTopic(topicName)
	require.NoErrorf(t, err, "cannot create client for topic - %s", err)

	sbSubscription, err := sbTopic.NewSubscription(subscriptionName)
	require.NoErrorf(t, err, "cannot create client for subscription - %s", err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = sbSubscription.Receive(ctx, messageHandlerFunc)

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
		TopicName:        topicName,
		SubscriptionName: subscriptionName,
	}

	// Delete kubernetes resources
	KubectlDeleteMultipleWithTemplate(t, data, secretTemplate, deploymentTemplate, triggerAuthTemplate, scaledObjectTemplate)

	Kc = GetKubernetesClient(t)
	DeleteNamespace(t, Kc, testNamespace)

	// Delete service bus topic.
	sbNamespace, err := servicebus.NewNamespace(servicebus.NamespaceWithConnectionString(connectionString))
	require.NoErrorf(t, err, "cannot connect to service bus namespace - %s", err)

	sbTopicManager := sbNamespace.NewTopicManager()
	err = sbTopicManager.Delete(context.Background(), topicName)
	require.NoErrorf(t, err, "cannot delete existing service bus topic - %s", err)
}
