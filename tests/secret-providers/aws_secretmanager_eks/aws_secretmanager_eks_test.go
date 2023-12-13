//go:build e2e
// +build e2e

package aws_secretmanager_eks_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "aws-secret-manage-pod-identity-test"
)

var (
	testNamespace              = fmt.Sprintf("%s-ns", testName)
	deploymentName             = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName           = fmt.Sprintf("%s-so", testName)
	triggerAuthenticationName  = fmt.Sprintf("%s-ta", testName)
	secretName                 = fmt.Sprintf("%s-secret", testName)
	postgreSQLStatefulSetName  = "postgresql"
	postgresqlPodName          = fmt.Sprintf("%s-0", postgreSQLStatefulSetName)
	postgreSQLUsername         = "test-user"
	postgreSQLPassword         = "test-password"
	postgreSQLDatabase         = "test_db"
	postgreSQLConnectionString = fmt.Sprintf("postgresql://%s:%s@postgresql.%s.svc.cluster.local:5432/%s?sslmode=disable",
		postgreSQLUsername, postgreSQLPassword, testNamespace, postgreSQLDatabase)
	minReplicaCount = 0
	maxReplicaCount = 2

	awsRegion                = os.Getenv("TF_AWS_REGION")
	awsAccessKeyID           = os.Getenv("TF_AWS_ACCESS_KEY")
	awsSecretAccessKey       = os.Getenv("TF_AWS_SECRET_KEY")
	awsCredentialsSecretName = fmt.Sprintf("%s-credentials-secret", testName)
	secretManagerSecretName  = fmt.Sprintf("connectionString-%d", GetRandomNumber())
)

type templateData struct {
	TestNamespace                    string
	DeploymentName                   string
	ScaledObjectName                 string
	TriggerAuthenticationName        string
	SecretName                       string
	PostgreSQLStatefulSetName        string
	PostgreSQLConnectionStringBase64 string
	PostgreSQLUsername               string
	PostgreSQLPassword               string
	PostgreSQLDatabase               string
	MinReplicaCount                  int
	MaxReplicaCount                  int
	AwsRegion                        string
	AwsCredentialsSecretName         string
	SecretManagerSecretName          string
	AwsAccessKeyID                   string
	AwsSecretAccessKey               string
}

const (
	deploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: postgresql-update-worker
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: postgresql-update-worker
  template:
    metadata:
      labels:
        app: postgresql-update-worker
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-postgresql
        imagePullPolicy: Always
        name: postgresql-processor-test
        command:
          - /app
          - update
        env:
          - name: TASK_INSTANCES_COUNT
            value: "6000"
          - name: CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: postgresql_conn_str
`

	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
data:
  postgresql_conn_str: {{.PostgreSQLConnectionStringBase64}}
`
	awsCredentialsSecretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.AwsCredentialsSecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
data:
  AWS_ACCESS_KEY_ID: {{.AwsAccessKeyID}}
  AWS_SECRET_ACCESS_KEY: {{.AwsSecretAccessKey}}
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  awsSecretManager:
 	podIdentity:
  	  provider: aws-eks
    secrets:
    - parameter: connection
      name: {{.SecretManagerSecretName}}
`

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  cooldownPeriod:  10
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  triggers:
  - type: postgresql
    metadata:
      targetQueryValue: "4"
      activationTargetQueryValue: "5"
      query: "SELECT CEIL(COUNT(*) / 5) FROM task_instance WHERE state='running' OR state='queued'"
    authenticationRef:
      name: {{.TriggerAuthenticationName}}
`

	postgreSQLStatefulSetTemplate = `apiVersion: apps/v1
kind: StatefulSet
metadata:
  labels:
    app: {{.PostgreSQLStatefulSetName}}
  name: {{.PostgreSQLStatefulSetName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 1
  serviceName: {{.PostgreSQLStatefulSetName}}
  selector:
    matchLabels:
      app: {{.PostgreSQLStatefulSetName}}
  template:
    metadata:
      labels:
        app: {{.PostgreSQLStatefulSetName}}
    spec:
      containers:
      - image: postgres:10.5
        name: postgresql
        env:
          - name: POSTGRES_USER
            value: {{.PostgreSQLUsername}}
          - name: POSTGRES_PASSWORD
            value: {{.PostgreSQLPassword}}
          - name: POSTGRES_DB
            value: {{.PostgreSQLDatabase}}
        ports:
          - name: postgresql
            protocol: TCP
            containerPort: 5432
`

	postgreSQLServiceTemplate = `apiVersion: v1
kind: Service
metadata:
  labels:
    app: {{.PostgreSQLStatefulSetName}}
  name: {{.PostgreSQLStatefulSetName}}
  namespace: {{.TestNamespace}}
spec:
  ports:
  - port: 5432
    protocol: TCP
    targetPort: 5432
  selector:
    app: {{.PostgreSQLStatefulSetName}}
  type: ClusterIP
`
	lowLevelRecordsJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  labels:
    app: postgresql-insert-low-level-job
  name: postgresql-insert-low-level-job
  namespace: {{.TestNamespace}}
spec:
  template:
    metadata:
      labels:
        app: postgresql-insert-low-level-job
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-postgresql
        imagePullPolicy: Always
        name: postgresql-processor-test
        command:
          - /app
          - insert
        env:
          - name: TASK_INSTANCES_COUNT
            value: "20"
          - name: CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: postgresql_conn_str
      restartPolicy: Never
  backoffLimit: 4
`
	insertRecordsJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  labels:
    app: postgresql-insert-job
  name: postgresql-insert-job
  namespace: {{.TestNamespace}}
spec:
  template:
    metadata:
      labels:
        app: postgresql-insert-job
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-postgresql
        imagePullPolicy: Always
        name: postgresql-processor-test
        command:
          - /app
          - insert
        env:
          - name: TASK_INSTANCES_COUNT
            value: "1000"
          - name: CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: postgresql_conn_str
      restartPolicy: Never
  backoffLimit: 4
`
)

func TestAwsSecretManager(t *testing.T) {
	require.NotEmpty(t, awsAccessKeyID, "TF_AWS_ACCESS_KEY env variable is required for AWS Secret Manager test")
	require.NotEmpty(t, awsSecretAccessKey, "TF_AWS_SECRET_KEY env variable is required for AWS Secret Manager test")

	// Create the secret in GCP
	err := createAWSSecret(t)
	assert.NoErrorf(t, err, "cannot create AWS Secret Manager secret - %s", err)

	// Create kubernetes resources for PostgreSQL server
	kc := GetKubernetesClient(t)
	data, postgreSQLtemplates := getPostgreSQLTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, postgreSQLtemplates)

	assert.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, postgreSQLStatefulSetName, testNamespace, 1, 60, 3),
		"replica count should be %d after 3 minutes", 1)

	createTableSQL := "CREATE TABLE task_instance (id serial PRIMARY KEY,state VARCHAR(10));"
	psqlCreateTableCmd := fmt.Sprintf("psql -U %s -d %s -c \"%s\"", postgreSQLUsername, postgreSQLDatabase, createTableSQL)

	ok, out, errOut, err := WaitForSuccessfulExecCommandOnSpecificPod(t, postgresqlPodName, testNamespace, psqlCreateTableCmd, 60, 3)
	assert.True(t, ok, "executing a command on PostreSQL Pod should work; Output: %s, ErrorOutput: %s, Error: %s", out, errOut, err)

	// Create kubernetes resources for testing
	data, templates := getTemplateData()

	KubectlApplyMultipleWithTemplate(t, data, templates)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc)

	// cleanup
	KubectlDeleteMultipleWithTemplate(t, data, templates)
	DeleteKubernetesResources(t, testNamespace, data, postgreSQLtemplates)

	// Delete the secret in GCP
	err = deleteAWSSecret(t)
	assert.NoErrorf(t, err, "cannot delete AWS Secret Manager secret - %s", err)
}

var data = templateData{
	TestNamespace:                    testNamespace,
	PostgreSQLStatefulSetName:        postgreSQLStatefulSetName,
	DeploymentName:                   deploymentName,
	ScaledObjectName:                 scaledObjectName,
	MinReplicaCount:                  minReplicaCount,
	MaxReplicaCount:                  maxReplicaCount,
	TriggerAuthenticationName:        triggerAuthenticationName,
	SecretName:                       secretName,
	PostgreSQLUsername:               postgreSQLUsername,
	PostgreSQLPassword:               postgreSQLPassword,
	PostgreSQLDatabase:               postgreSQLDatabase,
	PostgreSQLConnectionStringBase64: base64.StdEncoding.EncodeToString([]byte(postgreSQLConnectionString)),
	SecretManagerSecretName:          secretManagerSecretName,
	AwsAccessKeyID:                   base64.StdEncoding.EncodeToString([]byte(awsAccessKeyID)),
	AwsSecretAccessKey:               base64.StdEncoding.EncodeToString([]byte(awsSecretAccessKey)),
	AwsRegion:                        awsRegion,
	AwsCredentialsSecretName:         awsCredentialsSecretName,
}

func getPostgreSQLTemplateData() (templateData, []Template) {
	return data, []Template{
		{Name: "postgreSQLStatefulSetTemplate", Config: postgreSQLStatefulSetTemplate},
		{Name: "postgreSQLServiceTemplate", Config: postgreSQLServiceTemplate},
	}
}

func getTemplateData() (templateData, []Template) {
	return data, []Template{
		{Name: "secretTemplate", Config: secretTemplate},
		{Name: "awsCredentialsSecretTemplate", Config: awsCredentialsSecretTemplate},
		{Name: "deploymentTemplate", Config: deploymentTemplate},
		{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
		{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
	}
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	KubectlApplyWithTemplate(t, data, "lowLevelRecordsJobTemplate", lowLevelRecordsJobTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlApplyWithTemplate(t, data, "insertRecordsJobTemplate", insertRecordsJobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

func createAWSSecret(t *testing.T) error {
	ctx := context.Background()

	// Create AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(awsRegion), config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
		Value: aws.Credentials{
			AccessKeyID:     awsAccessKeyID,
			SecretAccessKey: awsSecretAccessKey,
		},
	}))
	if err != nil {
		return fmt.Errorf("failed to create AWS configuration: %w", err)
	}

	// Create a Secrets Manager client
	client := secretsmanager.NewFromConfig(cfg)

	// Create the secret value
	secretString := postgreSQLConnectionString
	_, err = client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         &secretManagerSecretName,
		SecretString: &secretString,
	})
	if err != nil {
		return fmt.Errorf("failed to create AWS Secret Manager secret: %w", err)
	}

	t.Log("Created secret in AWS Secret Manager.")

	return nil
}

func deleteAWSSecret(t *testing.T) error {
	ctx := context.Background()

	// Create AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(awsRegion), config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
		Value: aws.Credentials{
			AccessKeyID:     awsAccessKeyID,
			SecretAccessKey: awsSecretAccessKey,
		},
	}))
	if err != nil {
		return fmt.Errorf("failed to create AWS configuration: %w", err)
	}

	// Create a Secrets Manager client
	client := secretsmanager.NewFromConfig(cfg)

	// Delete the secret
	_, err = client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
		SecretId:                   &secretManagerSecretName,
		ForceDeleteWithoutRecovery: aws.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("failed to delete AWS Secret Manager secret: %w", err)
	}

	t.Log("Deleted secret from AWS Secret Manager.")

	return nil
}
