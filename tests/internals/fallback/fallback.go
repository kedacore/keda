//go:build e2e
// +build e2e

//nolint:dupl,goconst
package fallback

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	helper "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "fallback-test"
)

var (
	scaleTargetName             = testName
	metricsServerDeploymentName = fmt.Sprintf("%s-metrics-server", testName)
	serviceName                 = fmt.Sprintf("%s-service", testName)
	triggerAuthName             = fmt.Sprintf("%s-ta", testName)
	scaledObjectName            = fmt.Sprintf("%s-so", testName)
	scaledObjectNameBehavior    = fmt.Sprintf("%s-behavior-so", testName)
	secretName                  = fmt.Sprintf("%s-secret", testName)
	minReplicas                 = 0
	maxReplicas                 = 5
	defaultFallback             = 3
)

type templateData struct {
	Namespace                   string
	ScaleTargetName             string
	ScaleTargetAPIVersion       string
	ScaleTargetKind             string
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

	argoRolloutTemplate = `apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: {{.ScaleTargetName}}
  namespace: {{.Namespace}}
  labels:
    app: {{.ScaleTargetName}}
spec:
  replicas: 0
  strategy:
    canary:
      steps:
        - setWeight: 50
        - pause: {duration: 10}
  selector:
    matchLabels:
      app: {{.ScaleTargetName}}-rollout
  template:
    metadata:
      labels:
        app: {{.ScaleTargetName}}-rollout
    spec:
      containers:
      - name: nginx
        image: ghcr.io/nginx/nginx-unprivileged:1.26
        ports:
        - containerPort: 80
`

	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: {{.ScaleTargetName}}
  name: {{.ScaleTargetName}}
  namespace: {{.Namespace}}
spec:
  selector:
    matchLabels:
      app: {{.ScaleTargetName}}-deployment
  replicas: 0
  template:
    metadata:
      labels:
        app: {{.ScaleTargetName}}-deployment
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
    app: {{.ScaleTargetName}}
spec:
  scaleTargetRef:
    apiVersion: {{.ScaleTargetAPIVersion}}
    kind: {{.ScaleTargetKind}}
    name: {{.ScaleTargetName}}
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
    app: {{.ScaleTargetName}}
spec:
  scaleTargetRef:
    apiVersion: {{.ScaleTargetAPIVersion}}
    kind: {{.ScaleTargetKind}}
    name: {{.ScaleTargetName}}
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

	scaledObjectTemplateWithTriggersOfValueType = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObject}}
  namespace: {{.Namespace}}
  labels:
    app: {{.ScaleTargetName}}
spec:
  scaleTargetRef:
    apiVersion: {{.ScaleTargetAPIVersion}}
    kind: {{.ScaleTargetKind}}
    name: {{.ScaleTargetName}}
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
    metricType: Value
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
    app: {{.ScaleTargetName}}
spec:
  scaleTargetRef:
    apiVersion: {{.ScaleTargetAPIVersion}}
    kind: {{.ScaleTargetKind}}
    name: {{.ScaleTargetName}}
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
    app: {{.ScaleTargetName}}
spec:
  scaleTargetRef:
    apiVersion: {{.ScaleTargetAPIVersion}}
    kind: {{.ScaleTargetKind}}
    name: {{.ScaleTargetName}}
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
    app: {{.ScaleTargetName}}
spec:
  scaleTargetRef:
    apiVersion: {{.ScaleTargetAPIVersion}}
    kind: {{.ScaleTargetKind}}
    name: {{.ScaleTargetName}}
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
    app: {{.ScaleTargetName}}
spec:
  scaleTargetRef:
    apiVersion: {{.ScaleTargetAPIVersion}}
    kind: {{.ScaleTargetKind}}
    name: {{.ScaleTargetName}}
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
        image: docker.io/curlimages/curl
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

type ScaleTargetType string

const (
	Deployment ScaleTargetType = "Deployment"
	Rollout    ScaleTargetType = "Rollout"
)

type ScaleTarget struct {
	APIVersion                                  string
	Template                                    string
	TemplateName                                string
	WaitForReplicaReadyCount                    func(*testing.T, *kubernetes.Clientset, string, string, int, int, int) bool
	AssertReplicaCountNotChangeDuringTimePeriod func(*testing.T, *kubernetes.Clientset, string, string, int, int)
}

var scaleTargetMap = map[ScaleTargetType]ScaleTarget{
	Deployment: {
		APIVersion:               "apps/v1",
		Template:                 deploymentTemplate,
		TemplateName:             "deploymentTemplate",
		WaitForReplicaReadyCount: helper.WaitForDeploymentReplicaReadyCount,
		AssertReplicaCountNotChangeDuringTimePeriod: helper.AssertReplicaCountNotChangeDuringTimePeriod,
	},
	Rollout: {
		APIVersion:               "argoproj.io/v1alpha1",
		Template:                 argoRolloutTemplate,
		TemplateName:             "argoRolloutTemplate",
		WaitForReplicaReadyCount: helper.WaitForArgoRolloutReplicaReadyCount,
		AssertReplicaCountNotChangeDuringTimePeriod: helper.AssertReplicaCountNotChangeDuringTimePeriodRollout,
	},
}

// Main function, calls all of the functions.
// fallback_deployments_test.go simply calls this with `Deployment` type.
// fallback_rollouts_test.go calls this with `Rollout` type.
func TestFallback(t *testing.T, s ScaleTargetType) {
	TestFallbackWithAverageValueMetrics(t, s)
	TestFallbackWithValueMetrics(t, s)
	TestFallbackWithoutMetricType(t, s)
	TestFallbackWithCurrentReplicasIfHigher(t, s)
	TestFallbackWithCurrentReplicasIfLower(t, s)
	TestFallbackWithCurrentReplicas(t, s)
	TestFallbackWithStatic(t, s)
	TestFallbackFromZero(t, s)
}

func TestFallbackWithAverageValueMetrics(t *testing.T, s ScaleTargetType) {
	kc := helper.GetKubernetesClient(t)
	// setup
	t.Logf("--- running TestFallbackWithAverageValueMetrics for %s ---", s)
	data, templates := getTemplateData(s)

	helper.CreateKubernetesResources(t, kc, data.Namespace, data, templates)

	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, minReplicas, 180, 3),
		"replica count should be %d after 9 minutes", minReplicas)

	testScaleOut(t, kc, s, data)
	testFallback(t, kc, s, data)
	testRestoreAfterFallback(t, kc, s, data)

	helper.DeleteKubernetesResources(t, data.Namespace, data, templates)
}

func TestFallbackWithValueMetrics(t *testing.T, s ScaleTargetType) {
	kc := helper.GetKubernetesClient(t)
	// setup
	t.Logf("--- running TestFallbackWithValueMetrics test for %s ---", s)
	data, templates := getTemplateData(s)

	// Replace the default scaledObject template
	for i, tmpl := range templates {
		if tmpl.Name == "scaledObjectTemplate" {
			templates[i].Config = scaledObjectTemplateWithTriggersOfValueType
			break
		}
	}

	helper.CreateKubernetesResources(t, kc, data.Namespace, data, templates)

	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, minReplicas, 180, 3),
		"replica count should be %d after 9 minutes", minReplicas)
	testScaleOut(t, kc, s, data)
	testFallback(t, kc, s, data)
	testRestoreAfterFallback(t, kc, s, data)

	helper.DeleteKubernetesResources(t, data.Namespace, data, templates)
}

func TestFallbackWithoutMetricType(t *testing.T, s ScaleTargetType) {
	kc := helper.GetKubernetesClient(t)
	// setup
	t.Logf("--- running TestFallbackWithoutMetricType test for %s ---", s)
	data, templates := getTemplateData(s)

	// Replace the default scaledObject template
	for i, tmpl := range templates {
		if tmpl.Name == "scaledObjectTemplate" {
			templates[i].Config = scaledObjectTemplateWithoutMetricType
			break
		}
	}

	helper.CreateKubernetesResources(t, kc, data.Namespace, data, templates)

	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, minReplicas, 180, 3),
		"replica count should be %d after 9 minutes", minReplicas)

	// Scale out to 4 replicas (20 / 5 = 4)
	data.MetricValue = 20
	helper.KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, 4, 60, 3),
		"replica count should be 4 after 3 minutes")

	// Stop metrics server to trigger fallback
	helper.KubectlApplyWithTemplate(t, data, "fallbackMSDeploymentTemplate", fallbackMSDeploymentTemplate)

	// Should keep 4 replicas as it's higher than fallback value (3)
	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, 4, 30, 3),
		"replica count should remain at 4 after fallback")

	// Ensure the replica count remains stable
	scaleTargetMap[s].AssertReplicaCountNotChangeDuringTimePeriod(t, kc, scaleTargetName, data.Namespace, 4, 30)

	helper.DeleteKubernetesResources(t, data.Namespace, data, templates)
}

func TestFallbackWithCurrentReplicasIfHigher(t *testing.T, s ScaleTargetType) {
	kc := helper.GetKubernetesClient(t)
	// setup
	t.Logf("--- running TestFallbackWithCurrentReplicasIfHigher test for %s ---", s)
	data, templates := getTemplateData(s)

	// Replace the default scaledObject template
	for i, tmpl := range templates {
		if tmpl.Name == "scaledObjectTemplate" {
			templates[i].Config = scaledObjectTemplateWithCurrentReplicasIfHigher
			break
		}
	}

	helper.CreateKubernetesResources(t, kc, data.Namespace, data, templates)

	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, minReplicas, 180, 3),
		"replica count should be %d after 9 minutes", minReplicas)

	// Scale out to 4 replicas (20 / 5 = 4)
	data.MetricValue = 20
	helper.KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, 4, 60, 3),
		"replica count should be 4 after 3 minutes")

	// Stop metrics server to trigger fallback
	helper.KubectlApplyWithTemplate(t, data, "fallbackMSDeploymentTemplate", fallbackMSDeploymentTemplate)

	// Should keep 4 replicas as it's higher than fallback value (3)
	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, 4, 30, 3),
		"replica count should remain at 4 after fallback")

	// Ensure the replica count remains stable
	scaleTargetMap[s].AssertReplicaCountNotChangeDuringTimePeriod(t, kc, scaleTargetName, data.Namespace, 4, 30)

	helper.DeleteKubernetesResources(t, data.Namespace, data, templates)
}

func TestFallbackWithCurrentReplicasIfLower(t *testing.T, s ScaleTargetType) {
	kc := helper.GetKubernetesClient(t)
	// setup
	t.Logf("--- running TestFallbackWithCurrentReplicasIfLower test for %s ---", s)
	data, templates := getTemplateData(s)

	// Replace the default scaledObject template
	for i, tmpl := range templates {
		if tmpl.Name == "scaledObjectTemplate" {
			templates[i].Config = scaledObjectTemplateWithCurrentReplicasIfLower
			break
		}
	}

	helper.CreateKubernetesResources(t, kc, data.Namespace, data, templates)

	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, minReplicas, 180, 3),
		"replica count should be %d after 9 minutes", minReplicas)

	// Scale out to 4 replicas (20 / 5 = 4)
	data.MetricValue = 20
	helper.KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, 4, 60, 3),
		"replica count should be 4 after 3 minutes")

	// Stop metrics server to trigger fallback
	helper.KubectlApplyWithTemplate(t, data, "fallbackMSDeploymentTemplate", fallbackMSDeploymentTemplate)

	// Should keep fallback value (3) as it's lower than current replicas (4)
	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, 3, 30, 3),
		"replica count should remain at 3 after fallback")

	// Ensure the replica count remains stable
	scaleTargetMap[s].AssertReplicaCountNotChangeDuringTimePeriod(t, kc, scaleTargetName, data.Namespace, 3, 30)

	helper.DeleteKubernetesResources(t, data.Namespace, data, templates)
}

func TestFallbackWithCurrentReplicas(t *testing.T, s ScaleTargetType) {
	kc := helper.GetKubernetesClient(t)
	// setup
	t.Logf("--- running TestFallbackWithCurrentReplicas test for %s---", s)

	data, templates := getTemplateData(s)

	// Replace the default scaledObject template
	for i, tmpl := range templates {
		if tmpl.Name == "scaledObjectTemplate" {
			templates[i].Config = scaledObjectTemplateWithCurrentReplicas
			break
		}
	}

	helper.CreateKubernetesResources(t, kc, data.Namespace, data, templates)

	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, minReplicas, 180, 3),
		"replica count should be %d after 9 minutes", minReplicas)

	// Scale out to 4 replicas (20 / 5 = 4)
	data.MetricValue = 20
	helper.KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, 4, 60, 3),
		"replica count should be 4 after 3 minutes")

	// Stop metrics server to trigger fallback
	helper.KubectlApplyWithTemplate(t, data, "fallbackMSDeploymentTemplate", fallbackMSDeploymentTemplate)

	// Should keep current replicas (4)
	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, 4, 30, 3),
		"replica count should remain at 4 after fallback")

	// Ensure the replica count remains stable
	scaleTargetMap[s].AssertReplicaCountNotChangeDuringTimePeriod(t, kc, scaleTargetName, data.Namespace, 4, 30)

	helper.DeleteKubernetesResources(t, data.Namespace, data, templates)
}

func TestFallbackWithStatic(t *testing.T, s ScaleTargetType) {
	kc := helper.GetKubernetesClient(t)
	t.Logf("--- running TestFallbackWithStatic test for %s ---", s)
	data, templates := getTemplateData(s)

	// Replace the default scaledObject template
	for i, tmpl := range templates {
		if tmpl.Name == "scaledObjectTemplate" {
			templates[i].Config = scaledObjectTemplateWithStatic
			break
		}
	}

	helper.CreateKubernetesResources(t, kc, data.Namespace, data, templates)

	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, minReplicas, 180, 3),
		"replica count should be %d after 9 minutes", minReplicas)

	// Scale out to 4 replicas (20 / 5 = 4)
	data.MetricValue = 20
	helper.KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, 4, 60, 3),
		"replica count should be 4 after 3 minutes")

	// Stop metrics server to trigger fallback
	helper.KubectlApplyWithTemplate(t, data, "fallbackMSDeploymentTemplate", fallbackMSDeploymentTemplate)

	// Should keep fallback value (3) because of static
	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, 3, 30, 3),
		"replica count should remain at 3 after fallback")

	// Ensure the replica count remains stable
	scaleTargetMap[s].AssertReplicaCountNotChangeDuringTimePeriod(t, kc, scaleTargetName, data.Namespace, 3, 30)

	helper.DeleteKubernetesResources(t, data.Namespace, data, templates)
}

func TestFallbackFromZero(t *testing.T, s ScaleTargetType) {
	kc := helper.GetKubernetesClient(t)
	t.Logf("--- running TestFallbackFromZero test for %s ---", s)
	data, templates := getTemplateData(s)

	// Replace the default ScaledObject template
	for i, tmpl := range templates {
		if tmpl.Name == "scaledObjectTemplate" {
			templates[i].Config = scaledObjectTemplateWithStatic
			break
		}
	}

	helper.CreateKubernetesResources(t, kc, data.Namespace, data, templates)

	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, minReplicas, 180, 3),
		"replica count should be %d after 9 minutes", minReplicas)

	// Stop metrics server to trigger fallback
	helper.KubectlApplyWithTemplate(t, data, "fallbackMSDeploymentTemplate", fallbackMSDeploymentTemplate)

	// Should go to fallback value (3) because of static
	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, 3, 30, 3),
		"replica count should be 3 after fallback")

	// Ensure the replica count remains stable
	scaleTargetMap[s].AssertReplicaCountNotChangeDuringTimePeriod(t, kc, scaleTargetName, data.Namespace, 3, 30)

	helper.DeleteKubernetesResources(t, data.Namespace, data, templates)
}

// scale out to max replicas first
func testScaleOut(t *testing.T, kc *kubernetes.Clientset, s ScaleTargetType, data templateData) {
	t.Log("--- testing scale out ---")
	data.MetricValue = 50
	helper.KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, maxReplicas, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicas)
}

// MS replicas set to 0 to envoke fallback
func testFallback(t *testing.T, kc *kubernetes.Clientset, s ScaleTargetType, data templateData) {
	t.Log("--- testing fallback ---")
	helper.KubectlApplyWithTemplate(t, data, "fallbackMSDeploymentTemplate", fallbackMSDeploymentTemplate)
	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, defaultFallback, 60, 3),
		"replica count should be %d after 3 minutes", defaultFallback)
	// We need to ensure that the fallback value is stable to cover this regression
	// https://github.com/kedacore/keda/issues/4249
	scaleTargetMap[s].AssertReplicaCountNotChangeDuringTimePeriod(t, kc, scaleTargetName, data.Namespace, defaultFallback, 30)
}

// restore MS to scale back from fallback replicas
func testRestoreAfterFallback(t *testing.T, kc *kubernetes.Clientset, s ScaleTargetType, data templateData) {
	t.Log("--- testing after fallback ---")
	helper.KubectlApplyWithTemplate(t, data, "metricsServerDeploymentTemplate", metricsServerDeploymentTemplate)
	data.MetricValue = 50
	helper.KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	assert.True(t, scaleTargetMap[s].WaitForReplicaReadyCount(t, kc, scaleTargetName, data.Namespace, maxReplicas, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicas)
}

func getTemplateData(s ScaleTargetType) (templateData, []helper.Template) {
	namespace := fmt.Sprintf("%s-ns-%s", testName, strings.ToLower(string(s)))
	return templateData{
			Namespace:                   namespace,
			ScaleTargetName:             scaleTargetName,
			ScaleTargetAPIVersion:       scaleTargetMap[s].APIVersion,
			ScaleTargetKind:             string(s),
			MetricsServerDeploymentName: metricsServerDeploymentName,
			ServiceName:                 serviceName,
			TriggerAuthName:             triggerAuthName,
			ScaledObject:                scaledObjectName,
			ScaledObjectNameBehavior:    scaledObjectNameBehavior,
			SecretName:                  secretName,
			MetricsServerEndpoint:       fmt.Sprintf("http://%s.%s.svc.cluster.local:8080/api/value", serviceName, namespace),
			MinReplicas:                 fmt.Sprintf("%v", minReplicas),
			MaxReplicas:                 fmt.Sprintf("%v", maxReplicas),
			MetricValue:                 0,
			DefaultFallback:             defaultFallback,
		}, []helper.Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "metricsServerDeploymentTemplate", Config: metricsServerDeploymentTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: scaleTargetMap[s].TemplateName, Config: scaleTargetMap[s].Template},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
