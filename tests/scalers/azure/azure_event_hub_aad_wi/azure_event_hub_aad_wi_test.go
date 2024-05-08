//go:build e2e
// +build e2e

package azure_event_hub_aad_wi_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName              = "azure-event-hub-aad-wi"
	eventhubConsumerGroup = "$Default"
)

var (
	eventHubName              = fmt.Sprintf("keda-eh-%d", GetRandomNumber())
	namespaceConnectionString = os.Getenv("TF_AZURE_EVENTHBUS_MANAGEMENT_CONNECTION_STRING")
	eventhubConnectionString  = fmt.Sprintf("%s;EntityPath=%s", namespaceConnectionString, eventHubName)
	storageConnectionString   = os.Getenv("TF_AZURE_STORAGE_CONNECTION_STRING")
	checkpointContainerName   = fmt.Sprintf("go-checkpoint-%d", GetRandomNumber())
	testNamespace             = fmt.Sprintf("%s-ns", testName)
	secretName                = fmt.Sprintf("%s-secret", testName)
	deploymentName            = fmt.Sprintf("%s-deployment", testName)
	triggerAuthName           = fmt.Sprintf("%s-ta", testName)
	scaledObjectName          = fmt.Sprintf("%s-so", testName)
	eventHubNamespaceName     = ""
	accountName               = ""
)

type templateData struct {
	TestNamespace           string
	SecretName              string
	EventHubConnection      string
	StorageConnection       string
	DeploymentName          string
	TriggerAuthName         string
	ScaledObjectName        string
	CheckpointContainerName string
	ConsumerGroup           string
	AccountName             string
	EventHubName            string
	EventHubNamespaceName   string
}

const (
	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  connection: {{.EventHubConnection}}
  storageConnection: {{.StorageConnection}}
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
      containers:
        - name: {{.DeploymentName}}
          image: ghcr.io/kedacore/tests-azure-eventhub-golang
          env:
            - name: EVENTHUB_CONNECTION_STRING
              valueFrom:
                secretKeyRef:
                  name: {{.SecretName}}
                  key: connection
            - name: STORAGE_CONNECTION_STRING
              valueFrom:
                secretKeyRef:
                  name: {{.SecretName}}
                  key: storageConnection
            - name: CHECKPOINT_CONTAINER
              value: {{.CheckpointContainerName}}
            - name: EVENTHUB_CONSUMERGROUP
              value: {{.ConsumerGroup}}
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
        checkpointStrategy: goSdk
        consumerGroup: {{.ConsumerGroup}}
        unprocessedEventThreshold: '64'
        eventHubName: {{.EventHubName}}
        eventHubNamespace: {{.EventHubNamespaceName}}
        storageAccountName: {{.AccountName}}
      type: azure-eventhub
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	require.NotEmpty(t, namespaceConnectionString, "TF_AZURE_EVENTHBUS_MANAGEMENT_CONNECTION_STRING env variable is required for azure eventhub test")
	require.NotEmpty(t, storageConnectionString, "TF_AZURE_STORAGE_CONNECTION_STRING env variable is required for azure eventhub test")

	adminClient, client := createEventHub(t)
	container := createContainer(t)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// We need to wait till consumer creates the checkpoint
	addEvents(t, client, 1)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 1),
		"replica count should be 1 after 1 minute")
	time.Sleep(time.Duration(60) * time.Second)
	KubectlApplyMultipleWithTemplate(t, data, []Template{{Name: "scaledObjectTemplate", Config: scaledObjectTemplate}})

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testActivation(t, kc, client)
	testScaleOut(t, kc, client)
	testScaleIn(t, kc)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
	deleteEventHub(t, adminClient)
	deleteContainer(t, container)
}

func createEventHub(t *testing.T) (*eventhub.HubManager, *eventhub.Hub) {
	eventhubManager, err := eventhub.NewHubManagerFromConnectionString(namespaceConnectionString)
	assert.NoErrorf(t, err, "cannot create eventhubManager client - %s", err)

	eventHubNamespaceURL, err := url.Parse(eventhubManager.Host)
	assert.NoErrorf(t, err, "cannot parse event hub namespace url - %s", err)
	domains := strings.Split(eventHubNamespaceURL.Hostname(), ".")
	eventHubNamespaceName = domains[0]

	opts := []eventhub.HubManagementOption{
		eventhub.HubWithPartitionCount(1),
		eventhub.HubWithMessageRetentionInDays(1),
	}
	_, err = eventhubManager.Put(context.Background(), eventHubName, opts...)
	assert.NoErrorf(t, err, "cannot create event hub - %s", err)

	eventhub, err := eventhub.NewHubFromConnectionString(eventhubConnectionString)
	assert.NoErrorf(t, err, "cannot create eventhub client - %s", err)
	return eventhubManager, eventhub
}

func deleteEventHub(t *testing.T, adminClient *eventhub.HubManager) {
	err := adminClient.Delete(context.Background(), eventHubName)
	assert.NoErrorf(t, err, "cannot delete event hub - %s", err)
}

func createContainer(t *testing.T) azblob.ContainerURL {
	// Create Blob Container
	credential, endpoint, err := azure.ParseAzureStorageBlobConnection(
		context.Background(), kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		storageConnectionString, "", "")
	assert.NoErrorf(t, err, "cannot parse storage connection string - %s", err)

	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})
	serviceURL := azblob.NewServiceURL(*endpoint, p)
	containerURL := serviceURL.NewContainerURL(checkpointContainerName)

	_, err = containerURL.Create(context.Background(), azblob.Metadata{}, azblob.PublicAccessContainer)
	assert.NoErrorf(t, err, "cannot create blob container - %s", err)

	domains := strings.Split(endpoint.Hostname(), ".")
	accountName = domains[0]
	return containerURL
}

func deleteContainer(t *testing.T, containerURL azblob.ContainerURL) {
	t.Log("--- cleaning up ---")
	_, err := containerURL.Delete(context.Background(), azblob.ContainerAccessConditions{})
	assert.NoErrorf(t, err, "cannot delete storage container - %s", err)
}

func getTemplateData() (templateData, []Template) {
	base64EventhubConnection := base64.StdEncoding.EncodeToString([]byte(eventhubConnectionString))
	base64StorageConnection := base64.StdEncoding.EncodeToString([]byte(storageConnectionString))

	return templateData{
			TestNamespace:           testNamespace,
			SecretName:              secretName,
			EventHubConnection:      base64EventhubConnection,
			StorageConnection:       base64StorageConnection,
			CheckpointContainerName: checkpointContainerName,
			DeploymentName:          deploymentName,
			ScaledObjectName:        scaledObjectName,
			TriggerAuthName:         triggerAuthName,
			ConsumerGroup:           eventhubConsumerGroup,
			AccountName:             accountName,
			EventHubName:            eventHubName,
			EventHubNamespaceName:   eventHubNamespaceName,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
		}
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, client *eventhub.Hub) {
	t.Log("--- testing activation ---")
	addEvents(t, client, 8)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, client *eventhub.Hub) {
	t.Log("--- testing scale out ---")
	addEvents(t, client, 8)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 1),
		"replica count should be 1 after 1 minute")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

func addEvents(t *testing.T, client *eventhub.Hub, count int) {
	for i := 0; i < count; i++ {
		now := time.Now()
		formatted := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
			now.Year(), now.Month(), now.Day(),
			now.Hour(), now.Minute(), now.Second())
		msg := fmt.Sprintf("Message - %s", formatted)
		err := client.Send(context.Background(), eventhub.NewEventFromString(msg))
		assert.NoErrorf(t, err, "cannot enqueue event - %s", err)
		t.Logf("event queued")
	}
}
