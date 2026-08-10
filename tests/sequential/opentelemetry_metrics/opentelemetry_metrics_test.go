//go:build e2e
// +build e2e

package opentelemetry_metrics_test

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	prommodel "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kedacore/keda/v2/pkg/metricscollector"
	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName              = "opentelemetry-metrics-test"
	labelScaledObject     = "scaledObject"
	labelScaledJob        = "scaledJob"
	labelType             = "type"
	labelCloudEventSource = "cloudEventSource"
	eventsink             = "eventsink"
	eventsinkValue        = "opentelemetry-metrics-test-ce"
	eventsinkType         = "eventsinktype"
	eventsinkTypeValue    = "http"

	// The operator pushes metrics to the collector on an interval rather than on change, so a
	// value only becomes observable some time after the cluster state changed. The e2e setup
	// exports every 3s (OTEL_METRIC_EXPORT_INTERVAL) and the scaled objects here poll every
	// 2-5s, so a minute is more than ten of those cycles and is only reached when something is
	// genuinely broken. Do not raise it much further: the collector's prometheus exporter
	// drops a series 5 minutes after its last push, so beyond that a value stops being
	// observable at all and waiting longer cannot help.
	metricWaitTimeout = time.Minute

	// Each poll of the operator totals lists every ScaledObject, ScaledJob and
	// TriggerAuthentication in the cluster, so it runs less aggressively than a plain
	// metric read.
	operatorMetricsInterval = 5 * time.Second
)

var (
	testNamespace                            = fmt.Sprintf("%s-ns", testName)
	deploymentName                           = fmt.Sprintf("%s-deployment", testName)
	monitoredDeploymentName                  = fmt.Sprintf("%s-monitored", testName)
	scaledObjectName                         = fmt.Sprintf("%s-so", testName)
	resourceMetricDeploymentName             = fmt.Sprintf("%s-resource-deployment", testName)
	resourceMetricScaledObjectName           = fmt.Sprintf("%s-resource-so", testName)
	resourceMetricScalerName                 = fmt.Sprintf("%s-resource-cpu-scaler", testName)
	httpClientScaledObjectName               = fmt.Sprintf("%s-so-http-client", testName)
	wrongScaledObjectName                    = fmt.Sprintf("%s-so-wrong", testName)
	scaledObjectGrpcName                     = fmt.Sprintf("%s-so-grpc", testName)
	scaledJobName                            = fmt.Sprintf("%s-sj", testName)
	wrongScaledJobName                       = fmt.Sprintf("%s-sj-wrong", testName)
	wrongScalerName                          = fmt.Sprintf("%s-wrong-scaler", testName)
	emptyUpstreamScaledObjectName            = fmt.Sprintf("%s-so-empty-upstream", testName)
	httpClientScalerName                     = fmt.Sprintf("%s-http-client-scaler", testName)
	cronScaledJobName                        = fmt.Sprintf("%s-cron-sj", testName)
	clientName                               = fmt.Sprintf("%s-client", testName)
	cloudEventSourceName                     = fmt.Sprintf("%s-ce", testName)
	wrongCloudEventSourceName                = fmt.Sprintf("%s-ce-w", testName)
	cloudEventHTTPReceiverName               = fmt.Sprintf("%s-cloudevent-http-receiver", testName)
	cloudEventHTTPServiceName                = fmt.Sprintf("%s-cloudevent-http-service", testName)
	cloudEventHTTPServiceURL                 = fmt.Sprintf("http://%s.%s.svc.cluster.local:8899", cloudEventHTTPServiceName, testNamespace)
	kedaOperatorCollectorPrometheusExportURL = "http://opentelemetry-collector.open-telemetry-system.svc.cluster.local:8889/metrics"
	otlpGrpcClientEndpoint                   = "http://opentelemetry-collector.open-telemetry-system.svc.cluster.local:4317"
	otlpHTTPClientEndpoint                   = "http://opentelemetry-collector.open-telemetry-system.svc.cluster.local:4318"
	namespaceString                          = "namespace"
	kedaNamespace                            = "keda"
	kedaOperatorDeploymentName               = "keda-operator"
	operatorLabelSelector                    = "app=keda-operator"
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
	ScaledObjectGrpcName           string
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

	scaledObjectGrpcTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectGrpcName}}
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

func TestOpenTelemetryMetrics(t *testing.T) {
	// setup
	t.Log("--- setting up ---")

	// If opentelemetry is not enabled, skip the test
	if EnableOpentelemetry == "" || EnableOpentelemetry == StringFalse {
		t.Skip("skipping opentelemetry test as EnableOpentelemetry is not set to true")
	}

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	// A metric that never converges fails the test from inside a helper, which ends the whole
	// function, so teardown has to be deferred rather than run at the end. Otherwise a single
	// timeout leaks this namespace and the operator's OTLP configuration into the sequential
	// tests that run after this one.
	defer DeleteKubernetesResources(t, testNamespace, data, templates)

	// restart KEDA operator to ensure that all the metrics are sent to the collector
	DeletePodsInNamespaceBySelector(t, kc, operatorLabelSelector, kedaNamespace)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, kedaOperatorDeploymentName, kedaNamespace, 1, 60, 2),
		"replica count should be 1 after 2 minute")

	// scaling to max replica count to ensure the counter is registered before we test it
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, 2),
		"replica count should be 2 after 2 minute")

	testScalerMetricValue(t)
	testScalerMetricLatency(t)
	testScalerActiveMetric(t, kc)
	// Run this before any prometheus-based scaler scenarios, otherwise the collector
	// already contains the HTTP duration histogram family from earlier requests.
	testHighCardinalityLabelsDisabled(t, kc, data)
	testScaledObjectErrors(t, data)
	testScaledJobErrors(t, data)
	testScalerErrors(t, data)
	testOperatorMetrics(t, kc, data)
	testScalableObjectMetrics(t)
	testScaledObjectPausedMetric(t, data)
	testCloudEventEmitted(t, data)
	testCloudEventEmittedError(t, data)
	testEmptyUpstreamResponse(t, data)
	testHTTPClientMetrics(t, kc, data)

	changeOtlpProtocolInOperator(t, kc, "keda-operator", "keda")
	defer fallbackHTTPProtocolInOperator(t, kc, "keda-operator", "keda")
	testScalerGrpcMetricValue(t, kc, data)
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
			ScaledObjectGrpcName:           scaledObjectGrpcName,
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

func changeOtlpProtocolInOperator(t *testing.T, kc *kubernetes.Clientset, name string, namespace string) {
	operator, _ := kc.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	// Modify the environment variables
	t.Log("changeOtlpProtocolInOperator")
	for i, container := range operator.Spec.Template.Spec.Containers {
		if container.Name == name {
			container.Env = slices.DeleteFunc(container.Env, func(n corev1.EnvVar) bool {
				return n.Name == "OTEL_EXPORTER_OTLP_ENDPOINT"
			})

			container.Env = append(container.Env, corev1.EnvVar{Name: "OTEL_EXPORTER_OTLP_PROTOCOL", Value: "grpc"})
			container.Env = append(container.Env, corev1.EnvVar{Name: "OTEL_EXPORTER_OTLP_ENDPOINT", Value: otlpGrpcClientEndpoint})
			operator.Spec.Template.Spec.Containers[i].Env = container.Env
		}
	}

	_, err := kc.AppsV1().Deployments(namespace).Update(context.TODO(), operator, metav1.UpdateOptions{})

	require.NoErrorf(t, err, "error change keda operator - %s", err)
	WaitForDeploymentReplicaReadyCount(t, kc, operator.Name, "keda", 1, 60, 2)
}

func fallbackHTTPProtocolInOperator(t *testing.T, kc *kubernetes.Clientset, name string, namespace string) {
	t.Log("fallbacek HTTP OTLP protocol in operator")

	operator, _ := kc.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	// Modify the environment variables
	for i, container := range operator.Spec.Template.Spec.Containers {
		if container.Name == name {
			container.Env = slices.DeleteFunc(container.Env, func(n corev1.EnvVar) bool {
				if n.Name == "OTEL_EXPORTER_OTLP_ENDPOINT" || n.Name == "OTEL_EXPORTER_OTLP_PROTOCOL" {
					return true
				}
				return false
			})
			container.Env = append(container.Env, corev1.EnvVar{Name: "OTEL_EXPORTER_OTLP_ENDPOINT", Value: otlpHTTPClientEndpoint})
			operator.Spec.Template.Spec.Containers[i].Env = container.Env
		}
	}

	_, err := kc.AppsV1().Deployments(namespace).Update(context.TODO(), operator, metav1.UpdateOptions{})

	require.NoErrorf(t, err, "error change keda operator - %s", err)
	WaitForDeploymentReplicaReadyCount(t, kc, operator.Name, "keda", 1, 60, 2)
}

func testScalerGrpcMetricValue(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scaler grpc metric value ---")
	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlApplyWithTemplate(t, data, "scaledObjectGrpcTemplate", scaledObjectGrpcTemplate)
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 0, testNamespace)
	WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 2)

	// The deployment reaching zero does not mean the metric has caught up: the value only
	// changes once the scaler re-polls and the operator's next push reaches the collector.
	families := waitForCollectorMetric(t, "keda_scaler_metrics_value", func(family *prommodel.MetricFamily) bool {
		value, found := scaledObjectGaugeValue(family, scaledObjectGrpcName)
		return found && value == 0
	})

	value, found := scaledObjectGaugeValue(families["keda_scaler_metrics_value"], scaledObjectGrpcName)
	assert.Truef(t, found, "no keda_scaler_metrics_value reported for %s", scaledObjectGrpcName)
	assert.Equal(t, float64(0), value)

	KubectlDeleteWithTemplate(t, data, "scaledObjectGrpcTemplate", scaledObjectGrpcTemplate)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
}

// scaledObjectGaugeValue returns the gauge value labelled with the given ScaledObject name,
// and whether such a metric was present at all.
func scaledObjectGaugeValue(family *prommodel.MetricFamily, soName string) (float64, bool) {
	for _, metric := range family.GetMetric() {
		for _, label := range metric.GetLabel() {
			if *label.Name == labelScaledObject && *label.Value == soName {
				return metric.GetGauge().GetValue(), true
			}
		}
	}
	return 0, false
}

func fetchAndParsePrometheusMetrics(t *testing.T, cmd string) map[string]*prommodel.MetricFamily {
	out, _, err := ExecCommandOnSpecificPod(t, clientName, testNamespace, cmd)
	assert.NoErrorf(t, err, "cannot execute command - %s", err)

	parser := expfmt.NewTextParser(model.UTF8Validation)
	// Ensure EOL
	reader := strings.NewReader(strings.ReplaceAll(out, "\r\n", "\n"))
	families, err := parser.TextToMetricFamilies(reader)
	assert.NoErrorf(t, err, "cannot parse metrics - %s", err)

	return families
}

// waitForCollectorMetric polls the collector's Prometheus export endpoint until the named
// metric exists and familyValidator accepts it, then returns the parsed families.
//
// Waiting for the value itself is what keeps these tests independent of how fast the cluster
// is. The alternative is sleeping for long enough to cover the slowest plausible export
// cycle, which is both slower in the common case and still not long enough under load.
//
// A metric that never arrives ends the test on the spot, so anything that has to be undone
// afterwards belongs in a defer.
func waitForCollectorMetric(t *testing.T, metricToWaitFor string, familyValidator func(family *prommodel.MetricFamily) bool) map[string]*prommodel.MetricFamily {
	contextWithTimeout, cancel := context.WithTimeout(context.Background(), metricWaitTimeout)
	defer cancel()

	var families map[string]*prommodel.MetricFamily
	err := KedaEventually(contextWithTimeout, func(ctx context.Context) (bool, error) {
		t.Logf("Waiting for metric %s", metricToWaitFor)
		families = fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorCollectorPrometheusExportURL))

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
				if (*label.Name == labelScaledObject && *label.Value == scaledObjectName) ||
					(*label.Name == labelScaledJob && *label.Value == scaledJobName) {
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
	KubectlApplyWithTemplate(t, data, "wrongScaledObjectTemplate", wrongScaledObjectTemplate)

	// The broken ScaledObject has to be reconciled and polled once before the counter is
	// non-zero, and then polled again for the comparison below to mean anything. Wait for
	// each of those steps rather than guessing how long they take.
	families := waitForCollectorMetric(t, "keda_scaledobject_errors_total", func(family *prommodel.MetricFamily) bool {
		return getErrorMetricsValue(family) > 0
	})
	errCounterVal1 := getErrorMetricsValue(families["keda_scaledobject_errors_total"])

	families = waitForCollectorMetric(t, "keda_scaledobject_errors_total", func(family *prommodel.MetricFamily) bool {
		return getErrorMetricsValue(family) > errCounterVal1
	})
	errCounterVal2 := getErrorMetricsValue(families["keda_scaledobject_errors_total"])

	assert.NotEqual(t, errCounterVal2, float64(0))
	assert.GreaterOrEqual(t, errCounterVal2, errCounterVal1)

	KubectlDeleteWithTemplate(t, data, "wrongScaledObjectTemplate", wrongScaledObjectTemplate)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
}

func testScaledJobErrors(t *testing.T, data templateData) {
	t.Log("--- testing scaled job errors ---")

	KubectlDeleteWithTemplate(t, data, "scaledJobTemplate", scaledJobTemplate)
	KubectlApplyWithTemplate(t, data, "wrongScaledJobTemplate", wrongScaledJobTemplate)

	families := waitForCollectorMetric(t, "keda_scaledjob_errors_total", func(family *prommodel.MetricFamily) bool {
		return getErrorMetricsValue(family) > 0
	})
	errCounterVal1 := getErrorMetricsValue(families["keda_scaledjob_errors_total"])

	families = waitForCollectorMetric(t, "keda_scaledjob_errors_total", func(family *prommodel.MetricFamily) bool {
		return getErrorMetricsValue(family) > errCounterVal1
	})
	errCounterVal2 := getErrorMetricsValue(families["keda_scaledjob_errors_total"])

	assert.NotEqual(t, errCounterVal2, float64(0))
	assert.GreaterOrEqual(t, errCounterVal2, errCounterVal1)

	KubectlDeleteWithTemplate(t, data, "wrongScaledJobTemplate", wrongScaledJobTemplate)
	KubectlApplyWithTemplate(t, data, "scaledJobTemplate", scaledJobTemplate)
}

func testScalerErrors(t *testing.T, data templateData) {
	t.Log("--- testing scaler errors ---")

	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlApplyWithTemplate(t, data, "wrongScaledObjectTemplate", wrongScaledObjectTemplate)

	KubectlDeleteWithTemplate(t, data, "scaledJobTemplate", scaledJobTemplate)
	KubectlApplyWithTemplate(t, data, "wrongScaledJobTemplate", wrongScaledJobTemplate)

	families := waitForCollectorMetric(t, "keda_scaler_errors_total", func(family *prommodel.MetricFamily) bool {
		return getErrorMetricsValue(family) > 0
	})
	errCounterVal1 := getErrorMetricsValue(families["keda_scaler_errors_total"])

	// Waiting for the counter to be strictly greater is what makes this check meaningful:
	// waiting for >= would be satisfied by the value we already read.
	families = waitForCollectorMetric(t, "keda_scaler_errors_total", func(family *prommodel.MetricFamily) bool {
		return getErrorMetricsValue(family) > errCounterVal1
	})
	errCounterVal2 := getErrorMetricsValue(families["keda_scaler_errors_total"])

	assert.NotEqual(t, errCounterVal2, float64(0))
	assert.GreaterOrEqual(t, errCounterVal2, errCounterVal1)

	KubectlDeleteWithTemplate(t, data, "wrongScaledJobTemplate", wrongScaledJobTemplate)
	KubectlApplyWithTemplate(t, data, "scaledJobTemplate", scaledJobTemplate)

	KubectlDeleteWithTemplate(t, data, "wrongScaledObjectTemplate", wrongScaledObjectTemplate)
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
	case "keda_scaledjob_errors_total":
		metrics := val.GetMetric()
		for _, metric := range metrics {
			labels := metric.GetLabel()
			for _, label := range labels {
				if *label.Name == "scaledJob" && *label.Value == wrongScaledJobName {
					return *metric.Counter.Value
				}
			}
		}
	case "keda_scaled_job_errors":
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

func testScalerMetricLatency(t *testing.T) {
	t.Log("--- testing scaler metric latency ---")

	family := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorCollectorPrometheusExportURL))

	val, ok := family["keda_scaler_metrics_latency"]
	assert.True(t, ok, "keda_scaler_metrics_latency not available")
	if ok {
		var found bool
		metrics := val.GetMetric()
		for _, metric := range metrics {
			t.Log("--- latency metric detail info ---", "metric", metric)
			labels := metric.GetLabel()
			for _, label := range labels {
				if (*label.Name == labelScaledObject && *label.Value == scaledObjectName) ||
					(*label.Name == labelScaledJob && *label.Value == scaledJobName) {
					assert.Equal(t, float64(0), *metric.Gauge.Value)
					found = true
				}
			}
		}
		assert.Equal(t, true, found)
	}
	val, ok = family["keda_scaler_metrics_latency_seconds"]
	assert.True(t, ok, "keda_scaler_metrics_latency_seconds not available")
	if ok {
		var found bool
		metrics := val.GetMetric()
		for _, metric := range metrics {
			t.Log("--- latency metric detail info ---", "metric", metric)
			labels := metric.GetLabel()
			for _, label := range labels {
				if (*label.Name == labelScaledObject && *label.Value == scaledObjectName) ||
					(*label.Name == labelScaledJob && *label.Value == scaledJobName) {
					assert.InDelta(t, float64(0), *metric.Gauge.Value, 0.001)
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

	val, ok = family["keda_internal_scale_loop_latency_seconds"]
	assert.True(t, ok, "keda_internal_scale_loop_latency_seconds not available")
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

func testScalerActiveMetric(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scaler active metric ---")

	resourceScalerLabels := map[string]string{
		"namespace":    testNamespace,
		"scaledObject": resourceMetricScaledObjectName,
		"scaler":       resourceMetricScalerName,
		"scalerIndex":  "0",
		"metric":       "cpu",
	}

	families := waitForCollectorMetric(t, "keda_scaler_active", func(family *prommodel.MetricFamily) bool {
		return hasMetricWithLabelsAndGauge(family, resourceScalerLabels, 1)
	})
	assertScaledObjectFlagMetric(t, families, scaledObjectName, "keda_scaler_active", true)
	assert.True(t, hasMetricWithLabelsAndGauge(families["keda_scaler_active"], resourceScalerLabels, 1),
		"expected keda_scaler_active for CPU resource scaler")

	t.Log("--- testing scaler active metric scaled down ---")
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 0, testNamespace)
	WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 2)
	families = waitForCollectorMetric(t, "keda_scaler_active", func(family *prommodel.MetricFamily) bool {
		value, found := scaledObjectGaugeValue(family, scaledObjectName)
		return found && value == 0
	})

	assertScaledObjectFlagMetric(t, families, scaledObjectName, "keda_scaler_active", false)
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 4, testNamespace)
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
	families := waitForCollectorMetric(t, "keda_scaled_object_paused", func(family *prommodel.MetricFamily) bool {
		value, found := scaledObjectGaugeValue(family, scaledObjectName)
		return found && value == 1
	})
	assertScaledObjectFlagMetric(t, families, scaledObjectName, "keda_scaled_object_paused", true)

	// Unpause the ScaledObject
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	// Check that the paused metric is back to false
	families = waitForCollectorMetric(t, "keda_scaled_object_paused", func(family *prommodel.MetricFamily) bool {
		value, found := scaledObjectGaugeValue(family, scaledObjectName)
		return found && value == 0
	})
	assertScaledObjectFlagMetric(t, families, scaledObjectName, "keda_scaled_object_paused", false)
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

	namespaceList, err := kc.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	assert.NoErrorf(t, err, "failed to list namespaces - %s", err)

	clusterTriggerAuthenticationList, err := kedaKc.ClusterTriggerAuthentications().List(context.Background(), metav1.ListOptions{})
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

		scaledObjectList, err := kedaKc.ScaledObjects(namespace.Name).List(context.Background(), metav1.ListOptions{})
		assert.NoErrorf(t, err, "failed to list scaledObjects in namespace - %s with err - %s", namespace.Name, err)

		crTotals[metricscollector.ScaledObjectResource][namespaceName] = len(scaledObjectList.Items)
		for _, scaledObject := range scaledObjectList.Items {
			for _, trigger := range scaledObject.Spec.Triggers {
				triggerTotals[trigger.Type]++
			}
		}

		scaledJobList, err := kedaKc.ScaledJobs(namespace.Name).List(context.Background(), metav1.ListOptions{})
		assert.NoErrorf(t, err, "failed to list scaledJobs in namespace - %s with err - %s", namespace.Name, err)

		crTotals[metricscollector.ScaledJobResource][namespaceName] = len(scaledJobList.Items)
		for _, scaledJob := range scaledJobList.Items {
			for _, trigger := range scaledJob.Spec.Triggers {
				triggerTotals[trigger.Type]++
			}
		}

		triggerAuthList, err := kedaKc.TriggerAuthentications(namespace.Name).List(context.Background(), metav1.ListOptions{})
		assert.NoErrorf(t, err, "failed to list triggerAuthentications in namespace - %s with err - %s", namespace.Name, err)

		crTotals[metricscollector.TriggerAuthenticationResource][namespaceName] = len(triggerAuthList.Items)
	}

	return triggerTotals, crTotals
}

// failureCollector satisfies assert.TestingT by recording failures instead of failing the
// test, so that assertions can be retried. testify's EventuallyWithT does the same thing, but
// it runs its condition in a separate goroutine, which does not mix with the helpers here
// taking *testing.T.
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
	// the operator has to reconcile it and then push the new value. Retrying the assertions
	// themselves, rather than a predicate that has to be kept in step with them, means a
	// timeout still reports which total diverged.
	err := KedaEventually(ctx, func(_ context.Context) (bool, error) {
		families = fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorCollectorPrometheusExportURL))
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

// checkTriggerTotalValues takes assert.TestingT rather than *testing.T so that the caller can
// retry it against a failureCollector until the totals converge.
func checkTriggerTotalValues(t assert.TestingT, families map[string]*prommodel.MetricFamily, expectedValues map[string]int) {
	expected := map[string]int{}

	family, ok := families["keda_trigger_totals"]
	assert.True(t, ok, "keda_trigger_totals not available")
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

	assert.Equal(t, 0, len(expected))

	family, ok = families["keda_trigger_registered_count"]
	assert.True(t, ok, "keda_trigger_registered_count not available")
	if !ok {
		return
	}
	maps.Copy(expected, expectedValues)
	metrics = family.GetMetric()
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

func checkCRTotalValues(t assert.TestingT, families map[string]*prommodel.MetricFamily, expected map[string]map[string]int) {
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

	family, ok = families["keda_resource_registered_count"]
	assert.True(t, ok, "keda_resource_registered_count not available")
	if !ok {
		return
	}

	metrics = family.GetMetric()
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

func assertScaledObjectFlagMetric(t *testing.T, families map[string]*prommodel.MetricFamily, scaledObjectName string, metricName string, expected bool) {
	family, ok := families[metricName]
	assert.True(t, ok, "%s not available", metricName)
	if !ok {
		return
	}

	// Read the series the waits above match on, so that a wait which observed the expected
	// value cannot be followed by an assertion reading a different series of the same family.
	metricValue, found := scaledObjectGaugeValue(family, scaledObjectName)
	t.Log("scaledobject flag metric detail info ---", "metricName", metricName,
		"scaledObjectName", scaledObjectName, "value", metricValue, "found", found)

	expectedMetricValue := 0.0
	if expected {
		expectedMetricValue = 1
	}
	assert.Equal(t, expectedMetricValue, metricValue)
}

func testCloudEventEmitted(t *testing.T, data templateData) {
	t.Log("--- testing cloudevent emitted ---")

	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlApplyWithTemplate(t, data, "cloudEventSourceTemplate", cloudEventSourceTemplate)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	emitted := func(metric *prommodel.Metric) bool {
		labels := metric.GetLabel()
		return len(labels) >= 5 &&
			ExtractPrometheusLabelValue("cloudEventSource", labels) == "opentelemetry-metrics-test-ce" &&
			ExtractPrometheusLabelValue("eventsink", labels) == "http" &&
			ExtractPrometheusLabelValue("namespace", labels) == "opentelemetry-metrics-test-ns" &&
			ExtractPrometheusLabelValue("state", labels) == "emitted"
	}

	// The event is emitted asynchronously once the ScaledObject is reconciled, so wait for
	// the counter to show up instead of assuming it has by now.
	families := waitForCollectorMetric(t, "keda_cloudeventsource_events_emitted_count_total", func(family *prommodel.MetricFamily) bool {
		for _, metric := range family.GetMetric() {
			if emitted(metric) && metric.GetCounter().GetValue() >= 1 {
				return true
			}
		}
		return false
	})

	var found bool
	for _, metric := range families["keda_cloudeventsource_events_emitted_count_total"].GetMetric() {
		if emitted(metric) {
			assert.GreaterOrEqual(t, *metric.Counter.Value, float64(1))
			found = true
		}
	}
	assert.True(t, found, "no emitted cloudevent metric found")
}

func testCloudEventEmittedError(t *testing.T, data templateData) {
	t.Log("--- testing cloudevent emitted error ---")

	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	KubectlDeleteWithTemplate(t, data, "cloudEventSourceTemplate", cloudEventSourceTemplate)
	KubectlApplyWithTemplate(t, data, "wrongCloudEventSourceTemplate", wrongCloudEventSourceTemplate)
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	failed := func(metric *prommodel.Metric) bool {
		labels := metric.GetLabel()
		return len(labels) >= 5 &&
			ExtractPrometheusLabelValue("cloudEventSource", labels) == "opentelemetry-metrics-test-ce-w" &&
			ExtractPrometheusLabelValue("eventsink", labels) == "http" &&
			ExtractPrometheusLabelValue("namespace", labels) == "opentelemetry-metrics-test-ns" &&
			ExtractPrometheusLabelValue("state", labels) == "failed"
	}

	// The emitter retries before giving up, so the counter only reaches 5 after several
	// attempts have been made and pushed to the collector.
	families := waitForCollectorMetric(t, "keda_cloudeventsource_events_emitted_count_total", func(family *prommodel.MetricFamily) bool {
		for _, metric := range family.GetMetric() {
			if failed(metric) && metric.GetCounter().GetValue() >= 5 {
				return true
			}
		}
		return false
	})

	var found bool
	for _, metric := range families["keda_cloudeventsource_events_emitted_count_total"].GetMetric() {
		if failed(metric) {
			assert.GreaterOrEqual(t, *metric.Counter.Value, float64(5))
			found = true
		}
	}
	assert.True(t, found, "no failed cloudevent metric found")

	KubectlDeleteWithTemplate(t, data, "wrongCloudEventSourceTemplate", wrongCloudEventSourceTemplate)
	KubectlApplyWithTemplate(t, data, "cloudEventSourceTemplate", cloudEventSourceTemplate)
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

	matchesEmptyUpstream := func(metric *prommodel.Metric) bool {
		labels := metric.GetLabel()
		return ExtractPrometheusLabelValue("namespace", labels) == testNamespace &&
			ExtractPrometheusLabelValue("scaledResource", labels) == emptyUpstreamScaledObjectName &&
			ExtractPrometheusLabelValue("triggerName", labels) == "empty-upstream-trigger" &&
			ExtractPrometheusLabelValue("metricName", labels) == "s0-prometheus" &&
			ExtractPrometheusLabelValue("isScaledObject", labels) == "true" &&
			ExtractPrometheusLabelValue("ignoreNullValues", labels) == "false" &&
			metric.GetCounter().GetValue() >= 1
	}

	// The counter only moves once the scaler has polled the empty upstream at least once.
	families := waitForCollectorMetric(t, "keda_scaler_empty_upstream_responses_total", func(family *prommodel.MetricFamily) bool {
		return slices.ContainsFunc(family.GetMetric(), matchesEmptyUpstream)
	})

	assert.True(t, slices.ContainsFunc(families["keda_scaler_empty_upstream_responses_total"].GetMetric(), matchesEmptyUpstream),
		"keda_scaler_empty_upstream_responses_total not found with expected labels")
}

func testHTTPClientMetrics(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing HTTP client metrics ---")
	SetDeploymentContainerArg(t, kc, kedaOperatorDeploymentName, kedaNamespace, kedaOperatorDeploymentName, "--enable-high-cardinality-metrics-labels", "true")
	defer SetDeploymentContainerArg(t, kc, kedaOperatorDeploymentName, kedaNamespace, kedaOperatorDeploymentName, "--enable-high-cardinality-metrics-labels", "false")

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
		return ExtractPrometheusLabelValue("namespace", labels) == data.TestNamespace &&
			ExtractPrometheusLabelValue("scaled_resource", labels) == data.HTTPClientScaledObjectName &&
			ExtractPrometheusLabelValue("scaler", labels) == "prometheus" &&
			ExtractPrometheusLabelValue("trigger_name", labels) == data.HTTPClientScalerName &&
			ExtractPrometheusLabelValue("metric_name", labels) == "s0-prometheus"
	}

	// The requests counter carries these labels whether or not high-cardinality labels are
	// enabled, and the collector keeps a series for 5 minutes after its last push, so waiting
	// on the counter can be satisfied by the series testHighCardinalityLabelsDisabled left
	// behind, before this scenario's operator has recorded anything. Wait for the duration
	// histogram carrying the labels instead: only an operator running with the flag enabled
	// can produce that.
	families := waitForCollectorMetric(t, "keda_scaler_http_request_duration_seconds", func(f *prommodel.MetricFamily) bool {
		return slices.ContainsFunc(f.GetMetric(), func(metric *prommodel.Metric) bool {
			return matchLabels(metric.GetLabel()) && metric.GetHistogram().GetSampleCount() > 0
		})
	})

	val, ok := families["keda_scaler_http_requests_count_total"]
	assert.True(t, ok, "keda_scaler_http_requests_count_total not available")
	if ok {
		var found bool
		for _, metric := range val.GetMetric() {
			if matchLabels(metric.GetLabel()) {
				assert.GreaterOrEqual(t, metric.GetCounter().GetValue(), float64(1))
				found = true
				break
			}
		}
		assert.True(t, found,
			"expected keda_scaler_http_requests_count_total with namespace=%s, scaled_resource=%s, scaler=prometheus, trigger_name=%s, metric_name=s0-prometheus",
			data.TestNamespace, data.HTTPClientScaledObjectName, data.HTTPClientScalerName)
	}

	val, ok = families["keda_scaler_http_request_duration_seconds"]
	assert.True(t, ok, "keda_scaler_http_request_duration_seconds not available")
	if ok {
		var found bool
		for _, metric := range val.GetMetric() {
			if matchLabels(metric.GetLabel()) {
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

	SetDeploymentContainerArg(t, kc, kedaOperatorDeploymentName, kedaNamespace, kedaOperatorDeploymentName, "--enable-high-cardinality-metrics-labels", "false")

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

	families := waitForCollectorMetric(t, "keda_scaler_http_requests_count_total", func(family *prommodel.MetricFamily) bool {
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
}
