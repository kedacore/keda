//go:build e2e
// +build e2e

package external_push_scaler_old_proto_test

import (
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "external-push-scaler-old-proto-test"
)

var (
	testNamespace         = fmt.Sprintf("%s-ns", testName)
	serviceName           = fmt.Sprintf("%s-service", testName)
	deploymentName        = fmt.Sprintf("%s-deployment", testName)
	scalerName            = fmt.Sprintf("%s-scaler", testName)
	scaledObjectName      = fmt.Sprintf("%s-so", testName)
	metricsServerEndpoint = fmt.Sprintf("http://%s.%s.svc.cluster.local:8080/api/value", serviceName, testNamespace)
)

type templateData struct {
	TestNamespace                string
	ServiceName                  string
	DeploymentName               string
	ScalerName                   string
	ScaledObjectName             string
	MetricsServerEndpoint        string
	MetricThreshold, MetricValue int
}

const (
	serviceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: {{.ServiceName}}
  namespace: {{.TestNamespace}}
spec:
  ports:
    - port: 6000
      name: grpc
      targetPort: 6000
    - port: 8080
      name: http
      targetPort: 8080
  selector:
    app: {{.ScalerName}}
`

	scalerTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.ScalerName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.ScalerName}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.ScalerName}}
  template:
    metadata:
      labels:
        app: {{.ScalerName}}
    spec:
      containers:
        - name: scaler
          # old proto -> testing backward compatibility
          image: ghcr.io/kedacore/tests-external-scaler:5167ec1
          imagePullPolicy: Always
          ports:
          - containerPort: 6000
          - containerPort: 8080
`

	deploymentTemplate = `
apiVersion: apps/v1
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
        - name: nginx
          image: ghcr.io/nginx/nginx-unprivileged:1.26
`

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  cooldownPeriod: 10
  idleReplicaCount: 0
  minReplicaCount: 1
  maxReplicaCount: 2
  triggers:
    - type: external-push
      metadata:
        scalerAddress: {{.ServiceName}}.{{.TestNamespace}}:6000
        metricThreshold: "{{.MetricThreshold}}"
`
	updateMetricTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: update-metric-value
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - name: curl-client
        image: curlimages/curl
        imagePullPolicy: Always
        command: ["curl", "-X", "POST", "{{.MetricsServerEndpoint}}/{{.MetricValue}}"]
      restartPolicy: Never`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, scalerName, testNamespace, 1, 60, 1),
		"replica count should be 1 after 1 minute")

	// test scaling
	testScaleOut(t, kc, data)
	testScaleIn(t, kc, data)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:         testNamespace,
			ServiceName:           serviceName,
			DeploymentName:        deploymentName,
			ScalerName:            scalerName,
			ScaledObjectName:      scaledObjectName,
			MetricThreshold:       10,
			MetricsServerEndpoint: metricsServerEndpoint,
		}, []Template{
			{Name: "scalerTemplate", Config: scalerTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")

	t.Log("scaling to min replicas")
	data.MetricValue = data.MetricThreshold
	KubectlReplaceWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 1),
		"replica count should be 1 after 1 minute")
	KubectlDeleteWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	t.Log("scaling to max replicas")
	data.MetricValue = data.MetricThreshold * 2
	KubectlReplaceWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 2),
		"replica count should be 2 after 2 minutes")
	KubectlDeleteWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in ---")

	t.Log("scaling to idle replicas")
	data.MetricValue = 0
	KubectlReplaceWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 2),
		"replica count should be 0 after 2 minutes")
	KubectlDeleteWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)
}
