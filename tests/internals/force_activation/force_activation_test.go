//go:build e2e
// +build e2e

package force_activation_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file

const (
	testName = "pause-scalein-test"
)

var (
	testNamespace           = fmt.Sprintf("%s-ns", testName)
	deploymentName          = fmt.Sprintf("%s-deployment", testName)
	monitoredDeploymentName = fmt.Sprintf("%s-monitored", testName)
	scaledObjectName        = fmt.Sprintf("%s-so", testName)
	testScaleOutWaitMin     = 1
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
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  pollingInterval: 5
  idleReplicaCount: 0
  minReplicaCount: 2
  maxReplicaCount: 5
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

	// assert that the deployment did not scale down after one minute
	WaitForDeploymentReplicaCountChange(t, kc, deploymentName, testNamespace, 0, 60)

	// test activation
	testForceActivation(t, kc)
	testUnforceActivation(t, kc)

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

func testForceActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing forcing activation ---")

	upsertScaledObjectAnnotation(t)

	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 2, 60, testScaleOutWaitMin),
		"deploymentName replica count should be 2 after %d minute(s)", testScaleOutWaitMin)
}

func testUnforceActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing unforcing activation ---")

	removeScaledObjectAnnotation(t)

	assert.Truef(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, 0, 60, testScaleOutWaitMin),
		"deploymentName replica count should be 0 after %d minute(s)", testScaleOutWaitMin)
}

func upsertScaledObjectAnnotation(t assert.TestingT) {
	_, err := ExecuteCommand(fmt.Sprintf("kubectl annotate scaledobject/%s -n %s autoscaling.keda.sh/force-activation=true --overwrite", scaledObjectName, testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}

func removeScaledObjectAnnotation(t assert.TestingT) {
	_, err := ExecuteCommand(fmt.Sprintf("kubectl annotate scaledobject/%s -n %s autoscaling.keda.sh/force-activation- --overwrite", scaledObjectName, testNamespace))
	assert.NoErrorf(t, err, "cannot execute command - %s", err)
}
