//go:build e2e
// +build e2e

package accurate_scaling_strategy_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper" // For helper methods
)

var _ = godotenv.Load("../../.env") // For loading env variables from .env

const (
	testName = "accurate-scaling-strategy-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	scaledJobName    = fmt.Sprintf("%s-sj", testName)
	connectionString = os.Getenv("TF_AZURE_STORAGE_CONNECTION_STRING")
	queueName        = fmt.Sprintf("queue-%d", GetRandomNumber())
	secretName       = fmt.Sprintf("%s-secret", testName)
)

// YAML templates for your Kubernetes resources
const (
	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  AzureWebJobsStorage: {{.Connection}}
`

	scaledJobTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{.ScaledJobName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.ScaledJobName}}
spec:
  jobTargetRef:
    template:
      spec:
        containers:
          - name: sleeper
            image: docker.io/library/busybox
            command:
            - sleep
            - "30"
            imagePullPolicy: IfNotPresent
            envFrom:
            - secretRef:
                name: {{.SecretName}}
        restartPolicy: Never
    backoffLimit: 1
  pollingInterval: 5
  maxReplicaCount: 10
  scalingStrategy:
    strategy: "accurate"
  triggers:
    - type: azure-queue
      metadata:
        queueName: {{.QueueName}}
        connectionFromEnv: AzureWebJobsStorage
        queueLength: '1'
`
)

type templateData struct {
	ScaledJobName string
	TestNamespace string
	QueueName     string
	SecretName    string
	Connection    string
}

func TestScalingStrategy(t *testing.T) {
	// Setup
	ctx := context.Background()
	t.Log("--- setting up ---")
	require.NotEmpty(t, connectionString, "TF_AZURE_STORAGE_CONNECTION_STRING env variable is required for azure queue test")

	queueClient, err := azqueue.NewQueueClientFromConnectionString(connectionString, queueName, nil)
	assert.NoErrorf(t, err, "cannot create the queue client - %s", err)
	_, err = queueClient.Create(ctx, nil)
	assert.NoErrorf(t, err, "cannot create the queue - %s", err)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
		_, err := queueClient.Delete(ctx, nil)
		assert.NoErrorf(t, err, "cannot delete the queue - %s", err)
	})

	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	testAccurateScaling(ctx, t, kc, queueClient)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			// Populate fields required in YAML templates
			ScaledJobName: scaledJobName,
			TestNamespace: testNamespace,
			QueueName:     queueName,
			Connection:    base64.StdEncoding.EncodeToString([]byte(connectionString)),
			SecretName:    secretName,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "scaledJobTemplate", Config: scaledJobTemplate},
		}
}

func testAccurateScaling(ctx context.Context, t *testing.T, kc *kubernetes.Clientset, client *azqueue.QueueClient) {
	iterationCount := 60

	// Base case (number of scale = maxScale since pendingJobs = 0). Enqueue up 4 messages
	enqueueMessages(ctx, t, client, 4)
	assert.True(t, WaitForRunningPodCount(t, kc, scaledJobName, testNamespace, 4, iterationCount, 1),
		"running pod count should be %d after %d iterations", 4, iterationCount)

	// Clear the messages to simulate message consumption
	_, err := client.ClearMessages(ctx, nil)
	assert.NoErrorf(t, err, "cannot clear queue - %s", err)

	// Wait for job completion
	WaitForAllJobsSuccess(t, kc, testNamespace, 90, 1)

	// Test the cap condition (maxScale + runningJobs > maxReplicaCount). Enqueue 4 messages
	enqueueMessages(ctx, t, client, 4)
	assert.True(t, WaitForRunningPodCount(t, kc, scaledJobName, testNamespace, 4, iterationCount, 1),
		"running pod count should be %d after %d iterations", 4, iterationCount)

	// Clear the messages to simulate message consumption
	_, err = client.ClearMessages(ctx, nil)
	assert.NoErrorf(t, err, "cannot clear queue - %s", err)

	// Enqueue 8 more messages to trigger the cap condition
	enqueueMessages(ctx, t, client, 8)
	assert.True(t, WaitForRunningPodCount(t, kc, scaledJobName, testNamespace, 10, iterationCount, 1),
		"running pod count should be %d after %d iterations", 10, iterationCount)

	_, err = client.ClearMessages(ctx, nil)
	assert.NoErrorf(t, err, "cannot clear queue - %s", err)

	// Message cleanup and wait for jobs to complete
	WaitForAllJobsSuccess(t, kc, testNamespace, 120, 1)
}

func enqueueMessages(ctx context.Context, t *testing.T, client *azqueue.QueueClient, count int) {
	for i := 0; i < count; i++ {
		msg := fmt.Sprintf("Message - %d", i)
		_, err := client.EnqueueMessage(ctx, msg, nil)
		assert.NoErrorf(t, err, "cannot enqueue message - %s", err)
		t.Logf("Message queued")
	}
}
