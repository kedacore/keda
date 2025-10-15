//go:build e2e
// +build e2e

package external_scale_sj_test

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
	testName = "min-replica-test"
)

var (
	testNamespace         = fmt.Sprintf("%s-ns", testName)
	serviceName           = fmt.Sprintf("%s-service", testName)
	scalerName            = fmt.Sprintf("%s-scaler", testName)
	scaledJobName         = fmt.Sprintf("%s-sj", testName)
	metricsServerEndpoint = fmt.Sprintf("http://%s.%s.svc.cluster.local:8080/api/value", serviceName, testNamespace)
	iterationCount        = 60
)

type templateData struct {
	TestNamespace                    string
	ServiceName                      string
	ScalerName                       string
	ScaledJobName                    string
	MetricsServerEndpoint            string
	MetricThreshold, MetricValue     int
	MinReplicaCount, MaxReplicaCount int
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
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
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

func TestMinReplicaCount(t *testing.T) {
	kc := GetKubernetesClient(t)
	minReplicaCount := 2
	maxReplicaCount := 10

	data, templates := getTemplateData(minReplicaCount, maxReplicaCount)

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForScaledJobCount(t, kc, scaledJobName, testNamespace, minReplicaCount, iterationCount, 1),
		"job count should be %d after %d iterations", minReplicaCount, iterationCount)

	testMinReplicaCountWithMetricValue(t, kc, data)
	testMinReplicaCountGreaterMaxReplicaCountScalesOnlyToMaxReplicaCount(t, kc, data)
	testMinReplicaCountWithMetricValueGreaterMaxReplicaCountScalesOnlyToMaxReplicaCount(t, kc, data)

	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func testMinReplicaCountWithMetricValue(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing min replica count with metric value ---")

	data.MinReplicaCount = 1
	data.MaxReplicaCount = 10
	data.MetricValue = 1

	KubectlApplyWithTemplate(t, data, "scaledJobTemplate", scaledJobTemplate)
	KubectlApplyWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	expectedTarget := data.MinReplicaCount + data.MetricValue
	assert.True(t, WaitForScaledJobCount(t, kc, scaledJobName, testNamespace, expectedTarget, iterationCount, 1),
		"job count should be %d after %d iterations", expectedTarget, iterationCount)
	KubectlDeleteWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)
}

func testMinReplicaCountGreaterMaxReplicaCountScalesOnlyToMaxReplicaCount(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing min replica count greater max replica count scales only to max replica count ---")

	data.MinReplicaCount = 2
	data.MaxReplicaCount = 1
	data.MetricValue = 0

	KubectlApplyWithTemplate(t, data, "scaledJobTemplate", scaledJobTemplate)
	KubectlApplyWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	assert.True(t, WaitForScaledJobCount(t, kc, scaledJobName, testNamespace, data.MaxReplicaCount, iterationCount, 1),
		"job count should be %d after %d iterations", data.MaxReplicaCount, iterationCount)
	KubectlDeleteWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)
}

func testMinReplicaCountWithMetricValueGreaterMaxReplicaCountScalesOnlyToMaxReplicaCount(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing min replica count with metric value greater max replica count scales only to max replica count ---")

	data.MinReplicaCount = 2
	data.MaxReplicaCount = 4
	data.MetricValue = 3

	KubectlApplyWithTemplate(t, data, "scaledJobTemplate", scaledJobTemplate)
	KubectlApplyWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)

	assert.True(t, WaitForScaledJobCount(t, kc, scaledJobName, testNamespace, data.MaxReplicaCount, iterationCount, 1),
		"job count should be %d after %d iterations", data.MaxReplicaCount, iterationCount)
	KubectlDeleteWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)
}

func getTemplateData(minReplicaCount int, maxReplicaCount int) (templateData, []Template) {
	return templateData{
			TestNamespace:         testNamespace,
			ServiceName:           serviceName,
			ScalerName:            scalerName,
			ScaledJobName:         scaledJobName,
			MetricThreshold:       1,
			MetricsServerEndpoint: metricsServerEndpoint,
			MinReplicaCount:       minReplicaCount,
			MaxReplicaCount:       maxReplicaCount,
		}, []Template{
			{Name: "scalerTemplate", Config: scalerTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "scaledJobTemplate", Config: scaledJobTemplate},
		}
}
