//go:build e2e
// +build e2e

package azure_service_bus_topic_regex_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	servicebus "github.com/Azure/azure-service-bus-go"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName = "azure-sb-topic-regex-test"
)

var (
	connectionString   = os.Getenv("AZURE_SERVICE_BUS_CONNECTION_STRING")
	testNamespace      = fmt.Sprintf("%s-ns", testName)
	secretName         = fmt.Sprintf("%s-secret", testName)
	deploymentName     = fmt.Sprintf("%s-deployment", testName)
	triggerAuthName    = fmt.Sprintf("%s-ta", testName)
	scaledObjectName   = fmt.Sprintf("%s-so", testName)
	topicName          = fmt.Sprintf("%s-topic-%d", testName, GetRandomNumber())
	subscriptionPrefix = fmt.Sprintf("%s-subscription-%d", testName, GetRandomNumber())
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
	Operation        string
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
  maxReplicaCount: 10
  triggers:
    - type: azure-servicebus
      metadata:
        topicName: {{.TopicName}}
        subscriptionName: {{.SubscriptionName}}
        messageCount: "1"
        useRegex: "true"
        operation: {{.Operation}}
      authenticationRef:
        name: {{.TriggerAuthName}}
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 5
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	require.NotEmpty(t, connectionString, "AZURE_SERVICE_BUS_CONNECTION_STRING env variable is required for service bus tests")

	sbTopicManager, sbTopic := setupServiceBusTopicAndSubscription(t)

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testScaleUp(t, kc, sbTopic, data)

	// cleanup
	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
	cleanupServiceBusTopic(t, sbTopicManager)
}

func setupServiceBusTopicAndSubscription(t *testing.T) (*servicebus.TopicManager, *servicebus.Topic) {
	// Connect to service bus namespace.
	sbNamespace, err := servicebus.NewNamespace(servicebus.NamespaceWithConnectionString(connectionString))
	assert.NoErrorf(t, err, "cannot connect to service bus namespace - %s", err)

	sbTopicManager := sbNamespace.NewTopicManager()

	createTopicAndSubscriptions(t, sbNamespace, sbTopicManager)

	sbTopic, err := sbNamespace.NewTopic(topicName)
	assert.NoErrorf(t, err, "cannot create client for topic - %s", err)

	return sbTopicManager, sbTopic
}

func createTopicAndSubscriptions(t *testing.T, sbNamespace *servicebus.Namespace, sbTopicManager *servicebus.TopicManager) {
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

	sbSub1, err := sbSubscriptionManager.Put(context.Background(), fmt.Sprintf("%s-1", subscriptionPrefix))
	assert.NoErrorf(t, err, "cannot create subscription 1 for topic - %s", err)

	err = sbSubscriptionManager.DeleteRule(context.Background(), sbSub1.Name, "$Default")
	assert.NoErrorf(t, err, "cannot delete default rule for subscription 1 - %s", err)

	label1 := "SUB1"
	_, err = sbSubscriptionManager.PutRule(context.Background(), sbSub1.Name, "testRule", servicebus.CorrelationFilter{Label: &label1})
	assert.NoErrorf(t, err, "cannot create filter rule for subscription 1 - %s", err)

	sbSub2, err := sbSubscriptionManager.Put(context.Background(), fmt.Sprintf("%s-2", subscriptionPrefix))
	assert.NoErrorf(t, err, "cannot create subscription 2 for topic - %s", err)

	err = sbSubscriptionManager.DeleteRule(context.Background(), sbSub2.Name, "$Default")
	assert.NoErrorf(t, err, "cannot delete default rule for subscription 2 - %s", err)

	label2 := "SUB2"
	_, err = sbSubscriptionManager.PutRule(context.Background(), sbSub2.Name, "testRule", servicebus.CorrelationFilter{Label: &label2})
	assert.NoErrorf(t, err, "cannot create filter rule for subscription - %s", err)
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
			SubscriptionName: fmt.Sprintf("%s.*", subscriptionPrefix),
			Operation:        "sum",
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testScaleUp(t *testing.T, kc *kubernetes.Clientset, sbTopic *servicebus.Topic, data templateData) {
	t.Log("--- testing scale up ---")

	// send messages to subscription 1
	addMessages(sbTopic, 2, "SUB1")
	// send messages to subscription 2
	addMessages(sbTopic, 4, "SUB2")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 6, 60, 1),
		"replica count should be 1 after 1 minute")

	// check different aggregation operations
	data.Operation = "max"
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 4, 60, 1),
		"replica count should be 4 after 1 minute")

	data.Operation = "avg"
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 3, 60, 1),
		"replica count should be 3 after 1 minute")
}

func addMessages(sbTopic *servicebus.Topic, count int, label string) {
	for i := 0; i < count; i++ {
		msg := servicebus.NewMessageFromString(fmt.Sprintf("Message - %d", i))
		if label != "" {
			msg.Label = label
		}
		_ = sbTopic.Send(context.Background(), msg)
	}
}

func cleanupServiceBusTopic(t *testing.T, sbTopicManager *servicebus.TopicManager) {
	t.Log("--- cleaning up ---")
	err := sbTopicManager.Delete(context.Background(), topicName)
	assert.NoErrorf(t, err, "cannot delete service bus topic - %s", err)
}
