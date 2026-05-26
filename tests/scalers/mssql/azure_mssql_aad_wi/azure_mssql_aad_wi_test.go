//go:build e2e
// +build e2e

package azure_mssql_aad_wi_test

import (
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
	testName = "azure-mssql-aad-wi-test"
)

var (
	testNamespace              = fmt.Sprintf("%s-ns", testName)
	deploymentName             = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName           = fmt.Sprintf("%s-so", testName)
	triggerAuthenticationName  = fmt.Sprintf("%s-ta", testName)
	secretName                 = fmt.Sprintf("%s-secret", testName)
	mssqlHelperName            = fmt.Sprintf("%s-helper", testName)
	mssqlHelperPodName         = fmt.Sprintf("%s-0", mssqlHelperName)
	azureMSSQLAdminUsername    = os.Getenv("TF_AZURE_SQL_SERVER_ADMIN_USERNAME")
	azureMSSQLAdminPassword    = os.Getenv("TF_AZURE_SQL_SERVER_ADMIN_PASSWORD")
	azureMSSQLFQDN             = os.Getenv("TF_AZURE_SQL_SERVER_FQDN")
	azureMSSQLDatabase         = os.Getenv("TF_AZURE_SQL_SERVER_DB_NAME")
	azureMSSQLUamiName         = os.Getenv("TF_AZURE_IDENTITY_1_NAME")
	azureMSSQLConnectionString = fmt.Sprintf("Server=%s;Database=%s;User ID=%s;Password=%s;Encrypt=true;TrustServerCertificate=false;",
		azureMSSQLFQDN, azureMSSQLDatabase, azureMSSQLAdminUsername, azureMSSQLAdminPassword)
	minReplicaCount = 0
	maxReplicaCount = 2
)

type templateData struct {
	TestNamespace              string
	DeploymentName             string
	ScaledObjectName           string
	TriggerAuthenticationName  string
	SecretName                 string
	MssqlHelperName            string
	AzureMSSQLAdminUsername    string
	AzureMSSQLAdminPassword    string
	AzureMSSQLFQDN             string
	AzureMSSQLDatabase         string
	AzureMSSQLUamiName         string
	AzureMSSQLConnectionString string
	MinReplicaCount            int
	MaxReplicaCount            int
}

const (
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
      - image: ghcr.io/kedacore/tests-mssql:latest
        imagePullPolicy: Always
        name: mssql-consumer-worker
        command: ["/app"]
        args: ["-mode", "consumer"]
        env:
          - name: SQL_CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: mssql-connection-string
`

	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
stringData:
  mssql-sa-password: {{.AzureMSSQLAdminPassword}}
  mssql-connection-string: {{.AzureMSSQLConnectionString}}
`

	triggerAuthTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
    provider: azure-workload
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
  - type: mssql
    metadata:
      host: {{.AzureMSSQLFQDN}}
      port: "1433"
      database: {{.AzureMSSQLDatabase}}
      query: "SELECT COUNT(*) FROM tasks WHERE [status]='running' OR [status]='queued'"
      targetValue: "4"
      activationTargetValue: "5"
    authenticationRef:
      name: {{.TriggerAuthenticationName}}
`

	mssqlHelperStatefulSetTemplate = `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{.MssqlHelperName}}
  namespace: {{.TestNamespace}}
  labels:
    app: mssql-helper
spec:
  replicas: 1
  serviceName: {{.MssqlHelperName}}
  selector:
    matchLabels:
      app: mssql-helper
  template:
    metadata:
      labels:
        app: mssql-helper
    spec:
      terminationGracePeriodSeconds: 30
      containers:
      - name: mssql-tools
        image: mcr.microsoft.com/mssql/server:2019-latest
        command: ["sleep"]
        args: ["infinity"]
`

	// inserts 3 records
	insertLowLevelRecordsJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  labels:
    app: mssql-producer-job
  name: mssql-low-level-producer-job
  namespace: {{.TestNamespace}}
spec:
  template:
    metadata:
      labels:
        app: mssql-producer-job
    spec:
      containers:
      - name: mssql-producer
        image: mcr.microsoft.com/mssql/server:2019-latest
        command:
        - /bin/sh
        - -c
        - |
          for i in 1 2 3; do
            /opt/mssql-tools18/bin/sqlcmd -S {{.AzureMSSQLFQDN}} -C -U {{.AzureMSSQLAdminUsername}} -P "{{.AzureMSSQLAdminPassword}}" -d {{.AzureMSSQLDatabase}} -Q "INSERT INTO tasks ([status]) VALUES ('running')"
          done
      restartPolicy: Never
  backoffLimit: 4
`

	// inserts 10 records
	insertRecordsJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  labels:
    app: mssql-producer-job
  name: mssql-producer-job
  namespace: {{.TestNamespace}}
spec:
  template:
    metadata:
      labels:
        app: mssql-producer-job
    spec:
      containers:
      - name: mssql-producer
        image: mcr.microsoft.com/mssql/server:2019-latest
        command:
        - /bin/sh
        - -c
        - |
          for i in 1 2 3 4 5 6 7 8 9 10; do
            /opt/mssql-tools18/bin/sqlcmd -S {{.AzureMSSQLFQDN}} -C -U {{.AzureMSSQLAdminUsername}} -P "{{.AzureMSSQLAdminPassword}}" -d {{.AzureMSSQLDatabase}} -Q "INSERT INTO tasks ([status]) VALUES ('running')"
          done
      restartPolicy: Never
  backoffLimit: 4
`
)

func TestAzureMSSQLWorkloadIdentityScaler(t *testing.T) {
	if azureMSSQLFQDN == "" || azureMSSQLAdminUsername == "" ||
		azureMSSQLAdminPassword == "" || azureMSSQLDatabase == "" ||
		azureMSSQLUamiName == "" {
		t.Skip("Skipping: Azure MSSQL AAD WI test requires TF_AZURE_MSSQL_* and TF_AZURE_IDENTITY_1_NAME env vars")
	}

	kc := GetKubernetesClient(t)
	_, helperTemplates := getHelperTemplateData()
	_, templates := getTemplateData()
	t.Cleanup(func() {
		// Drop table on remote Azure SQL
		dropTableSQL := fmt.Sprintf(
			"/opt/mssql-tools18/bin/sqlcmd -S %s -C -U %s -P \"%s\" -d %s -Q \"DROP TABLE IF EXISTS tasks\"",
			azureMSSQLFQDN, azureMSSQLAdminUsername, azureMSSQLAdminPassword, azureMSSQLDatabase)
		_, _, _, _ = WaitForSuccessfulExecCommandOnSpecificPod(t, mssqlHelperPodName, testNamespace, dropTableSQL, 60, 3)

		KubectlDeleteMultipleWithTemplate(t, data, templates)
		DeleteKubernetesResources(t, testNamespace, data, helperTemplates)
	})

	// Create helper pod for sqlcmd access
	CreateKubernetesResources(t, kc, testNamespace, data, helperTemplates)
	require.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, mssqlHelperName, testNamespace, 1, 60, 3),
		"replica count should be %d after 3 minutes", 1)

	// Drop any existing table
	dropTableSQL := fmt.Sprintf(
		"/opt/mssql-tools18/bin/sqlcmd -S %s -C -U %s -P \"%s\" -d %s -Q \"DROP TABLE IF EXISTS tasks\"",
		azureMSSQLFQDN, azureMSSQLAdminUsername, azureMSSQLAdminPassword, azureMSSQLDatabase)
	ok, out, errOut, err := WaitForSuccessfulExecCommandOnSpecificPod(t, mssqlHelperPodName, testNamespace, dropTableSQL, 60, 3)
	require.True(t, ok, "executing a command on MSSQL helper Pod should work; Output: %s, ErrorOutput: %s, Error: %s", out, errOut, err)

	// Create table on remote Azure SQL
	createTableSQL := fmt.Sprintf(
		"/opt/mssql-tools18/bin/sqlcmd -S %s -C -U %s -P \"%s\" -d %s -Q \"CREATE TABLE tasks ([id] int identity primary key, [status] varchar(10))\"",
		azureMSSQLFQDN, azureMSSQLAdminUsername, azureMSSQLAdminPassword, azureMSSQLDatabase)
	ok, out, errOut, err = WaitForSuccessfulExecCommandOnSpecificPod(t, mssqlHelperPodName, testNamespace, createTableSQL, 60, 3)
	require.True(t, ok, "executing a command on MSSQL helper Pod should work; Output: %s, ErrorOutput: %s, Error: %s", out, errOut, err)

	// Create kubernetes resources for testing
	KubectlApplyMultipleWithTemplate(t, data, templates)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	KubectlReplaceWithTemplate(t, data, "insertLowLevelRecordsJobTemplate", insertLowLevelRecordsJobTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlReplaceWithTemplate(t, data, "insertRecordsJobTemplate", insertRecordsJobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	// Update all records to processed
	updateRecords := fmt.Sprintf(
		"/opt/mssql-tools18/bin/sqlcmd -S %s -C -U %s -P \"%s\" -d %s -Q \"UPDATE tasks SET [status] = 'processed'\"",
		azureMSSQLFQDN, azureMSSQLAdminUsername, azureMSSQLAdminPassword, azureMSSQLDatabase)
	ok, out, errOut, err := WaitForSuccessfulExecCommandOnSpecificPod(t, mssqlHelperPodName, testNamespace, updateRecords, 60, 3)
	require.True(t, ok, "executing a command on MSSQL helper Pod should work; Output: %s, ErrorOutput: %s, Error: %s", out, errOut, err)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

var data = templateData{
	TestNamespace:              testNamespace,
	DeploymentName:             deploymentName,
	ScaledObjectName:           scaledObjectName,
	MinReplicaCount:            minReplicaCount,
	MaxReplicaCount:            maxReplicaCount,
	TriggerAuthenticationName:  triggerAuthenticationName,
	SecretName:                 secretName,
	MssqlHelperName:            mssqlHelperName,
	AzureMSSQLAdminUsername:    azureMSSQLAdminUsername,
	AzureMSSQLAdminPassword:    azureMSSQLAdminPassword,
	AzureMSSQLFQDN:             azureMSSQLFQDN,
	AzureMSSQLDatabase:         azureMSSQLDatabase,
	AzureMSSQLUamiName:         azureMSSQLUamiName,
	AzureMSSQLConnectionString: azureMSSQLConnectionString,
}

func getHelperTemplateData() (templateData, []Template) {
	return data, []Template{
		{Name: "mssqlHelperStatefulSetTemplate", Config: mssqlHelperStatefulSetTemplate},
	}
}

func getTemplateData() (templateData, []Template) {
	return data, []Template{
		{Name: "secretTemplate", Config: secretTemplate},
		{Name: "deploymentTemplate", Config: deploymentTemplate},
		{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
		{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
	}
}
