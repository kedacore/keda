import * as sh from 'shelljs'
import * as tmp from 'tmp'
import * as fs from 'fs'
import {createNamespace, waitForRollout} from "./helpers";

export class PrometheusServer {

    static install(t, prometheusServerNamespace: string) {
        const prometheusServerTmpFile = this.getPrometheusFile(prometheusServerNamespace)
        createNamespace(prometheusServerNamespace)

        t.is(0, sh.exec(`kubectl apply -f ${prometheusServerTmpFile.name} --namespace ${prometheusServerNamespace}`).code, 'creating a Prometheus Server deployment should work.')
        // wait for prometheus server
        t.is(0, waitForRollout('deployment', 'prometheus-server', prometheusServerNamespace))
    }

    static uninstall(prometheusServerNamespace: string) {
      const prometheusServerTmpFile = this.getPrometheusFile(prometheusServerNamespace)
      sh.exec(`kubectl delete -f ${prometheusServerTmpFile.name} --namespace ${prometheusServerNamespace}`)

      sh.exec(`kubectl delete namespace ${prometheusServerNamespace}`)
    }

    static getPrometheusFile(prometheusServerNamespace: string){
      let prometheusServerTmpFile = tmp.fileSync()
      fs.writeFileSync(prometheusServerTmpFile.name, prometheusServerYaml
        .replace(/{{NAMESPACE}}/g, prometheusServerNamespace))
        return prometheusServerTmpFile;
    }
}

const prometheusServerYaml = `apiVersion: v1
kind: ConfigMap
metadata:
  labels:
    component: "server"
    app: prometheus
    release: prometheus
  name: prometheus-server
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

---
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    component: "server"
    app: prometheus
    release: prometheus
    chart: prometheus-11.1.0
  name: prometheus-server-{{NAMESPACE}}
  annotations:
    {}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    component: "server"
    app: prometheus
    release: prometheus
    chart: prometheus-11.1.0
  name: prometheus-server-{{NAMESPACE}}
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
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    component: "server"
    app: prometheus
    release: prometheus
    chart: prometheus-11.1.0
  name: prometheus-server-{{NAMESPACE}}
subjects:
  - kind: ServiceAccount
    name: prometheus-server-{{NAMESPACE}}
    namespace: {{NAMESPACE}}
  - kind: ServiceAccount
    name: prometheus-server
    namespace: argo-monitoring
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: prometheus-server-{{NAMESPACE}}
---
apiVersion: v1
kind: Service
metadata:
  labels:
    component: "server"
    app: prometheus
    release: prometheus
    chart: prometheus-11.1.0
  name: prometheus-server
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
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    component: "server"
    app: prometheus
    release: prometheus
    chart: prometheus-11.1.0
  name: prometheus-server
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
      serviceAccountName: prometheus-server-{{NAMESPACE}}
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
            name: prometheus-server
        - name: storage-volume
          emptyDir: {}`
