//go:build e2e
// +build e2e

package aws_dynamodb_pod_identity_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../../.env")

const (
	testName = "aws-dynamodb-pod-identity-test"
)

type templateData struct {
	TestNamespace             string
	DeploymentName            string
	ScaledObjectName          string
	SecretName                string
	AwsAccessKeyID            string
	AwsSecretAccessKey        string
	AwsRegion                 string
	DynamoDBTableName         string
	ExpressionAttributeNames  string
	KeyConditionExpression    string
	ExpressionAttributeValues string
}

const (
	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-aws-credentials
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: aws
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
    - type: aws-dynamodb
      authenticationRef:
        name: keda-trigger-auth-aws-credentials
      metadata:
        awsRegion: {{.AwsRegion}}
        tableName: {{.DynamoDBTableName}}
        expressionAttributeNames: '{{.ExpressionAttributeNames}}'
        keyConditionExpression: '{{.KeyConditionExpression}}'
        expressionAttributeValues: '{{.ExpressionAttributeValues}}'
        targetValue: '1'
        activationTargetValue: '4'
        identityOwner: operator
`
)

var (
	testNamespace             = fmt.Sprintf("%s-ns", testName)
	deploymentName            = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName          = fmt.Sprintf("%s-so", testName)
	secretName                = fmt.Sprintf("%s-secret", testName)
	dynamoDBTableName         = fmt.Sprintf("table-identity-%d", GetRandomNumber())
	awsAccessKeyID            = os.Getenv("TF_AWS_ACCESS_KEY")
	awsSecretAccessKey        = os.Getenv("TF_AWS_SECRET_KEY")
	awsRegion                 = os.Getenv("TF_AWS_REGION")
	expressionAttributeNames  = "{ \"#k\" : \"event_type\"}"
	keyConditionExpression    = "#k = :key"
	expressionAttributeValues = "{ \":key\" : {\"S\":\"scaling_event\"}}"
	maxReplicaCount           = 2
	minReplicaCount           = 0
)

func TestDynamoDBScaler(t *testing.T) {
	// setup dynamodb
	dynamodbClient := createDynamoDBClient()
	createDynamoDBTable(t, dynamodbClient)

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 1),
		"replica count should be %d after 1 minute", minReplicaCount)

	// test scaling
	testActivation(t, kc, dynamodbClient)
	testScaleOut(t, kc, dynamodbClient)
	testScaleIn(t, kc, dynamodbClient)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
	cleanupTable(t, dynamodbClient)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, dynamodbClient *dynamodb.Client) {
	t.Log("--- testing activation ---")
	addMessages(t, dynamodbClient, 3)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, dynamodbClient *dynamodb.Client) {
	t.Log("--- testing scale out ---")
	addMessages(t, dynamodbClient, 6)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, dynamodbClient *dynamodb.Client) {
	t.Log("--- testing scale in ---")

	for i := 0; i < 6; i++ {
		_, err := dynamodbClient.DeleteItem(context.Background(), &dynamodb.DeleteItemInput{
			TableName: aws.String(dynamoDBTableName),
			Key: map[string]types.AttributeValue{
				"event_type": &types.AttributeValueMemberS{
					Value: "scaling_event",
				},
				"event_id": &types.AttributeValueMemberS{
					Value: strconv.Itoa(i),
				},
			},
		})
		assert.NoErrorf(t, err, "failed to delete item - %s", err)
	}

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

func addMessages(t *testing.T, dynamodbClient *dynamodb.Client, messages int) {
	for i := 0; i < messages; i++ {
		_, err := dynamodbClient.PutItem(context.Background(), &dynamodb.PutItemInput{
			TableName: aws.String(dynamoDBTableName),
			Item: map[string]types.AttributeValue{
				"event_type": &types.AttributeValueMemberS{
					Value: "scaling_event",
				},
				"event_id": &types.AttributeValueMemberS{
					Value: strconv.Itoa(i),
				},
			},
		})
		t.Log("Message enqueued")
		assert.NoErrorf(t, err, "failed to create item - %s", err)
	}
}

func createDynamoDBTable(t *testing.T, dynamodbClient *dynamodb.Client) {
	_, err := dynamodbClient.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		TableName: aws.String(dynamoDBTableName),
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("event_type"), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String("event_id"), KeyType: types.KeyTypeRange},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("event_type"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("event_id"), AttributeType: types.ScalarAttributeTypeS},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
	})
	assert.NoErrorf(t, err, "failed to create table - %s", err)
	done := waitForTableActiveStatus(t, dynamodbClient)
	if !done {
		assert.True(t, true, "failed to create dynamodb")
	}
}

func waitForTableActiveStatus(t *testing.T, dynamodbClient *dynamodb.Client) bool {
	for i := 0; i < 30; i++ {
		describe, _ := dynamodbClient.DescribeTable(context.Background(), &dynamodb.DescribeTableInput{
			TableName: aws.String(dynamoDBTableName),
		})
		t.Logf("Waiting for table ACTIVE status. current status - %s", describe.Table.TableStatus)
		if describe.Table.TableStatus == "ACTIVE" {
			return true
		}
		time.Sleep(time.Second * 2)
	}
	return false
}

func cleanupTable(t *testing.T, dynamodbClient *dynamodb.Client) {
	t.Log("--- cleaning up ---")
	_, err := dynamodbClient.DeleteTable(context.Background(), &dynamodb.DeleteTableInput{
		TableName: aws.String(dynamoDBTableName),
	})
	assert.NoErrorf(t, err, "cannot delete stream - %s", err)
}

func createDynamoDBClient() *dynamodb.Client {
	configOptions := make([]func(*config.LoadOptions) error, 0)
	configOptions = append(configOptions, config.WithRegion(awsRegion))
	cfg, _ := config.LoadDefaultConfig(context.TODO(), configOptions...)
	cfg.Credentials = credentials.NewStaticCredentialsProvider(awsAccessKeyID, awsSecretAccessKey, "")
	return dynamodb.NewFromConfig(cfg)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:             testNamespace,
			DeploymentName:            deploymentName,
			ScaledObjectName:          scaledObjectName,
			SecretName:                secretName,
			AwsAccessKeyID:            base64.StdEncoding.EncodeToString([]byte(awsAccessKeyID)),
			AwsSecretAccessKey:        base64.StdEncoding.EncodeToString([]byte(awsSecretAccessKey)),
			AwsRegion:                 awsRegion,
			DynamoDBTableName:         dynamoDBTableName,
			ExpressionAttributeNames:  expressionAttributeNames,
			KeyConditionExpression:    keyConditionExpression,
			ExpressionAttributeValues: expressionAttributeValues,
		}, []Template{
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
