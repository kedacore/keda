//go:build e2e
// +build e2e

package prometheus_metrics_test

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"os/exec"
	"strings"
	"testing"
	"time"

	prommodel "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kedacore/keda/v2/pkg/metricscollector"
	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName              = "prometheus-metrics-test"
	labelScaledObject     = "scaledObject"
	labelScaledJob        = "scaledJob"
	labelType             = "type"
	labelCloudEventSource = "cloudeventsource"
	eventsink             = "eventsink"
	eventsinkValue        = "prometheus-metrics-test-ce"
	eventsinkType         = "eventsinktype"
	eventsinkTypeValue    = "http"

	// The operator records a metric as it reconciles and polls, so a value only becomes
	// observable some time after the cluster state changed. Nothing buffers the metrics in
	// between - the operator's own endpoint is scraped directly - so the wait only has to
	// cover a reconcile and a poll. The scaled objects here poll every 2-5s, so a minute is
	// more than ten of those cycles and is only reached when something is genuinely broken.
	metricWaitTimeout = time.Minute

	// Each poll of the operator totals lists every ScaledObject, ScaledJob and
	// TriggerAuthentication in the cluster, so it runs less aggressively than a plain
	// metric read.
	operatorMetricsInterval = 5 * time.Second
)

var (
	testNamespace                  = fmt.Sprintf("%s-ns", testName)
	deploymentName                 = fmt.Sprintf("%s-deployment", testName)
	monitoredDeploymentName        = fmt.Sprintf("%s-monitored", testName)
	scaledObjectName               = fmt.Sprintf("%s-so", testName)
	resourceMetricDeploymentName   = fmt.Sprintf("%s-resource-deployment", testName)
	resourceMetricScaledObjectName = fmt.Sprintf("%s-resource-so", testName)
	resourceMetricScalerName       = fmt.Sprintf("%s-resource-cpu-scaler", testName)
	httpClientScaledObjectName     = fmt.Sprintf("%s-so-http-client", testName)
	wrongScaledObjectName          = fmt.Sprintf("%s-so-wrong", testName)
	scaledJobName                  = fmt.Sprintf("%s-sj", testName)
	wrongScaledJobName             = fmt.Sprintf("%s-sj-wrong", testName)
	wrongScalerName                = fmt.Sprintf("%s-wrong-scaler", testName)
	emptyUpstreamScaledObjectName  = fmt.Sprintf("%s-so-empty-upstream", testName)
	httpClientScalerName           = fmt.Sprintf("%s-http-client-scaler", testName)
	cronScaledJobName              = fmt.Sprintf("%s-cron-sj", testName)
	clientName                     = fmt.Sprintf("%s-client", testName)
	cloudEventSourceName           = fmt.Sprintf("%s-ce", testName)
	wrongCloudEventSourceName      = fmt.Sprintf("%s-ce-w", testName)
	cloudEventHTTPReceiverName     = fmt.Sprintf("%s-cloudevent-http-receiver", testName)
	cloudEventHTTPServiceName      = fmt.Sprintf("%s-cloudevent-http-service", testName)
	cloudEventHTTPServiceURL       = fmt.Sprintf("http://%s.%s.svc.cluster.local:8899", cloudEventHTTPServiceName, testNamespace)
	kedaOperatorPrometheusURL      = "http://keda-operator.keda.svc.cluster.local:8080/metrics"
	kedaMetricsServerPrometheusURL = "http://keda-metrics-apiserver.keda.svc.cluster.local:8080/metrics"
	kedaWebhookPrometheusURL       = "http://keda-admission-webhooks.keda.svc.cluster.local:8080/metrics"
	namespaceString                = "namespace"
)

type templateData struct {
	TestName                       string
	TestNamespace                  string
	DeploymentName                 string
	ScaledObjectName               string
	ResourceMetricDeploymentName   string
	ResourceMetricScaledObjectName string
	ResourceMetricScalerName       string
	HTTPClientScaledObjectName     string
	ScaledJobName                  string
	WrongScaledObjectName          string
	WrongScaledJobName             string
	WrongScalerName                string
	EmptyUpstreamScaledObjectName  string
	HTTPClientScalerName           string
	CronScaledJobName              string
	MonitoredDeploymentName        string
	ClientName                     string
	CloudEventSourceName           string
	WrongCloudEventSourceName      string
	CloudEventHTTPReceiverName     string
	CloudEventHTTPServiceName      string
	CloudEventHTTPServiceURL       string
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
          image: ghcr.io/nginx/nginx-unprivileged:1.26
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
          image: ghcr.io/nginx/nginx-unprivileged:1.26
`

	resourceMetricDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.ResourceMetricDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.ResourceMetricDeploymentName}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.ResourceMetricDeploymentName}}
  template:
    metadata:
      labels:
        app: {{.ResourceMetricDeploymentName}}
    spec:
      containers:
        - name: {{.ResourceMetricDeploymentName}}
          image: ghcr.io/nginx/nginx-unprivileged:1.26
          resources:
            requests:
              cpu: "200m"
            limits:
              cpu: "500m"
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

	resourceMetricScaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ResourceMetricScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.ResourceMetricDeploymentName}}
  pollingInterval: 5
  minReplicaCount: 1
  maxReplicaCount: 2
  triggers:
    - type: cpu
      name: {{.ResourceMetricScalerName}}
      metricType: Utilization
      metadata:
        value: "50"
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

	httpClientScaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.HTTPClientScaledObjectName}}
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
      name: {{.HTTPClientScalerName}}
      metadata:
        serverAddress: http://keda-prometheus.keda.svc.cluster.local:8080
        metricName: keda_scaler_errors_total
        threshold: '1'
        query: 'keda_scaler_errors_total{namespace="{{.TestNamespace}}",scaledObject="{{.HTTPClientScaledObjectName}}"}'
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
    - type: kubernetes-workload
      metadata:
        podSelector: 'app={{.MonitoredDeploymentName}}'
        value: '1'
`

	wrongScaledJobTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{.WrongScaledJobName}}
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
  pollingInterval: 2
  maxReplicaCount: 3
  successfulJobsHistoryLimit: 0
  failedJobsHistoryLimit: 0
  triggers:
    - type: prometheus
      name: {{.WrongScalerName}}
      metadata:
        serverAddress: http://keda-prometheus.keda.svc.cluster.local:8080
        metricName: keda_scaler_errors_total
        threshold: '1'
        query: 'keda_scaler_errors_total{namespace="{{.TestNamespace}}",scaledJob="{{.WrongScaledJobName}}"}'
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
    image: docker.io/curlimages/curl
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

	emptyUpstreamPrometheusConfigMapTemplate = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-empty-upstream-config
  namespace: {{.TestNamespace}}
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
      evaluation_interval: 15s
`

	emptyUpstreamPrometheusDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus-empty-upstream
  namespace: {{.TestNamespace}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: prometheus-empty-upstream
  template:
    metadata:
      labels:
        app: prometheus-empty-upstream
    spec:
      containers:
        - name: prometheus
          image: docker.io/prom/prometheus:v2.47.1
          args:
            - --config.file=/etc/config/prometheus.yml
          ports:
            - containerPort: 9090
          volumeMounts:
            - name: config
              mountPath: /etc/config
      volumes:
        - name: config
          configMap:
            name: prometheus-empty-upstream-config
`

	emptyUpstreamPrometheusServiceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: prometheus-empty-upstream
  namespace: {{.TestNamespace}}
spec:
  selector:
    app: prometheus-empty-upstream
  ports:
    - port: 9090
      targetPort: 9090
`

	emptyUpstreamResponseScaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.EmptyUpstreamScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 2
  idleReplicaCount: 0
  minReplicaCount: 0
  maxReplicaCount: 2
  cooldownPeriod: 10
  triggers:
    - type: prometheus
      name: empty-upstream-trigger
      metadata:
        serverAddress: http://prometheus-empty-upstream.{{.TestNamespace}}.svc.cluster.local:9090
        threshold: '1'
        query: 'nonexistent_metric_empty_upstream_response'
        ignoreNullValues: 'false'
`
)

func TestPrometheusMetrics(t *testing.T) {
	// setup
	t.Log("--- setting up ---")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	// A metric that never converges fails the test from inside a helper, which ends the whole
	// function, so teardown has to be deferred rather than run at the end. Otherwise a single
	// timeout leaks this namespace into the sequential tests that run after this one.
	defer DeleteKubernetesResources(t, testNamespace, data, templates)

	// scaling to max replica count to ensure the counter is registered before we test it
	require.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 2),
		"replica count should be 2 after 2 minute")

	testScalerMetricValue(t)
	testScalerMetricLatency(t)
	testScalerActiveMetric(t)
	testScaledObjectErrors(t, data)
	testScaledJobErrors(t, data)
	testScalerErrors(t, data)
	testOperatorMetrics(t, kc, data)
	testMetricServerMetrics(t)
	testWebhookMetrics(t, data)
	testScalableObjectMetrics(t)
	testScaledObjectPausedMetric(t, data)
	testCloudEventEmitted(t, data)
	testCloudEventEmittedError(t, data)
	testEmptyUpstreamResponse(t, data)
	testHTTPClientMetrics(t, kc, data)
	testHighCardinalityLabelsDisabled(t, kc, data)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestName:                       testName,
			TestNamespace:                  testNamespace,
			DeploymentName:                 deploymentName,
			ScaledObjectName:               scaledObjectName,
			ResourceMetricDeploymentName:   resourceMetricDeploymentName,
			ResourceMetricScaledObjectName: resourceMetricScaledObjectName,
			ResourceMetricScalerName:       resourceMetricScalerName,
			HTTPClientScaledObjectName:     httpClientScaledObjectName,
			WrongScaledObjectName:          wrongScaledObjectName,
			ScaledJobName:                  scaledJobName,
			WrongScaledJobName:             wrongScaledJobName,
			WrongScalerName:                wrongScalerName,
			EmptyUpstreamScaledObjectName:  emptyUpstreamScaledObjectName,
			HTTPClientScalerName:           httpClientScalerName,
			MonitoredDeploymentName:        monitoredDeploymentName,
			ClientName:                     clientName,
			CronScaledJobName:              cronScaledJobName,
			CloudEventSourceName:           cloudEventSourceName,
			WrongCloudEventSourceName:      wrongCloudEventSourceName,
			CloudEventHTTPReceiverName:     cloudEventHTTPReceiverName,
			CloudEventHTTPServiceName:      cloudEventHTTPServiceName,
			CloudEventHTTPServiceURL:       cloudEventHTTPServiceURL,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "monitoredDeploymentTemplate", Config: monitoredDeploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
			{Name: "resourceMetricDeploymentTemplate", Config: resourceMetricDeploymentTemplate},
			{Name: "resourceMetricScaledObjectTemplate", Config: resourceMetricScaledObjectTemplate},
			{Name: "scaledJobTemplate", Config: scaledJobTemplate},
			{Name: "clientTemplate", Config: clientTemplate},
			{Name: "authenticatioNTemplate", Config: authenticationTemplate},
			{Name: "cloudEventHTTPReceiverTemplate", Config: cloudEventHTTPReceiverTemplate},
			{Name: "cloudEventHTTPServiceTemplate", Config: cloudEventHTTPServiceTemplate},
		}
}

func fetchAndParsePrometheusMetrics(t *testing.T, cmd string) map[string]*prommodel.MetricFamily {
	families, err := fetchPrometheusMetrics(t, cmd)
	assert.NoErrorf(t, err, "cannot scrape metrics - %s", err)

	return families
}

// fetchPrometheusMetrics reports a failed scrape instead of failing the test, so that callers
// polling an endpoint can treat it as "not yet" and try again. A pod that is still rolling out
// refuses the connection, which is a normal thing to observe part way through a wait.
func fetchPrometheusMetrics(t *testing.T, cmd string) (map[string]*prommodel.MetricFamily, error) {
	out, _, err := ExecCommandOnSpecificPodWithoutTTY(t, clientName, testNamespace, cmd)
	if err != nil {
		return nil, fmt.Errorf("cannot execute command: %w", err)
	}

	parser := expfmt.NewTextParser(model.UTF8Validation)
	// Ensure EOL
	reader := strings.NewReader(strings.ReplaceAll(out, "\r\n", "\n"))
	families, err := parser.TextToMetricFamilies(reader)
	if err != nil {
		return nil, fmt.Errorf("cannot parse metrics: %w", err)
	}

	return families, nil
}

// WaitForPrometheusMetric waits for a specific metric to appear in the KEDA operator Prometheus endpoint
// and validates that the MetricFamily has certain conditions using the provided familyValidator function.
// Returns the parsed MetricFamily.
func WaitForPrometheusMetric(t *testing.T, metricToWaitFor string, familyValidator func(family *prommodel.MetricFamily) bool) map[string]*prommodel.MetricFamily {
	return WaitForPrometheusMetricAtURL(t, kedaOperatorPrometheusURL, metricToWaitFor, familyValidator)
}

// WaitForPrometheusMetricAtURL waits for a specific metric to appear in the provided Prometheus endpoint
// and validates that the MetricFamily has certain conditions using the provided familyValidator function.
// Returns the parsed MetricFamily.
//
// A metric that never arrives ends the test on the spot, so anything that has to be undone
// afterwards belongs in a defer.
func WaitForPrometheusMetricAtURL(t *testing.T, metricsURL string, metricToWaitFor string, familyValidator func(family *prommodel.MetricFamily) bool) map[string]*prommodel.MetricFamily {
	contextWithTimeout, cancel := context.WithTimeout(context.Background(), metricWaitTimeout)
	defer cancel()

	var families map[string]*prommodel.MetricFamily
	err := KedaEventually(contextWithTimeout, func(ctx context.Context) (bool, error) {
		t.Logf("Waiting for metric %s on %s", metricToWaitFor, metricsURL)

		scraped, scrapeErr := fetchPrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", metricsURL))
		if scrapeErr != nil {
			t.Logf("cannot scrape %s, retrying: %v", metricsURL, scrapeErr)
			return false, nil
		}
		families = scraped

		family, ok := families[metricToWaitFor]
		if !ok {
			return false, nil
		}

		return familyValidator(family), nil
	}, IntervalShort)
	if err != nil {
		// The timeout alone does not say how far off the metric was, and the test ends here.
		t.Logf("last observed %s: %v", metricToWaitFor, families[metricToWaitFor])
	}
	require.NoErrorf(t, err, "error waiting for metric %s", metricToWaitFor)

	return families
}

func metricFamilyCounterSumGreaterThanZero(family *prommodel.MetricFamily) bool {
	sum := 0.0
	for _, metric := range family.GetMetric() {
		sum += metric.GetCounter().GetValue()
	}
	return sum > 0
}

// testResourceGaugeValues returns the gauge of every series in family that belongs to the scaled
// object or the scaled job under test.
func testResourceGaugeValues(family *prommodel.MetricFamily) []float64 {
	var values []float64
	for _, metric := range family.GetMetric() {
		for _, label := range metric.GetLabel() {
			if (label.GetName() == labelScaledObject && label.GetValue() == scaledObjectName) ||
				(label.GetName() == labelScaledJob && label.GetValue() == scaledJobName) {
				values = append(values, metric.GetGauge().GetValue())
				break
			}
		}
	}
	return values
}

func allValuesEqual(values []float64, expected float64) bool {
	for _, value := range values {
		if value != expected {
			return false
		}
	}
	return true
}

func testScalerMetricValue(t *testing.T) {
	t.Log("--- testing scaler metric value ---")

	// The value is recorded when the scaler is polled, so it appears a poll cycle after the
	// scaled object was accepted rather than as soon as it exists.
	families := WaitForPrometheusMetric(t, "keda_scaler_metrics_value", func(family *prommodel.MetricFamily) bool {
		values := testResourceGaugeValues(family)
		return len(values) > 0 && allValuesEqual(values, 4)
	})

	values := testResourceGaugeValues(families["keda_scaler_metrics_value"])
	assert.NotEmpty(t, values, "no keda_scaler_metrics_value for %s or %s", scaledObjectName, scaledJobName)
	for _, value := range values {
		assert.Equal(t, float64(4), value)
	}
}

func testScaledObjectErrors(t *testing.T, data templateData) {
	t.Log("--- testing scaled object errors ---")

	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlApplyWithTemplate(t, data, "wrongScaledObjectTemplate", wrongScaledObjectTemplate)

	// Wait for the series to appear, then for it to keep climbing. Both steps have to be
	// separate waits: nesting one inside the other bounds the inner wait by the outer budget
	// and reports a failure for every attempt the outer wait was still allowed to retry.
	families := WaitForPrometheusMetric(t, "keda_scaled_object_errors_total", func(family *prommodel.MetricFamily) bool {
		return getErrorMetricsValue(family) > 0
	})
	errCounterVal1 := getErrorMetricsValue(families["keda_scaled_object_errors_total"])

	families = WaitForPrometheusMetric(t, "keda_scaled_object_errors_total", func(family *prommodel.MetricFamily) bool {
		return getErrorMetricsValue(family) > errCounterVal1
	})
	assert.Greater(t, getErrorMetricsValue(families["keda_scaled_object_errors_total"]), errCounterVal1)

	KubectlDeleteWithTemplate(t, data, "wrongScaledObjectTemplate", wrongScaledObjectTemplate)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
}

func testScaledJobErrors(t *testing.T, data templateData) {
	t.Log("--- testing scaled job errors ---")

	KubectlDeleteWithTemplate(t, data, "scaledJobTemplate", scaledJobTemplate)
	KubectlApplyWithTemplate(t, data, "wrongScaledJobTemplate", wrongScaledJobTemplate)

	families := WaitForPrometheusMetric(t, "keda_scaled_job_errors_total", func(family *prommodel.MetricFamily) bool {
		return getErrorMetricsValue(family) > 0
	})
	errCounterVal1 := getErrorMetricsValue(families["keda_scaled_job_errors_total"])

	families = WaitForPrometheusMetric(t, "keda_scaled_job_errors_total", func(family *prommodel.MetricFamily) bool {
		return getErrorMetricsValue(family) > errCounterVal1
	})
	assert.Greater(t, getErrorMetricsValue(families["keda_scaled_job_errors_total"]), errCounterVal1)

	KubectlDeleteWithTemplate(t, data, "wrongScaledJobTemplate", wrongScaledJobTemplate)
	KubectlApplyWithTemplate(t, data, "scaledJobTemplate", scaledJobTemplate)
}

func testScalerErrors(t *testing.T, data templateData) {
	t.Log("--- testing scaler errors ---")

	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlApplyWithTemplate(t, data, "wrongScaledObjectTemplate", wrongScaledObjectTemplate)
	KubectlDeleteWithTemplate(t, data, "scaledJobTemplate", scaledJobTemplate)
	KubectlApplyWithTemplate(t, data, "wrongScaledJobTemplate", wrongScaledJobTemplate)

	families := WaitForPrometheusMetric(t, "keda_scaler_detail_errors_total", func(family *prommodel.MetricFamily) bool {
		return getErrorMetricsValue(family) > 0
	})
	errCounterVal1 := getErrorMetricsValue(families["keda_scaler_detail_errors_total"])

	families = WaitForPrometheusMetric(t, "keda_scaler_detail_errors_total", func(family *prommodel.MetricFamily) bool {
		return getErrorMetricsValue(family) > errCounterVal1
	})
	assert.Greater(t, getErrorMetricsValue(families["keda_scaler_detail_errors_total"]), errCounterVal1)

	KubectlDeleteWithTemplate(t, data, "wrongScaledJobTemplate", wrongScaledJobTemplate)
	KubectlApplyWithTemplate(t, data, "scaledJobTemplate", scaledJobTemplate)
	KubectlDeleteWithTemplate(t, data, "wrongScaledObjectTemplate", wrongScaledObjectTemplate)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
}

func getErrorMetricsValue(val *prommodel.MetricFamily) float64 {
	switch val.GetName() {
	case "keda_scaler_detail_errors_total":
		metrics := val.GetMetric()
		result := 0.
		for _, metric := range metrics {
			result += metric.GetCounter().GetValue()
		}
		return result
	case "keda_scaled_object_errors_total":
		metrics := val.GetMetric()
		for _, metric := range metrics {
			labels := metric.GetLabel()
			for _, label := range labels {
				if *label.Name == "scaledObject" && *label.Value == wrongScaledObjectName {
					return *metric.Counter.Value
				}
			}
		}
	case "keda_scaled_job_errors_total":
		metrics := val.GetMetric()
		for _, metric := range metrics {
			labels := metric.GetLabel()
			for _, label := range labels {
				if *label.Name == "scaledJob" && *label.Value == wrongScaledJobName {
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

func assertScaledObjectPausedMetric(t *testing.T, families map[string]*prommodel.MetricFamily, scaledObjectName string, expected bool) {
	family, ok := families["keda_scaled_object_paused"]
	if !ok {
		t.Errorf("keda_scaled_object_paused metric not available")
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

func testScalerMetricLatency(t *testing.T) {
	t.Log("--- testing scaler metric latency ---")

	families := WaitForPrometheusMetric(t, "keda_scaler_metrics_latency_seconds", func(family *prommodel.MetricFamily) bool {
		return len(testResourceGaugeValues(family)) > 0
	})

	values := testResourceGaugeValues(families["keda_scaler_metrics_latency_seconds"])
	assert.NotEmpty(t, values, "no keda_scaler_metrics_latency_seconds for %s or %s", scaledObjectName, scaledJobName)
	for _, value := range values {
		assert.InDelta(t, float64(0), value, 0.001)
	}
}

func testScalableObjectMetrics(t *testing.T) {
	t.Log("--- testing scalable objects latency ---")

	// Each loop reports its own latency the first time it runs, and the two run independently,
	// so the scaledjob entry can trail the scaledobject one.
	families := WaitForPrometheusMetric(t, "keda_internal_scale_loop_latency_seconds", func(family *prommodel.MetricFamily) bool {
		return hasMetricWithTypeLabel(family, "scaledobject") && hasMetricWithTypeLabel(family, "scaledjob")
	})

	family := families["keda_internal_scale_loop_latency_seconds"]
	assert.True(t, hasMetricWithTypeLabel(family, "scaledobject"), "no scale loop latency reported for scaledobject")
	assert.True(t, hasMetricWithTypeLabel(family, "scaledjob"), "no scale loop latency reported for scaledjob")
}

func hasMetricWithTypeLabel(family *prommodel.MetricFamily, expectedType string) bool {
	for _, metric := range family.GetMetric() {
		if ExtractPrometheusLabelValue(labelType, metric.GetLabel()) == expectedType {
			return true
		}
	}
	return false
}

func testScalerActiveMetric(t *testing.T) {
	t.Log("--- testing scaler active metric ---")

	resourceScalerLabels := map[string]string{
		"namespace":    testNamespace,
		"scaledObject": resourceMetricScaledObjectName,
		"scaler":       resourceMetricScalerName,
		"triggerIndex": "0",
		"metric":       "cpu",
		"type":         "scaledobject",
	}

	// The CPU scaler and the scalers of the scaled object and the scaled job become active
	// independently, so the wait has to cover every series the assertions below read.
	families := WaitForPrometheusMetric(t, "keda_scaler_active", func(family *prommodel.MetricFamily) bool {
		values := testResourceGaugeValues(family)
		return hasMetricWithLabelsAndGauge(family, resourceScalerLabels, 1) &&
			len(values) > 0 && allValuesEqual(values, 1)
	})

	family := families["keda_scaler_active"]
	values := testResourceGaugeValues(family)
	assert.NotEmpty(t, values, "no keda_scaler_active for %s or %s", scaledObjectName, scaledJobName)
	for _, value := range values {
		assert.Equal(t, float64(1), value)
	}
	assert.True(t, hasMetricWithLabelsAndGauge(family, resourceScalerLabels, 1),
		"expected keda_scaler_active for CPU resource scaler")
}

func hasMetricWithLabelsAndGauge(family *prommodel.MetricFamily, expectedLabels map[string]string, expectedValue float64) bool {
	for _, metric := range family.GetMetric() {
		if metric.GetGauge().GetValue() == expectedValue && hasPrometheusLabels(metric.GetLabel(), expectedLabels) {
			return true
		}
	}
	return false
}

func hasPrometheusLabels(labels []*prommodel.LabelPair, expectedLabels map[string]string) bool {
	for name, value := range expectedLabels {
		if ExtractPrometheusLabelValue(name, labels) != value {
			return false
		}
	}
	return true
}

func testScaledObjectPausedMetric(t *testing.T, data templateData) {
	t.Log("--- testing scaleobject pause metric ---")

	// Pause the ScaledObject
	KubectlApplyWithTemplate(t, data, "scaledObjectPausedTemplate", scaledObjectPausedTemplate)

	// Check that the paused metric is now true
	families := WaitForPrometheusMetric(t, "keda_scaled_object_paused", func(family *prommodel.MetricFamily) bool {
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

		return metricValue == float64(1)
	})
	assertScaledObjectPausedMetric(t, families, scaledObjectName, true)

	// Unpause the ScaledObject
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	// Check that the paused metric is back to false
	families = WaitForPrometheusMetric(t, "keda_scaled_object_paused", func(family *prommodel.MetricFamily) bool {
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
		return metricValue == float64(0)
	})
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

func testWebhookMetricValues(t *testing.T) {
	families := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaWebhookPrometheusURL))
	checkWebhookValues(t, families)
}

func testMetricServerMetrics(t *testing.T) {
	families := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaMetricsServerPrometheusURL))
	checkMetricServerValues(t, families)
	checkGRPCClientMetrics(t, families)
}

type failureCollector struct {
	failures []string
}

func (f *failureCollector) Errorf(format string, args ...interface{}) {
	f.failures = append(f.failures, fmt.Sprintf(format, args...))
}

func testOperatorMetricValues(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing trigger and resource total metrics ---")

	ctx, cancel := context.WithTimeout(context.Background(), metricWaitTimeout)
	defer cancel()

	var (
		families map[string]*prommodel.MetricFamily
		attempt  failureCollector
	)

	// A created or deleted CR takes an unpredictable amount of time to reach the totals, since
	// the operator has to reconcile it before it records the new value. Retrying the assertions
	// themselves, rather than a predicate that has to be kept in step with them, means a
	// timeout still reports which total diverged.
	err := KedaEventually(ctx, func(_ context.Context) (bool, error) {
		scraped, scrapeErr := fetchPrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorPrometheusURL))
		if scrapeErr != nil {
			t.Logf("cannot scrape the operator, retrying: %v", scrapeErr)
			return false, nil
		}
		families = scraped
		expectedTriggerTotals, expectedCrTotals := getOperatorMetricsManually(t, kc)

		attempt = failureCollector{}
		checkTriggerTotalValues(&attempt, families, expectedTriggerTotals)
		checkCRTotalValues(&attempt, families, expectedCrTotals)

		return len(attempt.failures) == 0, nil
	}, operatorMetricsInterval)

	// Empty unless the last attempt failed, which only happens once the wait has given up.
	for _, failure := range attempt.failures {
		t.Error(failure)
	}
	require.NoError(t, err, "exported totals never agreed with the cluster")

	checkGRPCServerMetrics(t, families)
	checkBuildInfo(t, families)
}

func checkBuildInfo(t *testing.T, families map[string]*prommodel.MetricFamily) {
	t.Log("--- testing build info metric ---")

	family, ok := families["keda_build_info"]
	assert.True(t, ok, "keda_build_info not available")
	if !ok {
		t.Errorf("metric keda_build_info not available")
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

// checkTriggerTotalValues takes an assert.TestingT rather than a *testing.T so that a caller can
// retry it and only report the failures of the final attempt.
func checkTriggerTotalValues(t assert.TestingT, families map[string]*prommodel.MetricFamily, expectedValues map[string]int) {
	expected := map[string]int{}
	family, ok := families["keda_trigger_registered_total"]
	assert.True(t, ok, "keda_trigger_registered_total not available")
	if !ok {
		return
	}
	maps.Copy(expected, expectedValues)
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

	assert.Empty(t, expected, "trigger types missing from keda_trigger_registered_total")
}

func checkCRTotalValues(t assert.TestingT, families map[string]*prommodel.MetricFamily, expected map[string]map[string]int) {
	family, ok := families["keda_resource_registered_total"]
	assert.True(t, ok, "keda_resource_registered_total not available")
	if !ok {
		return
	}

	metrics := family.GetMetric()
	for _, metric := range metrics {
		labels := metric.GetLabel()
		var namespace, crType string
		for _, label := range labels {
			switch *label.Name {
			case labelType:
				crType = *label.Value
			case namespaceString:
				namespace = *label.Value
			}
		}

		metricValue := *metric.Gauge.Value
		expectedMetricValue := float64(expected[crType][namespace])

		assert.Equalf(t, expectedMetricValue, metricValue, "expected %f got %f for cr type %s & namespace %s",
			expectedMetricValue, metricValue, crType, namespace)
	}
}

func checkGRPCServerMetrics(t *testing.T, families map[string]*prommodel.MetricFamily) {
	t.Log("--- testing grpc server metrics ---")

	family, ok := families["keda_internal_metricsservice_grpc_server_handled_total"]
	if !ok {
		t.Errorf("metric keda_internal_metricsservice_grpc_server_handled_total not available")
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
	assert.GreaterOrEqual(t, metricValue, 1.0, "keda_internal_metricsservice_grpc_server_handled_total has to be greater than 0")

	family, ok = families["keda_internal_metricsservice_grpc_server_started_total"]
	if !ok {
		t.Errorf("metric keda_internal_metricsservice_grpc_server_started_total not available")
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
	assert.GreaterOrEqual(t, metricValue, 1.0, "keda_internal_metricsservice_grpc_server_started_total has to be greater than 0")

	family, ok = families["keda_internal_metricsservice_grpc_server_msg_received_total"]
	if !ok {
		t.Errorf("metric keda_internal_metricsservice_grpc_server_msg_received_total not available")
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
	assert.GreaterOrEqual(t, metricValue, 1.0, "keda_internal_metricsservice_grpc_server_msg_received_total has to be greater than 0")

	family, ok = families["keda_internal_metricsservice_grpc_server_msg_sent_total"]
	if !ok {
		t.Errorf("metric keda_internal_metricsservice_grpc_server_msg_sent_total not available")
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
	assert.GreaterOrEqual(t, metricValue, 1.0, "keda_internal_metricsservice_grpc_server_msg_sent_total has to be greater than 0")

	family, ok = families["keda_internal_metricsservice_grpc_server_handling_seconds"]
	if !ok {
		t.Errorf("metric keda_internal_metricsservice_grpc_server_handling_seconds not available")
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
		metricValue += *metric.Histogram.SampleSum
	}
	assert.Greater(t, metricValue, 0.0, "keda_internal_metricsservice_grpc_server_handling_seconds has to be greater than 0")
}

func checkGRPCClientMetrics(t *testing.T, families map[string]*prommodel.MetricFamily) {
	t.Log("--- testing grpc client metrics ---")

	family, ok := families["keda_internal_metricsservice_grpc_client_handled_total"]
	if !ok {
		t.Errorf("metric keda_internal_metricsservice_grpc_client_handled_total not available")
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
	assert.GreaterOrEqual(t, metricValue, 1.0, "keda_internal_metricsservice_grpc_client_handled_total has to be greater than 0")

	family, ok = families["keda_internal_metricsservice_grpc_client_started_total"]
	if !ok {
		t.Errorf("metric keda_internal_metricsservice_grpc_client_started_total not available")
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
	assert.GreaterOrEqual(t, metricValue, 1.0, "keda_internal_metricsservice_grpc_client_started_total has to be greater than 0")

	family, ok = families["keda_internal_metricsservice_grpc_client_msg_received_total"]
	if !ok {
		t.Errorf("metric keda_internal_metricsservice_grpc_client_msg_received_total not available")
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
	assert.GreaterOrEqual(t, metricValue, 1.0, "keda_internal_metricsservice_grpc_client_msg_received_total has to be greater than 0")

	family, ok = families["keda_internal_metricsservice_grpc_client_msg_sent_total"]
	if !ok {
		t.Errorf("metric keda_internal_metricsservice_grpc_client_msg_sent_total not available")
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
	assert.GreaterOrEqual(t, metricValue, 1.0, "keda_internal_metricsservice_grpc_client_msg_sent_total has to be greater than 0")
}

func checkWebhookValues(t *testing.T, families map[string]*prommodel.MetricFamily) {
	t.Log("--- testing webhook metrics ---")

	family, ok := families["keda_webhook_scaled_object_validation_errors_total"]
	if !ok {
		t.Errorf("metric keda_webhook_scaled_object_validation_errors_total not available")
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
	assert.GreaterOrEqual(t, metricValue, 1.0, "keda_webhook_scaled_object_validation_errors_total has to be greater than 0")

	family, ok = families["keda_webhook_scaled_object_validations_total"]
	if !ok {
		t.Errorf("metric keda_webhook_scaled_object_validations_total not available")
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
	assert.GreaterOrEqual(t, metricValue, 1.0, "keda_webhook_scaled_object_validations_total has to be greater than 0")
}

func checkMetricServerValues(t *testing.T, families map[string]*prommodel.MetricFamily) {
	t.Log("--- testing metric server metrics ---")

	family, ok := families["workqueue_adds_total"]
	if !ok {
		t.Errorf("metric workqueue_adds_total not available")
		return
	}

	metricValue := 0.0
	metrics := family.GetMetric()
	for _, metric := range metrics {
		metricValue += *metric.Counter.Value
	}
	assert.GreaterOrEqual(t, metricValue, 1.0, "workqueue_adds_total has to be greater than 0")

	family, ok = families["apiserver_request_total"]
	if !ok {
		t.Errorf("metric apiserver_request_total not available")
		return
	}

	metricValue = 0.0
	metrics = family.GetMetric()
	for _, metric := range metrics {
		labels := metric.GetLabel()
		for _, label := range labels {
			if *label.Name == "group" && *label.Value == "external.metrics.k8s.io" {
				metricValue = *metric.Counter.Value
			}
		}
	}
	assert.GreaterOrEqual(t, metricValue, 1.0, "apiserver_request_total has to be greater than 0")
}

func testCloudEventEmitted(t *testing.T, data templateData) {
	t.Log("--- testing cloudevent emitted ---")

	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	KubectlApplyWithTemplate(t, data, "cloudEventSourceTemplate", cloudEventSourceTemplate)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	defer KubectlDeleteWithTemplate(t, data, "cloudEventSourceTemplate", cloudEventSourceTemplate)

	familyValidator := func(family *prommodel.MetricFamily) bool {
		labels := family.GetMetric()
		found := false
		for _, metric := range labels {
			labels := metric.GetLabel()
			if len(labels) >= 4 &&
				ExtractPrometheusLabelValue("cloudeventsource", labels) == "prometheus-metrics-test-ce" &&
				ExtractPrometheusLabelValue("eventsink", labels) == "http" &&
				ExtractPrometheusLabelValue("namespace", labels) == "prometheus-metrics-test-ns" &&
				ExtractPrometheusLabelValue("state", labels) == "emitted" &&
				metric.GetCounter().GetValue() >= 1 {
				found = true
			}
		}
		return found
	}

	families := WaitForPrometheusMetric(t, "keda_cloudeventsource_events_emitted_total", familyValidator)
	metric := families["keda_cloudeventsource_events_emitted_total"]

	assert.True(t, familyValidator(metric))
}

func testCloudEventEmittedError(t *testing.T, data templateData) {
	t.Log("--- testing cloudevent emitted error ---")

	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	KubectlApplyWithTemplate(t, data, "wrongCloudEventSourceTemplate", wrongCloudEventSourceTemplate)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	defer KubectlDeleteWithTemplate(t, data, "wrongCloudEventSourceTemplate", wrongCloudEventSourceTemplate)

	familyValidator := func(family *prommodel.MetricFamily) bool {
		labels := family.GetMetric()
		found := false
		for _, metric := range labels {
			labels := metric.GetLabel()
			if len(labels) >= 4 &&
				ExtractPrometheusLabelValue("cloudeventsource", labels) == "prometheus-metrics-test-ce-w" &&
				ExtractPrometheusLabelValue("eventsink", labels) == "http" &&
				ExtractPrometheusLabelValue("namespace", labels) == "prometheus-metrics-test-ns" &&
				ExtractPrometheusLabelValue("state", labels) == "failed" &&
				metric.GetCounter().GetValue() >= 5 {
				found = true
			}
		}
		return found
	}

	families := WaitForPrometheusMetric(t, "keda_cloudeventsource_events_emitted_total", familyValidator)
	metric := families["keda_cloudeventsource_events_emitted_total"]

	assert.True(t, familyValidator(metric))
}

func testEmptyUpstreamResponse(t *testing.T, data templateData) {
	t.Log("--- testing empty upstream response metric ---")

	kc := GetKubernetesClient(t)

	KubectlApplyWithTemplate(t, data, "emptyUpstreamPrometheusConfigMapTemplate", emptyUpstreamPrometheusConfigMapTemplate)
	KubectlApplyWithTemplate(t, data, "emptyUpstreamPrometheusDeploymentTemplate", emptyUpstreamPrometheusDeploymentTemplate)
	KubectlApplyWithTemplate(t, data, "emptyUpstreamPrometheusServiceTemplate", emptyUpstreamPrometheusServiceTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, "prometheus-empty-upstream", testNamespace, 1, 60, 3),
		"prometheus-empty-upstream deployment should be ready")

	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlApplyWithTemplate(t, data, "emptyUpstreamResponseScaledObjectTemplate", emptyUpstreamResponseScaledObjectTemplate)
	defer func() {
		KubectlDeleteWithTemplate(t, data, "emptyUpstreamResponseScaledObjectTemplate", emptyUpstreamResponseScaledObjectTemplate)
		KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
		KubectlDeleteWithTemplate(t, data, "emptyUpstreamPrometheusServiceTemplate", emptyUpstreamPrometheusServiceTemplate)
		KubectlDeleteWithTemplate(t, data, "emptyUpstreamPrometheusDeploymentTemplate", emptyUpstreamPrometheusDeploymentTemplate)
		KubectlDeleteWithTemplate(t, data, "emptyUpstreamPrometheusConfigMapTemplate", emptyUpstreamPrometheusConfigMapTemplate)
	}()

	familyValidator := func(family *prommodel.MetricFamily) bool {
		for _, metric := range family.GetMetric() {
			labels := metric.GetLabel()
			if ExtractPrometheusLabelValue("namespace", labels) == testNamespace &&
				ExtractPrometheusLabelValue("scaledResource", labels) == emptyUpstreamScaledObjectName &&
				ExtractPrometheusLabelValue("triggerName", labels) == "empty-upstream-trigger" &&
				ExtractPrometheusLabelValue("metricName", labels) == "s0-prometheus" &&
				ExtractPrometheusLabelValue("resourceType", labels) == "ScaledObject" &&
				ExtractPrometheusLabelValue("ignoreNullValues", labels) == "false" &&
				metric.GetCounter().GetValue() >= 1 {
				return true
			}
		}
		return false
	}

	families := WaitForPrometheusMetric(t, "keda_scaler_empty_upstream_responses_total", familyValidator)
	metric := families["keda_scaler_empty_upstream_responses_total"]

	assert.True(t, familyValidator(metric))
}

func testHTTPClientMetrics(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing HTTP client metrics ---")
	SetDeploymentContainerArg(t, kc, KEDAOperator, KEDANamespace, KEDAOperator, "--enable-high-cardinality-metrics-labels", "true")
	defer SetDeploymentContainerArg(t, kc, KEDAOperator, KEDANamespace, KEDAOperator, "--enable-high-cardinality-metrics-labels", "false")

	// The dedicated HTTP client metrics ScaledObject uses a prometheus-type
	// scaler that makes real HTTP requests on every poll interval, so its
	// records should be present once at least one poll cycle has completed.
	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlApplyWithTemplate(t, data, "httpClientScaledObjectTemplate", httpClientScaledObjectTemplate)
	defer func() {
		KubectlDeleteWithTemplate(t, data, "httpClientScaledObjectTemplate", httpClientScaledObjectTemplate)
		KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	}()

	matchLabels := func(labels []*prommodel.LabelPair) bool {
		return ExtractPrometheusLabelValue("namespace", labels) == testNamespace &&
			ExtractPrometheusLabelValue("scaled_resource", labels) == data.HTTPClientScaledObjectName &&
			ExtractPrometheusLabelValue("scaler", labels) == "prometheus" &&
			ExtractPrometheusLabelValue("trigger_name", labels) == data.HTTPClientScalerName &&
			ExtractPrometheusLabelValue("metric_name", labels) == "s0-prometheus"
	}

	familyValidator := func(family *prommodel.MetricFamily) bool {
		for _, metric := range family.GetMetric() {
			if matchLabels(metric.GetLabel()) && metric.GetCounter().GetValue() >= 1 {
				return true
			}
		}
		return false
	}

	families := WaitForPrometheusMetric(t, "keda_scaler_http_requests_total", familyValidator)
	assert.True(t, familyValidator(families["keda_scaler_http_requests_total"]),
		"expected keda_scaler_http_requests_total with namespace=%s, scaled_resource=%s, scaler=prometheus, trigger_name=%s, metric_name=s0-prometheus",
		testNamespace, data.HTTPClientScaledObjectName, data.HTTPClientScalerName)

	matchHistogramLabels := func(labels []*prommodel.LabelPair) bool {
		return matchLabels(labels)
	}
	family, ok := families["keda_scaler_http_request_duration_seconds"]
	assert.True(t, ok, "keda_scaler_http_request_duration_seconds not present")
	if ok {
		var found bool
		for _, metric := range family.GetMetric() {
			if matchHistogramLabels(metric.GetLabel()) {
				assert.Greater(t, metric.GetHistogram().GetSampleCount(), uint64(0),
					"keda_scaler_http_request_duration_seconds sample count should be > 0")
				found = true
				break
			}
		}
		assert.True(t, found, "expected keda_scaler_http_request_duration_seconds histogram for prometheus scaler")
	}
}

func testHighCardinalityLabelsDisabled(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing high-cardinality labels disabled ---")

	SetDeploymentContainerArg(t, kc, KEDAOperator, KEDANamespace, KEDAOperator, "--enable-high-cardinality-metrics-labels", "false")

	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlApplyWithTemplate(t, data, "httpClientScaledObjectTemplate", httpClientScaledObjectTemplate)
	defer func() {
		KubectlDeleteWithTemplate(t, data, "httpClientScaledObjectTemplate", httpClientScaledObjectTemplate)
		KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	}()

	matchLabels := func(labels []*prommodel.LabelPair) bool {
		return ExtractPrometheusLabelValue("namespace", labels) == data.TestNamespace &&
			ExtractPrometheusLabelValue("scaled_resource", labels) == data.HTTPClientScaledObjectName &&
			ExtractPrometheusLabelValue("scaler", labels) == "prometheus" &&
			ExtractPrometheusLabelValue("trigger_name", labels) == data.HTTPClientScalerName &&
			ExtractPrometheusLabelValue("metric_name", labels) == "s0-prometheus"
	}

	families := WaitForPrometheusMetricAtURL(t, kedaOperatorPrometheusURL, "keda_scaler_http_requests_total", func(family *prommodel.MetricFamily) bool {
		for _, metric := range family.GetMetric() {
			if matchLabels(metric.GetLabel()) && metric.GetCounter().GetValue() >= 1 {
				return true
			}
		}
		return false
	})
	family, ok := families["keda_scaler_http_request_duration_seconds"]
	assert.True(t, ok, "keda_scaler_http_request_duration_seconds should be emitted when high-cardinality labels are disabled")
	if ok {
		var found bool
		for _, metric := range family.GetMetric() {
			labels := metric.GetLabel()
			if ExtractPrometheusLabelValue("scaler", labels) == "prometheus" &&
				ExtractPrometheusLabelValue("namespace", labels) == "" &&
				ExtractPrometheusLabelValue("scaled_resource", labels) == "" &&
				ExtractPrometheusLabelValue("trigger_name", labels) == "" &&
				ExtractPrometheusLabelValue("metric_name", labels) == "" {
				assert.Greater(t, metric.GetHistogram().GetSampleCount(), uint64(0),
					"keda_scaler_http_request_duration_seconds sample count should be > 0")
				found = true
				break
			}
		}
		assert.True(t, found, "expected keda_scaler_http_request_duration_seconds histogram without high-cardinality labels")
	}

	families = WaitForPrometheusMetricAtURL(t, kedaOperatorPrometheusURL, "keda_internal_metricsservice_grpc_server_handled_total", metricFamilyCounterSumGreaterThanZero)
	_, ok = families["keda_internal_metricsservice_grpc_server_handling_seconds"]
	assert.True(t, ok, "keda_internal_metricsservice_grpc_server_handling_seconds should still be emitted when high-cardinality labels are disabled")

	families = WaitForPrometheusMetricAtURL(t, kedaMetricsServerPrometheusURL, "keda_internal_metricsservice_grpc_client_handled_total", metricFamilyCounterSumGreaterThanZero)
	_, ok = families["keda_internal_metricsservice_grpc_client_handling_seconds"]
	assert.True(t, ok, "keda_internal_metricsservice_grpc_client_handling_seconds should still be emitted when high-cardinality labels are disabled")
}
