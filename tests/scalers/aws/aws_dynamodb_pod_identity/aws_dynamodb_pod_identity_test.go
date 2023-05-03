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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
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

func testActivation(t *testing.T, kc *kubernetes.Clientset, dynamodbClient *dynamodb.DynamoDB) {
	t.Log("--- testing activation ---")
	addMessages(t, dynamodbClient, 3)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, dynamodbClient *dynamodb.DynamoDB) {
	t.Log("--- testing scale out ---")
	addMessages(t, dynamodbClient, 6)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, dynamodbClient *dynamodb.DynamoDB) {
	t.Log("--- testing scale in ---")

	for i := 0; i < 6; i++ {
		_, err := dynamodbClient.DeleteItemWithContext(context.Background(), &dynamodb.DeleteItemInput{
			TableName: aws.String(dynamoDBTableName),
			Key: map[string]*dynamodb.AttributeValue{
				"event_type": {S: aws.String("scaling_event")},
				"event_id":   {S: aws.String(strconv.Itoa(i))},
			},
		})
		assert.NoErrorf(t, err, "failed to delete item - %s", err)
	}

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

func addMessages(t *testing.T, dynamodbClient *dynamodb.DynamoDB, messages int) {
	for i := 0; i < messages; i++ {
		_, err := dynamodbClient.PutItemWithContext(context.Background(), &dynamodb.PutItemInput{
			TableName: aws.String(dynamoDBTableName),
			Item: map[string]*dynamodb.AttributeValue{
				"event_type": {S: aws.String("scaling_event")},
				"event_id":   {S: aws.String(strconv.Itoa(i))},
			},
		})
		t.Log("Message enqueued")
		assert.NoErrorf(t, err, "failed to create item - %s", err)
	}
}

func createDynamoDBTable(t *testing.T, dynamodbClient *dynamodb.DynamoDB) {
	_, err := dynamodbClient.CreateTableWithContext(context.Background(), &dynamodb.CreateTableInput{
		TableName: aws.String(dynamoDBTableName),
		KeySchema: []*dynamodb.KeySchemaElement{
			{AttributeName: aws.String("event_type"), KeyType: aws.String("HASH")},
			{AttributeName: aws.String("event_id"), KeyType: aws.String("RANGE")},
		},
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{AttributeName: aws.String("event_type"), AttributeType: aws.String("S")},
			{AttributeName: aws.String("event_id"), AttributeType: aws.String("S")},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
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

func waitForTableActiveStatus(t *testing.T, dynamodbClient *dynamodb.DynamoDB) bool {
	for i := 0; i < 30; i++ {
		describe, _ := dynamodbClient.DescribeTableWithContext(context.Background(), &dynamodb.DescribeTableInput{
			TableName: aws.String(dynamoDBTableName),
		})
		t.Logf("Waiting for table ACTIVE status. current status - %s", *describe.Table.TableStatus)
		if *describe.Table.TableStatus == "ACTIVE" {
			return true
		}
		time.Sleep(time.Second * 2)
	}
	return false
}

func cleanupTable(t *testing.T, dynamodbClient *dynamodb.DynamoDB) {
	t.Log("--- cleaning up ---")
	_, err := dynamodbClient.DeleteTableWithContext(context.Background(), &dynamodb.DeleteTableInput{
		TableName: aws.String(dynamoDBTableName),
	})
	assert.NoErrorf(t, err, "cannot delete stream - %s", err)
}

func createDynamoDBClient() *dynamodb.DynamoDB {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
	}))

	return dynamodb.New(sess, &aws.Config{
		Region:      aws.String(awsRegion),
		Credentials: credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, ""),
	})
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
