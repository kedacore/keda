//go:build e2e
// +build e2e

package scaling_modifiers_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "scaling-modifiers-test"
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
	secretName                  = fmt.Sprintf("%s-secret", testName)
	metricsServerEndpoint       = fmt.Sprintf("http://%s.%s.svc.cluster.local:8080/api/value", serviceName, namespace)
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

	// for metrics-api trigger
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
        readinessProbe:
          httpGet:
            path: /api/value
            port: 8080
`
	soFallbackTemplate = `
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
          stabilizationWindowSeconds: 5
    scalingModifiers:
      formula: metrics_api + kw_trig
      target: '2'
      activationTarget: '2'
  pollingInterval: 5
  cooldownPeriod: 5
  minReplicaCount: 0
  maxReplicaCount: 10
  fallback:
    replicas: 5
    failureThreshold: 3
  triggers:
  - type: metrics-api
    name: metrics_api
    metadata:
      url: "{{.MetricsServerEndpoint}}"
      valueLocation: 'value'
      method: "query"
    metricType: "AverageValue"
    authenticationRef:
      name: {{.TriggerAuthName}}
  - type: kubernetes-workload
    name: kw_trig
    metadata:
      podSelector: pod=workload-test
    metricType: "AverageValue"
`

	soComplexFormula = `
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
          stabilizationWindowSeconds: 5
    scalingModifiers:
      formula: "count([kw_trig,metrics_api],{#>1}) > 1 ? 5 : 0"
      target: '2'
      activationTarget: '2'
  pollingInterval: 5
  cooldownPeriod: 5
  minReplicaCount: 0
  maxReplicaCount: 10
  fallback:
    replicas: 5
    failureThreshold: 3
  triggers:
  - type: metrics-api
    name: metrics_api
    metadata:
      url: "{{.MetricsServerEndpoint}}"
      valueLocation: 'value'
      method: "query"
    metricType: "AverageValue"
    authenticationRef:
      name: {{.TriggerAuthName}}
  - type: kubernetes-workload
    name: kw_trig
    metadata:
      podSelector: pod=workload-test
    metricType: "AverageValue"
`

	workloadDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: depl-workload-base
  namespace: {{.TestNamespace}}
  labels:
    deploy: workload-test
spec:
  replicas: 0
  selector:
    matchLabels:
      pod: workload-test
  template:
    metadata:
      labels:
        pod: workload-test
    spec:
      containers:
        - name: nginx
          image: 'ghcr.io/nginx/nginx-unprivileged:1.26'`

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
        image: curlimages/curl
        imagePullPolicy: Always
        command: ["curl", "-X", "POST", "{{.MetricsServerEndpoint}}/{{.MetricValue}}"]
      restartPolicy: OnFailure
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
)

func TestScalingModifiers(t *testing.T) {
	// setup
	t.Log("-- setting up ---")
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, namespace, data, templates)

	// we ensure that the metrics api server is up and ready
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, metricsServerDeploymentName, namespace, 1, 60, 2),
		"replica count should be 1 after 1 minute")

	testFormula(t, kc, data)

	templates = append(templates, Template{Name: "soComplexFormula", Config: soComplexFormula})
	DeleteKubernetesResources(t, namespace, data, templates)
}

func testFormula(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testFormula ---")

	// formula simply adds 2 metrics together (0+2=2; activationTarget = 2 -> replicas should be 0)
	KubectlApplyWithTemplate(t, data, "soFallbackTemplate", soFallbackTemplate)
	data.MetricValue = 0
	KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, namespace, 0, 60)

	// formula simply adds 2 metrics together (3+2=5; target = 2 -> 5/2 replicas should be 3)
	data.MetricValue = 3
	KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	_, err := ExecuteCommand(fmt.Sprintf("kubectl scale deployment/depl-workload-base --replicas=2 -n %s", namespace))
	assert.NoErrorf(t, err, "cannot scale workload deployment - %s", err)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, "depl-workload-base", namespace, 2, 12, 10),
		"replica count should be %d after 1 minute", 2)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 3, 12, 10),
		"replica count should be %d after 2 minutes", 3)

	// apply fallback
	_, err = ExecuteCommand(fmt.Sprintf("kubectl scale deployment/%s --replicas=0 -n %s", metricsServerDeploymentName, namespace))
	assert.NoErrorf(t, err, "cannot scale metricsServer deployment - %s", err)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 5, 12, 10),
		"replica count should be %d after 2 minutes", 5)
	time.Sleep(45 * time.Second) // waiting for passing failureThreshold
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, namespace, 5, 60)

	// ensure state returns to normal after error resolved and triggers are healthy
	_, err = ExecuteCommand(fmt.Sprintf("kubectl scale deployment/%s --replicas=1 -n %s", metricsServerDeploymentName, namespace))
	assert.NoErrorf(t, err, "cannot scale metricsServer deployment - %s", err)

	// we ensure that the metrics api server is up and ready
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, metricsServerDeploymentName, namespace, 1, 60, 2),
		"replica count should be 1 after 1 minute")

	data.MetricValue = 2
	KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)
	// 2+2=4; target = 2 -> 4/2 replicas should be 2
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 2, 12, 10),
		"replica count should be %d after 2 minutes", 2)

	data.MetricValue = 0
	KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	// apply new SO
	KubectlApplyWithTemplate(t, data, "soComplexFormula", soComplexFormula)

	// formula has count() which needs atleast 2 metrics to have value over 1 to scale up
	// now should be 0
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 0, 12, 10),
		"replica count should be %d after 2 minutes", 0)

	data.MetricValue = 2
	KubectlReplaceWithTemplate(t, data, "updateMetricsTemplate", updateMetricsTemplate)

	// 5//2 = 3
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, namespace, 3, 12, 10),
		"replica count should be %d after 2 minutes", 3)
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
			MetricValue:                 0,
		}, []Template{
			// basic: scaled deployment, metrics-api trigger server & authentication
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "metricsServerDeploymentTemplate", Config: metricsServerDeploymentTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			// workload base
			{Name: "workloadDeploymentTemplate", Config: workloadDeploymentTemplate},
		}
}
