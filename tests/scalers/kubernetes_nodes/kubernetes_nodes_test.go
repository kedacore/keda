//go:build e2e
// +build e2e

package kubernetes_nodes_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "kubernetes-nodes-test"
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
        image: 'ghcr.io/nginx/nginx-unprivileged:1.26'`

	// Active on every cluster: at least one Ready node carries
	// kubernetes.io/os=linux, so the count exceeds the default
	// activationValue of 0 and HPA aims for ceil(count/1) replicas.
	scaledObjectActiveTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  cooldownPeriod: 5
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 5
  triggers:
  - type: kubernetes-nodes
    metadata:
      nodeSelector: "kubernetes.io/os=linux"
      value: "1"`

	// Inactive on every cluster: no fleet reaches a million Ready nodes,
	// so the count never exceeds the activationValue and the target holds
	// at minReplicaCount (0).
	scaledObjectIdleTemplate = `apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  cooldownPeriod: 5
  minReplicaCount: {{.MinReplicaCount}}
  maxReplicaCount: {{.MaxReplicaCount}}
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 5
  triggers:
  - type: kubernetes-nodes
    metadata:
      nodeSelector: "kubernetes.io/os=linux"
      value: "1"
      activationValue: "1000000"`
)

func TestScaler(t *testing.T) {
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	// Create kubernetes resources (namespace, deployment)
	CreateKubernetesResources(t, kc, testNamespace, data, templates)
	defer DeleteKubernetesResources(t, testNamespace, data, templates)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	t.Run("Ready node count activates scaling", func(t *testing.T) {
		testNodeCountActivates(t, kc, data)
	})

	t.Run("Unreachable activation holds at zero", func(t *testing.T) {
		testActivationHoldsAtZero(t, kc, data)
	})
}

func testNodeCountActivates(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- test Ready node count activates scaling ---")

	KubectlApplyWithTemplate(t, data, "scaledObjectActiveTemplate", scaledObjectActiveTemplate)
	defer KubectlDeleteWithTemplate(t, data, "scaledObjectActiveTemplate", scaledObjectActiveTemplate)

	// The Ready count is at least 1 on any cluster, so the target must
	// scale away from 0 (HPA aims for the node count, capped at max).
	assert.NotEqual(t, 0, WaitForDeploymentReplicaCountChange(t, kc, deploymentName, testNamespace, 60, 2),
		"replica count should change from 0 when nodes are Ready")
}

func testActivationHoldsAtZero(t *testing.T, kc *kubernetes.Clientset, data templateData) {
	t.Log("--- test unreachable activation holds at zero ---")

	KubectlApplyWithTemplate(t, data, "scaledObjectIdleTemplate", scaledObjectIdleTemplate)
	defer KubectlDeleteWithTemplate(t, data, "scaledObjectIdleTemplate", scaledObjectIdleTemplate)

	// The previous subtest may have left replicas behind; the inactive
	// trigger scales back to minReplicaCount (0) after the cooldown, then
	// the HPA goes dormant at zero and the count must hold there.
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 2),
		"replica count should scale back to 0 with an inactive trigger")

	// The count never exceeds the activationValue, so the trigger stays
	// inactive and the target holds at minReplicaCount (0).
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 30)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:    testNamespace,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
			MinReplicaCount:  minReplicaCount,
			MaxReplicaCount:  maxReplicaCount,
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
		}
}
