//go:build e2e
// +build e2e

package azure_workload_identity_user_assigned_test

import (
	"context"
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

	client, adminClient, namespace := setupServiceBusQueue(t)

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	data.ServiceBusNamespace = namespace

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testScaleOutWithIncorrectIdentity(t, kc, client)
	testScaleOutWithCorrectIdentity(t, kc, data)
	testScaleIn(t, kc, adminClient)

	// cleanup
	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
	cleanupServiceBusQueue(t, adminClient)
}

func setupServiceBusQueue(t *testing.T) (*azservicebus.Client, *admin.Client, string) {
	adminClient, err := admin.NewClientFromConnectionString(connectionString, nil)
	assert.NoErrorf(t, err, "cannot connect to service bus namespace - %s", err)

	// Delete the queue if already exists
	_, _ = adminClient.DeleteQueue(context.Background(), queueName, nil)

	_, err = adminClient.CreateQueue(context.Background(), queueName, nil)
	assert.NoErrorf(t, err, "cannot create the queue - %s", err)

	namespace, err := adminClient.GetNamespaceProperties(context.Background(), nil)
	assert.NoErrorf(t, err, "cannot get namespace info - %s", err)

	client, err := azservicebus.NewClientFromConnectionString(connectionString, nil)
	assert.NoErrorf(t, err, "cannot connect to service bus namespace - %s", err)

	return client, adminClient, namespace.Name
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

func testScaleOutWithIncorrectIdentity(t *testing.T, kc *kubernetes.Clientset, client *azservicebus.Client) {
	t.Log("--- testing scale out with incorrect identity ---")
	sender, err := client.NewSender(queueName, nil)
	assert.NoErrorf(t, err, "cannot create the sender - %s", err)
	for i := 0; i < 5; i++ {
		msg := fmt.Sprintf("Message - %d", i)
		_ = sender.SendMessage(context.Background(), &azservicebus.Message{
			Body: []byte(msg),
		}, nil)
	}

	// scale out should fail as we are using the incorrect identity
	assert.True(t, WaitForDeploymentReplicaCountChange(t, kc, deploymentName, testNamespace, 30, 1) == 0,
		"replica count should be 0 after 1 minute")
}

func testScaleOutWithCorrectIdentity(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out with correct identity ---")
	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlDeleteWithTemplate(t, data, "triggerAuthTemplate", triggerAuthTemplate)

	KubectlApplyWithTemplate(t, data, "triggerAuthTemplateWithIdentityID", triggerAuthTemplateWithIdentityID)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 1),
		"replica count should be 1 after 1 minute")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, adminClient *admin.Client) {
	t.Log("--- testing scale in ---")

	_, err := adminClient.DeleteQueue(context.Background(), queueName, nil)
	assert.NoErrorf(t, err, "cannot delete the queue - %s", err)
	_, err = adminClient.CreateQueue(context.Background(), queueName, nil)
	assert.NoErrorf(t, err, "cannot create the queue - %s", err)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

func cleanupServiceBusQueue(t *testing.T, adminClient *admin.Client) {
	t.Log("--- cleaning up ---")
	_, err := adminClient.DeleteQueue(context.Background(), queueName, nil)
	assert.NoErrorf(t, err, "cannot delete service bus queue - %s", err)
}
