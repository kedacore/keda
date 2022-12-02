//go:build e2e
// +build e2e

package cassandra_test

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
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
        image: nginx:1.14.2
        ports:
        - containerPort: 80
`

	cassandraDeploymentTemplate = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  labels:
    app: cassandra-app
  name: cassandra
  namespace: {{.TestNamespace}}
spec:
  serviceName: {{.DeploymentName}}
  replicas: 1
  selector:
    matchLabels:
      app: cassandra-app
  template:
    metadata:
      labels:
        app: cassandra-app
    spec:
      containers:
      - image: bitnami/cassandra:4.0.4
        imagePullPolicy: IfNotPresent
        name: cassandra
        ports:
        - containerPort: 9042
        env:
          - name: MAX_HEAP_SIZE
            value: 1024M
          - name: HEAP_NEWSIZE
            value: 100M
`

	cassandraClientDeploymentTemplate = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  labels:
    app: cassandra-client
  name: cassandra-client
  namespace: {{.TestNamespace}}
spec:
  serviceName: {{.DeploymentName}}
  replicas: 1
  selector:
    matchLabels:
      app: cassandra-client
  template:
    metadata:
      labels:
        app: cassandra-client
    spec:
      containers:
      - image: bitnami/cassandra:4.0.4
        imagePullPolicy: IfNotPresent
        name: cassandra-client
        ports:
        - containerPort: 9042
        env:
          - name: MAX_HEAP_SIZE
            value: 1024M
          - name: HEAP_NEWSIZE
            value: 100M
`

	serviceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
    ports:
      - name: cql
        port: 9042
        protocol: TCP
        targetPort: 9042
    selector:
        app: cassandra-app
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
      clusterIPAddress: "{{.DeploymentName}}.{{.TestNamespace}}"
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
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// setup elastic
	setupCassandra(t, kc)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	// test scaling
	testActivation(t, kc)
	testScaleOut(t, kc)
	testScaleIn(t, kc)

	// cleanup
	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
}

func setupCassandra(t *testing.T, kc *kubernetes.Clientset) {
	assert.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, "cassandra", testNamespace, 1, 60, 3),
		"cassandra should be up")
	err := checkIfCassandraStatusIsReady(t, "cassandra-0")
	assert.NoErrorf(t, err, "%s", err)
	assert.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, "cassandra-client", testNamespace, 1, 60, 3),
		"cassandra should be up")
	err = checkIfCassandraStatusIsReady(t, "cassandra-client-0")
	t.Log("--- cassandra is up ---")
	assert.NoErrorf(t, err, "%s", err)
	// Create the table
	out, errOut, _ := ExecCommandOnSpecificPod(t, "cassandra-client-0", testNamespace, fmt.Sprintf("bash cqlsh -u %s -p %s %s.%s --execute=\"%s\"", cassandraUsername, cassandraPassword, deploymentName, testNamespace, createKeyspace))
	t.Logf("Output: %s, Error: %s", out, errOut)
	out, errOut, _ = ExecCommandOnSpecificPod(t, "cassandra-client-0", testNamespace, fmt.Sprintf("bash cqlsh -u %s -p %s %s.%s --execute=\"%s\"", cassandraUsername, cassandraPassword, deploymentName, testNamespace, createTableCQL))
	t.Logf("Output: %s, Error: %s", out, errOut)
	t.Log("--- cassandra is ready ---")
}

func checkIfCassandraStatusIsReady(t *testing.T, name string) error {
	t.Log("--- checking cassandra status ---")
	time.Sleep(time.Second * 10)
	for i := 0; i < 60; i++ {
		out, errOut, _ := ExecCommandOnSpecificPod(t, name, testNamespace, "nodetool status")
		t.Logf("Output: %s, Error: %s", out, errOut)
		if !strings.Contains(out, "UN ") {
			time.Sleep(time.Second * 10)
			continue
		}
		return nil
	}
	return errors.New("cassandra is not ready")
}

func testActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing activation ---")
	result, err := getCassandraInsertCmd(insertDataTemplateA)
	assert.NoErrorf(t, err, "cannot parse log - %s", err)
	out, errOut, _ := ExecCommandOnSpecificPod(t, "cassandra-client-0", testNamespace, fmt.Sprintf("bash cqlsh -u %s -p %s %s.%s --execute=\"%s\"", cassandraUsername, cassandraPassword, deploymentName, testNamespace, result))
	t.Logf("Output: %s, Error: %s", out, errOut)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")
	result, err := getCassandraInsertCmd(insertDataTemplateB)
	assert.NoErrorf(t, err, "cannot parse log - %s", err)
	out, errOut, _ := ExecCommandOnSpecificPod(t, "cassandra-client-0", testNamespace, fmt.Sprintf("bash cqlsh -u %s -p %s %s.%s --execute=\"%s\"", cassandraUsername, cassandraPassword, deploymentName, testNamespace, result))
	t.Logf("Output: %s, Error: %s", out, errOut)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	out, errOut, _ := ExecCommandOnSpecificPod(t, "cassandra-client-0", testNamespace, fmt.Sprintf("bash cqlsh -u %s -p %s %s.%s --execute=\"%s\"", cassandraUsername, cassandraPassword, deploymentName, testNamespace, truncateData))
	t.Logf("Output: %s, Error: %s", out, errOut)

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
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "cassandraDeploymentTemplate", Config: cassandraDeploymentTemplate},
			{Name: "cassandraClientDeploymentTemplate", Config: cassandraClientDeploymentTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
