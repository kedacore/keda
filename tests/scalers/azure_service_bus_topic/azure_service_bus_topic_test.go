//go:build e2e
// +build e2e

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "azure-service-bus-topic-test"
)

var (
	connectionString = os.Getenv("TF_AZURE_SERVICE_BUS_CONNECTION_STRING")
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	triggerAuthName  = fmt.Sprintf("%s-ta", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	topicName        = fmt.Sprintf("topic-%d", GetRandomNumber())
	subscriptionName = fmt.Sprintf("subs-%d", GetRandomNumber())
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
      activationMessageCount: "5"
    authenticationRef:
      name: {{.TriggerAuthName}}
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	require.NotEmpty(t, connectionString, "TF_AZURE_SERVICE_BUS_CONNECTION_STRING env variable is required for service bus tests")

	sbTopicManager, sbTopic, sbSubscription := setupServiceBusTopicAndSubscription(t)

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testActivation(t, kc, sbTopic)
	testScaleUp(t, kc, sbTopic)
	testScaleDown(t, kc, sbSubscription)

	// cleanup
	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
	cleanupServiceBusTopic(t, sbTopicManager)
}

func setupServiceBusTopicAndSubscription(t *testing.T) (*servicebus.TopicManager, *servicebus.Topic, *servicebus.Subscription) {
	// Connect to service bus namespace.
	sbNamespace, err := servicebus.NewNamespace(servicebus.NamespaceWithConnectionString(connectionString))
	assert.NoErrorf(t, err, "cannot connect to service bus namespace - %s", err)

	sbTopicManager := sbNamespace.NewTopicManager()

	createTopicAndSubscription(t, sbNamespace, sbTopicManager)

	sbTopic, err := sbNamespace.NewTopic(topicName)
	assert.NoErrorf(t, err, "cannot create client for topic - %s", err)

	sbSubscription, err := sbTopic.NewSubscription(subscriptionName)
	assert.NoErrorf(t, err, "cannot create client for subscription - %s", err)

	return sbTopicManager, sbTopic, sbSubscription
}

func createTopicAndSubscription(t *testing.T, sbNamespace *servicebus.Namespace, sbTopicManager *servicebus.TopicManager) {
	// Delete service bus topic if already exists.
	sbTopics, err := sbTopicManager.List(context.Background())
	assert.NoErrorf(t, err, "cannot fetch topic list for service bus namespace - %s", err)

	// Delete service bus topic if already exists.
	for _, topic := range sbTopics {
		if topic.Name == topicName {
			t.Log("Service Bus Topic already exists. Deleting.")
			err := sbTopicManager.Delete(context.Background(), topicName)
			assert.NoErrorf(t, err, "cannot delete existing service bus topic - %s", err)
		}
	}

	// Create service bus topic.
	_, err = sbTopicManager.Put(context.Background(), topicName)
	assert.NoErrorf(t, err, "cannot create service bus topic - %s", err)

	// Create subscription within topic
	sbSubscriptionManager, err := sbNamespace.NewSubscriptionManager(topicName)
	assert.NoErrorf(t, err, "cannot create subscription manager for topic - %s", err)

	_, err = sbSubscriptionManager.Put(context.Background(), subscriptionName)
	assert.NoErrorf(t, err, "cannot create subscription for topic - %s", err)
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
			TopicName:        topicName,
			SubscriptionName: subscriptionName,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, sbTopic *servicebus.Topic) {
	t.Log("--- testing activation ---")
	addMessages(sbTopic, 4)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)
}

func testScaleUp(t *testing.T, kc *kubernetes.Clientset, sbTopic *servicebus.Topic) {
	t.Log("--- testing scale up ---")
	addMessages(sbTopic, 10)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 1),
		"replica count should be 1 after 1 minute")
}

func testScaleDown(t *testing.T, kc *kubernetes.Clientset, sbSubscription *servicebus.Subscription) {
	t.Log("--- testing scale down ---")
	var messageHandlerFunc servicebus.HandlerFunc = func(ctx context.Context, msg *servicebus.Message) error {
		return msg.Complete(ctx)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = sbSubscription.Receive(ctx, messageHandlerFunc)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

func addMessages(sbTopic *servicebus.Topic, count int) {
	for i := 0; i < count; i++ {
		msg := fmt.Sprintf("Message - %d", i)
		_ = sbTopic.Send(context.Background(), servicebus.NewMessageFromString(msg))
	}
}

func cleanupServiceBusTopic(t *testing.T, sbTopicManager *servicebus.TopicManager) {
	t.Log("--- cleaning up ---")
	err := sbTopicManager.Delete(context.Background(), topicName)
	assert.NoErrorf(t, err, "cannot delete service bus topic - %s", err)
}
