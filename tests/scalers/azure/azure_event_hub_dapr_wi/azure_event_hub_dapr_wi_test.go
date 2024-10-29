//go:build e2e
// +build e2e

package azure_event_hub_dapr_wi_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
	azurehelper "github.com/kedacore/keda/v2/tests/scalers/azure/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName              = "azure-event-hub-dapr"
	eventhubConsumerGroup = "$Default"
)

var (
	storageConnectionString = os.Getenv("TF_AZURE_STORAGE_CONNECTION_STRING")
	storageAccountName      = getValueFromConnectionString(storageConnectionString, "AccountName")
	storageAccountKey       = getValueFromConnectionString(storageConnectionString, "AccountKey")
	checkpointContainerName = fmt.Sprintf("blob-checkpoint-%d", GetRandomNumber())
	testNamespace           = fmt.Sprintf("%s-ns", testName)
	secretName              = fmt.Sprintf("%s-secret", testName)
	deploymentName          = fmt.Sprintf("%s-deployment", testName)
	triggerAuthName         = fmt.Sprintf("%s-ta", testName)
	scaledObjectName        = fmt.Sprintf("%s-so", testName)
)

type templateData struct {
	TestNamespace            string
	SecretName               string
	EventHubConnection       string
	StorageConnection        string
	Base64EventHubConnection string
	Base64StorageConnection  string
	StorageAccountName       string
	StorageAccountKey        string
	DeploymentName           string
	TriggerAuthName          string
	ScaledObjectName         string
	CheckpointContainerName  string
	ConsumerGroup            string
	EventHubName             string
	EventHubNamespaceName    string
}

const (
	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  connection: {{.Base64EventHubConnection}}
  storageConnection: {{.Base64StorageConnection}}
stringData:
  azure-eventhub-binding.yaml: |
    apiVersion: dapr.io/v1alpha1
    kind: Component
    metadata:
      name: azure-eventhub-dapr
    spec:
      type: bindings.azure.eventhubs
      version: v1
      metadata:
      - name: connectionString
        value: {{.EventHubConnection}}
      - name: consumerGroup
        value: {{.ConsumerGroup}}
      - name: storageAccountName
        value: {{.StorageAccountName}}
      - name: storageAccountKey
        value: {{.StorageAccountKey}}
      - name: storageContainerName
        value: {{.CheckpointContainerName}}
`

	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.DeploymentName}}
  template:
    metadata:
      labels:
        app: {{.DeploymentName}}
    spec:
      volumes:
      - name: {{.SecretName}}
        secret:
          secretName: {{.SecretName}}
      containers:
        - name: dapr
          image: daprio/daprd:1.9.5-mariner
          imagePullPolicy: Always
          command: ["./daprd", "-app-id", "azure-eventhub-dapr", "-app-port", "3000", "-components-path", "/components", "-log-level", "debug"]
          volumeMounts:
          - mountPath: "/components"
            name: {{.SecretName}}
            readOnly: true
        - name: {{.DeploymentName}}
          image: ghcr.io/kedacore/tests-azure-eventhub-dapr
          env:
            - name: APP_PORT
              value: "3000"
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

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  minReplicaCount: 0
  maxReplicaCount: 1
  cooldownPeriod: 10
  triggers:
    - authenticationRef:
        name: {{.TriggerAuthName}}
      metadata:
        activationUnprocessedEventThreshold: '10'
        blobContainer: {{.CheckpointContainerName}}
        checkpointStrategy: dapr
        consumerGroup: {{.ConsumerGroup}}
        unprocessedEventThreshold: '64'
        eventHubName: {{.EventHubName}}
        eventHubNamespace: {{.EventHubNamespaceName}}
        storageAccountName: {{.StorageAccountName}}
      type: azure-eventhub
`
)

func TestScaler(t *testing.T) {
	ctx := context.Background()
	t.Log("--- setting up ---")
	require.NotEmpty(t, storageConnectionString, "TF_AZURE_STORAGE_CONNECTION_STRING env variable is required for azure eventhub test")

	eventHubHelper := azurehelper.NewEventHubHelper(t)
	eventHubHelper.CreateEventHub(ctx, t)
	blobClient, err := azblob.NewClientFromConnectionString(storageConnectionString, nil)
	assert.NoErrorf(t, err, "cannot create the queue client - %s", err)
	_, err = blobClient.CreateContainer(ctx, checkpointContainerName, nil)
	assert.NoErrorf(t, err, "cannot create the container - %s", err)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData(eventHubHelper)

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// We need to wait till consumer creates the checkpoint
	eventHubHelper.PublishEventHubdEvents(ctx, t, 1)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 1),
		"replica count should be 1 after 1 minute")
	time.Sleep(time.Duration(60) * time.Second)
	KubectlApplyMultipleWithTemplate(t, data, []Template{{Name: "scaledObjectTemplate", Config: scaledObjectTemplate}})

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testActivation(ctx, t, kc, eventHubHelper)
	testScaleOut(ctx, t, kc, eventHubHelper)
	testScaleIn(t, kc)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
	eventHubHelper.DeleteEventHub(ctx, t)
	_, err = blobClient.DeleteContainer(ctx, checkpointContainerName, nil)
	assert.NoErrorf(t, err, "cannot delete the container - %s", err)
}

func getTemplateData(eventHubHelper azurehelper.EventHubHelper) (templateData, []Template) {
	base64EventhubConnection := base64.StdEncoding.EncodeToString([]byte(eventHubHelper.ConnectionString()))
	base64StorageConnection := base64.StdEncoding.EncodeToString([]byte(storageConnectionString))

	return templateData{
			TestNamespace:            testNamespace,
			SecretName:               secretName,
			EventHubConnection:       eventHubHelper.ConnectionString(),
			StorageConnection:        storageConnectionString,
			Base64EventHubConnection: base64EventhubConnection,
			Base64StorageConnection:  base64StorageConnection,
			StorageAccountName:       storageAccountName,
			StorageAccountKey:        storageAccountKey,
			CheckpointContainerName:  checkpointContainerName,
			DeploymentName:           deploymentName,
			ScaledObjectName:         scaledObjectName,
			TriggerAuthName:          triggerAuthName,
			ConsumerGroup:            eventhubConsumerGroup,
			EventHubName:             eventHubHelper.EventHub(),
			EventHubNamespaceName:    eventHubHelper.EventHubNamespace(),
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
		}
}

func testActivation(ctx context.Context, t *testing.T, kc *kubernetes.Clientset, eventHubHelper azurehelper.EventHubHelper) {
	t.Log("--- testing activation ---")
	eventHubHelper.PublishEventHubdEvents(ctx, t, 8)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)
}

func testScaleOut(ctx context.Context, t *testing.T, kc *kubernetes.Clientset, eventHubHelper azurehelper.EventHubHelper) {
	t.Log("--- testing scale out ---")
	eventHubHelper.PublishEventHubdEvents(ctx, t, 8)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 1),
		"replica count should be 1 after 1 minute")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

func getValueFromConnectionString(storageAccountConnectionString string, keyName string) string {
	items := strings.Split(storageAccountConnectionString, ";")
	for _, item := range items {
		keyValue := strings.SplitN(item, "=", 2)
		if keyValue[0] == keyName {
			return keyValue[1]
		}
	}

	return ""
}
