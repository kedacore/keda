//go:build e2e
// +build e2e

package couchdb_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "couchdb-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	triggerAuthName  = fmt.Sprintf("%s-ta", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	clientName       = fmt.Sprintf("%s-client", testName)
	couchdbUser      = "admin"
	couchdbHelmRepo  = "https://apache.github.io/couchdb-helm"
	couchdbDBName    = "animals"
	minReplicaCount  = 0
	maxReplicaCount  = 2
)

type templateData struct {
	TestNamespace                string
	DeploymentName               string
	ClientName                   string
	HostName                     string
	Port                         string
	Username                     string
	Password                     string
	SecretName                   string
	TriggerAuthName              string
	ScaledObjectName             string
	MinReplicaCount              int
	MaxReplicaCount              int
	Connection, Base64Connection string
	Database                     string
}

const (
	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  replicas: 1
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
`

	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  connectionString: {{.Base64Connection}}
  username: {{.Username}}
  password: {{.Password}}
`
	triggerAuthTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: password
    name: {{.SecretName}}
    key: password
  - parameter: username
    name: {{.SecretName}}
    key: username
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
  cooldownPeriod: 10
  pollingInterval: 10
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  triggers:
  - type: couchdb
    metadata:
      host: {{.HostName}}
      port: "5984"
      dbName: "animals"
      queryValue: "1"
      query: '{ "selector": { "feet": { "$gt": 0 } }, "fields": ["_id", "feet", "greeting"] }'
      activationQueryValue: "1"
      metricName: "global-metric"
    authenticationRef:
      name: {{.TriggerAuthName}}
`

	clientTemplate = `
apiVersion: v1
kind: Pod
metadata:
  name: {{.ClientName}}
  namespace: {{.TestNamespace}}
spec:
  containers:
  - name: {{.ClientName}}
    image: curlimages/curl
    command:
      - sh
      - -c
      - "exec tail -f /dev/null"`
)

func TestCouchDBScaler(t *testing.T) {
	kc := GetKubernetesClient(t)
	t.Cleanup(func() {
		data, templates := getTemplateData(t, kc)
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	// setup couchdb
	CreateNamespace(t, kc, testNamespace)
	installCouchDB(t)

	// Create kubernetes resources
	data, templates := getTemplateData(t, kc)
	KubectlApplyMultipleWithTemplate(t, data, templates)

	// wait until client is ready
	time.Sleep(10 * time.Second)
	// create database
	_, _, err := ExecCommandOnSpecificPod(t, clientName, testNamespace, fmt.Sprintf("curl -X PUT http://admin:%s@test-release-svc-couchdb.%s.svc.cluster.local:5984/animals", getPassword(t, kc), testNamespace))
	require.NoErrorf(t, err, "cannot execute command - %s", err)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	// test scaling
	testActivation(t, kc)
	testScaleUp(t, kc)
	testScaleDown(t, kc)
}

func installCouchDB(t *testing.T) {
	_, err := ExecuteCommand(fmt.Sprintf("helm repo add couchdb %s", couchdbHelmRepo))
	require.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand("helm repo update")
	require.NoErrorf(t, err, "cannot execute command - %s", err)
	uuid := strings.ReplaceAll(uuid.New().String(), "-", "")
	_, err = ExecuteCommand(fmt.Sprintf("helm install test-release  --set couchdbConfig.couchdb.uuid=%s --namespace %s couchdb/couchdb --wait", uuid, testNamespace))
	require.NoErrorf(t, err, "cannot execute command - %s", err)
}

func getPassword(t *testing.T, kc *kubernetes.Clientset) string {
	secret, err := kc.CoreV1().Secrets(testNamespace).Get(context.Background(), "test-release-couchdb", metav1.GetOptions{})
	require.NoError(t, err)
	encodedPassword := secret.Data["adminPassword"]
	password := string(encodedPassword)
	return password
}

func testActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing activation ---")
	record := `{
		"_id":"Cow",
		"feet":4,
		"greeting":"moo"
	}`
	_, _, err := ExecCommandOnSpecificPod(t, clientName, testNamespace, fmt.Sprintf("curl -X PUT http://admin:%s@test-release-svc-couchdb.%s.svc.cluster.local:5984/animals/001 -d '%s'", getPassword(t, kc), testNamespace, record))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleUp(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale up ---")
	record := `{
			"_id":"Cat",
			"feet":4,
			"greeting":"meow"
		}`
	_, _, err := ExecCommandOnSpecificPod(t, clientName, testNamespace, fmt.Sprintf("curl -X PUT http://admin:%s@test-release-svc-couchdb.%s.svc.cluster.local:5984/animals/002 -d '%s'", getPassword(t, kc), testNamespace, record))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 2),
		"replica count should be %d after 2 minute", maxReplicaCount)
}

func testScaleDown(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale down ---")

	// recreate database to clear it
	_, _, err := ExecCommandOnSpecificPod(t, clientName, testNamespace, fmt.Sprintf("curl -X DELETE http://admin:%s@test-release-svc-couchdb.%s.svc.cluster.local:5984/animals", getPassword(t, kc), testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
	_, _, err = ExecCommandOnSpecificPod(t, clientName, testNamespace, fmt.Sprintf("curl -X PUT http://admin:%s@test-release-svc-couchdb.%s.svc.cluster.local:5984/animals", getPassword(t, kc), testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 2),
		"replica count should be %d after 2 minutes", minReplicaCount)
}

func getTemplateData(t *testing.T, kc *kubernetes.Clientset) (templateData, []Template) {
	password := getPassword(t, kc)
	passwordEncoded := base64.StdEncoding.EncodeToString([]byte(password))
	connectionString := fmt.Sprintf("http://test-release-svc-couchdb.%s.svc.cluster.local:5984", testNamespace)
	hostName := fmt.Sprintf("test-release-svc-couchdb.%s.svc.cluster.local", testNamespace)
	base64ConnectionString := base64.StdEncoding.EncodeToString([]byte(connectionString))
	return templateData{
			TestNamespace:    testNamespace,
			DeploymentName:   deploymentName,
			ClientName:       clientName,
			HostName:         hostName,
			Port:             "5984",
			Username:         base64.StdEncoding.EncodeToString([]byte(couchdbUser)),
			Password:         passwordEncoded,
			TriggerAuthName:  triggerAuthName,
			SecretName:       secretName,
			ScaledObjectName: scaledObjectName,
			MinReplicaCount:  minReplicaCount,
			MaxReplicaCount:  maxReplicaCount,
			Database:         couchdbDBName,
			Connection:       connectionString,
			Base64Connection: base64ConnectionString,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthTemplate", Config: triggerAuthTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
			{Name: "clientTemplate", Config: clientTemplate},
		}
}
