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
	testNamespace = fmt.Sprintf("%s-ns", testName)
	serviceName   = fmt.Sprintf("%s-service", testName)
	scalerName    = fmt.Sprintf("%s-scaler", testName)
	scaledJobName = fmt.Sprintf("%s-sj", testName)
)

type templateData struct {
	TestNamespace                    string
	ServiceName                      string
	ScalerName                       string
	ScaledJobName                    string
	MetricThreshold, MetricValue     int
	MinReplicaCount, MaxReplicaCount int
}
type templateValues map[string]string

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
      targetPort: 6000
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
          image: ghcr.io/kedacore/tests-external-scaler-e2e:latest
          imagePullPolicy: Always
          ports:
          - containerPort: 6000
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
        metricValue: "{{.MetricValue}}"
`
)

func TestMinReplicaCount(t *testing.T) {
	kc := GetKubernetesClient(t)
	minReplicaCount := 2
	maxReplicaCount := 10
	metricValue := 0

	data, templates := getTemplateData(minReplicaCount, maxReplicaCount, metricValue)

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForJobCountUntilIteration(t, kc, testNamespace, minReplicaCount, 15, 1),
		"job count should be %d after 1 minute", minReplicaCount)

	testMinReplicaCountWithMetricValue(t, kc, data)
	testMinReplicaCountGreaterMaxReplicaCountScalesOnlyToMaxReplicaCount(t, kc, data)
	testMinReplicaCountWithMetricValueGreaterMaxReplicaCountScalesOnlyToMaxReplicaCount(t, kc, data)

	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
}

func testMinReplicaCountWithMetricValue(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing min replica count with metric value ---")

	data.MinReplicaCount = 1
	data.MaxReplicaCount = 10
	data.MetricValue = 1

	KubectlApplyWithTemplate(t, data, "scaledJobTemplate", scaledJobTemplate)

	expectedTarget := data.MinReplicaCount + data.MetricValue
	assert.True(t, WaitForJobCountUntilIteration(t, kc, testNamespace, expectedTarget, 15, 1),
		"job count should be %d after 1 minute", expectedTarget)
}

func testMinReplicaCountGreaterMaxReplicaCountScalesOnlyToMaxReplicaCount(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing min replica count greater max replica count scales only to max replica count ---")

	data.MinReplicaCount = 2
	data.MaxReplicaCount = 1
	data.MetricValue = 0

	KubectlApplyWithTemplate(t, data, "scaledJobTemplate", scaledJobTemplate)

	assert.True(t, WaitForJobCountUntilIteration(t, kc, testNamespace, data.MaxReplicaCount, 15, 1),
		"job count should be %d after 1 minute", data.MaxReplicaCount)
}

func testMinReplicaCountWithMetricValueGreaterMaxReplicaCountScalesOnlyToMaxReplicaCount(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing min replica count with metric value greater max replica count scales only to max replica count ---")

	data.MinReplicaCount = 2
	data.MaxReplicaCount = 4
	data.MetricValue = 3

	KubectlApplyWithTemplate(t, data, "scaledJobTemplate", scaledJobTemplate)

	assert.True(t, WaitForJobCountUntilIteration(t, kc, testNamespace, data.MaxReplicaCount, 15, 1),
		"job count should be %d after 1 minute", data.MaxReplicaCount)
}

func getTemplateData(minReplicaCount int, maxReplicaCount int, metricValue int) (templateData, templateValues) {
	return templateData{
			TestNamespace:   testNamespace,
			ServiceName:     serviceName,
			ScalerName:      scalerName,
			ScaledJobName:   scaledJobName,
			MetricThreshold: 1,
			MetricValue:     metricValue,
			MinReplicaCount: minReplicaCount,
			MaxReplicaCount: maxReplicaCount,
		}, templateValues{
			"scalerTemplate":    scalerTemplate,
			"serviceTemplate":   serviceTemplate,
			"scaledJobTemplate": scaledJobTemplate}
}
