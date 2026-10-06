//go:build e2e
// +build e2e

package polling_irrelevant_so_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "polling-irrelevant-so-test"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

// This test covers the configuration in which pollingInterval is not relevant
// (minReplicaCount > 0, no idleReplicaCount, no useCachedMetrics): the scale loop
// derives the trigger state from the metrics the HPA-driven metrics path observes
// instead of querying the trigger source itself. The replica counts are chosen so
// that every phase has its own distinct outcome: 1 (min, inactive), 2 (scaled out
// on the metric), 3 (fallback while the metrics server is down).
var (
	namespace                   = fmt.Sprintf("%s-ns", testName)
	deploymentName              = fmt.Sprintf("%s-deployment", testName)
	metricsServerDeploymentName = fmt.Sprintf("%s-metrics-server", testName)
	serviceName                 = fmt.Sprintf("%s-service", testName)
	triggerAuthName             = fmt.Sprintf("%s-ta", testName)
	scaledObjectName            = fmt.Sprintf("%s-so", testName)
	secretName                  = fmt.Sprintf("%s-secret", testName)
	metricsServerEndpoint       = fmt.Sprintf("http://%s.%s.svc.cluster.local:8080/api/value", serviceName, namespace)
	minReplicas                 = 1
	maxReplicas                 = 3
	activeReplicas              = 2
	fallbackReplicas            = 3
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
	MetricsServerReplicas       int
	MinReplicas                 string
	MaxReplicas                 string
	FallbackReplicas            string
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
  replicas: {{.MetricsServerReplicas}}
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

	scaledObjectTemplate = `
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
          stabilizationWindowSeconds: 1
  minReplicaCount: {{.MinReplicas}}
  maxReplicaCount: {{.MaxReplicas}}
  fallback:
    failureThreshold: 3
    replicas: {{.FallbackReplicas}}
  triggers:
  - type: metrics-api
    metricType: AverageValue
    metadata:
      targetValue: "1"
      url: "{{.MetricsServerEndpoint}}"
      valueLocation: 'value'
      method: "query"
    authenticationRef:
      name: {{.TriggerAuthName}}
`

	updateMetricsTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: update-ms-value
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 0
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
)

func TestPollingIrrelevant(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, namespace, data, templates)

	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	_, err := WaitForHpaCreation(t, kc, fmt.Sprintf("keda-hpa-%s", scaledObjectName), namespace, 60, 2)
	assert.NoError(t, err)

	testScaleOut(t, kc, data)
	testScaleIn(t, kc, data)
	testFallback(t, kc, data)
	testRestoreAfterFallback(t, kc, data)

	DeleteKubernetesResources(t, namespace, data, templates)
}

// the deployment scales out on the metric even though the scale loop no longer
// queries the trigger source itself, and the ScaledObject reports Active
func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")

	setMetricValue(t, kc, data, activeReplicas)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, activeReplicas, 18, 10),
		"replica count should be %d after 3 minutes", activeReplicas)
	assert.True(t, waitForScaledObjectCondition(t, "Active is True", func(so *kedav1alpha1.ScaledObject) bool {
		active := so.Status.Conditions.GetActiveCondition()
		return active.IsTrue()
	}), "ScaledObject should report an active trigger")
}

// the deployment scales back in to minReplicaCount and the ScaledObject reports
// inactive; the activity is derived from the metrics the HPA observed
func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in ---")

	setMetricValue(t, kc, data, 0)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, minReplicas, 18, 10),
		"replica count should be %d after 3 minutes", minReplicas)
	assert.True(t, waitForScaledObjectCondition(t, "Active is False", func(so *kedav1alpha1.ScaledObject) bool {
		active := so.Status.Conditions.GetActiveCondition()
		return active.IsFalse()
	}), "ScaledObject should report no active trigger")
}

// with the metrics server gone every metric query fails, so after failureThreshold
// failures the fallback replicas must be applied and reported
func testFallback(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing fallback ---")

	data.MetricsServerReplicas = 0
	KubectlApplyWithTemplate(t, data, "metricsServerDeploymentTemplate", metricsServerDeploymentTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, fallbackReplicas, 18, 10),
		"replica count should be %d after 3 minutes", fallbackReplicas)
	assert.True(t, waitForScaledObjectCondition(t, "Fallback is True", func(so *kedav1alpha1.ScaledObject) bool {
		fallback := so.Status.Conditions.GetFallbackCondition()
		return fallback.IsTrue()
	}), "ScaledObject should report active fallback")

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, namespace, fallbackReplicas, 30)
}

// restoring the metrics server ends the fallback and the deployment settles back
// on minReplicaCount
func testRestoreAfterFallback(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing restore after fallback ---")

	data.MetricsServerReplicas = 1
	KubectlApplyWithTemplate(t, data, "metricsServerDeploymentTemplate", metricsServerDeploymentTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, metricsServerDeploymentName, namespace, 1, 18, 10),
		"metrics server should be back after 3 minutes")

	setMetricValue(t, kc, data, 0)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, minReplicas, 18, 10),
		"replica count should be %d after 3 minutes", minReplicas)
	assert.True(t, waitForScaledObjectCondition(t, "Fallback is False", func(so *kedav1alpha1.ScaledObject) bool {
		fallback := so.Status.Conditions.GetFallbackCondition()
		return fallback.IsFalse()
	}), "ScaledObject should report no active fallback")
}

func setMetricValue(t *testing.T, kc *kubernetes.Clientset, data templateData, value int) {
	data.MetricValue = value
	KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	// wait some seconds to finish the job
	WaitForJobCount(t, kc, namespace, 0, 15, 2)
}

func waitForScaledObjectCondition(t *testing.T, description string, check func(*kedav1alpha1.ScaledObject) bool) bool {
	const iterations, intervalSeconds = 30, 2
	kedaClient := GetKedaKubernetesClient(t)
	for i := 0; i < iterations; i++ {
		so, err := kedaClient.ScaledObjects(namespace).Get(t.Context(), scaledObjectName, metav1.GetOptions{})
		if err == nil && check(so) {
			return true
		}
		t.Logf("Waiting for ScaledObject condition %q... (%d/%d)", description, i+1, iterations)
		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}
	return false
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
			MetricsServerReplicas:       1,
			MinReplicas:                 fmt.Sprintf("%v", minReplicas),
			MaxReplicas:                 fmt.Sprintf("%v", maxReplicas),
			FallbackReplicas:            fmt.Sprintf("%v", fallbackReplicas),
			MetricValue:                 0,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "metricsServerDeploymentTemplate", Config: metricsServerDeploymentTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
		}
}
