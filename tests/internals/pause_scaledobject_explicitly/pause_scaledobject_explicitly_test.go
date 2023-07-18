//go:build e2e
// +build e2e

package pause_scaledobject_explicitly_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file

const (
	testName = "pause-scaledobject-explicitly-test"
)

var (
	testNamespace           = fmt.Sprintf("%s-ns", testName)
	deploymentName          = fmt.Sprintf("%s-deployment", testName)
	monitoredDeploymentName = fmt.Sprintf("%s-monitored", testName)
	scaledObjectName        = fmt.Sprintf("%s-so", testName)
	maxReplicaCount         = 1
	minReplicaCount         = 0
	testScaleOutWaitMin     = 1
	testPauseAtNWaitMin     = 1
	testScaleInWaitMin      = 1
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
          image: nginxinc/nginx-unprivileged:alpine-slim
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
        - name: {{.DeploymentName}}
          image: nginxinc/nginx-unprivileged:alpine-slim
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
  minReplicaCount: 0
  maxReplicaCount: 5
  cooldownPeriod:  5
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

	// scaling to paused replica count
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, 1),
		"replica count should be 0 after 1 minute")

	// test scaling
	testPauseWhenScaleOut(t, kc)
	testScaleOut(t, kc)
	testPauseWhenScaleIn(t, kc)
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
			{Name: "scaledObjectAnnotatedTemplate", Config: scaledObjectTemplate},
		}
}

func upsertScaledObjectAnnotation(t assert.TestingT) {
	_, err := ExecuteCommand(fmt.Sprintf("kubectl annotate scaledobject/%s -n %s autoscaling.keda.sh/paused='true' --overwrite", scaledObjectName, testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func removeScaledObjectAnnotation(t assert.TestingT) {
	_, err := ExecuteCommand(fmt.Sprintf("kubectl annotate scaledobject/%s -n %s autoscaling.keda.sh/paused- --overwrite", scaledObjectName, testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func testPauseWhenScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing pausing at 0 ---")

	upsertScaledObjectAnnotation(t)
	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 2, testNamespace)
	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, monitoredDeploymentName, testNamespace, 2, 60, testScaleOutWaitMin),
		"monitoredDeploymentName replica count should be 2 after %d minute(s)", testScaleOutWaitMin)

	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, 0, 10)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")

	removeScaledObjectAnnotation(t)

	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 5, testNamespace)
	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, monitoredDeploymentName, testNamespace, 5, 60, testScaleOutWaitMin),
		"monitoredDeploymentName replica count should be 5 after %d minute(s)", testScaleOutWaitMin)

	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 5, 60, testScaleOutWaitMin),
		"replica count should be 5 after %d minute(s)", testScaleOutWaitMin)
}

func testPauseWhenScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing pausing at N ---")

	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 5, testNamespace)

	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 5, 10, testPauseAtNWaitMin),
		"replica count should be 5 after %d minute(s)", testPauseAtNWaitMin)

	upsertScaledObjectAnnotation(t)

	KubernetesScaleDeployment(t, kc, monitoredDeploymentName, 0, testNamespace)

	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, monitoredDeploymentName, testNamespace, 0, 60, testPauseAtNWaitMin),
		"monitoredDeploymentName replica count should be 0 after %d minute(s)", testPauseAtNWaitMin)
	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 5, 10, testPauseAtNWaitMin),
		"replica count should be 5 after %d minute(s)", testPauseAtNWaitMin)

}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")

	removeScaledObjectAnnotation(t)
	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, testScaleInWaitMin),
		"replica count should be 0 after %d minutes", testScaleInWaitMin)
}
