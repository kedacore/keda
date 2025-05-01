//go:build e2e
// +build e2e

package trigger_update_so_test

import (
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "trigger-update-so-test"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

var (
	namespace                   = fmt.Sprintf("%s-ns", testName)
	deploymentName              = fmt.Sprintf("%s-deployment", testName)
	workloadDeploymentName      = fmt.Sprintf("%s-workload-deployment", testName)
	scaledObjectName            = fmt.Sprintf("%s-so", testName)
	secretName                  = fmt.Sprintf("%s-secret", testName)
	triggerAuthName             = fmt.Sprintf("%s-ta", testName)
	serviceName                 = fmt.Sprintf("%s-service", testName)
	metricsServerDeploymentName = fmt.Sprintf("%s-metrics-server", testName)
	metricsServerEndpoint       = fmt.Sprintf("http://%s.%s.svc.cluster.local:8080/api/value", serviceName, namespace)
	minReplicas                 = 0
	midReplicas                 = 3
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
	WorkloadDeploymentName      string
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

	workloadDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.WorkloadDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    deploy: {{.WorkloadDeploymentName}}
spec:
  replicas: 0
  selector:
    matchLabels:
      pod: {{.WorkloadDeploymentName}}
  template:
    metadata:
      labels:
        pod: {{.WorkloadDeploymentName}}
    spec:
      containers:
        - name: nginx
          image: 'ghcr.io/nginx/nginx-unprivileged:1.26'`

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
          stabilizationWindowSeconds: 10
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

	scaledObjectTwoTriggerTemplate = `
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
          stabilizationWindowSeconds: 10
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
  - type: kubernetes-workload
    metadata:
      podSelector: "pod={{.WorkloadDeploymentName}}"
      value: '1'
`

	scaledObjectThreeTriggerTemplate = `
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
          stabilizationWindowSeconds: 10
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
  - type: kubernetes-workload
    metadata:
      podSelector: 'pod={{.WorkloadDeploymentName}}'
      value: '1'
  - type: metrics-api
    metadata:
      targetValue: "2"
      url: "invalid-invalid"
      valueLocation: 'value'
      method: "query"
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

func TestScaledObjectGeneral(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, namespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, minReplicas, 180, 3),
		"replica count should be %d after 3 minutes", minReplicas)

	testTargetValue(t, kc, data)          // one trigger target changes
	testTwoTriggers(t, kc, data)          // add trigger during active scaling
	testRemoveTrigger(t, kc, data)        // remove trigger during active scaling
	testThreeTriggersWithCPU(t, kc, data) // three triggers

	DeleteKubernetesResources(t, namespace, data, templates)
}

// tests basic scaling with one trigger based on metrics
func testTargetValue(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- test target value 1 ---")
	KubectlApplyWithTemplate(t, data, "scaledObjectTriggerTemplate", scaledObjectTriggerTemplate)

	data.MetricValue = 1
	KubectlReplaceWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 1, 180, 3),
		"replica count should be %d after 3 minutes", 1)

	t.Log("--- test target value 10 ---")
	data.MetricValue = 10
	KubectlReplaceWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, maxReplicas, 180, 3),
		"replica count should be %d after 3 minutes", maxReplicas)

	t.Log("--- test target value 0 ---")
	data.MetricValue = 0
	KubectlReplaceWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, minReplicas, 180, 3),
		"replica count should be %d after 3 minutes", minReplicas)
}

// test adding second trigger during scaling
func testTwoTriggers(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- test two triggers ---")
	KubectlApplyWithTemplate(t, data, "scaledObjectTriggerTemplate", scaledObjectTriggerTemplate)

	data.MetricValue = 1
	KubectlReplaceWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 1, 180, 3),
		"replica count should be %d after 3 minutes", 1)

	KubectlApplyWithTemplate(t, data, "scaledObjectTwoTriggerTemplate", scaledObjectTwoTriggerTemplate)
	// scale to max with k8s wl = second trigger
	KubernetesScaleDeployment(t, kc, workloadDeploymentName, int64(maxReplicas), namespace)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, maxReplicas, 180, 3),
		"replica count should be %d after 3 minutes", maxReplicas)
}

// scales to max with kubernetes worload(second trigger), removes it and
// scales to 3 replicas based on metric value (first trigger)
func testRemoveTrigger(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- test remove trigger 2 -> 1 ---")
	KubectlApplyWithTemplate(t, data, "scaledObjectTwoTriggerTemplate", scaledObjectTwoTriggerTemplate)
	data.MetricValue = 5 // 3 replicas (midReplicas)
	KubectlReplaceWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	KubernetesScaleDeployment(t, kc, workloadDeploymentName, int64(maxReplicas), namespace)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, maxReplicas, 180, 3),
		"replica count should be %d after 3 minutes", maxReplicas)

	// update SO -> remove k8s wl == second trigger
	KubectlApplyWithTemplate(t, data, "scaledObjectTriggerTemplate", scaledObjectTriggerTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, midReplicas, 180, 3),
		"replica count should be %d after 3 minutes", midReplicas)
}

// test 3 triggers scaling works including one cpu metric
func testThreeTriggersWithCPU(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- test 3 triggers (with cpu) ---")

	// update SO should scale up based on cpu
	KubectlApplyWithTemplate(t, data, "scaledObjectThreeTriggerTemplate", scaledObjectThreeTriggerTemplate)

	// scaling might take longer because of fetching of the cpu metrics (possibly increase iterations if needed)
	data.MetricValue = 10
	KubectlReplaceWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, maxReplicas, 180, 3),
		"replica count should be %d after 3 minutes", maxReplicas)
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
			WorkloadDeploymentName:      workloadDeploymentName,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "workloadDeploymentTemplate", Config: workloadDeploymentTemplate},
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "metricsServerDeploymentTemplate", Config: metricsServerDeploymentTemplate},
			{Name: "scaledObjectTriggerTemplate", Config: scaledObjectTriggerTemplate},
		}
}
