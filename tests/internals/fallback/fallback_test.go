//go:build e2e
// +build e2e

package fallback_test

import (
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "fallback-test"
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
	scaledObjectNameBehavior    = fmt.Sprintf("%s-behavior-so", testName)
	secretName                  = fmt.Sprintf("%s-secret", testName)
	metricsServerEndpoint       = fmt.Sprintf("http://%s.%s.svc.cluster.local:8080/api/value", serviceName, namespace)
	minReplicas                 = 0
	maxReplicas                 = 5
	defaultFallback             = 3
)

type templateData struct {
	Namespace                   string
	DeploymentName              string
	ScaledObject                string
	ScaledObjectNameBehavior    string
	TriggerAuthName             string
	ServiceName                 string
	MetricsServerDeploymentName string
	MetricsServerEndpoint       string
	MinReplicas                 string
	MaxReplicas                 string
	DefaultFallback             int
	MetricValue                 int
	SecretName                  string
}

const (
	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.Namespace}}
data:
  AUTH_PASSWORD: U0VDUkVUCg==
  AUTH_USERNAME: VVNFUgo=
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TriggerAuthName}}
  namespace: {{.Namespace}}
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
  namespace: {{.Namespace}}
spec:
  selector:
    matchLabels:
      app: {{.DeploymentName}}
  replicas: 0
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
  namespace: {{.Namespace}}
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

	fallbackMSDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.MetricsServerDeploymentName}}
  namespace: {{.Namespace}}
  labels:
    app: {{.MetricsServerDeploymentName}}
spec:
  replicas: 0
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

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObject}}
  namespace: {{.Namespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: {{.MinReplicas}}
  maxReplicaCount: {{.MaxReplicas}}
  fallback:
    failureThreshold: 3
    replicas: {{.DefaultFallback}}
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 1
  cooldownPeriod: 1
  pollingInterval: 5
  triggers:
  - type: metrics-api
    metricType: AverageValue
    metadata:
      targetValue: "5"
      url: "{{.MetricsServerEndpoint}}"
      valueLocation: 'value'
      method: "query"
    authenticationRef:
      name: {{.TriggerAuthName}}
`

	scaledObjectTemplateWithoutMetricType = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObject}}
  namespace: {{.Namespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: {{.MinReplicas}}
  maxReplicaCount: {{.MaxReplicas}}
  fallback:
    failureThreshold: 1
    replicas: {{.DefaultFallback}}
    behavior: currentReplicasIfHigher
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 1
  cooldownPeriod: 1
  pollingInterval: 5
  triggers:
  - type: metrics-api
    metadata:
      targetValue: "5"
      url: "{{.MetricsServerEndpoint}}"
      valueLocation: 'value'
      method: "query"
    authenticationRef:
      name: {{.TriggerAuthName}}
`

	scaledObjectTemplateWithCurrentReplicasIfHigher = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectNameBehavior}}
  namespace: {{.Namespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: 0
  maxReplicaCount: 5
  fallback:
    failureThreshold: 1
    replicas: {{.DefaultFallback}}
    behavior: currentReplicasIfHigher
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 1
  cooldownPeriod: 1
  pollingInterval: 5
  triggers:
  - type: metrics-api
    metricType: AverageValue
    metadata:
      targetValue: "5"
      url: "{{.MetricsServerEndpoint}}"
      valueLocation: 'value'
      method: "query"
    metricType: "AverageValue"
    authenticationRef:
      name: {{.TriggerAuthName}}
`

	scaledObjectTemplateWithCurrentReplicasIfLower = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectNameBehavior}}
  namespace: {{.Namespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: 0
  maxReplicaCount: 5
  fallback:
    failureThreshold: 1
    replicas: {{.DefaultFallback}}
    behavior: currentReplicasIfLower
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 1
  cooldownPeriod: 1
  pollingInterval: 5
  triggers:
  - type: metrics-api
    metricType: AverageValue
    metadata:
      targetValue: "5"
      url: "{{.MetricsServerEndpoint}}"
      valueLocation: 'value'
      method: "query"
    metricType: "AverageValue"
    authenticationRef:
      name: {{.TriggerAuthName}}
`

	scaledObjectTemplateWithCurrentReplicas = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectNameBehavior}}
  namespace: {{.Namespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: 0
  maxReplicaCount: 5
  fallback:
    failureThreshold: 1
    replicas: {{.DefaultFallback}}
    behavior: currentReplicas
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 1
  cooldownPeriod: 1
  pollingInterval: 5
  triggers:
  - type: metrics-api
    metricType: AverageValue
    metadata:
      targetValue: "5"
      url: "{{.MetricsServerEndpoint}}"
      valueLocation: 'value'
      method: "query"
    metricType: "AverageValue"
    authenticationRef:
      name: {{.TriggerAuthName}}
`

	scaledObjectTemplateWithStatic = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectNameBehavior}}
  namespace: {{.Namespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: 0
  maxReplicaCount: 5
  fallback:
    failureThreshold: 1
    replicas: {{.DefaultFallback}}
    behavior: static
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 1
  cooldownPeriod: 1
  pollingInterval: 5
  triggers:
  - type: metrics-api
    metricType: AverageValue
    metadata:
      targetValue: "5"
      url: "{{.MetricsServerEndpoint}}"
      valueLocation: 'value'
      method: "query"
    metricType: "AverageValue"
    authenticationRef:
      name: {{.TriggerAuthName}}
`

	updateMetricsTemplate = `
apiVersion: batch/v1
kind: Job
metadata:
  name: update-ms-value
  namespace: {{.Namespace}}
spec:
  ttlSecondsAfterFinished: 30
  template:
    spec:
      containers:
      - name: job-curl
        image: curlimages/curl
        imagePullPolicy: Always
        command: ["curl", "-X", "POST", "{{.MetricsServerEndpoint}}/{{.MetricValue}}"]
      restartPolicy: Never
`

	serviceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: {{.ServiceName}}
  namespace: {{.Namespace}}
spec:
  selector:
    app: {{.MetricsServerDeploymentName}}
  ports:
  - port: 8080
    targetPort: 8080
`
)

func TestFallback(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, namespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, minReplicas, 180, 3),
		"replica count should be %d after 3 minutes", minReplicas)

	testScaleOut(t, kc, data)
	testFallback(t, kc, data)
	testRestoreAfterFallback(t, kc, data)

	DeleteKubernetesResources(t, namespace, data, templates)
}

func TestFallbackWithScaledObjectWithoutMetricType(t *testing.T) {
	// setup
	t.Log("--- setting up ScaledObjectWithoutMetricType test ---")
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	// Replace the default scaledObject template
	for i, tmpl := range templates {
		if tmpl.Name == "scaledObjectTemplate" {
			templates[i].Config = scaledObjectTemplateWithoutMetricType
			break
		}
	}

	CreateKubernetesResources(t, kc, namespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, minReplicas, 180, 3),
		"replica count should be %d after 3 minutes", minReplicas)

	// Scale out to 4 replicas (20 / 5 = 4)
	data.MetricValue = 20
	KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 4, 60, 3),
		"replica count should be 4 after 3 minutes")

	// Stop metrics server to trigger fallback
	KubectlApplyWithTemplate(t, data, "fallbackMSDeploymentTemplate", fallbackMSDeploymentTemplate)

	// Should keep 4 replicas as it's higher than fallback value (3)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 4, 30, 3),
		"replica count should remain at 4 after fallback")

	// Ensure the replica count remains stable
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, namespace, 4, 30)

	DeleteKubernetesResources(t, namespace, data, templates)
}

func TestFallbackWithCurrentReplicasIfHigher(t *testing.T) {
	// setup
	t.Log("--- setting up CurrentReplicasIfHigher test ---")
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	// Replace the default scaledObject template
	for i, tmpl := range templates {
		if tmpl.Name == "scaledObjectTemplate" {
			templates[i].Config = scaledObjectTemplateWithCurrentReplicasIfHigher
			break
		}
	}

	CreateKubernetesResources(t, kc, namespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, minReplicas, 180, 3),
		"replica count should be %d after 3 minutes", minReplicas)

	// Scale out to 4 replicas (20 / 5 = 4)
	data.MetricValue = 20
	KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 4, 60, 3),
		"replica count should be 4 after 3 minutes")

	// Stop metrics server to trigger fallback
	KubectlApplyWithTemplate(t, data, "fallbackMSDeploymentTemplate", fallbackMSDeploymentTemplate)

	// Should keep 4 replicas as it's higher than fallback value (3)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 4, 30, 3),
		"replica count should remain at 4 after fallback")

	// Ensure the replica count remains stable
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, namespace, 4, 30)

	DeleteKubernetesResources(t, namespace, data, templates)
}

func TestFallbackWithCurrentReplicasIfLower(t *testing.T) {
	// setup
	t.Log("--- setting up CurrentReplicasIfLower test ---")
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	// Replace the default scaledObject template
	for i, tmpl := range templates {
		if tmpl.Name == "scaledObjectTemplate" {
			templates[i].Config = scaledObjectTemplateWithCurrentReplicasIfLower
			break
		}
	}

	CreateKubernetesResources(t, kc, namespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, minReplicas, 180, 3),
		"replica count should be %d after 3 minutes", minReplicas)

	// Scale out to 4 replicas (20 / 5 = 4)
	data.MetricValue = 20
	KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 4, 60, 3),
		"replica count should be 4 after 3 minutes")

	// Stop metrics server to trigger fallback
	KubectlApplyWithTemplate(t, data, "fallbackMSDeploymentTemplate", fallbackMSDeploymentTemplate)

	// Should keep fallback value (3) as it's lower than current replicas (4)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 3, 30, 3),
		"replica count should remain at 3 after fallback")

	// Ensure the replica count remains stable
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, namespace, 3, 30)

	DeleteKubernetesResources(t, namespace, data, templates)
}

func TestFallbackWithCurrentReplicas(t *testing.T) {
	// setup
	t.Log("--- setting up CurrentReplicas test ---")
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	// Replace the default scaledObject template
	for i, tmpl := range templates {
		if tmpl.Name == "scaledObjectTemplate" {
			templates[i].Config = scaledObjectTemplateWithCurrentReplicas
			break
		}
	}

	CreateKubernetesResources(t, kc, namespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, minReplicas, 180, 3),
		"replica count should be %d after 3 minutes", minReplicas)

	// Scale out to 4 replicas (20 / 5 = 4)
	data.MetricValue = 20
	KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 4, 60, 3),
		"replica count should be 4 after 3 minutes")

	// Stop metrics server to trigger fallback
	KubectlApplyWithTemplate(t, data, "fallbackMSDeploymentTemplate", fallbackMSDeploymentTemplate)

	// Should keep current replicas (4)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 4, 30, 3),
		"replica count should remain at 4 after fallback")

	// Ensure the replica count remains stable
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, namespace, 4, 30)

	DeleteKubernetesResources(t, namespace, data, templates)
}

func TestFallbackWithStatic(t *testing.T) {
	// setup
	t.Log("--- setting up CurrentReplicas test ---")
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	// Replace the default scaledObject template
	for i, tmpl := range templates {
		if tmpl.Name == "scaledObjectTemplate" {
			templates[i].Config = scaledObjectTemplateWithStatic
			break
		}
	}

	CreateKubernetesResources(t, kc, namespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, minReplicas, 180, 3),
		"replica count should be %d after 3 minutes", minReplicas)

	// Scale out to 4 replicas (20 / 5 = 4)
	data.MetricValue = 20
	KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 4, 60, 3),
		"replica count should be 4 after 3 minutes")

	// Stop metrics server to trigger fallback
	KubectlApplyWithTemplate(t, data, "fallbackMSDeploymentTemplate", fallbackMSDeploymentTemplate)

	// Should keep fallback value (3) because of static
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 3, 30, 3),
		"replica count should remain at 3 after fallback")

	// Ensure the replica count remains stable
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, namespace, 3, 30)

	DeleteKubernetesResources(t, namespace, data, templates)
}

// scale out to max replicas first
func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	data.MetricValue = 50
	KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, maxReplicas, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicas)
}

// MS replicas set to 0 to envoke fallback
func testFallback(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing fallback ---")
	KubectlApplyWithTemplate(t, data, "fallbackMSDeploymentTemplate", fallbackMSDeploymentTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, defaultFallback, 60, 3),
		"replica count should be %d after 3 minutes", defaultFallback)
	// We need to ensure that the fallback value is stable to cover this regression
	// https://github.com/kedacore/keda/issues/4249
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, namespace, defaultFallback, 180)
}

// restore MS to scale back from fallback replicas
func testRestoreAfterFallback(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing after fallback ---")
	KubectlApplyWithTemplate(t, data, "metricsServerDeploymentTemplate", metricsServerDeploymentTemplate)
	data.MetricValue = 50
	KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, maxReplicas, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicas)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			Namespace:                   namespace,
			DeploymentName:              deploymentName,
			MetricsServerDeploymentName: metricsServerDeploymentName,
			ServiceName:                 serviceName,
			TriggerAuthName:             triggerAuthName,
			ScaledObject:                scaledObjectName,
			ScaledObjectNameBehavior:    scaledObjectNameBehavior,
			SecretName:                  secretName,
			MetricsServerEndpoint:       metricsServerEndpoint,
			MinReplicas:                 fmt.Sprintf("%v", minReplicas),
			MaxReplicas:                 fmt.Sprintf("%v", maxReplicas),
			MetricValue:                 0,
			DefaultFallback:             defaultFallback,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "metricsServerDeploymentTemplate", Config: metricsServerDeploymentTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
