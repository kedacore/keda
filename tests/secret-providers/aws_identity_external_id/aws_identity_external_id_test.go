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
	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
stringData:
  AWS_ROLE_ARN: {{.RoleArn}}
  AWS_EXTERNAL_ID: {{.ExternalID}}
`

	triggerAuthWithExternalIDTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: awsRoleArn
      name: {{.SecretName}}
      key: AWS_ROLE_ARN
    - parameter: awsExternalId
      name: {{.SecretName}}
      key: AWS_EXTERNAL_ID
`

	triggerAuthWithoutExternalIDTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: awsRoleArn
      name: {{.SecretName}}
      key: AWS_ROLE_ARN
`

	deploymentTemplate = `apiVersion: apps/v1
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

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
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
        identityOwner: operator
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

func TestExternalID(t *testing.T) {
	sqsClient := createSqsClient()
	queue := createSqsQueue(t, sqsQueueName, sqsClient)

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData(*queue.QueueUrl)
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 1),
		"replica count should be 0 after 1 minute")

	// Test 1: scaling works with correct ExternalId
	testScaleWithExternalID(t, kc, data, sqsClient, queue.QueueUrl)
	// Test 2: scaling fails without ExternalId when role requires it
	testScaleFailsWithoutExternalID(t, kc, data, sqsClient, queue.QueueUrl)

	DeleteKubernetesResources(t, testNamespace, data, templates)
	cleanupQueue(t, sqsClient, queue.QueueUrl)
}

func testScaleWithExternalID(t *testing.T, kc *kubernetes.Clientset, data templateData, sqsClient *sqs.Client, queueURL *string) {
	t.Log("--- testing scaling with correct ExternalId ---")
	data.ScaledObjectName = "scale-with-external-id"
	data.TriggerAuthenticationName = "auth-with-external-id"
	KubectlApplyWithTemplate(t, data, "triggerAuthWithExternalIDTemplate", triggerAuthWithExternalIDTemplate)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	addMessages(t, sqsClient, queueURL, sqsMessageCount)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 180, 1),
		"replica count should be 1 after 3 minutes")
	testScaleIn(t, kc, sqsClient, queueURL)
	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlDeleteWithTemplate(t, data, "triggerAuthWithExternalIDTemplate", triggerAuthWithExternalIDTemplate)
}

func testScaleFailsWithoutExternalID(t *testing.T, kc *kubernetes.Clientset, data templateData, sqsClient *sqs.Client, queueURL *string) {
	t.Log("--- testing scaling fails without ExternalId ---")
	data.ScaledObjectName = "scale-without-external-id"
	data.TriggerAuthenticationName = "auth-without-external-id"
	KubectlApplyWithTemplate(t, data, "triggerAuthWithoutExternalIDTemplate", triggerAuthWithoutExternalIDTemplate)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	addMessages(t, sqsClient, queueURL, sqsMessageCount)
	// replicas shouldn't change — AssumeRole fails without ExternalId
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
	cleanupMessages(t, sqsClient, queueURL)
	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlDeleteWithTemplate(t, data, "triggerAuthWithoutExternalIDTemplate", triggerAuthWithoutExternalIDTemplate)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, sqsClient *sqs.Client, queueURL *string) {
	t.Log("--- testing scaling in ---")
	cleanupMessages(t, sqsClient, queueURL)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 180, 1),
		"replica count should be 0 after 3 minutes")
}

func cleanupMessages(t *testing.T, sqsClient *sqs.Client, queueURL *string) {
	for {
		response, _ := sqsClient.ReceiveMessage(context.Background(), &sqs.ReceiveMessageInput{
			QueueUrl:            queueURL,
			MaxNumberOfMessages: 10,
		})
		if response == nil || len(response.Messages) == 0 {
			break
		}
		for _, message := range response.Messages {
			_, err := sqsClient.DeleteMessage(context.Background(), &sqs.DeleteMessageInput{
				QueueUrl:      queueURL,
				ReceiptHandle: message.ReceiptHandle,
			})
			assert.NoErrorf(t, err, "cannot delete message - %s", err)
		}
		time.Sleep(time.Second)
	}
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

func createSqsQueue(t *testing.T, queueName string, sqsClient *sqs.Client) *sqs.CreateQueueOutput {
	queue, err := sqsClient.CreateQueue(context.Background(), &sqs.CreateQueueInput{
		QueueName: &queueName,
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
			{Name: "secretTemplate", Config: secretTemplate},
		}
}
