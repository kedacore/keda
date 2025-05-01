//go:build e2e
// +build e2e

package azure_service_bus_topic_regex_test

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
	testName = "azure-sb-topic-regex-test"
)

var (
	connectionString   = os.Getenv("TF_AZURE_SERVICE_BUS_CONNECTION_STRING")
	testNamespace      = fmt.Sprintf("%s-ns", testName)
	secretName         = fmt.Sprintf("%s-secret", testName)
	deploymentName     = fmt.Sprintf("%s-deployment", testName)
	triggerAuthName    = fmt.Sprintf("%s-ta", testName)
	scaledObjectName   = fmt.Sprintf("%s-so", testName)
	topicName          = fmt.Sprintf("topic-%d", GetRandomNumber())
	subscriptionPrefix = fmt.Sprintf("subs-%d", GetRandomNumber())
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
	require.NotEmpty(t, connectionString, "TF_AZURE_SERVICE_BUS_CONNECTION_STRING env variable is required for service bus tests")

	client, adminClient := setupServiceBusTopicAndSubscription(t)

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testScale(t, kc, client, data)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
	cleanupServiceBusTopic(t, adminClient, topicName)
}

func setupServiceBusTopicAndSubscription(t *testing.T) (*azservicebus.Client, *admin.Client) {
	adminClient, err := admin.NewClientFromConnectionString(connectionString, nil)
	assert.NoErrorf(t, err, "cannot connect to service bus namespace - %s", err)

	// Delete the topic if already exists
	_, _ = adminClient.DeleteTopic(context.Background(), topicName, nil)

	_, err = adminClient.CreateTopic(context.Background(), topicName, nil)
	assert.NoErrorf(t, err, "cannot create the topic - %s", err)

	subscriptionName1 := fmt.Sprintf("%s-1", subscriptionPrefix)
	setupSubscription(t, adminClient, topicName, subscriptionName1)
	subscriptionName2 := fmt.Sprintf("%s-2", subscriptionPrefix)
	setupSubscription(t, adminClient, topicName, subscriptionName2)

	client, err := azservicebus.NewClientFromConnectionString(connectionString, nil)
	assert.NoErrorf(t, err, "cannot connect to service bus namespace - %s", err)

	return client, adminClient
}

func setupSubscription(t *testing.T, adminClient *admin.Client, topicName, subscriptionName string) {
	_, err := adminClient.CreateSubscription(context.Background(), topicName, subscriptionName, nil)
	assert.NoErrorf(t, err, "cannot create the subscription 1 - %s", err)

	_, err = adminClient.DeleteRule(context.Background(), topicName, subscriptionName, "$Default", &admin.DeleteRuleOptions{})
	assert.NoErrorf(t, err, "cannot delete default filter rule for subscription - %s", err)

	ruleName := "filterRule"
	_, err = adminClient.CreateRule(context.Background(), topicName, subscriptionName, &admin.CreateRuleOptions{
		Name: &ruleName,
		Filter: &admin.CorrelationFilter{
			To: &subscriptionName,
		},
	})
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

func testScale(t *testing.T, kc *kubernetes.Clientset, client *azservicebus.Client, data templateData) {
	t.Log("--- testing scale out ---")

	// send messages to subscription 1
	addMessages(t, client, fmt.Sprintf("%s-1", subscriptionPrefix), 2)
	// send messages to subscription 2
	addMessages(t, client, fmt.Sprintf("%s-2", subscriptionPrefix), 4)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 6, 60, 1),
		"replica count should be 1 after 1 minute")

	// check different aggregation operations
	data.Operation = "max"
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 4, 60, 3),
		"replica count should be 4 after 3 minute")

	data.Operation = "avg"
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 3, 60, 1),
		"replica count should be 3 after 1 minute")
}

func addMessages(t *testing.T, client *azservicebus.Client, subscriptionName string, count int) {
	sender, err := client.NewSender(topicName, nil)
	assert.NoErrorf(t, err, "cannot create the sender - %s", err)
	for i := 0; i < count; i++ {
		msg := fmt.Sprintf("Message - %d", i)
		_ = sender.SendMessage(context.Background(), &azservicebus.Message{
			Body: []byte(msg),
			To:   &subscriptionName,
		}, nil)
	}
}

func cleanupServiceBusTopic(t *testing.T, adminClient *admin.Client, topicName string) {
	t.Log("--- cleaning up ---")
	_, err := adminClient.DeleteTopic(context.Background(), topicName, nil)
	assert.NoErrorf(t, err, "cannot delete service bus topic - %s", err)
}
