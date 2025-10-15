//go:build e2e
// +build e2e

package external_scale_sj_test

import (
	"fmt"
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
	testName = "external-scaler-sj-test"
)

var (
	testNamespace         = fmt.Sprintf("%s-ns", testName)
	serviceName           = fmt.Sprintf("%s-service", testName)
	scalerName            = fmt.Sprintf("%s-scaler", testName)
	scaledJobName         = fmt.Sprintf("%s-sj", testName)
	metricsServerEndpoint = fmt.Sprintf("http://%s.%s.svc.cluster.local:8080/api/value", serviceName, testNamespace)
)

type templateData struct {
	TestNamespace                string
	ServiceName                  string
	ScalerName                   string
	ScaledJobName                string
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
          image: ghcr.io/kedacore/tests-external-scaler:latest
          imagePullPolicy: Always
          ports:
          - containerPort: 6000
          - containerPort: 8080
`

	scaledJobTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{.ScaledJobName}}
  namespace: {{.TestNamespace}}
spec:
  jobTargetRef:
    template:
      spec:
        containers:
          - name: external-executor
            image: busybox
            command:
            - sleep
            - "30"
            imagePullPolicy: IfNotPresent
        restartPolicy: Never
    backoffLimit: 1
  pollingInterval: 5
  maxReplicaCount: 3
  successfulJobsHistoryLimit: 0
  failedJobsHistoryLimit: 0
  triggers:
    - type: external
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
        image: docker.io/curlimages/curl
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

	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, scalerName, testNamespace, 1, 60, 1),
		"replica count should be 1 after 1 minute")

	assert.True(t, WaitForJobCount(t, kc, testNamespace, 0, 60, 1),
		"job count should be 0 after 1 minute")

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
			ScalerName:            scalerName,
			ScaledJobName:         scaledJobName,
			MetricThreshold:       10,
			MetricsServerEndpoint: metricsServerEndpoint,
		}, []Template{
			{Name: "scalerTemplate", Config: scalerTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "scaledJobTemplate", Config: scaledJobTemplate},
		}
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")

	t.Log("scaling to max replicas")
	data.MetricValue = data.MetricThreshold * 3
	KubectlReplaceWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	assert.True(t, WaitForScaledJobCount(t, kc, scaledJobName, testNamespace, 3, 60, 1),
		"job count should be 3 after 1 minute")
	KubectlDeleteWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in ---")

	t.Log("scaling to idle replicas")
	data.MetricValue = 0
	KubectlReplaceWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	assert.True(t, WaitForScaledJobCount(t, kc, scaledJobName, testNamespace, 0, 120, 1),
		"job count should be 0 after 2 minute")
	KubectlDeleteWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)
}
