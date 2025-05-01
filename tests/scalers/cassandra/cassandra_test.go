//go:build e2e
// +build e2e

package cassandra_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"testing"
	"text/template"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "cassandra-test"
)

var (
	testNamespace      = fmt.Sprintf("%s-ns", testName)
	deploymentName     = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName   = fmt.Sprintf("%s-so", testName)
	secretName         = fmt.Sprintf("%s-secret", testName)
	cassandraKeyspace  = "test_keyspace"
	cassandraTableName = "test_table"
	cassandraUsername  = "cassandra"
	cassandraPassword  = "cassandra"
	createKeyspace     = fmt.Sprintf("CREATE KEYSPACE IF NOT EXISTS %s WITH REPLICATION = {'class': 'NetworkTopologyStrategy', 'datacenter1' : '1'};", cassandraKeyspace)
	createTableCQL     = fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (name text, surname text, age int, PRIMARY KEY (name, surname));", cassandraKeyspace, cassandraTableName)
	truncateData       = fmt.Sprintf("TRUNCATE %s.%s;", cassandraKeyspace, cassandraTableName)
	minReplicaCount    = 0
	maxReplicaCount    = 2
)

type templateData struct {
	TestNamespace           string
	DeploymentName          string
	ScaledObjectName        string
	SecretName              string
	Command                 string
	CassandraPasswordBase64 string
	CassandraKeyspace       string
	CassandraTableName      string
}

const (
	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  cassandra_password: {{.CassandraPasswordBase64}}
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-cassandra-secret
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: password
    name: {{.SecretName}}
    key: cassandra_password
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

	jobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: client
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - name: client
        image: "bitnami/cassandra"
        imagePullPolicy: Always
        command:
        - sh
        - -c
        - "{{.Command}}"
        env:
        - name: MAX_HEAP_SIZE
          value: 1024M
        - name: HEAP_NEWSIZE
          value: 100M
      restartPolicy: OnFailure
  backoffLimit: 4
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
  minReplicaCount: 0
  maxReplicaCount: 2
  pollingInterval: 1
  cooldownPeriod:  1
  triggers:
  - type: cassandra
    metadata:
      username: "cassandra"
      clusterIPAddress: "cassandra.{{.TestNamespace}}"
      consistency: "Quorum"
      protocolVersion: "4"
      port: "9042"
      keyspace: "{{.CassandraKeyspace}}"
      query: "SELECT COUNT(*) FROM {{.CassandraKeyspace}}.{{.CassandraTableName}};"
      targetQueryValue: "1"
      activationTargetQueryValue: "4"
      metricName: "{{.CassandraKeyspace}}"
    authenticationRef:
      name: keda-trigger-auth-cassandra-secret
`
	insertDataTemplateA = `BEGIN BATCH
    INSERT INTO {{.CassandraKeyspace}}.{{.CassandraTableName}} (name, surname, age) VALUES ('Mary', 'Paul', 30);
    INSERT INTO {{.CassandraKeyspace}}.{{.CassandraTableName}} (name, surname, age) VALUES ('James', 'Miller', 25);
    INSERT INTO {{.CassandraKeyspace}}.{{.CassandraTableName}} (name, surname, age) VALUES ('Lisa', 'Wilson', 29);
    APPLY BATCH;`

	insertDataTemplateB = `BEGIN BATCH
    INSERT INTO {{.CassandraKeyspace}}.{{.CassandraTableName}} (name, surname, age) VALUES ('Bob', 'Taylor', 33);
    INSERT INTO {{.CassandraKeyspace}}.{{.CassandraTableName}} (name, surname, age) VALUES ('Carol', 'Moore', 31);
    INSERT INTO {{.CassandraKeyspace}}.{{.CassandraTableName}} (name, surname, age) VALUES ('Richard', 'Brown', 23);
    APPLY BATCH;`
)

func TestCassandraScaler(t *testing.T) {
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateNamespace(t, kc, testNamespace)
	// cleanup
	t.Cleanup(func() {
		uninstallCassandra(t)
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	// setup cassandra
	installCassandra(t)
	setupCassandra(t, kc, data)

	// deploy test resources
	KubectlApplyMultipleWithTemplate(t, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	// test scaling
	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc, data)
}

func installCassandra(t *testing.T) {
	_, err := ExecuteCommand("helm repo add bitnami https://charts.bitnami.com/bitnami")
	require.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand("helm repo update")
	require.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand(fmt.Sprintf("helm install cassandra --set resourcesPreset=none --set persistence.enabled=false --set dbUser.user=%s --set dbUser.password=%s --namespace %s bitnami/cassandra --wait", cassandraUsername, cassandraPassword, testNamespace))
	require.NoErrorf(t, err, "cannot execute command - %s", err)
}

func uninstallCassandra(t *testing.T) {
	_, err := ExecuteCommand(fmt.Sprintf("helm uninstall cassandra --namespace %s", testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func setupCassandra(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	// Create the key space
	data.Command = fmt.Sprintf("cqlsh -u %s -p %s cassandra.%s --execute=\\\"%s\\\"", cassandraUsername, cassandraPassword, testNamespace, createKeyspace)
	KubectlApplyWithTemplate(t, data, "jobTemplate", jobTemplate)
	require.True(t, WaitForJobSuccess(t, kc, "client", testNamespace, 6, 10), "create database job failed")
	KubectlDeleteWithTemplate(t, data, "jobTemplate", jobTemplate)
	// Create the table
	data.Command = fmt.Sprintf("cqlsh -u %s -p %s cassandra.%s --execute=\\\"%s\\\"", cassandraUsername, cassandraPassword, testNamespace, createTableCQL)
	KubectlApplyWithTemplate(t, data, "jobTemplate", jobTemplate)
	require.True(t, WaitForJobSuccess(t, kc, "client", testNamespace, 6, 10), "create database job failed")
	KubectlDeleteWithTemplate(t, data, "jobTemplate", jobTemplate)

	t.Log("--- cassandra is ready ---")
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	result, err := getCassandraInsertCmd(insertDataTemplateA)
	assert.NoErrorf(t, err, "cannot parse log - %s", err)
	data.Command = fmt.Sprintf("cqlsh -u %s -p %s cassandra.%s --execute=\\\"%s\\\"", cassandraUsername, cassandraPassword, testNamespace, result)
	KubectlApplyWithTemplate(t, data, "jobTemplate", jobTemplate)
	assert.True(t, WaitForJobSuccess(t, kc, "client", testNamespace, 6, 10), "insert job failed")
	KubectlDeleteWithTemplate(t, data, "jobTemplate", jobTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	result, err := getCassandraInsertCmd(insertDataTemplateB)
	assert.NoErrorf(t, err, "cannot parse log - %s", err)
	data.Command = fmt.Sprintf("cqlsh -u %s -p %s cassandra.%s --execute=\\\"%s\\\"", cassandraUsername, cassandraPassword, testNamespace, result)
	KubectlApplyWithTemplate(t, data, "jobTemplate", jobTemplate)
	assert.True(t, WaitForJobSuccess(t, kc, "client", testNamespace, 6, 10), "insert job failed")
	KubectlDeleteWithTemplate(t, data, "jobTemplate", jobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in ---")

	data.Command = fmt.Sprintf("cqlsh -u %s -p %s cassandra.%s --execute=\\\"%s\\\"", cassandraUsername, cassandraPassword, testNamespace, truncateData)
	KubectlApplyWithTemplate(t, data, "jobTemplate", jobTemplate)
	assert.True(t, WaitForJobSuccess(t, kc, "client", testNamespace, 6, 10), "insert job failed")
	KubectlDeleteWithTemplate(t, data, "jobTemplate", jobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

func getCassandraInsertCmd(insertDataTemplate string) (string, error) {
	tmpl, err := template.New("cassandra insert").Parse(insertDataTemplate)
	var tpl bytes.Buffer
	if err := tmpl.Execute(&tpl, templateData{CassandraKeyspace: cassandraKeyspace, CassandraTableName: cassandraTableName}); err != nil {
		return "", err
	}
	result := tpl.String()
	return result, err
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:           testNamespace,
			DeploymentName:          deploymentName,
			ScaledObjectName:        scaledObjectName,
			SecretName:              secretName,
			CassandraPasswordBase64: base64.StdEncoding.EncodeToString([]byte(cassandraPassword)),
			CassandraKeyspace:       cassandraKeyspace,
			CassandraTableName:      cassandraTableName,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
