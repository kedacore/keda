//go:build e2e
// +build e2e

package pause_scale_in_restore_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file

const (
	testName = "pause-scalein-restore-test"
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
	ScaledObjectName        string
	MonitoredDeploymentName string
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
  replicas: 2
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
  annotations:
    autoscaling.keda.sh/paused-scale-in: "true"
    test-annotation: "preserve-this"
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  minReplicaCount: 0
  maxReplicaCount: 10
  cooldownPeriod:  0
  triggers:
    - type: kubernetes-workload
      metadata:
        podSelector: 'app={{.MonitoredDeploymentName}}'
        value: '1'
`

	scaledObjectWithoutPauseTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  annotations:
    test-annotation: "preserve-this"
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  minReplicaCount: 0
  maxReplicaCount: 10
  cooldownPeriod:  0
  triggers:
    - type: kubernetes-workload
      metadata:
        podSelector: 'app={{.MonitoredDeploymentName}}'
        value: '1'
`
)

func TestScaler(t *testing.T) {
	// setup
	t.Log("--- setting up ---")

	// Create kubernetes resources
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()

	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// assert that the deployment did not scale down after one minute (scale-in is paused)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 2, 60)

	// remove the paused-scale-in annotation
	KubectlReplaceWithTemplate(t, data, "scaledObjectWithoutPauseTemplate", scaledObjectWithoutPauseTemplate)

	// assert that the deployment scales down to 0 after removing the annotation
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 2),
		"replica count should be 0 after removing paused-scale-in annotation")

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
		}
}
