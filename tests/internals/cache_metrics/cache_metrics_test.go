//go:build e2e
// +build e2e

package cache_metrics_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "cache-metrics-test"
)

var (
	testNamespace                      = fmt.Sprintf("%s-ns", testName)
	deploymentName                     = fmt.Sprintf("%s-deployment", testName)
	monitoredDeploymentName            = fmt.Sprintf("%s-monitored", testName)
	scaledObjectName                   = fmt.Sprintf("%s-so", testName)
	defaultMonitoredDeploymentReplicas = 8
)

type templateData struct {
	TestNamespace               string
	DeploymentName              string
	ScaledObjectName            string
	MonitoredDeploymentName     string
	MonitoredDeploymentReplicas int
	EnableUseCachedMetrics      bool
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
  replicas: {{.MonitoredDeploymentReplicas}}
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

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 1
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 1000
  minReplicaCount: 0
  maxReplicaCount: 10
  cooldownPeriod: 0
  triggers:
    - type: kubernetes-workload
      useCachedMetrics: {{.EnableUseCachedMetrics}}
      metadata:
        podSelector: 'app={{.MonitoredDeploymentName}}'
        value: '2'
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// test direct metrics query (the standard behavior)
	testDirectQuery(t, kc, data)

	// test querying metrics on polling interval
	testCacheMetricsOnPollingInterval(t, kc, data)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:               testNamespace,
			DeploymentName:              deploymentName,
			ScaledObjectName:            scaledObjectName,
			MonitoredDeploymentName:     monitoredDeploymentName,
			MonitoredDeploymentReplicas: defaultMonitoredDeploymentReplicas,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "monitoredDeploymentTemplate", Config: monitoredDeploymentTemplate},
		}
}

func testCacheMetricsOnPollingInterval(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing caching metrics on polling interval ---")

	KubectlApplyWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)
	data.MonitoredDeploymentReplicas = defaultMonitoredDeploymentReplicas
	KubectlApplyWithTemplate(t, data, "monitoredDeploymentTemplate", monitoredDeploymentTemplate)

	// initial replica count for a deployment
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 1),
		"replica count should be 1 after 1 minute")

	// initial replica count for a monitored deployment
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, monitoredDeploymentName, testNamespace, defaultMonitoredDeploymentReplicas, 60, 1),
		fmt.Sprintf("replica count should be %d after 1 minute", defaultMonitoredDeploymentReplicas))

	data.EnableUseCachedMetrics = true
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	// Metric Value = 8, DesiredAverageMetricValue = 2
	// should scale in to 8/2 = 4 replicas, irrespective of current replicas
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 4, 60, 3),
		"replica count should be 4 after 3 minute")

	// Changing Metric Value to 4, but because we have a long polling interval, the replicas number should remain the same
	data.MonitoredDeploymentReplicas = 4
	KubectlApplyWithTemplate(t, data, "monitoredDeploymentTemplate", monitoredDeploymentTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, monitoredDeploymentName, testNamespace, 4, 60, 1),
		fmt.Sprintf("replica count should be %d after 1 minute", defaultMonitoredDeploymentReplicas))

	// Let's wait at least 60s
	// the usual setting for `horizontal-pod-autoscaler-sync-period` is 15s)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 4, 60)

	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
}

func testDirectQuery(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- testing query metrics directly ---")

	KubectlApplyWithTemplate(t, data, "deploymentTemplate", deploymentTemplate)
	data.MonitoredDeploymentReplicas = defaultMonitoredDeploymentReplicas
	KubectlApplyWithTemplate(t, data, "monitoredDeploymentTemplate", monitoredDeploymentTemplate)

	// initial replica count for a deployment
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 1),
		"replica count should be 1 after 1 minute")

	// initial replica count for a monitored deployment
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, monitoredDeploymentName, testNamespace, defaultMonitoredDeploymentReplicas, 60, 1),
		fmt.Sprintf("replica count should be %d after 1 minute", defaultMonitoredDeploymentReplicas))

	data.EnableUseCachedMetrics = false
	KubectlApplyWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)

	// Metric Value = 8, DesiredAverageMetricValue = 2
	// should scale in to 8/2 = 4 replicas, irrespective of current replicas
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 4, 60, 3),
		"replica count should be 4 after 3 minute")

	// Changing Metric Value to 4, deployment should scale to 2
	data.MonitoredDeploymentReplicas = 4
	KubectlApplyWithTemplate(t, data, "monitoredDeploymentTemplate", monitoredDeploymentTemplate)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, monitoredDeploymentName, testNamespace, 4, 60, 1),
		fmt.Sprintf("replica count should be %d after 1 minute", defaultMonitoredDeploymentReplicas))

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 120, 1),
		"replica count should be 2 after 2 minutes")

	KubectlDeleteWithTemplate(t, data, "scaledObjectTemplate", scaledObjectTemplate)
}
