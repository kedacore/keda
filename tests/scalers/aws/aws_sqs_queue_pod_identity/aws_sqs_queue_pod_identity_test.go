//go:build e2e
// +build e2e

package aws_sqs_queue_pod_identity_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName = "aws-sqs-queue-pod-identity-test"
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
	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-aws-credentials
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: aws-eks
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
      serviceAccountName: default
      containers:
      - name: nginx
        image: nginxinc/nginx-unprivileged
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
        identityOwner: operator
`
)

var (
	testNamespace      = fmt.Sprintf("%s-ns", testName)
	deploymentName     = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName   = fmt.Sprintf("%s-so", testName)
	secretName         = fmt.Sprintf("%s-secret", testName)
	sqsQueueName       = fmt.Sprintf("queue-identity-%d", GetRandomNumber())
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

func testActivation(t *testing.T, kc *kubernetes.Clientset, sqsClient *sqs.SQS, queueURL *string) {
	t.Log("--- testing activation ---")
	addMessages(t, sqsClient, queueURL, 4)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, sqsClient *sqs.SQS, queueURL *string) {
	t.Log("--- testing scale out ---")
	addMessages(t, sqsClient, queueURL, 6)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 180, 1),
		"replica count should be 2 after 3 minutes")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, sqsClient *sqs.SQS, queueURL *string) {
	t.Log("--- testing scale in ---")
	_, err := sqsClient.PurgeQueueWithContext(context.Background(), &sqs.PurgeQueueInput{
		QueueUrl: queueURL,
	})
	assert.NoErrorf(t, err, "cannot clear queue - %s", err)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 180, 1),
		"replica count should be 0 after 3 minutes")
}

func addMessages(t *testing.T, sqsClient *sqs.SQS, queueURL *string, messages int) {
	for i := 0; i < messages; i++ {
		msg := fmt.Sprintf("Message - %d", i)
		_, err := sqsClient.SendMessageWithContext(context.Background(), &sqs.SendMessageInput{
			QueueUrl:     queueURL,
			MessageBody:  aws.String(msg),
			DelaySeconds: aws.Int64(10),
		})
		assert.NoErrorf(t, err, "cannot send message - %s", err)
	}
}

func createSqsQueue(t *testing.T, sqsClient *sqs.SQS) *sqs.CreateQueueOutput {
	queue, err := sqsClient.CreateQueueWithContext(context.Background(), &sqs.CreateQueueInput{
		QueueName: &sqsQueueName,
		Attributes: map[string]*string{
			"DelaySeconds":           aws.String("60"),
			"MessageRetentionPeriod": aws.String("86400"),
		}})
	assert.NoErrorf(t, err, "failed to create queue - %s", err)
	return queue
}

func cleanupQueue(t *testing.T, sqsClient *sqs.SQS, queueURL *string) {
	t.Log("--- cleaning up ---")
	_, err := sqsClient.DeleteQueueWithContext(context.Background(), &sqs.DeleteQueueInput{
		QueueUrl: queueURL,
	})
	assert.NoErrorf(t, err, "cannot delete queue - %s", err)
}

func createSqsClient() *sqs.SQS {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
	}))

	return sqs.New(sess, &aws.Config{
		Region:      aws.String(awsRegion),
		Credentials: credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, ""),
	})
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
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
