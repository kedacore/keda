//go:build e2e
// +build e2e

package ionos_monitoring_test

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
	testName = "ionos-monitoring-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	secretName       = fmt.Sprintf("%s-secret", testName)
	triggerAuthName  = fmt.Sprintf("%s-ta", testName)
	// IONOS_MONITORING_HOST is the httpEndpoint of the IONOS Monitoring pipeline,
	// e.g. https://123456789-metrics.987654321.monitoring.de-txl.ionos.com
	ionosHost   = os.Getenv("IONOS_MONITORING_HOST")
	ionosAPIKey = os.Getenv("IONOS_MONITORING_API_KEY")
	// IONOS_MONITORING_QUERY is a PromQL expression that returns a single scalar.
	ionosQuery      = os.Getenv("IONOS_MONITORING_QUERY")
	minReplicaCount = 0
	maxReplicaCount = 2
)

type templateData struct {
	TestNamespace    string
	DeploymentName   string
	ScaledObjectName string
	TriggerAuthName  string
	SecretName       string
	IONOSHost        string
	IONOSAPIKey      string
	IONOSQuery       string
	MinReplicaCount  string
	MaxReplicaCount  string
}

const (
	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  apiKey: {{.IONOSAPIKey}}
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: apiKey
    name: {{.SecretName}}
    key: apiKey
`

	deploymentTemplate = `apiVersion: apps/v1
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
        image: nginxinc/nginx-unprivileged
`

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  pollingInterval: 5
  cooldownPeriod: 10
  triggers:
  - type: ionos-monitoring
    metadata:
      host: {{.IONOSHost}}
      query: "{{.IONOSQuery}}"
      threshold: "1"
    authenticationRef:
      name: {{.TriggerAuthName}}
`
)

func TestIONOSMonitoringScaler(t *testing.T) {
	require.NotEmpty(t, ionosHost, "IONOS_MONITORING_HOST env variable is required for e2e tests")
	require.NotEmpty(t, ionosAPIKey, "IONOS_MONITORING_API_KEY env variable is required for e2e tests")
	require.NotEmpty(t, ionosQuery, "IONOS_MONITORING_QUERY env variable is required for e2e tests")

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateNamespace(t, kc, testNamespace)
	defer DeleteNamespace(t, kc, testNamespace)

	KubectlApplyMultipleWithTemplate(t, data, templates)
	defer KubectlDeleteMultipleWithTemplate(t, data, templates)

	testActivation(t, kc)
	testScaleOut(t, kc)
	testScaleIn(t, kc)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing activation ---")
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after scale out", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after scale in", minReplicaCount)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:    testNamespace,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
			TriggerAuthName:  triggerAuthName,
			SecretName:       secretName,
			IONOSHost:        ionosHost,
			IONOSAPIKey:      ionosAPIKey,
			IONOSQuery:       ionosQuery,
			MinReplicaCount:  fmt.Sprintf("%v", minReplicaCount),
			MaxReplicaCount:  fmt.Sprintf("%v", maxReplicaCount),
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
