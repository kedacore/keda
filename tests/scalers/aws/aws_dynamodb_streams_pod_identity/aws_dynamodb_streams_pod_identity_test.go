//go:build e2e
// +build e2e

package aws_dynamodb_streams_pod_identity_test

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/dynamodbstreams"
	"github.com/aws/aws-sdk-go/service/dynamodbstreams/dynamodbstreamsiface"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName = "aws-dynamodb-streams-pod-identity-test"
)

var (
	awsRegion            = os.Getenv("TF_AWS_REGION")
	awsAccessKey         = os.Getenv("TF_AWS_ACCESS_KEY")
	awsSecretKey         = os.Getenv("TF_AWS_SECRET_KEY")
	testNamespace        = fmt.Sprintf("%s-ns", testName)
	secretName           = fmt.Sprintf("%s-secret", testName)
	deploymentName       = fmt.Sprintf("%s-deployment", testName)
	triggerAuthName      = fmt.Sprintf("%s-ta", testName)
	scaledObjectName     = fmt.Sprintf("%s-so", testName)
	tableName            = fmt.Sprintf("stream-identity-%d", GetRandomNumber())
	shardCount           = 2 // default count
	activationShardCount = 0 // default count
)

type templateData struct {
	TestNamespace        string
	SecretName           string
	AwsRegion            string
	AwsAccessKey         string
	AwsSecretKey         string
	DeploymentName       string
	TriggerAuthName      string
	ScaledObjectName     string
	TableName            string
	ShardCount           int64
	ActivationShardCount int64
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
        image: nginxinc/nginx-unprivileged
`

	triggerAuthTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: aws-eks
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
  maxReplicaCount: 2
  minReplicaCount: 0
  pollingInterval: 5  # Optional. Default: 30 seconds
  cooldownPeriod:  1  # Optional. Default: 300 seconds
  triggers:
  - type: aws-dynamodb-streams
    authenticationRef:
      name: {{.TriggerAuthName}}
    metadata:
      awsRegion: {{.AwsRegion}}     # Required
      tableName: {{.TableName}}     # Required
      shardCount: "{{.ShardCount}}" # Optional. Default: 2
      activationShardCount: "{{.ActivationShardCount}}" # Optional. Default: 0
      identityOwner: operator
`
)

func TestScaler(t *testing.T) {
	t.Log("--- setting up ---")
	require.NotEmpty(t, awsAccessKey, "AWS_ACCESS_KEY env variable is required for dynamodb streams tests")
	require.NotEmpty(t, awsSecretKey, "AWS_SECRET_KEY env variable is required for dynamodb streams tests")
	data, templates := getTemplateData()

	// Create DynamoDB table and the latest stream Arn for the table
	dbClient, dbStreamsClient := setupDynamoDBStreams(t)
	streamArn, err := getLatestStreamArn(dbClient)
	assert.NoErrorf(t, err, "cannot get latest stream arn for the table - %s", err)
	time.Sleep(10 * time.Second)

	// Get Shard Count
	shardCount, err := getDynamoDBStreamShardCount(dbStreamsClient, streamArn)
	assert.True(t, shardCount >= 2, "dynamodb stream shard count should be 2 or higher - %s", err)

	// Deploy nginx, secret, and triggerAuth
	kc := GetKubernetesClient(t)
	CreateNamespace(t, kc, testNamespace)
	KubectlApplyWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)
	KubectlApplyWithTemplate(t, data, "triggerAuthTemplate", triggerAuthTemplate)

	// Wait for nginx to load
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 30, 3),
		"replica count should start out as 0")

	// test scaling
	testActivation(t, kc, data)
	testScaleOut(t, kc, data, shardCount)
	testScaleIn(t, kc, data, shardCount)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
	cleanupDynamoDBTable(t, dbClient)
}

func setupDynamoDBStreams(t *testing.T) (*dynamodb.DynamoDB, *dynamodbstreams.DynamoDBStreams) {
	sess := session.Must(session.NewSession())

	var dbClient *dynamodb.DynamoDB
	var dbStreamClient *dynamodbstreams.DynamoDBStreams

	creds := credentials.NewStaticCredentials(awsAccessKey, awsSecretKey, "")
	cfg := aws.NewConfig().WithRegion(awsRegion).WithCredentials(creds)
	dbClient = dynamodb.New(sess, cfg)
	dbStreamClient = dynamodbstreams.New(sess, cfg)

	err := createTable(dbClient)
	assert.NoErrorf(t, err, "cannot create dynamodb table - %s", err)

	return dbClient, dbStreamClient
}

func createTable(db dynamodbiface.DynamoDBAPI) error {
	keySchema := []*dynamodb.KeySchemaElement{
		{
			AttributeName: aws.String("id"),
			KeyType:       aws.String("HASH"),
		},
	}
	attributeDefinitions := []*dynamodb.AttributeDefinition{
		{
			AttributeName: aws.String("id"),
			AttributeType: aws.String("S"),
		},
	}
	streamSpecification := &dynamodb.StreamSpecification{
		StreamEnabled:  aws.Bool(true),
		StreamViewType: aws.String("NEW_IMAGE"),
	}
	_, err := db.CreateTableWithContext(context.Background(), &dynamodb.CreateTableInput{
		TableName:            &tableName,
		KeySchema:            keySchema,
		AttributeDefinitions: attributeDefinitions,
		BillingMode:          aws.String("PAY_PER_REQUEST"),
		StreamSpecification:  streamSpecification,
	})
	return err
}

func getLatestStreamArn(db dynamodbiface.DynamoDBAPI) (*string, error) {
	input := dynamodb.DescribeTableInput{
		TableName: &tableName,
	}
	tableInfo, err := db.DescribeTableWithContext(context.Background(), &input)
	if err != nil {
		return nil, err
	}
	if nil == tableInfo.Table.LatestStreamArn {
		return nil, errors.New("empty table stream arn")
	}
	return tableInfo.Table.LatestStreamArn, nil
}

func getDynamoDBStreamShardCount(dbs dynamodbstreamsiface.DynamoDBStreamsAPI, streamArn *string) (int64, error) {
	input := dynamodbstreams.DescribeStreamInput{
		StreamArn: streamArn,
	}
	des, err := dbs.DescribeStreamWithContext(context.Background(), &input)
	if err != nil {
		return -1, err
	}
	return int64(len(des.StreamDescription.Shards)), nil
}

func getTemplateData() (templateData, []Template) {
	base64AwsAccessKey := base64.StdEncoding.EncodeToString([]byte(awsAccessKey))
	base64AwsSecretKey := base64.StdEncoding.EncodeToString([]byte(awsSecretKey))

	return templateData{
			TestNamespace:    testNamespace,
			SecretName:       secretName,
			AwsRegion:        awsRegion,
			AwsAccessKey:     base64AwsAccessKey,
			AwsSecretKey:     base64AwsSecretKey,
			DeploymentName:   deploymentName,
			TriggerAuthName:  triggerAuthName,
			ScaledObjectName: scaledObjectName,
			TableName:        tableName,
			ShardCount:       int64(shardCount),
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	data.ActivationShardCount = 10
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData, shardCount int64) {
	t.Log("--- testing scale out ---")
	// Deploy scalerObject with its target shardCount = the current dynamodb streams shard count and check if replicas scale out to 1
	t.Log("replicas should scale out to 1")
	data.ShardCount = shardCount
	data.ActivationShardCount = int64(activationShardCount)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 180, 1),
		"replica count should increase to 1")

	// Deploy scalerObject with its shardCount = 1 and check if replicas scale out to 2 (maxReplicaCount)
	t.Log("then, replicas should scale out to 2")
	data.ShardCount = 1
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 180, 1),
		"replica count should increase to 2")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData, shardCount int64) {
	t.Log("--- testing scale in ---")
	// Deploy scalerObject with its target shardCount = the current dynamodb streams shard count and check if replicas scale in to 1
	data.ShardCount = shardCount
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 330, 1),
		"replica count should decrease to 1 in 330 seconds")
}

func cleanupDynamoDBTable(t *testing.T, db dynamodbiface.DynamoDBAPI) {
	t.Log("--- cleaning up ---")
	_, err := db.DeleteTableWithContext(context.Background(),
		&dynamodb.DeleteTableInput{
			TableName: &tableName,
		})
	assert.NoErrorf(t, err, "cannot delete dynamodb table - %s", err)
}
