//go:build e2e
// +build e2e

package prometheus_metrics_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	promModel "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kedacore/keda/v2/pkg/prommetrics"
	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName          = "prometheus-metrics-test"
	labelScaledObject = "scaledObject"
)

var (
	testNamespace             = fmt.Sprintf("%s-ns", testName)
	deploymentName            = fmt.Sprintf("%s-deployment", testName)
	monitoredDeploymentName   = fmt.Sprintf("%s-monitored", testName)
	scaledObjectName          = fmt.Sprintf("%s-so", testName)
	cronScaledJobName         = fmt.Sprintf("%s-cron-sj", testName)
	clientName                = fmt.Sprintf("%s-client", testName)
	kedaOperatorPrometheusURL = "http://keda-operator.keda.svc.cluster.local:8080/metrics"
	kedaWebhookPrometheusURL  = "http://keda-admission-webhooks.keda.svc.cluster.local:8080/metrics"
	namespaceString           = "namespace"
)

type templateData struct {
	TestName                string
	TestNamespace           string
	DeploymentName          string
	ScaledObjectName        string
	CronScaledJobName       string
	MonitoredDeploymentName string
	ClientName              string
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
      end: 1 * * * *
      desiredReplicas: '4'
  - type: cron
    metadata:
      timezone: Etc/UTC
      start: 1 * * * *
      end: 2 * * * *
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

	authenticationTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: {{.TestName}}-secret
  namespace: {{.TestNamespace}}
type: Opaque
stringData:
  key: value
---
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TestName}}-ta1
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: param
    name: {{.TestName}}-secret
    key: key
---
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{.TestName}}-ta2
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: param
    name: {{.TestName}}-secret
    key: key
---
apiVersion: keda.sh/v1alpha1
kind: ClusterTriggerAuthentication
metadata:
  name: {{.TestName}}-cta
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
  - parameter: param
    name: {{.TestName}}-secret
    key: key
---
`
)

func TestPrometheusMetrics(t *testing.T) {
	// setup
	t.Log("--- setting up ---")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// scaling to max replica count to ensure the counter is registered before we test it
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 2),
		"replica count should be 2 after 2 minute")

	testScalerMetricValue(t)
	testScalerMetricLatency(t)
	testScalerActiveMetric(t)
	testMetricsServerScalerMetricValue(t)
	testOperatorMetrics(t, kc, data)
	testWebhookMetrics(t, data)

	// cleanup
	DeleteKubernetesResources(t, kc, testNamespace, data, templates)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestName:                testName,
			TestNamespace:           testNamespace,
			DeploymentName:          deploymentName,
			ScaledObjectName:        scaledObjectName,
			MonitoredDeploymentName: monitoredDeploymentName,
			ClientName:              clientName,
			CronScaledJobName:       cronScaledJobName,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "monitoredDeploymentTemplate", Config: monitoredDeploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
			{Name: "clientTemplate", Config: clientTemplate},
			{Name: "authenticatioNTemplate", Config: authenticationTemplate},
		}
}

func fetchAndParsePrometheusMetrics(t *testing.T, cmd string) map[string]*promModel.MetricFamily {
	out, _, err := ExecCommandOnSpecificPod(t, clientName, testNamespace, cmd)
	assert.NoErrorf(t, err, "cannot execute command - %s", err)

	parser := expfmt.TextParser{}
	// Ensure EOL
	reader := strings.NewReader(strings.ReplaceAll(out, "\r\n", "\n"))
	families, err := parser.TextToMetricFamilies(reader)
	assert.NoErrorf(t, err, "cannot parse metrics - %s", err)

	return families
}

func testScalerMetricValue(t *testing.T) {
	t.Log("--- testing scaler metric value ---")

	family := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorPrometheusURL))

	if val, ok := family["keda_scaler_metrics_value"]; ok {
		var found bool
		metrics := val.GetMetric()
		for _, metric := range metrics {
			labels := metric.GetLabel()
			for _, label := range labels {
				if *label.Name == labelScaledObject && *label.Value == scaledObjectName {
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

func testScalerMetricLatency(t *testing.T) {
	t.Log("--- testing scaler metric latency ---")

	family := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorPrometheusURL))

	if val, ok := family["keda_scaler_metrics_latency"]; ok {
		var found bool
		metrics := val.GetMetric()
		for _, metric := range metrics {
			labels := metric.GetLabel()
			for _, label := range labels {
				if *label.Name == labelScaledObject && *label.Value == scaledObjectName {
					assert.Equal(t, float64(0), *metric.Gauge.Value)
					found = true
				}
			}
		}
		assert.Equal(t, true, found)
	} else {
		t.Errorf("metric not available")
	}
}

func testScalerActiveMetric(t *testing.T) {
	t.Log("--- testing scaler active metric ---")

	family := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorPrometheusURL))

	if val, ok := family["keda_scaler_active"]; ok {
		var found bool
		metrics := val.GetMetric()
		for _, metric := range metrics {
			labels := metric.GetLabel()
			for _, label := range labels {
				if *label.Name == labelScaledObject && *label.Value == scaledObjectName {
					assert.Equal(t, float64(1), *metric.Gauge.Value)
					found = true
				}
			}
		}
		assert.Equal(t, true, found)
	} else {
		t.Errorf("metric not available")
	}
}

// [DEPRECATED] handle exporting Prometheus metrics from Operator to Metrics Server
func testMetricsServerScalerMetricValue(t *testing.T) {
	t.Log("--- testing scaler metric value in metrics server ---")

	family := fetchAndParsePrometheusMetrics(t, "curl --insecure http://keda-metrics-apiserver.keda:9022/metrics")

	if val, ok := family["keda_metrics_adapter_scaler_metrics_value"]; ok {
		var found bool
		metrics := val.GetMetric()
		for _, metric := range metrics {
			labels := metric.GetLabel()
			for _, label := range labels {
				if *label.Name == labelScaledObject && *label.Value == scaledObjectName {
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

func testOperatorMetrics(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing operator metrics ---")
	testOperatorMetricValues(t, kc)

	KubectlApplyWithTemplate(t, data, "cronScaledJobTemplate", cronScaledJobTemplate)
	testOperatorMetricValues(t, kc)

	KubectlDeleteWithTemplate(t, data, "cronScaledJobTemplate", cronScaledJobTemplate)
	testOperatorMetricValues(t, kc)
}

func testWebhookMetrics(t *testing.T, data templateData) {
	t.Log("--- testing webhook metrics ---")

	data.ScaledObjectName = "other-so"
	err := KubectlApplyWithErrors(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	assert.Errorf(t, err, "can deploy the scaledObject - %s", err)
	testWebhookMetricValues(t)
	data.ScaledObjectName = scaledObjectName
}

func getOperatorMetricsManually(t *testing.T, kc *kubernetes.Clientset) (map[string]int, map[string]map[string]int) {
	kedaKc := GetKedaKubernetesClient(t)

	triggerTotals := make(map[string]int)
	crTotals := map[string]map[string]int{
		"scaled_object":                  {},
		"scaled_job":                     {},
		"trigger_authentication":         {},
		"cluster_trigger_authentication": {},
	}

	namespaceList, err := kc.CoreV1().Namespaces().List(context.Background(), v1.ListOptions{})
	assert.NoErrorf(t, err, "failed to list namespaces - %s", err)

	clusterTriggerAuthenticationList, err := kedaKc.ClusterTriggerAuthentications().List(context.Background(), v1.ListOptions{})
	assert.NoErrorf(t, err, "failed to list clusterTriggerAuthentications with err - %s")

	for _, clusterTriggerAuth := range clusterTriggerAuthenticationList.Items {
		namespace := clusterTriggerAuth.Namespace
		if namespace == "" {
			namespace = "default"
		}
		crTotals[prommetrics.ClusterTriggerAuthenticationResource][namespace]++
	}

	for _, namespace := range namespaceList.Items {
		namespaceName := namespace.Name
		if namespace.Name == "" {
			namespaceName = "default"
		}

		scaledObjectList, err := kedaKc.ScaledObjects(namespace.Name).List(context.Background(), v1.ListOptions{})
		assert.NoErrorf(t, err, "failed to list scaledObjects in namespace - %s with err - %s", namespace.Name, err)

		crTotals[prommetrics.ScaledObjectResource][namespaceName] = len(scaledObjectList.Items)
		for _, scaledObject := range scaledObjectList.Items {
			for _, trigger := range scaledObject.Spec.Triggers {
				triggerTotals[trigger.Type]++
			}
		}

		scaledJobList, err := kedaKc.ScaledJobs(namespace.Name).List(context.Background(), v1.ListOptions{})
		assert.NoErrorf(t, err, "failed to list scaledJobs in namespace - %s with err - %s", namespace.Name, err)

		crTotals[prommetrics.ScaledJobResource][namespaceName] = len(scaledJobList.Items)
		for _, scaledJob := range scaledJobList.Items {
			for _, trigger := range scaledJob.Spec.Triggers {
				triggerTotals[trigger.Type]++
			}
		}

		triggerAuthList, err := kedaKc.TriggerAuthentications(namespace.Name).List(context.Background(), v1.ListOptions{})
		assert.NoErrorf(t, err, "failed to list triggerAuthentications in namespace - %s with err - %s", namespace.Name, err)

		crTotals[prommetrics.TriggerAuthenticationResource][namespaceName] = len(triggerAuthList.Items)
	}

	return triggerTotals, crTotals
}

func testWebhookMetricValues(t *testing.T) {
	families := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaWebhookPrometheusURL))
	checkWebhookValues(t, families)
}

func testOperatorMetricValues(t *testing.T, kc *kubernetes.Clientset) {
	families := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorPrometheusURL))
	expectedTriggerTotals, expectedCrTotals := getOperatorMetricsManually(t, kc)

	checkTriggerTotalValues(t, families, expectedTriggerTotals)
	checkCRTotalValues(t, families, expectedCrTotals)
}

func checkTriggerTotalValues(t *testing.T, families map[string]*promModel.MetricFamily, expected map[string]int) {
	t.Log("--- testing trigger total metrics ---")

	family, ok := families["keda_trigger_totals"]
	if !ok {
		t.Errorf("metric not available")
		return
	}

	metrics := family.GetMetric()
	for _, metric := range metrics {
		labels := metric.GetLabel()
		for _, label := range labels {
			if *label.Name == "type" {
				triggerType := *label.Value
				metricValue := *metric.Gauge.Value
				expectedMetricValue := float64(expected[triggerType])

				assert.Equalf(t, expectedMetricValue, metricValue, "expected %f got %f for trigger type %s",
					expectedMetricValue, metricValue, triggerType)

				delete(expected, triggerType)
			}
		}
	}

	assert.Equal(t, 0, len(expected))
}

func checkCRTotalValues(t *testing.T, families map[string]*promModel.MetricFamily, expected map[string]map[string]int) {
	t.Log("--- testing resource total metrics ---")

	family, ok := families["keda_resource_totals"]
	if !ok {
		t.Errorf("metric not available")
		return
	}

	metrics := family.GetMetric()
	for _, metric := range metrics {
		labels := metric.GetLabel()
		var namespace, crType string
		for _, label := range labels {
			if *label.Name == "type" {
				crType = *label.Value
			} else if *label.Name == namespaceString {
				namespace = *label.Value
			}
		}

		metricValue := *metric.Gauge.Value
		expectedMetricValue := float64(expected[crType][namespace])

		assert.Equalf(t, expectedMetricValue, metricValue, "expected %f got %f for cr type %s & namespace %s",
			expectedMetricValue, metricValue, crType, namespace)
	}
}

func checkWebhookValues(t *testing.T, families map[string]*promModel.MetricFamily) {
	t.Log("--- testing webhook metrics ---")

	family, ok := families["keda_webhook_scaled_object_validation_errors"]
	if !ok {
		t.Errorf("metric keda_webhook_scaled_object_validation_errors not available")
		return
	}

	metricValue := 0.0
	metrics := family.GetMetric()
	for _, metric := range metrics {
		labels := metric.GetLabel()
		for _, label := range labels {
			if *label.Name == namespaceString && *label.Value != testNamespace {
				continue
			}
		}
		metricValue += *metric.Counter.Value
	}
	assert.GreaterOrEqual(t, metricValue, 1.0, "keda_webhook_scaled_object_validation_errors has to be greater than 0")

	family, ok = families["keda_webhook_scaled_object_validation_total"]
	if !ok {
		t.Errorf("metric keda_webhook_scaled_object_validation_total not available")
		return
	}

	metricValue = 0.0
	metrics = family.GetMetric()
	for _, metric := range metrics {
		labels := metric.GetLabel()
		for _, label := range labels {
			if *label.Name == namespaceString && *label.Value != testNamespace {
				continue
			}
		}
		metricValue += *metric.Counter.Value
	}
	assert.GreaterOrEqual(t, metricValue, 1.0, "keda_webhook_scaled_object_validation_total has to be greater than 0")
}
