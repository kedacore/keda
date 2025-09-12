//go:build e2e
// +build e2e

package aws_kinesis_stream_pod_identity_eks_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName = "aws-kinesis-stream-pod-identity-eks-test"
)

type templateData struct {
	TestNamespace      string
	DeploymentName     string
	ScaledObjectName   string
	SecretName         string
	AwsAccessKeyID     string
	AwsSecretAccessKey string
	AwsRegion          string
	KinesisStream      string
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
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 15
  triggers:
    - type: aws-kinesis-stream
      authenticationRef:
        name: keda-trigger-auth-aws-credentials
      metadata:
        awsRegion: {{.AwsRegion}}
        streamName: {{.KinesisStream}}
        shardCount: "3"
        activationShardCount: "4"
        identityOwner: operator
`
)

var (
	testNamespace      = fmt.Sprintf("%s-ns", testName)
	deploymentName     = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName   = fmt.Sprintf("%s-so", testName)
	secretName         = fmt.Sprintf("%s-secret", testName)
	kinesisStreamName  = fmt.Sprintf("kinesis-identity-%d", GetRandomNumber())
	awsAccessKeyID     = os.Getenv("TF_AWS_ACCESS_KEY")
	awsSecretAccessKey = os.Getenv("TF_AWS_SECRET_KEY")
	awsRegion          = os.Getenv("TF_AWS_REGION")
	maxReplicaCount    = 2
	minReplicaCount    = 0
)

func TestKiensisScaler(t *testing.T) {
	// setup kinesis
	kinesisClient := createKinesisClient()
	createKinesisStream(t, kinesisClient)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 1),
		"replica count should be %d after 1 minute", minReplicaCount)

	// test scaling
	testActivation(t, kc, kinesisClient)
	testScaleOut(t, kc, kinesisClient)
	testScaleIn(t, kc, kinesisClient)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
	cleanupStream(t, kinesisClient)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, kinesisClient *kinesis.Client) {
	t.Log("--- testing activation ---")
	updateShardCount(t, kinesisClient, 3)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, kinesisClient *kinesis.Client) {
	t.Log("--- testing scale out ---")
	updateShardCount(t, kinesisClient, 6)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, kinesisClient *kinesis.Client) {
	t.Log("--- testing scale in ---")
	updateShardCount(t, kinesisClient, 3)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

func updateShardCount(t *testing.T, kinesisClient *kinesis.Client, shardCount int64) {
	done := waitForStreamActiveStatus(t, kinesisClient)
	if done {
		_, err := kinesisClient.UpdateShardCount(context.Background(), &kinesis.UpdateShardCountInput{
			StreamName:       &kinesisStreamName,
			TargetShardCount: aws.Int32(int32(shardCount)),
			ScalingType:      types.ScalingTypeUniformScaling,
		})
		assert.NoErrorf(t, err, "cannot update shard count - %s", err)
	}
	assert.True(t, true, "failed to update shard count")
}

func createKinesisStream(t *testing.T, kinesisClient *kinesis.Client) {
	_, err := kinesisClient.CreateStream(context.Background(), &kinesis.CreateStreamInput{
		StreamName: &kinesisStreamName,
		ShardCount: aws.Int32(2),
	})
	assert.NoErrorf(t, err, "failed to create stream - %s", err)
	done := waitForStreamActiveStatus(t, kinesisClient)
	if !done {
		assert.True(t, true, "failed to create kinesis")
	}
}

func waitForStreamActiveStatus(t *testing.T, kinesisClient *kinesis.Client) bool {
	for i := 0; i < 30; i++ {
		describe, _ := kinesisClient.DescribeStream(context.Background(), &kinesis.DescribeStreamInput{
			StreamName: &kinesisStreamName,
		})
		t.Logf("Waiting for stream ACTIVE status. current status - %s", describe.StreamDescription.StreamStatus)
		if describe.StreamDescription.StreamStatus == "ACTIVE" {
			return true
		}
		time.Sleep(time.Second * 2)
	}
	return false
}

func cleanupStream(t *testing.T, kinesisClient *kinesis.Client) {
	t.Log("--- cleaning up ---")
	_, err := kinesisClient.DeleteStream(context.Background(), &kinesis.DeleteStreamInput{
		StreamName: &kinesisStreamName,
	})
	assert.NoErrorf(t, err, "cannot delete stream - %s", err)
}

func createKinesisClient() *kinesis.Client {
	configOptions := make([]func(*config.LoadOptions) error, 0)
	configOptions = append(configOptions, config.WithRegion(awsRegion))
	cfg, _ := config.LoadDefaultConfig(context.TODO(), configOptions...)
	cfg.Credentials = credentials.NewStaticCredentialsProvider(awsAccessKeyID, awsSecretAccessKey, "")
	return kinesis.NewFromConfig(cfg)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:      testNamespace,
			DeploymentName:     deploymentName,
			ScaledObjectName:   scaledObjectName,
			SecretName:         secretName,
			AwsAccessKeyID:     base64.StdEncoding.EncodeToString([]byte(awsAccessKeyID)),
			AwsSecretAccessKey: base64.StdEncoding.EncodeToString([]byte(awsSecretAccessKey)),
			AwsRegion:          awsRegion,
			KinesisStream:      kinesisStreamName,
		}, []Template{
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
