//go:build e2e
// +build e2e

package honeycomb_test

import (
	"encoding/base64"
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
	testName = "honeycomb-test"
)

var (
	testNamespace           = fmt.Sprintf("%s-ns", testName)
	deploymentName          = fmt.Sprintf("%s-deployment", testName)
	monitoredDeploymentName = fmt.Sprintf("%s-monitored-deployment", testName)
	serviceName             = fmt.Sprintf("%s-service-%d", testName, GetRandomNumber())
	scaledObjectName        = fmt.Sprintf("%s-so", testName)
	secretName              = fmt.Sprintf("%s-secret", testName)
	triggerAuthName         = fmt.Sprintf("%s-ta", testName)
	honeycombAPIKey         = os.Getenv("HONEYCOMB_API_KEY")
	honeycombDataset        = os.Getenv("HONEYCOMB_DATASET")
	deploymentReplicas      = 1
	minReplicaCount         = 0
	maxReplicaCount         = 2
)

type templateData struct {
	TestNamespace           string
	DeploymentName          string
	MonitoredDeploymentName string
	ServiceName             string
	ScaledObjectName        string
	TriggerAuthName         string
	SecretName              string
	HoneycombAPIKey         string
	HoneycombDataset        string
	DeploymentReplicas      string
	MinReplicaCount         string
	MaxReplicaCount         string
}

const (
	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  honeycombApiKey: {{.HoneycombAPIKey}}
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
    key: honeycombApiKey
`

	monitoredDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.MonitoredDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.MonitoredDeploymentName}}
spec:
  replicas: {{.DeploymentReplicas}}
  selector:
    matchLabels:
      app: {{.MonitoredDeploymentName}}
  template:
    metadata:
      labels:
        app: {{.MonitoredDeploymentName}}
    spec:
      containers:
      - name: prom-test-app
        image: tbickford/simple-web-app-prometheus:a13ade9
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 8080
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
        image: ghcr.io/nginx/nginx-unprivileged:1.26
        ports:
        - containerPort: 80
`

	serviceTemplate = `apiVersion: v1
kind: Service
metadata:
  labels:
    name: {{.ServiceName}}
  annotations:
    prometheus.io/scrape: "true"
  name: {{.ServiceName}}
  namespace: {{.TestNamespace}}
spec:
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: 8080
  selector:
    app: {{.MonitoredDeploymentName}}
`

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  pollingInterval: 5
  cooldownPeriod: 10
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 15
  triggers:
    - type: honeycomb
      metadata:
        dataset: "{{.HoneycombDataset}}"
        threshold: "2"
        activationThreshold: "3"
        calculation: "COUNT"
        timeRange: "300"
        queryRaw: |
          {
            "breakdowns": ["service.name"],
            "calculations": [{"op": "COUNT"}],
            "filters": [
              {
                "column": "service.name",
                "op": "=",
                "value": "{{.ServiceName}}"
              },
              {
                "column": "name",
                "op": "=", 
                "value": "{{.TestNamespace}}"
              }
            ],
            "time_range": 300
          }
        resultField: "COUNT"
      authenticationRef:
        name: {{.TriggerAuthName}}
`

	lightLoadTemplate = `apiVersion: v1
kind: Pod
metadata:
  name: fake-light-traffic
  namespace: {{.TestNamespace}}
spec:
  containers:
  - image: busybox
    name: test
    command: ["/bin/sh"]
    args: ["-c", "while true; do wget -O /dev/null -o /dev/null http://{{.ServiceName}}/; sleep 1; done"]
`

	heavyLoadTemplate = `apiVersion: v1
kind: Pod
metadata:
  name: fake-heavy-traffic
  namespace: {{.TestNamespace}}
spec:
  containers:
  - image: busybox
    name: test
    command: ["/bin/sh"]
    args: ["-c", "while true; do wget -O /dev/null -o /dev/null http://{{.ServiceName}}/; sleep 0.1; done"]
`
)

func TestHoneycombScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	require.NotEmpty(t, honeycombAPIKey, "HONEYCOMB_API_KEY env variable is required for Honeycomb tests")
	require.NotEmpty(t, honeycombDataset, "HONEYCOMB_DATASET env variable is required for Honeycomb tests")

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	// Create kubernetes resources
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// Wait for monitored deployment to be ready
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, monitoredDeploymentName, testNamespace, deploymentReplicas, 60, 1),
		"monitored deployment replica count should be %d after a minute", deploymentReplicas)

	// Ensure test deployment is at min replica count
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 1),
		"replica count should be %d after a minute", minReplicaCount)

	// test scaling
	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc, data)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	KubectlApplyWithTemplate(t, data, "lightLoadTemplate", lightLoadTemplate)

	// With light load, we should not scale out beyond activation threshold
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlApplyWithTemplate(t, data, "heavyLoadTemplate", heavyLoadTemplate)

	// With heavy load, we should scale out to max replicas
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 180, 5),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in ---")
	KubectlDeleteWithTemplate(t, data, "lightLoadTemplate", lightLoadTemplate)
	KubectlDeleteWithTemplate(t, data, "heavyLoadTemplate", heavyLoadTemplate)

	// After removing load, we should scale back down to min replicas
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 180, 5),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

func getTemplateData() (templateData, []Template) {
	// Set default dataset if not provided
	dataset := honeycombDataset
	if dataset == "" {
		dataset = "test-dataset"
	}

	return templateData{
			TestNamespace:           testNamespace,
			DeploymentName:          deploymentName,
			MonitoredDeploymentName: monitoredDeploymentName,
			ServiceName:             serviceName,
			TriggerAuthName:         triggerAuthName,
			ScaledObjectName:        scaledObjectName,
			SecretName:              secretName,
			HoneycombDataset:        dataset,
			MinReplicaCount:         fmt.Sprintf("%v", minReplicaCount),
			MaxReplicaCount:         fmt.Sprintf("%v", maxReplicaCount),
			DeploymentReplicas:      fmt.Sprintf("%v", deploymentReplicas),
			HoneycombAPIKey:         base64.StdEncoding.EncodeToString([]byte(honeycombAPIKey)),
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "monitoredDeploymentTemplate", Config: monitoredDeploymentTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
