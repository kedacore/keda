//go:build e2e
// +build e2e

package prometheus_metrics_test

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/kedacore/keda/v2/tests/helper"
	prommodel "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
)

const (
	testName              = "prometheus-metrics-test"
	labelCloudEventSource = "cloudeventsource"
	labelType             = "type"
)

var (
	testNamespace              = fmt.Sprintf("%s-ns", testName)
	scaledObjectName           = fmt.Sprintf("%s-so", testName)
	sutDeploymentName          = fmt.Sprintf("%s-sut", testName)
	clientName                 = fmt.Sprintf("%s-client", testName)
	cloudeventSourceName       = fmt.Sprintf("%s-ce", testName)
	cloudEventHTTPReceiverName = fmt.Sprintf("%s-cloudevent-http-receiver", testName)
	cloudEventHTTPServiceName  = fmt.Sprintf("%s-cloudevent-http-service", testName)
	cloudEventHTTPServiceURL   = fmt.Sprintf("http://%s.%s.svc.cluster.local:8899", cloudEventHTTPServiceName, testNamespace)
	kedaOperatorPrometheusURL  = "http://keda-operator.keda.svc.cluster.local:8080/metrics"
)

type templateData struct {
	TestNamespace              string
	ScaledObject               string
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

func TestPrometheusMetrics(t *testing.T) {
	// setup
	t.Log("--- setting up ---")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	// scaling to max replica count to ensure the counter is registered before we test it
	assert.True(t, WaitForAllPodRunningInNamespace(t, kc, testNamespace, 5, 20), "all pods should be running")

	// testCloudEventEmitted(t)
	testCloudEventQueueStatus(t)

	// testScalerMetricValue(t)

	// cleanup
	// DeleteKubernetesResources(t, testNamespace, data, templates)
}

// help function to load template data
func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:              testNamespace,
			ScaledObject:               scaledObjectName,
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
	// t.Log(fmt.Sprintf("--- vvvvvvvvvvvvvv --- %s", out))
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
					assert.Equal(t, float64(1), *metric.Counter.Value)
					found = true
				}
			}
		}
		assert.Equal(t, true, found)
	} else {
		t.Errorf("metric not available")
	}
}

func testCloudEventQueueStatus(t *testing.T) {
	t.Log("--- testing cloudevent emitted ---")

	family := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorPrometheusURL))
	for k, _ := range family {
		t.Log(fmt.Sprintf("--- vvvvvvvvvvvvvv --- %s", k))
	}
	if val, ok := family["keda_cloudeventsource_queue"]; ok {
		var found bool
		metrics := val.GetMetric()
		for _, metric := range metrics {
			labels := metric.GetLabel()
			t.Log(fmt.Sprintf("--- ssssssssss --- %s", metric))
			for _, label := range labels {
				t.Log(fmt.Sprintf("--- ssssssssss --- %s %f", *label.Name, *metric.Gauge.Value))
				if *label.Name == labelCloudEventSource && *label.Value == cloudeventSourceName {
					t.Log(fmt.Sprintf("--- vvvvvvvvvvvvvv --- %s %f", *label.Name, *metric.Gauge.Value))
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

// func testScalerErrors(t *testing.T, data templateData) {
// 	t.Log("--- testing scaler errors ---")

// 	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
// 	KubectlApplyWithTemplate(t, data, "wrongScaledObjectTemplate", wrongScaledObjectTemplate)

// 	family := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorPrometheusURL))
// 	if val, ok := family["keda_scaler_errors"]; ok {
// 		errCounterVal1 := getErrorMetricsValue(val)

// 		// wait for 20 seconds to correctly fetch metrics.
// 		time.Sleep(20 * time.Second)

// 		family = fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorPrometheusURL))
// 		if val, ok := family["keda_scaler_errors"]; ok {
// 			errCounterVal2 := getErrorMetricsValue(val)
// 			assert.NotEqual(t, errCounterVal2, float64(0))
// 			assert.GreaterOrEqual(t, errCounterVal2, errCounterVal1)
// 		} else {
// 			t.Errorf("metric not available")
// 		}
// 	} else {
// 		t.Errorf("metric not available")
// 	}

// 	KubectlDeleteWithTemplate(t, data, "wrongScaledObjectTemplate", wrongScaledObjectTemplate)
// 	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
// }

// func testScalerErrorsTotal(t *testing.T, data templateData) {
// 	t.Log("--- testing scaler errors total ---")

// 	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
// 	KubectlApplyWithTemplate(t, data, "wrongScaledObjectTemplate", wrongScaledObjectTemplate)

// 	family := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorPrometheusURL))
// 	if val, ok := family["keda_scaler_errors_total"]; ok {
// 		errCounterVal1 := getErrorMetricsValue(val)

// 		// wait for 2 seconds as pollinginterval is 2
// 		time.Sleep(2 * time.Second)

// 		family = fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorPrometheusURL))
// 		if val, ok := family["keda_scaler_errors_total"]; ok {
// 			errCounterVal2 := getErrorMetricsValue(val)
// 			assert.NotEqual(t, errCounterVal2, float64(0))
// 			assert.GreaterOrEqual(t, errCounterVal2, errCounterVal1)
// 		} else {
// 			t.Errorf("metric not available")
// 		}
// 	} else {
// 		t.Errorf("metric not available")
// 	}

// 	KubectlDeleteWithTemplate(t, data, "wrongScaledObjectTemplate", wrongScaledObjectTemplate)
// 	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
// }

// func getErrorMetricsValue(val *prommodel.MetricFamily) float64 {
// 	switch val.GetName() {
// 	case "keda_scaler_errors_total":
// 		metrics := val.GetMetric()
// 		for _, metric := range metrics {
// 			return metric.GetCounter().GetValue()
// 		}
// 	case "keda_scaled_object_errors":
// 		metrics := val.GetMetric()
// 		for _, metric := range metrics {
// 			labels := metric.GetLabel()
// 			for _, label := range labels {
// 				if *label.Name == "scaledObject" && *label.Value == wrongScaledObjectName {
// 					return *metric.Counter.Value
// 				}
// 			}
// 		}
// 	case "keda_scaler_errors":
// 		metrics := val.GetMetric()
// 		for _, metric := range metrics {
// 			labels := metric.GetLabel()
// 			for _, label := range labels {
// 				if *label.Name == "scaler" && *label.Value == wrongScalerName {
// 					return *metric.Counter.Value
// 				}
// 			}
// 		}
// 	}
// 	return 0
// }

// func assertScaledObjectPausedMetric(t *testing.T, families map[string]*prommodel.MetricFamily, scaledObjectName string, expected bool) {
// 	family, ok := families["keda_scaled_object_paused"]
// 	if !ok {
// 		t.Errorf("keda_scaled_object_paused metric not available")
// 		return
// 	}

// 	metricValue := 0.0
// 	metrics := family.GetMetric()
// 	for _, metric := range metrics {
// 		labels := metric.GetLabel()
// 		for _, label := range labels {
// 			if *label.Name == labelScaledObject && *label.Value == scaledObjectName {
// 				metricValue = *metric.Gauge.Value
// 			}
// 		}
// 	}

// 	expectedMetricValue := 0
// 	if expected {
// 		expectedMetricValue = 1
// 	}
// 	assert.Equal(t, float64(expectedMetricValue), metricValue)
// }

// func testScalerMetricLatency(t *testing.T) {
// 	t.Log("--- testing scaler metric latency ---")

// 	family := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorPrometheusURL))

// 	if val, ok := family["keda_scaler_metrics_latency"]; ok {
// 		var found bool
// 		metrics := val.GetMetric()
// 		for _, metric := range metrics {
// 			labels := metric.GetLabel()
// 			for _, label := range labels {
// 				if *label.Name == labelScaledObject && *label.Value == scaledObjectName {
// 					assert.Equal(t, float64(0), *metric.Gauge.Value)
// 					found = true
// 				}
// 			}
// 		}
// 		assert.Equal(t, true, found)
// 	} else {
// 		t.Errorf("metric not available")
// 	}
// }

// func testScalableObjectMetrics(t *testing.T) {
// 	t.Log("--- testing scalable objects latency ---")

// 	family := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorPrometheusURL))

// 	if val, ok := family["keda_internal_scale_loop_latency"]; ok {
// 		var found bool
// 		metrics := val.GetMetric()

// 		// check scaledobject loop
// 		found = false
// 		for _, metric := range metrics {
// 			labels := metric.GetLabel()
// 			for _, label := range labels {
// 				if *label.Name == labelType && *label.Value == "scaledobject" {
// 					found = true
// 				}
// 			}
// 		}
// 		assert.Equal(t, true, found)

// 		// check scaledjob loop
// 		found = false
// 		for _, metric := range metrics {
// 			labels := metric.GetLabel()
// 			for _, label := range labels {
// 				if *label.Name == labelType && *label.Value == "scaledjob" {
// 					found = true
// 				}
// 			}
// 		}
// 		assert.Equal(t, true, found)
// 	} else {
// 		t.Errorf("scaledobject metric not available")
// 	}
// }

// func testScalerActiveMetric(t *testing.T) {
// 	t.Log("--- testing scaler active metric ---")

// 	family := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorPrometheusURL))

// 	if val, ok := family["keda_scaler_active"]; ok {
// 		var found bool
// 		metrics := val.GetMetric()
// 		for _, metric := range metrics {
// 			labels := metric.GetLabel()
// 			for _, label := range labels {
// 				if *label.Name == labelScaledObject && *label.Value == scaledObjectName {
// 					assert.Equal(t, float64(1), *metric.Gauge.Value)
// 					found = true
// 				}
// 			}
// 		}
// 		assert.Equal(t, true, found)
// 	} else {
// 		t.Errorf("metric not available")
// 	}
// }

// func testScaledObjectPausedMetric(t *testing.T, data templateData) {
// 	t.Log("--- testing scaleobject pause metric ---")

// 	// Pause the ScaledObject
// 	KubectlApplyWithTemplate(t, data, "scaledObjectPausedTemplate", scaledObjectPausedTemplate)
// 	time.Sleep(20 * time.Second)

// 	// Check that the paused metric is now true
// 	families := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorPrometheusURL))
// 	assertScaledObjectPausedMetric(t, families, scaledObjectName, true)

// 	// Unpause the ScaledObject
// 	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
// 	time.Sleep(20 * time.Second)

// 	// Check that the paused metric is back to false
// 	families = fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorPrometheusURL))
// 	assertScaledObjectPausedMetric(t, families, scaledObjectName, false)
// }

// func testOperatorMetrics(t *testing.T, kc *kubernetes.Clientset, data templateData) {
// 	t.Log("--- testing operator metrics ---")
// 	testOperatorMetricValues(t, kc)

// 	KubectlApplyWithTemplate(t, data, "cronScaledJobTemplate", cronScaledJobTemplate)
// 	testOperatorMetricValues(t, kc)

// 	KubectlDeleteWithTemplate(t, data, "cronScaledJobTemplate", cronScaledJobTemplate)
// 	testOperatorMetricValues(t, kc)
// }

// func testWebhookMetrics(t *testing.T, data templateData) {
// 	t.Log("--- testing webhook metrics ---")

// 	data.ScaledObjectName = "other-so"
// 	err := KubectlApplyWithErrors(t, data, "scaledObjectTemplate", scaledObjectTemplate)
// 	assert.Errorf(t, err, "can deploy the scaledObject - %s", err)
// 	testWebhookMetricValues(t)
// 	data.ScaledObjectName = scaledObjectName
// }

// func getOperatorMetricsManually(t *testing.T, kc *kubernetes.Clientset) (map[string]int, map[string]map[string]int) {
// 	kedaKc := GetKedaKubernetesClient(t)

// 	triggerTotals := make(map[string]int)
// 	crTotals := map[string]map[string]int{
// 		"scaled_object":                  {},
// 		"scaled_job":                     {},
// 		"trigger_authentication":         {},
// 		"cluster_trigger_authentication": {},
// 	}

// 	namespaceList, err := kc.CoreV1().Namespaces().List(context.Background(), v1.ListOptions{})
// 	assert.NoErrorf(t, err, "failed to list namespaces - %s", err)

// 	clusterTriggerAuthenticationList, err := kedaKc.ClusterTriggerAuthentications().List(context.Background(), v1.ListOptions{})
// 	assert.NoErrorf(t, err, "failed to list clusterTriggerAuthentications with err - %s")

// 	for _, clusterTriggerAuth := range clusterTriggerAuthenticationList.Items {
// 		namespace := clusterTriggerAuth.Namespace
// 		if namespace == "" {
// 			namespace = "default"
// 		}
// 		crTotals[metricscollector.ClusterTriggerAuthenticationResource][namespace]++
// 	}

// 	for _, namespace := range namespaceList.Items {
// 		namespaceName := namespace.Name
// 		if namespace.Name == "" {
// 			namespaceName = "default"
// 		}

// 		scaledObjectList, err := kedaKc.ScaledObjects(namespace.Name).List(context.Background(), v1.ListOptions{})
// 		assert.NoErrorf(t, err, "failed to list scaledObjects in namespace - %s with err - %s", namespace.Name, err)

// 		crTotals[metricscollector.ScaledObjectResource][namespaceName] = len(scaledObjectList.Items)
// 		for _, scaledObject := range scaledObjectList.Items {
// 			for _, trigger := range scaledObject.Spec.Triggers {
// 				triggerTotals[trigger.Type]++
// 			}
// 		}

// 		scaledJobList, err := kedaKc.ScaledJobs(namespace.Name).List(context.Background(), v1.ListOptions{})
// 		assert.NoErrorf(t, err, "failed to list scaledJobs in namespace - %s with err - %s", namespace.Name, err)

// 		crTotals[metricscollector.ScaledJobResource][namespaceName] = len(scaledJobList.Items)
// 		for _, scaledJob := range scaledJobList.Items {
// 			for _, trigger := range scaledJob.Spec.Triggers {
// 				triggerTotals[trigger.Type]++
// 			}
// 		}

// 		triggerAuthList, err := kedaKc.TriggerAuthentications(namespace.Name).List(context.Background(), v1.ListOptions{})
// 		assert.NoErrorf(t, err, "failed to list triggerAuthentications in namespace - %s with err - %s", namespace.Name, err)

// 		crTotals[metricscollector.TriggerAuthenticationResource][namespaceName] = len(triggerAuthList.Items)
// 	}

// 	return triggerTotals, crTotals
// }

// func testWebhookMetricValues(t *testing.T) {
// 	families := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaWebhookPrometheusURL))
// 	checkWebhookValues(t, families)
// }

// func testMetricServerMetrics(t *testing.T) {
// 	families := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaMetricsServerPrometheusURL))
// 	checkMetricServerValues(t, families)
// }

// func testOperatorMetricValues(t *testing.T, kc *kubernetes.Clientset) {
// 	families := fetchAndParsePrometheusMetrics(t, fmt.Sprintf("curl --insecure %s", kedaOperatorPrometheusURL))
// 	expectedTriggerTotals, expectedCrTotals := getOperatorMetricsManually(t, kc)

// 	checkTriggerTotalValues(t, families, expectedTriggerTotals)
// 	checkCRTotalValues(t, families, expectedCrTotals)
// 	checkBuildInfo(t, families)
// }

// func checkBuildInfo(t *testing.T, families map[string]*prommodel.MetricFamily) {
// 	t.Log("--- testing build info metric ---")

// 	family, ok := families["keda_build_info"]
// 	if !ok {
// 		t.Errorf("metric not available")
// 		return
// 	}

// 	latestCommit := getLatestCommit(t)
// 	expected := map[string]string{
// 		"git_commit": latestCommit,
// 		"goos":       "linux",
// 	}

// 	metrics := family.GetMetric()
// 	for _, metric := range metrics {
// 		labels := metric.GetLabel()
// 		for _, labelPair := range labels {
// 			if expectedValue, ok := expected[*labelPair.Name]; ok {
// 				assert.EqualValues(t, expectedValue, *labelPair.Value, "values do not match for label %s", *labelPair.Name)
// 			}
// 		}
// 		assert.EqualValues(t, 1, metric.GetGauge().GetValue())
// 	}
// }

// func getLatestCommit(t *testing.T) string {
// 	cmd := exec.Command("git", "rev-parse", "HEAD")
// 	var out bytes.Buffer
// 	cmd.Stdout = &out
// 	err := cmd.Run()
// 	require.NoError(t, err)

// 	return strings.Trim(out.String(), "\n")
// }

// func checkTriggerTotalValues(t *testing.T, families map[string]*prommodel.MetricFamily, expected map[string]int) {
// 	t.Log("--- testing trigger total metrics ---")

// 	family, ok := families["keda_trigger_totals"]
// 	if !ok {
// 		t.Errorf("metric not available")
// 		return
// 	}

// 	metrics := family.GetMetric()
// 	for _, metric := range metrics {
// 		labels := metric.GetLabel()
// 		for _, label := range labels {
// 			if *label.Name == labelType {
// 				triggerType := *label.Value
// 				metricValue := *metric.Gauge.Value
// 				expectedMetricValue := float64(expected[triggerType])

// 				assert.Equalf(t, expectedMetricValue, metricValue, "expected %f got %f for trigger type %s",
// 					expectedMetricValue, metricValue, triggerType)

// 				delete(expected, triggerType)
// 			}
// 		}
// 	}

// 	assert.Equal(t, 0, len(expected))
// }

// func checkCRTotalValues(t *testing.T, families map[string]*prommodel.MetricFamily, expected map[string]map[string]int) {
// 	t.Log("--- testing resource total metrics ---")

// 	family, ok := families["keda_resource_totals"]
// 	if !ok {
// 		t.Errorf("metric not available")
// 		return
// 	}

// 	metrics := family.GetMetric()
// 	for _, metric := range metrics {
// 		labels := metric.GetLabel()
// 		var namespace, crType string
// 		for _, label := range labels {
// 			if *label.Name == labelType {
// 				crType = *label.Value
// 			} else if *label.Name == namespaceString {
// 				namespace = *label.Value
// 			}
// 		}

// 		metricValue := *metric.Gauge.Value
// 		expectedMetricValue := float64(expected[crType][namespace])

// 		assert.Equalf(t, expectedMetricValue, metricValue, "expected %f got %f for cr type %s & namespace %s",
// 			expectedMetricValue, metricValue, crType, namespace)
// 	}
// }

// func checkWebhookValues(t *testing.T, families map[string]*prommodel.MetricFamily) {
// 	t.Log("--- testing webhook metrics ---")

// 	family, ok := families["keda_webhook_scaled_object_validation_errors"]
// 	if !ok {
// 		t.Errorf("metric keda_webhook_scaled_object_validation_errors not available")
// 		return
// 	}

// 	metricValue := 0.0
// 	metrics := family.GetMetric()
// 	for _, metric := range metrics {
// 		labels := metric.GetLabel()
// 		for _, label := range labels {
// 			if *label.Name == namespaceString && *label.Value != testNamespace {
// 				continue
// 			}
// 		}
// 		metricValue += *metric.Counter.Value
// 	}
// 	assert.GreaterOrEqual(t, metricValue, 1.0, "keda_webhook_scaled_object_validation_errors has to be greater than 0")

// 	family, ok = families["keda_webhook_scaled_object_validation_total"]
// 	if !ok {
// 		t.Errorf("metric keda_webhook_scaled_object_validation_total not available")
// 		return
// 	}

// 	metricValue = 0.0
// 	metrics = family.GetMetric()
// 	for _, metric := range metrics {
// 		labels := metric.GetLabel()
// 		for _, label := range labels {
// 			if *label.Name == namespaceString && *label.Value != testNamespace {
// 				continue
// 			}
// 		}
// 		metricValue += *metric.Counter.Value
// 	}
// 	assert.GreaterOrEqual(t, metricValue, 1.0, "keda_webhook_scaled_object_validation_total has to be greater than 0")
// }

// func checkMetricServerValues(t *testing.T, families map[string]*prommodel.MetricFamily) {
// 	t.Log("--- testing metric server metrics ---")

// 	family, ok := families["workqueue_adds_total"]
// 	if !ok {
// 		t.Errorf("metric workqueue_adds_total not available")
// 		return
// 	}

// 	metricValue := 0.0
// 	metrics := family.GetMetric()
// 	for _, metric := range metrics {
// 		metricValue += *metric.Counter.Value
// 	}
// 	assert.GreaterOrEqual(t, metricValue, 1.0, "workqueue_adds_total has to be greater than 0")

// 	family, ok = families["apiserver_request_total"]
// 	if !ok {
// 		t.Errorf("metric apiserver_request_total not available")
// 		return
// 	}

// 	metricValue = 0.0
// 	metrics = family.GetMetric()
// 	for _, metric := range metrics {
// 		labels := metric.GetLabel()
// 		for _, label := range labels {
// 			if *label.Name == "group" && *label.Value == "external.metrics.k8s.io" {
// 				metricValue = *metric.Counter.Value
// 			}
// 		}
// 	}
// 	assert.GreaterOrEqual(t, metricValue, 1.0, "apiserver_request_total has to be greater than 0")
// }
