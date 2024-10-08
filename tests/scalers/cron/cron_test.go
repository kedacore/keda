//go:build e2e
// +build e2e

package cron_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

const (
	testName = "cron-test"
)

var (
	testNamespace    = fmt.Sprintf("%s-ns", testName)
	deploymentName   = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName = fmt.Sprintf("%s-so", testName)
)

type templateData struct {
	TestNamespace    string
	DeploymentName   string
	ScaledObjectName string
	StartMin         string
	EndMin           string
}

const (
	deploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    deploy: {{.DeploymentName}}
spec:
  replicas: 1
  selector:
    matchLabels:
      pod: {{.DeploymentName}}
  template:
    metadata:
      labels:
        pod: {{.DeploymentName}}
    spec:
      containers:
        - name: nginx
          image: 'nginxinc/nginx-unprivileged'
`

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  cooldownPeriod: 5
  minReplicaCount: 1
  maxReplicaCount: 10
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 15
  triggers:
  - type: cron
    metadata:
      timezone: UTC
      start: {{.StartMin}} * * * *
      end: {{.EndMin}} * * * *
      desiredReplicas: '4'
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, []Template{
		{Name: "deploymentTemplate", Config: deploymentTemplate},
	})

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 2),
		"replica count should be 1 within 1 minute")

	t.Log("--- installing ScaledObject ---")
	CreateKubernetesResources(t, kc, testNamespace, data, []Template{
		{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
	})

	// test scaling
	testScaleOut(t, kc)
	testScaleIn(t, kc)

	// cleanup
	DeleteKubernetesResources(t, testNamespace, data, templates)
}

func getTemplateData() (templateData, []Template) {
	now := time.Now().UTC()
	start := (now.Minute()) % 60
	end := (start + 1) % 60

	return templateData{
			TestNamespace:    testNamespace,
			DeploymentName:   deploymentName,
			ScaledObjectName: scaledObjectName,
			StartMin:         fmt.Sprintf("%d", start),
			EndMin:           fmt.Sprintf("%d", end),
		}, []Template{
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 4, 60, 2),
		"replica count should be 4 within 1 minute")
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 1, 60, 2),
		"replica count should be 1 within 1 minute")
}
