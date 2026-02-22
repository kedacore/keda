//go:build e2e
// +build e2e

package replicaset_scale_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "replicaset-scale-test"
)

var (
	testNamespace           = fmt.Sprintf("%s-ns", testName)
	monitoredDeploymentName = fmt.Sprintf("%s-monitored", testName)
	replicaSetName          = fmt.Sprintf("%s-rs", testName)
	scaledObjectName        = fmt.Sprintf("%s-so", testName)
)

type templateData struct {
	TestNamespace           string
	MonitoredDeploymentName string
	ReplicaSetName          string
	ScaledObjectName        string
}

const (
	monitoredDeploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.MonitoredDeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.MonitoredDeploymentName}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: {{.MonitoredDeploymentName}}
  template:
    metadata:
      labels:
        app: {{.MonitoredDeploymentName}}
    spec:
      containers:
        - name: nginx
          image: 'ghcr.io/nginx/nginx-unprivileged:1.26'`

	replicaSetTemplate = `apiVersion: apps/v1
kind: ReplicaSet
metadata:
  name: {{.ReplicaSetName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.ReplicaSetName}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: {{.ReplicaSetName}}
  template:
    metadata:
      labels:
        app: {{.ReplicaSetName}}
    spec:
      containers:
        - name: nginx
          image: 'ghcr.io/nginx/nginx-unprivileged:1.26'`

	scaledObjectTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: ReplicaSet
    name: {{.ReplicaSetName}}
  pollingInterval: 5
  cooldownPeriod: 5
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
      podSelector: 'app={{.MonitoredDeploymentName}}'
      value: '1'
`
)

func TestReplicaSetScaling(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	assert.True(t, waitForReplicaSetReplicaCount(t, kc, replicaSetName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testScaleOut(t, kc)
	testScaleIn(t, kc)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")

	// scale monitored deployment to 5 replicas
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 5, testNamespace)
	assert.True(t, waitForReplicaSetReplicaCount(t, kc, replicaSetName, testNamespace, 5, 60, 1),
		"replica count should be 5 after 1 minute")

	// scale monitored deployment to 10 replicas
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 10, testNamespace)
	assert.True(t, waitForReplicaSetReplicaCount(t, kc, replicaSetName, testNamespace, 10, 60, 1),
		"replica count should be 10 after 1 minute")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	// scale monitored deployment to 5 replicas
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 5, testNamespace)
	assert.True(t, waitForReplicaSetReplicaCount(t, kc, replicaSetName, testNamespace, 5, 60, 1),
		"replica count should be 5 after 1 minute")

	// scale monitored deployment to 0 replicas
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 0, testNamespace)
	assert.True(t, waitForReplicaSetReplicaCount(t, kc, replicaSetName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:           testNamespace,
			MonitoredDeploymentName: monitoredDeploymentName,
			ReplicaSetName:          replicaSetName,
			ScaledObjectName:        scaledObjectName,
		}, []Template{
			{Name: "monitoredDeploymentTemplate", Config: monitoredDeploymentTemplate},
			{Name: "replicaSetTemplate", Config: replicaSetTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

// waitForReplicaSetReplicaCount waits until replicaset replica count hits target or iterations are exhausted
func waitForReplicaSetReplicaCount(t *testing.T, kc *kubernetes.Clientset, name, namespace string, target, iterations, intervalSeconds int) bool {
	for i := 0; i < iterations; i++ {
		rs, _ := kc.AppsV1().ReplicaSets(namespace).Get(context.Background(), name, metav1.GetOptions{})

		// Use spec.replicas when target is 0 (status.readyReplicas won't be set)
		var replicas int32
		if target == 0 {
			if rs.Spec.Replicas != nil {
				replicas = *rs.Spec.Replicas
			}
		} else {
			replicas = rs.Status.ReadyReplicas
		}

		t.Logf("Waiting for replicaset replicas to hit target. ReplicaSet - %s, Current - %d, Target - %d",
			name, replicas, target)

		if replicas == int32(target) {
			return true
		}

		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}

	return false
}
