//go:build e2e
// +build e2e

package broken_scaledobject_tolerancy_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "broken-scaledobject-tolerancy-test"
)

var (
	testNamespace           = fmt.Sprintf("%s-ns", testName)
	deploymentName          = fmt.Sprintf("%s-deployment", testName)
	monitoredDeploymentName = fmt.Sprintf("%s-monitored", testName)
	scaledObjectName        = fmt.Sprintf("%s-so", testName)
)

type templateData struct {
	TestNamespace           string
	DeploymentName          string
	MonitoredDeploymentName string
	ScaledObjectName        string
}

const (
	monitoredDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.MonitoredDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    deploy: workload-test
spec:
  replicas: 0
  selector:
    matchLabels:
      pod: workload-test
  template:
    metadata:
      labels:
        pod: workload-test
    spec:
      containers:
        - name: nginx
          image: 'ghcr.io/nginx/nginx-unprivileged:1.26'`

	deploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
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
      - name: nginx
        image: 'ghcr.io/nginx/nginx-unprivileged:1.26'`

	brokenScaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}-broken
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.MonitoredDeploymentName}}
  minReplicaCount: 0
  maxReplicaCount: 1
  triggers:
  - metadata:
      activationLagThreshold: '1'
      bootstrapServers: 1.2.3.4:9092
      consumerGroup: earliest
      lagThreshold: '1'
      offsetResetPolicy: earliest
      topic: kafka-topic
    type: kafka
`

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 1
  cooldownPeriod: 0
  minReplicaCount: 0
  maxReplicaCount: 10
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 5
  triggers:
  - type: kubernetes-workload
    metadata:
      podSelector: 'pod=workload-test'
      value: '1'
`
)

// As we need to ensure that a broken ScaledObject doesn't impact
// to other ScaledObjects https://github.com/kedacore/keda/issues/5083,
// this test deploys a broken ScaledObject pointing to missing endpoint
// which produces timeouts. In the meantime, we deploy another ScaledObject
// and validate that it works although the broken ScaledObject produces timeouts.
// all the time. This prevents us for introducing deadlocks on internal scalers cache
func TestBrokenScaledObjectTolerance(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	testScaleOut(t, kc)
	testScaleIn(t, kc)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:           testNamespace,
			DeploymentName:          deploymentName,
			ScaledObjectName:        scaledObjectName,
			MonitoredDeploymentName: monitoredDeploymentName,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "monitoredDeploymentTemplate", Config: monitoredDeploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
			{Name: "brokenScaledObjectTemplate", Config: brokenScaledObjectTemplate},
		}
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	// scale monitored deployment to 2 replicas
	replicas := 2
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, int64(replicas), testNamespace)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, replicas, 10, 6),
		fmt.Sprintf("replica count should be %d after 1 minute", replicas))

	// scale monitored deployment to 4 replicas
	replicas = 4
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, int64(replicas), testNamespace)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, replicas, 10, 6),
		fmt.Sprintf("replica count should be %d after 1 minute", replicas))
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	// scale monitored deployment to 2 replicas
	replicas := 2
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, int64(replicas), testNamespace)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, replicas, 10, 6),
		fmt.Sprintf("replica count should be %d after 1 minute", replicas))

	// scale monitored deployment to 0 replicas
	replicas = 0
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, int64(replicas), testNamespace)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, replicas, 10, 6),
		fmt.Sprintf("replica count should be %d after 1 minute", replicas))
}
