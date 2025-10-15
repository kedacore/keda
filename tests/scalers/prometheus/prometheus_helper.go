//go:build e2e
// +build e2e

package prometheus

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	"github.com/kedacore/keda/v2/tests/helper"
)

type templateData struct {
	Namespace            string
	PrometheusServerName string
}

type VaultPkiData struct {
	CaCertificate     string
	ServerKey         string
	ServerCertificate string
}

func getPrometheusTemplates(pki *VaultPkiData) []helper.Template {
	return []helper.Template{
		{Name: "prometheusServerServiceAccountTemplate", Config: prometheusServerServiceAccountTemplate},
		{Name: "prometheusServerConfigMapTemplate", Config: getPrometheusServerConfigMapTemplate(pki)},
		{Name: "prometheusServerCertificateSecretTemplate", Config: getPrometheusServerCertificateSecretTemplate(pki)},
		{Name: "prometheusServerClusterRoleTemplate", Config: prometheusServerClusterRoleTemplate},
		{Name: "prometheusServerClusterRoleBindingTemplate", Config: prometheusServerClusterRoleBindingTemplate},
		{Name: "prometheusServerDeploymentTemplate", Config: getPrometheusServerDeploymentTemplate(pki)},
		{Name: "prometheusServerServiceTemplate", Config: prometheusServerServiceTemplate},
	}
}

func getPrometheusServerCertificateSecretTemplate(pki *VaultPkiData) string {
	template := `apiVersion: v1
kind: Secret
metadata:
  labels:
    component: "server"
    app: prometheus
    release: prometheus
  name: {{.PrometheusServerName}}-certificates
  namespace: {{.Namespace}}
data:`
	if pki == nil {
		template += " {}"
		return template
	}
	template += fmt.Sprintf(`
  cert.pem: %s
  key.pem: %s
  ca.pem: %s`, pki.ServerCertificate, pki.ServerKey, pki.CaCertificate)
	return template
}

func getPrometheusServerConfigMapTemplate(pki *VaultPkiData) string {
	var webConfig string
	if pki != nil {
		webConfig = `tls_server_config:
      client_auth_type: VerifyClientCertIfGiven # Force to use this as k8s probe does not support mtls yet
      cert_file: /etc/tls/cert.pem
      key_file: /etc/tls/key.pem
      client_ca_file: /etc/tls/ca.pem`
	} else {
		webConfig = "{}"
	}
	return fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  labels:
    component: "server"
    app: prometheus
    release: prometheus
  name: {{.PrometheusServerName}}
  namespace: {{.Namespace}}
data:
  alerting_rules.yml: |
    {}

  alerts: |
    {}

  web-config.yml: |
    %s

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
    {}`, webConfig)
}

func getPrometheusServerDeploymentTemplate(pki *VaultPkiData) string {
	var probeScheme string
	if pki != nil {
		probeScheme = "HTTPS"
	} else {
		probeScheme = "HTTP"
	}
	return fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    component: "server"
    app: prometheus
    release: prometheus
    chart: prometheus-11.1.0
  name: {{.PrometheusServerName}}
  namespace: {{.Namespace}}
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
          image: "docker.io/jimmidyson/configmap-reload:v0.3.0"
          imagePullPolicy: "IfNotPresent"
          args:
            - --volume-dir=/etc/config
            - --webhook-url=http://127.0.0.1:9090/-/reload

          volumeMounts:
            - name: config-volume
              mountPath: /etc/config
              readOnly: true

        - name: prometheus-server
          image: "docker.io/prom/prometheus:v2.47.1"
          imagePullPolicy: "IfNotPresent"
          args:
            - --storage.tsdb.retention.time=15d
            - --config.file=/etc/config/prometheus.yml
            - --storage.tsdb.path=/data
            - --web.console.libraries=/etc/prometheus/console_libraries
            - --web.console.templates=/etc/prometheus/consoles
            - --web.config.file=/etc/config/web-config.yml
            - --web.enable-lifecycle
          ports:
            - containerPort: 9090
          readinessProbe:
            httpGet:
              path: /-/ready
              port: 9090
              scheme: %s
            initialDelaySeconds: 30
            timeoutSeconds: 30
            failureThreshold: 3
            successThreshold: 1
          livenessProbe:
            httpGet:
              path: /-/healthy
              port: 9090
              scheme: %s
            initialDelaySeconds: 30
            timeoutSeconds: 30
            failureThreshold: 3
            successThreshold: 1

          volumeMounts:
            - name: config-volume
              mountPath: /etc/config
            - name: certificates
              mountPath: /etc/tls
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
        - name: certificates
          secret:
            secretName: {{.PrometheusServerName}}-certificates
        - name: storage-volume
          emptyDir: {}`, probeScheme, probeScheme,
	)
}

const (
	prometheusServerServiceAccountTemplate = `apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    component: "server"
    app: prometheus
    release: prometheus
    chart: prometheus-11.1.0
  name: {{.PrometheusServerName}}
  namespace: {{.Namespace}}
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
  namespace: {{.Namespace}}
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
  namespace: {{.Namespace}}
subjects:
  - kind: ServiceAccount
    name: {{.PrometheusServerName}}
    namespace: {{.Namespace}}
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
  namespace: {{.Namespace}}
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
)

func Install(t *testing.T, kc *kubernetes.Clientset, name, namespace string, pki *VaultPkiData) {
	helper.CreateNamespace(t, kc, namespace)
	var data = templateData{
		Namespace:            namespace,
		PrometheusServerName: name,
	}
	helper.KubectlApplyMultipleWithTemplate(t, data, getPrometheusTemplates(pki))
	require.True(t, helper.WaitForDeploymentReplicaReadyCount(t, kc, name, namespace, 1, 60, 3),
		"replica count should be 1 after 3 minutes")
}

func Uninstall(t *testing.T, name, namespace string, pki *VaultPkiData) {
	var data = templateData{
		Namespace:            namespace,
		PrometheusServerName: name,
	}
	helper.KubectlDeleteMultipleWithTemplate(t, data, getPrometheusTemplates(pki))
}
