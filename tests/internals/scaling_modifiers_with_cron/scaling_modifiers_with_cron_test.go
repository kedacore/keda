//go:build e2e
// +build e2e

package scaling_modifiers_with_cron_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "scaling-modifiers-with-cron-test"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

var (
	namespace                   = fmt.Sprintf("%s-ns", testName)
	deploymentName              = fmt.Sprintf("%s-deployment", testName)
	metricsServerDeploymentName = fmt.Sprintf("%s-metrics-server", testName)
	serviceName                 = fmt.Sprintf("%s-service", testName)
	triggerAuthName             = fmt.Sprintf("%s-ta", testName)
	scaledObjectName            = fmt.Sprintf("%s-so", testName)
	secretName                  = fmt.Sprintf("%s-secret", testName)
	metricsServerEndpoint       = fmt.Sprintf("http://%s.%s.svc.cluster.local:8080/api/value", serviceName, namespace)
)

type templateData struct {
	TestNamespace               string
	DeploymentName              string
	ScaledObject                string
	TriggerAuthName             string
	SecretName                  string
	ServiceName                 string
	MetricsServerDeploymentName string
	MetricsServerEndpoint       string
	MetricValue                 int
	JobTimestamp                string
}

const (
	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  AUTH_PASSWORD: U0VDUkVUCg==
  AUTH_USERNAME: VVNFUgo=
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: username
      name: {{.SecretName}}
      key: AUTH_USERNAME
    - parameter: password
      name: {{.SecretName}}
      key: AUTH_PASSWORD
`

	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: {{.DeploymentName}}
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    matchLabels:
      app: {{.DeploymentName}}
  replicas: 3
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

	metricsServerDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.MetricsServerDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.MetricsServerDeploymentName}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.MetricsServerDeploymentName}}
  template:
    metadata:
      labels:
        app: {{.MetricsServerDeploymentName}}
    spec:
      containers:
      - name: metrics
        image: ghcr.io/kedacore/tests-metrics-api
        ports:
        - containerPort: 8080
        envFrom:
        - secretRef:
            name: {{.SecretName}}
        imagePullPolicy: Always
        readinessProbe:
          httpGet:
            path: /api/value
            port: 8080
`

	// This ScaledObject uses multiple cron triggers and a metrics-api trigger
	// Formula: if any cron trigger is active (> 0), use max of cron triggers, else use metrics_api_trigger
	// The cron triggers are short term and on specific date so there is little chance of it being used
	soMultiTriggerFormulaTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObject}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 10
    scalingModifiers:
      formula: |
        if (max([cron_trigger_morning, cron_trigger_afternoon, cron_trigger_evening]) > 0) {
          max([cron_trigger_morning, cron_trigger_afternoon, cron_trigger_evening])
        } else {
          metrics_api_trigger
        }
      target: '1'
      metricType: 'AverageValue'
  pollingInterval: 5
  cooldownPeriod: 10
  minReplicaCount: 3
  maxReplicaCount: 30
  triggers:
  - type: metrics-api
    name: metrics_api_trigger
    metadata:
      url: "{{.MetricsServerEndpoint}}"
      valueLocation: 'value'
      method: "query"
    metricType: "Value"
    authenticationRef:
      name: {{.TriggerAuthName}}
  - type: cron
    name: cron_trigger_morning
    metadata:
      timezone: UTC
      start: 0 8 25 8 *
      end: 1 8 25 8 *
      desiredReplicas: "10"
    metricType: "Value"
  - type: cron
    name: cron_trigger_afternoon
    metadata:
      timezone: UTC
      start: 0 16 25 8 *
      end: 1 16 25 8 *
      desiredReplicas: "15"
    metricType: "Value"
  - type: cron
    name: cron_trigger_evening
    metadata:
      timezone: UTC
      start: 0 20 25 8 *
      end: 1 20 25 8 *
      desiredReplicas: "20"
    metricType: "Value"
`

	updateMetricsTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: update-ms-value-{{.JobTimestamp}}
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 10
  backoffLimit: 4
  template:
    spec:
      containers:
      - name: job-curl
        image: docker.io/curlimages/curl
        imagePullPolicy: Always
        command: ["curl", "-X", "POST", "{{.MetricsServerEndpoint}}/{{.MetricValue}}"]
      restartPolicy: OnFailure
`

	serviceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: {{.ServiceName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    app: {{.MetricsServerDeploymentName}}
  ports:
  - port: 8080
    targetPort: 8080
`
)

func TestScalingModifiersMultiTrigger(t *testing.T) {
	// setup
	t.Log("=== Setting up test environment ===")
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, namespace, data, templates)

	// Ensure metrics API server is ready
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, metricsServerDeploymentName, namespace, 1, 60, 2),
		"metrics server replica count should be 1 after 2 minutes")

	testMultiTriggerFormula(t, kc, data)

	// cleanup
	DeleteKubernetesResources(t, namespace, data, templates)
}

func testMultiTriggerFormula(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	// Apply ScaledObject with multi-trigger formula
	KubectlApplyWithTemplate(t, data, "soMultiTriggerFormulaTemplate", soMultiTriggerFormulaTemplate)

	t.Log("Test 1")
	t.Log("Setting metrics_api_trigger = 6")
	t.Log("Expected: 6 replicas (6 / target 1 = 6)")

	data.MetricValue = 6
	data.JobTimestamp = fmt.Sprintf("%d", time.Now().Unix())
	KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 6, 60, 2),
		"replica count should be 6 after 2 minute")

	t.Log("Test 2")
	t.Log("Change metrics_api_trigger value to 7")
	t.Log("Expected: 7 replicas (7 / target 1 = 7)")

	data.MetricValue = 7
	data.JobTimestamp = fmt.Sprintf("%d", time.Now().Unix())
	KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 7, 60, 2),
		"replica count should be 7 after 2 minute")

	t.Log("Test 3")
	t.Log("Change metrics_api_trigger value to 1, should respect minReplicaCount")
	t.Log("Expected: 3 replicas")

	data.MetricValue = 1
	data.JobTimestamp = fmt.Sprintf("%d", time.Now().Unix())
	KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 3, 60, 2),
		"replica count should be 3 after 2 minute")

	t.Log("Test 4")
	t.Log("Change metrics_api_trigger value to 8")
	t.Log("Expected: 8 replicas")

	data.MetricValue = 8
	data.JobTimestamp = fmt.Sprintf("%d", time.Now().Unix())
	KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 8, 60, 2),
		"replica count should be 8 after 2 minute")

	t.Log("Test 5")
	t.Log("Setting metrics_api_trigger = 0")
	t.Log("Expected: 3 replicas (minReplicaCount)")

	data.MetricValue = 0
	data.JobTimestamp = fmt.Sprintf("%d", time.Now().Unix())
	KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 3, 60, 2),
		"replica count should be 3 after 2 minute")
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:               namespace,
			DeploymentName:              deploymentName,
			MetricsServerDeploymentName: metricsServerDeploymentName,
			ServiceName:                 serviceName,
			TriggerAuthName:             triggerAuthName,
			ScaledObject:                scaledObjectName,
			SecretName:                  secretName,
			MetricsServerEndpoint:       metricsServerEndpoint,
			MetricValue:                 0,
			JobTimestamp:                fmt.Sprintf("%d", time.Now().Unix()),
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "metricsServerDeploymentTemplate", Config: metricsServerDeploymentTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "soMultiTriggerFormulaTemplate", Config: soMultiTriggerFormulaTemplate},
		}
}
