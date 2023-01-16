//go:build e2e
// +build e2e

package azure_service_bus_topic_aad_pod_identity_test

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
	testName = "azure-service-bus-topic-aad-pod-identity-test"
)

var (
	connectionString = os.Getenv("TF_AZURE_SERVICE_BUS_CONNECTION_STRING")
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	triggerAuthName  = fmt.Sprintf("%s-ta", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	topicName        = fmt.Sprintf("topic-%d", GetRandomNumber())
	subscriptionName = fmt.Sprintf("subs-%d", GetRandomNumber())
)

type templateData struct {
	TestNamespace       string
	Connection          string
	DeploymentName      string
	TriggerAuthName     string
	ScaledObjectName    string
	TopicName           string
	SubscriptionName    string
	ServiceBusNamespace string
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
    provider: azure
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
      namespace: {{.ServiceBusNamespace}}
      activationMessageCount: "5"
    authenticationRef:
      name: {{.TriggerAuthName}}
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	require.NotEmpty(t, connectionString, "TF_AZURE_SERVICE_BUS_CONNECTION_STRING env variable is required for service bus tests")

	client, adminClient, namespace := setupServiceBusTopicAndSubscription(t)

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	data.ServiceBusNamespace = namespace

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testActivation(t, kc, client)
	testScaleOut(t, kc, client)
	testScaleIn(t, kc, adminClient)

	// cleanup
	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
	cleanupServiceBusTopic(t, adminClient)
}

func setupServiceBusTopicAndSubscription(t *testing.T) (*azservicebus.Client, *admin.Client, string) {
	adminClient, err := admin.NewClientFromConnectionString(connectionString, nil)
	assert.NoErrorf(t, err, "cannot connect to service bus namespace - %s", err)

	// Delete the topic if already exists
	_, _ = adminClient.DeleteTopic(context.Background(), topicName, nil)

	_, err = adminClient.CreateTopic(context.Background(), topicName, nil)
	assert.NoErrorf(t, err, "cannot create the topic - %s", err)
	_, err = adminClient.CreateSubscription(context.Background(), topicName, subscriptionName, nil)
	assert.NoErrorf(t, err, "cannot create the subscription - %s", err)

	namespace, err := adminClient.GetNamespaceProperties(context.Background(), nil)
	assert.NoErrorf(t, err, "cannot get namespace info - %s", err)

	client, err := azservicebus.NewClientFromConnectionString(connectionString, nil)
	assert.NoErrorf(t, err, "cannot connect to service bus namespace - %s", err)

	return client, adminClient, namespace.Name
}

func getTemplateData() (templateData, []Template) {
	base64ConnectionString := base64.StdEncoding.EncodeToString([]byte(connectionString))

	return templateData{
			TestNamespace:    testNamespace,
			Connection:       base64ConnectionString,
			DeploymentName:   deploymentName,
			TriggerAuthName:  triggerAuthName,
			ScaledObjectName: scaledObjectName,
			TopicName:        topicName,
			SubscriptionName: subscriptionName,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, client *azservicebus.Client) {
	t.Log("--- testing activation ---")
	addMessages(t, client, 4)

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

	_, err := adminClient.DeleteTopic(context.Background(), topicName, nil)
	assert.NoErrorf(t, err, "cannot delete the topic - %s", err)
	_, err = adminClient.CreateTopic(context.Background(), topicName, nil)
	assert.NoErrorf(t, err, "cannot create the topic - %s", err)
	_, err = adminClient.CreateSubscription(context.Background(), topicName, subscriptionName, nil)
	assert.NoErrorf(t, err, "cannot create the subscription - %s", err)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

func addMessages(t *testing.T, client *azservicebus.Client, count int) {
	sender, err := client.NewSender(topicName, nil)
	assert.NoErrorf(t, err, "cannot create the sender - %s", err)
	for i := 0; i < count; i++ {
		msg := fmt.Sprintf("Message - %d", i)
		_ = sender.SendMessage(context.Background(), &azservicebus.Message{
			Body: []byte(msg),
		}, nil)
	}
}

func cleanupServiceBusTopic(t *testing.T, adminClient *admin.Client) {
	t.Log("--- cleaning up ---")
	_, err := adminClient.DeleteTopic(context.Background(), topicName, nil)
	assert.NoErrorf(t, err, "cannot delete service bus topic - %s", err)
}
