//go:build e2e
// +build e2e

package mssql_test

import (
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName   = "mssql-test"
	wiTestName = "mssql-wi-test"
)

var (
	// Regular test variables
	testNamespace             = fmt.Sprintf("%s-ns", testName)
	deploymentName            = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName          = fmt.Sprintf("%s-so", testName)
	triggerAuthenticationName = fmt.Sprintf("%s-ta", testName)
	secretName                = fmt.Sprintf("%s-secret", testName)
	mssqlServerName           = fmt.Sprintf("%s-server", testName)
	mssqlServerPodName        = fmt.Sprintf("%s-0", mssqlServerName)
	mssqlPassword             = "Pass@word1"
	mssqlDatabase             = "TestDB"
	mssqlHostname             = fmt.Sprintf("%s.%s.svc.cluster.local", mssqlServerName, testNamespace)
	mssqlConnectionString     = fmt.Sprintf("Server=%s;Database=%s;User ID=sa;Password=%s;",
		mssqlHostname, mssqlDatabase, mssqlPassword)
	minReplicaCount = 0
	maxReplicaCount = 5

	// Workload Identity test variables
	wiTestNamespace     = fmt.Sprintf("%s-ns", wiTestName)
	wiDeploymentName    = fmt.Sprintf("%s-deployment", wiTestName)
	wiScaledObjectName  = fmt.Sprintf("%s-so", wiTestName)
	wiTriggerAuthName   = fmt.Sprintf("%s-ta", wiTestName)
	wiSecretName        = fmt.Sprintf("%s-secret", wiTestName)
	wiTriggerSecretName = fmt.Sprintf("%s-ta-secret", wiTestName)
	azureSQLServerFQDN  = os.Getenv("TF_AZURE_SQL_SERVER_FQDN")
	azureSQLDBName      = os.Getenv("TF_AZURE_SQL_SERVER_DB_NAME")
	azureADTenantID     = os.Getenv("TF_AZURE_SP_TENANT")
	azureClientID       = os.Getenv("AZURE_CLIENT_ID")
)

type templateData struct {
	TestNamespace             string
	DeploymentName            string
	ScaledObjectName          string
	TriggerAuthenticationName string
	SecretName                string
	MssqlServerName           string
	MssqlHostname             string
	MssqlPassword             string
	MssqlDatabase             string
	MssqlConnectionString     string
	MinReplicaCount           int
	MaxReplicaCount           int
}

type wiTemplateData struct {
	TestNamespace                  string
	DeploymentName                 string
	ScaledObjectName               string
	TriggerAuthName                string
	SecretName                     string
	TriggerSecretName              string
	AzureSQLServerFQDN             string
	AzureSQLDBName                 string
	AzureADTenantID                string
	AzureClientID                  string
	Base64WorkloadIdentityResource string
	MinReplicaCount                int
	MaxReplicaCount                int
}

var data = templateData{
	TestNamespace:             testNamespace,
	DeploymentName:            deploymentName,
	ScaledObjectName:          scaledObjectName,
	MinReplicaCount:           minReplicaCount,
	MaxReplicaCount:           maxReplicaCount,
	TriggerAuthenticationName: triggerAuthenticationName,
	SecretName:                secretName,
	MssqlServerName:           mssqlServerName,
	MssqlHostname:             mssqlHostname,
	MssqlPassword:             mssqlPassword,
	MssqlDatabase:             mssqlDatabase,
	MssqlConnectionString:     mssqlConnectionString,
}

func getMssqlTemplateData() (templateData, []Template) {
	return data, []Template{
		{Name: "mssqlStatefulSetTemplate", Config: mssqlStatefulSetTemplate},
		{Name: "mssqlServiceTemplate", Config: mssqlServiceTemplate},
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

const (
	// Original test templates
	deploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: mssql-consumer-worker
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: mssql-consumer-worker
  template:
    metadata:
      labels:
        app: mssql-consumer-worker
    spec:
      containers:
      - image: docker.io/cgillum/mssqlscalertest:latest
        imagePullPolicy: Always
        name: mssql-consumer-worker
        args: [consumer]
        env:
          - name: SQL_CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: mssql-connection-string`

	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
stringData:
  mssql-sa-password: {{.MssqlPassword}}
  mssql-connection-string: {{.MssqlConnectionString}}`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: password
    name: {{.SecretName}}
    key: mssql-sa-password`

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  cooldownPeriod: 10
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  triggers:
  - type: mssql
    metadata:
      host: {{.MssqlHostname}}
      port: "1433"
      database: {{.MssqlDatabase}}
      username: sa
      query: "SELECT COUNT(*) FROM tasks WHERE [status]='running' OR [status]='queued'"
      targetValue: "1"
      activationTargetValue: "15"
    authenticationRef:
      name: {{.TriggerAuthenticationName}}`

	mssqlStatefulSetTemplate = `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{.MssqlServerName}}
  namespace: {{.TestNamespace}}
  labels:
    app: mssql
spec:
  replicas: 1
  serviceName: {{.MssqlServerName}}
  selector:
    matchLabels:
      app: mssql
  template:
    metadata:
      labels:
        app: mssql
    spec:
      terminationGracePeriodSeconds: 30
      containers:
      - name: mssql
        image: mcr.microsoft.com/mssql/server:2019-latest
        ports:
        - containerPort: 1433
        env:
        - name: MSSQL_PID
          value: "Developer"
        - name: ACCEPT_EULA
          value: "Y"
        - name: SA_PASSWORD
          value: {{.MssqlPassword}}
        readinessProbe:
          exec:
            command:
            - /bin/sh
            - -c
            - "/opt/mssql-tools18/bin/sqlcmd -S . -C -U sa -P '{{.MssqlPassword}}' -Q 'SELECT @@Version'"`

	mssqlServiceTemplate = `apiVersion: v1
kind: Service
metadata:
  name: {{.MssqlServerName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    app: mssql
  ports:
    - protocol: TCP
      port: 1433
      targetPort: 1433
  type: ClusterIP`

	insertRecordsJobTemplate1 = `apiVersion: batch/v1
kind: Job
metadata:
  labels:
    app: mssql-producer-job
  name: mssql-producer-job1
  namespace: {{.TestNamespace}}
spec:
  template:
    metadata:
      labels:
        app: mssql-producer-job
    spec:
      containers:
      - image: docker.io/cgillum/mssqlscalertest:latest
        imagePullPolicy: Always
        name: mssql-test-producer
        args: ["producer"]
        env:
          - name: SQL_CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: mssql-connection-string
      restartPolicy: Never
  backoffLimit: 4`

	insertRecordsJobTemplate2 = `apiVersion: batch/v1
kind: Job
metadata:
  labels:
    app: mssql-producer-job
  name: mssql-producer-job2
  namespace: {{.TestNamespace}}
spec:
  template:
    metadata:
      labels:
        app: mssql-producer-job
    spec:
      containers:
      - image: docker.io/cgillum/mssqlscalertest:latest
        imagePullPolicy: Always
        name: mssql-test-producer
        args: ["producer"]
        env:
          - name: SQL_CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: mssql-connection-string
      restartPolicy: Never
  backoffLimit: 4`

	// Workload Identity templates for Azure SQL
	wiSecretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.TriggerSecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
data:
  workloadIdentityResource: {{.Base64WorkloadIdentityResource}}`

	wiTriggerAuthTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: azure-workload
  secretTargetRef:
    - parameter: workloadIdentityResource
      name: {{.TriggerSecretName}}
      key: workloadIdentityResource`

	wiScaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  cooldownPeriod: 10
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  triggers:
    - type: mssql
      metadata:
        host: {{.AzureSQLServerFQDN}}
        database: {{.AzureSQLDBName}}
        query: "SELECT COUNT(*) FROM tasks WHERE [status]='running' OR [status]='queued'"
        targetValue: "1"
        activationTargetValue: "15"
      authenticationRef:
        name: {{.TriggerAuthName}}`

	wiDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: mssql-wi-worker
  template:
    metadata:
      labels:
        app: mssql-wi-worker
      annotations:
        azure.workload.identity/use: "true"
        azure.workload.identity/tenant-id: {{.AzureADTenantID}}
        azure.workload.identity/client-id: {{.AzureClientID}}
    spec:
      serviceAccountName: keda-tests
      containers:
        - name: mssql-worker
          image: docker.io/cgillum/mssqlscalertest:latest
          imagePullPolicy: Always`

	wiServiceAccountTemplate = `apiVersion: v1
kind: ServiceAccount
metadata:
  name: keda-tests
  namespace: {{.TestNamespace}}
  annotations:
    azure.workload.identity/client-id: {{.AzureClientID}}
    azure.workload.identity/tenant-id: {{.AzureADTenantID}}`

	wiJobTemplate1 = `apiVersion: batch/v1
kind: Job
metadata:
  name: mssql-producer-job1
  namespace: {{.TestNamespace}}
spec:
  template:
    metadata:
      labels:
        app: mssql-producer-job
      annotations:
        azure.workload.identity/use: "true"
    spec:
      serviceAccountName: keda-tests
      containers:
      - image: docker.io/cgillum/mssqlscalertest:latest
        imagePullPolicy: Always
        name: mssql-test-producer
        args: ["producer"]
        env:
          - name: SQL_CONNECTION_STRING
            value: "Server={{.AzureSQLServerFQDN}};Database={{.AzureSQLDBName}};Authentication=ActiveDirectoryManagedIdentity"
      restartPolicy: Never
  backoffLimit: 4`

	wiJobTemplate2 = `apiVersion: batch/v1
kind: Job
metadata:
  name: mssql-producer-job2
  namespace: {{.TestNamespace}}
spec:
  template:
    metadata:
      labels:
        app: mssql-producer-job
      annotations:
        azure.workload.identity/use: "true"
    spec:
      serviceAccountName: keda-tests
      containers:
      - image: docker.io/cgillum/mssqlscalertest:latest
        imagePullPolicy: Always
        name: mssql-test-producer
        args: ["producer"]
        env:
          - name: SQL_CONNECTION_STRING
            value: "Server={{.AzureSQLServerFQDN}};Database={{.AzureSQLDBName}};Authentication=ActiveDirectoryManagedIdentity"
      restartPolicy: Never
  backoffLimit: 4`
)

func TestMssqlScaler(t *testing.T) {
	// Create kubernetes resources for MS SQL server
	kc := GetKubernetesClient(t)
	_, mssqlTemplates := getMssqlTemplateData()
	_, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	CreateKubernetesResources(t, kc, testNamespace, data, mssqlTemplates)

	require.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, mssqlServerName, testNamespace, 1, 60, 3),
		"replica count should be %d after 3 minutes", 1)

	createDatabaseCommand := fmt.Sprintf("/opt/mssql-tools18/bin/sqlcmd -S . -C -U sa -P \"%s\" -Q \"CREATE DATABASE [%s]\"", mssqlPassword, mssqlDatabase)

	ok, out, errOut, err := WaitForSuccessfulExecCommandOnSpecificPod(t, mssqlServerPodName, testNamespace, createDatabaseCommand, 60, 3)
	require.True(t, ok, "executing a command on MS SQL Pod should work; Output: %s, ErrorOutput: %s, Error: %s", out, errOut, err)

	createTableCommand := fmt.Sprintf("/opt/mssql-tools18/bin/sqlcmd -S . -C -U sa -P \"%s\" -d %s -Q \"CREATE TABLE tasks ([id] int identity primary key, [status] varchar(10))\"",
		mssqlPassword, mssqlDatabase)
	ok, out, errOut, err = WaitForSuccessfulExecCommandOnSpecificPod(t, mssqlServerPodName, testNamespace, createTableCommand, 60, 3)
	require.True(t, ok, "executing a command on MS SQL Pod should work; Output: %s, ErrorOutput: %s, Error: %s", out, errOut, err)

	// Create kubernetes resources for testing
	KubectlApplyMultipleWithTemplate(t, data, templates)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc)
}

func TestMssqlWorkloadIdentityScaler(t *testing.T) {
	// Verify required environment variables
	require.NotEmpty(t, azureSQLServerFQDN, "TF_AZURE_SQL_SERVER_FQDN env variable is required")
	require.NotEmpty(t, azureSQLDBName, "TF_AZURE_SQL_SERVER_DB_NAME env variable is required")
	require.NotEmpty(t, azureADTenantID, "TF_AZURE_SP_TENANT env variable is required")
	require.NotEmpty(t, azureClientID, "AZURE_CLIENT_ID env variable is required")
	require.NotEmpty(t, os.Getenv("TF_AZURE_SQL_SERVER_ADMIN_USERNAME"), "TF_AZURE_SQL_SERVER_ADMIN_USERNAME env variable is required")
	require.NotEmpty(t, os.Getenv("TF_AZURE_SQL_SERVER_ADMIN_PASSWORD"), "TF_AZURE_SQL_SERVER_ADMIN_PASSWORD env variable is required")

	kc := GetKubernetesClient(t)

	// Create namespace
	CreateNamespace(t, kc, wiTestNamespace)

	// Create table in Azure SQL Database using admin credentials
	createTableCommand := fmt.Sprintf("sqlcmd -S %s -d %s -U %s -P %s -Q \"IF NOT EXISTS (SELECT * FROM sys.objects WHERE object_id = OBJECT_ID(N'[dbo].[tasks]') AND type in (N'U')) BEGIN CREATE TABLE tasks ([id] int identity primary key, [status] varchar(10)) END\"",
		azureSQLServerFQDN,
		azureSQLDBName,
		os.Getenv("TF_AZURE_SQL_SERVER_ADMIN_USERNAME"),
		os.Getenv("TF_AZURE_SQL_SERVER_ADMIN_PASSWORD"))

	// Create a temporary pod to run the SQL command
	tmpPodName := "sqlcmd-create-table"
	tmpPodTemplate := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  containers:
  - name: sqlcmd
    image: mcr.microsoft.com/mssql-tools
    command:
      - /bin/bash
      - -c
      - "sleep 5 && %s"
  restartPolicy: Never`, tmpPodName, wiTestNamespace, createTableCommand)

	KubectlApplyWithTemplate(t, wiTemplateData{}, "sqlcmdPod", tmpPodTemplate)

	// Setup workload identity test data
	wiData := wiTemplateData{
		TestNamespace:                  wiTestNamespace,
		DeploymentName:                 wiDeploymentName,
		ScaledObjectName:               wiScaledObjectName,
		TriggerAuthName:                wiTriggerAuthName,
		SecretName:                     wiSecretName,
		TriggerSecretName:              wiTriggerSecretName,
		AzureSQLServerFQDN:             azureSQLServerFQDN,
		AzureSQLDBName:                 azureSQLDBName,
		AzureADTenantID:                azureADTenantID,
		AzureClientID:                  azureClientID,
		Base64WorkloadIdentityResource: base64.StdEncoding.EncodeToString([]byte(azureClientID)),
		MinReplicaCount:                0,
		MaxReplicaCount:                4,
	}

	// Create service account with workload identity
	KubectlApplyWithTemplate(t, wiData, "serviceAccount", wiServiceAccountTemplate)

	// Create scaler resources
	templates := []Template{
		{Name: "wiSecretTemplate", Config: wiSecretTemplate},
		{Name: "wiDeploymentTemplate", Config: wiDeploymentTemplate},
		{Name: "wiTriggerAuthTemplate", Config: wiTriggerAuthTemplate},
		{Name: "wiScaledObjectTemplate", Config: wiScaledObjectTemplate},
	}

	CreateKubernetesResources(t, kc, wiTestNamespace, wiData, templates)

	// Cleanup
	t.Cleanup(func() {
		DeleteKubernetesResources(t, wiTestNamespace, wiData, templates)
		DeleteNamespace(t, wiTestNamespace)
	})

	// Wait for initial deployment
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, wiDeploymentName, wiTestNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// Run tests
	testWIActivation(t, kc)
	testWIScaleOut(t, kc, wiData)
	testWIScaleIn(t, kc, wiData)
}

// Original test helper functions
func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	KubectlReplaceWithTemplate(t, data, "insertRecordsJobTemplate1", insertRecordsJobTemplate1)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlReplaceWithTemplate(t, data, "insertRecordsJobTemplate2", insertRecordsJobTemplate2)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

// Workload Identity test helper functions
func testWIActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing workload identity activation ---")
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, wiDeploymentName, wiTestNamespace, 0, 60)
}

func testWIScaleOut(t *testing.T, kc *kubernetes.Clientset, wiData wiTemplateData) {
	t.Log("--- testing workload identity scale out ---")
	KubectlApplyWithTemplate(t, wiData, "wiJobTemplate1", wiJobTemplate1)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, wiDeploymentName, wiTestNamespace, 4, 60, 1),
		"replica count should be 4 after 1 minute")
}

func testWIScaleIn(t *testing.T, kc *kubernetes.Clientset, wiData wiTemplateData) {
	t.Log("--- testing workload identity scale in ---")
	KubectlApplyWithTemplate(t, wiData, "wiJobTemplate2", wiJobTemplate2)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, wiDeploymentName, wiTestNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}
