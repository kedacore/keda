//go:build e2e
// +build e2e

package replica_update_so_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "replica-update-so-test"
)

var (
	namespace                   = fmt.Sprintf("%s-ns", testName)
	deploymentName              = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName            = fmt.Sprintf("%s-so", testName)
	secretName                  = fmt.Sprintf("%s-secret", testName)
	triggerAuthName             = fmt.Sprintf("%s-ta", testName)
	serviceName                 = fmt.Sprintf("%s-service", testName)
	metricsServerDeploymentName = fmt.Sprintf("%s-metrics-server", testName)
	metricsServerEndpoint       = fmt.Sprintf("http://%s.%s.svc.cluster.local:8080/api/value", serviceName, namespace)
	minReplicas                 = 0
	maxReplicas                 = 5
)

type templateData struct {
	TestNamespace               string
	DeploymentName              string
	ScaledObject                string
	TriggerAuthName             string
	ServiceName                 string
	SecretName                  string
	MinReplicas                 string
	MaxReplicas                 string
	MetricsServerDeploymentName string
	MetricsServerEndpoint       string
	MetricValue                 int
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
    deploy: {{.DeploymentName}}
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    matchLabels:
      app: {{.DeploymentName}}
  replicas: {{.MinReplicas}}
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
`

	scaledObjectTriggerTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObject}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 0
  pollingInterval: 10
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: {{.MinReplicas}}
  maxReplicaCount: {{.MaxReplicas}}
  cooldownPeriod: 1
  triggers:
  - type: metrics-api
    metadata:
      targetValue: "2"
      url: "{{.MetricsServerEndpoint}}"
      valueLocation: 'value'
      method: "query"
    authenticationRef:
      name: {{.TriggerAuthName}}
`

	updateMetricTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: update-ms-value
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 0
  template:
    spec:
      containers:
      - name: job-curl
        image: curlimages/curl
        imagePullPolicy: Always
        command: ["curl", "-X", "POST", "{{.MetricsServerEndpoint}}/{{.MetricValue}}"]
      restartPolicy: Never
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, namespace, data, templates)

	scaleMaxReplicasUp(t, kc, data)
	scaleMaxReplicasDown(t, kc, data)
	scaleMinReplicasUpFromZero(t, kc, data)
	scaleMinReplicasDownToZero(t, kc, data)

	// cleanup
	DeleteKubernetesResources(t, namespace, data, templates)
}

// expect replicas to scale up because maxReplicas was updated
func scaleMaxReplicasUp(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- scale up after MaxReplicas change ---")
	data.MetricValue = 100
	KubectlReplaceWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	KubectlApplyWithTemplate(t, data, "scaledObjectTriggerTemplate", scaledObjectTriggerTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, maxReplicas, 180, 3),
		"replica count should be %d after 3 minutes", maxReplicas)

	updatedMaxReplicas := maxReplicas + 10
	data.MaxReplicas = strconv.Itoa(updatedMaxReplicas)
	KubectlApplyWithTemplate(t, data, "scaledObjectTriggerTemplate", scaledObjectTriggerTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, updatedMaxReplicas, 180, 3),
		"replica count should be %d after 3 minutes", updatedMaxReplicas)
}

// expect replicas to decrease because maxReplicas was updated
func scaleMaxReplicasDown(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- scale max replicas down ---")
	data.MetricValue = 100
	KubectlReplaceWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	updatedMaxReplicas := maxReplicas + 10
	data.MaxReplicas = strconv.Itoa(updatedMaxReplicas)
	KubectlApplyWithTemplate(t, data, "scaledObjectTriggerTemplate", scaledObjectTriggerTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, updatedMaxReplicas, 180, 3),
		"replica count should be %d after 3 minutes", updatedMaxReplicas)

	data.MaxReplicas = strconv.Itoa(maxReplicas)
	KubectlApplyWithTemplate(t, data, "scaledObjectTriggerTemplate", scaledObjectTriggerTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, maxReplicas, 180, 3),
		"replica count should be %d after 3 minutes", maxReplicas)
}

// starts with minReplicas 0 -> update to higher, expect to scale up
func scaleMinReplicasUpFromZero(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- scale min replicas up from zero ---")
	data.MetricValue = 0
	KubectlReplaceWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	KubectlApplyWithTemplate(t, data, "scaledObjectTriggerTemplate", scaledObjectTriggerTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, minReplicas, 180, 3),
		"replica count should be %d after 3 minutes", minReplicas)

	updatedMinReplicas := minReplicas + 5
	data.MinReplicas = strconv.Itoa(updatedMinReplicas)
	KubectlApplyWithTemplate(t, data, "scaledObjectTriggerTemplate", scaledObjectTriggerTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, updatedMinReplicas, 180, 3),
		"replica count should be %d after 3 minutes", updatedMinReplicas)
}

// starts with 5 replicas as minReplicas -> update to 0, expect to scale down
func scaleMinReplicasDownToZero(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- scale min replicas down to zero ---")
	data.MetricValue = 0
	KubectlReplaceWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	// set minReplicas to higher number at first
	updatedMinReplicas := minReplicas + 5
	data.MinReplicas = strconv.Itoa(updatedMinReplicas)
	KubectlApplyWithTemplate(t, data, "scaledObjectTriggerTemplate", scaledObjectTriggerTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, updatedMinReplicas, 180, 3),
		"replica count should be %d after 3 minutes", updatedMinReplicas)

	// change minReplicas to default (0)
	data.MinReplicas = strconv.Itoa(minReplicas)
	KubectlApplyWithTemplate(t, data, "scaledObjectTriggerTemplate", scaledObjectTriggerTemplate)

	// check it scales down to 0
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, minReplicas, 180, 3),
		"replica count should be %d after 3 minutes", minReplicas)
}

// help function to load template data
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
			MinReplicas:                 fmt.Sprintf("%v", minReplicas),
			MaxReplicas:                 fmt.Sprintf("%v", maxReplicas),
			MetricValue:                 0,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "metricsServerDeploymentTemplate", Config: metricsServerDeploymentTemplate},
			{Name: "scaledObjectTriggerTemplate", Config: scaledObjectTriggerTemplate},
		}
}
