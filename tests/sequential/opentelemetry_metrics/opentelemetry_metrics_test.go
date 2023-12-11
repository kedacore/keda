//go:build e2e
// +build e2e

package opentelemetry_metrics_test

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	prommodel "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kedacore/keda/v2/pkg/metricscollector"
	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName              = "opentelemetry-metrics-test"
	labelScaledObject     = "scaledObject"
	labelType             = "type"
	labelCloudEventSource = "cloudEventSource"
	eventsink             = "eventsink"
	eventsinkValue        = "opentelemetry-metrics-test-ce"
	eventsinkType         = "eventsinktype"
	eventsinkTypeValue    = "http"
)

var (
	testNamespace                            = fmt.Sprintf("%s-ns", testName)
	deploymentName                           = fmt.Sprintf("%s-deployment", testName)
	monitoredDeploymentName                  = fmt.Sprintf("%s-monitored", testName)
	scaledObjectName                         = fmt.Sprintf("%s-so", testName)
	wrongScaledObjectName                    = fmt.Sprintf("%s-wrong", testName)
	wrongScalerName                          = fmt.Sprintf("%s-wrong-scaler", testName)
	cronScaledJobName                        = fmt.Sprintf("%s-cron-sj", testName)
	clientName                               = fmt.Sprintf("%s-client", testName)
	cloudEventSourceName                     = fmt.Sprintf("%s-ce", testName)
	wrongCloudEventSourceName                = fmt.Sprintf("%s-ce-w", testName)
	cloudEventHTTPReceiverName               = fmt.Sprintf("%s-cloudevent-http-receiver", testName)
	cloudEventHTTPServiceName                = fmt.Sprintf("%s-cloudevent-http-service", testName)
	cloudEventHTTPServiceURL                 = fmt.Sprintf("http://%s.%s.svc.cluster.local:8899", cloudEventHTTPServiceName, testNamespace)
	kedaOperatorCollectorPrometheusExportURL = "http://opentelemetry-collector.open-telemetry-system.svc.cluster.local:8889/metrics"
	namespaceString                          = "namespace"
	kedaNamespace                            = "keda"
	kedaOperatorDeploymentName               = "keda-operator"
	operatorLabelSelector                    = "app=keda-operator"
)

type templateData struct {
	TestName                   string
	TestNamespace              string
	DeploymentName             string
	ScaledObjectName           string
	WrongScaledObjectName      string
	WrongScalerName            string
	CronScaledJobName          string
	MonitoredDeploymentName    string
	ClientName                 string
	CloudEventSourceName       string
	WrongCloudEventSourceName  string
	CloudEventHTTPReceiverName string
	CloudEventHTTPServiceName  string
	CloudEventHTTPServiceURL   string
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
          image: nginxinc/nginx-unprivileged
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
          image: nginxinc/nginx-unprivileged
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

	wrongScaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.WrongScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 2
  idleReplicaCount: 0
  minReplicaCount: 1
  maxReplicaCount: 2
  cooldownPeriod: 10
  triggers:
    - type: prometheus
      name: {{.WrongScalerName}}
      metadata:
        serverAddress: http://keda-prometheus.keda.svc.cluster.local:8080
        metricName: keda_scaler_errors_total
        threshold: '1'
        query: 'keda_scaler_errors_total{namespace="{{.TestNamespace}}",scaledObject="{{.WrongScaledObjectName}}"}'
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
	scaledObjectPausedTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  annotations:
    autoscaling.keda.sh/paused-replicas: "2"
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

	cloudEventSourceTemplate = `
apiVersion: eventing.keda.sh/v1alpha1
kind: CloudEventSource
metadata:
  name: {{.CloudEventSourceName}}
  namespace: {{.TestNamespace}}
spec:
  clusterName: cluster-sample
  destination:
    http:
      uri: {{.CloudEventHTTPServiceURL}}
`
	wrongCloudEventSourceTemplate = `
apiVersion: eventing.keda.sh/v1alpha1
kind: CloudEventSource
metadata:
  name: {{.WrongCloudEventSourceName}}
  namespace: {{.TestNamespace}}
spec:
  clusterName: cluster-sample
  destination:
    http:
      uri: http://fo.wo
`

	cloudEventHTTPServiceTemplate = `
  apiVersion: v1
  kind: Service
  metadata:
    name: {{.CloudEventHTTPServiceName}}
    namespace: {{.TestNamespace}}
  spec:
    type: ClusterIP
    ports:
    - protocol: TCP
      port: 8899
      targetPort: 8899
    selector:
      app: {{.CloudEventHTTPReceiverName}}
  `

	cloudEventHTTPReceiverTemplate = `
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    labels:
      deploy: {{.CloudEventHTTPReceiverName}}
    name: {{.CloudEventHTTPReceiverName}}
    namespace: {{.TestNamespace}}
  spec:
    selector:
      matchLabels:
        app: {{.CloudEventHTTPReceiverName}}
    replicas: 1
    template:
      metadata:
        labels:
          app: {{.CloudEventHTTPReceiverName}}
      spec:
        containers:
        - name: httpreceiver
          image: ghcr.io/kedacore/tests-cloudevents-http:latest
          ports:
          - containerPort: 8899
          resources:
            requests:
              cpu: "200m"
            limits:
              cpu: "500m"
  `
)

func TestOpenTelemetryMetrics(t *testing.T) {
	// setup
	t.Log("--- setting up ---")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// restart KEDA operator to ensure that all the metrics are sent to the collector
	DeletePodsInNamespaceBySelector(t, kc, operatorLabelSelector, kedaNamespace)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, kedaOperatorDeploymentName, kedaNamespace, 1, 60, 2),
		"replica count should be 1 after 2 minute")

	// scaling to max replica count to ensure the counter is registered before we test it
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 2),
		"replica count should be 2 after 2 minute")

	testScalerMetricValue(t)
	testScalerMetricLatency(t)
	testScalerActiveMetric(t)
	testScaledObjectErrors(t, data)
	testScalerErrors(t, data)
	testOperatorMetrics(t, kc, data)
	testScalableObjectMetrics(t)
	testScaledObjectPausedMetric(t, data)
	testCloudEventEmitted(t, data)
	testCloudEventEmittedError(t, data)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestName:                   testName,
			TestNamespace:              testNamespace,
			DeploymentName:             deploymentName,
			ScaledObjectName:           scaledObjectName,
			WrongScaledObjectName:      wrongScaledObjectName,
			WrongScalerName:            wrongScalerName,
			MonitoredDeploymentName:    monitoredDeploymentName,
			ClientName:                 clientName,
			CronScaledJobName:          cronScaledJobName,
			CloudEventSourceName:       cloudEventSourceName,
			WrongCloudEventSourceName:  wrongCloudEventSourceName,
			CloudEventHTTPReceiverName: cloudEventHTTPReceiverName,
			CloudEventHTTPServiceName:  cloudEventHTTPServiceName,
			CloudEventHTTPServiceURL:   cloudEventHTTPServiceURL,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "monitoredDeploymentTemplate", Config: monitoredDeploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
			{Name: "clientTemplate", Config: clientTemplate},
			{Name: "authenticatioNTemplate", Config: authenticationTemplate},
			{Name: "cloudEventHTTPReceiverTemplate", Config: cloudEventHTTPReceiverTemplate},
			{Name: "cloudEventHTTPServiceTemplate", Config: cloudEventHTTPServiceTemplate},
		}
}

func fetchAndParsePrometheusMetrics(t *testing.T, cmd string) map[string]*prommodel.MetricFamily {
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
	family := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorCollectorPrometheusExportURL))

	val, ok := family["keda_scaler_metrics_value"]
	assert.True(t, ok, "keda_scaler_metrics_value not available")
	if ok {
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
	}
}

func testScaledObjectErrors(t *testing.T, data templateData) {
	t.Log("--- testing scaled object errors ---")

	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	time.Sleep(2 * time.Second)
	KubectlApplyWithTemplate(t, data, "wrongScaledObjectTemplate", wrongScaledObjectTemplate)

	time.Sleep(20 * time.Second)

	family := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorCollectorPrometheusExportURL))
	val, ok := family["keda_scaledobject_errors_total"]
	assert.True(t, ok, "keda_scaledobject_errors_total not available")
	if ok {
		errCounterVal1 := getErrorMetricsValue(val)

		// wait for 2 seconds as pollinginterval is 2
		time.Sleep(5 * time.Second)

		family = fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorCollectorPrometheusExportURL))
		val, ok := family["keda_scaledobject_errors_total"]
		assert.True(t, ok, "keda_scaledobject_errors_total not available")
		if ok {
			errCounterVal2 := getErrorMetricsValue(val)
			assert.NotEqual(t, errCounterVal2, float64(0))
			assert.GreaterOrEqual(t, errCounterVal2, errCounterVal1)
		}
	}

	KubectlDeleteWithTemplate(t, data, "wrongScaledObjectTemplate", wrongScaledObjectTemplate)
	time.Sleep(2 * time.Second)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	// wait for 10 seconds to correctly fetch metrics.
	time.Sleep(10 * time.Second)
}

func testScalerErrors(t *testing.T, data templateData) {
	t.Log("--- testing scaler errors ---")

	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	time.Sleep(2 * time.Second)
	KubectlApplyWithTemplate(t, data, "wrongScaledObjectTemplate", wrongScaledObjectTemplate)

	time.Sleep(15 * time.Second)

	family := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorCollectorPrometheusExportURL))
	val, ok := family["keda_scaler_errors_total"]
	assert.True(t, ok, "keda_scaler_errors_total not available")
	if ok {
		errCounterVal1 := getErrorMetricsValue(val)

		// wait for 10 seconds to correctly fetch metrics.
		time.Sleep(5 * time.Second)

		family = fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorCollectorPrometheusExportURL))
		val, ok := family["keda_scaler_errors_total"]
		assert.True(t, ok, "keda_scaler_errors_total not available")
		if ok {
			errCounterVal2 := getErrorMetricsValue(val)
			assert.NotEqual(t, errCounterVal2, float64(0))
			assert.GreaterOrEqual(t, errCounterVal2, errCounterVal1)
		}
	}

	KubectlDeleteWithTemplate(t, data, "wrongScaledObjectTemplate", wrongScaledObjectTemplate)
	time.Sleep(2 * time.Second)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
}

func getErrorMetricsValue(val *prommodel.MetricFamily) float64 {
	switch val.GetName() {
	case "keda_scaledobject_errors_total":
		metrics := val.GetMetric()
		for _, metric := range metrics {
			labels := metric.GetLabel()
			for _, label := range labels {
				if *label.Name == "scaledObject" && *label.Value == wrongScaledObjectName {
					return *metric.Counter.Value
				}
			}
		}
	case "keda_scaler_errors_total":
		metrics := val.GetMetric()
		for _, metric := range metrics {
			labels := metric.GetLabel()
			for _, label := range labels {
				if *label.Name == "scaler" && *label.Value == wrongScalerName {
					return *metric.Counter.Value
				}
			}
		}
	}
	return 0
}

func testScalerMetricLatency(t *testing.T) {
	t.Log("--- testing scaler metric latency ---")

	family := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorCollectorPrometheusExportURL))

	val, ok := family["keda_scaler_metrics_latency"]
	assert.True(t, ok, "keda_scaler_metrics_latency not available")
	if ok {
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
	}
}

func testScalableObjectMetrics(t *testing.T) {
	t.Log("--- testing scalable objects latency ---")

	family := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorCollectorPrometheusExportURL))

	val, ok := family["keda_internal_scale_loop_latency"]
	assert.True(t, ok, "keda_internal_scale_loop_latency not available")
	if ok {
		var found bool
		metrics := val.GetMetric()

		// check scaledobject loop
		found = false
		for _, metric := range metrics {
			labels := metric.GetLabel()
			for _, label := range labels {
				if *label.Name == labelType && *label.Value == "scaledobject" {
					found = true
				}
			}
		}
		assert.Equal(t, true, found)

		// check scaledjob loop
		found = false
		for _, metric := range metrics {
			labels := metric.GetLabel()
			for _, label := range labels {
				if *label.Name == labelType && *label.Value == "scaledjob" {
					found = true
				}
			}
		}
		assert.Equal(t, true, found)
	}
}

func testScalerActiveMetric(t *testing.T) {
	t.Log("--- testing scaler active metric ---")

	family := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorCollectorPrometheusExportURL))

	val, ok := family["keda_scaler_active"]
	assert.True(t, ok, "keda_scaler_active not available")
	if ok {
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
	}
}

func testScaledObjectPausedMetric(t *testing.T, data templateData) {
	t.Log("--- testing scaleobject pause metric ---")

	// Pause the ScaledObject
	KubectlApplyWithTemplate(t, data, "scaledObjectPausedTemplate", scaledObjectPausedTemplate)

	time.Sleep(20 * time.Second)
	// Check that the paused metric is now true
	families := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorCollectorPrometheusExportURL))
	assertScaledObjectPausedMetric(t, families, scaledObjectName, true)

	// Unpause the ScaledObject
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	time.Sleep(20 * time.Second)
	// Check that the paused metric is back to false
	families = fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorCollectorPrometheusExportURL))
	assertScaledObjectPausedMetric(t, families, scaledObjectName, false)
}

func testOperatorMetrics(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing operator metrics ---")
	testOperatorMetricValues(t, kc)

	KubectlApplyWithTemplate(t, data, "cronScaledJobTemplate", cronScaledJobTemplate)
	testOperatorMetricValues(t, kc)

	KubectlDeleteWithTemplate(t, data, "cronScaledJobTemplate", cronScaledJobTemplate)
	testOperatorMetricValues(t, kc)
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
		crTotals[metricscollector.ClusterTriggerAuthenticationResource][namespace]++
	}

	for _, namespace := range namespaceList.Items {
		namespaceName := namespace.Name
		if namespace.Name == "" {
			namespaceName = "default"
		}

		scaledObjectList, err := kedaKc.ScaledObjects(namespace.Name).List(context.Background(), v1.ListOptions{})
		assert.NoErrorf(t, err, "failed to list scaledObjects in namespace - %s with err - %s", namespace.Name, err)

		crTotals[metricscollector.ScaledObjectResource][namespaceName] = len(scaledObjectList.Items)
		for _, scaledObject := range scaledObjectList.Items {
			for _, trigger := range scaledObject.Spec.Triggers {
				triggerTotals[trigger.Type]++
			}
		}

		scaledJobList, err := kedaKc.ScaledJobs(namespace.Name).List(context.Background(), v1.ListOptions{})
		assert.NoErrorf(t, err, "failed to list scaledJobs in namespace - %s with err - %s", namespace.Name, err)

		crTotals[metricscollector.ScaledJobResource][namespaceName] = len(scaledJobList.Items)
		for _, scaledJob := range scaledJobList.Items {
			for _, trigger := range scaledJob.Spec.Triggers {
				triggerTotals[trigger.Type]++
			}
		}

		triggerAuthList, err := kedaKc.TriggerAuthentications(namespace.Name).List(context.Background(), v1.ListOptions{})
		assert.NoErrorf(t, err, "failed to list triggerAuthentications in namespace - %s with err - %s", namespace.Name, err)

		crTotals[metricscollector.TriggerAuthenticationResource][namespaceName] = len(triggerAuthList.Items)
	}

	return triggerTotals, crTotals
}

func testOperatorMetricValues(t *testing.T, kc *kubernetes.Clientset) {
	// wait for 5 seconds to correctly fetch metrics.
	time.Sleep(5 * time.Second)

	families := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorCollectorPrometheusExportURL))
	expectedTriggerTotals, expectedCrTotals := getOperatorMetricsManually(t, kc)

	checkTriggerTotalValues(t, families, expectedTriggerTotals)
	checkCRTotalValues(t, families, expectedCrTotals)
	checkBuildInfo(t, families)
}

func checkBuildInfo(t *testing.T, families map[string]*prommodel.MetricFamily) {
	t.Log("--- testing build info metric ---")

	family, ok := families["keda_build_info"]
	assert.True(t, ok, "keda_build_info not available")
	if !ok {
		return
	}

	latestCommit := getLatestCommit(t)
	expected := map[string]string{
		"git_commit": latestCommit,
		"goos":       "linux",
	}

	metrics := family.GetMetric()
	for _, metric := range metrics {
		labels := metric.GetLabel()
		for _, labelPair := range labels {
			if expectedValue, ok := expected[*labelPair.Name]; ok {
				assert.EqualValues(t, expectedValue, *labelPair.Value, "values do not match for label %s", *labelPair.Name)
			}
		}
		assert.EqualValues(t, 1, metric.GetGauge().GetValue())
	}
}

func getLatestCommit(t *testing.T) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	require.NoError(t, err)

	return strings.Trim(out.String(), "\n")
}

func checkTriggerTotalValues(t *testing.T, families map[string]*prommodel.MetricFamily, expected map[string]int) {
	t.Log("--- testing trigger total metrics ---")

	family, ok := families["keda_trigger_totals"]
	assert.True(t, ok, "keda_trigger_totals not available")
	if !ok {
		return
	}

	metrics := family.GetMetric()
	for _, metric := range metrics {
		labels := metric.GetLabel()
		for _, label := range labels {
			if *label.Name == labelType {
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

func checkCRTotalValues(t *testing.T, families map[string]*prommodel.MetricFamily, expected map[string]map[string]int) {
	t.Log("--- testing resource total metrics ---")

	family, ok := families["keda_resource_totals"]
	assert.True(t, ok, "keda_resource_totals not available")
	if !ok {
		return
	}

	metrics := family.GetMetric()
	for _, metric := range metrics {
		labels := metric.GetLabel()
		var namespace, crType string
		for _, label := range labels {
			if *label.Name == labelType {
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

func assertScaledObjectPausedMetric(t *testing.T, families map[string]*prommodel.MetricFamily, scaledObjectName string, expected bool) {
	family, ok := families["keda_scaled_object_paused"]
	assert.True(t, ok, "keda_scaled_object_paused not available")
	if !ok {
		return
	}

	metricValue := 0.0
	metrics := family.GetMetric()
	for _, metric := range metrics {
		labels := metric.GetLabel()
		for _, label := range labels {
			if *label.Name == labelScaledObject && *label.Value == scaledObjectName {
				metricValue = *metric.Gauge.Value
			}
		}
	}
	expectedMetricValue := 0
	if expected {
		expectedMetricValue = 1
	}
	assert.Equal(t, float64(expectedMetricValue), metricValue)
}

func testCloudEventEmitted(t *testing.T, data templateData) {
	t.Log("--- testing cloudevent emitted ---")

	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlApplyWithTemplate(t, data, "cloudEventSourceTemplate", cloudEventSourceTemplate)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	time.Sleep(10 * time.Second)
	family := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorCollectorPrometheusExportURL))

	if val, ok := family["keda_cloudeventsource_events_emitted_count_total"]; ok {
		var found bool
		metrics := val.GetMetric()
		for _, metric := range metrics {
			labels := metric.GetLabel()
			if len(labels) >= 5 &&
				*labels[0].Value == "opentelemetry-metrics-test-ce" &&
				*labels[1].Value == "http" &&
				*labels[3].Value == "opentelemetry-metrics-test-ns" &&
				*labels[4].Value == "emitted" {
				assert.GreaterOrEqual(t, *metric.Counter.Value, float64(1))
				found = true
			}
		}
		assert.Equal(t, true, found)
	} else {
		t.Errorf("metric not available")
	}
}

func testCloudEventEmittedError(t *testing.T, data templateData) {
	t.Log("--- testing cloudevent emitted error ---")

	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlDeleteWithTemplate(t, data, "cloudEventSourceTemplate", cloudEventSourceTemplate)
	KubectlApplyWithTemplate(t, data, "wrongCloudEventSourceTemplate", wrongCloudEventSourceTemplate)
	time.Sleep(1 * time.Second)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	time.Sleep(10 * time.Second)
	family := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorCollectorPrometheusExportURL))

	if val, ok := family["keda_cloudeventsource_events_emitted_count_total"]; ok {
		var found bool
		metrics := val.GetMetric()
		for _, metric := range metrics {
			labels := metric.GetLabel()
			if len(labels) >= 5 &&
				*labels[0].Value == "opentelemetry-metrics-test-ce-w" &&
				*labels[1].Value == "http" &&
				*labels[3].Value == "opentelemetry-metrics-test-ns" &&
				*labels[4].Value == "failed" {
				assert.GreaterOrEqual(t, *metric.Counter.Value, float64(5))
				found = true
			}
		}
		assert.Equal(t, true, found)
	} else {
		t.Errorf("metric not available")
	}

	KubectlDeleteWithTemplate(t, data, "wrongCloudEventSourceTemplate", wrongCloudEventSourceTemplate)
	KubectlApplyWithTemplate(t, data, "cloudEventSourceTemplate", cloudEventSourceTemplate)
}
