//go:build e2e
// +build e2e

package helper

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	"github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

// Common for pod and workload identity tests
var (
	PrometheusQueryEndpoint = os.Getenv("TF_AZURE_MANAGED_PROMETHEUS_QUERY_ENDPOINT")
	MinReplicaCount         = 0
	MaxReplicaCount         = 2
)

type TemplateData struct {
	TestNamespace           string
	DeploymentName          string
	MonitoredAppName        string
	PublishDeploymentName   string
	ScaledObjectName        string
	PodIdentityProvider     string
	PrometheusQueryEndpoint string
	MinReplicaCount         int
	MaxReplicaCount         int
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
      app: test-app
  template:
    metadata:
      labels:
        app: test-app
        type: keda-testing
    spec:
      containers:
      - name: prom-test-app
        image: ghcr.io/kedacore/tests-prometheus:latest
        imagePullPolicy: IfNotPresent
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          capabilities:
            drop:
              - ALL
          seccompProfile:
            type: RuntimeDefault
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
        type: {{.MonitoredAppName}}
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
    spec:
      containers:
      - name: prom-test-app
        image: ghcr.io/kedacore/tests-prometheus:latest
        imagePullPolicy: IfNotPresent
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          capabilities:
            drop:
              - ALL
          seccompProfile:
            type: RuntimeDefault
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
    type: {{.MonitoredAppName}}
`

	azureManagedPrometheusAuthTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: azure-managed-prometheus-trigger-auth
  namespace: {{.TestNamespace}}
spec:
  podIdentity:
      provider: {{.PodIdentityProvider}}
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
  - type: prometheus
    metadata:
      serverAddress: {{.PrometheusQueryEndpoint}}
      threshold: '20'
      activationThreshold: '20'
      query: sum(rate(http_requests_total{app="{{.MonitoredAppName}}"}[2m]))
    authenticationRef:
      name: azure-managed-prometheus-trigger-auth
`

	generateLowLevelLoadJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: generate-low-level-requests-job
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-hey
        name: test
        command: ["/bin/sh"]
        args: ["-c", "for i in $(seq 1 60);do echo $i;/hey -c 5 -n 30 http://{{.MonitoredAppName}}.{{.TestNamespace}}.svc;sleep 1;done"]
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          capabilities:
            drop:
              - ALL
          seccompProfile:
            type: RuntimeDefault
      restartPolicy: Never
  activeDeadlineSeconds: 100
  backoffLimit: 2
  `

	generateLoadJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: generate-requests-job
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - image: ghcr.io/kedacore/tests-hey
        name: test
        command: ["/bin/sh"]
        args: ["-c", "for i in $(seq 1 60);do echo $i;/hey -c 5 -n 80 http://{{.MonitoredAppName}}.{{.TestNamespace}}.svc;sleep 1;done"]
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          capabilities:
            drop:
              - ALL
          seccompProfile:
            type: RuntimeDefault
      restartPolicy: Never
  activeDeadlineSeconds: 100
  backoffLimit: 2
`
)

func TestAzureManagedPrometheusScaler(t *testing.T, data TemplateData) {
	require.NotEmpty(t, PrometheusQueryEndpoint, "TF_AZURE_MANAGED_PROMETHEUS_QUERY_ENDPOINT env variable is required for azure managed prometheus tests")

	kc := helper.GetKubernetesClient(t)

	// Create kubernetes resources for testing
	helper.CreateNamespace(t, kc, data.TestNamespace)

	templates := getTemplates()
	helper.KubectlApplyMultipleWithTemplate(t, data, templates)
	assert.True(t, helper.WaitForDeploymentReplicaReadyCount(t, kc, data.MonitoredAppName, data.TestNamespace, 1, 60, 3),
		"replica count should be %d after 3 minutes", MinReplicaCount)
	assert.True(t, helper.WaitForDeploymentReplicaReadyCount(t, kc, data.DeploymentName, data.TestNamespace, MinReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", MinReplicaCount)

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc, data)

	// cleanup
	helper.KubectlDeleteMultipleWithTemplate(t, data, templates)
	helper.DeleteNamespace(t, data.TestNamespace)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data TemplateData) {
	t.Log("--- testing activation ---")
	helper.KubectlReplaceWithTemplate(t, data, "generateLowLevelLoadJobTemplate", generateLowLevelLoadJobTemplate)

	helper.AssertReplicaCountNotChangeDuringTimePeriod(t, kc, data.DeploymentName, data.TestNamespace, MinReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data TemplateData) {
	t.Log("--- testing scale out ---")
	helper.KubectlReplaceWithTemplate(t, data, "generateLoadJobTemplate", generateLoadJobTemplate)

	assert.True(t, helper.WaitForDeploymentReplicaReadyCount(t, kc, data.DeploymentName, data.TestNamespace, MaxReplicaCount, 144, 5),
		"replica count should be %d after 12 minutes", MaxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data TemplateData) {
	t.Log("--- testing scale in ---")
	assert.True(t, helper.WaitForDeploymentReplicaReadyCount(t, kc, data.DeploymentName, data.TestNamespace, MinReplicaCount, 144, 5),
		"replica count should be %d after 12 minutes", MinReplicaCount)
}

func getTemplates() []helper.Template {
	return []helper.Template{
		{Name: "monitoredAppDeploymentTemplate", Config: monitoredAppDeploymentTemplate},
		{Name: "deploymentTemplate", Config: deploymentTemplate},
		{Name: "monitoredAppServiceTemplate", Config: monitoredAppServiceTemplate},
		{Name: "azureManagedPrometheusAuthTemplate", Config: azureManagedPrometheusAuthTemplate},
		{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
	}
}
