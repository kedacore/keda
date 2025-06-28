//go:build e2e
// +build e2e

package aws_sqs_queue_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"

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
	testName = "aws-sqs-queue-test"
)

type templateData struct {
	TestNamespace      string
	DeploymentName     string
	ScaledObjectName   string
	SecretName         string
	AwsAccessKeyID     string
	AwsSecretAccessKey string
	AwsRegion          string
	SqsQueue           string
}

const (
	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  AWS_ACCESS_KEY_ID: {{.AwsAccessKeyID}}
  AWS_SECRET_ACCESS_KEY: {{.AwsSecretAccessKey}}
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-aws-credentials
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: awsAccessKeyID     # Required.
    name: {{.SecretName}}         # Required.
    key: AWS_ACCESS_KEY_ID        # Required.
  - parameter: awsSecretAccessKey # Required.
    name: {{.SecretName}}         # Required.
    key: AWS_SECRET_ACCESS_KEY    # Required.
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
  maxReplicaCount: 2
  minReplicaCount: 0
  cooldownPeriod: 1
  triggers:
    - type: aws-sqs-queue
      authenticationRef:
        name: keda-trigger-auth-aws-credentials
      metadata:
        awsRegion: {{.AwsRegion}}
        queueURL: {{.SqsQueue}}
        queueLength: "1"
        activationQueueLength: "5"
`
)

var (
	testNamespace      = fmt.Sprintf("%s-ns", testName)
	deploymentName     = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName   = fmt.Sprintf("%s-so", testName)
	secretName         = fmt.Sprintf("%s-secret", testName)
	sqsQueueName       = fmt.Sprintf("queue-%d", GetRandomNumber())
	awsAccessKeyID     = os.Getenv("TF_AWS_ACCESS_KEY")
	awsSecretAccessKey = os.Getenv("TF_AWS_SECRET_KEY")
	awsRegion          = os.Getenv("TF_AWS_REGION")
	maxReplicaCount    = 2
	minReplicaCount    = 0
)

func TestSqsScaler(t *testing.T) {
	// setup SQS
	sqsClient := createSqsClient()
	queue := createSqsQueue(t, sqsClient)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData(*queue.QueueUrl)
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testActivation(t, kc, sqsClient, queue.QueueUrl)
	testScaleOut(t, kc, sqsClient, queue.QueueUrl)
	testScaleIn(t, kc, sqsClient, queue.QueueUrl)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
	cleanupQueue(t, sqsClient, queue.QueueUrl)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, sqsClient *sqs.Client, queueURL *string) {
	t.Log("--- testing activation ---")
	addMessages(t, sqsClient, queueURL, 4)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, sqsClient *sqs.Client, queueURL *string) {
	t.Log("--- testing scale out ---")
	addMessages(t, sqsClient, queueURL, 6)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 180, 1),
		"replica count should be 2 after 3 minutes")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, sqsClient *sqs.Client, queueURL *string) {
	t.Log("--- testing scale in ---")
	_, err := sqsClient.PurgeQueue(context.Background(), &sqs.PurgeQueueInput{
		QueueUrl: queueURL,
	})
	assert.NoErrorf(t, err, "cannot clear queue - %s", err)

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
	cfg, _ := config.LoadDefaultConfig(context.TODO(), configOptions...)
	cfg.Credentials = credentials.NewStaticCredentialsProvider(awsAccessKeyID, awsSecretAccessKey, "")
	return sqs.NewFromConfig(cfg)
}

func getTemplateData(sqsQueue string) (templateData, []Template) {
	return templateData{
			TestNamespace:      testNamespace,
			DeploymentName:     deploymentName,
			ScaledObjectName:   scaledObjectName,
			SecretName:         secretName,
			AwsAccessKeyID:     base64.StdEncoding.EncodeToString([]byte(awsAccessKeyID)),
			AwsSecretAccessKey: base64.StdEncoding.EncodeToString([]byte(awsSecretAccessKey)),
			AwsRegion:          awsRegion,
			SqsQueue:           sqsQueue,
		}, []Template{
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
