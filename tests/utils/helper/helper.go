package helper

type EmptyTemplateData struct{}

const (
	AzureManagedPrometheusConfigMapTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: ama-metrics-prometheus-config
  namespace: kube-system
data:
  prometheus-config: |-
    global:
      evaluation_interval: 1m
      scrape_interval: 1m
      scrape_timeout: 10s
    scrape_configs:
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
`
	OtlpConfig = `mode: deployment
image:
  repository: "ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector-contrib"
config:
  exporters:
    debug: {}
    prometheus:
      endpoint: 0.0.0.0:8889
  receivers:
    jaeger: null
    prometheus: null
    zipkin: null
    otlp:
      protocols:
        grpc:
          endpoint: 0.0.0.0:4317
        http:
          endpoint: 0.0.0.0:4318
  service:
    pipelines:
      traces: null
      metrics:
        receivers:
          - otlp
        exporters:
          - debug
          - prometheus
      logs: null
`
	OtlpServicePatch = `apiVersion: v1
kind: Service
metadata:
  name: opentelemetry-collector
spec:
  selector:
    app.kubernetes.io/name: opentelemetry-collector
  ports:
    - protocol: TCP
      port: 8889
      targetPort: 8889
      name: prometheus
  type: ClusterIP

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: opentelemetry-collector
spec:
  template:
    spec:
      containers:
      - name: opentelemetry-collector
        ports:
        - containerPort: 8889
          name: prometheus
          protocol: TCP
`
)
