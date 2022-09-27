//go:build e2e
// +build e2e

package prometheus_metrics_test

import (
	"fmt"
	"strings"
	"testing"

	promModel "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "prometheus-metrics-test"
)

var (
	testNamespace           = fmt.Sprintf("%s-ns", testName)
	deploymentName          = fmt.Sprintf("%s-deployment", testName)
	monitoredDeploymentName = fmt.Sprintf("%s-monitored", testName)
	scaledObjectName        = fmt.Sprintf("%s-so", testName)
	cronScaledJobName       = fmt.Sprintf("%s-cron-sj", testName)
	clientName              = fmt.Sprintf("%s-client", testName)
	serviceName             = fmt.Sprintf("%s-service", testName)
)

type templateData struct {
	TestNamespace           string
	DeploymentName          string
	ScaledObjectName        string
	CronScaledJobName       string
	MonitoredDeploymentName string
	ClientName              string
	ServiceName             string
}

const (
	monitoredDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.MonitoredDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.MonitoredDeploymentName}}
spec:
  replicas: 4
  selector:
    matchLabels:
      app: {{.MonitoredDeploymentName}}
  template:
    metadata:
      labels:
        app: {{.MonitoredDeploymentName}}
    spec:
      containers:
        - name: {{.MonitoredDeploymentName}}
          image: nginx
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
  replicas: 1
  selector:
    matchLabels:
      app: {{.DeploymentName}}
  template:
    metadata:
      labels:
        app: {{.DeploymentName}}
    spec:
      containers:
        - name: {{.DeploymentName}}
          image: nginx
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
  idleReplicaCount: 0
  minReplicaCount: 1
  maxReplicaCount: 2
  cooldownPeriod: 10
  triggers:
    - type: kubernetes-workload
      metadata:
        podSelector: 'app={{.MonitoredDeploymentName}}'
        value: '1'
`

	cronScaledJobTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{.CronScaledJobName}}
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
  - type: cron
    metadata:
      timezone: Etc/UTC
      start: 0 * * * *
      end: 0 * * * *
      desiredReplicas: '4'
  - type: cron
    metadata:
      timezone: Etc/UTC
      start: 0 * * * *
      end: 0 * * * *
      desiredReplicas: '4'
`

	clientTemplate = `
apiVersion: v1
kind: Pod
metadata:
  name: {{.ClientName}}
  namespace: {{.TestNamespace}}
spec:
  containers:
  - name: {{.ClientName}}
    image: curlimages/curl
    command:
      - sh
      - -c
      - "exec tail -f /dev/null"`

	serviceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: {{.ServiceName}}
  namespace: keda
spec:
  ports:
  - name: metrics
    port: 8080
    targetPort: 8080
  selector:
    app: keda-operator
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// scaling to max replica count to ensure the counter is registered before we test it
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 2),
		"replica count should be 2 after 2 minute")

	testHPAScalerMetricValue(t)
	testTriggerTotalMetric(t, data)

	// cleanup
	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:           testNamespace,
			DeploymentName:          deploymentName,
			ScaledObjectName:        scaledObjectName,
			MonitoredDeploymentName: monitoredDeploymentName,
			ClientName:              clientName,
			ServiceName:             serviceName,
			CronScaledJobName:       cronScaledJobName,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "monitoredDeploymentTemplate", Config: monitoredDeploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
			{Name: "clientTemplate", Config: clientTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
		}
}

func fetchAndParsePrometheusMetrics(t *testing.T, cmd string) map[string]*promModel.MetricFamily {
	out, _, err := ExecCommandOnSpecificPod(t, clientName, testNamespace, cmd)
	assert.NoErrorf(t, err, "cannot execute command - %s", err)

	parser := expfmt.TextParser{}
	// Ensure EOL
	reader := strings.NewReader(strings.ReplaceAll(out, "\r\n", "\n"))
	family, err := parser.TextToMetricFamilies(reader)
	assert.NoErrorf(t, err, "cannot parse metrics - %s", err)

	return family
}

func testHPAScalerMetricValue(t *testing.T) {
	t.Log("--- testing hpa scaler metric value ---")

	family := fetchAndParsePrometheusMetrics(t, "curl --insecure http://keda-metrics-apiserver.keda:9022/metrics")

	if val, ok := family["keda_metrics_adapter_scaler_metrics_value"]; ok {
		var found bool
		metrics := val.GetMetric()
		for _, metric := range metrics {
			labels := metric.GetLabel()
			for _, label := range labels {
				if *label.Name == "scaledObject" && *label.Value == scaledObjectName {
					assert.Equal(t, float64(4), *metric.Gauge.Value)
					found = true
				}
			}
		}
		assert.Equal(t, true, found)
	} else {
		t.Errorf("metric not available")
	}
}

func testTriggerTotalMetric(t *testing.T, data templateData) {
	t.Log("--- testing trigger total metric ---")
	testTriggerTotalMetricValue(t, map[string]int{"kubernetes-workload": 1})

	KubectlApplyWithTemplate(t, data, "cronScaledJobTemplate", cronScaledJobTemplate)
	testTriggerTotalMetricValue(t, map[string]int{"kubernetes-workload": 1, "cron": 2})

	KubectlDeleteWithTemplate(t, data, "cronScaledJobTemplate", cronScaledJobTemplate)
	testTriggerTotalMetricValue(t, map[string]int{"kubernetes-workload": 1})
}

func testTriggerTotalMetricValue(t *testing.T, expected map[string]int) {
	family := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure http://%s.keda:8080/metrics", serviceName))

	if val, ok := family["keda_operator_trigger_totals"]; ok {
		var found bool
		metrics := val.GetMetric()
		for _, metric := range metrics {
			labels := metric.GetLabel()
			for _, label := range labels {
				if *label.Name == "type" {
					assert.Equalf(t, float64(expected[*label.Value]), *metric.Gauge.Value, "expected %f got %f for type %s",
						float64(expected[*label.Value]), *metric.Gauge.Value, *label.Value)
					found = true
				}
			}
		}
		assert.Equal(t, true, found)
	} else {
		t.Errorf("metric not available")
	}
}
