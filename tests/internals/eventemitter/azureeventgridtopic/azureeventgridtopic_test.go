//go:build e2e
// +build e2e

package azureeventgridtopic_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/messaging"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/admin"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "eventemitter-azureeventgridtopic-test"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

var (
	connectionString        = os.Getenv("TF_AZURE_SERVICE_BUS_EVENTGRID_CONNECTION_STRING")
	topicName               = os.Getenv("TF_AZURE_SB_EVENT_GRID_RECEIVE_TOPIC")
	eventGridEndpoint       = os.Getenv("TF_AZURE_EVENT_GRID_TOPIC_ENDPOINT")
	eventGridKey            = os.Getenv("TF_AZURE_EVENT_GRID_TOPIC_KEY")
	subscriptionName        = fmt.Sprintf("subs-%d", GetRandomNumber())
	namespace               = fmt.Sprintf("%s-ns", testName)
	clientName              = fmt.Sprintf("%s-client", testName)
	cloudeventSourceName    = fmt.Sprintf("%s-aeg", testName)
	clusterName             = "test-cluster"
	expectedSubject         = fmt.Sprintf("/%s/%s/scaledobject/%s", clusterName, namespace, scaledObjectName)
	expectedSource          = fmt.Sprintf("/%s/keda/keda", clusterName)
	expectedType            = "keda.scaledobject.ready.v1"
	monitoredDeploymentName = "monitored-deployment"
	sutDeploymentName       = "sut-deployment"
	scaledObjectName        = fmt.Sprintf("%s-so", testName)
	secretName              = fmt.Sprintf("%s-secret", testName)
	triggerAuthName         = fmt.Sprintf("%s-triggerauth", testName)
)

type templateData struct {
	TestNamespace              string
	EventGridEndpoint          string
	EventGridKey               string
	ClientName                 string
	CloudEventSourceName       string
	CloudEventHTTPReceiverName string
	CloudEventHTTPServiceName  string
	CloudEventHTTPServiceURL   string
	ClusterName                string
	MonitoredDeploymentName    string
	SutDeploymentName          string
	ScaledObjectName           string
	SecretName                 string
	TriggerAuthName            string
}

const (
	cloudEventSourceTemplate = `
  apiVersion: eventing.keda.sh/v1alpha1
  kind: CloudEventSource
  metadata:
    name: {{.CloudEventSourceName}}
    namespace: {{.TestNamespace}}
  spec:
    authenticationRef:
      name: {{.TriggerAuthName}}
    clusterName: {{.ClusterName}}
    destination:
      azureEventGridTopic:
        endpoint: {{.EventGridEndpoint}}
  `
	monitoredDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.MonitoredDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    deploy: workload-test
spec:
    replicas: 0
    selector:
      matchLabels:
        pod: workload-test
    template:
      metadata:
        labels:
          pod: workload-test
      spec:
        containers:
        - name: nginx
          image: 'ghcr.io/nginx/nginx-unprivileged:1.26'`

	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  key: {{.EventGridKey}}
`

	triggerAuthTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: accessKey
    name: {{.SecretName}}
    key: key
`

	sutDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.SutDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    deploy: workload-sut
spec:
  replicas: 0
  selector:
    matchLabels:
      pod: workload-sut
  template:
    metadata:
      labels:
        pod: workload-sut
    spec:
      containers:
      - name: nginx
        image: 'ghcr.io/nginx/nginx-unprivileged:1.26'`

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.SutDeploymentName}}
  pollingInterval: 5
  cooldownPeriod: 5
  minReplicaCount: 0
  maxReplicaCount: 10
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 5
  triggers:
  - type: kubernetes-workload
    metadata:
      podSelector: 'pod=workload-test'
      value: '1'
      activationValue: '3'`
)

func TestEventEmitter(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, namespace, data, templates)

	client, adminClient := setupServiceBusTopicAndSubscription(t)
	testEventSourceEmitValue(t, kc, data, client)

	DeleteKubernetesResources(t, namespace, data, templates)
	cleanupServiceBusSubscription(t, adminClient)
}

// tests error events emitted
func testEventSourceEmitValue(t *testing.T, _ *kubernetes.Clientset, data templateData, client *azservicebus.Client) {
	t.Log("--- test emitting eventsource about scaledobject err---")

	var wg sync.WaitGroup
	wg.Add(1)
	go func(t *testing.T, count int, client *azservicebus.Client) {
		checkMessage(t, count, client)
		wg.Done()
	}(t, 1, client)

	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	wg.Wait()
}

// help function to load template data
func getTemplateData() (templateData, []Template) {
	base64EventGridKey := base64.StdEncoding.EncodeToString([]byte(eventGridKey))
	return templateData{
			TestNamespace:           namespace,
			ClientName:              clientName,
			CloudEventSourceName:    cloudeventSourceName,
			ClusterName:             clusterName,
			MonitoredDeploymentName: monitoredDeploymentName,
			SutDeploymentName:       sutDeploymentName,
			ScaledObjectName:        scaledObjectName,
			EventGridEndpoint:       eventGridEndpoint,
			EventGridKey:            base64EventGridKey,
			TriggerAuthName:         triggerAuthName,
			SecretName:              secretName,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
			{Name: "cloudEventSourceTemplate", Config: cloudEventSourceTemplate},
			{Name: "monitoredDeploymentTemplate", Config: monitoredDeploymentTemplate},
			{Name: "sutDeploymentTemplate", Config: sutDeploymentTemplate},
		}
}

func setupServiceBusTopicAndSubscription(t *testing.T) (*azservicebus.Client, *admin.Client) {
	adminClient, err := admin.NewClientFromConnectionString(connectionString, nil)
	assert.NoErrorf(t, err, "cannot connect to service bus namespace - %s", err)

	_, err = adminClient.GetTopic(context.Background(), topicName, nil)
	require.NoErrorf(t, err, "cannot get the topic - %s", err)

	subscription, err := adminClient.GetSubscription(context.Background(), topicName, subscriptionName, nil)
	assert.NoErrorf(t, err, "cannot get the Subscription - %s", err)

	if subscription == nil {
		_, err = adminClient.CreateSubscription(context.Background(), topicName, subscriptionName, nil)
		assert.NoErrorf(t, err, "cannot create the subscription - %s", err)
	}

	client, err := azservicebus.NewClientFromConnectionString(connectionString, nil)
	assert.NoErrorf(t, err, "cannot connect to service bus namespace - %s", err)

	return client, adminClient
}

func cleanupServiceBusSubscription(t *testing.T, adminClient *admin.Client) {
	t.Log("--- cleaning up ---")
	_, err := adminClient.DeleteSubscription(context.Background(), topicName, subscriptionName, nil)
	assert.NoErrorf(t, err, "cannot delete service bus topic - %s", err)
}

func checkMessage(t *testing.T, count int, client *azservicebus.Client) {
	t.Log("--- waiting getMessage ---")
	receiver, err := client.NewReceiverForSubscription(
		topicName,
		subscriptionName,
		&azservicebus.ReceiverOptions{
			ReceiveMode: azservicebus.ReceiveModePeekLock,
		},
	)
	if err != nil {
		assert.NoErrorf(t, err, "cannot create receiver - %s", err)
	}
	defer receiver.Close(context.Background())

	// We try to read the messages 3 times with a second of delay
	tries := 3
	found := false
	for i := 0; i < tries && !found; i++ {
		messages, err := receiver.ReceiveMessages(context.Background(), count, nil)
		assert.NoErrorf(t, err, "cannot receive messages - %s", err)
		assert.NotEmpty(t, messages)

		for _, message := range messages {
			event := messaging.CloudEvent{}
			err = json.Unmarshal(message.Body, &event)
			assert.NoErrorf(t, err, "cannot retrieve message - %s", err)
			if expectedSubject == *event.Subject &&
				expectedSource == event.Source &&
				expectedType == event.Type {
				found = true
			}
		}
	}

	assert.True(t, found)
}
