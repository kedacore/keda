//go:build e2e
// +build e2e

package clickhouse_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

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
	clickhouseUser             = "default"
	clickhousePassword         = "keda-test-password"
	clickhouseConnectionString = fmt.Sprintf("clickhouse://%s:%s@%s:9000/default", clickhouseUser, clickhousePassword, clickhouseHostname)
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
	ClickhouseUser                   string
	ClickhousePassword               string
	MinReplicaCount                  int
	MaxReplicaCount                  int
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
        image: clickhouse/clickhouse-server:26.4.3.37
        imagePullPolicy: Always
        ports:
        - containerPort: 9000
          name: native
        - containerPort: 8123
          name: http
        env:
        - name: CLICKHOUSE_USER
          value: {{.ClickhouseUser}}
        - name: CLICKHOUSE_PASSWORD
          value: {{.ClickhousePassword}}
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
)

func query(t *testing.T, kc *kubernetes.Clientset, query string) {
	pods, err := kc.CoreV1().Pods(testNamespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: "app=clickhouse",
	})
	require.NoError(t, err, "failed to list ClickHouse pods")
	require.NotEmpty(t, pods.Items, "no ClickHouse pods found")
	clickhousePodName := pods.Items[0].Name
	command := fmt.Sprintf(
		"clickhouse client --user %s --password %s -q '%s'",
		clickhouseUser, clickhousePassword, query,
	)
	ok, out, errOut, err := WaitForSuccessfulExecCommandOnSpecificPod(t, clickhousePodName, testNamespace, command, 60, 3)
	require.True(t, ok, "executing SQL %s on ClickHouse pod should work; Output: %s, ErrorOutput: %s, Error: %v", query, out, errOut, err)
}

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

	query(t, kc, "CREATE TABLE default.task_instance (id UInt64) ENGINE = MergeTree() ORDER BY id")

	KubectlApplyMultipleWithTemplate(t, data, testTemplates)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d", minReplicaCount)

	testActivation(t, kc)
	testScaleOut(t, kc)
	testScaleIn(t, kc)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing activation ---")
	query(t, kc, "INSERT INTO default.task_instance VALUES (1),(2),(3)")
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")
	query(t, kc, "INSERT INTO default.task_instance VALUES (4),(5),(6)")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 120, 3),
		"replica count should be %d", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")
	query(t, kc, "TRUNCATE TABLE default.task_instance")
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
		ClickhouseUser:                   clickhouseUser,
		ClickhousePassword:               clickhousePassword,
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
