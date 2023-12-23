//go:build e2e
// +build e2e

package gcp_secret_manager_workload_identity_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "gcp-secret-manager-workload-identity-test"
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

	gcpKey                = os.Getenv("TF_GCP_SA_CREDENTIALS")
	creds                 = make(map[string]interface{})
	errGcpKey             = json.Unmarshal([]byte(gcpKey), &creds)
	projectID             = creds["project_id"]
	secretManagerSecretID = "connectionStringWorkloadIdentity"
)

type templateData struct {
	TestNamespace                    string
	DeploymentName                   string
	ScaledObjectName                 string
	TriggerAuthenticationName        string
	SecretManagerSecretName          string
	SecretManagerSecretVersion       string
	SecretName                       string
	PostgreSQLStatefulSetName        string
	PostgreSQLConnectionStringBase64 string
	PostgreSQLUsername               string
	PostgreSQLPassword               string
	PostgreSQLDatabase               string
	MinReplicaCount                  int
	MaxReplicaCount                  int
	SecretManagerSecretID            string
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

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  gcpSecretManager:
    secrets:
      - parameter: connection
        id: {{.SecretManagerSecretID}}
        version: "1"
    podIdentity:
      provider: gcp
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
            value: "10000"
          - name: CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: postgresql_conn_str
      restartPolicy: Never
  backoffLimit: 4
`
)

func TestPostgreSQLScaler(t *testing.T) {
	require.NotEmpty(t, gcpKey, "TF_GCP_SA_CREDENTIALS env variable is required for GCP Secret Manager test")
	require.NoErrorf(t, errGcpKey, "Failed to load credentials from gcpKey - %s", errGcpKey)

	// Create the secret in GCP
	err := createGCPSecret(t)
	assert.NoErrorf(t, err, "cannot create GCP Secret Manager secret - %s", err)

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
	err = deleteGCPSecret(t)
	assert.NoErrorf(t, err, "cannot delete GCP Secret Manager secret - %s", err)
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
	SecretManagerSecretID:            secretManagerSecretID,
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

func createGCPSecret(t *testing.T) error {
	ctx := context.Background()

	gcpCreds, err := google.CredentialsFromJSON(ctx, []byte(gcpKey), secretmanager.DefaultAuthScopes()...)
	if err != nil {
		return fmt.Errorf("failed to get credentials from json: %w", err)
	}

	client, err := secretmanager.NewClient(ctx, option.WithCredentials(gcpCreds))
	if err != nil {
		return fmt.Errorf("failed to create secretmanager client: %w", err)
	}
	defer client.Close()

	// Create a new secret version
	createReq := &secretmanagerpb.CreateSecretRequest{
		Parent:   fmt.Sprintf("projects/%s", projectID),
		SecretId: secretManagerSecretID,
		Secret: &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{},
				},
			},
		},
	}

	secret, err := client.CreateSecret(ctx, createReq)
	if err != nil {
		return fmt.Errorf("failed to create secret, %w", err)
	}

	// Create a new secret version
	payload := []byte(postgreSQLConnectionString)
	createVersionReq := &secretmanagerpb.AddSecretVersionRequest{
		Parent: secret.Name,
		Payload: &secretmanagerpb.SecretPayload{
			Data: payload,
		},
	}

	_, err = client.AddSecretVersion(ctx, createVersionReq)
	if err != nil {
		return fmt.Errorf("failed to create secret version: %v", err)
	}

	t.Log("Created secret in GCP Secret Manager.")

	return nil
}

func deleteGCPSecret(t *testing.T) error {
	ctx := context.Background()

	gcpCreds, err := google.CredentialsFromJSON(ctx, []byte(gcpKey), secretmanager.DefaultAuthScopes()...)
	if err != nil {
		return fmt.Errorf("failed to get credentials from json: %w", err)
	}

	client, err := secretmanager.NewClient(ctx, option.WithCredentials(gcpCreds))
	if err != nil {
		return fmt.Errorf("failed to create secretmanager client: %w", err)
	}
	defer client.Close()

	// Build the secret name.
	secretName := fmt.Sprintf("projects/%s/secrets/%s", projectID, secretManagerSecretID)

	// Create a request to delete the secret.
	req := &secretmanagerpb.DeleteSecretRequest{
		Name: secretName,
	}

	// Delete the secret.
	if err := client.DeleteSecret(ctx, req); err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	t.Log("Deleted secret from GCP Secret Manager.")

	return nil
}
