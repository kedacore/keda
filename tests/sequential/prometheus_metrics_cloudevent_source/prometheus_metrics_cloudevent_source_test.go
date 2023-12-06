//go:build e2e
// +build e2e

package prometheus_metrics_cloudeventsource_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	prommodel "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName              = "prometheus-metrics-ce-test"
	labelCloudEventSource = "cloudeventsource"
	labelType             = "type"
	eventsink             = "eventsink"
	eventsinkValue        = "prometheus-metrics-ce-test-ce"
	eventsinkType         = "eventsinktype"
	eventsinkTypeValue    = "http"
)

var (
	testNamespace              = fmt.Sprintf("%s-ns", testName)
	scaledObjectName           = fmt.Sprintf("%s-so", testName)
	sutDeploymentName          = fmt.Sprintf("%s-sut", testName)
	clientName                 = fmt.Sprintf("%s-client", testName)
	cloudeventSourceName       = fmt.Sprintf("%s-ce", testName)
	wrongCloudEventSourceName  = fmt.Sprintf("%s-ce-w", testName)
	cloudEventHTTPReceiverName = fmt.Sprintf("%s-cloudevent-http-receiver", testName)
	cloudEventHTTPServiceName  = fmt.Sprintf("%s-cloudevent-http-service", testName)
	cloudEventHTTPServiceURL   = fmt.Sprintf("http://%s.%s.svc.cluster.local:8899", cloudEventHTTPServiceName, testNamespace)
	kedaOperatorPrometheusURL  = "http://keda-operator.keda.svc.cluster.local:8080/metrics"
)

type templateData struct {
	TestNamespace              string
	ScaledObject               string
	WrongCloudEventSourceName  string
	SutDeploymentName          string
	ClientName                 string
	CloudEventSourceName       string
	CloudEventHTTPReceiverName string
	CloudEventHTTPServiceName  string
	CloudEventHTTPServiceURL   string
}

const (
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

	scaledObjectTemplate = `
  apiVersion: keda.sh/v1alpha1
  kind: ScaledObject
  metadata:
    name: {{.ScaledObject}}
    namespace: {{.TestNamespace}}
  spec:
    advanced:
      horizontalPodAutoscalerConfig:
        behavior:
          scaleDown:
            stabilizationWindowSeconds: 0
    maxReplicaCount: 2
    minReplicaCount: 1
    scaleTargetRef:
      name: {{.SutDeploymentName}}
    triggers:
      - type: cpu
        metadata:
          type: Utilization
          value: "50"
  `
	sutDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.SutDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    deploy: workload-sut
spec:
  replicas: 0
  selector:
    matchLabels:
      pod: workload-sut
  template:
    metadata:
      labels:
        pod: workload-sut
    spec:
      containers:
      - name: {{.SutDeploymentName}}
        image: registry.k8s.io/hpa-example
        ports:
        - containerPort: 80
        resources:
          limits:
            cpu: 500m
          requests:
            cpu: 200m
        imagePullPolicy: IfNotPresent`

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
)

func TestPrometheusCloudEventSourceMetrics(t *testing.T) {
	// setup
	t.Log("--- setting up ---")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	// scaling to max replica count to ensure the counter is registered before we test it
	assert.True(t, WaitForAllPodRunningInNamespace(t, kc, testNamespace, 5, 20), "all pods should be running")

	testCloudEventEmitted(t)
	testCloudeventSourceSink(t)
	testCloudEventEmittedError(t, data)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

// help function to load template data
func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:              testNamespace,
			ScaledObject:               scaledObjectName,
			WrongCloudEventSourceName:  wrongCloudEventSourceName,
			SutDeploymentName:          sutDeploymentName,
			ClientName:                 clientName,
			CloudEventSourceName:       cloudeventSourceName,
			CloudEventHTTPReceiverName: cloudEventHTTPReceiverName,
			CloudEventHTTPServiceName:  cloudEventHTTPServiceName,
			CloudEventHTTPServiceURL:   cloudEventHTTPServiceURL,
		}, []Template{
			{Name: "cloudEventHTTPReceiverTemplate", Config: cloudEventHTTPReceiverTemplate},
			{Name: "cloudEventHTTPServiceTemplate", Config: cloudEventHTTPServiceTemplate},
			{Name: "clientTemplate", Config: clientTemplate},
			{Name: "cloudEventSourceTemplate", Config: cloudEventSourceTemplate},
			{Name: "sutDeploymentTemplate", Config: sutDeploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
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

func testCloudEventEmitted(t *testing.T) {
	t.Log("--- testing cloudevent emitted ---")

	family := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorPrometheusURL))

	if val, ok := family["keda_cloudeventsource_emitted_total"]; ok {
		var found bool
		metrics := val.GetMetric()
		for _, metric := range metrics {
			labels := metric.GetLabel()
			for _, label := range labels {
				if *label.Name == labelCloudEventSource && *label.Value == cloudeventSourceName {
					assert.GreaterOrEqual(t, *metric.Counter.Value, float64(1))
					found = true
				}
			}
		}
		assert.Equal(t, true, found)
	} else {
		t.Errorf("metric not available")
	}
}

func testCloudeventSourceSink(t *testing.T) {
	t.Log("--- testing cloudevent sink ---")

	family := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorPrometheusURL))

	if val, ok := family["keda_cloudeventsource_sink"]; ok {
		var found bool
		metrics := val.GetMetric()
		for _, metric := range metrics {
			labels := metric.GetLabel()
			var matcheventsink = false
			var matcheventsinktype = false
			for _, label := range labels {
				if *label.Name == eventsinkType && *label.Value == eventsinkTypeValue {
					matcheventsink = true
				}
				if *label.Name == eventsink && *label.Value == eventsinkValue {
					matcheventsinktype = true
				}
			}

			if matcheventsink && matcheventsinktype {
				assert.Equal(t, float64(1), *metric.Gauge.Value)
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

	family := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorPrometheusURL))

	if val, ok := family["keda_cloudeventsource_emitted_errors_total"]; ok {
		var found bool
		metrics := val.GetMetric()
		for _, metric := range metrics {
			labels := metric.GetLabel()
			for _, label := range labels {
				if *label.Name == labelCloudEventSource && *label.Value == cloudeventSourceName {
					assert.Equal(t, float64(5), *metric.Counter.Value)
					found = true
				}
			}
		}
		assert.Equal(t, true, found)
	} else {
		t.Errorf("metric not available")
	}

	KubectlDeleteWithTemplate(t, data, "wrongCloudEventSourceTemplate", wrongCloudEventSourceTemplate)
	KubectlApplyWithTemplate(t, data, "cloudEventSourceTemplate", cloudEventSourceTemplate)
}
