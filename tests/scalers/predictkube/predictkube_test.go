//go:build e2e
// +build e2e

package predictkube_test

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
	prometheus "github.com/kedacore/keda/v2/tests/scalers/prometheus"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "predictkube-test"
)

var (
	testNamespace        = fmt.Sprintf("%s-ns", testName)
	deploymentName       = fmt.Sprintf("%s-deployment", testName)
	monitoredAppName     = fmt.Sprintf("%s-monitored-app", testName)
	triggerAuthName      = fmt.Sprintf("%s-ta", testName)
	scaledObjectName     = fmt.Sprintf("%s-so", testName)
	secretName           = fmt.Sprintf("%s-secret", testName)
	prometheusServerName = fmt.Sprintf("%s-server", testName)
	predictkubeAPIKey    = os.Getenv("PREDICTKUBE_API_KEY")
	minReplicaCount      = 0
	maxReplicaCount      = 2
)

type templateData struct {
	TestNamespace        string
	DeploymentName       string
	MonitoredAppName     string
	ScaledObjectName     string
	TriggerAuthName      string
	SecretName           string
	PrometheusServerName string
	PredictkubeAPIKey    string
	MinReplicaCount      int
	MaxReplicaCount      int
}

const (
	deploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: test-app
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: {{.DeploymentName}}
  template:
    metadata:
      labels:
        app: {{.DeploymentName}}
        type: keda-testing
    spec:
      containers:
      - name: prom-test-app
        image: tbickford/simple-web-app-prometheus:a13ade9
        imagePullPolicy: IfNotPresent
---
`

	monitoredAppDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: {{.MonitoredAppName}}
  name: {{.MonitoredAppName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.MonitoredAppName}}
  template:
    metadata:
      labels:
        app: {{.MonitoredAppName}}
        type: keda-testing
    spec:
      containers:
      - name: prom-test-app
        image: tbickford/simple-web-app-prometheus:a13ade9
        imagePullPolicy: IfNotPresent
---
`

	monitoredAppServiceTemplate = `apiVersion: v1
kind: Service
metadata:
  labels:
    app: {{.MonitoredAppName}}
  name: {{.MonitoredAppName}}
  namespace: {{.TestNamespace}}
  annotations:
    prometheus.io/scrape: "true"
spec:
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: 8080
  selector:
    app: {{.MonitoredAppName}}
`

	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  apiKey: {{.PredictkubeAPIKey}}
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
  pollingInterval: 3
  cooldownPeriod:  1
  triggers:
  - type: predictkube
    metadata:
      predictHorizon: "2h"
      historyTimeWindow: "7d"
      prometheusAddress: http://{{.PrometheusServerName}}.{{.TestNamespace}}.svc
      threshold: '100'
      activationThreshold: '50'
      query: sum(rate(http_requests_total{app="{{.MonitoredAppName}}"}[2m]))
      queryStep: "2m"
    authenticationRef:
      name: {{.TriggerAuthName}}
`

	generateHeavyLoadJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: generate-heavy-requests
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 0
  template:
    spec:
      containers:
      - image: jordi/ab
        name: test
        command: ["/bin/sh"]
        args: ["-c", "for i in $(seq 1 45);do echo $i;ab -c 5 -n 1000 -v 2 http://{{.MonitoredAppName}}.{{.TestNamespace}}.svc/;sleep 1;done"]
      restartPolicy: Never
  activeDeadlineSeconds: 120
  backoffLimit: 2
  `

	generateLightLoadJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: generate-light-requests
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 0
  template:
    spec:
      containers:
      - image: jordi/ab
        name: test
        command: ["/bin/sh"]
        args: ["-c", "for i in $(seq 1 45);do echo $i;ab -c 1 -n 10 -v 2 http://{{.MonitoredAppName}}.{{.TestNamespace}}.svc/;sleep 1;done"]
      restartPolicy: Never
  activeDeadlineSeconds: 120
  backoffLimit: 2
    `
)

func TestScaler(t *testing.T) {
	require.NotEmpty(t, predictkubeAPIKey, "PREDICTKUBE_API_KEY env variable is required for predictkube test")

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		prometheus.Uninstall(t, prometheusServerName, testNamespace, nil)
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	// Create kubernetes resources
	prometheus.Install(t, kc, prometheusServerName, testNamespace, nil)

	// Create kubernetes resources for testing
	KubectlApplyMultipleWithTemplate(t, data, templates)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, monitoredAppName, testNamespace, 1, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	KubectlReplaceWithTemplate(t, data, "generateLoadJobTemplate", generateLightLoadJobTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlReplaceWithTemplate(t, data, "generateLoadJobTemplate", generateHeavyLoadJobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 5),
		"replica count should be %d after 5 minutes", minReplicaCount)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:        testNamespace,
			DeploymentName:       deploymentName,
			PredictkubeAPIKey:    base64.StdEncoding.EncodeToString([]byte(predictkubeAPIKey)),
			SecretName:           secretName,
			TriggerAuthName:      triggerAuthName,
			ScaledObjectName:     scaledObjectName,
			MonitoredAppName:     monitoredAppName,
			PrometheusServerName: prometheusServerName,
			MinReplicaCount:      minReplicaCount,
			MaxReplicaCount:      maxReplicaCount,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "monitoredAppDeploymentTemplate", Config: monitoredAppDeploymentTemplate},
			{Name: "monitoredAppServiceTemplate", Config: monitoredAppServiceTemplate},
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
