//go:build e2e
// +build e2e

package external_scale_sj_test

import (
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "external-scaler-sj-test"
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
	maxReplicaCount := 50
	metricValue := 0

	data, templates := getTemplateData(minReplicaCount, maxReplicaCount, metricValue)

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForJobCount(t, kc, testNamespace, 2, 60, 1),
		"job count should be 2 after 1 minute")

	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
}

func TestMinReplicaCountGreaterMaxReplicaCountScalesOnlyToMaxReplicaCount(t *testing.T) {
	kc := GetKubernetesClient(t)
	minReplicaCount := 2
	maxReplicaCount := 1
	metricValue := 0

	data, templates := getTemplateData(minReplicaCount, maxReplicaCount, metricValue)

	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	data.MetricValue = data.MetricThreshold * 3

	assert.True(t, WaitForJobCount(t, kc, testNamespace, 1, 60, 1),
		"job count should be 1 after 1 minute")

	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
}

func TestMinReplicaCountWithMetricValue(t *testing.T) {
	kc := GetKubernetesClient(t)
	minReplicaCount := 2
	maxReplicaCount := 10
	metricValue := 1

	data, templates := getTemplateData(minReplicaCount, maxReplicaCount, metricValue)

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForJobCount(t, kc, testNamespace, 3, 60, 1),
		"job count should be 3 after 1 minute")

	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
}

func TestMinReplicaCountWithMetricValueDoesNotExceedMaxReplicaCount(t *testing.T) {
	kc := GetKubernetesClient(t)
	minReplicaCount := 2
	maxReplicaCount := 3
	metricValue := 2

	data, templates := getTemplateData(minReplicaCount, maxReplicaCount, metricValue)

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForJobCount(t, kc, testNamespace, 3, 60, 1),
		"job count should be 3 after 1 minute")

	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
}

func getTemplateData(minReplicaCount int, maxReplicaCount int, metricValue int) (templateData, templateValues) {
	return templateData{
			TestNamespace:   testNamespace,
			ServiceName:     serviceName,
			ScalerName:      scalerName,
			ScaledJobName:   scaledJobName,
			MetricThreshold: 10,
			MetricValue:     metricValue,
			MinReplicaCount: minReplicaCount,
			MaxReplicaCount: maxReplicaCount,
		}, templateValues{
			"scalerTemplate":    scalerTemplate,
			"serviceTemplate":   serviceTemplate,
			"scaledJobTemplate": scaledJobTemplate}
}
