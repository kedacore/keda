//go:build e2e
// +build e2e

//nolint:goconst
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
        command: ["curl", "--connect-timeout", "5", "--retry", "20", "--retry-delay", "3", "--retry-all-errors", "-X", "POST", "{{.MetricsServerEndpoint}}/{{.MetricValue}}"]
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

// getTemplateData returns the template data for a scenario. Every scenario has its own namespace, so
// that the scenarios can run as independent tests at the same time.
func getTemplateData(s ScaleTargetType, scenario string) (templateData, []helper.Template) {
	namespace := fmt.Sprintf("%s-ns-%s-%s", testName, strings.ToLower(string(s)), scenario)
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
