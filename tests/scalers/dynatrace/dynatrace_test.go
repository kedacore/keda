//go:build e2e
// +build e2e

package dynatrace_test

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
	testName = "dynatrace-test"
)

var (
	testNamespace           = fmt.Sprintf("%s-ns", testName)
	deploymentName          = fmt.Sprintf("%s-deployment", testName)
	monitoredDeploymentName = fmt.Sprintf("%s-monitored-deployment", testName)
	serviceName             = fmt.Sprintf("%s-service-%d", testName, GetRandomNumber())
	scaledObjectName        = fmt.Sprintf("%s-so", testName)
	secretName              = fmt.Sprintf("%s-secret", testName)
	triggerAuthName         = fmt.Sprintf("%s-ta", testName)
	dynatraceHost           = os.Getenv("DYNATRACE_HOST")
	dynatraceToken          = os.Getenv("DYNATRACE_METRICS_TOKEN")
	kubernetesClusterName   = "keda-dynatrace-cluster"
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
	DynatraceToken          string
	DeploymentReplicas      string
	DynatraceHost           string
	KubernetesClusterName   string
	MinReplicaCount         string
	MaxReplicaCount         string
}

const (
	dynakubeTemplate = `apiVersion: dynatrace.com/v1beta1
kind: DynaKube
metadata:
name: {{.KubernetesClusterName}}
namespace: {{.TestNamespace}}
spec:
  tokens: {{.SecretName}}
  apiUrl: "{{.DynatraceHost}}/api"
  networkZone: {{.KubernetesClusterName}}
  oneAgent:
    cloudNativeFullStack:
      args:
        - --set-host-group={{.KubernetesClusterName}}
  activeGate:
    capabilities:
    - routing
    - dynatrace-api
    - metrics-ingest
    group: {{.KubernetesClusterName}}
`
	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  apiToken: {{.DynatraceToken}}
  dataIngestToken: {{.DynatraceToken}}
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: token
    name: {{.SecretName}}
    key: apiToken
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
      annotations:
        data-ingest.dynatrace.com/inject: "true"
        dynatrace.com/inject: "true"
        oneagent.dynatrace.com/inject: "true"
      labels:
        app: {{.MonitoredDeploymentName}}
    spec:
      containers:
      - name: prom-test-app
        image: tbickford/simple-web-app-prometheus:a13ade9
        imagePullPolicy: IfNotPresent
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
      - name: prom-test-app
        image: tbickford/simple-web-app-prometheus:a13ade9
        imagePullPolicy: IfNotPresent
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
  pollingInterval: 1
  cooldownPeriod:  1
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 10
  triggers:
    - type: dynatrace
      metadata:
        host: {{.DynatraceHost}}
        threshold: "2"
        activationThreshold: "3"
        metricSelector: "builtin:service.requestCount.total:splitBy():fold"
        from: now-2m
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
    args: ["-c", "while true; do wget -O /dev/null -o /dev/null http://{{.ServiceName}}/; sleep 0.5; done"]`

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
    args: ["-c", "while true; do wget -O /dev/null -o /dev/null http://{{.ServiceName}}/; sleep 0.1; done"]`
)

func TestDynatraceScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	require.NotEmpty(t, dynatraceToken, "DYNATRACE_METRICS_TOKEN env variable is required for dynatrace tests")
	require.NotEmpty(t, dynatraceHost, "DYNATRACE_HOST env variable is required for dynatrace tests")

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	// Create kubernetes resources
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	installDynatrace(t)

	data, templates = getDynatraceTemplateData()
	// Create Dynatrace-specific kubernetes resources
	KubectlApplyMultipleWithTemplate(t, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 1),
		"replica count should be %s after a minute", minReplicaCount)

	// test scaling
	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc, data)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	KubectlApplyWithTemplate(t, data, "lightLoadTemplate", lightLoadTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlApplyWithTemplate(t, data, "heavyLoadTemplate", heavyLoadTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in ---")
	KubectlDeleteWithTemplate(t, data, "lightLoadTemplate", lightLoadTemplate)
	KubectlDeleteWithTemplate(t, data, "heavyLoadTemplate", heavyLoadTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

func installDynatrace(t *testing.T) {
	cmd := fmt.Sprintf(`helm upgrade dynatrace-operator oci://public.ecr.aws/dynatrace/dynatrace-operator --atomic --install --set platform=kubernetes --timeout 600s --namespace %s`,
		testNamespace)

	_, err := ExecuteCommand(cmd)
	require.NoErrorf(t, err, "cannot execute command - %s", err)
}

func getDynatraceTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:         testNamespace,
			SecretName:            secretName,
			DynatraceHost:         dynatraceHost,
			KubernetesClusterName: kubernetesClusterName,
		}, []Template{
			{Name: "dynakubeTemplate", Config: dynakubeTemplate},
		}
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:           testNamespace,
			DeploymentName:          deploymentName,
			MonitoredDeploymentName: monitoredDeploymentName,
			ServiceName:             serviceName,
			TriggerAuthName:         triggerAuthName,
			ScaledObjectName:        scaledObjectName,
			SecretName:              secretName,
			KubernetesClusterName:   kubernetesClusterName,
			MinReplicaCount:         fmt.Sprintf("%v", minReplicaCount),
			MaxReplicaCount:         fmt.Sprintf("%v", maxReplicaCount),
			DeploymentReplicas:      fmt.Sprintf("%v", deploymentReplicas),
			DynatraceToken:          base64.StdEncoding.EncodeToString([]byte(dynatraceToken)),
			DynatraceHost:           dynatraceHost,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "monitoredDeploymentTemplate", Config: monitoredDeploymentTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
