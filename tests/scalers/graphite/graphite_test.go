//go:build e2e
// +build e2e

package graphite_test

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
	testName = "graphite-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
	minReplicaCount  = 0
	maxReplicaCount  = 5
)

type templateData struct {
	TestNamespace    string
	DeploymentName   string
	ScaledObjectName string
	SecretName       string
	MinReplicaCount  int
	MaxReplicaCount  int
}

const (
	deploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
spec:
  selector:
    matchLabels:
      run: {{.DeploymentName}}
  replicas: 0
  template:
    metadata:
      labels:
        run: {{.DeploymentName}}
    spec:
      containers:
      - name: php-apache-graphite
        image: registry.k8s.io/hpa-example
        ports:
        - containerPort: 80
`

	// Source: graphite/templates/configmap-statsd.yaml
	graphiteStatsdConfigMapTemplate = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: graphite-statsd-configmap
  namespace: {{.TestNamespace}}
  labels:
    app.kubernetes.io/name: graphite
    helm.sh/chart: graphite-0.7.2
    app.kubernetes.io/instance: RELEASE-NAME
    app.kubernetes.io/managed-by: Helm
data:
  config_tcp.js: |-
    {
      "graphiteHost": "127.0.0.1",
      "graphitePort": 2003,
      "port": 8125,
      "flushInterval": 10000,
      "servers": [{
        "server": "./servers/tcp",
        "address": "0.0.0.0",
        "port": 8125
      }]
    }
  config_udp.js: |-
    {
      "graphiteHost": "127.0.0.1",
      "graphitePort": 2003,
      "port": 8125,
      "flushInterval": 10000,
      "servers": [{
        "server": "./servers/udp",
        "address": "0.0.0.0",
        "port": 8125
      }]
    }
`

	// Source: graphite/templates/configmap.yaml
	graphiteConfigMapTemplate = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: graphite-configmap
  namespace: {{.TestNamespace}}
  labels:
    app: graphite
    chart: graphite-0.7.2
    release: RELEASE-NAME
    heritage: Helm
data:
  aggregation-rules.conf: |-

  carbon.conf: |-
    [cache]
    DATABASE = whisper
    ENABLE_LOGROTATION = True
    USER =
    MAX_CACHE_SIZE = inf
    MAX_UPDATES_PER_SECOND = 500
    MAX_CREATES_PER_MINUTE = 50
    MIN_TIMESTAMP_RESOLUTION = 1
    LINE_RECEIVER_INTERFACE = 0.0.0.0
    LINE_RECEIVER_PORT = 2003
    ENABLE_UDP_LISTENER = False
    UDP_RECEIVER_INTERFACE = 0.0.0.0
    UDP_RECEIVER_PORT = 2003
    PICKLE_RECEIVER_INTERFACE = 0.0.0.0
    PICKLE_RECEIVER_PORT = 2004
    USE_INSECURE_UNPICKLER = False
    CACHE_QUERY_INTERFACE = 0.0.0.0
    CACHE_QUERY_PORT = 7002
    USE_FLOW_CONTROL = True
    LOG_UPDATES = False
    LOG_CREATES = False
    LOG_CACHE_HITS = False
    LOG_CACHE_QUEUE_SORTS = False
    CACHE_WRITE_STRATEGY = sorted
    WHISPER_AUTOFLUSH = False
    WHISPER_FALLOCATE_CREATE = True
    CARBON_METRIC_INTERVAL = 10

    GRAPHITE_URL = http://127.0.0.1:8080

    [relay]
    LINE_RECEIVER_INTERFACE = 0.0.0.0
    LINE_RECEIVER_PORT = 2013
    PICKLE_RECEIVER_INTERFACE = 0.0.0.0
    PICKLE_RECEIVER_PORT = 2014

    RELAY_METHOD = rules
    REPLICATION_FACTOR = 1
    DESTINATIONS = 127.0.0.1:2004
    MAX_QUEUE_SIZE = 10000
    MAX_DATAPOINTS_PER_MESSAGE = 500
    QUEUE_LOW_WATERMARK_PCT = 0.8
    TIME_TO_DEFER_SENDING = 0.0001
    USE_FLOW_CONTROL = True
    CARBON_METRIC_INTERVAL = 10
    USE_RATIO_RESET=False
    MIN_RESET_STAT_FLOW=1000
    MIN_RESET_RATIO=0.9
    MIN_RESET_INTERVAL=121

    [aggregator]
    LINE_RECEIVER_INTERFACE = 0.0.0.0
    LINE_RECEIVER_PORT = 2023

    PICKLE_RECEIVER_INTERFACE = 0.0.0.0
    PICKLE_RECEIVER_PORT = 2024

    # If set true, metric received will be forwarded to DESTINATIONS in addition to
    # the output of the aggregation rules. If set false the carbon-aggregator will
    # only ever send the output of aggregation.
    FORWARD_ALL = True
    DESTINATIONS = 127.0.0.1:2004
    REPLICATION_FACTOR = 1
    MAX_QUEUE_SIZE = 10000
    USE_FLOW_CONTROL = True
    MAX_DATAPOINTS_PER_MESSAGE = 500
    MAX_AGGREGATION_INTERVALS = 5
    CARBON_METRIC_INTERVAL = 10
  dashboard.conf: |-
    # This configuration file controls the behavior of the Dashboard UI, available
    # at http://my-graphite-server/dashboard/.
    #
    # This file must contain a [ui] section that defines values for all of the
    # following settings.
    [ui]
    default_graph_width = 400
    default_graph_height = 250
    automatic_variants = true
    refresh_interval = 60
    autocomplete_delay = 375
    merge_hover_delay = 750

    # You can set this 'default', 'white', or a custom theme name.
    # To create a custom theme, copy the dashboard-default.css file
    # to dashboard-myThemeName.css in the content/css directory and
    # modify it to your liking.
    theme = default

    [keyboard-shortcuts]
    toggle_toolbar = ctrl-z
    toggle_metrics_panel = ctrl-space
    erase_all_graphs = alt-x
    save_dashboard = alt-s
    completer_add_metrics = alt-enter
    completer_del_metrics = alt-backspace
    give_completer_focus = shift-space
  graphTemplates.conf: |-
    [default]
    background = black
    foreground = white
    majorLine = white
    minorLine = grey
    lineColors = blue,green,red,purple,brown,yellow,aqua,grey,magenta,pink,gold,rose
    fontName = Sans
    fontSize = 10
    fontBold = False
    fontItalic = False

    [noc]
    background = black
    foreground = white
    majorLine = white
    minorLine = grey
    lineColors = blue,green,red,yellow,purple,brown,aqua,grey,magenta,pink,gold,rose
    fontName = Sans
    fontSize = 10
    fontBold = False
    fontItalic = False

    [plain]
    background = white
    foreground = black
    minorLine = grey
    majorLine = rose

    [summary]
    background = black
    lineColors = #6666ff, #66ff66, #ff6666

    [alphas]
    background = white
    foreground = black
    majorLine = grey
    minorLine = rose
    lineColors = 00ff00aa,ff000077,00337799
  graphite.wsgi.example: |-
    import sys
    sys.path.append('/opt/graphite/webapp')

    from graphite.wsgi import application
  relay-rules.conf: |-
    [default]
    default = true
    destinations = 0.0.0.0:2004
  rewrite-rules.conf: |-

  storage-aggregation.conf: |-
    [min]
    pattern = \.lower$
    xFilesFactor = 0.1
    aggregationMethod = min

    [max]
    pattern = \.upper(_\d+)?$
    xFilesFactor = 0.1
    aggregationMethod = max

    [sum]
    pattern = \.sum$
    xFilesFactor = 0
    aggregationMethod = sum

    [count]
    pattern = \.count$
    xFilesFactor = 0
    aggregationMethod = sum

    [count_legacy]
    pattern = ^stats_counts.*
    xFilesFactor = 0
    aggregationMethod = sum

    [default_average]
    pattern = .*
    xFilesFactor = 0.3
    aggregationMethod = average
  storage-schemas.conf: |-
    [carbon]
    pattern = ^carbon\.
    retentions = 10s:6h,1m:90d

    [default_1min_for_1day]
    pattern = .*
    retentions = 10s:6h,1m:6d,10m:1800d
`

	// Source: graphite/templates/service.yaml
	graphiteServiceTemplate = `apiVersion: v1
kind: Service
metadata:
  name: graphite
  namespace: {{.TestNamespace}}
  labels:
    app.kubernetes.io/name: graphite
    helm.sh/chart: graphite-0.7.2
    app.kubernetes.io/instance: RELEASE-NAME
    app.kubernetes.io/managed-by: Helm
spec:
  type: ClusterIP
  ports:
  - name: graphite-pickle
    port: 2004
    protocol: TCP
  - name: graphite-plain
    port: 2003
    protocol: TCP
  - name: graphite-udp
    port: 2003
    protocol: UDP
  - name: graphite-gui
    port: 8080
    protocol: TCP
  - name: aggregate-plain
    port: 2023
    protocol: TCP
  - name: aggregate-pickl
    port: 2024
    protocol: TCP
  - name: statsd
    port: 8125
    protocol: UDP
  - name: statsd-admin
    port: 8126
    protocol: TCP
  selector:
    app.kubernetes.io/name: graphite
    app.kubernetes.io/instance: RELEASE-NAME
`

	// Source: graphite/templates/statefulset.yaml
	graphiteStatefulSetTemplate = `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: graphite
  namespace: {{.TestNamespace}}
  labels:
    app.kubernetes.io/name: graphite
    helm.sh/chart: graphite-0.7.2
    app.kubernetes.io/instance: RELEASE-NAME
    app.kubernetes.io/managed-by: Helm
spec:
  updateStrategy:
    type: RollingUpdate
  selector:
    matchLabels:
      app.kubernetes.io/name: graphite
      app.kubernetes.io/instance: RELEASE-NAME
  serviceName: graphite
  template:
    metadata:
      labels:
        app.kubernetes.io/name: graphite
        app.kubernetes.io/instance: RELEASE-NAME
    spec:
      containers:
      - image: graphiteapp/graphite-statsd:1.1.7-6
        name: graphite
        ports:
        - name: graphite-gui
          containerPort: 8080
        - name: graphite-plain
          containerPort: 2003
        - name: graphite-udp
          containerPort: 2003
          protocol: UDP
        - name: graphite-pickle
          containerPort: 2004
        - name: aggregate-plain
          containerPort: 2023
        - name: aggregate-pickl
          containerPort: 2024
        - name: statsd
          protocol: UDP
          containerPort: 8125
        - name: statsd-admin
          containerPort: 8126
        env:
        - name: "STATSD_INTERFACE"
          value: udp
        - name: "GRAPHITE_TIME_ZONE"
          value: Etc/UTC
        livenessProbe:
          httpGet:
            path: /
            port: graphite-gui
        readinessProbe:
          httpGet:
            path: /
            port: graphite-gui

        volumeMounts:
          - name: graphite-configmap
            mountPath: /opt/graphite/conf/
          - name: graphite-statsd-configmap
            subPath: config_tcp.js
            mountPath: /opt/statsd/config/tcp.js
          - name: graphite-statsd-configmap
            subPath: config_udp.js
            mountPath: /opt/statsd/config/udp.js
          - name: graphite-pvc
            mountPath: /opt/graphite/storage/
      volumes:
        - name: graphite-configmap
          configMap:
            name: graphite-configmap
        - name: graphite-statsd-configmap
          configMap:
            name: graphite-statsd-configmap
        - name: graphite-pvc
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
  pollingInterval: 5
  cooldownPeriod:  10
  triggers:
  - type: graphite
    metadata:
      serverAddress: http://graphite.{{.TestNamespace}}.svc:8080
      threshold: '100'
      activationThreshold: '50'
      query: "https_metric"
      queryTime: '-10Seconds'
`

	requestsJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: generate-load-graphite-metrics
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 0
  template:
    spec:
      containers:
      - image: busybox
        name: generate-graphite-metrics
        command: ["/bin/sh"]
        args:
        - -c
        - for i in $(seq 1 60);do echo $i; echo "https_metric 1000 $(date +%s)" | nc graphite.{{.TestNamespace}}.svc 2003; echo 'data sent :)'; sleep 1; done
      restartPolicy: Never
  activeDeadlineSeconds: 120
  backoffLimit: 2
`

	lowLevelRequestsJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: generate-low-load-graphite-metrics
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 0
  template:
    spec:
      containers:
      - image: busybox
        name: generate-graphite-metrics
        command: ["/bin/sh"]
        args:
        - -c
        - for i in $(seq 1 60);do echo $i; echo "https_metric 10 $(date +%s)" | nc graphite.{{.TestNamespace}}.svc 2003; echo 'data sent :)'; sleep 1; done
      restartPolicy: Never
  activeDeadlineSeconds: 120
  backoffLimit: 2
`

	emptyRequestsJobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: generate-empty-load-graphite-metrics
  namespace: {{.TestNamespace}}
spec:
  ttlSecondsAfterFinished: 0
  template:
    spec:
      containers:
      - image: busybox
        name: generate-graphite-metrics
        command: ["/bin/sh"]
        args:
        - -c
        - for i in $(seq 1 60);do echo $i; echo "https_metric 0 $(date +%s)" | nc graphite.{{.TestNamespace}}.svc 2003; echo 'data sent :)'; sleep 1; done
      restartPolicy: Never
  activeDeadlineSeconds: 120
  backoffLimit: 2
`
)

func TestGraphiteScaler(t *testing.T) {
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, "graphite", testNamespace, 1, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	testActivation(t, kc, data)
	testScaleOut(t, kc, data)
	testScaleIn(t, kc, data)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func testActivation(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing activation ---")
	KubectlReplaceWithTemplate(t, data, "lowLevelRequestsJobTemplate", lowLevelRequestsJobTemplate)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale out ---")
	KubectlReplaceWithTemplate(t, data, "requestsJobTemplate", requestsJobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing scale in ---")
	KubectlReplaceWithTemplate(t, data, "emptyRequestsJobTemplate", emptyRequestsJobTemplate)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:    testNamespace,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
			MinReplicaCount:  minReplicaCount,
			MaxReplicaCount:  maxReplicaCount,
		}, []Template{
			{Name: "graphiteStatsdConfigMapTemplate", Config: graphiteStatsdConfigMapTemplate},
			{Name: "graphiteConfigMapTemplate", Config: graphiteConfigMapTemplate},
			{Name: "graphiteServiceTemplate", Config: graphiteServiceTemplate},
			{Name: "graphiteStatefulSetTemplate", Config: graphiteStatefulSetTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
