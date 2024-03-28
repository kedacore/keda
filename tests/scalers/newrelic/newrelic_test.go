//go:build e2e
// +build e2e

package newrelic_test

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
	testName = "new-relic-test"
)

var (
	testNamespace           = fmt.Sprintf("%s-ns", testName)
	deploymentName          = fmt.Sprintf("%s-deployment", testName)
	monitoredDeploymentName = fmt.Sprintf("%s-monitored-deployment", testName)
	serviceName             = fmt.Sprintf("%s-service-%d", testName, GetRandomNumber())
	scaledObjectName        = fmt.Sprintf("%s-so", testName)
	secretName              = fmt.Sprintf("%s-secret", testName)
	triggerAuthName         = fmt.Sprintf("%s-ta", testName)
	newRelicAPIKey          = os.Getenv("NEWRELIC_API_KEY")
	newRelicLicenseKey      = os.Getenv("NEWRELIC_LICENSE")
	newRelicRegion          = os.Getenv("NEWRELIC_REGION")
	newRelicAccountID       = os.Getenv("NEWRELIC_ACCOUNT_ID")
	kuberneteClusterName    = "keda-new-relic-cluster"
	newRelicHelmRepoURL     = "https://helm-charts.newrelic.com"
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
	NewRelicAPIKey          string
	DeploymentReplicas      string
	NewRelicRegion          string
	NewRelicAccountID       string
	KuberneteClusterName    string
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
  newRelicApiKey: {{.NewRelicAPIKey}}
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: queryKey           # Required.
    name: {{.SecretName}}         # Required.
    key: newRelicApiKey           # Required.
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
    - type: new-relic
      metadata:
        account: "{{.NewRelicAccountID}}"
        region: {{.NewRelicRegion}}
        threshold: "2"
        activationThreshold: "3"
        nrql: SELECT rate(sum(http_requests_total), 1 SECONDS) FROM Metric where serviceName='{{.ServiceName}}' and namespaceName='{{.TestNamespace}}' since 120 seconds ago
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

func TestNewRelicScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	require.NotEmpty(t, newRelicAPIKey, "NEWRELIC_API_KEY env variable is required for new relic tests")
	require.NotEmpty(t, newRelicLicenseKey, "NEWRELIC_LICENSE env variable is required for new relic tests")
	require.NotEmpty(t, newRelicAccountID, "NEWRELIC_ACCOUNT_ID env variable is required for new relic tests")

	if len(newRelicRegion) == 0 {
		newRelicRegion = "EU"
	}

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	// Create kubernetes resources
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	installNewRelic(t)

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

func installNewRelic(t *testing.T) {
	_, err := ExecuteCommand(fmt.Sprintf("helm repo add new-relic %s", newRelicHelmRepoURL))
	require.NoErrorf(t, err, "cannot execute command - %s", err)
	_, err = ExecuteCommand("helm repo update")
	require.NoErrorf(t, err, "cannot execute command - %s", err)

	cmd := fmt.Sprintf(`helm upgrade --install --set global.cluster=%s --set prometheus.enabled=true --set ksm.enabled=true --set global.lowDataMode=true --set global.licenseKey=%s --timeout 600s --set logging.enabled=false --set ksm.enabled=true --set logging.enabled=true --namespace %s ri-keda new-relic/nri-bundle`,
		kuberneteClusterName,
		newRelicLicenseKey,
		testNamespace)

	_, err = ExecuteCommand(cmd)
	require.NoErrorf(t, err, "cannot execute command - %s", err)
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
			NewRelicRegion:          newRelicRegion,
			KuberneteClusterName:    kuberneteClusterName,
			MinReplicaCount:         fmt.Sprintf("%v", minReplicaCount),
			MaxReplicaCount:         fmt.Sprintf("%v", maxReplicaCount),
			DeploymentReplicas:      fmt.Sprintf("%v", deploymentReplicas),
			NewRelicAccountID:       newRelicAccountID,
			NewRelicAPIKey:          base64.StdEncoding.EncodeToString([]byte(newRelicAPIKey)),
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "monitoredDeploymentTemplate", Config: monitoredDeploymentTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
