//go:build e2e
// +build e2e

package clickhouse_test

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
	testName = "clickhouse-test"
)

var (
	testNamespace              = fmt.Sprintf("%s-ns", testName)
	deploymentName             = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName           = fmt.Sprintf("%s-so", testName)
	triggerAuthenticationName  = fmt.Sprintf("%s-ta", testName)
	secretName                 = fmt.Sprintf("%s-secret", testName)
	clickhouseDeploymentName   = "clickhouse"
	clickhouseServiceName      = "clickhouse"
	clickhouseHostname         = fmt.Sprintf("%s.%s.svc.cluster.local", clickhouseServiceName, testNamespace)
	clickhouseConnectionString = fmt.Sprintf("clickhouse://default@%s:9000/default", clickhouseHostname)
	minReplicaCount            = 0
	maxReplicaCount            = 2
)

type templateData struct {
	TestNamespace                    string
	DeploymentName                   string
	ScaledObjectName                 string
	TriggerAuthenticationName        string
	SecretName                       string
	ClickhouseDeploymentName         string
	ClickhouseServiceName            string
	ClickhouseHostname               string
	ClickhouseConnectionStringBase64 string
	MinReplicaCount                  int
	MaxReplicaCount                  int
	JobCommand                       string
}

const (
	clickhouseDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.ClickhouseDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: clickhouse
spec:
  replicas: 1
  selector:
    matchLabels:
      app: clickhouse
  template:
    metadata:
      labels:
        app: clickhouse
    spec:
      containers:
      - name: clickhouse
        image: clickhouse/clickhouse-server:24.3
        imagePullPolicy: Always
        ports:
        - containerPort: 9000
          name: native
        - containerPort: 8123
          name: http
        readinessProbe:
          tcpSocket:
            port: 9000
          initialDelaySeconds: 10
          periodSeconds: 5
`

	clickhouseServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: {{.ClickhouseServiceName}}
  namespace: {{.TestNamespace}}
spec:
  ports:
  - port: 9000
    targetPort: 9000
    name: native
  selector:
    app: clickhouse
  type: ClusterIP
`

	secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
type: Opaque
data:
  clickhouse_conn_str: {{.ClickhouseConnectionStringBase64}}
`

	triggerAuthenticationTemplate = `
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthenticationName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: connectionString
    name: {{.SecretName}}
    key: clickhouse_conn_str
`

	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: clickhouse-consumer
spec:
  replicas: 0
  selector:
    matchLabels:
      app: clickhouse-consumer
  template:
    metadata:
      labels:
        app: clickhouse-consumer
    spec:
      containers:
      - name: nginx
        image: ghcr.io/nginx/nginx-unprivileged:1.26
        ports:
        - containerPort: 80
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
  cooldownPeriod: 10
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  triggers:
  - type: clickhouse
    metadata:
      query: "SELECT count() FROM default.task_instance"
      targetQueryValue: "1"
      activationTargetQueryValue: "4"
    authenticationRef:
      name: {{.TriggerAuthenticationName}}
`

	jobTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: clickhouse-job
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 0
  backoffLimit: 4
  template:
    spec:
      containers:
      - name: clickhouse-client
        image: clickhouse/clickhouse-client:24.3
        imagePullPolicy: Always
        command:
          - sh
          - -c
          - {{.JobCommand}}
      restartPolicy: Never
`
)

func TestClickHouseScaler(t *testing.T) {
	kc := GetKubernetesClient(t)
	data := getTemplateData()
	chTemplates := getClickHouseTemplates()
	testTemplates := getTestTemplates()
	allTemplates := make([]Template, 0, len(chTemplates)+len(testTemplates))
	allTemplates = append(allTemplates, chTemplates...)
	allTemplates = append(allTemplates, testTemplates...)
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, allTemplates)
	})

	CreateKubernetesResources(t, kc, testNamespace, data, chTemplates)

	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, clickhouseDeploymentName, testNamespace, 1, 120, 3),
		"ClickHouse should be ready")

	// Allow ClickHouse to fully accept connections after port is open
	t.Log("Waiting for ClickHouse to accept connections...")
	time.Sleep(15 * time.Second)

	setupClickHouseTable(t, kc, data)

	KubectlApplyMultipleWithTemplate(t, data, testTemplates)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d", minReplicaCount)

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc, data)
}

func setupClickHouseTable(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	data.JobCommand = fmt.Sprintf(
		"clickhouse-client --host %s --port 9000 -q \"CREATE TABLE default.task_instance (id UInt64) ENGINE = MergeTree() ORDER BY id\"",
		clickhouseHostname,
	)
	KubectlApplyWithTemplate(t, data, "jobTemplate", jobTemplate)
	require.True(t, WaitForJobSuccess(t, kc, "clickhouse-job", testNamespace, 60, 3), "create table job failed")
	KubectlDeleteWithTemplate(t, data, "jobTemplate", jobTemplate)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	data.JobCommand = fmt.Sprintf(
		"clickhouse-client --host %s --port 9000 -q \"INSERT INTO default.task_instance VALUES (1),(2),(3)\"",
		clickhouseHostname,
	)
	KubectlApplyWithTemplate(t, data, "jobTemplate", jobTemplate)
	require.True(t, WaitForJobSuccess(t, kc, "clickhouse-job", testNamespace, 60, 3), "activation insert job failed")
	KubectlDeleteWithTemplate(t, data, "jobTemplate", jobTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	data.JobCommand = fmt.Sprintf(
		"clickhouse-client --host %s --port 9000 -q \"INSERT INTO default.task_instance VALUES (4),(5),(6)\"",
		clickhouseHostname,
	)
	KubectlApplyWithTemplate(t, data, "jobTemplate", jobTemplate)
	require.True(t, WaitForJobSuccess(t, kc, "clickhouse-job", testNamespace, 60, 3), "scale out insert job failed")
	KubectlDeleteWithTemplate(t, data, "jobTemplate", jobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 120, 3),
		"replica count should be %d", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in ---")
	data.JobCommand = fmt.Sprintf(
		"clickhouse-client --host %s --port 9000 -q \"TRUNCATE TABLE default.task_instance\"",
		clickhouseHostname,
	)
	KubectlApplyWithTemplate(t, data, "jobTemplate", jobTemplate)
	require.True(t, WaitForJobSuccess(t, kc, "clickhouse-job", testNamespace, 60, 3), "truncate job failed")
	KubectlDeleteWithTemplate(t, data, "jobTemplate", jobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 120, 3),
		"replica count should be %d", minReplicaCount)
}

func getTemplateData() templateData {
	return templateData{
		TestNamespace:                    testNamespace,
		DeploymentName:                   deploymentName,
		ScaledObjectName:                 scaledObjectName,
		TriggerAuthenticationName:        triggerAuthenticationName,
		SecretName:                       secretName,
		ClickhouseDeploymentName:         clickhouseDeploymentName,
		ClickhouseServiceName:            clickhouseServiceName,
		ClickhouseHostname:               clickhouseHostname,
		ClickhouseConnectionStringBase64: base64.StdEncoding.EncodeToString([]byte(clickhouseConnectionString)),
		MinReplicaCount:                  minReplicaCount,
		MaxReplicaCount:                  maxReplicaCount,
	}
}

func getClickHouseTemplates() []Template {
	return []Template{
		{Name: "clickhouseDeploymentTemplate", Config: clickhouseDeploymentTemplate},
		{Name: "clickhouseServiceTemplate", Config: clickhouseServiceTemplate},
	}
}

func getTestTemplates() []Template {
	return []Template{
		{Name: "secretTemplate", Config: secretTemplate},
		{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
		{Name: "deploymentTemplate", Config: deploymentTemplate},
		{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
	}
}
