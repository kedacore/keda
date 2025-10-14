//go:build e2e
// +build e2e

package mssql_test

import (
	"fmt"
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
	testName = "mssql-test"
)

var (
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
  mssql-sa-password: {{.MssqlPassword}}
  mssql-connection-string: {{.MssqlConnectionString}}
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
    secretTargetRef:
    - parameter: password
      name: {{.SecretName}}
      key: mssql-sa-password
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
      host: {{.MssqlHostname}}
      port: "1433"
      database: {{.MssqlDatabase}}
      username: sa
      query: "SELECT COUNT(*) FROM tasks WHERE [status]='running' OR [status]='queued'"
      targetValue: "1" # one replica per row
      activationTargetValue: "15"
    authenticationRef:
      name: {{.TriggerAuthenticationName}}
`

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
            - "/opt/mssql-tools18/bin/sqlcmd -S . -C -U sa -P '{{.MssqlPassword}}' -Q 'SELECT @@Version'"
`

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
  type: ClusterIP
`

	// inserts 10 records in the table
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
      - image: ghcr.io/kedacore/tests-mssql:latest
        imagePullPolicy: Always
        name: mssql-test-producer
        command: ["/app"]
        args: ["-mode", "producer"]
        env:
          - name: SQL_CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: mssql-connection-string
      restartPolicy: Never
  backoffLimit: 4
`

	// inserts 10 records in the table
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
      - image: ghcr.io/kedacore/tests-mssql:latest
        imagePullPolicy: Always
        name: mssql-test-producer
        command: ["/app"]
        args: ["-mode", "producer"]
        env:
          - name: SQL_CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: mssql-connection-string
      restartPolicy: Never
  backoffLimit: 4
  `
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

// insert 10 records in the table -> activation should not happen (activationTargetValue = 15)
func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	KubectlReplaceWithTemplate(t, data, "insertRecordsJobTemplate1", insertRecordsJobTemplate1)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

// insert another 10 records in the table, which in total is 20 -> should be scaled up
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
