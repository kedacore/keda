//go:build e2e
// +build e2e

package mysql_test

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "mysql-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)

	mySQLUsername     = "test-user"
	mySQLPassword     = "test-password"
	mySQLDatabase     = "test_db"
	mySQLRootPassword = "some-password"
)

type templateData struct {
	TestNamespace         string
	DeploymentName        string
	ScaledObjectName      string
	SecretName            string
	MySQLUsername         string
	MySQLPassword         string
	MySQLDatabase         string
	MySQLRootPassword     string
	MySQLConnectionString string
	ItemsToWrite          int
}

const (
	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: mysql-update-worker
spec:
  replicas: 0
  selector:
    matchLabels:
      app: mysql-update-worker
  template:
    metadata:
      labels:
        app: mysql-update-worker
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-mysql
        imagePullPolicy: Always
        name: mysql-processor-test
        command:
          - /app
          - update
        env:
          - name: TASK_INSTANCES_COUNT
            value: "4000"
          - name: CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: mysql_conn_str
`

	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
data:
  mysql_conn_str: {{.MySQLConnectionString}}
`

	triggerAuthTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-mysql-secret
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: connectionString
    name: {{.SecretName}}
    key: mysql_conn_str
`

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  cooldownPeriod:  10
  minReplicaCount: 0
  maxReplicaCount: 2
  triggers:
  - type: mysql
    metadata:
      queryValue: "4"
      activationQueryValue: "100"
      query: "SELECT CEIL(COUNT(*) / 5) FROM task_instance WHERE state='running' OR state='queued'"
    authenticationRef:
      name: keda-trigger-auth-mysql-secret
`

	insertRecordsJobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  labels:
    app: mysql-insert-job
  name: mysql-insert-job
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 0
  template:
    metadata:
      labels:
        app: mysql-insert-job
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-mysql
        imagePullPolicy: Always
        name: mysql-processor-test
        command:
          - /app
          - insert
        env:
          - name: TASK_INSTANCES_COUNT
            value: "{{.ItemsToWrite}}"
          - name: CONNECTION_STRING
            valueFrom:
              secretKeyRef:
                name: {{.SecretName}}
                key: mysql_conn_str
      restartPolicy: Never
  backoffLimit: 4
`

	mysqlDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: mysql
  name: mysql
  namespace: {{.TestNamespace}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mysql
  template:
    metadata:
      labels:
        app: mysql
    spec:
      containers:
      - image: mysql:8.0.20
        name: mysql
        env:
          - name: MYSQL_ROOT_PASSWORD
            value: {{.MySQLRootPassword}}
          - name: MYSQL_USER
            value: {{.MySQLUsername}}
          - name: MYSQL_PASSWORD
            value: {{.MySQLPassword}}
          - name: MYSQL_DATABASE
            value: {{.MySQLDatabase}}
        ports:
          - name: mysql
            protocol: TCP
            containerPort: 3600
        readinessProbe:
          exec:
            command:
            - sh
            - -c
            - "mysqladmin ping -u root -p{{.MySQLRootPassword}}"
`

	mysqlServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  labels:
    app: mysql
  name: mysql
  namespace: {{.TestNamespace}}
spec:
  ports:
  - port: 3306
    protocol: TCP
    targetPort: 3306
  selector:
    app: mysql
  type: ClusterIP
`
)

func TestMySQLScaler(t *testing.T) {
	t.Log("--- setting up ---")
	kc := GetKubernetesClient(t)
	CreateNamespace(t, kc, testNamespace)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	// setup MySQL
	setupMySQL(t, kc, data, templates)

	// test scaling
	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc)
}

func setupMySQL(t *testing.T, kc *kubernetes.Clientset, data templateData, templates []Template) {
	// Deploy mysql
	KubectlApplyWithTemplate(t, data, "mysqlDeploymentTemplate", mysqlDeploymentTemplate)
	KubectlApplyWithTemplate(t, data, "mysqlServiceTemplate", mysqlServiceTemplate)
	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, "mysql", testNamespace, 1, 30, 2), "mysql is not in a ready state")
	// Wait 30 sec which would be enought for mysql to be accessible
	time.Sleep(30 * time.Second)

	// Create table that used by the job and the worker
	createTableSQL := fmt.Sprintf("CREATE TABLE %s.task_instance (id INT AUTO_INCREMENT PRIMARY KEY,state VARCHAR(10));", mySQLDatabase)
	out, err := ExecuteCommand(fmt.Sprintf("kubectl get pods -n %s -o jsonpath='{.items[0].metadata.name}'", testNamespace))
	mysqlPod := string(out)
	if assert.NoErrorf(t, err, "cannot execute command - %s", err) {
		require.NotEmpty(t, mysqlPod)
	}
	_, err = ExecuteCommand(fmt.Sprintf("kubectl exec -n %s %s -- mysql -u%s -p%s -e '%s'", testNamespace, mysqlPod, mySQLUsername, mySQLPassword, createTableSQL))
	require.NoErrorf(t, err, "cannot execute command - %s", err)

	// Deploy mysql consumer app, scaled object and trigger auth, etc.
	KubectlApplyMultipleWithTemplate(t, data, templates)
	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should start out as 0")
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	t.Log("--- applying job ---")
	data.ItemsToWrite = 50
	KubectlReplaceWithTemplate(t, data, "insertRecordsJobTemplate", insertRecordsJobTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	t.Log("--- applying job ---")
	data.ItemsToWrite = 4000
	KubectlReplaceWithTemplate(t, data, "insertRecordsJobTemplate", insertRecordsJobTemplate)
	// Check if deployment scale to 2 (the max)
	maxReplicaCount := 2
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 120, 1),
		"Replica count should scale out in next 2 minutes")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")
	// Check if deployment scale in to 0 after 6 minutes
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 360, 1),
		"Replica count should be 0 after 6 minutes")
}

func getTemplateData() (templateData, []Template) {
	base64MySQLConnectionString := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s@tcp(mysql.%s.svc.cluster.local:3306)/%s", mySQLUsername, mySQLPassword, testNamespace, mySQLDatabase)))
	return templateData{
			TestNamespace:         testNamespace,
			DeploymentName:        deploymentName,
			ScaledObjectName:      scaledObjectName,
			SecretName:            secretName,
			MySQLUsername:         mySQLUsername,
			MySQLPassword:         mySQLPassword,
			MySQLDatabase:         mySQLDatabase,
			MySQLRootPassword:     mySQLRootPassword,
			MySQLConnectionString: base64MySQLConnectionString,
			ItemsToWrite:          0,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
		}
}
