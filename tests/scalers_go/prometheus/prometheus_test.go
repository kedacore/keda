//go:build e2e
// +build e2e

package prometheus_test

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
	testName = "prometheus-test"
)

var (
	testNamespace         = fmt.Sprintf("%s-ns", testName)
	deploymentName        = fmt.Sprintf("%s-deployment", testName)
	monitoredAppName      = fmt.Sprintf("%s-monitored-app", testName)
	publishDeploymentName = fmt.Sprintf("%s-publish", testName)
	scaledObjectName      = fmt.Sprintf("%s-so", testName)
	prometheusServerName  = fmt.Sprintf("%s-server", testName)
	minReplicaCount       = 0
	maxReplicaCount       = 2
)

type templateData struct {
	TestNamespace         string
	DeploymentName        string
	MonitoredAppName      string
	PublishDeploymentName string
	ScaledObjectName      string
	PrometheusServerName  string
	MinReplicaCount       int
	MaxReplicaCount       int
}

type templateValues map[string]string

const (
	deploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: test-app
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: test-app
  template:
    metadata:
      labels:
        app: test-app
        type: keda-testing
    spec:
      containers:
      - name: prom-test-app
        image: quay.io/zroubalik/prometheus-app:latest
        imagePullPolicy: IfNotPresent
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          capabilities:
            drop:
              - ALL
          seccompProfile:
            type: RuntimeDefault
---
`

	monitoredAppDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: {{.MonitoredAppName}}
  name: {{.MonitoredAppName}}
  namespace: {{.TestNamespace}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.MonitoredAppName}}
  template:
    metadata:
      labels:
        app: {{.MonitoredAppName}}
        type: {{.MonitoredAppName}}
    spec:
      containers:
      - name: prom-test-app
        image: quay.io/zroubalik/prometheus-app:latest
        imagePullPolicy: IfNotPresent
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          capabilities:
            drop:
              - ALL
          seccompProfile:
            type: RuntimeDefault
---
`

	monitoredAppServiceTemplate = `apiVersion: v1
kind: Service
metadata:
  labels:
    app: {{.MonitoredAppName}}
  name: {{.MonitoredAppName}}
  namespace: {{.TestNamespace}}
  annotations:
    prometheus.io/scrape: "true"
spec:
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: 8080
  selector:
    type: {{.MonitoredAppName}}
`

	prometheusServerConfigMapTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  labels:
    component: "server"
    app: prometheus
    release: prometheus
  name: {{.PrometheusServerName}}
  namespace: {{.TestNamespace}}
data:
  alerting_rules.yml: |
    {}

  alerts: |
    {}

  prometheus.yml: |
    global:
      evaluation_interval: 1m
      scrape_interval: 1m
      scrape_timeout: 10s
    rule_files:
    - /etc/config/recording_rules.yml
    - /etc/config/alerting_rules.yml
    - /etc/config/rules
    - /etc/config/alerts
    scrape_configs:
    - job_name: prometheus
      static_configs:
      - targets:
        - localhost:9090
    - bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
      job_name: kubernetes-apiservers
      kubernetes_sd_configs:
      - role: endpoints
      relabel_configs:
      - action: keep
        regex: default;kubernetes;https
        source_labels:
        - __meta_kubernetes_namespace
        - __meta_kubernetes_service_name
        - __meta_kubernetes_endpoint_port_name
      scheme: https
      tls_config:
        ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        insecure_skip_verify: true
    - bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
      job_name: kubernetes-nodes
      kubernetes_sd_configs:
      - role: node
      relabel_configs:
      - action: labelmap
        regex: __meta_kubernetes_node_label_(.+)
      - replacement: kubernetes.default.svc:443
        target_label: __address__
      - regex: (.+)
        replacement: /api/v1/nodes/$1/proxy/metrics
        source_labels:
        - __meta_kubernetes_node_name
        target_label: __metrics_path__
      scheme: https
      tls_config:
        ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        insecure_skip_verify: true
    - bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
      job_name: kubernetes-nodes-cadvisor
      kubernetes_sd_configs:
      - role: node
      relabel_configs:
      - action: labelmap
        regex: __meta_kubernetes_node_label_(.+)
      - replacement: kubernetes.default.svc:443
        target_label: __address__
      - regex: (.+)
        replacement: /api/v1/nodes/$1/proxy/metrics/cadvisor
        source_labels:
        - __meta_kubernetes_node_name
        target_label: __metrics_path__
      scheme: https
      tls_config:
        ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        insecure_skip_verify: true
    - job_name: kubernetes-service-endpoints
      kubernetes_sd_configs:
      - role: endpoints
      relabel_configs:
      - action: keep
        regex: true
        source_labels:
        - __meta_kubernetes_service_annotation_prometheus_io_scrape
      - action: replace
        regex: (https?)
        source_labels:
        - __meta_kubernetes_service_annotation_prometheus_io_scheme
        target_label: __scheme__
      - action: replace
        regex: (.+)
        source_labels:
        - __meta_kubernetes_service_annotation_prometheus_io_path
        target_label: __metrics_path__
      - action: replace
        regex: ([^:]+)(?::\d+)?;(\d+)
        replacement: $1:$2
        source_labels:
        - __address__
        - __meta_kubernetes_service_annotation_prometheus_io_port
        target_label: __address__
      - action: labelmap
        regex: __meta_kubernetes_service_label_(.+)
      - action: replace
        source_labels:
        - __meta_kubernetes_namespace
        target_label: kubernetes_namespace
      - action: replace
        source_labels:
        - __meta_kubernetes_service_name
        target_label: kubernetes_name
      - action: replace
        source_labels:
        - __meta_kubernetes_pod_node_name
        target_label: kubernetes_node
    - job_name: kubernetes-service-endpoints-slow
      kubernetes_sd_configs:
      - role: endpoints
      relabel_configs:
      - action: keep
        regex: true
        source_labels:
        - __meta_kubernetes_service_annotation_prometheus_io_scrape_slow
      - action: replace
        regex: (https?)
        source_labels:
        - __meta_kubernetes_service_annotation_prometheus_io_scheme
        target_label: __scheme__
      - action: replace
        regex: (.+)
        source_labels:
        - __meta_kubernetes_service_annotation_prometheus_io_path
        target_label: __metrics_path__
      - action: replace
        regex: ([^:]+)(?::\d+)?;(\d+)
        replacement: $1:$2
        source_labels:
        - __address__
        - __meta_kubernetes_service_annotation_prometheus_io_port
        target_label: __address__
      - action: labelmap
        regex: __meta_kubernetes_service_label_(.+)
      - action: replace
        source_labels:
        - __meta_kubernetes_namespace
        target_label: kubernetes_namespace
      - action: replace
        source_labels:
        - __meta_kubernetes_service_name
        target_label: kubernetes_name
      - action: replace
        source_labels:
        - __meta_kubernetes_pod_node_name
        target_label: kubernetes_node
      scrape_interval: 5m
      scrape_timeout: 30s
    - honor_labels: true
      job_name: prometheus-pushgateway
      kubernetes_sd_configs:
      - role: service
      relabel_configs:
      - action: keep
        regex: pushgateway
        source_labels:
        - __meta_kubernetes_service_annotation_prometheus_io_probe
    - job_name: kubernetes-services
      kubernetes_sd_configs:
      - role: service
      metrics_path: /probe
      params:
        module:
        - http_2xx
      relabel_configs:
      - action: keep
        regex: true
        source_labels:
        - __meta_kubernetes_service_annotation_prometheus_io_probe
      - source_labels:
        - __address__
        target_label: __param_target
      - replacement: blackbox
        target_label: __address__
      - source_labels:
        - __param_target
        target_label: instance
      - action: labelmap
        regex: __meta_kubernetes_service_label_(.+)
      - source_labels:
        - __meta_kubernetes_namespace
        target_label: kubernetes_namespace
      - source_labels:
        - __meta_kubernetes_service_name
        target_label: kubernetes_name
    - job_name: kubernetes-pods
      kubernetes_sd_configs:
      - role: pod
      relabel_configs:
      - action: keep
        regex: true
        source_labels:
        - __meta_kubernetes_pod_annotation_prometheus_io_scrape
      - action: replace
        regex: (.+)
        source_labels:
        - __meta_kubernetes_pod_annotation_prometheus_io_path
        target_label: __metrics_path__
      - action: replace
        regex: ([^:]+)(?::\d+)?;(\d+)
        replacement: $1:$2
        source_labels:
        - __address__
        - __meta_kubernetes_pod_annotation_prometheus_io_port
        target_label: __address__
      - action: labelmap
        regex: __meta_kubernetes_pod_label_(.+)
      - action: replace
        source_labels:
        - __meta_kubernetes_namespace
        target_label: kubernetes_namespace
      - action: replace
        source_labels:
        - __meta_kubernetes_pod_name
        target_label: kubernetes_pod_name
    - job_name: kubernetes-pods-slow
      kubernetes_sd_configs:
      - role: pod
      relabel_configs:
      - action: keep
        regex: true
        source_labels:
        - __meta_kubernetes_pod_annotation_prometheus_io_scrape_slow
      - action: replace
        regex: (.+)
        source_labels:
        - __meta_kubernetes_pod_annotation_prometheus_io_path
        target_label: __metrics_path__
      - action: replace
        regex: ([^:]+)(?::\d+)?;(\d+)
        replacement: $1:$2
        source_labels:
        - __address__
        - __meta_kubernetes_pod_annotation_prometheus_io_port
        target_label: __address__
      - action: labelmap
        regex: __meta_kubernetes_pod_label_(.+)
      - action: replace
        source_labels:
        - __meta_kubernetes_namespace
        target_label: kubernetes_namespace
      - action: replace
        source_labels:
        - __meta_kubernetes_pod_name
        target_label: kubernetes_pod_name
      scrape_interval: 5m
      scrape_timeout: 30s

  recording_rules.yml: |
    {}

  rules: |
    {}
`

	prometheusServerServiceAccountTemplate = `apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    component: "server"
    app: prometheus
    release: prometheus
    chart: prometheus-11.1.0
  name: {{.PrometheusServerName}}
  namespace: {{.TestNamespace}}
  annotations:
    {}
`

	prometheusServerClusterRoleTemplate = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    component: "server"
    app: prometheus
    release: prometheus
    chart: prometheus-11.1.0
  name: {{.PrometheusServerName}}
  namespace: {{.TestNamespace}}
rules:
  - apiGroups:
      - ""
    resources:
      - nodes
      - nodes/proxy
      - nodes/metrics
      - services
      - endpoints
      - pods
      - ingresses
      - configmaps
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - "extensions"
    resources:
      - ingresses/status
      - ingresses
    verbs:
      - get
      - list
      - watch
  - nonResourceURLs:
      - "/metrics"
    verbs:
      - get
`

	prometheusServerClusterRoleBindingTemplate = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    component: "server"
    app: prometheus
    release: prometheus
    chart: prometheus-11.1.0
  name: {{.PrometheusServerName}}
  namespace: {{.TestNamespace}}
subjects:
  - kind: ServiceAccount
    name: {{.PrometheusServerName}}
    namespace: {{.TestNamespace}}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: {{.PrometheusServerName}}
`

	prometheusServerServiceTemplate = `apiVersion: v1
kind: Service
metadata:
  labels:
    component: "server"
    app: prometheus
    release: prometheus
    chart: prometheus-11.1.0
  name: {{.PrometheusServerName}}
  namespace: {{.TestNamespace}}
spec:
  ports:
    - name: http
      port: 80
      protocol: TCP
      targetPort: 9090
  selector:
    component: "server"
    app: prometheus
    release: prometheus
  sessionAffinity: None
  type: "ClusterIP"
`

	prometheusServerDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    component: "server"
    app: prometheus
    release: prometheus
    chart: prometheus-11.1.0
  name: {{.PrometheusServerName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    matchLabels:
      component: "server"
      app: prometheus
      release: prometheus
  replicas: 1
  template:
    metadata:
      labels:
        component: "server"
        app: prometheus
        release: prometheus
        chart: prometheus-11.1.0
    spec:
      serviceAccountName: {{.PrometheusServerName}}
      containers:
        - name: prometheus-server-configmap-reload
          image: "jimmidyson/configmap-reload:v0.3.0"
          imagePullPolicy: "IfNotPresent"
          args:
            - --volume-dir=/etc/config
            - --webhook-url=http://127.0.0.1:9090/-/reload
          resources:
            {}

          volumeMounts:
            - name: config-volume
              mountPath: /etc/config
              readOnly: true

        - name: prometheus-server
          image: "prom/prometheus:v2.16.0"
          imagePullPolicy: "IfNotPresent"
          args:
            - --storage.tsdb.retention.time=15d
            - --config.file=/etc/config/prometheus.yml
            - --storage.tsdb.path=/data
            - --web.console.libraries=/etc/prometheus/console_libraries
            - --web.console.templates=/etc/prometheus/consoles
            - --web.enable-lifecycle
          ports:
            - containerPort: 9090
          readinessProbe:
            httpGet:
              path: /-/ready
              port: 9090
            initialDelaySeconds: 30
            timeoutSeconds: 30
            failureThreshold: 3
            successThreshold: 1
          livenessProbe:
            httpGet:
              path: /-/healthy
              port: 9090
            initialDelaySeconds: 30
            timeoutSeconds: 30
            failureThreshold: 3
            successThreshold: 1
          resources:
            {}

          volumeMounts:
            - name: config-volume
              mountPath: /etc/config
            - name: storage-volume
              mountPath: /data
              subPath: ""
      securityContext:
        fsGroup: 65534
        runAsGroup: 65534
        runAsNonRoot: true
        runAsUser: 65534

      terminationGracePeriodSeconds: 300
      volumes:
        - name: config-volume
          configMap:
            name: {{.PrometheusServerName}}
        - name: storage-volume
          emptyDir: {}
`

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  pollingInterval: 3
  cooldownPeriod:  1
  triggers:
  - type: prometheus
    metadata:
      serverAddress: http://{{.PrometheusServerName}}.{{.TestNamespace}}.svc
      metricName: http_requests_total
      threshold: '20'
      activationThreshold: '20'
      query: sum(rate(http_requests_total{app="{{.MonitoredAppName}}"}[2m]))
`

	generateLowLevelLoadJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: generate-low-level-requests-job
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - image: quay.io/zroubalik/hey
        name: test
        command: ["/bin/sh"]
        args: ["-c", "for i in $(seq 1 60);do echo $i;/hey -c 5 -n 30 http://{{.MonitoredAppName}}.{{.TestNamespace}}.svc;sleep 1;done"]
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          capabilities:
            drop:
              - ALL
          seccompProfile:
            type: RuntimeDefault
      restartPolicy: Never
  activeDeadlineSeconds: 100
  backoffLimit: 2
  `

	generateLoadJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: generate-requests-job
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - image: quay.io/zroubalik/hey
        name: test
        command: ["/bin/sh"]
        args: ["-c", "for i in $(seq 1 60);do echo $i;/hey -c 5 -n 80 http://{{.MonitoredAppName}}.{{.TestNamespace}}.svc;sleep 1;done"]
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          capabilities:
            drop:
              - ALL
          seccompProfile:
            type: RuntimeDefault
      restartPolicy: Never
  activeDeadlineSeconds: 100
  backoffLimit: 2
`
)

// TestPrometheusScaler creates deployments - there are two deployments - both using the same image but one deployment
// is directly tied to the KEDA HPA while the other is isolated that can be used for metrics
// even when the KEDA deployment is at zero - the service points to both deployments
func TestPrometheusScaler(t *testing.T) {
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, prometheusServerTemplates := getPrometheusServerTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, prometheusServerTemplates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, prometheusServerName, testNamespace, 1, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	// Create kubernetes resources for testing
	data, templates := getTemplateData()
	KubectlApplyMultipleWithTemplate(t, data, templates)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, monitoredAppName, testNamespace, 1, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	testActivation(t, kc, data)
	testScaleUp(t, kc, data)
	testScaleDown(t, kc)

	// cleanup
	KubectlDeleteMultipleWithTemplate(t, data, templates)
	DeleteKubernetesResources(t, kc, testNamespace, data, prometheusServerTemplates)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	templateTriggerJob := templateValues{"generateLowLevelLoadJobTemplate": generateLowLevelLoadJobTemplate}
	KubectlApplyMultipleWithTemplate(t, data, templateTriggerJob)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleUp(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale up ---")
	templateTriggerDeployment := templateValues{"generateLoadJobTemplate": generateLoadJobTemplate}
	KubectlApplyMultipleWithTemplate(t, data, templateTriggerDeployment)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleDown(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale down ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 5),
		"replica count should be %d after 5 minutes", minReplicaCount)
}

func getPrometheusServerTemplateData() (templateData, map[string]string) {
	return templateData{
			TestNamespace:         testNamespace,
			DeploymentName:        deploymentName,
			PublishDeploymentName: publishDeploymentName,
			ScaledObjectName:      scaledObjectName,
			MonitoredAppName:      monitoredAppName,
			PrometheusServerName:  prometheusServerName,
			MinReplicaCount:       minReplicaCount,
			MaxReplicaCount:       maxReplicaCount,
		}, templateValues{
			"prometheusServerConfigMapTemplate":          prometheusServerConfigMapTemplate,
			"prometheusServerServiceAccountTemplate":     prometheusServerServiceAccountTemplate,
			"prometheusServerClusterRoleTemplate":        prometheusServerClusterRoleTemplate,
			"prometheusServerClusterRoleBindingTemplate": prometheusServerClusterRoleBindingTemplate,
			"prometheusServerServiceTemplate":            prometheusServerServiceTemplate,
			"prometheusServerDeploymentTemplate":         prometheusServerDeploymentTemplate,
		}
}

func getTemplateData() (templateData, map[string]string) {
	return templateData{
			TestNamespace:         testNamespace,
			DeploymentName:        deploymentName,
			PublishDeploymentName: publishDeploymentName,
			ScaledObjectName:      scaledObjectName,
			MonitoredAppName:      monitoredAppName,
			PrometheusServerName:  prometheusServerName,
			MinReplicaCount:       minReplicaCount,
			MaxReplicaCount:       maxReplicaCount,
		}, templateValues{
			"deploymentTemplate":             deploymentTemplate,
			"monitoredAppDeploymentTemplate": monitoredAppDeploymentTemplate,
			"monitoredAppServiceTemplate":    monitoredAppServiceTemplate,
			"scaledObjectTemplate":           scaledObjectTemplate,
		}
}
