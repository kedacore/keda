//go:build e2e
// +build e2e

package aws_identity_external_id_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName = "aws-identity-external-id-test"
)

type templateData struct {
	TestNamespace             string
	DeploymentName            string
	ScaledObjectName          string
	TriggerAuthenticationName string
	SecretName                string
	AwsRegion                 string
	RoleArn                   string
	ExternalID                string
	SqsQueue                  string
}

const (
	triggerAuthTemplateWithRoleArnAndExternalID = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: aws
    roleArn: {{.RoleArn}}
    externalID: {{.ExternalID}}
`

	triggerAuthTemplateWithRoleArnOnly = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: aws
    roleArn: {{.RoleArn}}
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
        ports:
        - containerPort: 80
`

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  maxReplicaCount: 1
  minReplicaCount: 0
  pollingInterval: 5
  cooldownPeriod: 1
  triggers:
    - type: aws-sqs-queue
      authenticationRef:
        name: {{.TriggerAuthenticationName}}
      metadata:
        awsRegion: {{.AwsRegion}}
        queueURL: {{.SqsQueue}}
        queueLength: "1"
`
)

var (
	testNamespace      = fmt.Sprintf("%s-ns", testName)
	deploymentName     = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName   = fmt.Sprintf("%s-so", testName)
	secretName         = fmt.Sprintf("%s-secret", testName)
	sqsQueueName       = fmt.Sprintf("external-id-queue-%d", GetRandomNumber())
	awsAccessKeyID     = os.Getenv("TF_AWS_ACCESS_KEY")
	awsSecretAccessKey = os.Getenv("TF_AWS_SECRET_KEY")
	awsRegion          = os.Getenv("TF_AWS_REGION")
	awsRoleArn         = os.Getenv("TF_AWS_ROLE_ARN_EXTERNAL_ID")
	awsExternalID      = os.Getenv("TF_AWS_EXTERNAL_ID")
	maxReplicaCount    = 1
	minReplicaCount    = 0
	sqsMessageCount    = 2
)

func TestAwsExternalIDPodIdentity(t *testing.T) {
	// Skip if required environment variables are not set
	if awsRoleArn == "" || awsExternalID == "" {
		t.Skip("Skipping test: TF_AWS_ROLE_ARN_EXTERNAL_ID and TF_AWS_EXTERNAL_ID must be set")
	}

	// setup SQS
	sqsClient := createSqsClient()
	queue := createSqsQueue(t, sqsClient)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData(*queue.QueueUrl)
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling with correct external ID - should scale
	testScaleWithCorrectExternalID(t, kc, data, sqsClient, queue.QueueUrl)
	// test scaling without external ID - should not scale (role requires external ID)
	testScaleWithoutExternalID(t, kc, data, sqsClient, queue.QueueUrl)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
	cleanupQueue(t, sqsClient, queue.QueueUrl)
}

// testScaleWithCorrectExternalID verifies that scaling works when the correct external ID is provided
func testScaleWithCorrectExternalID(t *testing.T, kc *kubernetes.Clientset, data templateData, sqsClient *sqs.Client, queueURL *string) {
	t.Log("--- testing scaling out with correct external ID ---")
	data.ScaledObjectName = "scale-with-external-id"
	data.TriggerAuthenticationName = "scale-with-external-id"
	KubectlApplyWithTemplate(t, data, "triggerAuthTemplateWithRoleArnAndExternalID", triggerAuthTemplateWithRoleArnAndExternalID)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	addMessages(t, sqsClient, queueURL, sqsMessageCount)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 180, 1),
		"replica count should be 1 after 3 minutes")
	testScaleIn(t, kc, sqsClient, queueURL)
	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlDeleteWithTemplate(t, data, "triggerAuthTemplateWithRoleArnAndExternalID", triggerAuthTemplateWithRoleArnAndExternalID)
}

// testScaleWithoutExternalID verifies that scaling fails when external ID is not provided
// but the role requires it (the role should have an ExternalId condition in its trust policy)
func testScaleWithoutExternalID(t *testing.T, kc *kubernetes.Clientset, data templateData, sqsClient *sqs.Client, queueURL *string) {
	t.Log("--- testing scaling with missing external ID (should not scale) ---")
	data.ScaledObjectName = "scale-without-external-id"
	data.TriggerAuthenticationName = "scale-without-external-id"
	KubectlApplyWithTemplate(t, data, "triggerAuthTemplateWithRoleArnOnly", triggerAuthTemplateWithRoleArnOnly)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	addMessages(t, sqsClient, queueURL, sqsMessageCount)
	// replicas shouldn't change because the role requires an external ID
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
	testScaleIn(t, kc, sqsClient, queueURL)
	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlDeleteWithTemplate(t, data, "triggerAuthTemplateWithRoleArnOnly", triggerAuthTemplateWithRoleArnOnly)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, sqsClient *sqs.Client, queueURL *string) {
	t.Log("--- testing scaling in ---")
	totalDeletedMessages := 0

	for {
		response, _ := sqsClient.ReceiveMessage(context.Background(), &sqs.ReceiveMessageInput{
			QueueUrl:            queueURL,
			MaxNumberOfMessages: int32(sqsMessageCount),
		})
		if response != nil {
			for _, message := range response.Messages {
				_, err := sqsClient.DeleteMessage(context.Background(), &sqs.DeleteMessageInput{
					QueueUrl:      queueURL,
					ReceiptHandle: message.ReceiptHandle,
				})
				assert.NoErrorf(t, err, "cannot delete message - %s", err)
				totalDeletedMessages++
			}
		}
		if totalDeletedMessages >= sqsMessageCount {
			break
		}

		time.Sleep(time.Second)
	}

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 180, 1),
		"replica count should be 0 after 3 minutes")
}

func addMessages(t *testing.T, sqsClient *sqs.Client, queueURL *string, messages int) {
	for i := 0; i < messages; i++ {
		msg := fmt.Sprintf("Message - %d", i)
		_, err := sqsClient.SendMessage(context.Background(), &sqs.SendMessageInput{
			QueueUrl:     queueURL,
			MessageBody:  aws.String(msg),
			DelaySeconds: 10,
		})
		assert.NoErrorf(t, err, "cannot send message - %s", err)
	}
}

func createSqsQueue(t *testing.T, sqsClient *sqs.Client) *sqs.CreateQueueOutput {
	queue, err := sqsClient.CreateQueue(context.Background(), &sqs.CreateQueueInput{
		QueueName: &sqsQueueName,
		Attributes: map[string]string{
			"DelaySeconds":           "60",
			"MessageRetentionPeriod": "86400",
		}})
	assert.NoErrorf(t, err, "failed to create queue - %s", err)
	return queue
}

func cleanupQueue(t *testing.T, sqsClient *sqs.Client, queueURL *string) {
	t.Log("--- cleaning up ---")
	_, err := sqsClient.DeleteQueue(context.Background(), &sqs.DeleteQueueInput{
		QueueUrl: queueURL,
	})
	assert.NoErrorf(t, err, "cannot delete queue - %s", err)
}

func createSqsClient() *sqs.Client {
	configOptions := make([]func(*config.LoadOptions) error, 0)
	configOptions = append(configOptions, config.WithRegion(awsRegion))
	cfg, _ := config.LoadDefaultConfig(context.Background(), configOptions...)
	cfg.Credentials = credentials.NewStaticCredentialsProvider(awsAccessKeyID, awsSecretAccessKey, "")
	return sqs.NewFromConfig(cfg)
}

func getTemplateData(sqsQueue string) (templateData, []Template) {
	return templateData{
			TestNamespace:    testNamespace,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
			SecretName:       secretName,
			AwsRegion:        awsRegion,
			RoleArn:          awsRoleArn,
			ExternalID:       awsExternalID,
			SqsQueue:         sqsQueue,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
		}
}
