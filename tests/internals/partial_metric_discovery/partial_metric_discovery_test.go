//go:build e2e
// +build e2e

package partial_metric_discovery_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "partial-metric-discovery-test"
)

// While an external-push scaler is unreachable, its trigger has to keep its
// place in the HPA and in ScaledObject.Status.ExternalMetricNames. The push handler
// resolves the metric name of an activation through that status field, so a trigger that
// is missing from it drops every activation, even after the scaler has recovered.
//
// pollingInterval is far longer than the test on purpose: reaching one replica can then
// only come from a pushed activation, not from the scale loop.
var (
	testNamespace         = fmt.Sprintf("%s-ns", testName)
	serviceName           = fmt.Sprintf("%s-service", testName)
	deploymentName        = fmt.Sprintf("%s-deployment", testName)
	scalerName            = fmt.Sprintf("%s-scaler", testName)
	scaledObjectName      = fmt.Sprintf("%s-so", testName)
	metricsServerEndpoint = fmt.Sprintf("http://%s.%s.svc.cluster.local:8080/api/value", serviceName, testNamespace)
)

type templateData struct {
	TestNamespace         string
	ServiceName           string
	DeploymentName        string
	ScalerName            string
	ScaledObjectName      string
	MetricsServerEndpoint string
	MetricThreshold       string
	MetricValue           string
	MaxReplicaCount       string
}

const (
	serviceTemplate = `
apiVersion: v1
kind: Service
metadata:
  name: {{.ServiceName}}
  namespace: {{.TestNamespace}}
spec:
  ports:
    - port: 6000
      name: grpc
      targetPort: 6000
    - port: 8080
      name: http
      targetPort: 8080
  selector:
    app: {{.ScalerName}}
`

	scalerTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.ScalerName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.ScalerName}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.ScalerName}}
  template:
    metadata:
      labels:
        app: {{.ScalerName}}
    spec:
      containers:
        - name: scaler
          image: ghcr.io/kedacore/tests-external-scaler:latest
          imagePullPolicy: Always
          ports:
          - containerPort: 6000
          - containerPort: 8080
          readinessProbe:
            httpGet:
              path: /ready
              port: 8080
              scheme: HTTP
            initialDelaySeconds: 5
            timeoutSeconds: 1
            periodSeconds: 5
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
  replicas: 0
  selector:
    matchLabels:
      app: {{.DeploymentName}}
  template:
    metadata:
      labels:
        app: {{.DeploymentName}}
    spec:
      containers:
        - name: nginx
          image: ghcr.io/nginx/nginx-unprivileged:1.26
`

	// The first trigger always discovers its metric spec and matches no pods, so it
	// never asks for replicas of its own.
	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 600
  cooldownPeriod: 10
  idleReplicaCount: 0
  minReplicaCount: 1
  maxReplicaCount: {{.MaxReplicaCount}}
  triggers:
    - type: kubernetes-workload
      metadata:
        podSelector: app=no-such-workload
        value: "1"
    - type: external-push
      metadata:
        scalerAddress: {{.ServiceName}}.{{.TestNamespace}}:6000
        metricThreshold: "{{.MetricThreshold}}"
`

	updateMetricTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: update-metric-value
  namespace: {{.TestNamespace}}
spec:
  template:
    spec:
      containers:
      - name: curl-client
        image: docker.io/curlimages/curl
        imagePullPolicy: Always
        command: ["curl", "-X", "POST", "{{.MetricsServerEndpoint}}/{{.MetricValue}}"]
      restartPolicy: Never`
)

func TestPartialMetricDiscovery(t *testing.T) {
	// setup
	t.Log("--- setting up ---")

	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, scalerName, testNamespace, 1, 60, 1),
		"external scaler should be ready after 1 minute")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// Created only once the external scaler can answer, so the test starts from a
	// complete discovery.
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	testBothTriggersDiscovered(t)
	testFailingDiscoveryKeepsTriggers(t, kc, data)
	testPushActivationAfterRecovery(t, kc, data)

	// cleanup
	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:         testNamespace,
			ServiceName:           serviceName,
			DeploymentName:        deploymentName,
			ScalerName:            scalerName,
			ScaledObjectName:      scaledObjectName,
			MetricThreshold:       "10",
			MaxReplicaCount:       "2",
			MetricsServerEndpoint: metricsServerEndpoint,
		}, []Template{
			{Name: "scalerTemplate", Config: scalerTemplate},
			{Name: "serviceTemplate", Config: serviceTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
		}
}

// externalMetricNames returns the metric names persisted in the ScaledObject status.
func externalMetricNames(t *testing.T) []string {
	t.Helper()

	kedaKc := GetKedaKubernetesClient(t)
	so, err := kedaKc.ScaledObjects(testNamespace).Get(context.Background(), scaledObjectName, metav1.GetOptions{})
	assert.NoErrorf(t, err, "cannot get ScaledObject %s/%s", testNamespace, scaledObjectName)
	if err != nil {
		return nil
	}
	return so.Status.ExternalMetricNames
}

func testBothTriggersDiscovered(t *testing.T) {
	t.Log("--- testing initial discovery ---")

	assert.Eventuallyf(t, func() bool {
		names := externalMetricNames(t)
		t.Logf("ScaledObject %s externalMetricNames: %v", scaledObjectName, names)
		return len(names) == 2
	}, 3*time.Minute, time.Second, "both triggers should end up in the ScaledObject status")
}

func testFailingDiscoveryKeepsTriggers(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing that a failing discovery keeps the trigger ---")

	// Take the external scaler down, so its metric spec can no longer be discovered.
	KubernetesScaleDeployment(t, kc, scalerName, 0, testNamespace)
	assert.True(t, WaitForPodsTerminated(t, kc, fmt.Sprintf("app=%s", scalerName), testNamespace, 60, 1),
		"external scaler pods should be gone after 1 minute")

	// Reconcile the ScaledObject while the scaler is down, which is what rebuilds the
	// HPA and rewrites the metric names in the status.
	data.MaxReplicaCount = "3"
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	// The unreachable trigger has to stay, on the metric spec it reported last.
	for range 12 {
		names := externalMetricNames(t)
		assert.Lenf(t, names, 2, "the unreachable trigger must keep its place in the status, got %v", names)
		if len(names) != 2 {
			return
		}
		time.Sleep(5 * time.Second)
	}
}

func testPushActivationAfterRecovery(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing push activation after the scaler recovered ---")

	KubernetesScaleDeployment(t, kc, scalerName, 1, testNamespace)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, scalerName, testNamespace, 1, 60, 2),
		"external scaler should be ready again after 2 minutes")

	// Reaching 1 replica proves the pushed activation was routed to the trigger instead
	// of being dropped, the scale loop cannot do it within this window.
	data.MetricValue = data.MetricThreshold
	KubectlReplaceWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)
	assert.True(t, WaitForJobSuccess(t, kc, "update-metric-value", testNamespace, 60, 2),
		"the job setting the metric value should succeed")

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 90, 2),
		"replica count should be 1 after 3 minutes")

	KubectlDeleteWithTemplate(t, data, "updateMetricTemplate", updateMetricTemplate)
}
