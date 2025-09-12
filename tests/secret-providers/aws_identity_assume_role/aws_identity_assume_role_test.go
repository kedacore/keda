//go:build e2e
// +build e2e

package aws_identity_assume_role_test

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
	testName = "aws-identity-assume-role-test"
)

type templateData struct {
	TestNamespace             string
	DeploymentName            string
	ScaledObjectName          string
	TriggerAuthenticationName string
	SecretName                string
	AwsRegion                 string
	RoleArn                   string
	SqsQueue                  string
}

const (
	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: aws
`

	triggerAuthTemplateWithRoleArn = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: aws
    roleArn: {{.RoleArn}}
`

	triggerAuthTemplateWithIdentityOwner = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: aws
    identityOwner: workload
`

	serviceAccountTemplate = `apiVersion: v1
kind: ServiceAccount
metadata:
  name: workload
  namespace: {{.TestNamespace}}
  annotations:
    eks.amazonaws.com/role-arn: {{.RoleArn}}
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
      serviceAccountName: workload
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
	testNamespace         = fmt.Sprintf("%s-ns", testName)
	deploymentName        = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName      = fmt.Sprintf("%s-so", testName)
	secretName            = fmt.Sprintf("%s-secret", testName)
	sqsWorkload1QueueName = fmt.Sprintf("assume-role-workload1-queue-%d", GetRandomNumber())
	sqsWorkload2QueueName = fmt.Sprintf("assume-role-workload2-queue-%d", GetRandomNumber())
	awsAccessKeyID        = os.Getenv("TF_AWS_ACCESS_KEY")
	awsSecretAccessKey    = os.Getenv("TF_AWS_SECRET_KEY")
	awsRegion             = os.Getenv("TF_AWS_REGION")
	awsWorkload1RoleArn   = os.Getenv("TF_AWS_WORKLOAD1_ROLE")
	awsWorkload2RoleArn   = os.Getenv("TF_AWS_WORKLOAD2_ROLE")
	maxReplicaCount       = 1
	minReplicaCount       = 0
	sqsMessageCount       = 2
)

func TestSqsScaler(t *testing.T) {
	// setup SQS
	sqsClient := createSqsClient()
	queueWorkload1 := createSqsQueue(t, sqsWorkload1QueueName, sqsClient)
	queueWorkload2 := createSqsQueue(t, sqsWorkload2QueueName, sqsClient)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData(*queueWorkload1.QueueUrl)
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling using KEDA identity
	testScaleWithKEDAIdentity(t, kc, data, sqsClient, queueWorkload1.QueueUrl)
	// test scaling using correct identity provided via podIdentity.RoleArn
	// for a role that can be assumed
	testScaleWithExplicitRoleArnUsingRoleAssumtion(t, kc, data, sqsClient, queueWorkload1.QueueUrl)
	// test scaling using correct identity provided via podIdentity.RoleArn
	// for a role to be used with web indentity (workload-2 role allows it)
	testScaleWithExplicitRoleArnUsingWebIdentityRole(t, kc, data, sqsClient, queueWorkload2.QueueUrl)
	// test scaling using correct identity provided via workload
	testScaleWithWorkloadArn(t, kc, data, sqsClient, queueWorkload1.QueueUrl)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
	cleanupQueue(t, sqsClient, queueWorkload1.QueueUrl)
	cleanupQueue(t, sqsClient, queueWorkload2.QueueUrl)
}

// testScaleWithKEDAIdentity checks that we don't scale out because KEDA identity
// doesn't have access to the queue, so even though there are messages, the workload
// won't scale
func testScaleWithKEDAIdentity(t *testing.T, kc *kubernetes.Clientset, data templateData, sqsClient *sqs.Client, queueURL *string) {
	t.Log("--- testing scalig out with KEDA role ---")
	data.ScaledObjectName = "scale-with-keda-identity"
	data.TriggerAuthenticationName = "scale-with-keda-identity"
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlApplyWithTemplate(t, data, "triggerAuthTemplate", triggerAuthenticationTemplate)
	addMessages(t, sqsClient, queueURL, sqsMessageCount)
	// replicas shouldn't change
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
	testScaleIn(t, kc, sqsClient, queueURL)
	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlDeleteWithTemplate(t, data, "triggerAuthTemplate", triggerAuthenticationTemplate)
}

func testScaleWithExplicitRoleArnUsingRoleAssumtion(t *testing.T, kc *kubernetes.Clientset, data templateData, sqsClient *sqs.Client, queueURL *string) {
	t.Log("--- testing scalig out with explicit arn role with role assumption ---")
	data.ScaledObjectName = "scale-using-role-assumtion"
	data.TriggerAuthenticationName = "scale-using-role-assumtion"
	KubectlApplyWithTemplate(t, data, "triggerAuthTemplateWithIdentityID", triggerAuthTemplateWithRoleArn)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	addMessages(t, sqsClient, queueURL, sqsMessageCount)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 180, 1),
		"replica count should be 2 after 3 minutes")
	testScaleIn(t, kc, sqsClient, queueURL)
	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlDeleteWithTemplate(t, data, "triggerAuthTemplate", triggerAuthTemplateWithRoleArn)
}

func testScaleWithExplicitRoleArnUsingWebIdentityRole(t *testing.T, kc *kubernetes.Clientset, data templateData, sqsClient *sqs.Client, queueURL *string) {
	t.Log("--- testing scalig out with explicit arn role with web indentity role ---")
	data.RoleArn = awsWorkload2RoleArn
	data.SqsQueue = *queueURL
	data.ScaledObjectName = "scale-using-web-identity"
	data.TriggerAuthenticationName = "scale-using-web-identity"
	KubectlApplyWithTemplate(t, data, "triggerAuthTemplateWithIdentityID", triggerAuthTemplateWithRoleArn)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	addMessages(t, sqsClient, queueURL, sqsMessageCount)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 180, 1),
		"replica count should be 2 after 3 minutes")
	testScaleIn(t, kc, sqsClient, queueURL)
	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlDeleteWithTemplate(t, data, "triggerAuthTemplate", triggerAuthTemplateWithRoleArn)
}

func testScaleWithWorkloadArn(t *testing.T, kc *kubernetes.Clientset, data templateData, sqsClient *sqs.Client, queueURL *string) {
	t.Log("--- testing scalig out with workload arn role ---")
	data.ScaledObjectName = "scale-using-workload-arn"
	data.TriggerAuthenticationName = "scale-using-workload-arn"
	KubectlApplyWithTemplate(t, data, "triggerAuthTemplateWithIdentityOwner", triggerAuthTemplateWithIdentityOwner)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	addMessages(t, sqsClient, queueURL, sqsMessageCount)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 180, 1),
		"replica count should be 2 after 3 minutes")
	testScaleIn(t, kc, sqsClient, queueURL)
	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlDeleteWithTemplate(t, data, "triggerAuthTemplateWithIdentityOwner", triggerAuthTemplateWithIdentityOwner)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, sqsClient *sqs.Client, queueURL *string) {
	t.Log("--- testing scalig in ---")
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
		if totalDeletedMessages == sqsMessageCount {
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
	cfg, _ := config.LoadDefaultConfig(context.TODO(), configOptions...)
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
			RoleArn:          awsWorkload1RoleArn,
			SqsQueue:         sqsQueue,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "serviceAccountTemplate", Config: serviceAccountTemplate},
		}
}
